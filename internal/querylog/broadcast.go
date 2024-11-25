package querylog

import (
	"sync"
)

var (
	Broadcaster *QueryBroadcaster
)

type QueryBroadcaster struct {
	sync.RWMutex
	clients map[chan *QueryLog]bool
}

func (b *QueryBroadcaster) Subscribe() chan *QueryLog {
	b.Lock()
	defer b.Unlock()

	ch := make(chan *QueryLog, 10)
	b.clients[ch] = true
	return ch
}

func (b *QueryBroadcaster) Unsubscribe(ch chan *QueryLog) {
	b.Lock()
	defer b.Unlock()

	delete(b.clients, ch)
	close(ch)
}

func (b *QueryBroadcaster) broadcast(query *QueryLog) {
	b.RLock()
	defer b.RUnlock()

	for ch := range b.clients {
		select {
		case ch <- query:
		default:
			// Skip if the client is not consuming fast enough
		}
	}
}
