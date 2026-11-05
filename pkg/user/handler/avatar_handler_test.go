package user_handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evolution-foundation/evolution-go/pkg/config"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	instance_service "github.com/evolution-foundation/evolution-go/pkg/instance/service"
	auth_middleware "github.com/evolution-foundation/evolution-go/pkg/middleware"
	user_service "github.com/evolution-foundation/evolution-go/pkg/user/service"
	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type avatarUserServiceFake struct {
	user_service.UserService
	query func(context.Context, *user_service.GetAvatarStruct, *instance_model.Instance) (*types.ProfilePictureInfo, error)
}

func (f avatarUserServiceFake) GetAvatar(ctx context.Context, data *user_service.GetAvatarStruct, instance *instance_model.Instance) (*types.ProfilePictureInfo, error) {
	return f.query(ctx, data, instance)
}

type avatarInstancesFake struct {
	instance_service.InstanceService
}

func (avatarInstancesFake) GetInstanceByToken(token string) (*instance_model.Instance, error) {
	if token == "token-one" {
		return &instance_model.Instance{Id: "one"}, nil
	}
	if token == "token-two" {
		return &instance_model.Instance{Id: "two"}, nil
	}
	if token == "nil-instance" {
		return nil, nil
	}
	return nil, errors.New("unknown token")
}

func avatarRouter(service user_service.UserService) *gin.Engine {
	r := gin.New()
	middleware := auth_middleware.NewMiddleware(&config.Config{}, avatarInstancesFake{})
	validation := auth_middleware.NewJIDValidationMiddleware()
	h := &userHandler{userService: service}
	r.POST("/user/avatar", middleware.Auth, validation.ValidateNumberField(), h.GetAvatar)
	return r
}

func avatarHTTPRequest(r *gin.Engine, token, body string, ctx context.Context) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/user/avatar", strings.NewReader(body)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("apikey", token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAvatarHTTPRejectsInvalidBodiesAndMissingAuth(t *testing.T) {
	service := avatarUserServiceFake{query: func(context.Context, *user_service.GetAvatarStruct, *instance_model.Instance) (*types.ProfilePictureInfo, error) {
		t.Fatal("service called for rejected request")
		return nil, nil
	}}
	r := avatarRouter(service)
	for _, body := range []string{"null", "{}", "{", "", "[]", `{"number":null}`, `{"number":42}`, `{"number":[]}`, `{"number":"14155550100","preview":"yes"}`} {
		t.Run(body, func(t *testing.T) {
			w := avatarHTTPRequest(r, "token-one", body, context.Background())
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
	for _, token := range []string{"", "invalid-token"} {
		w := avatarHTTPRequest(r, token, `{"number":"14155550100"}`, context.Background())
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("auth status=%d", w.Code)
		}
	}
	w := avatarHTTPRequest(r, "nil-instance", `{"number":"14155550100"}`, context.Background())
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("nil instance status=%d", w.Code)
	}
}

func TestAvatarHTTPSessionScopeAndSuccessContract(t *testing.T) {
	for _, tc := range []struct {
		token, id string
		preview   bool
	}{{"token-one", "one", false}, {"token-two", "two", true}} {
		t.Run(tc.id, func(t *testing.T) {
			calls := 0
			var hash []byte
			if tc.preview {
				hash = []byte{1, 2, 3}
			}
			picture := &types.ProfilePictureInfo{URL: "https://example.test/photo", ID: "photo", Type: "image", DirectPath: "/photo", Hash: hash}
			service := avatarUserServiceFake{query: func(ctx context.Context, data *user_service.GetAvatarStruct, instance *instance_model.Instance) (*types.ProfilePictureInfo, error) {
				calls++
				if instance.Id != tc.id || data.Preview != tc.preview || data.Number == "" {
					t.Fatal("request switched session or lost preview")
				}
				return picture, nil
			}}
			body := fmt.Sprintf(`{"number":"14155550100","preview":%t,"instanceId":"other"}`, tc.preview)
			w := avatarHTTPRequest(avatarRouter(service), tc.token, body, context.Background())
			wantHash := "null"
			if tc.preview {
				wantHash = `"AQID"`
			}
			// Assert the wire contract independently of the dependency's JSON tags.
			want := fmt.Sprintf(`{"message":"success","data":{"url":"https://example.test/photo","id":"photo","type":"image","direct_path":"/photo","hash":%s}}`, wantHash)
			var gotJSON, wantJSON interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &gotJSON); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(want), &wantJSON); err != nil {
				t.Fatal(err)
			}
			gotNormalized, _ := json.Marshal(gotJSON)
			wantNormalized, _ := json.Marshal(wantJSON)
			if w.Code != http.StatusOK || calls != 1 || string(gotNormalized) != string(wantNormalized) {
				t.Fatalf("status=%d got=%s want=%s calls=%d", w.Code, gotNormalized, wantNormalized, calls)
			}
		})
	}
}

func TestAvatarHTTPContextAndErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"invalid number", user_service.ErrInvalidUserNumber, http.StatusBadRequest, "AVATAR_INVALID_NUMBER"},
		{"unauthenticated session", user_service.ErrAvatarSessionUnavailable, http.StatusUnauthorized, "WHATSAPP_SESSION_UNAUTHENTICATED"},
		{"rate limit", fmt.Errorf("private details: %w", whatsmeow.ErrIQRateOverLimit), http.StatusTooManyRequests, "WHATSAPP_RATE_LIMITED"},
		{"IQ timeout", whatsmeow.ErrIQTimedOut, http.StatusGatewayTimeout, "AVATAR_QUERY_TIMEOUT"},
		{"deadline", context.DeadlineExceeded, http.StatusGatewayTimeout, "AVATAR_QUERY_TIMEOUT"},
		{"cancellation", context.Canceled, http.StatusGatewayTimeout, "AVATAR_QUERY_TIMEOUT"},
		{"private photo", fmt.Errorf("private details: %w", whatsmeow.ErrProfilePictureUnauthorized), http.StatusForbidden, "PROFILE_PICTURE_UNAUTHORIZED"},
		{"missing photo", fmt.Errorf("private details: %w", whatsmeow.ErrProfilePictureNotSet), http.StatusNotFound, "PROFILE_PICTURE_NOT_SET"},
		{"internal error", errors.New("postgres://user:private-password@database"), http.StatusInternalServerError, "AVATAR_QUERY_FAILED"},
		{"upstream bad request", whatsmeow.ErrIQBadRequest, http.StatusInternalServerError, "AVATAR_QUERY_FAILED"},
		{"upstream server error", whatsmeow.ErrIQInternalServerError, http.StatusInternalServerError, "AVATAR_QUERY_FAILED"},
		{"nil result", nil, http.StatusNotFound, "PROFILE_PICTURE_NOT_SET"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			service := avatarUserServiceFake{query: func(gotCtx context.Context, _ *user_service.GetAvatarStruct, instance *instance_model.Instance) (*types.ProfilePictureInfo, error) {
				if !errors.Is(gotCtx.Err(), context.Canceled) || instance.Id != "one" {
					t.Fatal("lost request context or instance")
				}
				return nil, tc.err
			}}
			w := avatarHTTPRequest(avatarRouter(service), "token-one", `{"number":"14155550100"}`, ctx)
			var body map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != tc.status || body["code"] != tc.code || body["error"] == "" || len(body) != 2 || strings.Contains(w.Body.String(), "private") || strings.Contains(w.Body.String(), "postgres") {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAvatarHTTPUnavailablePhotoLiteralContract(t *testing.T) {
	for _, tc := range []struct {
		name    string
		picture *types.ProfilePictureInfo
		err     error
		status  int
		body    string
	}{
		{"private", nil, whatsmeow.ErrProfilePictureUnauthorized, 403, `{"code":"PROFILE_PICTURE_UNAUTHORIZED","error":"failed to query WhatsApp"}`},
		{"missing", nil, whatsmeow.ErrProfilePictureNotSet, 404, `{"code":"PROFILE_PICTURE_NOT_SET","error":"failed to query WhatsApp"}`},
		{"nil photo", nil, nil, 404, `{"code":"PROFILE_PICTURE_NOT_SET","error":"failed to query WhatsApp"}`},
		{"empty URL", &types.ProfilePictureInfo{ID: "picture"}, nil, 404, `{"code":"PROFILE_PICTURE_NOT_SET","error":"failed to query WhatsApp"}`},
		{"blank URL", &types.ProfilePictureInfo{URL: " \t\n"}, nil, 404, `{"code":"PROFILE_PICTURE_NOT_SET","error":"failed to query WhatsApp"}`},
		{"real error", nil, whatsmeow.ErrIQInternalServerError, 500, `{"code":"AVATAR_QUERY_FAILED","error":"failed to query WhatsApp"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := avatarUserServiceFake{query: func(context.Context, *user_service.GetAvatarStruct, *instance_model.Instance) (*types.ProfilePictureInfo, error) {
				return tc.picture, tc.err
			}}
			w := avatarHTTPRequest(avatarRouter(service), "token-one", `{"number":"14155550100"}`, context.Background())
			if w.Code != tc.status || w.Body.String() != tc.body {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}
