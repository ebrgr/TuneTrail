"""Measure the current binary on fictional disposable data, never an operational server."""

import argparse
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import urllib.request

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'scripts'))
from validate_multiuser import Server, WEB


def main():
    """Collect selected headers and browser events without exporting session credentials."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=False)
    version = (ROOT / 'internal/version/VERSION').read_text().strip()
    with tempfile.TemporaryDirectory(prefix='tunetrail-study-106-') as tmp:
        server = Server(ROOT / 'tunetrail', Path(tmp) / 'data')
        try:
            server.start()
            server.login()
            server.request('POST', WEB + '/profiles', {
                'display_name': 'Study Alpha',
                'slug': 'study-alpha',
                'avatar_icon': 'headphones',
            }, admin=True, expected=201)
            rows = []
            paths = [
                '/web-assets/ui.css', '/web-assets/themes.css',
                '/web-assets/theme.js', '/web-assets/themes.json',
                '/study-alpha?period=all_time&tz=UTC', '/admin',
            ]
            headers = [
                'Cache-Control', 'ETag', 'Last-Modified', 'Vary',
                'Content-Type', 'X-Content-Type-Options',
            ]
            for path in paths:
                with urllib.request.urlopen(server.origin + path) as response:
                    rows.append({
                        'path': path,
                        'status': response.status,
                        'bodyBytes': len(response.read()),
                        'headers': {key: response.headers.get(key) for key in headers},
                    })
            report = {'version': version, 'fixtureOnly': True, 'requests': rows}
            (args.output / f'http-base-{version}.json').write_text(
                json.dumps(report, indent=2) + '\n')
            result = subprocess.run(
                ['node', str(Path(__file__).with_suffix('.cjs')), server.origin],
                capture_output=True, text=True, timeout=120,
            )
            if result.returncode:
                raise RuntimeError(result.stderr)
            (args.output / f'chrome-base-{version}.json').write_text(result.stdout)
        finally:
            server.stop()


if __name__ == '__main__':
    main()
