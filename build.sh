#!/bin/sh

set -e

make test
bash <(curl -s https://codecov.io/bash)
