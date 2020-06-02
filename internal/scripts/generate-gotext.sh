#!/bin/sh
if ! [ -x "$(command -v gotext)" ]; then
  echo 'Error: gotext is not installed.' >&2
  exit 1
fi
go generate ./internal/translations/translations.go