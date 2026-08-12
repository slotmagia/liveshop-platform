package iam

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) transaction(ctx context.Context, operation func(*sql.Tx) error) error {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err != nil {
			return err
		}
		last = operation(tx)
		if last == nil {
			last = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
		var postgresError *pgconn.PgError
		if !errors.As(last, &postgresError) || (postgresError.Code != "40001" && postgresError.Code != "40P01") {
			return last
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 20 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return last
}

func (s *Store) putDepartmentSQL(ctx context.Context, tenant Tenant, input Department, expectedVersion int64) (Department, error) {
	if !tenant.Valid() || input.ID <= 0 || strings.TrimSpace(input.Name) == "" || !validStatus(input.Status) || expectedVersion < 0 {
		return Department{}, ErrInvalid
	}
	input.Name = strings.TrimSpace(input.Name)
	var output Department
	err := s.transaction(ctx, func(tx *sql.Tx) error {
		current, exists, err := departmentSQL(ctx, tx, tenant, input.ID, true)
		if err != nil {
			return err
		}
		if exists && sameDepartment(current, input) && (expectedVersion == 0 || expectedVersion == current.Version || expectedVersion == current.Version-1) {
			output = current
			return nil
		}
		if (!exists && expectedVersion != 0) || (exists && current.Version != expectedVersion) {
			return ErrConflict
		}
		if input.ParentID != nil {
			var status string
			if err := tx.QueryRowContext(ctx, `SELECT status FROM platform_department WHERE app_id=$1 AND merchant_id=$2 AND department_id=$3`, tenant.AppID, tenant.MerchantID, *input.ParentID).Scan(&status); err != nil || status != StatusActive {
				return ErrInvalid
			}
			var cycle bool
			if err := tx.QueryRowContext(ctx, `WITH RECURSIVE ancestors AS (
                    SELECT department_id, parent_department_id FROM platform_department WHERE app_id=$1 AND merchant_id=$2 AND department_id=$3
                    UNION ALL
                    SELECT d.department_id, d.parent_department_id FROM platform_department d JOIN ancestors a ON d.department_id=a.parent_department_id WHERE d.app_id=$1 AND d.merchant_id=$2
                ) SELECT EXISTS(SELECT 1 FROM ancestors WHERE department_id=$4)`, tenant.AppID, tenant.MerchantID, *input.ParentID, input.ID).Scan(&cycle); err != nil {
				return err
			}
			if cycle {
				return ErrInvalid
			}
		}
		if exists {
			input.Version = current.Version + 1
			result, err := tx.ExecContext(ctx, `UPDATE platform_department SET parent_department_id=$1,name=$2,status=$3,version=$4,updated_at=NOW() WHERE app_id=$5 AND merchant_id=$6 AND department_id=$7 AND version=$8`, input.ParentID, input.Name, input.Status, input.Version, tenant.AppID, tenant.MerchantID, input.ID, expectedVersion)
			if err != nil {
				return err
			}
			rows, _ := result.RowsAffected()
			if rows != 1 {
				return ErrConflict
			}
		} else {
			input.Version = 1
			if _, err := tx.ExecContext(ctx, `INSERT INTO platform_department(app_id,merchant_id,department_id,parent_department_id,name,status,version) VALUES($1,$2,$3,$4,$5,$6,1)`, tenant.AppID, tenant.MerchantID, input.ID, input.ParentID, input.Name, input.Status); err != nil {
				return err
			}
		}
		if err := bumpRevision(ctx, tx, tenant); err != nil {
			return err
		}
		if err := auditIAMMutation(ctx, tx, tenant, "iam.department.put", "platform.department", input.ID, input); err != nil {
			return err
		}
		output = input
		return nil
	})
	return output, err
}

func departmentSQL(ctx context.Context, tx *sql.Tx, tenant Tenant, id int64, lock bool) (Department, bool, error) {
	query := `SELECT department_id,parent_department_id,name,status,version FROM platform_department WHERE app_id=$1 AND merchant_id=$2 AND department_id=$3`
	if lock {
		query += ` FOR UPDATE`
	}
	var item Department
	var parent sql.NullInt64
	err := tx.QueryRowContext(ctx, query, tenant.AppID, tenant.MerchantID, id).Scan(&item.ID, &parent, &item.Name, &item.Status, &item.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return Department{}, false, nil
	}
	if err != nil {
		return Department{}, false, err
	}
	if parent.Valid {
		item.ParentID = &parent.Int64
	}
	return item, true, nil
}

func (s *Store) putRoleSQL(ctx context.Context, tenant Tenant, input Role, expectedVersion int64) (Role, error) {
	if !tenant.Valid() || input.ID <= 0 || strings.TrimSpace(input.Name) == "" || !validStatus(input.Status) || expectedVersion < 0 {
		return Role{}, ErrInvalid
	}
	input.Name = strings.TrimSpace(input.Name)
	var output Role
	err := s.transaction(ctx, func(tx *sql.Tx) error {
		current, exists, err := roleSQL(ctx, tx, tenant, input.ID, true)
		if err != nil {
			return err
		}
		if exists && current.SuperAdmin != input.SuperAdmin {
			return ErrInvalid
		}
		if exists && current.Name == input.Name && current.Status == input.Status && current.SuperAdmin == input.SuperAdmin && (expectedVersion == 0 || expectedVersion == current.Version || expectedVersion == current.Version-1) {
			output = current
			return nil
		}
		if (!exists && expectedVersion != 0) || (exists && current.Version != expectedVersion) {
			return ErrConflict
		}
		if exists {
			input.Version = current.Version + 1
			result, err := tx.ExecContext(ctx, `UPDATE platform_role SET name=$1,status=$2,is_super_admin=$3,version=$4,updated_at=NOW() WHERE app_id=$5 AND merchant_id=$6 AND role_id=$7 AND version=$8`, input.Name, input.Status, input.SuperAdmin, input.Version, tenant.AppID, tenant.MerchantID, input.ID, expectedVersion)
			if err != nil {
				return err
			}
			rows, _ := result.RowsAffected()
			if rows != 1 {
				return ErrConflict
			}
		} else {
			input.Version = 1
			if _, err := tx.ExecContext(ctx, `INSERT INTO platform_role(app_id,merchant_id,role_id,name,status,is_super_admin,version) VALUES($1,$2,$3,$4,$5,$6,1)`, tenant.AppID, tenant.MerchantID, input.ID, input.Name, input.Status, input.SuperAdmin); err != nil {
				return err
			}
		}
		if err := bumpRevision(ctx, tx, tenant); err != nil {
			return err
		}
		if err := auditIAMMutation(ctx, tx, tenant, "iam.role.put", "platform.role", input.ID, input); err != nil {
			return err
		}
		output = input
		return nil
	})
	return output, err
}

func roleSQL(ctx context.Context, tx *sql.Tx, tenant Tenant, id int64, lock bool) (Role, bool, error) {
	query := `SELECT role_id,name,status,is_super_admin,version FROM platform_role WHERE app_id=$1 AND merchant_id=$2 AND role_id=$3`
	if lock {
		query += ` FOR UPDATE`
	}
	var item Role
	err := tx.QueryRowContext(ctx, query, tenant.AppID, tenant.MerchantID, id).Scan(&item.ID, &item.Name, &item.Status, &item.SuperAdmin, &item.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return Role{}, false, nil
	}
	return item, err == nil, err
}

func (s *Store) setRolePolicySQL(ctx context.Context, tenant Tenant, roleID, expectedVersion int64, policy RolePolicy) (Role, error) {
	if !tenant.Valid() || roleID <= 0 || expectedVersion <= 0 {
		return Role{}, ErrInvalid
	}
	policy = normalizePolicy(policy)
	seen := map[string]bool{}
	for _, scope := range policy.Scopes {
		if scope.Resource == "" || !validScope(scope.Type) || seen[scope.Resource] || (scope.Type == ScopeCustom && len(scope.DepartmentIDs) == 0) {
			return Role{}, ErrInvalid
		}
		seen[scope.Resource] = true
	}
	var output Role
	err := s.transaction(ctx, func(tx *sql.Tx) error {
		role, ok, err := roleSQL(ctx, tx, tenant, roleID, true)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		current, err := rolePolicySQL(ctx, tx, tenant, roleID)
		if err != nil {
			return err
		}
		if policiesEqual(current, policy) && (expectedVersion == role.Version || expectedVersion == role.Version-1) {
			output = role
			return nil
		}
		if role.Version != expectedVersion {
			return ErrConflict
		}
		for _, code := range policy.Permissions {
			var exists bool
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM platform_permission_catalog WHERE permission_code=$1 AND active=TRUE)`, code).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return ErrInvalid
			}
		}
		for _, scope := range policy.Scopes {
			for _, departmentID := range scope.DepartmentIDs {
				var exists bool
				if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM platform_department WHERE app_id=$1 AND merchant_id=$2 AND department_id=$3 AND status='ACTIVE')`, tenant.AppID, tenant.MerchantID, departmentID).Scan(&exists); err != nil {
					return err
				}
				if !exists {
					return ErrInvalid
				}
			}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM platform_role_permission WHERE app_id=$1 AND merchant_id=$2 AND role_id=$3`, tenant.AppID, tenant.MerchantID, roleID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM platform_role_data_scope WHERE app_id=$1 AND merchant_id=$2 AND role_id=$3`, tenant.AppID, tenant.MerchantID, roleID); err != nil {
			return err
		}
		for _, code := range policy.Permissions {
			if _, err := tx.ExecContext(ctx, `INSERT INTO platform_role_permission(app_id,merchant_id,role_id,permission_code) VALUES($1,$2,$3,$4)`, tenant.AppID, tenant.MerchantID, roleID, code); err != nil {
				return err
			}
		}
		for _, scope := range policy.Scopes {
			if _, err := tx.ExecContext(ctx, `INSERT INTO platform_role_data_scope(app_id,merchant_id,role_id,resource_code,scope_type) VALUES($1,$2,$3,$4,$5)`, tenant.AppID, tenant.MerchantID, roleID, scope.Resource, scope.Type); err != nil {
				return err
			}
			for _, departmentID := range scope.DepartmentIDs {
				if _, err := tx.ExecContext(ctx, `INSERT INTO platform_role_scope_department(app_id,merchant_id,role_id,resource_code,department_id) VALUES($1,$2,$3,$4,$5)`, tenant.AppID, tenant.MerchantID, roleID, scope.Resource, departmentID); err != nil {
					return err
				}
			}
		}
		role.Version++
		if _, err := tx.ExecContext(ctx, `UPDATE platform_role SET version=$1,updated_at=NOW() WHERE app_id=$2 AND merchant_id=$3 AND role_id=$4`, role.Version, tenant.AppID, tenant.MerchantID, roleID); err != nil {
			return err
		}
		if err := bumpRevision(ctx, tx, tenant); err != nil {
			return err
		}
		if err := auditIAMMutation(ctx, tx, tenant, "iam.role.policy.put", "platform.role", role.ID, policy); err != nil {
			return err
		}
		output = role
		return nil
	})
	return output, err
}

