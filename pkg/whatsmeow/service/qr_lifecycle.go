package whatsmeow_service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"
)

// A pairing completion and a QR side effect use the same lock. Cancellation of
// the rotation is separate from the connection's stop signal: PairSuccess must
// never stop the connection which has just authenticated.
type qrLifecycle struct {
	mu     sync.Mutex
	paired bool
	active *qrRotation
}

type qrRotation struct {
	done      chan struct{}
	finished  chan struct{}
	refresh   chan struct{}
	codes     []string // guarded by qrLifecycle.mu
	cancelled bool     // guarded by qrLifecycle.mu
	completed bool     // guarded by qrLifecycle.mu
}

type qrWaitResult uint8

const (
	qrWaitStopped qrWaitResult = iota
	qrWaitNext
	qrWaitRefresh
)

func (rotation *qrRotation) cancel() {
	if rotation != nil && !rotation.cancelled {
		rotation.cancelled = true
		close(rotation.done)
	}
}

func (state *qrLifecycle) begin(current func() bool, codes []string) *qrRotation {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.paired || state.active != nil || !current() {
		return nil
	}
	rotation := &qrRotation{
		done: make(chan struct{}), finished: make(chan struct{}), refresh: make(chan struct{}, 1),
		codes: append([]string(nil), codes...),
	}
	state.active = rotation
	return rotation
}

func (state *qrLifecycle) finish(rotation *qrRotation) {
	state.mu.Lock()
	defer state.mu.Unlock()
	rotation.cancel()
	if !rotation.completed {
		rotation.completed = true
		close(rotation.finished)
	}
	if state.active == rotation {
		state.active = nil
	}
}

// apply keeps persistence/enqueue ordered before or after pairing persistence.
// A successful pair cannot clear a QR and then have this worker overwrite it.
func (state *qrLifecycle) apply(rotation *qrRotation, current func() bool, action func() bool) bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.paired || state.active != rotation || rotation == nil || rotation.cancelled || !current() {
		return false
	}
	return action()
}

// Keep the current reference and remaining references on the same worker. A
// renewal changes only the ADV field; it must not reset the connection or expose
// the old QR after the renewal has been accepted under the pairing lock.
func (state *qrLifecycle) rotateADVSecret(current func() bool, oldSecret, newSecret string) {
	state.mu.Lock()
	defer state.mu.Unlock()
	rotation := state.active
	if state.paired || rotation == nil || rotation.cancelled || !current() || oldSecret == "" || newSecret == "" || oldSecret == newSecret {
		return
	}
	changed := false
	for index, code := range rotation.codes {
		parts := strings.Split(code, ",")
		if len(parts) >= 4 && parts[3] == oldSecret {
			parts[3] = newSecret
			rotation.codes[index] = strings.Join(parts, ",")
			changed = true
		}
	}
	if changed {
		select {
		case rotation.refresh <- struct{}{}:
		default:
		}
	}
}

// beginPairing returns with the lock held until the event's persistence finishes.
// The generation cleanup uses the same barrier before acknowledging completion.
func (state *qrLifecycle) beginPairing(current func() bool) (func(), bool) {
	state.mu.Lock()
	if !current() {
		state.mu.Unlock()
		return nil, false
	}
	state.paired = true
	state.active.cancel()
	return state.mu.Unlock, true
}

func (state *qrLifecycle) beginPasskey(current func() bool) (func(), bool) {
	state.mu.Lock()
	if state.paired || !current() {
		state.mu.Unlock()
		return nil, false
	}
	state.active.cancel()
	return state.mu.Unlock, true
}

func (state *qrLifecycle) barrier(action func()) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if action != nil {
		action()
	}
}

func (mycli *MyClient) currentQRGeneration() bool {
	if mycli.stopSignal == nil || mycli.killChannel == nil {
		return false
	}
	select {
	case <-mycli.stopSignal.Done():
		return false
	default:
		return mycli.killChannel.Get(mycli.userID) == mycli.stopSignal
	}
}

func (mycli *MyClient) passkeyActive() bool {
	return mycli.passkeyCeremony != nil && mycli.passkeyCeremony.HasActiveByInstance(mycli.userID)
}

func (mycli *MyClient) waitQR(rotation *qrRotation, duration time.Duration) qrWaitResult {
	if mycli.qrWait != nil {
		if mycli.qrWait(mycli.stopSignal.Done(), rotation.done, duration) {
			return qrWaitNext
		}
		return qrWaitStopped
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-mycli.stopSignal.Done():
		return qrWaitStopped
	case <-rotation.done:
		return qrWaitStopped
	case <-rotation.refresh:
		return qrWaitRefresh
	case <-timer.C:
		return qrWaitNext
	}
}

