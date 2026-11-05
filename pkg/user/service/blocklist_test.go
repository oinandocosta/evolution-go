package user_service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type blocklistClientFake struct {
	get    func(context.Context, string) (*types.Blocklist, error)
	update func(context.Context, types.JID, events.BlocklistChangeAction, string) (*types.Blocklist, error)
}

func (f blocklistClientFake) GetBlocklist(ctx context.Context, dhash string) (*types.Blocklist, error) {
	return f.get(ctx, dhash)
}

func (f blocklistClientFake) UpdateBlocklist(ctx context.Context, jid types.JID, action events.BlocklistChangeAction, dhash string) (*types.Blocklist, error) {
	return f.update(ctx, jid, action, dhash)
}

func TestBlocklistPreservesLegacyJSONAndFiltersInactiveItems(t *testing.T) {
	list := &types.Blocklist{
		AddressingMode: types.AddressingMode("lid"),
		DHash:          "server-hash",
		Items: []types.BlocklistItem{
			{LID: types.NewJID("100", types.HiddenUserServer), PN: types.NewJID("+14155550100", types.DefaultUserServer), Active: true},
			{LID: types.NewJID("200", types.HiddenUserServer), Active: true},
			{LID: types.NewJID("300", types.HiddenUserServer), PN: types.NewJID("14155550300", types.DefaultUserServer), Active: false},
			{LID: types.NewJID("400", types.HiddenUserServer), PN: types.NewJID("invalid", types.DefaultUserServer), Active: true},
			{LID: types.NewJID("120363000000000001", types.GroupServer), Active: true},
			{LID: types.NewJID("14155550500", types.DefaultUserServer), Active: true},
			{Active: true},
		},
	}
	list.Items[0].PN.Device = 4
	list.Items[1].LID.Device = 7
	response, err := blocklistResponse(list)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"DHash":"server-hash","JIDs":["14155550100@s.whatsapp.net","200@lid","400@lid","14155550500@s.whatsapp.net"]}`
	if string(encoded) != want {
		t.Fatalf("JSON contract changed: got %s want %s", encoded, want)
	}
	if list.Items[0].PN.User != "+14155550100" || list.Items[0].PN.Device != 4 || len(list.Items) != 7 {
		t.Fatal("response conversion mutated the provider's list")
	}
	empty, err := blocklistResponse(&types.Blocklist{DHash: "empty"})
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ = json.Marshal(empty)
	if string(encoded) != `{"DHash":"empty","JIDs":null}` {
		t.Fatalf("empty-list contract changed: %s", encoded)
	}
}

func TestBlocklistRequestsFullListsAndKeepsPNLIDAndAction(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "owning-request")
	providerList := &types.Blocklist{DHash: "fresh"}
	getCalls, updateCalls := 0, 0
	client := blocklistClientFake{
		get: func(got context.Context, dhash string) (*types.Blocklist, error) {
			getCalls++
			if got != ctx || dhash != "" {
				t.Fatal("blocklist fetch changed context or used a conditional dHash")
			}
			return providerList, nil
		},
	}
	if response, err := fetchFullBlocklist(ctx, client); err != nil || response.DHash != "fresh" || getCalls != 1 {
		t.Fatalf("get result = %+v %v calls=%d", response, err, getCalls)
	}
	for _, number := range []string{"+14155550100:4@s.whatsapp.net", "123:8@lid"} {
		jid, err := parseBlocklistJID(number)
		if err != nil {
			t.Fatal(err)
		}
		for _, action := range []events.BlocklistChangeAction{events.BlocklistChangeActionBlock, events.BlocklistChangeActionUnblock} {
			client.update = func(got context.Context, target types.JID, gotAction events.BlocklistChangeAction, dhash string) (*types.Blocklist, error) {
				updateCalls++
				if got != ctx || target != jid || target.Device != 0 || gotAction != action || dhash != "" {
					t.Fatalf("unexpected update target=%s action=%s dhash=%q", target, gotAction, dhash)
				}
				return providerList, nil
			}
			if response, err := updateFullBlocklist(ctx, client, jid, action); err != nil || response.DHash != "fresh" {
				t.Fatalf("update result = %+v %v", response, err)
			}
		}
	}
	if updateCalls != 4 || getCalls != 1 {
		t.Fatalf("extra provider calls: get=%d update=%d", getCalls, updateCalls)
	}
}

func TestBlocklistInvalidTargetsDoNotStartSessions(t *testing.T) {
	var u userService
	instance := &instance_model.Instance{Id: "one"}
	for _, data := range []*BlockStruct{nil, {}, {Number: "abc@s.whatsapp.net"}, {Number: "abc@lid"}, {Number: "120363000000000001@g.us"}, {Number: "status@broadcast"}, {Number: "123@s.whatsapp.net.invalid"}} {
		for _, operation := range []func(*BlockStruct, *instance_model.Instance) (*Blocklist, error){u.BlockContact, u.UnlockContact} {
			if _, err := operation(data, instance); !errors.Is(err, ErrInvalidUserNumber) {
				t.Fatalf("invalid target %+v cause = %v", data, err)
			}
		}
	}
}

func TestBlocklistPreservesErrorCausesAndDoesNotRetry(t *testing.T) {
	for _, cause := range []error{whatsmeow.ErrRecipientNotOnWhatsApp, whatsmeow.ErrIQRateOverLimit, context.Canceled, errors.New("storage failure")} {
		calls := 0
		wrapped := fmt.Errorf("provider: %w", cause)
		client := blocklistClientFake{
			get: func(context.Context, string) (*types.Blocklist, error) { calls++; return nil, wrapped },
			update: func(context.Context, types.JID, events.BlocklistChangeAction, string) (*types.Blocklist, error) {
				calls++
				return nil, wrapped
			},
		}
		if response, err := fetchFullBlocklist(context.Background(), client); response != nil || !errors.Is(err, cause) {
			t.Fatalf("get swallowed error %v: %+v %v", cause, response, err)
		}
		if response, err := updateFullBlocklist(context.Background(), client, types.NewJID("123", types.HiddenUserServer), events.BlocklistChangeActionBlock); response != nil || !errors.Is(err, cause) {
			t.Fatalf("update swallowed error %v: %+v %v", cause, response, err)
		}
		if calls != 2 {
			t.Fatalf("unexpected retries = %d", calls)
		}
	}
	client := blocklistClientFake{get: func(context.Context, string) (*types.Blocklist, error) { return nil, nil }}
	if response, err := fetchFullBlocklist(context.Background(), client); response != nil || err == nil {
		t.Fatal("missing full response was reported as success")
	}
}
