// SPDX-License-Identifier: Apache-2.0

package udp

import (
	"reflect"
	"testing"
)

func TestMessageIdentifier(t *testing.T) {
	msg := NewBroadcastMessage("127.0.0.1")

	b, err := msg.Bytes()
	if err != nil {
		t.Fatal("failed to turn CARISMA message into byte array")
	}

	isCARISMAMessage, err := IsCARISMAMessage(b)
	if !isCARISMAMessage || err != nil {
		t.Fatal("failed to identify CARISMA message as such")
	}
}

func TestMessageTypeIdentifier(t *testing.T) {
	msg := NewBroadcastMessage("127.0.0.1")

	msgType, err := GetMessageType(msg)
	if msgType != Broadcast || err != nil {
		t.Fatal("failed to identify CARISMA message type correctly")
	}
}

func TestMessageDecoding(t *testing.T) {
	msg := NewBroadcastMessage("127.0.0.1")

	b, err := msg.Bytes()
	if err != nil {
		t.Fatal("error occurred while encoding message")

		return
	}

	msgDecoded, err := DecodeBroadcastMessage(b)
	if err != nil {
		t.Fatal("error occurred while decoding message")

		return
	}

	if !reflect.DeepEqual(msg, msgDecoded) {
		t.Fatal("the two ip addresses should be the same")
	}
}
