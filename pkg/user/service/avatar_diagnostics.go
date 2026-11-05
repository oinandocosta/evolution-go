package user_service

import (
	"context"
	"errors"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// AvatarErrorCode classifies an error without exposing upstream error strings,
// which may contain private session, database, proxy or signed-URL details.
func AvatarErrorCode(err error) string {
	switch {
	case err == nil:
		return "AVATAR_QUERY_SUCCEEDED"
	case errors.Is(err, ErrInvalidUserNumber):
		return "AVATAR_INVALID_NUMBER"
	case errors.Is(err, ErrAvatarSessionUnavailable):
		return "WHATSAPP_SESSION_UNAUTHENTICATED"
	case errors.Is(err, whatsmeow.ErrProfilePictureUnauthorized):
		return "PROFILE_PICTURE_UNAUTHORIZED"
	case errors.Is(err, whatsmeow.ErrProfilePictureNotSet):
		return "PROFILE_PICTURE_NOT_SET"
	case errors.Is(err, whatsmeow.ErrIQRateOverLimit):
		return "WHATSAPP_RATE_LIMITED"
	case errors.Is(err, whatsmeow.ErrIQTimedOut), errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return "AVATAR_QUERY_TIMEOUT"
	default:
		return "AVATAR_QUERY_FAILED"
	}
}

func avatarTargetServer(jid types.JID) string {
	// Server is derived from input too: only log known protocol server names.
	switch jid.Server {
	case types.DefaultUserServer, types.HiddenUserServer, types.GroupServer, types.BroadcastServer, types.NewsletterServer:
		return jid.Server
	default:
		return "other"
	}
}

func (u *userService) logAvatarResult(instanceID, stage string, jid types.JID, preview bool, elapsed time.Duration, err error) {
	if u.loggerWrapper == nil {
		return
	}
	code := AvatarErrorCode(err)
	iqCode := 0
	var iqError *whatsmeow.IQError
	if errors.As(err, &iqError) && iqError != nil {
		iqCode = iqError.Code
	}
	format := "avatar_query category=%s stage=%s iq_code=%d preview=%t elapsed_ms=%d target_server=%s instance_id=%q"
	args := []interface{}{code, stage, iqCode, preview, elapsed.Milliseconds(), avatarTargetServer(jid), instanceID}
	log := u.loggerWrapper.GetLogger(instanceID)
	if err == nil {
		log.LogInfo(format, args...)
	} else if code == "AVATAR_QUERY_FAILED" {
		log.LogError(format, args...)
	} else {
		log.LogWarn(format, args...)
	}
}
