#!/bin/bash

set -e

make clean
make test
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 make build
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make build
CGO_ENABLED=0 GOOS=windows EXTENSION=".exe" GOARCH=amd64 make build

bash <(curl -s https://codecov.io/bash)
