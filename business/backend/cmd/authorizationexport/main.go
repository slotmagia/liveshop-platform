// Command authorizationexport performs the one-time, fail-closed handoff of
// retired Platform browser-user authorization data. It writes a canonical JSON
// bundle and immutable digest before optionally dropping the source tables.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	schemaVersion  = 1
	exportLockName = "liveshop-platform.authorizationexport"
)

type tableSpec struct {
	Name    string
	OrderBy string
}

var authorizationTables = []tableSpec{
	{Name: "platform_authorization_domain", OrderBy: "domain_type,domain_id"},
	{Name: "platform_authorization_role", OrderBy: "domain_type,domain_id,role_id"},
	{Name: "platform_authorization_role_permission", OrderBy: "domain_type,domain_id,role_id,permission_code"},
	{Name: "platform_authorization_role_scope", OrderBy: "domain_type,domain_id,role_id,resource_code,scope_type"},
	{Name: "platform_subject_grant", OrderBy: "grant_id"},
	{Name: "platform_authorization_operation", OrderBy: "operation_id"},
	{Name: "platform_business_delegation", OrderBy: "domain_type,domain_id,subject,scope_type,merchant_id,shop_id"},
	{Name: "platform_entitlement_projection", OrderBy: "merchant_id,permission_code"},
	{Name: "platform_inbox", OrderBy: "event_id"},
	{Name: "platform_outbox", OrderBy: "event_id"},
	{Name: "platform_role", OrderBy: "id"},
	{Name: "platform_role_permission", OrderBy: "role_id,permission_code"},
	{Name: "platform_role_data_scope", OrderBy: "role_id,resource_code"},
	{Name: "platform_role_scope_department", OrderBy: "role_id,resource_code,department_id"},
	{Name: "platform_subject_role", OrderBy: "subject,role_id"},
	{Name: "platform_subject_department", OrderBy: "subject,department_id"},
}

type bundle struct {
	SchemaVersion int           `json:"schemaVersion"`
	Tables        []tableBundle `json:"tables"`
}

type tableBundle struct {
	Name    string           `json:"name"`
	Present bool             `json:"present"`
	Rows    []map[string]any `json:"rows"`
}

type envelope struct {
	SchemaVersion int             `json:"schemaVersion"`
	SHA256        string          `json:"sha256"`
	RowCount      int64           `json:"rowCount"`
	Payload       json.RawMessage `json:"payload"`
}

