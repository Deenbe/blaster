#!/bin/bash

set -e

if [ ! -d .tmp ]
then
    mkdir .tmp
fi

if [ ! -f .tmp/blaster ]
then
curl -L -o .tmp/blaster https://github.com/buddyspike/blaster/releases/download/v0.1.16/blaster_linux_amd64
chmod +x .tmp/blaster
fi

docker build -t blaster-handler .
