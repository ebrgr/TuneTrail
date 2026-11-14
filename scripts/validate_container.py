#!/usr/bin/env python3
"""Verify a local image with a disposable container and a new named volume."""
import argparse
import http.cookiejar
import json
import sqlite3
import subprocess
import tempfile
import time
import urllib.request
import urllib.error
import uuid
from pathlib import Path


def docker(*args, check=True):
    """Run Docker without a shell and keep output out of user data paths."""
    return subprocess.run(['docker', *map(str, args)], check=check, text=True, capture_output=True)


def request_json(origin, path):
    """Wait for the host port forward as well as Docker's internal healthcheck."""
    for attempt in range(40):
        try:
            with urllib.request.urlopen(origin + path, timeout=3) as response:
                return response.status, response.read()
        except (OSError, urllib.error.URLError):
            if attempt == 39:
                raise
            time.sleep(.25)


def main():
    """Validate health, static runtime dependencies, persistent SQLite and SIGTERM."""
    parser = argparse.ArgumentParser()
    parser.add_argument('--image', required=True)
    parser.add_argument('--tls-probe', type=Path, help='optional static Go TLS probe compiled for the container architecture')
    args = parser.parse_args()
    suffix = uuid.uuid4().hex[:12]
    container, volume = 'koito034-' + suffix, 'koito034-' + suffix
    report = {}
    created_volume = created_container = False
    try:
        docker('volume', 'create', volume); created_volume = True
        docker('run', '-d', '--name', container, '--health-interval=2s', '--health-start-period=0s',
               '--health-retries=15', '-p', '127.0.0.1::4110', '-v', f'{volume}:/etc/tunetrail',
               '-e', 'TUNETRAIL_DEFAULT_USERNAME=validation-admin', '-e', 'TUNETRAIL_DEFAULT_PASSWORD=fictional-validation-password',
               '-e', 'TUNETRAIL_FORCE_TZ=America/Sao_Paulo', '-e', 'TUNETRAIL_SKIP_IMPORT=true',
               '-e', 'TUNETRAIL_DISABLE_MUSICBRAINZ=true', '-e', 'TUNETRAIL_DISABLE_DEEZER=true',
               '-e', 'TUNETRAIL_DISABLE_COVER_ART_ARCHIVE=true', '-e', 'TUNETRAIL_LOG_LEVEL=warn', args.image)
        created_container = True
        address = docker('port', container, '4110/tcp').stdout.strip().splitlines()[0]
        origin = 'http://' + address
        for _ in range(45):
            status = docker('inspect', '-f', '{{.State.Health.Status}}', container).stdout.strip()
            if status == 'healthy':
                break
            if status == 'unhealthy':
                raise RuntimeError('container became unhealthy')
            time.sleep(1)
        else:
            raise TimeoutError('container did not become healthy')
        status, _ = request_json(origin, '/apis/web/v1/health')
        if status != 200: raise RuntimeError('health endpoint failed')
        _, body = request_json(origin, '/apis/web/v1/profiles')
        profiles = json.loads(body)
        if len(profiles) != 1 or profiles[0]['slug'] != 'default':
            raise RuntimeError('unexpected profiles')
        # Persist an extra fictional profile so a recreated database cannot pass.
        client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
        def post(path, payload):
            req = urllib.request.Request(origin + path, data=json.dumps(payload).encode(), method='POST',
                                         headers={'Content-Type': 'application/json', 'Origin': origin})
            with client.open(req, timeout=5) as response:
                return response.status, response.read()
        login_status, _ = post('/apis/web/v1/login', {'username': 'validation-admin', 'password': 'fictional-validation-password'})
        if login_status != 204: raise AssertionError('login failed')
        create_status, body = post('/apis/web/v1/profiles', {'display_name': 'Persistence marker', 'slug': 'persistence-marker', 'avatar_icon': 'music'})
        if create_status != 201: raise AssertionError('profile creation failed')
        marker_id = json.loads(body)['id']
        docker('exec', container, 'sh', '-c', '! dpkg-query -W libvips42 >/dev/null 2>&1 && test -s /etc/ssl/certs/ca-certificates.crt && test -f /usr/share/zoneinfo/America/Sao_Paulo')
        zone = docker('exec', '-e', 'TZ=America/Sao_Paulo', container, 'date', '-d', '@1550367000', '+%Y-%m-%d %H:%M %z').stdout.strip()
        if zone != '2019-02-16 23:30 -0200':
            raise RuntimeError(f'unexpected timezone conversion: {zone}')
        tls = docker('exec', container, 'curl', '--fail', '--silent', '--show-error', '--max-time', '15', 'https://example.com/', check=False)
        if args.tls_probe:
            docker('cp', args.tls_probe, f'{container}:/tmp/koito-tls-probe')
            docker('exec', container, '/tmp/koito-tls-probe')
            report['go_https'] = 'ok'
        report.update(health='healthy', timezone=zone, libvips='absent', ca_bundle='present',
                      external_https='ok' if tls.returncode == 0 else 'unavailable', external_https_exit=tls.returncode)
        docker('stop', '--time', '20', container)
        code = int(docker('inspect', '-f', '{{.State.ExitCode}}', container).stdout.strip())
        if code != 0:
            raise RuntimeError(f'SIGTERM exit code {code}')
        with tempfile.TemporaryDirectory(prefix='koito034-sqlite-') as temporary:
            database = Path(temporary) / 'copy.db'
            docker('cp', f'{container}:/etc/tunetrail/tunetrail.db', database)
            with sqlite3.connect(database) as conn:
                if conn.execute('PRAGMA integrity_check').fetchall() != [('ok',)] or conn.execute('PRAGMA foreign_key_check').fetchall():
                    raise AssertionError('database integrity')
                version = conn.execute('SELECT max(version_id) FROM goose_db_version WHERE is_applied=1').fetchone()[0]
                if version != 3:
                    raise AssertionError(f'unexpected schema {version}')
        docker('start', container)
        for _ in range(30):
            if docker('inspect', '-f', '{{.State.Health.Status}}', container).stdout.strip() == 'healthy':
                break
            time.sleep(1)
        else:
            raise TimeoutError('restart healthcheck')
        address = docker('port', container, '4110/tcp').stdout.strip().splitlines()[0]
        _, body = request_json('http://' + address, '/apis/web/v1/profiles')
        if not any(p['slug'] == 'persistence-marker' and p['id'] == marker_id for p in json.loads(body)):
            raise AssertionError('fictional marker did not survive restart')
        docker('stop', '--time', '20', container)
        report.update(sigterm_exit=code, sqlite='v3 integrity ok', persistent_restart='ok')
        print(json.dumps(report, ensure_ascii=False, sort_keys=True))
    finally:
        if created_container:
            docker('rm', '-f', container, check=False)
        if created_volume:
            docker('volume', 'rm', volume, check=False)


if __name__ == '__main__':
    main()
