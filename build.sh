#!/bin/bash

set -e

make test
make clean
GOOS=darwin GOARCH=amd64 make build
GOOS=linux GOARCH=amd64 make build
GOOS=windows GOARCH=amd64 make build

bash <(curl -s https://codecov.io/bash)
