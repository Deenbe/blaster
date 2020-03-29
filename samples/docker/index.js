#!/usr/bin/env node 

const express = require('express');

const app = express();
app.use(express.json());

app.get('/', (req, res) => res.send('ok'));
app.post('/', (req, res) => { 
    console.log(`${req.body.messageId}: ${req.body.body}`);
    return res.send('ok');
});

// By default blaster forwards messages to http://localhost:8312
app.listen(8312, () => { console.log('listening'); });
