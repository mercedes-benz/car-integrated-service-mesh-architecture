#!/bin/bash
 # SPDX-License-Identifier: MIT
 # Copyright (c) 2026 MBition GmbH

for PROTO in proto/*.proto; do
    protoc -I=proto --cpp_out=./adapter --grpc_out=./adapter --plugin=protoc-gen-grpc=/usr/local/bin/grpc_cpp_plugin "$PROTO"
done