// A single worker rotates each batch. After it stops (including a passkey pause),
// a later batch may start. QR credentials are forwarded through existing event
// contracts, but are never printed in the console.
func (mycli *MyClient) handleQRCodes(codes []string) {
	rotation := mycli.qrState.begin(mycli.currentQRGeneration, codes)
	if rotation == nil {
		return
	}
	if !mycli.stopSignal.Run(func() {
		defer mycli.qrState.finish(rotation)
		published := false
		for index := 0; index < len(rotation.codes); {
			if mycli.qrBeforeAction != nil {
				mycli.qrBeforeAction("code")
			}
			if !mycli.qrState.apply(rotation, mycli.currentQRGeneration, func() bool {
				if mycli.passkeyActive() {
					return false
				}
				code := rotation.codes[index]
				image, err := qrcode.Encode(code, qrcode.Medium, 256)
				if err != nil {
					mycli.loggerWrapper.GetLogger(mycli.userID).LogError("[%s] Failed to encode QR code", mycli.userID)
					return false
				}
				base64qrcode := "data:image/png;base64," + base64.StdEncoding.EncodeToString(image)
				// Any queued wake-up is already represented by the codes read while
				// holding this lock. Later renewals wake the worker after publication.
				select {
				case <-rotation.refresh:
				default:
				}
				if !published {
					mycli.qrcodeCount++
					if mycli.config.QrcodeMaxCount > 0 && mycli.qrcodeCount >= mycli.config.QrcodeMaxCount {
						mycli.teardownQRLocked(fmt.Sprintf("Maximum QR code count (%d) reached", mycli.config.QrcodeMaxCount))
						return false
					}
				}
				mycli.loggerWrapper.GetLogger(mycli.userID).LogInfo("[%s] QR code generated #%d (max: %d)", mycli.userID, mycli.qrcodeCount, mycli.config.QrcodeMaxCount)
				if err := mycli.instanceRepository.UpdateQrcode(mycli.userID, base64qrcode+"|"+code); err != nil {
					mycli.loggerWrapper.GetLogger(mycli.userID).LogError("[%s] Error updating instance: %s", mycli.userID, err)
				}
				mycli.sendQREvent("QRCode", map[string]interface{}{
					"qrcode": base64qrcode, "code": code,
					"count": mycli.qrcodeCount, "maxCount": mycli.config.QrcodeMaxCount,
				})
				published = true
				return true
			}) {
				return
			}
			duration := 20 * time.Second
			if index == 0 {
				duration = 60 * time.Second
			}
			switch mycli.waitQR(rotation, duration) {
			case qrWaitStopped:
				return
			case qrWaitNext:
				index++
				published = false
			}
		}
		if mycli.qrBeforeAction != nil {
			mycli.qrBeforeAction("timeout")
		}
		mycli.qrState.apply(rotation, mycli.currentQRGeneration, func() bool {
			if !mycli.passkeyActive() {
				mycli.teardownQRLocked("")
			}
			return false
		})
	}) {
		mycli.qrState.finish(rotation)
	}
}

// Called only with qrState.mu held after checking the rotation and generation.
// QR expiration is not an instruction to revoke an authenticated WhatsApp device.
func (mycli *MyClient) teardownQRLocked(reason string) {
	if err := mycli.instanceRepository.UpdateQrcode(mycli.userID, ""); err != nil {
		mycli.loggerWrapper.GetLogger(mycli.userID).LogError("[%s] Error updating instance: %s", mycli.userID, err)
	}
	if reason != "" {
		if err := mycli.instanceRepository.UpdateConnected(mycli.userID, false, reason); err != nil {
			mycli.loggerWrapper.GetLogger(mycli.userID).LogError("[%s] Error updating instance status: %v", mycli.userID, err)
		}
	}
	data := map[string]interface{}{}
	if reason != "" {
		data["reason"] = reason
		data["qrcount"] = mycli.qrcodeCount
		data["maxCount"] = mycli.config.QrcodeMaxCount
		data["forceLogout"] = false
	}
	mycli.sendQREvent("QRTimeout", data)
	mycli.loggerWrapper.GetLogger(mycli.userID).LogWarn("[%s] QR timeout — stopping unpaired connection", mycli.userID)
	mycli.stopSignal.Stop()
}

// Enqueue under the QR lock so PairSuccess cannot be followed by a late enqueue
// from this rotation. Existing producers retain their own delivery/retry policy.
func (mycli *MyClient) sendQREvent(event string, data map[string]interface{}) {
	payload, err := json.Marshal(map[string]interface{}{
		"event": event, "data": data, "instanceToken": mycli.token,
		"instanceId": mycli.userID, "instanceName": mycli.Instance.Name,
	})
	if err != nil {
		mycli.loggerWrapper.GetLogger(mycli.userID).LogError("[%s] Failed to marshal QR event: %v", mycli.userID, err)
		return
	}
	mycli.service.CallWebhook(mycli.Instance, strings.ToLower(mycli.userID+"."+event), payload)
	if mycli.config.AmqpGlobalEnabled || mycli.config.NatsGlobalEnabled {
		mycli.service.SendToGlobalQueues(event, payload, mycli.userID)
	}
}
