#!/usr/bin/env node

const express = require('express');

const app = express();
app.use(express.json());

const aggregatedState = {}

app.post('/', (req, res) => {
    let messageId = req.body.messageId;
    let current = aggregatedState[messageId];
    if (!current) {
        current = 0;
    }
    aggregatedState[messageId] = current + 1;
    return res.send('ok');
});

app.post('/pre-commit', (_, res) => {
    console.log(aggregatedState)
    return res.send('ok');
})

// Bind to the port assigned by blaster or default port. Using default
// port would only work if the topic has a single partition.
const port = process.env['BLASTER_HANDLER_PORT'] || 8312;
app.listen(port, () => {
    console.log('kafka-node-streaming-sample handler listening on port ', port);
});
