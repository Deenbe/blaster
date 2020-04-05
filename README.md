![blaster](assets/blaster.png)

# Blaster
> Got a message? Just handle it!

[![Build Status](https://travis-ci.org/buddyspike/blaster.svg?branch=master)](https://travis-ci.org/buddyspike/blaster) [![codecov](https://codecov.io/gh/buddyspike/blaster/branch/master/graph/badge.svg)](https://codecov.io/gh/buddyspike/blaster) [![Go Report Card](https://goreportcard.com/badge/github.com/buddyspike/blaster)](https://goreportcard.com/report/github.com/buddyspike/blaster)

Blaster is a utility to read messages from a message broker and forward them to a message handler over a local http endpoint. Handlers focus on the application logic while Blaster takes care of the complex tasks like parallel execution, throttling and retrying.

## Example

**Step 1: Write a handler**

This example uses a script written in javascript to build the message handler.

```javascript
#!/usr/bin/env node

const express = require('express');

const app = express();
app.use(express.json());

app.post('/', (req, res) => {
    console.log(req.body);
    res.send('ok');
});

// By default blaster forwards messages to http://localhost:8312/
app.listen(8312, () => { 
    console.log('listening');
});
```

**Step 2: Launch it with blaster**

Launch the handler with blaster (this should be executed in the directory containing node script):

```
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


## Supported Brokers

- AWS SQS

## Contributing

```
git clone https://github.com/buddyspike/blaster
cd blaster

# Run tests
make test

# Build binary
make build

# Build binary and copy it to path
make build-local

# Build cross compiled binaries
./build.sh
```

## Credits

- Icons made by <a href="https://www.flaticon.com/authors/nikita-golubev" title="Nikita Golubev">Nikita Golubev</a> from <a href="https://www.flaticon.com/" title="Flaticon"> www.flaticon.com</a>

<sub><sup>Made in Australia with ❤ <sub><sup>
