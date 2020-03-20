# Blaster
> Universal message pump for message brokers

Blaster is a cli utility to pump messages out of a message broker and forward them to a handler
written in any language. Blaster communicates with the handler via traditional IPC mechanisms.

### Usage

Given that we have an http endpoint that can handle a message stored in a Amazon SQS queue,
we can start blaster like this:

```
blaster sqs --queue-name "test" --region "ap-southeast-2" --target "http://localhost:9000/"
```

### Road map
- Controls to throttle the pump based on various parameters and heuristics (CPU, Memory utilisation)
- Configuration based message routing
- Improve CLI to launch handler as part of boostrapping
- All the other crazy stuff...


