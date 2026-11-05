package whatsmeow_service

import (
	"encoding/json"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/evolution-foundation/evolution-go/pkg/config"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	instance_repository "github.com/evolution-foundation/evolution-go/pkg/instance/repository"
	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
	"github.com/evolution-foundation/evolution-go/pkg/passkey/ceremony"
	"github.com/evolution-foundation/evolution-go/pkg/runtimestate"
)

type qrTestGate struct {
	entered, release chan struct{}
	enterOnce, once  sync.Once
}

func newQRTestGate(t *testing.T) *qrTestGate {
	t.Helper()
	gate := &qrTestGate{entered: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(gate.open)
	return gate
}
func (gate *qrTestGate) hold() { gate.enterOnce.Do(func() { close(gate.entered) }); <-gate.release }
func (gate *qrTestGate) open() { gate.once.Do(func() { close(gate.release) }) }

func awaitQRTest(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("QR lifecycle operation did not complete")
	}
}

type qrTestRepository struct {
	instance_repository.InstanceRepository
	mu            sync.Mutex
	instance      instance_model.Instance
	operations    []string
	beforeQR      func(string)
	beforePair    func()
	beforeOffline func()
	qrWrites      int
	offlineWrites int
	pairWrites    int
}

func (repo *qrTestRepository) GetInstanceByID(string) (*instance_model.Instance, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	copy := repo.instance
	return &copy, nil
}
func (repo *qrTestRepository) Update(instance *instance_model.Instance) error {
	if repo.beforePair != nil {
		repo.beforePair()
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.instance = *instance
	repo.operations = append(repo.operations, "pair")
	repo.pairWrites++
	return nil
}
func (repo *qrTestRepository) UpdateQrcode(_ string, code string) error {
	if repo.beforeQR != nil {
		repo.beforeQR(code)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.instance.Qrcode = code
	repo.operations = append(repo.operations, "qr")
	repo.qrWrites++
	return nil
}
func (repo *qrTestRepository) UpdateConnected(_ string, connected bool, reason string) error {
	if !connected && repo.beforeOffline != nil {
		repo.beforeOffline()
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.instance.Connected = connected
	repo.instance.DisconnectReason = reason
	repo.operations = append(repo.operations, "offline")
	repo.offlineWrites++
	return nil
}
func (repo *qrTestRepository) snapshot() (instance_model.Instance, []string, int, int, int) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	return repo.instance, append([]string(nil), repo.operations...), repo.qrWrites, repo.offlineWrites, repo.pairWrites
}

type qrTestEvents struct {
	WhatsmeowService
	mu     sync.Mutex
	events []string
	forced bool
}

func (sink *qrTestEvents) CallWebhook(_ *instance_model.Instance, _ string, payload []byte) {
	var event struct {
		Event string `json:"event"`
		Data  struct {
			ForceLogout bool `json:"forceLogout"`
		} `json:"data"`
	}
	_ = json.Unmarshal(payload, &event)
	sink.mu.Lock()
	defer sink.mu.Unlock()
	sink.events = append(sink.events, event.Event)
	sink.forced = sink.forced || event.Data.ForceLogout
}
func (sink *qrTestEvents) count(event string) int {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	count := 0
	for _, got := range sink.events {
		if got == event {
			count++
		}
	}
	return count
}

func (sink *qrTestEvents) forcedLogout() bool {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	return sink.forced
}

func newQRTestClient(t *testing.T) (*MyClient, *qrTestRepository, *qrTestEvents) {
	t.Helper()
	instance := instance_model.Instance{Id: "qr-test-instance", Name: "QR test", Token: "disposable-token"}
	repository := &qrTestRepository{instance: instance}
	sink := &qrTestEvents{}
	stop := runtimestate.NewStopSignal()
	if !stop.Begin() {
		t.Fatal("could not start disposable generation")
	}
	registry := runtimestate.New[string, *runtimestate.StopSignal]()
	registry.Set(instance.Id, stop)
	client := &MyClient{
		service: sink, WAClient: &whatsmeow.Client{Store: &store.Device{}},
		userID: instance.Id, token: instance.Token, Instance: &instance,
		instanceRepository: repository, stopSignal: stop, killChannel: registry,
		userInfoCache: cache.New(time.Minute, time.Minute),
		config:        &config.Config{LogType: "json"}, passkeyCeremony: ceremony.NewStore(),
		loggerWrapper: logger_wrapper.NewLoggerManager(&config.Config{LogDirectory: t.TempDir()}),
	}
	t.Cleanup(func() {
		stop.Stop()
		go func() { client.qrState.barrier(nil); stop.Complete() }()
		if !stop.Wait(3 * time.Second) {
			t.Error("disposable QR generation did not stop")
		}
	})
	return client, repository, sink
}

func qrTestRotation(client *MyClient) *qrRotation {
	client.qrState.mu.Lock()
	defer client.qrState.mu.Unlock()
	return client.qrState.active
}
func qrTestWaitUntilCancelled(stop, paired <-chan struct{}, _ time.Duration) bool {
	select {
	case <-stop:
	case <-paired:
	}
	return false
}
func qrTestPair(client *MyClient) {
	client.myEventHandler(&events.PairSuccess{ID: types.NewJID("10000000000", types.DefaultUserServer)})
}
func assertQRTestRunning(t *testing.T, client *MyClient) {
	t.Helper()
	select {
	case <-client.stopSignal.Done():
		t.Fatal("pairing stopped the authenticated connection")
	default:
	}
}

func TestQRPairSuccessWinsBeforeCodeLimitOrTimeout(t *testing.T) {
	for _, scenario := range []struct {
		name, phase string
		limit       int
		priorWrites int
	}{
		{name: "late-code", phase: "code"},
		{name: "max-count", phase: "code", limit: 1},
		{name: "timeout", phase: "timeout", priorWrites: 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			client, repo, sink := newQRTestClient(t)
			client.config.QrcodeMaxCount = scenario.limit
			gate := newQRTestGate(t)
			client.qrBeforeAction = func(phase string) {
				if phase == scenario.phase {
					gate.hold()
				}
			}
			client.qrWait = func(<-chan struct{}, <-chan struct{}, time.Duration) bool { return true }
			client.handleQRCodes([]string{"disposable-qr"})
			awaitQRTest(t, gate.entered)
			rotation := qrTestRotation(client)
			qrTestPair(client)
			assertQRTestRunning(t, client)
			gate.open()
			awaitQRTest(t, rotation.finished)
			client.handleQRCodes([]string{"late-qr-after-pairing"})
			instance, _, writes, offline, pairs := repo.snapshot()
			if instance.Qrcode != "" || !instance.Connected || pairs != 1 || writes != scenario.priorWrites || offline != 0 {
				t.Fatalf("late QR changed paired state: instance=%+v qr=%d offline=%d pair=%d", instance, writes, offline, pairs)
			}
			if sink.count("QRCode") != scenario.priorWrites || sink.count("QRTimeout") != 0 {
				t.Fatal("a completed pairing was followed by QR or timeout emission")
			}
		})
	}
}

func TestQRPublicationFinishesBeforePairingClearsIt(t *testing.T) {
	client, repo, sink := newQRTestClient(t)
	gate := newQRTestGate(t)
	repo.beforeQR = func(code string) {
		if code != "" {
			gate.hold()
		}
	}
	client.qrWait = qrTestWaitUntilCancelled
	client.handleQRCodes([]string{"disposable-qr"})
	awaitQRTest(t, gate.entered)
	pairDone := make(chan struct{})
	go func() { qrTestPair(client); close(pairDone) }()
	gate.open()
	awaitQRTest(t, pairDone)
	if rotation := qrTestRotation(client); rotation != nil {
		awaitQRTest(t, rotation.finished)
	}
	instance, operations, writes, offline, pairs := repo.snapshot()
	if !reflect.DeepEqual(operations, []string{"qr", "pair"}) || writes != 1 || pairs != 1 || offline != 0 || instance.Qrcode != "" || !instance.Connected {
		t.Fatalf("publication and pairing were not ordered: operations=%v instance=%+v", operations, instance)
	}
	if sink.count("QRCode") != 1 || sink.count("QRTimeout") != 0 {
		t.Fatal("unexpected QR event after pairing")
	}
	assertQRTestRunning(t, client)
}

func TestQRTerminalActionWinsBeforeLatePairing(t *testing.T) {
	for _, maxCount := range []int{0, 1} {
		t.Run(map[int]string{0: "timeout", 1: "max-count"}[maxCount], func(t *testing.T) {
			client, repo, sink := newQRTestClient(t)
			client.config.QrcodeMaxCount = maxCount
			// Simulate credentials having arrived before their PairSuccess event.
			// Rotation must not read or revoke this asynchronously-owned Store.ID.
			jid := types.NewJID("10000000000", types.DefaultUserServer)
			client.WAClient.Store.ID = &jid
			gate := newQRTestGate(t)
			if maxCount > 0 {
				repo.beforeOffline = gate.hold
			} else {
				repo.beforeQR = func(code string) {
					if code == "" {
						gate.hold()
					}
				}
			}
			client.qrWait = func(<-chan struct{}, <-chan struct{}, time.Duration) bool { return true }
			client.handleQRCodes([]string{"disposable-qr"})
			awaitQRTest(t, gate.entered)
			pairDone := make(chan struct{})
			go func() { qrTestPair(client); close(pairDone) }()
			gate.open()
			awaitQRTest(t, client.stopSignal.Done())
			awaitQRTest(t, pairDone)
			if rotation := qrTestRotation(client); rotation != nil {
				awaitQRTest(t, rotation.finished)
			}
			instance, _, _, _, pairs := repo.snapshot()
			if pairs != 0 || instance.Qrcode != "" || sink.count("QRTimeout") != 1 {
				t.Fatal("late pairing modified an already-stopped generation")
			}
			if client.WAClient.Store.ID == nil || *client.WAClient.Store.ID != jid || sink.forcedLogout() {
				t.Fatal("QR expiration revoked the device credentials")
			}
		})
	}
}

func TestQRPairingMarksCompletionBeforeItsPersistence(t *testing.T) {
	client, repo, sink := newQRTestClient(t)
	codeGate := newQRTestGate(t)
	pairGate := newQRTestGate(t)
	client.qrBeforeAction = func(string) { codeGate.hold() }
	repo.beforePair = pairGate.hold
	client.handleQRCodes([]string{"disposable-qr"})
	awaitQRTest(t, codeGate.entered)
	rotation := qrTestRotation(client)
	pairDone := make(chan struct{})
	go func() { qrTestPair(client); close(pairDone) }()
	awaitQRTest(t, pairGate.entered)
	awaitQRTest(t, rotation.done)
	assertQRTestRunning(t, client)
	codeGate.open()
	pairGate.open()
	awaitQRTest(t, pairDone)
	awaitQRTest(t, rotation.finished)
	instance, _, writes, offline, pairs := repo.snapshot()
	if writes != 0 || offline != 0 || pairs != 1 || !instance.Connected || instance.Qrcode != "" || sink.count("QRCode") != 0 {
		t.Fatal("QR effects escaped while pairing persistence was in progress")
	}
}

func TestQRDuplicateBatchesHaveOnlyOneWorker(t *testing.T) {
	client, repo, sink := newQRTestClient(t)
	gate := newQRTestGate(t)
	var entries atomic.Int32
	client.qrBeforeAction = func(string) { entries.Add(1); gate.hold() }
	client.qrWait = qrTestWaitUntilCancelled
	client.handleQRCodes([]string{"first-disposable-qr"})
	awaitQRTest(t, gate.entered)
	rotation := qrTestRotation(client)
	var callers sync.WaitGroup
	for index := 0; index < 32; index++ {
		callers.Add(1)
		go func() { defer callers.Done(); client.handleQRCodes([]string{"duplicate-qr"}) }()
	}
	callers.Wait()
	gate.open()
	qrTestPair(client)
	awaitQRTest(t, rotation.finished)
	_, _, writes, _, _ := repo.snapshot()
	if entries.Load() != 1 || writes > 1 || sink.count("QRCode") > 1 {
		t.Fatalf("duplicate QR workers escaped: workers=%d writes=%d", entries.Load(), writes)
	}
}

func TestQRPasskeyPauseAllowsLaterBatch(t *testing.T) {
	client, repo, _ := newQRTestClient(t)
	client.passkeyCeremony.Start(client.userID, json.RawMessage(`{}`))
	gate := newQRTestGate(t)
	client.qrBeforeAction = func(string) { gate.hold() }
	client.handleQRCodes([]string{"paused-for-passkey"})
	awaitQRTest(t, gate.entered)
	rotation := qrTestRotation(client)
	gate.open()
	awaitQRTest(t, rotation.finished)
	_, _, writes, offline, _ := repo.snapshot()
	if writes != 0 || offline != 0 {
		t.Fatal("QR rotation changed an active passkey ceremony")
	}
	assertQRTestRunning(t, client)
	client.passkeyCeremony.Clear(client.userID)
	nextGate := newQRTestGate(t)
	client.qrBeforeAction = func(string) { nextGate.hold() }
	client.qrWait = qrTestWaitUntilCancelled
	client.handleQRCodes([]string{"new-batch-after-passkey"})
	awaitQRTest(t, nextGate.entered)
	next := qrTestRotation(client)
	if next == rotation {
		t.Fatal("finished rotation was reused")
	}
	nextGate.open()
	qrTestPair(client)
	awaitQRTest(t, next.finished)
	assertQRTestRunning(t, client)
}

func TestQRStaleGenerationCannotPublishStopOrPair(t *testing.T) {
	client, repo, sink := newQRTestClient(t)
	gate := newQRTestGate(t)
	client.qrBeforeAction = func(string) { gate.hold() }
	client.handleQRCodes([]string{"stale-disposable-qr"})
	awaitQRTest(t, gate.entered)
	rotation := qrTestRotation(client)
	client.stopSignal.Stop()
	next := runtimestate.NewStopSignal()
	next.Begin()
	client.killChannel.Set(client.userID, next)
	gate.open()
	awaitQRTest(t, rotation.finished)
	qrTestPair(client)
	_, _, writes, offline, pairs := repo.snapshot()
	if writes != 0 || offline != 0 || pairs != 0 || sink.count("QRCode") != 0 || sink.count("QRTimeout") != 0 {
		t.Fatal("old generation changed persisted or emitted QR state")
	}
	select {
	case <-next.Done():
		t.Fatal("old QR worker stopped its replacement generation")
	default:
	}
	next.Stop()
	next.Complete()
}
