// Copyright (c) 2017 - The Event Horizon authors.
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

package cron

import (
	"context"
	"time"

	"github.com/gorhill/cronexpr"

	eh "github.com/looplab/eventhorizon"
)

// EventHandler is a cron runner that inserts timed events into the event stream.
// It uses the cron syntax from https://github.com/gorhill/cronexpr.
type EventHandler struct {
	eh.EventHandler

	eventsCh chan data
	errCh    chan error
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(eventHandler eh.EventHandler) *EventHandler {
	h := &EventHandler{
		EventHandler: eventHandler,
		eventsCh:     make(chan data),
		errCh:        make(chan error, 1),
	}

	go h.run()

	return h
}

// ScheduleEvent schedules an event to be sent on regular intervals, using
// a line in the crontab format to setup the timing. The eventFunc should create
// the event to send given the triggered time as input. Cancelling the context
// will stop the triggering of more events.
func (h *EventHandler) ScheduleEvent(ctx context.Context, cronLine string, eventFunc func(time.Time) eh.Event) error {
	expr, err := cronexpr.Parse(cronLine)
	if err != nil {
		return err
	}

	go func() {
		for {
			nextTime := expr.Next(time.Now())
			select {
			case <-time.After(nextTime.Sub(time.Now())):
				h.eventsCh <- data{ctx, eventFunc(nextTime)}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Error returns the error channel.
func (h *EventHandler) Error() <-chan error {
	return h.errCh
}

type data struct {
	ctx   context.Context
	event eh.Event
}

func (h *EventHandler) run() {
	for data := range h.eventsCh {
		if err := h.HandleEvent(data.ctx, data.event); err != nil {
			select {
			case h.errCh <- err:
			default:
			}
		}
	}
}
