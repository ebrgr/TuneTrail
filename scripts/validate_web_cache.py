#!/usr/bin/env python3
"""Exercise real cache and history behavior against fictional disposable data only."""
import argparse
from pathlib import Path
import subprocess
import tempfile
import time

from validate_multiuser import Server, WEB, LBZ, PASSWORD, scrobble


def main():
    """Build isolated credentials/listens and leave operational directories untouched."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--browser', choices=['chrome', 'firefox', 'https'], required=True)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=False)
    with tempfile.TemporaryDirectory(prefix='tunetrail-cache106-', dir='/tmp') as tmp:
        root = Path(tmp)
        server = Server(args.binary.resolve(strict=True), root/'data')
        try:
            server.start()
            server.login()
            profile = server.request('POST', WEB+'/profiles', {
                'display_name': 'Cache Study', 'slug': 'cache-study', 'avatar_icon': 'headphones',
            }, admin=True, expected=201)
            keys = server.request('GET', WEB+'/profiles/cache-study/user/apikeys', admin=True)
            for offset in range(3):
                server.request('POST', LBZ+'/submit-listens', scrobble(int(time.time())-3600-offset), token=keys[0]['key'])
            database = root/'data/tunetrail.db'
            (database.parent/'CACHE_TEST_FIXTURE').write_text('fictional disposable fixture')
            options = [server.origin, str(database), str(args.output.resolve()), args.browser, str(profile['id'])]
            if args.browser == 'https':
                config = root/'certificate.cnf'
                config.write_text('[req]\nprompt=no\ndistinguished_name=dn\nx509_extensions=ext\n[dn]\nCN=localhost\n[ext]\nsubjectAltName=IP:127.0.0.1,DNS:localhost\n')
                key, cert = root/'key.pem', root/'cert.pem'
                subprocess.run(['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes',
                                '-keyout', str(key), '-out', str(cert), '-days', '1', '-config', str(config)],
                               check=True, capture_output=True)
                options += [str(key), str(cert)]
            subprocess.run(['node', str(Path(__file__).with_suffix('.cjs')), *options], check=True, timeout=180)
        finally:
            server.stop()


if __name__ == '__main__':
    main()
