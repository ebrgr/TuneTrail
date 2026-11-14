#!/usr/bin/env python3
"""Check that VS Code's default launch starts only Go and opens its address."""
import json
import re
from pathlib import Path


def main():
    """Validate launch settings and the actual server log pattern."""
    config = json.loads((Path(__file__).resolve().parents[1] / '.vscode/launch.json').read_text())
    first = config['configurations'][0]
    assert first['type'] == 'go' and first['request'] == 'launch'
    assert first['env']['KOITO_WEB_UI'] == 'go'
    assert 'preLaunchTask' not in first
    ready = first['serverReadyAction']
    assert ready['action'] == 'openExternally'
    match = re.search(ready['pattern'], 'Engine: Listening on 127.0.0.1:4110')
    assert match and ready['uriFormat'] % match.group(1) == 'http://127.0.0.1:4110'
    assert all(item.get('env', {}).get('CGO_ENABLED') == '0' for item in config['configurations'])
    assert all("React" not in item['name'] for item in config['configurations'])
    print('PASS: VS Code F5 uses Go and opens its local address')


if __name__ == '__main__':
    main()
