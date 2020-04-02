#!/usr/bin/env node

const express = require('express');

const app = express();
app.use(express.json());

app.post('/', (req, res) => {
    console.log(`pid: ${process.pid} partion: ${req.body.properties.partitionId} offset: ${req.body.properties.offset} messageId: ${req.body.messageId}: ${req.body.body}`);
    return res.send('ok');
});

// By default blaster forwards messages to http://localhost:8312
const port = process.env['BLASTER_HANDLER_PORT'] || 8312;
app.listen(port, () => {
    console.log('listening on port ', port);
});
