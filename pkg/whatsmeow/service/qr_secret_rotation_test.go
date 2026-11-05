package whatsmeow_service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/config"
	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
	"github.com/evolution-foundation/evolution-go/pkg/runtimestate"
	"go.mau.fi/whatsmeow/types/events"
)

func qrTestSecret(value byte) string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{value}, 32))
}

func qrTestCode(reference, secret string) string {
	return reference + ",noise-key,identity-key," + secret + ",browser,suffix"
}

func awaitQRTestCode(t *testing.T, client *MyClient, publications <-chan string) string {
	t.Helper()
	select {
	case publication := <-publications:
		client.qrState.barrier(nil)
		_, code, ok := strings.Cut(publication, "|")
		if !ok {
			t.Fatal("QR publication lost its raw-code envelope")
		}
		return code
	case <-time.After(3 * time.Second):
		t.Fatal("QR renewal was not published")
		return ""
	}
}

func TestQRADVSecretRenewalRepublishesCurrentAndUpdatesRemainingWithoutNewWorker(t *testing.T) {
	client, repo, sink := newQRTestClient(t)
	client.config.QrcodeMaxCount = 2
	logDirectory := t.TempDir()
	client.loggerWrapper = logger_wrapper.NewLoggerManager(&config.Config{LogDirectory: logDirectory})
	t.Cleanup(func() { _ = client.loggerWrapper.GetLogger(client.userID).Close() })
	publications := make(chan string, 8)
	repo.beforeQR = func(code string) {
		if code != "" {
			publications <- code
		}
	}
	oldSecret, nextSecret, finalSecret := qrTestSecret(1), qrTestSecret(2), qrTestSecret(3)
	codes := []string{qrTestCode("reference-one", oldSecret), qrTestCode("reference-two", oldSecret)}
	client.handleQRCodes(codes)
	if awaitQRTestCode(t, client, publications) != codes[0] {
		t.Fatal("initial QR did not preserve the server reference")
	}
	rotation := qrTestRotation(client)
	codes[1] = "caller-owned batch was modified"
	for _, renewal := range []struct{ old, new string }{{oldSecret, nextSecret}, {nextSecret, finalSecret}} {
		client.myEventHandler(&events.RotateADVSecret{OldSecret: renewal.old, NewSecret: renewal.new})
		if awaitQRTestCode(t, client, publications) != qrTestCode("reference-one", renewal.new) {
			t.Fatal("renewal did not replace exactly the ADV field of the current reference")
		}
		client.qrState.mu.Lock()
		remaining := append([]string(nil), rotation.codes...)
		count, sameWorker := client.qrcodeCount, client.qrState.active == rotation
		client.qrState.mu.Unlock()
		if !sameWorker || count != 1 || !reflect.DeepEqual(remaining, []string{qrTestCode("reference-one", renewal.new), qrTestCode("reference-two", renewal.new)}) {
			t.Fatal("renewal changed the worker, exhausted the QR limit, or lost remaining references")
		}
		assertQRTestRunning(t, client)
	}
	_, _, writes, offline, _ := repo.snapshot()
	if writes != 3 || offline != 0 || sink.count("QRCode") != 3 || sink.count("QRTimeout") != 0 || sink.forcedLogout() {
		t.Fatal("renewal stopped pairing or duplicated QR workers")
	}
	qrTestPair(client)
	awaitQRTest(t, rotation.finished)
	client.myEventHandler(&events.RotateADVSecret{OldSecret: finalSecret, NewSecret: oldSecret})
	instance, _, finalWrites, _, _ := repo.snapshot()
	if instance.Qrcode != "" || finalWrites != writes || !instance.Connected {
		t.Fatal("late renewal overwrote a completed pairing")
	}
	assertQRTestRunning(t, client)
	logs, err := os.ReadFile(filepath.Join(logDirectory, client.userID, "instance.log"))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{oldSecret, nextSecret, finalSecret, "reference-one", "noise-key", "identity-key"} {
		if bytes.Contains(logs, []byte(secret)) {
			t.Fatal("QR credentials were written to the instance log")
		}
	}
}

