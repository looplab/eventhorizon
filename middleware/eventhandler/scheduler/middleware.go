// Copyright (c) 2020 - The Event Horizon authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scheduler

import (
	"context"
	"time"

	"github.com/gorhill/cronexpr"

	eh "github.com/looplab/eventhorizon"
)

// NewMiddleware creates a new scheduler middleware that runs until the context is canceled.
func NewMiddleware(ctx context.Context) (eh.EventHandlerMiddleware, *Scheduler) {
	s := &Scheduler{ctx: ctx}
	return eh.EventHandlerMiddleware(func(h eh.EventHandler) eh.EventHandler {
		m := &eventHandler{h, s.newChannel()}
		go m.run(ctx)
		return m
	}), s
}

// Scheduler is a event scheduler that periodically inserts events into the event stream.
// It uses the cron syntax from https://github.com/gorhill/cronexpr.
type Scheduler struct {
	ctx      context.Context
	eventChs []chan data
}

func (s *Scheduler) newChannel() <-chan data {
	ch := make(chan data)
	s.eventChs = append(s.eventChs, ch)
	return ch
}

// ScheduleEvent schedules an event to be sent on regular intervals, using
// a line in the crontab format to setup the timing. The eventFunc should create
// the event to send given the triggered time as input. Cancelling the context
// will stop the triggering of more events.
func (s *Scheduler) ScheduleEvent(ctx context.Context, cronLine string, eventFunc func(time.Time) eh.Event) error {
	if err := s.ctx.Err(); err != nil {
		return err
	}

	expr, err := cronexpr.Parse(cronLine)
	if err != nil {
		return err
	}

	// Schedule until either this schedule is canceled or the full scheduler is stopped.
	go func() {
		for {
			nextTime := expr.Next(time.Now())
			select {
			case <-time.After(nextTime.Sub(time.Now())):
				for _, eventCh := range s.eventChs {
					eventCh <- data{ctx, eventFunc(nextTime)}
				}
			case <-ctx.Done():
				// Stop when this individual scheduling is canceled.
				return
			case <-s.ctx.Done():
				// Stop when the full scheduler is stopped.
				return
			}
		}
	}()

	return nil
}

type data struct {
	ctx   context.Context
	event eh.Event
}

type eventHandler struct {
	eh.EventHandler
	eventsCh <-chan data
}

func (h *eventHandler) run(ctx context.Context) error {
	for {
		select {
		case data := <-h.eventsCh:
			if err := h.HandleEvent(data.ctx, data.event); err != nil {
				return err
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil && err != context.Canceled {
				return err
			}
			return nil
		}
	}
}
