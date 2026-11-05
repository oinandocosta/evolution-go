package whatsmeow_service

import (
	"sync"
	"time"

	"go.mau.fi/whatsmeow/store"
)

var handshakeVersionMu sync.Mutex

// A rejected revision must not remain cached for the normal one-hour TTL.
// The existing reconnect path performs the lookup; no logout is needed.
func invalidateCachedWhatsAppWebVersion() {
	cachedWebVersionMu.Lock()
	defer cachedWebVersionMu.Unlock()
	cachedWebVersionAt = time.Time{}
}

// SetWAVersion updates the login user agent and registration build hash.
// DeviceProps.Version alone does not update either handshake payload.
// A lower web revision must not undo an update; explicit configuration may.
func applyHandshakeVersion(version clientVersion, allowDowngrade bool) bool {
	for _, component := range []int{version.Major, version.Minor, version.Patch} {
		if component <= 0 || uint64(component) > uint64(^uint32(0)) {
			return false
		}
	}
	next := store.WAVersionContainer{uint32(version.Major), uint32(version.Minor), uint32(version.Patch)}
	handshakeVersionMu.Lock()
	defer handshakeVersionMu.Unlock()
	current := store.GetWAVersion()
	if next == current || (!allowDowngrade && next.LessThan(current)) {
		return false
	}
	store.SetWAVersion(next)
	return true
}
