package instance_service

import (
	"fmt"

	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	"github.com/evolution-foundation/evolution-go/pkg/runtimestate"
)

type clientSessionState interface {
	IsConnected() bool
	IsLoggedIn() bool
}

// QR polling must never replace a session that has already authenticated.
// The returned flag tells GetQr whether it needs to wait for a fresh client.
func (i instances) startQRClient(instanceID string, client clientSessionState) (bool, error) {
	logger := i.loggerWrapper.GetLogger(instanceID)
	if client != nil && client.IsLoggedIn() {
		logger.LogInfo("[%s] Client is already logged in; skipping QR startup", instanceID)
		return false, fmt.Errorf("session already logged in")
	}
	if client != nil && client.IsConnected() {
		return false, nil
	}
	logger.LogInfo("[%s] Starting client for QR code", instanceID)
	if err := i.whatsmeowService.StartInstance(instanceID); err != nil {
		logger.LogError("[%s] Failed to start instance: %v", instanceID, err)
		return false, fmt.Errorf("failed to start instance: %w", err)
	}
	return true, nil
}

// Disconnect changes connection state only. Persisting the whole instance here
// used to clear event subscriptions and suppress webhooks after reconnect.
func (i instances) disconnectSession(instance *instance_model.Instance, client clientSessionState, signal *runtimestate.StopSignal) (*instance_model.Instance, error) {
	if client != nil && client.IsConnected() && client.IsLoggedIn() {
		i.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Disconnection successful", instance.Id)
		signal.Stop()
		instance.Connected = false
		instance.DisconnectReason = "Disconnected by API"
		if err := i.instanceRepository.UpdateConnected(instance.Id, false, instance.DisconnectReason); err != nil {
			return instance, err
		}
		return instance, nil
	}
	i.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] Ignoring disconnect as it was not connected", instance.Id)
	return instance, nil
}
