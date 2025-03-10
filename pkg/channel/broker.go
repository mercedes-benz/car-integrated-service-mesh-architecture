// SPDX-License-Identifier: Apache-2.0

package channel

// Broker implements a message relay that copies messages received from one source channel to multiple receiving channels.
type Broker[T any] struct {
	destinations chan chan T
	source       chan T
	quit         chan struct{}
}

// NewBroker creates a new instance of Broker.
func NewBroker[T any]() *Broker[T] {
	return &Broker[T]{
		destinations: make(chan chan T, 1),
		source:       make(chan T, 1),
		quit:         make(chan struct{}),
	}
}

// Listen blocks the caller lets the broker listen for messages.
func (b *Broker[T]) Listen() {
	listeners := map[chan T]struct{}{}
	for {
		select {
		case msgCh := <-b.destinations:
			listeners[msgCh] = struct{}{}
		case msg := <-b.source:
			for ch := range listeners {
				select {
				case ch <- msg:
				default:
				}
			}
		case <-b.quit:
			return
		}
	}
}

// Stop lets the broker end the listening for messages.
func (b *Broker[T]) Stop() {
	close(b.quit)
}

func (b *Broker[T]) Read() chan T {
	msgCh := make(chan T, 5)
	b.destinations <- msgCh
	return msgCh
}

func (b *Broker[T]) Write(msg T) {
	b.source <- msg
}
