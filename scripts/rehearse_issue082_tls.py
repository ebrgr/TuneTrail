#!/usr/bin/env python3
"""Compare real loopback TLS handshakes with and without claim-local reuse."""
import argparse
import http.client
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
from pathlib import Path
import socket
import ssl
import subprocess
import tempfile
import threading
import time


class Receiver(BaseHTTPRequestHandler):
    protocol_version = 'HTTP/1.1'

    def setup(self):
        self.request.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
        super().setup()
        self.server.handshakes += 1

    def do_POST(self):
        self.rfile.read(int(self.headers['Content-Length']))
        self.send_response(200)
        self.send_header('Content-Length', '0')
        self.end_headers()

    def log_message(self, *_):
        pass


def measure(client_ctx, server_ctx, claims, posts_per_claim, reuse):
    with ThreadingHTTPServer(('127.0.0.1', 0), Receiver) as server:
        server.handshakes = 0
        server.socket = server_ctx.wrap_socket(server.socket, server_side=True)
        worker = threading.Thread(target=server.serve_forever)
        worker.start()
        payload = b'{"payload":[' + b','.join([b'{"listened_at":1}'] * (100 // posts_per_claim)) + b']}'
        try:
            start = time.monotonic()
            for _ in range(claims):
                connection = http.client.HTTPSConnection('127.0.0.1', server.server_port, context=client_ctx, timeout=10)
                try:
                    for _ in range(posts_per_claim):
                        connection.request('POST', '/submit-listens', payload,
                                           headers={'Connection': 'keep-alive' if reuse else 'close'})
                        response = connection.getresponse()
                        assert response.status == 200
                        response.read()
                        if not reuse:
                            connection.close()
                finally:
                    connection.close()
            expected = claims if reuse else claims * posts_per_claim
            assert server.handshakes == expected
            return dict(claims=claims, posts_per_claim=posts_per_claim,
                        reused_within_claim=reuse, handshakes=expected,
                        seconds=round(time.monotonic() - start, 3))
        finally:
            server.shutdown()
            worker.join()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--claims', type=int, default=3000)
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='tunetrail-relay-tls-') as temp:
        key, cert = Path(temp) / 'key.pem', Path(temp) / 'cert.pem'
        subprocess.run(['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-days', '1',
                        '-subj', '/CN=relay-fixture', '-addext', 'subjectAltName=IP:127.0.0.1',
                        '-keyout', str(key), '-out', str(cert)], check=True, capture_output=True)
        client = ssl.create_default_context(cafile=cert)
        server = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        server.load_cert_chain(cert, key)
        for posts in (1, 4):
            for reuse in (False, True):
                print(json.dumps(measure(client, server, args.claims, posts, reuse)), flush=True)


if __name__ == '__main__':
    main()
