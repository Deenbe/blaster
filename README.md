# Blaster

## Problem Statement

- Messages are published to a queue for processing
- Number of processing nodes pull messages and process them
- It makes the process more efficient to pre-fetch messages as they are processed on each node
- However, when message processing runs out of available resources, pre-fetched messages are not going to be available for another host with enough resources.
