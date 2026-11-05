package whatsmeow_service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/evolution-foundation/evolution-go/pkg/config"
	instance_model "github.com/evolution-foundation/evolution-go/pkg/instance/model"
	instance_repository "github.com/evolution-foundation/evolution-go/pkg/instance/repository"
	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
	"github.com/evolution-foundation/evolution-go/pkg/runtimestate"
)

type startContextRepository struct {
	instance_repository.InstanceRepository
	load func(context.Context, string) (*instance_model.Instance, error)
}

func (r startContextRepository) GetInstanceByIDContext(ctx context.Context, id string) (*instance_model.Instance, error) {
	return r.load(ctx, id)
}

func TestQueryStartupCancellationDoesNotWaitForLifecycleOwner(t *testing.T) {
	signals := runtimestate.New[string, *runtimestate.StopSignal]()
	w := whatsmeowService{killChannel: signals}
	unlock := signals.LockKey("one")
	defer unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := w.StartInstanceContext(ctx, "one"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lock wait cause = %v", err)
	}
	if signals.Get("one") != nil {
		t.Fatal("canceled query created a generation")
	}
}

func TestQueryStartupReusesActiveAndPendingGenerations(t *testing.T) {
	signals := runtimestate.New[string, *runtimestate.StopSignal]()
	w := whatsmeowService{killChannel: signals}
	for _, started := range []bool{false, true} {
		signal := runtimestate.NewStopSignal()
		if started {
			signal.Begin()
		}
		signals.Set("one", signal)
		// No repository/client fields are installed: reuse must not do external
		// work or stop the generation while another query is still starting it.
		if err := w.StartInstanceContext(context.Background(), "one"); err != nil || signals.Get("one") != signal {
			t.Fatalf("reuse failed: %v", err)
		}
		select {
		case <-signal.Done():
			t.Fatal("avatar startup rotated an existing generation")
		default:
		}
		signal.Stop()
		signal.Complete()
	}
}

func TestQueryStartupSharesDatabaseDeadline(t *testing.T) {
	signals := runtimestate.New[string, *runtimestate.StopSignal]()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	deadline, _ := ctx.Deadline()
	w := whatsmeowService{killChannel: signals, instanceRepository: startContextRepository{
		load: func(got context.Context, id string) (*instance_model.Instance, error) {
			gotDeadline, ok := got.Deadline()
			if !ok || !gotDeadline.Equal(deadline) || id != "one" {
				t.Fatal("startup reset the request deadline or instance")
			}
			<-got.Done()
			return nil, got.Err()
		},
	}}
	if err := w.StartInstanceContext(ctx, "one"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("database wait cause = %v", err)
	}
	if signals.Get("one") != nil {
		t.Fatal("database cancellation still launched a generation")
	}
}

func TestCanceledQueryStartupDoesNoExternalWork(t *testing.T) {
	var w whatsmeowService
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := w.StartInstanceContext(ctx, "one"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cause = %v", err)
	}
}

func TestInvalidStartupSettingsDoNotLeavePendingGeneration(t *testing.T) {
	signals := runtimestate.New[string, *runtimestate.StopSignal]()
	w := whatsmeowService{
		killChannel:   signals,
		config:        &config.Config{},
		loggerWrapper: logger_wrapper.NewLoggerManager(&config.Config{LogDirectory: t.TempDir()}),
		instanceRepository: startContextRepository{load: func(context.Context, string) (*instance_model.Instance, error) {
			return &instance_model.Instance{Id: "one", Proxy: "{"}, nil
		}},
	}
	t.Cleanup(func() { w.loggerWrapper.GetLogger("one").Close() })
	if err := w.StartInstanceContext(context.Background(), "one"); err == nil {
		t.Fatal("invalid proxy JSON was accepted")
	}
	if signals.Get("one") != nil {
		t.Fatal("invalid settings left a pending generation which later queries would reuse")
	}
}
