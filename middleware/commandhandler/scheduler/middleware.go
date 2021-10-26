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

package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	eh "github.com/looplab/eventhorizon"
	"github.com/looplab/eventhorizon/uuid"
)

// ErrCanceled is when a scheduled command has been canceled.
var ErrCanceled = errors.New("canceled")

// Command is a scheduled command with an execution time.
type Command interface {
	eh.Command

	// ExecuteAt returns the time when the command will execute.
	ExecuteAt() time.Time
}

// CommandWithExecuteTime returns a wrapped command with a execution time set.
func CommandWithExecuteTime(cmd eh.Command, t time.Time) Command {
	return &command{Command: cmd, t: t}
}

// private implementation to wrap ordinary commands and add a execution time.
type command struct {
	eh.Command
	t time.Time
}

// ExecuteAt implements the ExecuteAt method of the Command interface.
func (c *command) ExecuteAt() time.Time {
	return c.t
}

// NewMiddleware returns a new command handler middleware and a scheduler helper.
func NewMiddleware(repo eh.ReadWriteRepo, codec eh.CommandCodec) (eh.CommandHandlerMiddleware, *Scheduler) {
	s := &Scheduler{
		repo:             repo,
		cmdCh:            make(chan *scheduledCommand, 100),
		cancelScheduling: map[uuid.UUID]chan struct{}{},
		errCh:            make(chan error, 100),
		codec:            codec,
	}

	return eh.CommandHandlerMiddleware(func(h eh.CommandHandler) eh.CommandHandler {
		s.setHandler(h)

		return eh.CommandHandlerFunc(func(ctx context.Context, cmd eh.Command) error {
			// Delayed command execution if there is time set.
			if c, ok := cmd.(Command); ok && !c.ExecuteAt().IsZero() {
				// Use the wrapped command when created by the helper func.
				innerCmd, ok := c.(*command)
				if ok {
					cmd = innerCmd.Command
				}

				// Ignore the persisted command ID in this case.
				_, err := s.ScheduleCommand(ctx, cmd, c.ExecuteAt())

				return err
			}

			// Immediate command execution.
			return h.HandleCommand(ctx, cmd)
		})
	}), s
}

// PersistedCommand is a persisted command.
type PersistedCommand struct {
	ID         uuid.UUID       `json:"_"                 bson:"_id"`
	IDStr      string          `json:"id"                bson:"_"`
	RawCommand []byte          `json:"command"           bson:"command"`
	ExecuteAt  time.Time       `json:"timestamp"         bson:"timestamp"`
	Command    eh.Command      `json:"-"                 bson:"-"`
	Context    context.Context `json:"-"                 bson:"-"`
}

// EntityID implements the EntityID method of the eventhorizon.Entity interface.
func (c *PersistedCommand) EntityID() uuid.UUID {
	return c.ID
}

// Scheduler is a scheduled of commands.
type Scheduler struct {
	h                  eh.CommandHandler
	hMu                sync.Mutex
	repo               eh.ReadWriteRepo
	cmdCh              chan *scheduledCommand
	cancelScheduling   map[uuid.UUID]chan struct{}
	cancelSchedulingMu sync.Mutex
	errCh              chan error
	cctx               context.Context
	cancel             context.CancelFunc
	done               chan struct{}
	codec              eh.CommandCodec
}

func (s *Scheduler) setHandler(h eh.CommandHandler) {
	s.hMu.Lock()
	defer s.hMu.Unlock()

	if s.h != nil {
		panic("eventhorizon: handler already set for outbox")
	}

	s.h = h
}

// Start starts the scheduler by first loading all persisted commands.
func (s *Scheduler) Start() error {
	if s.h == nil {
		return fmt.Errorf("command handler not set")
	}

	if err := s.loadCommands(); err != nil {
		return fmt.Errorf("could not load commands: %w", err)
	}

	s.cctx, s.cancel = context.WithCancel(context.Background())
	s.done = make(chan struct{})

	go s.run()

	return nil
}

// Stop stops all scheduled commands.
func (s *Scheduler) Stop() error {
	s.cancel()

	<-s.done

	return nil
}

// Errors returns an error channel that will receive errors from handling of
// scheduled commands.
func (s *Scheduler) Errors() <-chan error {
	return s.errCh
}

type scheduledCommand struct {
	id        uuid.UUID
	ctx       context.Context
	cmd       eh.Command
	executeAt time.Time
}

