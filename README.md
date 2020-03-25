# Blaster
> Got a message? Just handle it!

[![Build Status](https://travis-ci.org/buddyspike/blaster.svg?branch=master)](https://travis-ci.org/buddyspike/blaster) [![codecov](https://codecov.io/gh/buddyspike/blaster/branch/master/graph/badge.svg)](https://codecov.io/gh/buddyspike/blaster) [![Go Report Card](https://goreportcard.com/badge/github.com/buddyspike/blaster)](https://goreportcard.com/report/github.com/buddyspike/blaster)

Blaster is a cli utility to pump messages out of a message broker and forward them to a handler
written in any language. Handler must listen for incoming messages via an http endpoint. Blaster takes care of the complex tasks in message handling such as throttling and retrying so that you can just focus on message handling logic.

This project is currently under heavy churn. We invite all the adventurous folks to give it a whirl and help us make it solid.

Made with ❤ in Australia

### Usage

#### Step 1: Write a handler
Handler needs to expose the message handling function as an HTTP API. In this instance, we write a node script to achieve this.

```javascript
#!/usr/bin/env node

const express = require('express');

const app = express();
app.use(express.json());

app.post('/', (req, res) => {
    console.log(req.body);
    res.send('ok');
});

app.listen(8312, () => { // Default target URL is http://localhost:8312/
    console.log('listening');
});
```

#### Step 2: Launch blaster

Launch the handler with blaster (this should be executed in the directory containing node script):

```
blaster sqs --queue-name "test" --region "ap-southeast-2" --handler-command ./handler.js
```
