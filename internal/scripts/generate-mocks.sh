#!/bin/sh
if ! [ -x "$(command -v mockery)" ]; then
  echo 'Error: mockery is not installed.' >&2
  exit 1
fi
mockery -case underscore -recursive -name Store -output ./internal/mocks
mockery -case underscore -recursive -name UserLookup -output ./internal/mocks
mockery -case underscore -recursive -name MailClient -output ./internal/mocks
mockery -case underscore -recursive -name MessageDeliverer -output ./internal/mocks