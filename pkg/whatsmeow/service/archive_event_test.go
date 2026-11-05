package whatsmeow_service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/config"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type archivedWebhook struct {
	queue string
	data  []byte
}

type archiveWebhookRecorder struct {
	WhatsmeowService
	delivered chan archivedWebhook
}

func (recorder *archiveWebhookRecorder) CallWebhook(_ *instance_model.Instance, queue string, data []byte) {
	recorder.delivered <- archivedWebhook{queue: queue, data: append([]byte(nil), data...)}
}

func TestArchiveEventReachesWebhookWithTypedFields(t *testing.T) {
	const id = "synthetic-archive"
	configuration := &config.Config{LogDirectory: t.TempDir()}
	manager := logger_wrapper.NewLoggerManager(configuration)
	t.Cleanup(func() { _ = manager.GetLogger(id).Close() })
	recorder := &archiveWebhookRecorder{delivered: make(chan archivedWebhook, 1)}
	client := &MyClient{
		service: recorder, userID: id, token: "synthetic-token",
		Instance: &instance_model.Instance{Id: id, Name: "Synthetic"},
		config:   configuration, loggerWrapper: manager,
	}
	jid := types.NewJID("10000000000", types.DefaultUserServer)
	client.myEventHandler(&events.Archive{JID: jid, FromFullSync: true})
	select {
	case delivery := <-recorder.delivered:
		var envelope struct {
			Event string                     `json:"event"`
			Data  map[string]json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(delivery.data, &envelope); err != nil {
			t.Fatal(err)
		}
		if delivery.queue != id+".archive" || envelope.Event != "Archive" {
			t.Fatalf("archive envelope changed: queue=%s event=%s", delivery.queue, envelope.Event)
		}
		var deliveredJID string
		var fromFullSync bool
		if err := json.Unmarshal(envelope.Data["JID"], &deliveredJID); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(envelope.Data["FromFullSync"], &fromFullSync); err != nil {
			t.Fatal(err)
		}
		if deliveredJID != jid.String() || !fromFullSync || envelope.Data["Timestamp"] == nil || envelope.Data["Action"] == nil {
			t.Fatalf("archive fields missing or changed: %s", delivery.data)
		}
	case <-time.After(time.Second):
		t.Fatal("typed Archive event did not reach the webhook")
	}
}
