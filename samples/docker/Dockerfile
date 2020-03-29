from node:10.15.3-alpine

RUN mkdir /usr/local/handler
WORKDIR /usr/local/handler
COPY .tmp/blaster /usr/local/bin/
COPY *.js *.json /usr/local/handler/

RUN npm install

ENTRYPOINT ["blaster", "sqs", "--handler-command", "./index.js", "--startup-delay-seconds", "0"]
