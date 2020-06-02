#!/bin/sh
if ! [ -x "$(command -v protoc)" ]; then
  echo 'Error: protoc is not installed.' >&2
  exit 1
fi
protoc --proto_path=. --go_out=plugins=grpc:. api/proto/v1/gooser_service.proto