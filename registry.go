// Copyright (c) 2014 - The Event Horizon authors.
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

package eventhorizon

import (
	"fmt"
	"sync"
)

type typeRegistry[K comparable, F any] struct {
	mu    sync.RWMutex
	items map[K]F
	name  string
}

func newTypeRegistry[K comparable, F any](name string) *typeRegistry[K, F] {
	return &typeRegistry[K, F]{
		items: make(map[K]F),
		name:  name,
	}
}

func (r *typeRegistry[K, F]) register(key K, factory F) {
	var zero K
	if key == zero {
		panic(fmt.Sprintf("eventhorizon: attempt to register empty %s type", r.name))
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[key]; ok {
		panic(fmt.Sprintf("eventhorizon: registering duplicate types for %q", any(key)))
	}

	r.items[key] = factory
}

func (r *typeRegistry[K, F]) unregister(key K) {
	var zero K
	if key == zero {
		panic(fmt.Sprintf("eventhorizon: attempt to unregister empty %s type", r.name))
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[key]; !ok {
		panic(fmt.Sprintf("eventhorizon: unregister of non-registered type %q", any(key)))
	}

	delete(r.items, key)
}

func (r *typeRegistry[K, F]) get(key K) (F, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, ok := r.items[key]

	return f, ok
}

func (r *typeRegistry[K, F]) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items = make(map[K]F)
}

func (r *typeRegistry[K, F]) all() map[K]F {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[K]F, len(r.items))
	for k, v := range r.items {
		result[k] = v
	}

	return result
}
