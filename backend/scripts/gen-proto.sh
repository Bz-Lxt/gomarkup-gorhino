#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
export PATH="$(go env GOPATH)/bin:$PATH"
protoc --go_out=. --go_opt=module=gorhino \
  --go-grpc_out=. --go-grpc_opt=module=gorhino \
  proto/control.proto