func main() {
	dsn := flag.String("dsn", "", "MySQL DSN containing the retired Platform authorization tables")
	output := flag.String("output", "", "destination JSON bundle; must be outside the deployment image")
	finalize := flag.Bool("finalize", false, "drop retired authorization tables after durable export")
	identityReceipt := flag.String("identity-receipt", "", "Identity import receipt JSON required with -finalize")
	receiptKeyID := flag.String("identity-receipt-key-id", "", "expected key id for the signed Identity import receipt")
	receiptPublicKey := flag.String("identity-receipt-public-key", "", "base64url Ed25519 public key for the signed Identity import receipt")
	identityInstance := flag.String("identity-instance", "", "expected target Identity instance in the import receipt")
	identitySchemaVersion := flag.Int("identity-schema-version", 0, "expected target Identity authorization schema version")
	subscriptionReceipt := flag.String("subscription-receipt", "", "Subscription import receipt JSON required with -finalize")
	subscriptionReceiptKeyID := flag.String("subscription-receipt-key-id", "", "expected key id for the signed Subscription import receipt")
	subscriptionReceiptPublicKey := flag.String("subscription-receipt-public-key", "", "base64url Ed25519 public key for the signed Subscription import receipt")
	subscriptionInstance := flag.String("subscription-instance", "", "expected target Subscription instance")
	subscriptionSchemaVersion := flag.Int("subscription-schema-version", 0, "expected target Subscription schema version")
	flag.Parse()
	if strings.TrimSpace(*dsn) == "" || strings.TrimSpace(*output) == "" {
		fatal(errors.New("authorizationexport: -dsn and -output are required"))
	}
	db, err := sql.Open("mysql", *dsn)
	if err != nil {
		fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := run(ctx, db, *output, *finalize, *identityReceipt, *receiptKeyID, *receiptPublicKey, *identityInstance, *identitySchemaVersion, *subscriptionReceipt, *subscriptionReceiptKeyID, *subscriptionReceiptPublicKey, *subscriptionInstance, *subscriptionSchemaVersion); err != nil {
		fatal(err)
	}
}

func run(ctx context.Context, db *sql.DB, output string, finalize bool, identityReceipt, receiptKeyID, receiptPublicKey, identityInstance string, identitySchemaVersion int, subscriptionReceipt, subscriptionReceiptKeyID, subscriptionReceiptPublicKey, subscriptionInstance string, subscriptionSchemaVersion int) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := acquireExportLock(ctx, conn); err != nil {
		return err
	}
	defer releaseExportLock(conn)

	payload, digest, rowCount, finalized, found, err := readLedger(ctx, conn)
	if err != nil {
		return fmt.Errorf("read export ledger: %w", err)
	}
	if !found {
		payload, digest, rowCount, err = captureAndRecord(ctx, conn)
		if err != nil {
			return err
		}
	} else if digestOf(payload) != digest {
		return errors.New("authorizationexport: persisted export payload digest mismatch")
	}
	if err := writeEnvelope(output, payload, digest, rowCount); err != nil {
		return err
	}
	if !finalize {
		fmt.Printf("authorization export sha256=%s rows=%d finalized=%v\n", digest, rowCount, finalized)
		return nil
	}
	receipt, err := verifyIdentityReceipt(identityReceipt, receiptKeyID, receiptPublicKey, identityInstance, identitySchemaVersion, digest, rowCount)
	if err != nil {
		return err
	}
	canonicalReceipt, _ := json.Marshal(receipt)
	receiptDigest := digestOf(canonicalReceipt)
	if err := acknowledgeIdentityReceipt(ctx, conn, receipt, canonicalReceipt, receiptDigest); err != nil {
		return err
	}
	subscription, err := verifySubscriptionReceipt(subscriptionReceipt, subscriptionReceiptKeyID, subscriptionReceiptPublicKey, subscriptionInstance, subscriptionSchemaVersion, digest, rowCount)
	if err != nil {
		return err
	}
	canonicalSubscriptionReceipt, _ := json.Marshal(subscription)
	if err := acknowledgeSubscriptionReceipt(ctx, conn, subscription, canonicalSubscriptionReceipt, digestOf(canonicalSubscriptionReceipt)); err != nil {
		return err
	}
	if finalized {
		fmt.Printf("authorization export sha256=%s rows=%d finalized=true\n", digest, rowCount)
		return nil
	}
	if err := dropRetiredTables(ctx, conn); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `UPDATE platform_authorization_export_ledger SET finalized_at=COALESCE(finalized_at,NOW(3)) WHERE singleton_id=1 AND payload_sha256=?`, digest); err != nil {
		return fmt.Errorf("finalize export ledger: %w", err)
	}
	fmt.Printf("authorization export sha256=%s rows=%d finalized=true\n", digest, rowCount)
	return nil
}

