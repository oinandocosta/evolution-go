package instance_service

import (
	"errors"
	"strings"
	"testing"

	"github.com/evolution-foundation/evolution-go/pkg/config"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	instance_repository "github.com/evolution-foundation/evolution-go/pkg/instance/repository"
	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
	"github.com/evolution-foundation/evolution-go/pkg/runtimestate"
	whatsmeow_service "github.com/evolution-foundation/evolution-go/pkg/whatsmeow/service"
	"go.mau.fi/whatsmeow"
)

type recordedSession struct {
	connected bool
	loggedIn  bool
}

func (session recordedSession) IsConnected() bool { return session.connected }
func (session recordedSession) IsLoggedIn() bool  { return session.loggedIn }

type qrStartupRecorder struct {
	whatsmeow_service.WhatsmeowService
	starts []string
	err    error
}

func (recorder *qrStartupRecorder) StartInstance(id string) error {
	recorder.starts = append(recorder.starts, id)
	return recorder.err
}

type disconnectedRepository struct {
	instance_repository.InstanceRepository
	stored instance_model.Instance
	calls  int
	err    error
}

func (repository *disconnectedRepository) UpdateConnected(id string, connected bool, reason string) error {
	repository.calls++
	if repository.err != nil {
		return repository.err
	}
	if id != repository.stored.Id {
		return errors.New("unexpected instance")
	}
	repository.stored.Connected = connected
	repository.stored.DisconnectReason = reason
	return nil
}

func sessionTestLogger(t *testing.T, id string) *logger_wrapper.LoggerManager {
	t.Helper()
	manager := logger_wrapper.NewLoggerManager(&config.Config{LogDirectory: t.TempDir()})
	t.Cleanup(func() { _ = manager.GetLogger(id).Close() })
	return manager
}

func TestQRPollingDoesNotRestartAuthenticatedSession(t *testing.T) {
	const id = "synthetic-session"
	startup := &qrStartupRecorder{}
	service := instances{whatsmeowService: startup, loggerWrapper: sessionTestLogger(t, id)}
	for _, connected := range []bool{true, false} {
		for poll := 0; poll < 20; poll++ {
			started, err := service.startQRClient(id, recordedSession{connected: connected, loggedIn: true})
			if started || err == nil || !strings.Contains(err.Error(), "already logged in") {
				t.Fatalf("authenticated QR poll started a client or lost the session response: started=%v err=%v", started, err)
			}
		}
	}
	if len(startup.starts) != 0 {
		t.Fatalf("QR polling restarted authenticated session %d times", len(startup.starts))
	}
}

func TestQRPollingReusesPairingSocketAndStartsMissingSession(t *testing.T) {
	const id = "synthetic-session"
	startup := &qrStartupRecorder{}
	service := instances{whatsmeowService: startup, loggerWrapper: sessionTestLogger(t, id)}
	if started, err := service.startQRClient(id, recordedSession{connected: true}); err != nil || started {
		t.Fatalf("active pairing socket was restarted: %v, %v", started, err)
	}
	for _, client := range []clientSessionState{nil, recordedSession{}} {
		if started, err := service.startQRClient(id, client); err != nil || !started {
			t.Fatalf("missing/disconnected session did not start: %v, %v", started, err)
		}
	}
	if len(startup.starts) != 2 || startup.starts[0] != id || startup.starts[1] != id {
		t.Fatalf("unexpected startup actions: %v", startup.starts)
	}
	startup.err = errors.New("startup failed")
	if started, err := service.startQRClient(id, nil); started || !errors.Is(err, startup.err) {
		t.Fatalf("startup failure was not propagated: %v, %v", started, err)
	}
}

func TestGetQrStartsMissingTypedClientWithoutPanic(t *testing.T) {
	const id = "synthetic-session"
	startup := &qrStartupRecorder{err: errors.New("synthetic startup failure")}
	clients := runtimestate.New[string, *whatsmeow.Client]()
	service := instances{
		clientPointer: clients, whatsmeowService: startup,
		loggerWrapper: sessionTestLogger(t, id),
	}
	// Map.Get returns a typed *whatsmeow.Client nil. GetQr must keep that
	// absence as a nil interface before asking the session policy to start it.
	if result, err := service.GetQr(&instance_model.Instance{Id: id}); result != nil || !errors.Is(err, startup.err) {
		t.Fatalf("missing client startup was not attempted: result=%+v err=%v", result, err)
	}
	if len(startup.starts) != 1 || startup.starts[0] != id {
		t.Fatalf("unexpected startup actions: %v", startup.starts)
	}
}

func TestDisconnectPreservesStoredSubscriptionsAndSettings(t *testing.T) {
	instance := &instance_model.Instance{Id: "synthetic-session", Events: "MESSAGE,CONNECTION,QRCODE", Webhook: "https://example.invalid/webhook", Jid: "10000000000@s.whatsapp.net", Connected: true}
	repository := &disconnectedRepository{stored: *instance}
	service := instances{instanceRepository: repository, loggerWrapper: sessionTestLogger(t, instance.Id)}
	signal := runtimestate.NewStopSignal()
	result, err := service.disconnectSession(instance, recordedSession{connected: true, loggedIn: true}, signal)
	if err != nil || result != instance || repository.calls != 1 {
		t.Fatalf("disconnect did not persist its state: result=%p calls=%d err=%v", result, repository.calls, err)
	}
	select {
	case <-signal.Done():
	default:
		t.Fatal("disconnect did not stop its connection generation")
	}
	if instance.Connected || repository.stored.Connected || repository.stored.DisconnectReason != "Disconnected by API" {
		t.Fatalf("disconnected status was not persisted: %+v", repository.stored)
	}
	if repository.stored.Events != "MESSAGE,CONNECTION,QRCODE" || instance.Events != repository.stored.Events || repository.stored.Webhook != instance.Webhook || repository.stored.Jid != instance.Jid {
		t.Fatalf("disconnect damaged reconnection settings: %+v", repository.stored)
	}
}

func TestDisconnectPropagatesPersistenceFailure(t *testing.T) {
	instance := &instance_model.Instance{Id: "synthetic-session", Events: "MESSAGE", Connected: true}
	repository := &disconnectedRepository{stored: *instance, err: errors.New("database unavailable")}
	service := instances{instanceRepository: repository, loggerWrapper: sessionTestLogger(t, instance.Id)}
	if _, err := service.disconnectSession(instance, recordedSession{connected: true, loggedIn: true}, runtimestate.NewStopSignal()); !errors.Is(err, repository.err) {
		t.Fatalf("persistence failure was hidden: %v", err)
	}
	if instance.Events != "MESSAGE" || repository.stored.Events != "MESSAGE" {
		t.Fatal("failed disconnect erased subscriptions")
	}
}