func rolePolicySQL(ctx context.Context, tx *sql.Tx, tenant Tenant, roleID int64) (RolePolicy, error) {
	var policy RolePolicy
	rows, err := tx.QueryContext(ctx, `SELECT permission_code FROM platform_role_permission WHERE app_id=$1 AND merchant_id=$2 AND role_id=$3 ORDER BY permission_code`, tenant.AppID, tenant.MerchantID, roleID)
	if err != nil {
		return policy, err
	}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			rows.Close()
			return policy, err
		}
		policy.Permissions = append(policy.Permissions, code)
	}
	if err := rows.Close(); err != nil {
		return policy, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT s.resource_code,s.scope_type,d.department_id FROM platform_role_data_scope s LEFT JOIN platform_role_scope_department d ON d.app_id=s.app_id AND d.merchant_id=s.merchant_id AND d.role_id=s.role_id AND d.resource_code=s.resource_code WHERE s.app_id=$1 AND s.merchant_id=$2 AND s.role_id=$3 ORDER BY s.resource_code,d.department_id`, tenant.AppID, tenant.MerchantID, roleID)
	if err != nil {
		return policy, err
	}
	byResource := map[string]int{}
	for rows.Next() {
		var resource, scopeType string
		var department sql.NullInt64
		if err := rows.Scan(&resource, &scopeType, &department); err != nil {
			rows.Close()
			return policy, err
		}
		index, ok := byResource[resource]
		if !ok {
			policy.Scopes = append(policy.Scopes, ScopeRule{Resource: resource, Type: scopeType})
			index = len(policy.Scopes) - 1
			byResource[resource] = index
		}
		if department.Valid {
			policy.Scopes[index].DepartmentIDs = append(policy.Scopes[index].DepartmentIDs, department.Int64)
		}
	}
	return normalizePolicy(policy), rows.Close()
}

func (s *Store) setSubjectAssignmentSQL(ctx context.Context, tenant Tenant, subject string, assignment SubjectAssignment) error {
	if !tenant.Valid() || strings.TrimSpace(subject) == "" {
		return ErrInvalid
	}
	subject = strings.TrimSpace(subject)
	assignment.RoleIDs = uniqueInt64s(assignment.RoleIDs)
	primary := 0
	departmentSeen := map[int64]bool{}
	for _, item := range assignment.Departments {
		if item.DepartmentID <= 0 || departmentSeen[item.DepartmentID] {
			return ErrInvalid
		}
		departmentSeen[item.DepartmentID] = true
		if item.Primary {
			primary++
		}
	}
	if primary > 1 {
		return ErrInvalid
	}
	sort.Slice(assignment.Departments, func(i, j int) bool {
		return assignment.Departments[i].DepartmentID < assignment.Departments[j].DepartmentID
	})
	return s.transaction(ctx, func(tx *sql.Tx) error {
		current, err := subjectAssignmentSQL(ctx, tx, tenant, subject)
		if err != nil {
			return err
		}
		if fmtAssignment(current) == fmtAssignment(assignment) {
			return nil
		}
		for _, roleID := range assignment.RoleIDs {
			var exists bool
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM platform_role WHERE app_id=$1 AND merchant_id=$2 AND role_id=$3 AND status='ACTIVE')`, tenant.AppID, tenant.MerchantID, roleID).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return ErrInvalid
			}
		}
		for departmentID := range departmentSeen {
			var exists bool
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM platform_department WHERE app_id=$1 AND merchant_id=$2 AND department_id=$3 AND status='ACTIVE')`, tenant.AppID, tenant.MerchantID, departmentID).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return ErrInvalid
			}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM platform_subject_role WHERE app_id=$1 AND merchant_id=$2 AND subject=$3`, tenant.AppID, tenant.MerchantID, subject); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM platform_subject_department WHERE app_id=$1 AND merchant_id=$2 AND subject=$3`, tenant.AppID, tenant.MerchantID, subject); err != nil {
			return err
		}
		for _, roleID := range assignment.RoleIDs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO platform_subject_role(app_id,merchant_id,subject,role_id)VALUES($1,$2,$3,$4)`, tenant.AppID, tenant.MerchantID, subject, roleID); err != nil {
				return err
			}
		}
		for _, item := range assignment.Departments {
			if _, err := tx.ExecContext(ctx, `INSERT INTO platform_subject_department(app_id,merchant_id,subject,department_id,is_primary)VALUES($1,$2,$3,$4,$5)`, tenant.AppID, tenant.MerchantID, subject, item.DepartmentID, item.Primary); err != nil {
				return err
			}
		}
		if err := bumpRevision(ctx, tx, tenant); err != nil {
			return err
		}
		return auditIAMSubjectMutation(ctx, tx, tenant, "iam.subject.assignment.put", subject, assignment)
	})
}

