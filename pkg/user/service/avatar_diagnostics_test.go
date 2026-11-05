package user_service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/config"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func TestAvatarDiagnosticsNeverLogUpstreamDetailsOrJIDs(t *testing.T) {
	secret := "sensitive-token-https://example.test/photo?auth=secret"
	phone := "14155550100"
	for _, tc := range []struct {
		name, category, server string
		err                    error
		iqCode                 int
	}{
		{"private", "PROFILE_PICTURE_UNAUTHORIZED", types.DefaultUserServer, errors.Join(whatsmeow.ErrProfilePictureUnauthorized, &whatsmeow.IQError{Code: 401, Text: secret}), 401},
		{"missing", "PROFILE_PICTURE_NOT_SET", types.HiddenUserServer, whatsmeow.ErrProfilePictureNotSet, 0},
		{"unknown", "AVATAR_QUERY_FAILED", secret, fmt.Errorf("postgres://user:password@database/%s", secret), 0},
		{"IQ error", "AVATAR_QUERY_FAILED", types.DefaultUserServer, fmt.Errorf("%s: %w", secret, &whatsmeow.IQError{Code: 500, Text: secret}), 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			manager := logger_wrapper.NewLoggerManager(&config.Config{LogDirectory: dir, LogMaxSize: 1})
			u := &userService{loggerWrapper: manager}
			u.logAvatarResult("instance-safe", "query", types.NewJID(phone, tc.server), true, 12*time.Millisecond, tc.err)
			if err := manager.GetLogger("instance-safe").Close(); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(dir, "instance-safe", "instance.log"))
			if err != nil {
				t.Fatal(err)
			}
			var entry logger_wrapper.LogEntry
			if err := json.Unmarshal(data, &entry); err != nil {
				t.Fatal(err)
			}
			wantServer := tc.server
			if wantServer == secret {
				wantServer = "other"
			}
			want := fmt.Sprintf("avatar_query category=%s stage=query iq_code=%d preview=true elapsed_ms=12 target_server=%s instance_id=\"instance-safe\"", tc.category, tc.iqCode, wantServer)
			if entry.InstanceId != "instance-safe" || entry.Message != want {
				t.Fatalf("unexpected diagnostic: %#v", entry)
			}
			for _, forbidden := range []string{secret, phone, "postgres://", "password", "https://"} {
				if strings.Contains(string(data), forbidden) {
					t.Fatalf("diagnostic leaked %q", forbidden)
				}
			}
		})
	}
}

func TestAvatarValidationFailureIsLoggedAtRuntime(t *testing.T) {
	dir := t.TempDir()
	manager := logger_wrapper.NewLoggerManager(&config.Config{LogDirectory: dir, LogMaxSize: 1})
	u := &userService{loggerWrapper: manager}
	_, err := u.GetAvatar(context.Background(), &GetAvatarStruct{Number: "abc@s.whatsapp.net"}, &instance_model.Instance{Id: "runtime-safe"})
	if !errors.Is(err, ErrInvalidUserNumber) {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := manager.GetLogger("runtime-safe").Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "runtime-safe", "instance.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "category=AVATAR_INVALID_NUMBER stage=validation") || strings.Contains(string(data), "abc") {
		t.Fatalf("unexpected runtime diagnostic: %s", data)
	}
}
