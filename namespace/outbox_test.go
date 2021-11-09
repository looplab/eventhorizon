package namespace

import (
	"context"
	"math/rand"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/outbox"
	"github.com/looplab/eventhorizon/outbox/memory"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func TestOutbox(t *testing.T) {
	usedNamespaces := map[string]struct{}{}

	// Shorter sweeps for testing
	memory.PeriodicSweepInterval = 2 * time.Second
	memory.PeriodicSweepAge = 2 * time.Second

	o := NewOutbox(func(ns string) (eh.Outbox, error) {
		usedNamespaces[ns] = struct{}{}
		o, err := memory.NewOutbox()
		if err != nil {
			return nil, err
		}

		return o, nil
	})
	if o == nil {
		t.Fatal("there should be an outbox")
	}

	o.Start()

	t.Log("testing default namespace")
	outbox.AcceptanceTest(t, o, context.Background())

	ctx := NewContext(context.Background(), "other")

	t.Log("testing other namespace")
	outbox.AcceptanceTest(t, o, ctx)

	if _, ok := usedNamespaces["default"]; !ok {
		t.Error("the default namespace should have been used")
	}

	if _, ok := usedNamespaces["other"]; !ok {
		t.Error("the other namespace should have been used")
	}

	if err := o.Close(); err != nil {
		t.Error("there should be no error:", err)
	}
}
