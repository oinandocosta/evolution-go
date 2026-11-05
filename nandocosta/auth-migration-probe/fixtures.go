package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	instanceModel "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	labelModel "github.com/evolution-foundation/evolution-go/pkg/label/model"
	messageModel "github.com/evolution-foundation/evolution-go/pkg/message/model"
	"go.mau.fi/whatsmeow/proto/waAdv"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var peerPN = types.NewJID("20000000000", types.DefaultUserServer)
var peerLID = types.NewJID("30000000000", types.HiddenUserServer)
var groupJID = types.NewJID("10000000000-1000000000", types.GroupServer)

func ownJID(index int) types.JID {
	return types.JID{User: fmt.Sprintf("1000000000%d", index), Device: uint16(index + 1), Server: types.DefaultUserServer}
}

func ownLID(index int) types.JID {
	return types.NewJID(fmt.Sprintf("4000000000%d", index), types.HiddenUserServer)
}

func fixtureBytes(index, tag, length int) []byte {
	return bytes.Repeat([]byte{byte(index + 1), byte(tag)}, (length+1)/2)[:length]
}

func fixture32(index, tag int) [32]byte { return [32]byte(fixtureBytes(index, tag, 32)) }
func fixture128(index int) [128]byte    { return [128]byte(fixtureBytes(index, 9, 128)) }
func fixtureTime(index int) time.Time   { return time.Unix(1700000000+int64(index), 0).UTC() }
func fixtureName(index int) string      { return fmt.Sprintf("Synthetic contact %d", index) }
func fixtureNonce(index int) string     { return fmt.Sprintf("synthetic-companion-nonce-%d", index) }
func fixtureWASA(index int) string      { return fmt.Sprintf("SYNTHETIC-WASA-ROOT-%d", index) }

func seed(ctx context.Context, db *sql.DB, container *sqlstore.Container) error {
	appDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err = checked("wrap app database", err); err != nil {
		return err
	}
	// The actual application models create the same settings/message/label tables.
	if err = checked("create disposable app schema", appDB.WithContext(ctx).AutoMigrate(&instanceModel.Instance{}, &messageModel.Message{}, &labelModel.Label{})); err != nil {
		return err
	}
	for index := 0; index < 3; index++ {
		device := container.NewDevice()
		jid := ownJID(index)
		device.ID = &jid
		device.LID = ownLID(index)
		device.Platform = "synthetic-desktop"
		device.BusinessName = fmt.Sprintf("Synthetic business %d", index)
		device.PushName = fixtureName(index)
		device.LIDMigrationTimestamp = fixtureTime(index).Unix()
		device.Account = &waAdv.ADVSignedDeviceIdentity{
			Details: fixtureBytes(index, 1, 24), AccountSignature: fixtureBytes(index, 2, 64),
			AccountSignatureKey: fixtureBytes(index, 3, 32), DeviceSignature: fixtureBytes(index, 4, 64),
		}
		if err = checked("save synthetic device", device.Save(ctx)); err != nil {
			return err
		}
		if err = seedStoreData(ctx, device, index); err != nil {
			return err
		}
		instance := instanceModel.Instance{
			Id: fmt.Sprintf("00000000-0000-4000-8000-00000000000%d", index), Name: fmt.Sprintf("Synthetic instance %d", index),
			Token: fmt.Sprintf("synthetic-token-%d", index), Webhook: fmt.Sprintf("https://example.invalid/webhook/%d", index),
			RabbitmqEnable: "enabled", WebSocketEnable: "enabled", NatsEnable: "disabled", Jid: jid.String(),
			Connected: index%2 == 0, Expiration: 1900000000 + int64(index), DisconnectReason: "synthetic-only",
			Events: "MESSAGE,CONNECTION,QRCODE", OsName: "Synthetic Linux", Proxy: "http://proxy.invalid:8080",
			ClientName: "synthetic-client", CreatedAt: fixtureTime(index), AlwaysOnline: index%2 == 0,
			RejectCall: true, MsgRejectCall: "Synthetic rejection", ReadMessages: index%2 == 1, IgnoreGroups: true, IgnoreStatus: true,
		}
		if err = checked("save synthetic app settings", appDB.WithContext(ctx).Create(&instance).Error); err != nil {
			return err
		}
		message := messageModel.Message{MessageID: fmt.Sprintf("SYNTHETIC-MESSAGE-%d", index), Timestamp: "1700000000", Status: "DELIVERY_ACK", Source: "synthetic", Referral: json.RawMessage(`{"source":"synthetic"}`)}
		if err = checked("save synthetic app message", appDB.WithContext(ctx).Create(&message).Error); err != nil {
			return err
		}
		label := labelModel.Label{InstanceID: instance.Id, LabelID: fmt.Sprint(index), LabelName: "Synthetic label", LabelColor: "1", PredefinedId: "0"}
		if err = checked("save synthetic app label", appDB.WithContext(ctx).Create(&label).Error); err != nil {
			return err
		}
	}
	tables, err := snapshot(ctx, db)
	if err != nil {
		return err
	}
	for _, table := range trackedTables {
		if tables[table].Rows == 0 {
			return fmt.Errorf("fixture did not populate tracked table %s", table)
		}
	}
	if _, err = db.ExecContext(ctx, "CREATE TABLE nc5_probe_manifest (run text PRIMARY KEY, payload jsonb NOT NULL)"); err != nil {
		return checked("create probe ownership marker", err)
	}
	return saveManifest(ctx, db, manifest{Run: probeRun, Tables: tables})
}

