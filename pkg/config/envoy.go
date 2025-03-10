// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	upstreams "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	"google.golang.org/protobuf/types/known/anypb"
)

// HTTP2ProtocolOptions creates the HTTP2 protocol options required to enable gRPC within Envoy.
func HTTP2ProtocolOptions() map[string]*anypb.Any {
	a, err := anypb.New(
		&upstreams.HttpProtocolOptions{
			UpstreamProtocolOptions: &upstreams.HttpProtocolOptions_ExplicitHttpConfig_{
				ExplicitHttpConfig: &upstreams.HttpProtocolOptions_ExplicitHttpConfig{
					ProtocolConfig: &upstreams.HttpProtocolOptions_ExplicitHttpConfig_Http2ProtocolOptions{},
				},
			},
		})
	if err != nil {
		panic(fmt.Sprintf("cannot construct http2 protocol options: %s", err))
	}

	return map[string]*anypb.Any{
		"envoy.extensions.upstreams.http.v3.HttpProtocolOptions": a,
	}
}
