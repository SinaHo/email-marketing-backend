#!/usr/bin/env bash
set -e

PROTO_SRC=./api/v1/proto
OUT_PKG=github.com/SinaHo/email-marketing-backend/api/v1/proto

# Ensure output directory exists
mkdir -p "$(go env GOPATH)/pkg/mod" # generally unneeded if modules enabled
mkdir -p internal

# Generate Go code, including gRPC stubs
protoc \
  --proto_path="$PROTO_SRC" \
  --go_out="$PROTO_SRC" \
  --go_opt=paths=source_relative \
  --go-grpc_out="$PROTO_SRC" \
  --go-grpc_opt=paths=source_relative \
  "$PROTO_SRC"/*.proto

echo "✅ Protobuf code generated"