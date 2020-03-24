# Blaster
> Universal message pump for message brokers

[![Build Status](https://travis-ci.org/buddyspike/blaster.svg?branch=master)](https://travis-ci.org/buddyspike/blaster) [![codecov](https://codecov.io/gh/buddyspike/blaster/branch/master/graph/badge.svg)](https://codecov.io/gh/buddyspike/blaster)

Blaster is a cli utility to pump messages out of a message broker and forward them to a handler
written in any language. Blaster communicates with the handler via traditional IPC mechanisms.

### Usage

#### Step 1: Write a handler
Handler needs to expose the message handling function as an HTTP API. In this instance, we write a node script to achieve this.

```javascript
const express = require('express');

const app = express();
app.use(express.json());

app.post('/', (req, res) => {
    console.log(req.body);
    res.write('ok');
});

app.listen(8312, () => { // Default target URL is http://localhost:8312/
    console.log('listening');
});
```

#### Step 2: Launch blaster

Launch the handler with blaster (this should be executed in the directory containing node script):

```
blaster sqs --queue-name "test" --region "ap-southeast-2" --handler-name node --handler-args handler.js
```

### Road map
- Controls to throttle the pump based on various parameters and heuristics (CPU, Memory utilisation)
- Configuration based message routing
- Improve CLI to launch handler as part of boostrapping
- All the other crazy stuff...


