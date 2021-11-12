package namespace

import (
	"context"
	"math/rand"
	"testing"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/mocks"
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

	handlerAddedBefore := mocks.NewEventHandler("handler_before")
	if err := o.AddHandler(context.Background(), eh.MatchEvents{mocks.EventType}, handlerAddedBefore); err != nil {
		t.Fatal("there should be no error:", err)
	}

	t.Log("testing default namespace")
	outbox.AcceptanceTest(t, o, context.Background(), DefaultNamespace)

	ns := "other"
	ctx := NewContext(context.Background(), ns)

	t.Log("testing other namespace")
	outbox.AcceptanceTest(t, o, ctx, ns)

	if !handlerAddedBefore.Wait(time.Second) {
		t.Error("did not receive event in time")
	}

	handlerAddedBefore.Lock()

	if len(handlerAddedBefore.Events) != 6 {
		t.Errorf("there should be 6 event: %d", len(handlerAddedBefore.Events))
	}

	handlerAddedBefore.Unlock()

	if _, ok := usedNamespaces[DefaultNamespace]; !ok {
		t.Error("the default namespace should have been used")
	}

	if _, ok := usedNamespaces[ns]; !ok {
		t.Error("the other namespace should have been used")
	}

	if err := o.Close(); err != nil {
		t.Error("there should be no error:", err)
	}
}