// ScheduleCommand schedules a command to be executed at `executeAt`. It is persisted
// to the repo.
func (s *Scheduler) ScheduleCommand(ctx context.Context, cmd eh.Command, executeAt time.Time) (uuid.UUID, error) {
	b, err := s.codec.MarshalCommand(ctx, cmd)
	if err != nil {
		return uuid.Nil, &Error{
			Err:     fmt.Errorf("could not marshal command: %w", err),
			Ctx:     ctx,
			Command: cmd,
		}
	}

	// Use the command ID as persisted ID if available.
	var id uuid.UUID
	if cmd, ok := cmd.(eh.CommandIDer); ok {
		id = cmd.CommandID()
	} else {
		id = uuid.New()
	}

	pc := &PersistedCommand{
		ID:         id,
		IDStr:      id.String(),
		RawCommand: b,
		ExecuteAt:  executeAt,
	}

	if err := s.repo.Save(context.Background(), pc); err != nil {
		return uuid.Nil, &Error{
			Err:     fmt.Errorf("could not persist command: %w", err),
			Ctx:     ctx,
			Command: cmd,
		}
	}

	select {
	case s.cmdCh <- &scheduledCommand{id, ctx, cmd, executeAt}:
	default:
		return uuid.Nil, &Error{
			Err:     fmt.Errorf("command queue full"),
			Ctx:     ctx,
			Command: cmd,
		}
	}

	return pc.ID, nil
}

// Commands returns all scheduled commands.
func (s *Scheduler) Commands(ctx context.Context) ([]*PersistedCommand, error) {
	entities, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not load scheduled commands: %w", err)
	}

	commands := make([]*PersistedCommand, len(entities))

	for i, entity := range entities {
		c, ok := entity.(*PersistedCommand)
		if !ok {
			return nil, fmt.Errorf("command is not schedulable: %T", entity)
		}

		if c.Command, c.Context, err = s.codec.UnmarshalCommand(ctx, c.RawCommand); err != nil {
			return nil, fmt.Errorf("could not unmarshal command: %w", err)
		}

		c.RawCommand = nil

		if c.IDStr != "" {
			id, err := uuid.Parse(c.IDStr)
			if err != nil {
				return nil, fmt.Errorf("could not parse command ID: %w", err)
			}

			c.ID = id
		}

		commands[i] = c
	}

	return commands, nil
}

// CancelCommand cancels a scheduled command.
func (s *Scheduler) CancelCommand(ctx context.Context, id uuid.UUID) error {
	s.cancelSchedulingMu.Lock()
	defer s.cancelSchedulingMu.Unlock()

	cancel, ok := s.cancelScheduling[id]
	if !ok {
		return fmt.Errorf("command %s not scheduled", id)
	}

	close(cancel)

	return nil
}

func (s *Scheduler) loadCommands() error {
	commands, err := s.Commands(context.Background())
	if err != nil {
		return fmt.Errorf("could not load scheduled commands: %w", err)
	}

	for _, pc := range commands {
		sc := &scheduledCommand{
			id:        pc.ID,
			ctx:       pc.Context,
			cmd:       pc.Command,
			executeAt: pc.ExecuteAt,
		}

		select {
		case s.cmdCh <- sc:
		default:
			return fmt.Errorf("could not schedule command: %w", err)
		}
	}

	return nil
}

func (s *Scheduler) run() {
	var wg sync.WaitGroup

loop:
	for {
		select {
		case <-s.cctx.Done():
			break loop
		case sc := <-s.cmdCh:
			wg.Add(1)

			s.cancelSchedulingMu.Lock()
			cancel := make(chan struct{})
			s.cancelScheduling[sc.id] = cancel
			s.cancelSchedulingMu.Unlock()

			go func(cancel chan struct{}) {
				defer wg.Done()

				t := time.NewTimer(time.Until(sc.executeAt))
				defer t.Stop()

				select {
				case <-s.cctx.Done():
					// Stop without removing persisted cmd.
				case <-t.C:
					if err := s.h.HandleCommand(sc.ctx, sc.cmd); err != nil {
						// Always try to deliver errors.
						s.errCh <- &Error{
							Err:     err,
							Ctx:     sc.ctx,
							Command: sc.cmd,
						}
					}

					if err := s.repo.Remove(context.Background(), sc.id); err != nil {
						s.errCh <- &Error{
							Err:     fmt.Errorf("could not remove persisted command: %w", err),
							Ctx:     sc.ctx,
							Command: sc.cmd,
						}
					}
				case <-cancel:
					if err := s.repo.Remove(context.Background(), sc.id); err != nil {
						s.errCh <- &Error{
							Err:     fmt.Errorf("could not remove persisted command: %w", err),
							Ctx:     sc.ctx,
							Command: sc.cmd,
						}
					}

					s.errCh <- &Error{
						Err:     ErrCanceled,
						Ctx:     sc.ctx,
						Command: sc.cmd,
					}
				}
			}(cancel)
		}
	}

	wg.Wait()

	close(s.done)
}

// Error is an async error containing the error and the command.
type Error struct {
	// Err is the error that happened when handling the command.
	Err error
	// Ctx is the context used when the error happened.
	Ctx context.Context
	// Command is the command handeled when the error happened.
	Command eh.Command
}

// Error implements the Error method of the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("%s (%s): %s", e.Command.CommandType(), e.Command.AggregateID(), e.Err.Error())
}

// Unwrap implements the errors.Unwrap method.
func (e *Error) Unwrap() error {
	return e.Err
}

// Cause implements the github.com/pkg/errors Unwrap method.
func (e *Error) Cause() error {
	return e.Unwrap()
}
