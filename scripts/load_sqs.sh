#!/bin/bash

# Loads a specified number of messages to an SQS queue.

set -e

REGION=$(aws configure get region)
ACCOUNT=$1
QUEUE=$2

for i in {1..10}
do
    aws sqs send-message --queue-url "https://sqs.${REGION}.amazonaws.com/${ACCOUNT}/${QUEUE}" --message-body "message # ${i}" | cat
done
