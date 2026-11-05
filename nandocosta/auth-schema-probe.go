// Run only inside the named disposable homologation network.
// This source compiles against both pinned whatsmeow versions.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	_ "github.com/lib/pq"
	"go.mau.fi/whatsmeow/proto/waAdv"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
)

func main() {
	dsn := os.Getenv("EVOGO_TEST_POSTGRES_DSN")
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Hostname() != "evogo-homolog-pg" || parsed.Path != "/evogo_homolog_auth" {
		log.Fatal("refusing database outside the named disposable homologation container")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	container, err := sqlstore.New(ctx, "postgres", dsn, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer container.Close()
	jid := types.NewJID("10000000000", types.DefaultUserServer)
	if os.Getenv("EVOGO_PROBE_MODE") == "seed" {
		device := container.NewDevice()
		device.ID = &jid
		// Placeholder signatures satisfy storage lengths; this is not an
		// authenticated WhatsApp account and is never sent to WhatsApp.
		device.Account = &waAdv.ADVSignedDeviceIdentity{
			AccountSignature:    make([]byte, 64),
			AccountSignatureKey: make([]byte, 32),
			DeviceSignature:     make([]byte, 64),
		}
		if err := device.Save(ctx); err != nil {
			log.Fatal(err)
		}
	}
	device, err := container.GetDevice(ctx, jid)
	if err != nil || device == nil {
		log.Fatal("synthetic session was not preserved")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	var schema int
	if err := db.QueryRowContext(ctx, "SELECT version FROM whatsmeow_version").Scan(&schema); err != nil {
		log.Fatal(err)
	}
	fingerprint := sha256.Sum256(device.NoiseKey.Pub[:])
	fmt.Printf("schema=%d synthetic_session=preserved public_key_fingerprint=%x\n", schema, fingerprint)
}
