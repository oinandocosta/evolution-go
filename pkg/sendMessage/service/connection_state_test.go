package send_service

import (
	"testing"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/runtimestate"
	"go.mau.fi/whatsmeow"
)

type fakeConnectionState struct {
	connected bool
	loggedIn  bool
}

func (f fakeConnectionState) IsConnected() bool { return f.connected }
func (f fakeConnectionState) IsLoggedIn() bool  { return f.loggedIn }

func TestClientSessionReadyRequiresConnectedAndLoggedIn(t *testing.T) {
	for _, tt := range []struct {
		name  string
		state connectionState
		want  bool
	}{
		{"nil", nil, false},
		{"typed nil", (*whatsmeow.Client)(nil), false},
		{"socket without authentication", fakeConnectionState{connected: true}, false},
		{"authentication without socket", fakeConnectionState{loggedIn: true}, false},
		{"authenticated socket", fakeConnectionState{connected: true, loggedIn: true}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := clientSessionReady(tt.state); got != tt.want {
				t.Fatalf("ready = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSendDelayRejectsClientReplacement(t *testing.T) {
	clients := runtimestate.New[string, *whatsmeow.Client]()
	oldClient, replacement := &whatsmeow.Client{}, &whatsmeow.Client{}
	clients.Set("synthetic-instance", oldClient)
	started := make(chan struct{})
	result := make(chan bool, 1)
	go func() {
		first := true
		result <- waitForSendDelay(5*time.Second, func() bool {
			if first {
				close(started)
				first = false
			}
			return clients.Get("synthetic-instance") == oldClient
		})
	}()
	<-started
	clients.Set("synthetic-instance", replacement)
	select {
	case ready := <-result:
		if ready {
			t.Fatal("replaced client was allowed to send")
		}
	case <-time.After(time.Second):
		t.Fatal("replacement did not interrupt the presence delay")
	}
}

func TestSendDelayChecksSessionAgainAfterTimer(t *testing.T) {
	calls := 0
	if waitForSendDelay(0, func() bool { calls++; return calls == 1 }) {
		t.Fatal("client invalidated at delay completion was allowed to send")
	}
	if calls != 2 {
		t.Fatalf("readiness checked %d times, want 2", calls)
	}
}
