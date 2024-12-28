package threadsafe

import (
	"sync"
	"time"
)

type ExpiryMap[K comparable, V any] struct {
	sync.RWMutex
	items map[K]item[V]
}

type item[V any] struct {
	value      V
	expiration time.Time
}

func NewExpiryMap[K comparable, V any]() *ExpiryMap[K, V] {
	return &ExpiryMap[K, V]{
		items: make(map[K]item[V]),
	}
}

func (m *ExpiryMap[K, V]) Set(key K, value V, ttl time.Duration) {
	m.Lock()
	defer m.Unlock()

	m.items[key] = item[V]{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
}

func (m *ExpiryMap[K, V]) Get(key K) (V, bool) {
	m.RLock()
	item, ok := m.items[key]
	m.RUnlock()

	var zero V
	if !ok {
		return zero, false
	}

	if time.Now().After(item.expiration) {
		m.Lock()
		delete(m.items, key)
		m.Unlock()
		return zero, false
	}

	return item.value, true
}

func (m *ExpiryMap[K, V]) Has(key K) bool {
	m.RLock()
	_, ok := m.items[key]
	m.RUnlock()

	if !ok {
		return false
	}

	if time.Now().After(m.items[key].expiration) {
		m.Lock()
		delete(m.items, key)
		m.Unlock()
		return false
	}

	return true
}

func (m *ExpiryMap[K, V]) Clear() {
	m.Lock()
	clear(m.items)
	m.Unlock()
}
