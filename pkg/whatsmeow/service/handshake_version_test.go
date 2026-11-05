package whatsmeow_service

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func TestHandshakeVersionPayloads(t *testing.T) {
	previous := store.GetWAVersion()
	t.Cleanup(func() { store.SetWAVersion(previous) })
	want := store.WAVersionContainer{2, 3000, 1053085257}
	if !applyHandshakeVersion(clientVersion{2, 3000, 1053085257}, true) {
		t.Fatal("valid version rejected")
	}
	device := (&sqlstore.Container{}).NewDevice()
	registration := device.GetClientPayload()
	if got := registration.GetUserAgent().GetAppVersion().GetTertiary(); got != want[2] {
		t.Fatalf("registration version = %d, want %d", got, want[2])
	}
	hash := want.Hash()
	if !bytes.Equal(registration.GetDevicePairingData().GetBuildHash(), hash[:]) {
		t.Fatal("registration build hash differs from advertised version")
	}
	jid := types.NewJID("10000000000", types.DefaultUserServer)
	device.ID = &jid
	login := device.GetClientPayload()
	if got := login.GetUserAgent().GetAppVersion().GetTertiary(); got != want[2] {
		t.Fatalf("login version = %d, want %d", got, want[2])
	}
	if applyHandshakeVersion(clientVersion{2, 3000, 0}, false) || store.GetWAVersion() != want {
		t.Fatal("invalid version replaced the working version")
	}
	if applyHandshakeVersion(clientVersion{2, 3000, 1042386815}, false) || store.GetWAVersion() != want {
		t.Fatal("web lookup downgraded the working handshake")
	}
	if !applyHandshakeVersion(clientVersion{2, 3000, 1042386815}, true) {
		t.Fatal("explicit version override was ignored")
	}
}

// Opt-in live handshake only: ephemeral keys, no saved session, no QR scan,
// no production database, no phone number and no messages.
func TestHandshakeVersionLiveQR(t *testing.T) {
	if os.Getenv("EVOGO_TEST_LIVE_QR") != "1" {
		t.Skip("set EVOGO_TEST_LIVE_QR=1 to contact WhatsApp with a disposable device")
	}
	previous := store.GetWAVersion()
	t.Cleanup(func() { store.SetWAVersion(previous) })
	version, err := fetchWhatsAppWebVersion()
	if err != nil {
		t.Fatal(err)
	}
	applyHandshakeVersion(*version, false)
	t.Logf("handshake version: %s", store.GetWAVersion())
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	device := (&sqlstore.Container{}).NewDevice()
	client := whatsmeow.NewClient(device, waLog.Noop)
	client.EnableAutoReconnect = false
	defer client.Disconnect()
	qr := make(chan int, 1)
	failures := make(chan string, 1)
	client.AddEventHandler(func(event interface{}) {
		switch value := event.(type) {
		case *events.QR:
			select {
			case qr <- len(value.Codes):
			default:
			}
		case *events.ClientOutdated:
			select {
			case failures <- "ClientOutdated":
			default:
			}
		case *events.ConnectFailure:
			select {
			case failures <- "ConnectFailure":
			default:
			}
		}
	})
	if err := client.ConnectContext(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case count := <-qr:
		if count == 0 {
			t.Fatal("empty QR event")
		}
		t.Logf("WhatsApp accepted the fresh handshake and returned %d QR codes (contents omitted)", count)
	case reason := <-failures:
		t.Fatalf("WhatsApp rejected handshake: %s", reason)
	case <-ctx.Done():
		t.Fatal("timed out waiting for QR")
	}
}
