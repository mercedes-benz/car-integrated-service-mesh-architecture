// SPDX-License-Identifier: MIT
// Copyright (c) 2025 MBition GmbH

package channel

import (
	"sync"
	"sync/atomic"
)

const (
	brokerNew uint32 = iota
	brokerRunning
	brokerStopped
)

type request[T any] struct {
	channel chan T
	done    chan struct{}
	once    sync.Once
}

func (r *request[T]) close() {
	r.once.Do(func() {
		close(r.channel)
	})
}

// Broker implements a message relay that copies messages received from one source channel to multiple receiving channels.
type Broker[T any] struct {
	destinations chan *request[T]
	source       chan T
	quit         chan struct{}
	done         chan struct{}
	state        atomic.Uint32
}

// NewBroker creates a new instance of Broker.
func NewBroker[T any]() *Broker[T] {
	return &Broker[T]{
		destinations: make(chan *request[T], 1),
		source:       make(chan T),
		quit:         make(chan struct{}),
		done:         make(chan struct{}),
	}
}

// Listen blocks the caller lets the broker listen for messages.
func (b *Broker[T]) Listen() {
	if !b.state.CompareAndSwap(brokerNew, brokerRunning) {
		return
	}

	listeners := make(map[*request[T]]struct{})

	defer func() {
		for request := range listeners {
			request.close()
		}

		close(b.done)
	}()

	for {
		select {
		case request := <-b.destinations:
			listeners[request] = struct{}{}
			close(request.done)

		case msg := <-b.source:
			for request := range listeners {
				select {
				case request.channel <- msg:
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
	if b.state.CompareAndSwap(brokerNew, brokerStopped) {
		close(b.quit)
		close(b.done)
		return
	}

	if b.state.CompareAndSwap(brokerRunning, brokerStopped) {
		close(b.quit)
	}
}

func (b *Broker[T]) Read() chan T {
	request := &request[T]{
		channel: make(chan T, 5),
		done:    make(chan struct{}),
	}

	if b.state.Load() == brokerStopped {
		request.close()
		return request.channel
	}

	select {
	case b.destinations <- request:
		select {
		case <-request.done:
			return request.channel
		case <-b.done:
			request.close()
			return request.channel
		}

	case <-b.done:
		request.close()
		return request.channel
	}
}

func (b *Broker[T]) Write(msg T) {
	if b.state.Load() != brokerRunning {
		return
	}

	select {
	case b.source <- msg:
	case <-b.done:
	}
}
