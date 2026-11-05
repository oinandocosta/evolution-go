package user_service

import (
	"context"
	"errors"
	"strings"

	"github.com/evolution-foundation/evolution-go/pkg/utils"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// Blocklist preserves the public DHash/JIDs response while WhatsMeow's internal
// list now contains LID/PN pairs and inactive entries.
type Blocklist struct {
	DHash string      `json:"DHash"`
	JIDs  []types.JID `json:"JIDs"`
}

type blocklistClient interface {
	GetBlocklist(context.Context, string) (*types.Blocklist, error)
	UpdateBlocklist(context.Context, types.JID, events.BlocklistChangeAction, string) (*types.Blocklist, error)
}

func normalizeBlocklistJID(jid types.JID) (types.JID, bool) {
	jid = utils.CanonicalJID(jid).ToNonAD()
	valid := (jid.Server == types.DefaultUserServer || jid.Server == types.HiddenUserServer) &&
		jid.User != "" && strings.Trim(jid.User, "0123456789") == ""
	return jid, valid
}

func parseBlocklistJID(number string) (types.JID, error) {
	jid, ok := utils.ParseJID(number)
	if !ok {
		return types.EmptyJID, ErrInvalidUserNumber
	}
	jid, ok = normalizeBlocklistJID(jid)
	if !ok {
		return types.EmptyJID, ErrInvalidUserNumber
	}
	return jid, nil
}

func blocklistResponse(list *types.Blocklist) (*Blocklist, error) {
	if list == nil {
		return nil, errors.New("WhatsApp did not return the full block list")
	}
	response := &Blocklist{DHash: list.DHash}
	for _, item := range list.Items {
		if !item.Active {
			continue
		}
		jid, valid := normalizeBlocklistJID(item.PN)
		if !valid || jid.Server != types.DefaultUserServer {
			jid, valid = normalizeBlocklistJID(item.LID)
		}
		if valid {
			response.JIDs = append(response.JIDs, jid)
		}
	}
	return response, nil
}

func fetchFullBlocklist(ctx context.Context, client blocklistClient) (*Blocklist, error) {
	// A cached dHash can suppress the list entirely. This API has no conditional
	// response contract, so always request the full current list.
	list, err := client.GetBlocklist(ctx, "")
	if err != nil {
		return nil, err
	}
	return blocklistResponse(list)
}

func updateFullBlocklist(ctx context.Context, client blocklistClient, jid types.JID, action events.BlocklistChangeAction) (*Blocklist, error) {
	// WhatsMeow owns the PN->LID lookup, server lookup on cache miss and PN hint
	// for blocking a known LID. Do not resolve through another session's store.
	list, err := client.UpdateBlocklist(ctx, jid, action, "")
	if err != nil {
		return nil, err
	}
	return blocklistResponse(list)
}
