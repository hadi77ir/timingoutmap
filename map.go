// Copyright (C) 2022 Mohammad Hadi Hosseinpour
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package timingoutmap

import (
	"errors"
	"sync"
	"time"
)

// Hashable definition from https://github.com/cornelk/hashmap/blob/main/defines.go
type Hashable interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64 | ~string
}

type Item[V any] struct {
	value   V
	lastUse time.Time
}

type TimeoutMap[K Hashable, V any] struct {
	lock    *sync.Mutex
	store   map[K]*Item[V]
	timeout time.Duration
	renew   bool
}

var ErrNotFound = errors.New("not found")
var ErrExpired = errors.New("expired")

func New[K Hashable, V any](timeout time.Duration, renew bool) *TimeoutMap[K, V] {
	return &TimeoutMap[K, V]{
		lock:    &sync.Mutex{},
		timeout: timeout,
		renew:   renew,
		store:   make(map[K]*Item[V]),
	}
}

func NewWithMutex[K Hashable, V any](lock *sync.Mutex, timeout time.Duration, renew bool) *TimeoutMap[K, V] {
	return &TimeoutMap[K, V]{
		lock:    lock,
		timeout: timeout,
		renew:   renew,
		store:   make(map[K]*Item[V]),
	}
}
func (m *TimeoutMap[K, V]) Get(key K) (V, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	var none V
	item, found := m.store[key]
	if !found || item == nil {
		return none, ErrNotFound
	}

	if time.Since(item.lastUse) < m.timeout {
		if m.renew {
			item.lastUse = time.Now()
		}
		return item.value, nil
	}
	return item.value, ErrExpired
}
func (m *TimeoutMap[K, V]) GetOrNew(key K, fallback V) V {
	m.lock.Lock()
	defer m.lock.Unlock()
	item, found := m.store[key]
	if found && item != nil {
		// check expiry
		if time.Since(item.lastUse) < m.timeout {
			if m.renew {
				item.lastUse = time.Now()
			}
			return item.value
		}
	}
	// expired or not found
	item = &Item[V]{value: fallback, lastUse: time.Now()}
	m.store[key] = item
	return fallback
}
func (m *TimeoutMap[K, V]) SetOrReplace(key K, value V) (V, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	item := &Item[V]{value: value, lastUse: time.Now()}
	m.store[key] = item
	return value, nil
}

func (m *TimeoutMap[K, V]) Clear() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.store = make(map[K]*Item[V])
}

func (m *TimeoutMap[K, V]) SetTimeout(duration time.Duration) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.timeout = duration
}

func (m *TimeoutMap[K, V]) CleanDead() {
	m.lock.Lock()
	defer m.lock.Unlock()
	for key, item := range m.store {
		if time.Since(item.lastUse) > m.timeout {
			delete(m.store, key)
		}
	}
}
