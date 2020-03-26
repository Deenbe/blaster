# Blaster
> Got a message? Just handle it!

[![Build Status](https://travis-ci.org/buddyspike/blaster.svg?branch=master)](https://travis-ci.org/buddyspike/blaster) [![codecov](https://codecov.io/gh/buddyspike/blaster/branch/master/graph/badge.svg)](https://codecov.io/gh/buddyspike/blaster) [![Go Report Card](https://goreportcard.com/badge/github.com/buddyspike/blaster)](https://goreportcard.com/report/github.com/buddyspike/blaster)

Blaster is a utility to read messages from a message broker and forward them to a message handler over a local http endpoint. Handlers focus on the application logic while Blaster takes care of the complex tasks like parallel execution, throttling and retrying.

## Usage

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


## Road map

Currently blaster supports AWS SQS. Following broker support is under development.

- Kafka
- Rabbit MQ


<sub><sup>Made in Australia with ❤ <sub><sup>
