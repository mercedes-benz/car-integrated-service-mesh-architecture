// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/config"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/logging"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/udp"
	"net"
	"strconv"
	"time"
)

func handleDiscovery(ctx context.Context, cfg *config.Config) {
	broadcaster, err := udp.NewBroadcastService(ctx, cfg)
	if err != nil {
		logging.LogErr(err)

		return
	}
	defer func() {
		logging.LogErr(broadcaster.Close())
	}()

	if cfg.EnableCentralMode {
		go func() {
			discoveryMsg := udp.NewBroadcastMessage(net.JoinHostPort(cfg.NodeHostname, strconv.Itoa(cfg.GRPCPort)))
			err := broadcaster.RepeatedlyWriteMessage(
				ctx,
				strconv.Itoa(cfg.UDPPort),
				discoveryMsg,
				time.Duration(cfg.UDPDelay)*config.TimeUnit,
			)

			logging.LogErr(err)
		}()
	} else {
		// discover central node
		if cfg.CentralNodeHostname == "" {
			for {
				msg, _, err := broadcaster.ReadPacketWithTimeout(time.Duration(cfg.UDPTimeout) * config.TimeUnit)
				if err != nil {
					logging.LogErr(err)

					return
				}

				isCARISMAMessage, err := udp.IsCARISMAMessage(msg)
				if !isCARISMAMessage || err != nil {
					continue
				}

				decodedMessage, err := udp.DecodeBroadcastMessage(msg)
				if err != nil {
					logging.LogErr(err)

					continue
				}

				cfg.CentralNodeHostname = decodedMessage.Hostname

				break
			}
		}
	}
}
