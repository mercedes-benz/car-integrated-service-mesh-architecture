// SPDX-License-Identifier: Apache-2.0

package udp

import (
	"context"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/config"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/logging"
	"golang.org/x/sys/unix"
	"net"
	"strconv"
	"syscall"
	"time"
)

const (
	defaultUDPBroadcastAddr = "255.255.255.255"
)

// BroadcastService wraps a UDP connection.
type BroadcastService struct {
	p *net.PacketConn
}

// NewBroadcastService creates a new instance of BroadcastService.
func NewBroadcastService(ctx context.Context, cfg *config.Config) (*BroadcastService, error) {
	lc := net.ListenConfig{}

	// In debug mode we reuse address and port to ease up debugging
	if cfg.EnableDebugMode {
		lc.Control = func(network, address string, c syscall.RawConn) error {
			if err := c.Control(func(fd uintptr) {
				err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
				if err != nil {
					return
				}

				err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
				if err != nil {
					return
				}
			}); err != nil {
				return err
			}

			return nil
		}
	}

	packetConn, err := lc.ListenPacket(
		ctx,
		"udp4",
		net.JoinHostPort("", strconv.Itoa(cfg.UDPPort)),
	)
	if err != nil {
		return nil, err
	}

	return &BroadcastService{&packetConn}, nil
}

// Close closes the internal network connection.
func (b *BroadcastService) Close() error {
	return (*b.p).Close()
}

// RepeatedlyWriteMessage blocks the caller and repeatedly writes the supplied message w.r.t. the provided delay.
func (b *BroadcastService) RepeatedlyWriteMessage(ctx context.Context, port string, msg Message, delay time.Duration) error {
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logging.LogDbg("Stopped sending UDP broadcast packets")

			return nil
		case <-ticker.C:
			addr, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(defaultUDPBroadcastAddr, port))
			if err != nil {
				panic(err)
			}

			data, err := msg.Bytes()
			if err != nil {
				return err
			}

			_, err = (*b.p).WriteTo(data, addr)
			if err != nil {
				return err
			}

			logging.LogDbg("Sent UDP broadcast packet")
		}
	}
}

// ReadPacketWithTimeout tries to read a packet within the specified time.
func (b *BroadcastService) ReadPacketWithTimeout(timeout time.Duration) ([]byte, net.Addr, error) {
	buf := make([]byte, 1024)

	err := (*b.p).SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, nil, err
	}

	numBytes, addr, err := (*b.p).ReadFrom(buf[:cap(buf)])
	if err != nil {
		return nil, nil, err
	}

	buf = buf[:numBytes]

	logging.LogDbg("Received UDP broadcast packet")

	return buf, addr, nil
}
