package send_service

import (
	"time"

	"go.mau.fi/whatsmeow"
)

type connectionState interface {
	IsConnected() bool
	IsLoggedIn() bool
}

// A connected websocket alone does not mean WhatsApp accepted the session.
func clientSessionReady(client connectionState) bool {
	if client == nil {
		return false
	}
	if concrete, ok := client.(*whatsmeow.Client); ok && concrete == nil {
		return false
	}
	return client.IsConnected() && client.IsLoggedIn()
}

func (s *sendService) currentClientReady(instanceID string, client *whatsmeow.Client) bool {
	return s.whatsmeowService != nil && s.whatsmeowService.IsCurrentClient(instanceID, client) && clientSessionReady(client)
}

// Presence delays remain interruptible when logout/reconnect replaces a client.
// Recheck after the timer as well, so a simultaneous replacement cannot pass.
func waitForSendDelay(delay time.Duration, ready func() bool) bool {
	if !ready() {
		return false
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-timer.C:
			return ready()
		case <-ticker.C:
			if !ready() {
				return false
			}
		}
	}
}
