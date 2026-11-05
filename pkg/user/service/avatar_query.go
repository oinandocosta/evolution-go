package user_service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	"github.com/evolution-foundation/evolution-go/pkg/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
)

const (
	avatarRequestTimeout  = 8 * time.Second
	avatarClientReadyWait = 2 * time.Second
)

var (
	ErrInvalidUserNumber        = errors.New("invalid phone number")
	ErrAvatarSessionUnavailable = errors.New("no active authenticated session found")
)

type profilePictureClient interface {
	GetProfilePictureInfo(context.Context, types.JID, *whatsmeow.GetProfilePictureParams) (*types.ProfilePictureInfo, error)
}

func parseAvatarJID(number string) (types.JID, error) {
	jid, ok := utils.ParseJID(number)
	if !ok {
		return types.EmptyJID, ErrInvalidUserNumber
	}
	jid = utils.CanonicalJID(jid).ToNonAD()
	if (jid.Server == types.DefaultUserServer || jid.Server == types.HiddenUserServer) &&
		(jid.User == "" || strings.Trim(jid.User, "0123456789") != "") {
		return types.EmptyJID, ErrInvalidUserNumber
	}
	return jid, nil
}

// Resolve only from the owning session. An unknown LID remains a LID; a failed
// lookup must never select a different session or reset the request budget.
func resolveAvatarJID(ctx context.Context, lids store.LIDStore, jid types.JID) (types.JID, error) {
	if err := ctx.Err(); err != nil {
		return types.EmptyJID, err
	}
	jid = utils.CanonicalJID(jid).ToNonAD()
	if jid.Server == types.HiddenUserServer && lids != nil {
		pn, err := lids.GetPNForLID(ctx, jid)
		if ctx.Err() != nil {
			return types.EmptyJID, ctx.Err()
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return types.EmptyJID, err
		}
		if err == nil && pn.Server == types.DefaultUserServer {
			pn = utils.CanonicalJID(pn).ToNonAD()
			if pn.User != "" && strings.Trim(pn.User, "0123456789") == "" {
				jid = pn
			}
		}
	}
	return jid, nil
}

// queryAvatar is adapted from upstream PR #120. ExistingID stays empty so an
// unchanged-picture response does not omit the URL. No image is downloaded.
func queryAvatar(ctx context.Context, client profilePictureClient, lids store.LIDStore, jid types.JID, preview bool) (*types.ProfilePictureInfo, error) {
	jid, err := resolveAvatarJID(ctx, lids, jid)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	pic, err := client.GetProfilePictureInfo(ctx, jid, &whatsmeow.GetProfilePictureParams{Preview: preview})
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, fmt.Errorf("get profile picture: %w", err)
	}
	if pic == nil || strings.TrimSpace(pic.URL) == "" {
		return nil, whatsmeow.ErrProfilePictureNotSet
	}
	return pic, nil
}

func (u *userService) currentAvatarClient(instanceID string, client *whatsmeow.Client) bool {
	return client != nil && client.Store != nil && u.whatsmeowService.IsCurrentClient(instanceID, client) &&
		client.IsConnected() && client.IsLoggedIn()
}

func (u *userService) waitForAvatarClient(ctx context.Context, instanceID string, maxWait time.Duration) (*whatsmeow.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, maxWait)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("waiting for avatar session: %w", err)
		}
		client := u.clientPointer.Get(instanceID)
		if u.currentAvatarClient(instanceID, client) {
			return client, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("waiting for avatar session: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func (u *userService) ensureAvatarClient(ctx context.Context, instanceID string) (*whatsmeow.Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	client := u.clientPointer.Get(instanceID)
	if client == nil {
		if err := u.whatsmeowService.StartInstanceContext(ctx, instanceID); err != nil {
			return nil, fmt.Errorf("start avatar session: %w", err)
		}
		return u.waitForAvatarClient(ctx, instanceID, avatarClientReadyWait)
	}
	if !u.currentAvatarClient(instanceID, client) {
		return nil, ErrAvatarSessionUnavailable
	}
	return client, nil
}

type currentAvatarQuery struct {
	service    *userService
	client     *whatsmeow.Client
	instanceID string
	targetJID  *types.JID
}

func (q currentAvatarQuery) GetProfilePictureInfo(ctx context.Context, jid types.JID, params *whatsmeow.GetProfilePictureParams) (*types.ProfilePictureInfo, error) {
	if !q.service.currentAvatarClient(q.instanceID, q.client) {
		return nil, ErrAvatarSessionUnavailable
	}
	if q.targetJID != nil {
		*q.targetJID = jid
	}
	return q.client.GetProfilePictureInfo(ctx, jid, params)
}

func (u *userService) GetAvatar(ctx context.Context, data *GetAvatarStruct, instance *instance_model.Instance) (picture *types.ProfilePictureInfo, resultErr error) {
	started := time.Now()
	stage := "validation"
	var jid types.JID
	defer func() {
		if instance != nil && instance.Id != "" {
			u.logAvatarResult(instance.Id, stage, jid, data != nil && data.Preview, time.Since(started), resultErr)
		}
	}()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, avatarRequestTimeout)
	defer cancel()
	if data == nil || instance == nil || instance.Id == "" {
		return nil, ErrInvalidUserNumber
	}
	jid, err := parseAvatarJID(data.Number)
	if err != nil {
		return nil, err
	}
	stage = "session"
	client, err := u.ensureAvatarClient(ctx, instance.Id)
	if err != nil {
		return nil, err
	}
	stage = "query"
	pic, err := queryAvatar(ctx, currentAvatarQuery{service: u, client: client, instanceID: instance.Id, targetJID: &jid}, client.Store.LIDs, jid, data.Preview)
	if err != nil {
		return nil, err
	}
	stage = "session_check"
	if !u.currentAvatarClient(instance.Id, client) {
		return nil, ErrAvatarSessionUnavailable
	}
	stage = "complete"
	return pic, nil
}
