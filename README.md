![blaster](assets/blaster-128.png)

# Blaster
> A reliable message processing pipeline

[![Build Status](https://travis-ci.org/buddyspike/blaster.svg?branch=master)](https://travis-ci.org/buddyspike/blaster) [![codecov](https://codecov.io/gh/buddyspike/blaster/branch/master/graph/badge.svg)](https://codecov.io/gh/buddyspike/blaster) [![Go Report Card](https://goreportcard.com/badge/github.com/buddyspike/blaster)](https://goreportcard.com/report/github.com/buddyspike/blaster)

Blaster is a utility to read messages from a message broker and forward them to a message handler over a local http endpoint. Handlers focus on the application logic while Blaster takes care of the complex matters like parallel execution, fault tolerance and quality of service.

## Table of Contents

- [Introduction](#Introduction) 
- [Example](#Example)
- [Usage](#Usage)
   * [Global Options](#Global-Options)
   * [Broker Options](#Broker-Options)
     - [SQS](#SQS)
     - [Kafka](#Kafka)
- [Message Schema](#Message-Schema)
- [Deploy with Docker](#Deploy-With-Docker)
- [Contributing](#Contributing)
- [Credits](#Credits)

## Introduction
A typical workflow to consume messages in a message queue is; fetch one message, process, remove and repeat. This seemingly straightforward process however is often convoluted by the logic required to deal with subtleties in message processing. Following list summarises some of the common complexities without being too exhaustive. 

- Read messages in batches to reduce network round-trips.
- Enhance the work distribution by intelligently filling read ahead buffers. 
- Retry handling the messages when there are intermittent errors.
- Reduce the stress on recovering downstream services with circuit breakers.
- Process multiple messages simultaneously.
- Prevent exhausting the host resources by throttling the maximum number of messages processed at any given time.

Blaster simplifies the message handling code by providing a flexible message processing pipeline with built-in features to deal with the well-known complexities. 

It's built with minimum overhead (i.e. cpu/memory) to ensure that the handlers are cost effective when operated in pay-as-you-go infrastructures (e.g. AWS Fargate).

## Example

**Step 1: Write a handler**

Blaster message handler is any executable exposing the message handling logic as an HTTP endpoint. In this example, it is a script written in Javascript.

```javascript
#!/usr/bin/env node

const express = require('express');

const app = express();
app.use(express.json());

// Messages are submitted to the handler endpoint as HTTP POST requests
app.post('/', (req, res) => {
    console.log(req.body);
    res.send('ok');
});

// By default blaster forwards messages to http://localhost:8312/
app.listen(8312, () => { console.log('listening'); });
```

**Step 2: Launch it with blaster**

Now that we have a message handler, we can launch blaster to handle messages stored in a supported broker. For instance, to process messages in an AWS SQS queue called test with the script created in step 1, launch blaster with following command (this should be executed in the directory containing node script):

```
chmod +x ./handler.js
blaster sqs --queue-name "test" --region "ap-southeast-2" --handler-command ./handler.js
```

## Usage

```
blaster <broker> [options]

```

### Global Options
`--handler-command`

Command to launch the handler.

`--handler-args`

Comma separated list of arguments to the handler command.

`--max-handlers`

Maximum number of concurrent handlers to execute. By default this is calculated by multiplying the number of CPUs available to the process by 256.

`--startup-delay-seconds`

Number of seconds to wait on start before delivering messages to the handler. Default setting is five seconds. Turning startup delay off by setting it to zero will notify blaster that handler's readiness endpoint should be probed instead of a static delay. Readiness endpoint must listen for HTTP GET requests at the handler URL. When handler is ready to accept message, readiness endpoint must respond with an HTTP 200.

`--handler-url`

Endpoint that handler is listening on. Default value is http://localhost:8312/

`--retry-count`

When the handler does not respond with an HTTP 200, blaster retries the delivery for the number of times indicated by this option.

`--retry-delay-seconds`

Number of seconds to wait before retrying the delivery of a message.

`--version`

Show blaster version

### Broker Options

#### SQS

`--region`

AWS region for SQS queue

`--queue-name`

Name of the queue. Blaster will resolve the queue URL using its name and the region.

`--max-number-of-messages`

Maximum number of messages to receive in a single poll from SQS. Default setting is 1 and the maximum value supported is 10.

`--wait-time-seconds`

Blaster uses long polling when receiving messages from SQS. Use this option to control the delay between polls. Default setting is 1.

#### Kafka

`--brokers`

Comma separated list of broker addresses.

`--topic`

Name of the topic to read messages from.

`--group`

Name of the consumer group. Blaster creates a handler instance for each partition assigned to a member of the consumer group. Each message is sequentially delivered to the handler in the order they are received.

`--start-from-oldest`

Force blaster to start reading the partition from oldest available offset.

`--buffer-size`

Number of messages to be read into the local buffer.

## Message Schema

Since Blaster is designed to work with many different message brokers, it converts the message to a  general purpose format before forwarding it to the handler.

```
{
    "$schema": "http://json-schema.org/schema#",
    "$id": "https://github.com/buddyspike/blaster/message-schema.json",
    "title": "Message",
    "type": "object",
    "properties": {
        "messageId": {
            "type": "string",
            "description": "Unique message id that is generally assigned by the broker"
        },
        "body": {
            "type": "string",
            "description": "Message body with the content"
        },
        "properties": {
            "type": "object",
            "description": "Additional information available in the message such as headers"
        }
    }
}
```

## Deploy with Docker

To deploy blaster handler in a docker container, copy the linux binary from [Releases](https://github.com/buddyspike/blaster/releases) to the path and set the entry point with desired options.

```
from node:10.15.3-alpine

RUN mkdir /usr/local/handler
WORKDIR /usr/local/handler
COPY .tmp/blaster /usr/local/bin/
COPY *.js *.json /usr/local/handler/

RUN npm install

ENTRYPOINT ["blaster", "sqs", "--handler-command", "./index.js", "--startup-delay-seconds", "0"]
```

Full example can be found [here](https://github.com/buddyspike/blaster/tree/master/samples/docker).

## Contributing

```
git clone https://github.com/buddyspike/blaster
cd blaster

# Run tests
make test

# Build binary
make build

# Build binary and copy it to path
make install

# Build cross compiled binaries
./build.sh
```

## Credits

- Icons made by <a href="https://www.flaticon.com/authors/nikita-golubev" title="Nikita Golubev">Nikita Golubev</a> from <a href="https://www.flaticon.com/" title="Flaticon"> www.flaticon.com</a>

<sub><sup>Made in Australia with ❤ <sub><sup>
