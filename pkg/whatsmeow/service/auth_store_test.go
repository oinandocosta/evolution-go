package whatsmeow_service

import (
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/evolution-foundation/evolution-go/pkg/config"
	"go.mau.fi/whatsmeow/store/sqlstore"
	_ "modernc.org/sqlite"
)

func assertSharedAuthStore(t *testing.T, service *whatsmeowService) *sqlstore.Container {
	t.Helper()
	results := make(chan *sqlstore.Container, 64)
	errors := make(chan error, 64)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for index := 0; index < 64; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			// StartClient uses a value receiver: copies must share the holder.
			copy := *service
			container, err := copy.getStoreContainer()
			results <- container
			errors <- err
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	var first *sqlstore.Container
	for container := range results {
		if first == nil {
			first = container
		}
		if container != first {
			t.Fatal("client starts opened different auth containers")
		}
	}
	return first
}

func TestAuthStoreSharedAndRetry(t *testing.T) {
	directory := t.TempDir()
	service := &whatsmeowService{config: &config.Config{}, exPath: directory, storeContainer: &storeContainerHolder{}}
	// Missing directory makes the first open fail; the failure must not poison retries.
	if _, err := service.getStoreContainer(); err == nil {
		t.Fatal("expected the initial open to fail")
	}
	if err := os.Mkdir(filepath.Join(directory, "dbdata"), 0700); err != nil {
		t.Fatal(err)
	}
	container := assertSharedAuthStore(t, service)
	t.Cleanup(func() { _ = container.Close() })
}

func TestAuthStoreMissingPostgresHandle(t *testing.T) {
	service := &whatsmeowService{config: &config.Config{PostgresAuthDB: "configured"}, storeContainer: &storeContainerHolder{}}
	if _, err := service.getStoreContainer(); err == nil {
		t.Fatal("missing postgres handle was accepted")
	}
}

func TestAuthStorePostgres(t *testing.T) {
	dsn := os.Getenv("EVOGO_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requires the disposable homologation PostgreSQL container")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Hostname() != "evogo-homolog-pg" || parsed.Path != "/evogo_verify" {
		t.Fatal("refusing any database outside the named disposable homologation container")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)
	service := &whatsmeowService{config: &config.Config{PostgresAuthDB: dsn}, authDB: db, storeContainer: &storeContainerHolder{}}
	assertSharedAuthStore(t, service)
	if err := db.Ping(); err != nil {
		t.Fatalf("the caller-owned auth pool was closed: %v", err)
	}
	if stats := db.Stats(); stats.MaxOpenConnections != 2 || stats.OpenConnections > 2 {
		t.Fatalf("auth pool limits changed: %+v", stats)
	}
	t.Log("64 concurrent client starts reused the bounded caller-owned PostgreSQL auth pool")
}
