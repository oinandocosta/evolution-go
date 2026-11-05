// This opt-in probe only writes synthetic data to the named disposable nc.5 database.
// It uses APIs shared by nc.4 and nc.5; optional new fields are detected at runtime.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"runtime/debug"
	"sort"
	"sync"
	"time"

	"github.com/lib/pq"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

const probeRun = "evogo-nc5-20261104"

var trackedTables = []string{
	"whatsmeow_device", "whatsmeow_identity_keys", "whatsmeow_pre_keys", "whatsmeow_sessions",
	"whatsmeow_sender_keys", "whatsmeow_app_state_sync_keys", "whatsmeow_app_state_version",
	"whatsmeow_app_state_mutation_macs", "whatsmeow_contacts", "whatsmeow_chat_settings",
	"whatsmeow_message_secrets", "whatsmeow_privacy_tokens", "whatsmeow_nct_salt", "whatsmeow_lid_map",
	"whatsmeow_event_buffer", "whatsmeow_retry_buffer", "instances", "messages", "labels",
}

type tableDigest struct {
	Rows       int    `json:"rows"`
	DataSHA256 string `json:"data_sha256"`
	SchemaSHA  string `json:"legacy_columns_sha256"`
}

type manifest struct {
	Run    string                 `json:"run"`
	Tables map[string]tableDigest `json:"tables"`
	NewSHA string                 `json:"new_fields_sha256,omitempty"`
}

func main() {
	report, err := run()
	if err != nil {
		report["passed"] = false
		report["error"] = err.Error()
	} else {
		report["passed"] = true
	}
	_ = json.NewEncoder(os.Stdout).Encode(report)
	if err != nil {
		os.Exit(1)
	}
}

// Never include database error strings: PostgreSQL detail can contain key bytes.
func checked(operation string, err error) error {
	if err != nil {
		return fmt.Errorf("%s failed (%T; details intentionally omitted)", operation, err)
	}
	return nil
}