func fmtAssignment(value SubjectAssignment) string {
	value.RoleIDs = uniqueInt64s(value.RoleIDs)
	sort.Slice(value.Departments, func(i, j int) bool { return value.Departments[i].DepartmentID < value.Departments[j].DepartmentID })
	var builder strings.Builder
	for _, id := range value.RoleIDs {
		builder.WriteString(strconv.FormatInt(id, 10))
		builder.WriteByte(',')
	}
	builder.WriteByte('|')
	for _, item := range value.Departments {
		builder.WriteString(strconv.FormatInt(item.DepartmentID, 10))
		if item.Primary {
			builder.WriteByte('p')
		}
		builder.WriteByte(',')
	}
	return builder.String()
}

func subjectAssignmentSQL(ctx context.Context, tx *sql.Tx, tenant Tenant, subject string) (SubjectAssignment, error) {
	var output SubjectAssignment
	rows, err := tx.QueryContext(ctx, `SELECT role_id FROM platform_subject_role WHERE app_id=$1 AND merchant_id=$2 AND subject=$3 ORDER BY role_id`, tenant.AppID, tenant.MerchantID, subject)
	if err != nil {
		return output, err
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return output, err
		}
		output.RoleIDs = append(output.RoleIDs, id)
	}
	if err := rows.Close(); err != nil {
		return output, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT department_id,is_primary FROM platform_subject_department WHERE app_id=$1 AND merchant_id=$2 AND subject=$3 ORDER BY department_id`, tenant.AppID, tenant.MerchantID, subject)
	if err != nil {
		return output, err
	}
	for rows.Next() {
		var item DepartmentMembership
		if err := rows.Scan(&item.DepartmentID, &item.Primary); err != nil {
			rows.Close()
			return output, err
		}
		output.Departments = append(output.Departments, item)
	}
	return output, rows.Close()
}

func bumpRevision(ctx context.Context, tx *sql.Tx, tenant Tenant) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO platform_iam_revision(app_id,merchant_id,revision)VALUES($1,$2,1) ON CONFLICT(app_id,merchant_id)DO UPDATE SET revision=platform_iam_revision.revision+1,updated_at=NOW()`, tenant.AppID, tenant.MerchantID)
	return err
}