func acquireExportLock(ctx context.Context, conn *sql.Conn) error {
	var acquired sql.NullInt64
	if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK(?, 30)`, exportLockName).Scan(&acquired); err != nil {
		return fmt.Errorf("authorizationexport: acquire exclusive handoff lock: %w", err)
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		return errors.New("authorizationexport: another export/finalize process holds the handoff lock")
	}
	return nil
}

func releaseExportLock(conn *sql.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = conn.ExecContext(ctx, `SELECT RELEASE_LOCK(?)`, exportLockName)
}

type importReceipt struct {
	SchemaVersion               int    `json:"schemaVersion"`
	Source                      string `json:"source"`
	ImportID                    string `json:"importId"`
	SHA256                      string `json:"sha256"`
	RowCount                    int64  `json:"rowCount"`
	Imported                    bool   `json:"imported"`
	ImportedAt                  string `json:"importedAt"`
	TargetIdentityInstance      string `json:"targetIdentityInstance"`
	TargetIdentitySchemaVersion int    `json:"targetIdentitySchemaVersion"`
	KeyID                       string `json:"keyId"`
	Signature                   string `json:"signature"`
}

type subscriptionImportReceipt struct {
	SchemaVersion                   int    `json:"schemaVersion"`
	Source                          string `json:"source"`
	ImportID                        string `json:"importId"`
	SHA256                          string `json:"sha256"`
	RowCount                        int64  `json:"rowCount"`
	Imported                        bool   `json:"imported"`
	ImportedAt                      string `json:"importedAt"`
	TargetSubscriptionInstance      string `json:"targetSubscriptionInstance"`
	TargetSubscriptionSchemaVersion int    `json:"targetSubscriptionSchemaVersion"`
	TargetImportedRowCount          int64  `json:"targetImportedRowCount"`
	TargetProjectionDigest          string `json:"targetProjectionDigest"`
	KeyID                           string `json:"keyId"`
	Signature                       string `json:"signature"`
}

func verifySubscriptionReceipt(path, expectedKeyID, encodedPublicKey, expectedInstance string, expectedSchemaVersion int, digest string, rowCount int64) (subscriptionImportReceipt, error) {
	if strings.TrimSpace(path) == "" {
		return subscriptionImportReceipt{}, errors.New("authorizationexport: -subscription-receipt is required with -finalize")
	}
	if expectedKeyID == "" || encodedPublicKey == "" || expectedInstance == "" || expectedSchemaVersion <= 0 {
		return subscriptionImportReceipt{}, errors.New("authorizationexport: signed Subscription receipt trust and target schema are required with -finalize")
	}
	document, err := os.ReadFile(path)
	if err != nil {
		return subscriptionImportReceipt{}, fmt.Errorf("read Subscription import receipt: %w", err)
	}
	var receipt subscriptionImportReceipt
	decoder := json.NewDecoder(strings.NewReader(string(document)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return receipt, fmt.Errorf("decode Subscription import receipt: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return receipt, errors.New("authorizationexport: Subscription import receipt contains trailing data")
	}
	if receipt.SchemaVersion != schemaVersion || receipt.Source != "liveshop-platform-authorization" || receipt.ImportID == "" || !receipt.Imported || receipt.SHA256 != digest || receipt.RowCount != rowCount || receipt.KeyID != expectedKeyID || receipt.TargetSubscriptionInstance != expectedInstance || receipt.TargetSubscriptionSchemaVersion != expectedSchemaVersion || receipt.TargetImportedRowCount < 0 || len(receipt.TargetProjectionDigest) != 64 {
		return receipt, errors.New("authorizationexport: Subscription import receipt does not acknowledge the exact exported authorization bundle and target projection")
	}
	if _, err := time.Parse(time.RFC3339Nano, receipt.ImportedAt); err != nil {
		return receipt, errors.New("authorizationexport: Subscription import receipt has an invalid importedAt")
	}
	publicKey, err := base64.RawURLEncoding.DecodeString(encodedPublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return receipt, errors.New("authorizationexport: invalid Subscription receipt public key")
	}
	signature, err := base64.RawURLEncoding.DecodeString(receipt.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize || !ed25519.Verify(ed25519.PublicKey(publicKey), subscriptionReceiptSigningInput(receipt), signature) {
		return receipt, errors.New("authorizationexport: Subscription import receipt signature is invalid")
	}
	return receipt, nil
}

func subscriptionReceiptSigningInput(receipt subscriptionImportReceipt) []byte {
	type signedFields struct {
		SchemaVersion                   int    `json:"schemaVersion"`
		Source                          string `json:"source"`
		ImportID                        string `json:"importId"`
		SHA256                          string `json:"sha256"`
		RowCount                        int64  `json:"rowCount"`
		Imported                        bool   `json:"imported"`
		ImportedAt                      string `json:"importedAt"`
		TargetSubscriptionInstance      string `json:"targetSubscriptionInstance"`
		TargetSubscriptionSchemaVersion int    `json:"targetSubscriptionSchemaVersion"`
		TargetImportedRowCount          int64  `json:"targetImportedRowCount"`
		TargetProjectionDigest          string `json:"targetProjectionDigest"`
		KeyID                           string `json:"keyId"`
	}
	payload, _ := json.Marshal(signedFields{receipt.SchemaVersion, receipt.Source, receipt.ImportID, receipt.SHA256, receipt.RowCount, receipt.Imported, receipt.ImportedAt, receipt.TargetSubscriptionInstance, receipt.TargetSubscriptionSchemaVersion, receipt.TargetImportedRowCount, receipt.TargetProjectionDigest, receipt.KeyID})
	return payload
}

// verifyIdentityReceipt closes the destructive handoff window: Platform only
// finalizes after Identity has durably imported this exact digest and row
// count. The receipt is an explicit two-phase operator artifact because the
// two service databases cannot share a transaction.
func verifyIdentityReceipt(path, expectedKeyID, encodedPublicKey, expectedIdentityInstance string, expectedIdentitySchemaVersion int, digest string, rowCount int64) (importReceipt, error) {
	if strings.TrimSpace(path) == "" {
		return importReceipt{}, errors.New("authorizationexport: -identity-receipt is required with -finalize")
	}
	if strings.TrimSpace(expectedKeyID) == "" || strings.TrimSpace(encodedPublicKey) == "" || strings.TrimSpace(expectedIdentityInstance) == "" || expectedIdentitySchemaVersion <= 0 {
		return importReceipt{}, errors.New("authorizationexport: signed Identity receipt trust and target schema are required with -finalize")
	}
	document, err := os.ReadFile(path)
	if err != nil {
		return importReceipt{}, fmt.Errorf("read Identity import receipt: %w", err)
	}
	var receipt importReceipt
	decoder := json.NewDecoder(strings.NewReader(string(document)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return importReceipt{}, fmt.Errorf("decode Identity import receipt: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return importReceipt{}, errors.New("authorizationexport: Identity import receipt contains trailing data")
	}
	if receipt.SchemaVersion != schemaVersion || receipt.Source != "liveshop-platform-authorization" || strings.TrimSpace(receipt.ImportID) == "" || !receipt.Imported || receipt.SHA256 != digest || receipt.RowCount != rowCount || receipt.KeyID != expectedKeyID || receipt.TargetIdentityInstance != expectedIdentityInstance || receipt.TargetIdentitySchemaVersion != expectedIdentitySchemaVersion {
		return importReceipt{}, errors.New("authorizationexport: Identity import receipt does not acknowledge the exact exported authorization bundle and target")
	}
	if _, err := time.Parse(time.RFC3339Nano, receipt.ImportedAt); err != nil {
		return importReceipt{}, errors.New("authorizationexport: Identity import receipt has an invalid importedAt")
	}
	publicKey, err := base64.RawURLEncoding.DecodeString(encodedPublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return importReceipt{}, errors.New("authorizationexport: invalid Identity receipt public key")
	}
	signature, err := base64.RawURLEncoding.DecodeString(receipt.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize || !ed25519.Verify(ed25519.PublicKey(publicKey), receiptSigningInput(receipt), signature) {
		return importReceipt{}, errors.New("authorizationexport: Identity import receipt signature is invalid")
	}
	return receipt, nil
}

func receiptSigningInput(receipt importReceipt) []byte {
	type signedFields struct {
		SchemaVersion               int    `json:"schemaVersion"`
		Source                      string `json:"source"`
		ImportID                    string `json:"importId"`
		SHA256                      string `json:"sha256"`
		RowCount                    int64  `json:"rowCount"`
		Imported                    bool   `json:"imported"`
		ImportedAt                  string `json:"importedAt"`
		TargetIdentityInstance      string `json:"targetIdentityInstance"`
		TargetIdentitySchemaVersion int    `json:"targetIdentitySchemaVersion"`
		KeyID                       string `json:"keyId"`
	}
	payload, _ := json.Marshal(signedFields{receipt.SchemaVersion, receipt.Source, receipt.ImportID, receipt.SHA256, receipt.RowCount, receipt.Imported, receipt.ImportedAt, receipt.TargetIdentityInstance, receipt.TargetIdentitySchemaVersion, receipt.KeyID})
	return payload
}

// captureAndRecord freezes every legacy authorization table until the exact
// canonical payload has been written to the immutable Platform ledger. The
// old IAM runtime must already be stopped before this command starts; after
// this point the ledger, not another live read, is the export source of truth.
func captureAndRecord(ctx context.Context, conn *sql.Conn) ([]byte, string, int64, error) {
	present, err := discoverAuthorizationTables(ctx, conn)
	if err != nil {
		return nil, "", 0, err
	}
	if len(present) == 0 {
		return nil, "", 0, errors.New("authorizationexport: no retired authorization tables exist in this database")
	}
	locks := make([]string, 0, len(authorizationTables)+1)
	for _, table := range authorizationTables {
		if present[table.Name] {
			locks = append(locks, "`"+table.Name+"` READ")
		}
	}
	locks = append(locks, "`platform_authorization_export_ledger` WRITE")
	if _, err := conn.ExecContext(ctx, "LOCK TABLES "+strings.Join(locks, ",")); err != nil {
		return nil, "", 0, fmt.Errorf("lock retired authorization facts: %w", err)
	}
	defer conn.ExecContext(context.Background(), "UNLOCK TABLES")
	var existing int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM platform_authorization_export_ledger WHERE singleton_id=1`).Scan(&existing); err != nil {
		return nil, "", 0, fmt.Errorf("recheck export ledger under lock: %w", err)
	}
	if existing != 0 {
		return nil, "", 0, errors.New("authorizationexport: immutable export ledger appeared concurrently")
	}

	result := bundle{SchemaVersion: schemaVersion, Tables: make([]tableBundle, 0, len(authorizationTables))}
	var count int64
	for _, table := range authorizationTables {
		if !present[table.Name] {
			result.Tables = append(result.Tables, tableBundle{Name: table.Name, Present: false, Rows: []map[string]any{}})
			continue
		}
		rows, err := readTable(ctx, conn, table)
		if err != nil {
			return nil, "", 0, err
		}
		count += int64(len(rows))
		result.Tables = append(result.Tables, tableBundle{Name: table.Name, Present: true, Rows: rows})
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, "", 0, err
	}
	digest := digestOf(payload)
	if _, err := conn.ExecContext(ctx, `INSERT INTO platform_authorization_export_ledger(singleton_id,schema_version,payload_sha256,row_count,payload) VALUES(1,?,?,?,?)`, schemaVersion, digest, count, payload); err != nil {
		return nil, "", 0, fmt.Errorf("record immutable export ledger: %w", err)
	}
	return payload, digest, count, nil
}

