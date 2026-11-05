package whatsmeow_service

import (
	"testing"

	"github.com/evolution-foundation/evolution-go/pkg/runtimestate"
	"go.mau.fi/whatsmeow"
)

func TestIsCurrentClientRejectsStoppingOrReplacedGeneration(t *testing.T) {
	clients := runtimestate.New[string, *whatsmeow.Client]()
	signals := runtimestate.New[string, *runtimestate.StopSignal]()
	service := whatsmeowService{clientPointer: clients, killChannel: signals}
	current, stale := &whatsmeow.Client{}, &whatsmeow.Client{}
	if service.IsCurrentClient("instance", current) || service.IsCurrentClient("instance", nil) {
		t.Fatal("unregistered client was accepted")
	}
	clients.Set("instance", current)
	if service.IsCurrentClient("instance", current) {
		t.Fatal("client without active generation was accepted")
	}
	signal := runtimestate.NewStopSignal()
	signals.Set("instance", signal)
	if !service.IsCurrentClient("instance", current) || service.IsCurrentClient("instance", stale) {
		t.Fatal("current/stale client identity was not respected")
	}
	signal.Stop()
	if service.IsCurrentClient("instance", current) {
		t.Fatal("stopping generation was accepted before pointer cleanup")
	}
	signals.Set("instance", runtimestate.NewStopSignal())
	clients.Set("instance", stale)
	if service.IsCurrentClient("instance", current) || !service.IsCurrentClient("instance", stale) {
		t.Fatal("replacement generation was not respected")
	}
	clients.Delete("instance")
	if service.IsCurrentClient("instance", stale) {
		t.Fatal("removed client was accepted")
	}
}
