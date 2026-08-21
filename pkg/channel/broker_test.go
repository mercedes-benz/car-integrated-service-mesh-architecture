// SPDX-License-Identifier: MIT
// Copyright (c) 2025 MBition GmbH

package channel

import (
	"sync"
	"testing"
	"time"
)

func TestBrokerWithMultipleReaders(t *testing.T) {
	broker := NewBroker[string]()

	go broker.Listen()
	defer broker.Stop()

	var wgReader, wgReads sync.WaitGroup

	// three reader
	wgReader.Add(3)

	// three readers that receive three messages makes nine reads
	wgReads.Add(9)

	r := func() {
		ch := broker.Read()

		wgReader.Done()
		for i := 0; i < 3; i++ {
			_ = <-ch
			wgReads.Done()
		}
	}

	for i := 0; i < 3; i++ {
		go r()
	}

	// wait until readers are present before writing
	wgReader.Wait()

	var wgWriter sync.WaitGroup
	wgWriter.Add(1)

	go func() {
		defer wgWriter.Done()

		for i := 0; i < 3; i++ {
			broker.Write("Dummy")

			time.Sleep(300 * time.Millisecond)
		}
	}()

	wgWriter.Wait()
	wgReads.Wait()
}

func TestBrokerWithMultipleWriters(t *testing.T) {
	broker := NewBroker[string]()

	go broker.Listen()
	defer broker.Stop()

	ready := make(chan struct{})
	var readerWg sync.WaitGroup
	readerWg.Add(1)

	go func() {
		defer readerWg.Done()
		ch := broker.Read()
		close(ready)
		for i := 0; i < 9; i++ {
			_ = <-ch
		}
	}()

	<-ready

	var writerWg sync.WaitGroup
	writerWg.Add(3)

	w := func() {
		defer writerWg.Done()

		for i := 0; i < 3; i++ {
			broker.Write("Dummy")

			time.Sleep(300 * time.Millisecond)
		}
	}

	for i := 0; i < 3; i++ {
		// every goroutine writes three messages
		// plus one writer
		go w()
	}

	writerWg.Wait()
	readerWg.Wait()
}

func TestBrokerWithMultipleReadersAndWriters(t *testing.T) {
	broker := NewBroker[string]()

	go broker.Listen()
	defer broker.Stop()

	var wgReader, wgReads sync.WaitGroup

	// three reader
	wgReader.Add(3)

	// three readers that receive three messages from three different writers makes 27 reads
	wgReads.Add(27)

	r := func() {
		ch := broker.Read()

		wgReader.Done()
		for i := 0; i < 9; i++ {
			_ = <-ch
			wgReads.Done()
		}
	}

	for i := 0; i < 3; i++ {
		go r()
	}

	// wait until readers are present before writing
	wgReader.Wait()

	var wgWriter sync.WaitGroup
	wgWriter.Add(3)

	w := func() {
		defer wgWriter.Done()

		for i := 0; i < 3; i++ {
			broker.Write("Dummy")

			time.Sleep(300 * time.Millisecond)
		}
	}

	for i := 0; i < 3; i++ {
		go w()
	}

	wgWriter.Wait()
	wgReads.Wait()
}

func TestBrokerListenOnce(t *testing.T) {
	broker := NewBroker[string]()

	go broker.Listen()
	go broker.Listen()

	ch := broker.Read()
	broker.Write("Test")

	msg := <-ch
	if msg != "Test" {
		t.Fatalf("expected 'Test', got '%s'", msg)
	}

	broker.Stop()
}

func TestBrokerStopBeforeListen(t *testing.T) {
	broker := NewBroker[string]()
	broker.Stop()

	ch := broker.Read()

	_, ok := <-ch
	if ok {
		t.Fatal("expected closed channel when Stop called before Listen")
	}

	broker.Write("Test")
}

func TestBrokerReadAfterStop(t *testing.T) {
	broker := NewBroker[string]()

	go broker.Listen()
	broker.Stop()

	ch := broker.Read()

	_, ok := <-ch
	if ok {
		t.Fatal("expected closed channel after Stop")
	}
}
