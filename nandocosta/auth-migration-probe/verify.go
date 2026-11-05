package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"reflect"
	"time"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
)

// A local interface allows the identical probe to compile with the old dependency.
type wasaStore interface {
	PutWASARootSecretID(context.Context, types.JID, types.MessageID) error
	GetWASARootSecretID(context.Context, types.JID) (types.MessageID, error)
}

func verifyAPIs(ctx context.Context, container *sqlstore.Container, newFieldsWritten bool) error {
	devices, err := container.GetAllDevices(ctx)
	if err = checked("enumerate devices", err); err != nil {
		return err
	}
	if len(devices) != 3 {
		return fmt.Errorf("expected exactly three synthetic devices")
	}
	seen := make(map[[32]byte]bool)
	for index := 0; index < 3; index++ {
		device, err := container.GetDevice(ctx, ownJID(index))
		if err = checked("load device", err); err != nil {
			return err
		}
		if device == nil || device.NoiseKey == nil || device.IdentityKey == nil || device.SignedPreKey == nil ||
			device.Account == nil || device.LID != ownLID(index) || !device.Initialized {
			return fmt.Errorf("device identity or initialization was not preserved")
		}
		key := append(append(append([]byte{}, device.NoiseKey.Priv[:]...), device.IdentityKey.Priv[:]...), device.SignedPreKey.Priv[:]...)
		digest := sha256.Sum256(key)
		if seen[digest] {
			return fmt.Errorf("synthetic devices do not have distinct key material")
		}
		seen[digest] = true
		if err = verifyStoreData(ctx, device, index); err != nil {
			return err
		}
		if newFieldsWritten {
			field := reflect.ValueOf(device).Elem().FieldByName("CompanionMetaNonce")
			if field.IsValid() && field.String() != fixtureNonce(index) {
				return fmt.Errorf("companion nonce did not survive new-code read")
			}
			if settings, supported := device.ChatSettings.(wasaStore); supported {
				value, err := settings.GetWASARootSecretID(ctx, peerPN)
				if err = checked("read WASA root secret id", err); err != nil {
					return err
				}
				if value != fixtureWASA(index) {
					return fmt.Errorf("WASA root secret id did not survive new-code read")
				}
			}
		}
	}
	return nil
}

func verifyStoreData(ctx context.Context, device *store.Device, index int) error {
	trusted, err := device.Identities.IsTrustedIdentity(ctx, "20000000000:1", fixture32(index, 5))
	if err != nil || !trusted {
		return fmt.Errorf("stored identity is not trusted")
	}
	trusted, err = device.Identities.IsTrustedIdentity(ctx, "20000000000:1", fixture32((index+1)%3, 5))
	if err != nil || trusted {
		return fmt.Errorf("foreign instance identity was accepted")
	}
	session, err := device.Sessions.GetSession(ctx, "20000000000:1")
	if err != nil || !bytes.Equal(session, fixtureBytes(index, 6, 160)) {
		return fmt.Errorf("signal session bytes were not preserved or scoped")
	}
	for id := uint32(1); id <= 3; id++ {
		key, err := device.PreKeys.GetPreKey(ctx, id)
		if err != nil || key == nil || key.KeyID != id || key.Priv == nil || *key.Priv == [32]byte{} {
			return fmt.Errorf("prekey was not preserved")
		}
	}
	count, err := device.PreKeys.UploadedPreKeyCount(ctx)
	if err != nil || count != 2 {
		return fmt.Errorf("prekey upload flags changed")
	}
	sender, err := device.SenderKeys.GetSenderKey(ctx, groupJID.String(), "20000000000:1")
	if err != nil || !bytes.Equal(sender, fixtureBytes(index, 7, 96)) {
		return fmt.Errorf("sender key was not preserved or scoped")
	}
	appKey, err := device.AppStateKeys.GetAppStateSyncKey(ctx, []byte("synthetic-key-id"))
	if err != nil || appKey == nil || !bytes.Equal(appKey.Data, fixtureBytes(index, 8, 32)) || appKey.Timestamp != fixtureTime(index).Unix() {
		return fmt.Errorf("app-state sync key was not preserved or scoped")
	}
	version, hash, err := device.AppState.GetAppStateVersion(ctx, "regular")
	if err != nil || version != uint64(index+5) || hash != fixture128(index) {
		return fmt.Errorf("app-state version or hash changed")
	}
	mac, err := device.AppState.GetAppStateMutationMAC(ctx, "regular", fixtureBytes(index, 10, 32))
	if err != nil || !bytes.Equal(mac, fixtureBytes(index, 11, 32)) {
		return fmt.Errorf("app-state mutation mac changed")
	}
	contact, err := device.Contacts.GetContact(ctx, peerPN)
	if err != nil || !contact.Found || contact.FirstName != fixtureName(index) || contact.FullName != "Synthetic" || contact.PushName != fmt.Sprintf("Push %d", index) || contact.RedactedPhone != "+2····00" {
		return fmt.Errorf("contact names or redacted phone were not preserved or scoped")
	}
	settings, err := device.ChatSettings.GetChatSettings(ctx, peerPN)
	if err != nil || !settings.Found || !settings.Pinned || settings.Archived != (index%2 == 0) || !settings.MutedUntil.Equal(fixtureTime(index)) {
		return fmt.Errorf("chat settings changed")
	}
	secret, senderJID, err := device.MsgSecrets.GetMessageSecret(ctx, peerLID, peerLID, "SYNTHETIC-SECRET-MESSAGE")
	if err != nil || !bytes.Equal(secret, fixtureBytes(index, 12, 32)) || senderJID != peerPN {
		return fmt.Errorf("message secret or PN/LID lookup was not preserved or scoped")
	}
	token, err := device.PrivacyTokens.GetPrivacyToken(ctx, peerLID)
	if err != nil || token == nil || !bytes.Equal(token.Token, fixtureBytes(index, 13, 32)) ||
		!token.Timestamp.Equal(fixtureTime(index)) || !token.SenderTimestamp.Equal(fixtureTime(index).Add(time.Minute)) {
		return fmt.Errorf("privacy token or timestamps changed")
	}
	salt, err := device.NCTSalt.GetNCTSalt(ctx)
	if err != nil || !bytes.Equal(salt, fixtureBytes(index, 14, 32)) {
		return fmt.Errorf("nct salt changed")
	}
	mapping, err := device.LIDs.GetPNForLID(ctx, peerLID)
	if err != nil || mapping != peerPN {
		return fmt.Errorf("global peer LID mapping changed")
	}
	mapping, err = device.LIDs.GetLIDForPN(ctx, ownJID(index).ToNonAD())
	if err != nil || mapping != ownLID(index) {
		return fmt.Errorf("own LID mapping changed")
	}
	buffer, err := device.EventBuffer.GetBufferedEvent(ctx, fixture32(index, 15))
	if err != nil || buffer == nil || !bytes.Equal(buffer.Plaintext, fixtureBytes(index, 16, 64)) || !buffer.ServerTime.Equal(fixtureTime(index)) {
		return fmt.Errorf("decryption buffer changed")
	}
	format, outgoing, err := device.EventBuffer.GetOutgoingEvent(ctx, peerPN, peerLID, "SYNTHETIC-RETRY-MESSAGE")
	if err != nil || format != "protobuf" || !bytes.Equal(outgoing, fixtureBytes(index, 17, 64)) {
		return fmt.Errorf("outgoing retry buffer changed")
	}
	return nil
}

