// Copyright (c) 2021 - The Event Horizon authors.
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

package lock

import "sync"

// LocalLock is a Lock implemention using local locking only. Not suitable for
// use in distributed environments.
type LocalLock struct {
	locks map[string]struct{}
	mu    sync.Mutex
}

// NewLocalLock creates a new LocalLock.
func NewLocalLock() *LocalLock {
	return &LocalLock{
		locks: map[string]struct{}{},
	}
}

// Lock implements the Lock method of the Lock interface.
func (l *LocalLock) Lock(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.locks[id]; ok {
		return ErrLockExists
	}

	l.locks[id] = struct{}{}

	return nil
}

// Unlock implements the Unlock method of the Lock interface.
func (l *LocalLock) Unlock(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.locks[id]; !ok {
		return ErrNoLockExists
	}

	delete(l.locks, id)

	return nil
}