func TestQRADVSecretRenewalsBeforePublicationCoalesceAndPreserveOtherFields(t *testing.T) {
	client, repo, sink := newQRTestClient(t)
	gate := newQRTestGate(t)
	client.qrBeforeAction = func(string) { gate.hold() }
	publications := make(chan string, 4)
	repo.beforeQR = func(code string) { publications <- code }
	oldSecret, nextSecret, finalSecret := qrTestSecret(4), qrTestSecret(5), qrTestSecret(6)
	code := oldSecret + ",noise-key,identity-key," + oldSecret + ",browser," + oldSecret
	client.handleQRCodes([]string{code})
	awaitQRTest(t, gate.entered)
	rotation := qrTestRotation(client)
	client.myEventHandler(&events.RotateADVSecret{OldSecret: oldSecret, NewSecret: nextSecret})
	client.myEventHandler(&events.RotateADVSecret{OldSecret: nextSecret, NewSecret: finalSecret})
	gate.open()
	want := oldSecret + ",noise-key,identity-key," + finalSecret + ",browser," + oldSecret
	if awaitQRTestCode(t, client, publications) != want {
		t.Fatal("coalesced renewal changed the reference or a non-ADV field")
	}
	_, _, writes, _, _ := repo.snapshot()
	if writes != 1 || sink.count("QRCode") != 1 {
		t.Fatal("renewal already included in publication caused a duplicate QR")
	}
	qrTestPair(client)
	awaitQRTest(t, rotation.finished)
}

func TestQRADVSecretRenewalCannotPublishAfterPairingWins(t *testing.T) {
	client, repo, sink := newQRTestClient(t)
	gate := newQRTestGate(t)
	var publications atomic.Int32
	client.qrBeforeAction = func(string) {
		if publications.Add(1) == 2 {
			gate.hold()
		}
	}
	writes := make(chan string, 4)
	repo.beforeQR = func(code string) { writes <- code }
	oldSecret, nextSecret := qrTestSecret(7), qrTestSecret(8)
	client.handleQRCodes([]string{qrTestCode("reference", oldSecret)})
	_ = awaitQRTestCode(t, client, writes)
	rotation := qrTestRotation(client)
	client.myEventHandler(&events.RotateADVSecret{OldSecret: oldSecret, NewSecret: nextSecret})
	awaitQRTest(t, gate.entered)
	qrTestPair(client)
	gate.open()
	awaitQRTest(t, rotation.finished)
	instance, _, count, offline, pairs := repo.snapshot()
	if instance.Qrcode != "" || !instance.Connected || count != 1 || offline != 0 || pairs != 1 || sink.count("QRCode") != 1 {
		t.Fatal("in-flight renewal overwrote a completed pairing")
	}
	assertQRTestRunning(t, client)
}

func TestQRADVSecretRenewalIgnoresStaleGenerationAndPasskeyPause(t *testing.T) {
	for _, scenario := range []string{"stale generation", "active passkey"} {
		t.Run(scenario, func(t *testing.T) {
			client, repo, sink := newQRTestClient(t)
			gate := newQRTestGate(t)
			client.qrBeforeAction = func(string) { gate.hold() }
			oldSecret, nextSecret := qrTestSecret(9), qrTestSecret(10)
			code := qrTestCode("reference", oldSecret)
			client.handleQRCodes([]string{code})
			awaitQRTest(t, gate.entered)
			rotation := qrTestRotation(client)
			var next *runtimestate.StopSignal
			if scenario == "stale generation" {
				client.stopSignal.Stop()
				next = runtimestate.NewStopSignal()
				next.Begin()
				client.killChannel.Set(client.userID, next)
				defer func() { next.Stop(); next.Complete() }()
			} else {
				client.passkeyCeremony.Start(client.userID, json.RawMessage(`{}`))
			}
			client.myEventHandler(&events.RotateADVSecret{OldSecret: oldSecret, NewSecret: nextSecret})
			gate.open()
			awaitQRTest(t, rotation.finished)
			if rotation.codes[0] != code {
				t.Fatal("late renewal modified an inactive QR batch")
			}
			_, _, writes, offline, pairs := repo.snapshot()
			if writes != 0 || offline != 0 || pairs != 0 || sink.count("QRCode") != 0 || sink.count("QRTimeout") != 0 {
				t.Fatal("late renewal changed persisted or emitted state")
			}
			if next != nil {
				select {
				case <-next.Done():
					t.Fatal("old renewal stopped its replacement generation")
				default:
				}
			} else {
				assertQRTestRunning(t, client)
			}
		})
	}
}
