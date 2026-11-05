package send_service

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/config"
	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
)

func previewTestService(t *testing.T) (*sendService, string) {
	t.Helper()
	directory := t.TempDir()
	manager := logger_wrapper.NewLoggerManager(&config.Config{LogDirectory: directory, LogMaxSize: 1})
	t.Cleanup(func() { _ = manager.GetLogger("synthetic-instance").Close() })
	return &sendService{loggerWrapper: manager}, directory
}

func TestBuildLinkMessageMetadataFailureKeepsTextWithoutLeakingURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	service, directory := previewTestService(t)
	secretURL := strings.Replace(server.URL, "http://", "http://user:private-password@", 1) + "/private-path?token=private-token"
	data := &LinkStruct{Text: "Original text survives", Url: secretURL, Description: "Caller description"}
	msg := service.buildLinkMessage(nil, data, "synthetic-instance")
	if msg.GetExtendedTextMessage().GetText() != data.Text || msg.GetExtendedTextMessage().GetDescription() != data.Description {
		t.Fatal("failed preview discarded caller text or metadata")
	}
	contents, err := os.ReadFile(filepath.Join(directory, "synthetic-instance", "instance.log"))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-password", "private-token", "private-path", secretURL} {
		if strings.Contains(string(contents), secret) {
			t.Fatalf("preview log leaked %q", secret)
		}
	}
}

func TestBuildLinkMessageNewsletterUsesInlineThumbnailWithoutAuthenticatedClient(t *testing.T) {
	png := encodePNG(t, 640, 320)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(png) }))
	defer server.Close()
	service, _ := previewTestService(t)
	msg := service.buildLinkMessage(nil, &LinkStruct{
		Text: "Newsletter preview", Number: "synthetic@newsletter", Url: "https://synthetic.example",
		Title: "Caller title", ImgUrl: server.URL,
	}, "synthetic-instance")
	ext := msg.GetExtendedTextMessage()
	assertJPEG(t, ext.GetJPEGThumbnail(), 72, 36)
	if ext.GetThumbnailDirectPath() != "" || len(ext.GetMediaKey()) != 0 {
		t.Fatal("newsletter preview unexpectedly used encrypted thumbnail fields")
	}
}

func TestFetchLinkPreviewResourceTransportErrorOmitsCredentialsAndQuery(t *testing.T) {
	_, err := fetchLinkPreviewResource("unsupported://user:private-password@synthetic.invalid/private-path?token=private-token", 1024)
	if err == nil {
		t.Fatal("expected unsupported transport error")
	}
	for _, secret := range []string{"private-password", "private-token", "private-path"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("resource error leaked %q", secret)
		}
	}
}

func TestSetLinkThumbnailUploadFailureKeepsInlineImage(t *testing.T) {
	prepared, err := prepareLinkPreviewImage(encodePNG(t, 640, 320))
	if err != nil {
		t.Fatal(err)
	}
	ext := &waE2E.ExtendedTextMessage{}
	sentinel := errors.New("synthetic upload failure")
	err = setLinkThumbnail(ext, prepared, func(ctx context.Context, content []byte, mediaType whatsmeow.MediaType) (whatsmeow.UploadResponse, error) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > linkPreviewUploadTimeout || time.Until(deadline) <= 0 {
			t.Fatal("thumbnail upload must have a bounded deadline")
		}
		if mediaType != whatsmeow.MediaLinkThumbnail || !bytes.Equal(content, prepared.HighQuality) {
			t.Fatal("incorrect thumbnail upload content or media type")
		}
		return whatsmeow.UploadResponse{}, sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("upload failure = %v", err)
	}
	assertJPEG(t, ext.GetJPEGThumbnail(), 72, 36)
	if ext.GetThumbnailDirectPath() != "" || len(ext.GetMediaKey()) != 0 {
		t.Fatal("failed upload left invalid encrypted thumbnail metadata")
	}
}

func TestFetchLinkMetadataResolvesImageAgainstRedirectDestination(t *testing.T) {
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/products/item/page" {
			t.Errorf("unexpected destination path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`<html><head><meta property="og:image" content="images/photo.jpg"></head></html>`))
	}))
	defer destination.Close()
	shortLink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL+"/products/item/page", http.StatusFound)
	}))
	defer shortLink.Close()
	_, _, imageURL, err := fetchLinkMetadata(shortLink.URL + "/short")
	if err != nil {
		t.Fatal(err)
	}
	if want := destination.URL + "/products/item/images/photo.jpg"; imageURL != want {
		t.Fatalf("image URL = %q, want %q", imageURL, want)
	}
}
