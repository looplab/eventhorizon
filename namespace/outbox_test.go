package namespace

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	eh "github.com/2908755265/eventhorizon"
	"github.com/2908755265/eventhorizon/mocks"
	"github.com/2908755265/eventhorizon/outbox"
	"github.com/2908755265/eventhorizon/outbox/memory"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func TestOutbox(t *testing.T) {
	usedNamespaces := map[string]struct{}{}

	// Shorter sweeps for testing
	memory.PeriodicSweepInterval = 2 * time.Second
	memory.PeriodicSweepAge = 2 * time.Second

	var outboxCreated sync.WaitGroup

	outboxCreated.Add(2)

	o := NewOutbox(func(ns string) (eh.Outbox, error) {
		usedNamespaces[ns] = struct{}{}
		o, err := memory.NewOutbox()
		if err != nil {
			return nil, err
		}

		outboxCreated.Done()

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

	if err := o.PreRegisterNamespace(DefaultNamespace); err != nil {
		t.Error("there should be no error:", err)
	}

	ns := "other"
	if err := o.PreRegisterNamespace(ns); err != nil {
		t.Error("there should be no error:", err)
	}

	// Check that both outboxes have been created.
	outboxCreated.Wait()

	if _, ok := usedNamespaces[DefaultNamespace]; !ok {
		t.Error("the default namespace should have been used")
	}

	if _, ok := usedNamespaces[ns]; !ok {
		t.Error("the other namespace should have been used")
	}

	t.Log("testing default namespace")
	outbox.AcceptanceTest(t, o, context.Background(), DefaultNamespace)

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

	if err := o.Close(); err != nil {
		t.Error("there should be no error:", err)
	}
}
