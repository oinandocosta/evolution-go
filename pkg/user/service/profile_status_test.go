package user_service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go.mau.fi/whatsmeow/types"
)

type profileStatusClientFake struct {
	set func(context.Context, types.SetStatusInput) error
}

func (f profileStatusClientFake) SetStatusMessage(ctx context.Context, input types.SetStatusInput) error {
	return f.set(ctx, input)
}

func TestProfileStatusPreservesPlainTextAndNoAddedEmoji(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, text := range []string{"Atendimento 🛠️\n das 9h às 18h ", ""} {
		calls := 0
		client := profileStatusClientFake{set: func(got context.Context, input types.SetStatusInput) error {
			calls++
			if got != ctx || input.Text == nil || *input.Text != text || input.Emoji != nil || !input.Duration.IsZero() {
				t.Fatalf("status text or optional fields changed: %+v", input)
			}
			encoded, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			want := `{"text":"Atendimento 🛠️\n das 9h às 18h ","ephemeral_duration_sec":0}`
			if text == "" {
				want = `{"text":"","ephemeral_duration_sec":0}`
			}
			if string(encoded) != want {
				t.Fatalf("unexpected WhatsMeow input JSON: got %s want %s", encoded, want)
			}
			return nil
		}}
		if ok, err := setPlainProfileStatus(ctx, client, text); !ok || err != nil || calls != 1 {
			t.Fatalf("status result = %v %v calls=%d", ok, err, calls)
		}
	}
}

func TestProfileStatusPreservesProviderErrorWithoutFalseSuccess(t *testing.T) {
	cause := errors.New("status update rejected")
	calls := 0
	client := profileStatusClientFake{set: func(context.Context, types.SetStatusInput) error { calls++; return cause }}
	if ok, err := setPlainProfileStatus(context.Background(), client, "About"); ok || !errors.Is(err, cause) || calls != 1 {
		t.Fatalf("failed update = %v %v calls=%d", ok, err, calls)
	}
}
