package user_service

import (
	"context"

	"go.mau.fi/whatsmeow/types"
)

type profileStatusClient interface {
	SetStatusMessage(context.Context, types.SetStatusInput) error
}

func setPlainProfileStatus(ctx context.Context, client profileStatusClient, text string) (bool, error) {
	// This endpoint sets plain profile "About" text, not a broadcast status.
	// Keep Text present even for an empty string; nil has different semantics in
	// the new API. No prefix emoji or custom expiry is requested by our contract.
	err := client.SetStatusMessage(ctx, types.SetStatusInput{Text: &text})
	if err != nil {
		return false, err
	}
	return true, nil
}
