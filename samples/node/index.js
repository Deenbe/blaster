#!/usr/bin/env node 

const express = require('express');

const app = express();
app.use(express.json());

app.post('/', (req, res) => {
    console.log(`${req.body.messageId}: ${req.body.body}`);
    return new Promise((resolve) => {
        setTimeout(() => {
            res.send('ok');
            resolve();
        }, 2000);
    });
});

app.listen(8312, () => {
    console.log('listening');
});
