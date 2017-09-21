package configure

import (
	"reflect"
	"testing"

	"github.com/kr/pretty"

	"github.com/looplab/eventhorizon/aggregatestore/events"
	"github.com/looplab/eventhorizon/commandhandler/bus"
	eventbus "github.com/looplab/eventhorizon/eventbus/local"
	eventstore "github.com/looplab/eventhorizon/eventstore/memory"
	eventpublisher "github.com/looplab/eventhorizon/publisher/local"
)

func TestContainer(t *testing.T) {
	cases := map[string]struct {
		configs []AggregateConfig
	}{
		"test1": {
			configs: []AggregateConfig{
				NewAggregateConfig().SetFactory(NewTestAggregate).AddAggregateType("A1").AddCommandType("C1", "C2"),
			},
		},
	}

	for name, tc := range cases {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			eventStore := eventstore.NewEventStore()
			eventPublisher := eventpublisher.NewEventPublisher()
			eventBus := eventbus.NewEventBus()
			eventBus.SetPublisher(eventPublisher)

			as, err := events.NewAggregateStore(eventStore, eventBus)
			if err != nil {
				t.Error("Could not create aggregateStore")
				return
			}
			b := bus.NewCommandHandler()

			ic := NewContainer(as, b)
			if ic == nil {
				t.Errorf("test case '%s': NewContainer", name)
				t.Log("exp:", "{}")
				t.Log("got:", "nil")
			}

			for _, cfg := range tc.configs {
				ic = ic.RegisterAggregates(cfg)
			}

			c, ok := ic.(*container)
			if !ok {
				t.Errorf("test case '%s': Wrong container type", name)
				t.Log("exp:", "*container")
				t.Log("got:", c)
			}

			if !reflect.DeepEqual(c.aggregateConfigs, tc.configs) {
				t.Errorf("test case '%s': AggregateConfigs are wrong", name)
				t.Log("exp:\n", pretty.Sprint(tc.configs))
				t.Log("got:\n", pretty.Sprint(c.aggregateConfigs))
			}
		})
	}
}