// discoverAuthorizationTables permits upgrades from different historical
// generations while making absence explicit in the signed payload. A table
// discovered as present is subsequently locked and read; disappearance or a
// query error is fatal and can never be normalized to an empty table.
func discoverAuthorizationTables(ctx context.Context, conn *sql.Conn) (map[string]bool, error) {
	placeholders := make([]string, len(authorizationTables))
	args := make([]any, len(authorizationTables))
	for i, table := range authorizationTables {
		placeholders[i] = "?"
		args[i] = table.Name
	}
	rows, err := conn.QueryContext(ctx, `SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("discover retired authorization tables: %w", err)
	}
	defer rows.Close()
	present := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		present[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return present, nil
}

type receiptEvidence struct {
	SchemaVersion               int
	Source                      string
	ImportID                    string
	SHA256                      string
	RowCount                    int64
	ImportedAt                  string
	TargetIdentityInstance      string
	TargetIdentitySchemaVersion int
	KeyID                       string
	ReceiptSHA256               string
}

func evidenceFrom(receipt importReceipt, receiptDigest string) (receiptEvidence, error) {
	importedAt, err := time.Parse(time.RFC3339Nano, receipt.ImportedAt)
	if err != nil {
		return receiptEvidence{}, errors.New("authorizationexport: Identity import receipt has an invalid importedAt")
	}
	return receiptEvidence{
		SchemaVersion: receipt.SchemaVersion, Source: receipt.Source, ImportID: receipt.ImportID,
		SHA256: receipt.SHA256, RowCount: receipt.RowCount, ImportedAt: importedAt.UTC().Format(time.RFC3339Nano),
		TargetIdentityInstance:      receipt.TargetIdentityInstance,
		TargetIdentitySchemaVersion: receipt.TargetIdentitySchemaVersion,
		KeyID:                       receipt.KeyID, ReceiptSHA256: receiptDigest,
	}, nil
}

func (e receiptEvidence) same(other receiptEvidence) bool {
	return e.SchemaVersion == other.SchemaVersion && e.Source == other.Source && e.ImportID == other.ImportID &&
		e.SHA256 == other.SHA256 && e.RowCount == other.RowCount && e.ImportedAt == other.ImportedAt &&
		e.TargetIdentityInstance == other.TargetIdentityInstance && e.TargetIdentitySchemaVersion == other.TargetIdentitySchemaVersion &&
		e.KeyID == other.KeyID && e.ReceiptSHA256 == other.ReceiptSHA256
}

func acknowledgeIdentityReceipt(ctx context.Context, conn *sql.Conn, receipt importReceipt, canonicalReceipt []byte, receiptDigest string) error {
	expected, err := evidenceFrom(receipt, receiptDigest)
	if err != nil {
		return err
	}
	var current receiptEvidence
	err = conn.QueryRowContext(ctx, `SELECT receipt_schema_version,source,import_id,payload_sha256,row_count,imported_at,target_identity_instance,target_identity_schema_version,key_id,receipt_sha256 FROM platform_authorization_import_receipt WHERE singleton_id=1`).Scan(
		&current.SchemaVersion, &current.Source, &current.ImportID, &current.SHA256, &current.RowCount, &current.ImportedAt,
		&current.TargetIdentityInstance, &current.TargetIdentitySchemaVersion, &current.KeyID, &current.ReceiptSHA256,
	)
	if err == nil {
		if !current.same(expected) {
			return errors.New("authorizationexport: a different Identity import receipt was already acknowledged")
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read acknowledged Identity receipt: %w", err)
	}
	result, err := conn.ExecContext(ctx, `INSERT INTO platform_authorization_import_receipt(singleton_id,receipt_schema_version,source,import_id,payload_sha256,row_count,imported_at,target_identity_instance,target_identity_schema_version,key_id,receipt_sha256,receipt) VALUES(1,?,?,?,?,?,?,?,?,?,?,?)`,
		expected.SchemaVersion, expected.Source, expected.ImportID, expected.SHA256, expected.RowCount, expected.ImportedAt,
		expected.TargetIdentityInstance, expected.TargetIdentitySchemaVersion, expected.KeyID, expected.ReceiptSHA256, canonicalReceipt)
	if err != nil {
		return fmt.Errorf("record verified Identity receipt: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return errors.New("authorizationexport: verified Identity receipt was not recorded")
	}
	return nil
}

func acknowledgeSubscriptionReceipt(ctx context.Context, conn *sql.Conn, receipt subscriptionImportReceipt, canonicalReceipt []byte, receiptDigest string) error {
	importedAt, err := time.Parse(time.RFC3339Nano, receipt.ImportedAt)
	if err != nil {
		return err
	}
	var currentDigest string
	err = conn.QueryRowContext(ctx, `SELECT receipt_sha256 FROM platform_subscription_authorization_import_receipt WHERE singleton_id=1`).Scan(&currentDigest)
	if err == nil {
		if currentDigest != receiptDigest {
			return errors.New("authorizationexport: a different Subscription import receipt was already acknowledged")
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read acknowledged Subscription receipt: %w", err)
	}
	result, err := conn.ExecContext(ctx, `INSERT INTO platform_subscription_authorization_import_receipt
(singleton_id,receipt_schema_version,source,import_id,payload_sha256,export_row_count,target_imported_row_count,target_projection_digest,imported_at,target_subscription_instance,target_subscription_schema_version,key_id,receipt_sha256,receipt)
VALUES(1,?,?,?,?,?,?,?,?,?,?,?,?,?)`, receipt.SchemaVersion, receipt.Source, receipt.ImportID, receipt.SHA256, receipt.RowCount, receipt.TargetImportedRowCount, receipt.TargetProjectionDigest, importedAt.UTC().Format(time.RFC3339Nano), receipt.TargetSubscriptionInstance, receipt.TargetSubscriptionSchemaVersion, receipt.KeyID, receiptDigest, canonicalReceipt)
	if err != nil {
		return fmt.Errorf("record verified Subscription receipt: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return errors.New("authorizationexport: verified Subscription receipt was not recorded")
	}
	return nil
}

func readTable(ctx context.Context, conn *sql.Conn, table tableSpec) ([]map[string]any, error) {
	rows, err := conn.QueryContext(ctx, "SELECT * FROM `"+table.Name+"` ORDER BY "+table.OrderBy)
	if err != nil {
		return nil, fmt.Errorf("export %s: %w", table.Name, err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	output := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		targets := make([]any, len(columns))
		for i := range values {
			targets[i] = &values[i]
		}
		if err := rows.Scan(targets...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for i, value := range values {
			row[columns[i]] = normalizeValue(value, types[i].DatabaseTypeName())
		}
		output = append(output, row)
	}
	return output, rows.Err()
}

func normalizeValue(value any, databaseType string) any {
	if value == nil {
		return nil
	}
	if current, ok := value.(time.Time); ok {
		return current.UTC().Format(time.RFC3339Nano)
	}
	bytes, ok := value.([]byte)
	if !ok {
		return value
	}
	if strings.EqualFold(databaseType, "JSON") {
		var decoded any
		if json.Unmarshal(bytes, &decoded) == nil {
			return decoded
		}
	}
	return string(bytes)
}

func readLedger(ctx context.Context, conn *sql.Conn) (json.RawMessage, string, int64, bool, bool, error) {
	var payload []byte
	var digest string
	var rowCount int64
	var persistedSchemaVersion int
	var finalizedAt sql.NullTime
	err := conn.QueryRowContext(ctx, `SELECT schema_version,payload,payload_sha256,row_count,finalized_at FROM platform_authorization_export_ledger WHERE singleton_id=1`).Scan(&persistedSchemaVersion, &payload, &digest, &rowCount, &finalizedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", 0, false, false, nil
	}
	if err == nil && persistedSchemaVersion != schemaVersion {
		return nil, "", 0, false, false, fmt.Errorf("authorizationexport: unsupported persisted schema version %d", persistedSchemaVersion)
	}
	return payload, digest, rowCount, finalizedAt.Valid, err == nil, err
}

func writeEnvelope(path string, payload json.RawMessage, digest string, rowCount int64) error {
	document, err := json.MarshalIndent(envelope{SchemaVersion: schemaVersion, SHA256: digest, RowCount: rowCount, Payload: payload}, "", "  ")
	if err != nil {
		return err
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".authorization-export-*.tmp")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(append(document, '\n')); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, path)
}

func dropRetiredTables(ctx context.Context, conn *sql.Conn) error {
	if _, err := conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS=0"); err != nil {
		return err
	}
	defer conn.ExecContext(context.Background(), "SET FOREIGN_KEY_CHECKS=1")
	for i := len(authorizationTables) - 1; i >= 0; i-- {
		if _, err := conn.ExecContext(ctx, "DROP TABLE IF EXISTS `"+authorizationTables[i].Name+"`"); err != nil {
			return fmt.Errorf("drop retired table %s: %w", authorizationTables[i].Name, err)
		}
	}
	return nil
}

func digestOf(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