func seedStoreData(ctx context.Context, device *store.Device, index int) error {
	if err := checked("generate synthetic prekeys", func() error { _, err := device.PreKeys.GetOrGenPreKeys(ctx, 3); return err }()); err != nil {
		return err
	}
	steps := []struct {
		name string
		run  func() error
	}{
		{"uploaded prekeys", func() error { return device.PreKeys.MarkPreKeysAsUploaded(ctx, 2) }},
		{"signal identity", func() error { return device.Identities.PutIdentity(ctx, "20000000000:1", fixture32(index, 5)) }},
		{"signal session", func() error { return device.Sessions.PutSession(ctx, "20000000000:1", fixtureBytes(index, 6, 160)) }},
		{"sender key", func() error {
			return device.SenderKeys.PutSenderKey(ctx, groupJID.String(), "20000000000:1", fixtureBytes(index, 7, 96))
		}},
		{"app-state key", func() error {
			return device.AppStateKeys.PutAppStateSyncKey(ctx, []byte("synthetic-key-id"), store.AppStateSyncKey{Data: fixtureBytes(index, 8, 32), Timestamp: fixtureTime(index).Unix(), Fingerprint: fixtureBytes(index, 8, 16)})
		}},
		{"app-state version", func() error {
			return device.AppState.PutAppStateVersion(ctx, "regular", uint64(index+5), fixture128(index))
		}},
		{"app-state mutation", func() error {
			return device.AppState.PutAppStateMutationMACs(ctx, "regular", uint64(index+5), []store.AppStateMutationMAC{{IndexMAC: fixtureBytes(index, 10, 32), ValueMAC: fixtureBytes(index, 11, 32)}})
		}},
		{"contact name", func() error { return device.Contacts.PutContactName(ctx, peerPN, fixtureName(index), "Synthetic") }},
		{"contact push name", func() error {
			_, _, err := device.Contacts.PutPushName(ctx, peerPN, fmt.Sprintf("Push %d", index))
			return err
		}},
		{"contact business", func() error {
			_, _, err := device.Contacts.PutBusinessName(ctx, peerPN, fmt.Sprintf("Business %d", index))
			return err
		}},
		{"redacted contact", func() error {
			return device.Contacts.PutManyRedactedPhones(ctx, []store.RedactedPhoneEntry{{JID: peerPN, RedactedPhone: "+2····00"}})
		}},
		{"muted setting", func() error { return device.ChatSettings.PutMutedUntil(ctx, peerPN, fixtureTime(index)) }},
		{"pinned setting", func() error { return device.ChatSettings.PutPinned(ctx, peerPN, true) }},
		{"archived setting", func() error { return device.ChatSettings.PutArchived(ctx, peerPN, index%2 == 0) }},
		{"message secret", func() error {
			return device.MsgSecrets.PutMessageSecret(ctx, peerPN, peerPN, "SYNTHETIC-SECRET-MESSAGE", fixtureBytes(index, 12, 32))
		}},
		{"privacy token", func() error {
			return device.PrivacyTokens.PutPrivacyTokens(ctx, store.PrivacyToken{User: peerPN, Token: fixtureBytes(index, 13, 32), Timestamp: fixtureTime(index), SenderTimestamp: fixtureTime(index).Add(time.Minute)})
		}},
		{"nct salt", func() error { return device.NCTSalt.PutNCTSalt(ctx, fixtureBytes(index, 14, 32)) }},
		{"peer lid mapping", func() error { return device.LIDs.PutLIDMapping(ctx, peerLID, peerPN) }},
		{"own lid mapping", func() error { return device.LIDs.PutLIDMapping(ctx, ownLID(index), ownJID(index).ToNonAD()) }},
		{"decryption buffer", func() error {
			return device.EventBuffer.PutBufferedEvent(ctx, fixture32(index, 15), fixtureBytes(index, 16, 64), fixtureTime(index))
		}},
		{"outgoing retry buffer", func() error {
			return device.EventBuffer.AddOutgoingEvent(ctx, peerPN, "SYNTHETIC-RETRY-MESSAGE", "protobuf", fixtureBytes(index, 17, 64))
		}},
	}
	for _, step := range steps {
		if err := checked("seed "+step.name, step.run()); err != nil {
			return err
		}
	}
	return nil
}
