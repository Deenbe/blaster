#!/usr/bin/env python3

from http.server import HTTPServer, BaseHTTPRequestHandler
import json

class ServiceHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        length = int(self.headers.get('content-length'))
        body = json.loads(self.rfile.read(length))
        print(f'{body["messageId"]} {body["body"]}')
        self.wfile.write('ok'.encode())
        return

# Server Initialization
server = HTTPServer(('127.0.0.1', 8312), ServiceHandler)
server.serve_forever()
