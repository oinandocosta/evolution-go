package user_handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	auth_middleware "github.com/evolution-foundation/evolution-go/pkg/middleware"
	user_service "github.com/evolution-foundation/evolution-go/pkg/user/service"
	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type profileCompatUserServiceFake struct {
	user_service.UserService
	blocklist func(string, *user_service.BlockStruct, *instance_model.Instance) (*user_service.Blocklist, error)
	status    func(*user_service.SetProfileStatusStruct, *instance_model.Instance) (bool, error)
}

func (f profileCompatUserServiceFake) BlockContact(data *user_service.BlockStruct, instance *instance_model.Instance) (*user_service.Blocklist, error) {
	return f.blocklist("block", data, instance)
}

func (f profileCompatUserServiceFake) UnlockContact(data *user_service.BlockStruct, instance *instance_model.Instance) (*user_service.Blocklist, error) {
	return f.blocklist("unblock", data, instance)
}

func (f profileCompatUserServiceFake) GetBlockList(instance *instance_model.Instance) (*user_service.Blocklist, error) {
	return f.blocklist("list", nil, instance)
}

func (f profileCompatUserServiceFake) SetProfileStatus(data *user_service.SetProfileStatusStruct, instance *instance_model.Instance) (bool, error) {
	return f.status(data, instance)
}

func profileCompatRouter(service user_service.UserService) *gin.Engine {
	r := gin.New()
	r.Use(func(ctx *gin.Context) {
		ctx.Set("instance", &instance_model.Instance{Id: "owning-instance"})
		ctx.Next()
	})
	h := &userHandler{userService: service}
	validation := auth_middleware.NewJIDValidationMiddleware()
	r.POST("/user/block", validation.ValidateNumberField(), h.BlockContact)
	r.POST("/user/unblock", validation.ValidateNumberField(), h.UnblockContact)
	r.GET("/user/blocklist", h.GetBlockList)
	r.POST("/user/profileStatus", h.SetProfileStatus)
	return r
}

func profileCompatRequest(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestBlocklistHTTPPreservesLegacyResponseAndSessionScope(t *testing.T) {
	for _, operation := range []string{"block", "unblock", "list"} {
		t.Run(operation, func(t *testing.T) {
			calls := 0
			service := profileCompatUserServiceFake{blocklist: func(got string, data *user_service.BlockStruct, instance *instance_model.Instance) (*user_service.Blocklist, error) {
				calls++
				if got != operation || instance.Id != "owning-instance" {
					t.Fatal("operation or owning instance changed")
				}
				if operation != "list" && (data == nil || data.Number == "") {
					t.Fatal("missing contact target")
				}
				return &user_service.Blocklist{DHash: "fresh", JIDs: []types.JID{types.NewJID("14155550100", types.DefaultUserServer), types.NewJID("123", types.HiddenUserServer)}}, nil
			}}
			method, path := http.MethodPost, "/user/"+operation
			if operation == "list" {
				method, path = http.MethodGet, "/user/blocklist"
			}
			response := profileCompatRequest(profileCompatRouter(service), method, path, `{"number":"14155550100","instanceId":"different-instance"}`)
			want := `{"data":{"DHash":"fresh","JIDs":["14155550100@s.whatsapp.net","123@lid"]},"message":"success"}`
			if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != want || calls != 1 {
				t.Fatalf("status=%d body=%s calls=%d", response.Code, response.Body.String(), calls)
			}
		})
	}
}

func TestBlocklistHTTPSemanticErrors(t *testing.T) {
	for _, path := range []string{"/user/block", "/user/unblock"} {
		for _, tc := range []struct {
			cause  error
			status int
		}{
			{fmt.Errorf("provider: %w", user_service.ErrInvalidUserNumber), http.StatusBadRequest},
			{fmt.Errorf("resolve LID: %w", whatsmeow.ErrRecipientNotOnWhatsApp), http.StatusBadRequest},
			{errors.New("provider failure"), http.StatusInternalServerError},
		} {
			service := profileCompatUserServiceFake{blocklist: func(string, *user_service.BlockStruct, *instance_model.Instance) (*user_service.Blocklist, error) {
				return nil, tc.cause
			}}
			response := profileCompatRequest(profileCompatRouter(service), http.MethodPost, path, `{"number":"123@lid"}`)
			if response.Code != tc.status || !strings.Contains(response.Body.String(), `"error":`) {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		}
	}
}

func TestProfileUpdatesHTTPRejectsInvalidBodies(t *testing.T) {
	service := profileCompatUserServiceFake{
		blocklist: func(string, *user_service.BlockStruct, *instance_model.Instance) (*user_service.Blocklist, error) {
			t.Fatal("provider called for invalid body")
			return nil, nil
		},
		status: func(*user_service.SetProfileStatusStruct, *instance_model.Instance) (bool, error) {
			t.Fatal("status called for invalid body")
			return false, nil
		},
	}
	for _, path := range []string{"/user/block", "/user/unblock", "/user/profileStatus"} {
		for _, body := range []string{"null", "{}", "[]", "{"} {
			response := profileCompatRequest(profileCompatRouter(service), http.MethodPost, path, body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("%s body=%s status=%d", path, body, response.Code)
			}
		}
	}
}

func TestProfileStatusHTTPPreservesPlainTextResponse(t *testing.T) {
	calls := 0
	service := profileCompatUserServiceFake{status: func(data *user_service.SetProfileStatusStruct, instance *instance_model.Instance) (bool, error) {
		calls++
		if data.Status != "Atendimento 🛠️\n das 9h às 18h " || instance.Id != "owning-instance" {
			t.Fatal("profile text or owning instance changed")
		}
		return true, nil
	}}
	response := profileCompatRequest(profileCompatRouter(service), http.MethodPost, "/user/profileStatus", `{"status":"Atendimento 🛠️\n das 9h às 18h ","instanceId":"other"}`)
	want := `{"data":{"status":"Atendimento 🛠️\n das 9h às 18h "},"message":"success"}`
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != want || calls != 1 {
		t.Fatalf("status=%d body=%s calls=%d", response.Code, response.Body.String(), calls)
	}
}
