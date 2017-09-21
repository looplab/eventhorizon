package configure

import (
	"context"
	"reflect"
	"testing"

	"github.com/kr/pretty"
	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/aggregatestore/events"
)

type TestAggregate struct {
	*events.AggregateBase
}

func (a *TestAggregate) HandleCommand(ctx context.Context, cmd eh.Command) error {
	return nil
}

func NewTestAggregate(id eh.UUID) eh.Aggregate {
	return &TestAggregate{
		AggregateBase: events.NewAggregateBase("Test", id),
	}
}

func TestAggregateBuilder(t *testing.T) {
	cases := map[string]struct {
		fact      func(eh.UUID) eh.Aggregate
		aggType   eh.AggregateType
		commTypes []eh.CommandType
	}{
		"A1 with 2 Commands": {
			NewTestAggregate,
			"A1",
			[]eh.CommandType{"Test2", "Test3"},
		},
		"A2 with 3 Commands commands": {
			NewTestAggregate,
			"A2",
			[]eh.CommandType{"Test2", "Test3", "Test4"},
		},
	}

	for name, tc := range cases {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// build it.
			b := NewAggregateConfig().
				SetFactory(tc.fact).
				AddAggregateType(tc.aggType)

			for _, t := range tc.commTypes {
				b = b.AddCommandType(t)
			}

			cfg, ok := b.(*config)
			if !ok {
				t.Errorf("test case '%s': Wrong builder type", name)
				t.Log("exp:", "*config")
				t.Log("got:", cfg)
			}

			sf1 := reflect.ValueOf(cfg.factory)
			sf2 := reflect.ValueOf(tc.fact)
			if sf1.Pointer() != sf2.Pointer() {
				t.Errorf("test case '%s': Wrong factory", name)
			}

			if cfg.aggregateType != tc.aggType {
				t.Errorf("test case '%s': Wrong aggregate type", name)
				t.Log("exp:", tc.aggType)
				t.Log("got:", cfg.aggregateType)
			}

			if !reflect.DeepEqual(cfg.commandTypes, tc.commTypes) {
				t.Errorf("test case '%s': CommandTypes are wrong", name)
				t.Log("exp:\n", pretty.Sprint(tc.commTypes))
				t.Log("got:\n", pretty.Sprint(cfg.commandTypes))
			}
		})
	}
}
