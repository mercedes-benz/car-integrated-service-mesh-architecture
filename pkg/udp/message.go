// SPDX-License-Identifier: Apache-2.0

package udp

import (
	"bytes"
	"encoding/gob"
)

// MessageType encodes the type of the CARISMA message.
type MessageType byte

const (
	magicString = "$CARISMA$"
)

const (
	// Invalid message.
	Invalid MessageType = iota
	// Broadcast message.
	Broadcast // = 1
)

// Message interface is the common message interface implemented by all message types.
type Message interface {
	// Bytes encodes a BroadcastMessage as byte sequence.
	Bytes() ([]byte, error)
}

// IsCARISMAMessage checks whether the provided byte sequence is a CARISMA message.
func IsCARISMAMessage(b []byte) (bool, error) {
	if len(b) < len(magicString) {
		return false, nil
	}

	return bytes.Equal(b[:len(magicString)], []byte(magicString)), nil
}

// GetMessageType determines the concrete CARISMA message type of the provided message.
func GetMessageType(msg Message) (MessageType, error) {
	b, err := msg.Bytes()
	if err != nil {
		return Invalid, err
	}

	switch b[len(magicString)] {
	case byte(Broadcast):
		return Broadcast, nil
	default:
		return Invalid, nil
	}
}

// BroadcastMessage encodes the hostname of the sender.
type BroadcastMessage struct {
	Hostname string
}

// NewBroadcastMessage creates a new broadcast message.
func NewBroadcastMessage(hostname string) *BroadcastMessage {
	return &BroadcastMessage{hostname}
}

// DecodeBroadcastMessage decodes the provided byte sequence into an instance of BroadcastMessage.
func DecodeBroadcastMessage(msg []byte) (*BroadcastMessage, error) {
	buf := bytes.NewBuffer(msg[len(magicString)+1:])
	dec := gob.NewDecoder(buf)

	var g BroadcastMessage
	err := dec.Decode(&g)

	return &g, err
}

func (g BroadcastMessage) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	buf.Write([]byte(magicString))
	buf.Write([]byte{byte(Broadcast)})

	enc := gob.NewEncoder(&buf)
	err := enc.Encode(g)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