func (s *Store) permissionsSQL(ctx context.Context) ([]Permission, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT module_id,permission_code,name,resource_code,action,description FROM platform_permission_catalog WHERE active=TRUE ORDER BY permission_code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var output []Permission
	for rows.Next() {
		var item Permission
		if err := rows.Scan(&item.ModuleID, &item.Code, &item.Name, &item.Resource, &item.Action, &item.Description); err != nil {
			return nil, err
		}
		output = append(output, item)
	}
	return output, rows.Err()
}
func (s *Store) departmentsSQL(ctx context.Context, tenant Tenant) ([]Department, error) {
	if !tenant.Valid() {
		return nil, ErrInvalid
	}
	rows, err := s.db.QueryContext(ctx, `SELECT department_id,parent_department_id,name,status,version FROM platform_department WHERE app_id=$1 AND merchant_id=$2 ORDER BY department_id`, tenant.AppID, tenant.MerchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var output []Department
	for rows.Next() {
		var item Department
		var parent sql.NullInt64
		if err := rows.Scan(&item.ID, &parent, &item.Name, &item.Status, &item.Version); err != nil {
			return nil, err
		}
		if parent.Valid {
			item.ParentID = &parent.Int64
		}
		output = append(output, item)
	}
	return output, rows.Err()
}
func (s *Store) rolesSQL(ctx context.Context, tenant Tenant) ([]Role, error) {
	if !tenant.Valid() {
		return nil, ErrInvalid
	}
	rows, err := s.db.QueryContext(ctx, `SELECT role_id,name,status,is_super_admin,version FROM platform_role WHERE app_id=$1 AND merchant_id=$2 ORDER BY role_id`, tenant.AppID, tenant.MerchantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var output []Role
	for rows.Next() {
		var item Role
		if err := rows.Scan(&item.ID, &item.Name, &item.Status, &item.SuperAdmin, &item.Version); err != nil {
			return nil, err
		}
		output = append(output, item)
	}
	return output, rows.Err()
}

func (s *Store) effectiveSQL(ctx context.Context, tenant Tenant, subject string) (Authorization, error) {
	if !tenant.Valid() || strings.TrimSpace(subject) == "" {
		return Authorization{}, ErrInvalid
	}
	permissionsList, err := s.permissionsSQL(ctx)
	if err != nil {
		return Authorization{}, err
	}
	catalog := map[string]Permission{}
	for _, item := range permissionsList {
		catalog[item.Code] = item
	}
	var revision uint64
	err = s.db.QueryRowContext(ctx, `SELECT revision FROM platform_iam_revision WHERE app_id=$1 AND merchant_id=$2`, tenant.AppID, tenant.MerchantID).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		revision = 1
	} else if err != nil {
		return Authorization{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT r.role_id,r.is_super_admin FROM platform_role r JOIN platform_subject_role sr ON sr.app_id=r.app_id AND sr.merchant_id=r.merchant_id AND sr.role_id=r.role_id WHERE r.app_id=$1 AND r.merchant_id=$2 AND sr.subject=$3 AND r.status='ACTIVE'`, tenant.AppID, tenant.MerchantID, subject)
	if err != nil {
		return Authorization{}, err
	}
	var roleIDs []int64
	super := false
	for rows.Next() {
		var id int64
		var current bool
		if err := rows.Scan(&id, &current); err != nil {
			rows.Close()
			return Authorization{}, err
		}
		roleIDs = append(roleIDs, id)
		super = super || current
	}
	if err := rows.Close(); err != nil {
		return Authorization{}, err
	}
	granted := map[string]Permission{}
	var rules []ScopeRule
	if super {
		for code, item := range catalog {
			granted[code] = item
			rules = append(rules, ScopeRule{Resource: item.Resource, Type: ScopeAll})
		}
	} else if len(roleIDs) > 0 {
		rows, err = s.db.QueryContext(ctx, `SELECT DISTINCT p.permission_code FROM platform_role_permission p JOIN platform_permission_catalog c ON c.permission_code=p.permission_code AND c.active=TRUE JOIN platform_subject_role sr ON sr.app_id=p.app_id AND sr.merchant_id=p.merchant_id AND sr.role_id=p.role_id JOIN platform_role r ON r.app_id=p.app_id AND r.merchant_id=p.merchant_id AND r.role_id=p.role_id WHERE p.app_id=$1 AND p.merchant_id=$2 AND sr.subject=$3 AND r.status='ACTIVE'`, tenant.AppID, tenant.MerchantID, subject)
		if err != nil {
			return Authorization{}, err
		}
		for rows.Next() {
			var code string
			if err := rows.Scan(&code); err != nil {
				rows.Close()
				return Authorization{}, err
			}
			if item, ok := catalog[code]; ok {
				granted[code] = item
			}
		}
		if err := rows.Close(); err != nil {
			return Authorization{}, err
		}
		rows, err = s.db.QueryContext(ctx, `SELECT s.resource_code,s.scope_type,d.department_id FROM platform_role_data_scope s JOIN platform_subject_role sr ON sr.app_id=s.app_id AND sr.merchant_id=s.merchant_id AND sr.role_id=s.role_id JOIN platform_role r ON r.app_id=s.app_id AND r.merchant_id=s.merchant_id AND r.role_id=s.role_id LEFT JOIN platform_role_scope_department d ON d.app_id=s.app_id AND d.merchant_id=s.merchant_id AND d.role_id=s.role_id AND d.resource_code=s.resource_code WHERE s.app_id=$1 AND s.merchant_id=$2 AND sr.subject=$3 AND r.status='ACTIVE' ORDER BY s.resource_code`, tenant.AppID, tenant.MerchantID, subject)
		if err != nil {
			return Authorization{}, err
		}
		for rows.Next() {
			var resource, scopeType string
			var department sql.NullInt64
			if err := rows.Scan(&resource, &scopeType, &department); err != nil {
				rows.Close()
				return Authorization{}, err
			}
			rule := ScopeRule{Resource: resource, Type: scopeType}
			if department.Valid {
				rule.DepartmentIDs = []int64{department.Int64}
			}
			rules = append(rules, rule)
		}
		if err := rows.Close(); err != nil {
			return Authorization{}, err
		}
	}
	rows, err = s.db.QueryContext(ctx, `SELECT department_id,is_primary FROM platform_subject_department WHERE app_id=$1 AND merchant_id=$2 AND subject=$3`, tenant.AppID, tenant.MerchantID, subject)
	if err != nil {
		return Authorization{}, err
	}
	var memberships []DepartmentMembership
	for rows.Next() {
		var item DepartmentMembership
		if err := rows.Scan(&item.DepartmentID, &item.Primary); err != nil {
			rows.Close()
			return Authorization{}, err
		}
		memberships = append(memberships, item)
	}
	if err := rows.Close(); err != nil {
		return Authorization{}, err
	}
	departmentsList, err := s.departmentsSQL(ctx, tenant)
	if err != nil {
		return Authorization{}, err
	}
	departments := map[int64]Department{}
	for _, item := range departmentsList {
		departments[item.ID] = item
	}
	return resolveAuthorization(revision, subject, granted, rules, memberships, departments), nil
}
