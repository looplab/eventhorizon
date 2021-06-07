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

import "fmt"

var (
	// ErrLockExists is returned from Lock() when the lock is already taken.
	ErrLockExists = fmt.Errorf("lock exists")
	// ErrNoLockExists is returned from Unlock() when the lock does not exist.
	ErrNoLockExists = fmt.Errorf("no lock exists")
)

// Lock is a locker of IDs.
type Lock interface {
	// Lock sets a lock for the ID. Returns ErrLockExists if the lock is already
	// taken or another error if it was not possible to get the lock.
	Lock(id string) error
	// Unlock releases the lock for the ID. Returns ErrNoLockExists if there is
	// no lock for the ID or another error if it was not possible to unlock.
	Unlock(id string) error
}
