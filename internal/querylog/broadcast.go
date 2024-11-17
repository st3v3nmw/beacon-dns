package querylog

import (
	"sync"
)

var (
	Broadcaster *QueryBroadcaster
)

type QueryBroadcaster struct {
	clients map[chan QueryLog]bool
	lock    sync.RWMutex
}

func NewQueryBroadcaster() *QueryBroadcaster {
	return &QueryBroadcaster{
		clients: make(map[chan QueryLog]bool),
	}
}

func (b *QueryBroadcaster) Subscribe() chan QueryLog {
	b.lock.Lock()
	defer b.lock.Unlock()

	ch := make(chan QueryLog, 10)
	b.clients[ch] = true
	return ch
}

func (b *QueryBroadcaster) Unsubscribe(ch chan QueryLog) {
	b.lock.Lock()
	defer b.lock.Unlock()

	delete(b.clients, ch)
	close(ch)
}

func (b *QueryBroadcaster) Broadcast(query QueryLog) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	for ch := range b.clients {
		select {
		case ch <- query:
		default:
			// Skip if the client is not consuming fast enough
		}
	}
}
