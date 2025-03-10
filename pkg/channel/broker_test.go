// SPDX-License-Identifier: Apache-2.0

package channel

import (
	"sync"
	"testing"
	"time"
)

func TestBrokerWithMultipleReaders(t *testing.T) {
	broker := NewBroker[string]()

	go broker.Listen()

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

	var wg sync.WaitGroup
	w := func() {
		defer wg.Done()

		for i := 0; i < 3; i++ {
			broker.Write("Dummy")

			time.Sleep(300 * time.Millisecond)
		}
	}

	for i := 0; i < 3; i++ {
		// every goroutine writes three messages
		// plus one writer
		wg.Add(4)

		go w()
	}

	go func() {
		ch := broker.Read()
		for i := 0; i < 10; i++ {
			_ = <-ch

			wg.Done()
		}
	}()

	wg.Wait()
}

func TestBrokerWithMultipleReadersAndWriters(t *testing.T) {
	broker := NewBroker[string]()

	go broker.Listen()

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
