package webhook_producer

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evolution-foundation/evolution-go/pkg/config"
	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
)

func TestWebhookLogsOmitSecretsAndResponseBody(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusBadRequest} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte("private-response-secret"))
			}))
			defer server.Close()
			assertWebhookLogSafe(t, server.URL+"/private-path?token=private-query-secret")
		})
	}
	// URL parse and transport errors include the request URL in Error().
	t.Run("invalid URL", func(t *testing.T) {
		assertWebhookLogSafe(t, "http://private-user:private-password@%/private-path?token=private-query-secret")
	})
}

func assertWebhookLogSafe(t *testing.T, destination string) {
	t.Helper()
	dir := t.TempDir()
	manager := logger_wrapper.NewLoggerManager(&config.Config{LogDirectory: dir, LogMaxSize: 1})
	producer := &webhookProducer{loggerWrapper: manager}
	producer.sendWebhookWithRetry(destination, []byte(`{"secret":"private-payload-secret"}`), 1, 0, "synthetic")
	if err := manager.GetLogger("synthetic").Close(); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "synthetic", "instance.log"))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-", destination} {
		if strings.Contains(string(content), secret) {
			t.Fatalf("webhook log disclosed remote content or credentials: %s", content)
		}
	}
	if !strings.Contains(string(content), "attempt") {
		t.Fatal("log lost attempt diagnostics")
	}
}
