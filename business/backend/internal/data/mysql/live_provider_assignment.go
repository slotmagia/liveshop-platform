package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	providermodel "github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider/model"
)

func (r *LiveProviderRepository) GetAssignments(ctx context.Context, merchantID int64) (providermodel.AssignmentSet, error) {
	return readAssignments(ctx, r.db, merchantID)
}

func (r *LiveProviderRepository) PutAssignments(ctx context.Context, input providermodel.PutAssignments) (providermodel.AssignmentSet, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return providermodel.AssignmentSet{}, err
	}
	defer func() { _ = tx.Rollback() }()
	hash := providermodel.RequestHash(input)
	_, err = tx.ExecContext(ctx, `INSERT INTO live_provider_assignment_command(command_key,request_hash) VALUES(?,?)`, input.CommandKey, hash)
	if isMySQLDuplicate(err) {
		var stored string
		var document []byte
		if err := tx.QueryRowContext(ctx, `SELECT request_hash,response_json FROM live_provider_assignment_command WHERE command_key=? FOR UPDATE`, input.CommandKey).
			Scan(&stored, &document); err != nil {
			return providermodel.AssignmentSet{}, err
		}
		if stored != hash {
			return providermodel.AssignmentSet{}, providermodel.ErrConflict
		}
		var replay providermodel.AssignmentSet
		if json.Unmarshal(document, &replay) != nil {
			return providermodel.AssignmentSet{}, fmt.Errorf("live provider assignment replay incomplete")
		}
		if err := tx.Commit(); err != nil {
			return providermodel.AssignmentSet{}, err
		}
		return replay, nil
	}
	if err != nil {
		return providermodel.AssignmentSet{}, err
	}
	var current int64
	err = tx.QueryRowContext(ctx, `SELECT version FROM live_provider_assignment_state WHERE merchant_id=? FOR UPDATE`, input.MerchantID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		if input.ExpectedVersion != 0 {
			return providermodel.AssignmentSet{}, providermodel.ErrConflict
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO live_provider_assignment_state(merchant_id,version) VALUES(?,1)`, input.MerchantID); err != nil {
			return providermodel.AssignmentSet{}, err
		}
	} else if err != nil {
		return providermodel.AssignmentSet{}, err
	} else if current != input.ExpectedVersion {
		return providermodel.AssignmentSet{}, providermodel.ErrConflict
	} else if _, err := tx.ExecContext(ctx, `UPDATE live_provider_assignment_state SET version=version+1 WHERE merchant_id=? AND version=?`, input.MerchantID, input.ExpectedVersion); err != nil {
		return providermodel.AssignmentSet{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM live_provider_assignment WHERE merchant_id=?`, input.MerchantID); err != nil {
		return providermodel.AssignmentSet{}, err
	}
	for _, item := range input.Providers {
		if _, err := tx.ExecContext(ctx, `INSERT INTO live_provider_assignment(merchant_id,provider_code,enabled,is_default,version) VALUES(?,?,?,?,1)`,
			input.MerchantID, item.ProviderCode, boolToInt(item.Enabled), boolToInt(item.Default)); err != nil {
			return providermodel.AssignmentSet{}, err
		}
	}
	saved, err := readAssignments(ctx, tx, input.MerchantID)
	if err != nil {
		return providermodel.AssignmentSet{}, err
	}
	document, err := json.Marshal(saved)
	if err != nil {
		return providermodel.AssignmentSet{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE live_provider_assignment_command SET merchant_id=?,response_version=?,response_json=?,completed_at=CURRENT_TIMESTAMP(3) WHERE command_key=?`,
		saved.MerchantID, saved.Version, document, input.CommandKey); err != nil {
		return providermodel.AssignmentSet{}, err
	}
	if err := tx.Commit(); err != nil {
		return providermodel.AssignmentSet{}, err
	}
	return saved, nil
}

func readAssignments(ctx context.Context, query interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, merchantID int64) (providermodel.AssignmentSet, error) {
	out := providermodel.AssignmentSet{MerchantID: merchantID, Providers: []providermodel.Assignment{}}
	if err := query.QueryRowContext(ctx, `SELECT version FROM live_provider_assignment_state WHERE merchant_id=?`, merchantID).Scan(&out.Version); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return providermodel.AssignmentSet{}, err
	}
	rows, err := query.QueryContext(ctx, `SELECT a.provider_code,COALESCE(p.name,a.provider_code),a.enabled,a.is_default
FROM live_provider_assignment a
LEFT JOIN live_provider p ON p.code=a.provider_code
WHERE a.merchant_id=? ORDER BY a.provider_code`, merchantID)
	if err != nil {
		return providermodel.AssignmentSet{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var item providermodel.Assignment
		var enabled, isDefault int
		if err := rows.Scan(&item.ProviderCode, &item.Name, &enabled, &isDefault); err != nil {
			return providermodel.AssignmentSet{}, err
		}
		item.Enabled = enabled == 1
		item.Default = isDefault == 1
		out.Providers = append(out.Providers, item)
	}
	return out, rows.Err()
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func isMySQLDuplicate(err error) bool {
	var mysqlError *mysql.MySQLError
	return errors.As(err, &mysqlError) && mysqlError.Number == 1062
}