func run() (map[string]any, error) {
	mode := os.Getenv("EVOGO_PROBE_MODE")
	report := map[string]any{"mode": mode, "run": probeRun, "paired_account": false, "production_data": false}
	if mode != "seed" && mode != "verify" && mode != "new-fields" && mode != "rollback-write" {
		return report, fmt.Errorf("mode must be seed, verify, new-fields or rollback-write")
	}
	dsn := os.Getenv("EVOGO_TEST_POSTGRES_DSN")
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.User == nil || parsed.Hostname() != "evogo-homolog-pg" || parsed.Path != "/evogo_nc5_auth" ||
		parsed.User.Username() != "evogo_homolog" || os.Getenv("EVOGO_TEST_RUN") != probeRun ||
		(parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return report, fmt.Errorf("refusing any target outside the named disposable nc.5 test database")
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dependency := range info.Deps {
			if dependency.Path == "go.mau.fi/whatsmeow" {
				report["whatsmeow"] = dependency.Version
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	if err = checked("open disposable database", err); err != nil {
		return report, err
	}
	defer db.Close()
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)
	sqlstore.PostgresArrayWrapper = pq.Array
	if mode == "seed" {
		var tables int
		err = db.QueryRowContext(ctx, "SELECT count(*) FROM pg_tables WHERE schemaname='public'").Scan(&tables)
		if err = checked("assert empty test database", err); err != nil {
			return report, err
		}
		if tables != 0 {
			return report, fmt.Errorf("seed refuses a nonempty database")
		}
	} else {
		var owner string
		err = db.QueryRowContext(ctx, "SELECT run FROM nc5_probe_manifest WHERE run=$1", probeRun).Scan(&owner)
		if err = checked("assert existing probe ownership marker", err); err != nil {
			return report, err
		}
	}
	container := sqlstore.NewWithDB(db, "postgres", nil)
	if err = checked("upgrade with the library migration runner", container.Upgrade(ctx)); err != nil {
		return report, err
	}
	var version int
	if err = checked("read schema version", db.QueryRowContext(ctx, "SELECT version FROM whatsmeow_version").Scan(&version)); err != nil {
		return report, err
	}
	report["schema_version"] = version
	if mode == "seed" {
		if version != 14 {
			return report, fmt.Errorf("seed must be created by the nc.4 schema-14 dependency")
		}
		if err = seed(ctx, db, container); err != nil {
			return report, err
		}
	}
	want, err := loadManifest(ctx, db)
	if err != nil {
		return report, err
	}
	if mode == "new-fields" {
		if version != 16 {
			return report, fmt.Errorf("new fields require schema 16")
		}
		if err = writeNewFields(ctx, container); err != nil {
			return report, err
		}
		want.NewSHA, err = newFieldsDigest(ctx, db)
		if err != nil {
			return report, err
		}
		if err = saveManifest(ctx, db, want); err != nil {
			return report, err
		}
	}
	if mode == "rollback-write" {
		if version != 16 || want.NewSHA == "" {
			return report, fmt.Errorf("rollback writes require migrated data and new-field fixtures")
		}
		if err = rewriteLegacyFields(ctx, container); err != nil {
			return report, err
		}
	}
	if err = verifyAPIs(ctx, container, want.NewSHA != ""); err != nil {
		return report, err
	}
	actual, err := snapshot(ctx, db)
	if err != nil {
		return report, err
	}
	if !reflect.DeepEqual(actual, want.Tables) {
		return report, fmt.Errorf("legacy data or schema changed; digest comparison failed")
	}
	if want.NewSHA != "" {
		got, err := newFieldsDigest(ctx, db)
		if err != nil {
			return report, err
		}
		if got != want.NewSHA {
			return report, fmt.Errorf("new fields changed after restart or old-code access")
		}
		report["new_fields_sha256"] = got
	}
	// A second migration run and a fresh container simulate idempotent restart.
	if err = checked("repeat upgrade", container.Upgrade(ctx)); err != nil {
		return report, err
	}
	restarted := sqlstore.NewWithDB(db, "postgres", nil)
	if err = checked("upgrade restarted container", restarted.Upgrade(ctx)); err != nil {
		return report, err
	}
	if err = verifyConcurrentReads(ctx, restarted); err != nil {
		return report, err
	}
	afterRestart, err := snapshot(ctx, db)
	if err != nil {
		return report, err
	}
	if !reflect.DeepEqual(actual, afterRestart) {
		return report, fmt.Errorf("legacy snapshot changed after restart or concurrent reads")
	}
	if want.NewSHA != "" {
		got, err := newFieldsDigest(ctx, db)
		if err != nil {
			return report, err
		}
		if got != want.NewSHA {
			return report, fmt.Errorf("new fields changed after repeat upgrade or concurrent reads")
		}
	}
	stats := db.Stats()
	if stats.MaxOpenConnections != 2 || stats.OpenConnections > 2 {
		return report, fmt.Errorf("caller-owned pool exceeded two connections")
	}
	encoded, _ := json.Marshal(actual)
	report["legacy_snapshot_sha256"] = fmt.Sprintf("%x", sha256.Sum256(encoded))
	report["tables"] = actual
	report["synthetic_instances"] = 3
	report["store_api_roundtrips"] = true
	report["instance_scoping"] = "same peer/chat/message id with distinct per-device values"
	report["idempotent_upgrade_and_restart"] = true
	report["concurrent_device_reads"] = 64
	report["max_open_connections"] = stats.MaxOpenConnections
	report["sql_downgrade_or_manual_version_change"] = false
	return report, nil
}

func snapshot(ctx context.Context, db *sql.DB) (map[string]tableDigest, error) {
	output := make(map[string]tableDigest)
	for _, table := range trackedTables {
		// Table names are a fixed allowlist. Remove only the two additive v15/v16 columns.
		query := `SELECT (to_jsonb(t) - 'companion_meta_nonce' - 'wasa_root_secret_id')::text FROM "` + table + `" t`
		rows, err := db.QueryContext(ctx, query)
		if err = checked("snapshot "+table, err); err != nil {
			return nil, err
		}
		var values []string
		for rows.Next() {
			var value string
			if err = checked("read snapshot row", rows.Scan(&value)); err != nil {
				rows.Close()
				return nil, err
			}
			values = append(values, value)
		}
		if err = checked("finish snapshot rows", rows.Err()); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
		sort.Strings(values)
		data, _ := json.Marshal(values)
		var columns string
		err = db.QueryRowContext(ctx, `SELECT jsonb_agg(jsonb_build_array(column_name,udt_name,is_nullable,column_default,ordinal_position) ORDER BY ordinal_position)::text FROM information_schema.columns WHERE table_schema='public' AND table_name=$1 AND column_name NOT IN ('companion_meta_nonce','wasa_root_secret_id')`, table).Scan(&columns)
		if err = checked("snapshot legacy columns "+table, err); err != nil {
			return nil, err
		}
		output[table] = tableDigest{len(values), fmt.Sprintf("%x", sha256.Sum256(data)), fmt.Sprintf("%x", sha256.Sum256([]byte(columns)))}
	}
	return output, nil
}

func loadManifest(ctx context.Context, db *sql.DB) (manifest, error) {
	var output manifest
	var payload []byte
	if err := checked("load baseline", db.QueryRowContext(ctx, "SELECT payload FROM nc5_probe_manifest WHERE run=$1", probeRun).Scan(&payload)); err != nil {
		return output, err
	}
	if err := checked("decode baseline", json.Unmarshal(payload, &output)); err != nil {
		return output, err
	}
	if output.Run != probeRun || len(output.Tables) != len(trackedTables) {
		return output, fmt.Errorf("baseline ownership or coverage mismatch")
	}
	return output, nil
}

func saveManifest(ctx context.Context, db *sql.DB, value manifest) error {
	payload, _ := json.Marshal(value)
	_, err := db.ExecContext(ctx, "INSERT INTO nc5_probe_manifest (run,payload) VALUES ($1,$2) ON CONFLICT (run) DO UPDATE SET payload=excluded.payload", probeRun, string(payload))
	return checked("save digest-only baseline", err)
}

func newFieldsDigest(ctx context.Context, db *sql.DB) (string, error) {
	var values string
	err := db.QueryRowContext(ctx, `SELECT jsonb_build_array((SELECT jsonb_agg(jsonb_build_array(jid,companion_meta_nonce) ORDER BY jid) FROM whatsmeow_device),(SELECT jsonb_agg(jsonb_build_array(our_jid,chat_jid,wasa_root_secret_id) ORDER BY our_jid,chat_jid) FROM whatsmeow_chat_settings))::text`).Scan(&values)
	if err = checked("snapshot new columns", err); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(values))), nil
}

func verifyConcurrentReads(ctx context.Context, container *sqlstore.Container) error {
	var workers sync.WaitGroup
	failures := make(chan error, 64)
	for index := 0; index < 64; index++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			device, err := container.GetDevice(ctx, ownJID(index%3))
			if err != nil || device == nil || device.NoiseKey == nil || device.IdentityKey == nil || device.SignedPreKey == nil {
				failures <- fmt.Errorf("concurrent device read failed")
			}
		}(index)
	}
	workers.Wait()
	close(failures)
	for err := range failures {
		return err
	}
	return nil
}