func writeNewFields(ctx context.Context, container *sqlstore.Container) error {
	for index := 0; index < 3; index++ {
		device, err := container.GetDevice(ctx, ownJID(index))
		if err = checked("load new-field device", err); err != nil || device == nil {
			return fmt.Errorf("failed to load new-field fixture")
		}
		field := reflect.ValueOf(device).Elem().FieldByName("CompanionMetaNonce")
		settings, supported := device.ChatSettings.(wasaStore)
		if !field.IsValid() || !field.CanSet() || field.Kind() != reflect.String || !supported {
			return fmt.Errorf("new dependency API is not available")
		}
		previous, err := settings.GetWASARootSecretID(ctx, peerPN)
		if err = checked("read initial WASA default", err); err != nil {
			return err
		}
		if field.String() != "" || previous != "" {
			return fmt.Errorf("new columns did not start with the additive migration defaults")
		}
		field.SetString(fixtureNonce(index))
		if err = checked("save companion nonce via Device.Save", device.Save(ctx)); err != nil {
			return err
		}
		if err = checked("save WASA root id via ChatSettings API", settings.PutWASARootSecretID(ctx, peerPN, fixtureWASA(index))); err != nil {
			return err
		}
	}
	return nil
}

func rewriteLegacyFields(ctx context.Context, container *sqlstore.Container) error {
	for index := 0; index < 3; index++ {
		device, err := container.GetDevice(ctx, ownJID(index))
		if err != nil || device == nil {
			return fmt.Errorf("old-code device reload failed")
		}
		if reflect.ValueOf(device).Elem().FieldByName("CompanionMetaNonce").IsValid() {
			return fmt.Errorf("rollback-write must run with the nc.4 dependency")
		}
		if err = checked("old-code Device.Save", device.Save(ctx)); err != nil {
			return err
		}
		if err = checked("old-code PutMutedUntil", device.ChatSettings.PutMutedUntil(ctx, peerPN, fixtureTime(index))); err != nil {
			return err
		}
		if err = checked("old-code PutPinned", device.ChatSettings.PutPinned(ctx, peerPN, true)); err != nil {
			return err
		}
		if err = checked("old-code PutArchived", device.ChatSettings.PutArchived(ctx, peerPN, index%2 == 0)); err != nil {
			return err
		}
	}
	return nil
}
