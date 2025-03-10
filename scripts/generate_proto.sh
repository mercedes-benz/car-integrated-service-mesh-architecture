#!/bin/bash
# SPDX-License-Identifier: Apache-2.0

mkdir -p pkg/protobuf

protoc --go_out=pkg/protobuf \
       --go_opt=paths=source_relative \
       --go-grpc_out=pkg/protobuf \
       --go-grpc_opt=paths=source_relative\
       -I=api/proto \
       $(find api/proto -name "*.proto")