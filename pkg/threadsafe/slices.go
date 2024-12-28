package threadsafe

import "sync"

type Slice[T any] struct {
	sync.RWMutex
	items []T
}

func (s *Slice[T]) Append(item T) {
	s.Lock()
	defer s.Unlock()
	s.items = append(s.items, item)
}

func (s *Slice[T]) Len() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.items)
}

func (s *Slice[T]) Iterator() <-chan T {
	ch := make(chan T)
	go func() {
		s.RLock()
		defer s.RUnlock()

		for _, item := range s.items {
			ch <- item
		}
		close(ch)
	}()
	return ch
}

func (s *Slice[T]) Clear() {
	s.Lock()
	defer s.Unlock()
	s.items = s.items[:0]
}
