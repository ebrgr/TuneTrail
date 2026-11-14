#!/usr/bin/env python3
"""Validate Go pages using a real API process and a disposable SQLite v3 database.

--serve keeps the process for screenshots until Ctrl+C. Never loads the project's .env.
"""
import argparse
import json
from pathlib import Path
import re
import signal
import sqlite3
import time
import urllib.parse
import uuid
from validate_multiuser import Server, WEB, LBZ, ROOT


def upload_image(server, path, data):
    boundary = uuid.uuid4().hex
    body = (f'--{boundary}\r\nContent-Disposition: form-data; name="image"; filename="fixture.webp"\r\nContent-Type: image/webp\r\n\r\n').encode() + data + f'\r\n--{boundary}--\r\n'.encode()
    server.request('PATCH', path, body, admin=True, headers={'Content-Type': f'multipart/form-data; boundary={boundary}'})


def seed(server):
    server.login()
    profile = server.request('GET', WEB + '/profiles')[0]
    server.request('PATCH', WEB + f'/profiles/{profile["slug"]}', {'display_name': 'Mara Silva', 'slug': 'mara'}, admin=True)
    for name, slug in [('Luna Paz', 'luna'), ('Iara Costa', 'iara')]:
        server.request('POST', WEB + '/profiles', {'display_name': name, 'slug': slug, 'avatar_icon': 'music'}, admin=True, expected=201)
    key = server.request('GET', WEB + '/profiles/mara/user/apikeys', admin=True)[0]['key']
    now = int(time.time()) - 3600
    artists = ['Estação Azul', 'Maré Baixa', 'Órbita', '花と光', 'Entre Rios']
    titles = ['Primeira Luz', 'Mar Aberto', 'Dança Lenta', 'Um Lugar', 'Ao Longe', 'Brisa', 'Horizonte', 'Noite Clara', 'Caminhos', 'Chuva', 'Satélite', '花と光']
    payload = []
    for i in range(152):
        age = i * 1800 if i < 140 else (40 + i - 140) * 86400
        payload.append({'listened_at': now - age, 'track_metadata': {'artist_name': artists[i % 5], 'release_name': ['Horizonte', 'Depois da Chuva', 'Satélite', '夜の色', 'Caminhos'][i % 5], 'track_name': titles[i % 12], 'additional_info': {'duration': 180}}})
    server.request('POST', LBZ + '/submit-listens', {'listen_type': 'import', 'payload': payload}, token=key)
    second = server.request('GET', WEB + '/profiles/luna/user/apikeys', admin=True)[0]['key']
    server.request('POST', LBZ + '/submit-listens', {'listen_type': 'single', 'payload': [{'listened_at': now, 'track_metadata': {'artist_name': 'Luna Artist', 'release_name': 'Luna Album', 'track_name': 'Exclusive Luna Song'}}]}, token=second)
    # Local generated artwork, uploaded through the real administrative endpoint.
    image = (ROOT / 'internal/webui/static/avatar.webp').read_bytes()
    for kind in ['artists', 'albums']:
        records = server.request('GET', WEB + f'/profiles/mara/top/{kind}?period=all_time')['items']
        for record in records[:5]:
            upload_image(server, WEB + f'/{kind[:-1]}/{record["item"]["id"]}/image', image)
    return key


def validate(server, key):
    pages = ['/', '/mara?period=all_time', '/mara/chart/top/artists', '/mara/chart/top/albums', '/mara/chart/top/tracks', '/mara/listens?limit=100&page=2', '/iara']
    for path in pages:
        content = server.request('GET', path).decode()
        assert 'data-assets="/web-assets"' in content, f'Go HTML not served: {path}'
        assert key not in content and 'Exclusive Luna Song' not in content
    stats = server.request('GET', WEB + '/profiles/mara/stats?period=all_time')
    home = server.request('GET', '/mara?period=all_time').decode()
    assert f'data-listen-count="{stats["listen_count"]}"' in home
    assert stats['listen_count'] == 152
    result = server.request('GET', WEB + '/profiles/mara/listens?limit=100&page=2')
    history = server.request('GET', '/mara/listens?limit=100&page=2').decode()
    assert len(re.findall(r'<tr data-track-id=', history)) == len(result['items']) == 52
    for params in ['limit=-1', 'page=-1', 'month=13', 'year=2025&week=53', 'period=unknown', 'page=99999999999999']:
        server.request('GET', '/mara/listens?' + params, expected=400)
    server.request('GET', '/missing', expected=404)
    for kind in ['artist', 'album', 'track']:
        content = server.request('GET', f'/mara/{kind}/1').decode()
        assert 'data-assets="/web-assets"' in content
        assert f'/mara/manage/{kind}/1' in content
        server.request('GET', f'/mara/{kind}/0', expected=400)
        server.request('GET', f'/iara/{kind}/1', expected=404)
        assert 'data-admin-ui="go"' in server.request('GET', f'/mara/manage/{kind}/1', admin=True).decode()
    assert 'data-assets="/web-assets"' in server.request('GET', '/mara/rewind?year=2026&month=1').decode()
    server.request('GET', '/admin')
    assert 'Exclusive Luna Song' not in server.request('GET', '/mara/search?q=Exclusive').decode()
    assert 'Exclusive Luna Song' in server.request('GET', '/luna/search?q=Exclusive').decode()
    assert server.request('GET', WEB + '/config')['web_ui'] == 'go'
    # Interest keeps its shape, scoped counts and safe input bounds in the real API.
    interest_totals = {}
    for kind in ['artist', 'album', 'track']:
        entity = server.request('GET', WEB + f'/profiles/mara/{kind}/1')
        intervals = server.request('GET', WEB + f'/profiles/mara/{kind}/1/interest?buckets=16')
        assert len(intervals) == 16
        total = sum(row['listen_count'] for row in intervals)
        assert total == entity['listen_count']
        interest_totals[kind] = total
        for i in range(1, len(intervals)):
            assert intervals[i]['bucket_start'] == intervals[i-1]['bucket_end']
        other = server.request('GET', WEB + f'/profiles/luna/{kind}/1/interest?buckets=16')
        assert sum(row['listen_count'] for row in other) == 0
        for count in ['-1', '0', '513', '999999999999999999999']:
            server.request('GET', WEB + f'/profiles/mara/{kind}/1/interest?buckets={count}', expected=400)
        server.request('GET', WEB + f'/profiles/mara/{kind}/0/interest?buckets=16', expected=400)
    # Auth and origin checks use a disposable record, then restore it for comparisons.
    listen = server.request('GET', WEB + '/profiles/mara/listens?limit=1')['items'][0]
    from datetime import datetime
    timestamp = int(datetime.fromisoformat(listen['time'].replace('Z', '+00:00')).timestamp())
    suffix = '?track_id=' + str(listen['track']['id']) + '&unix=' + str(timestamp)
    target = WEB + '/profiles/mara/listens' + suffix
    server.request('DELETE', target, token=key, expected=401)
    server.request('DELETE', target, admin=True, headers={'Origin': 'https://elsewhere.invalid'}, expected=403)
    server.request('DELETE', target, admin=True, expected=204)
    server.request('POST', WEB + '/profiles/mara/listens', {'track_id': listen['track']['id'], 'unix': timestamp, 'client': 'UI validation'}, admin=True, expected=201)
    # The application database is tunetrail.db since ISSUE-046; koito.db is only the legacy source.
    with sqlite3.connect(server.directory / 'tunetrail.db') as conn:
        assert conn.execute('PRAGMA integrity_check').fetchone()[0] == 'ok'
        assert not conn.execute('PRAGMA foreign_key_check').fetchall()
        assert conn.execute('SELECT MAX(version_id) FROM goose_db_version WHERE is_applied=1').fetchone()[0] == 3
    return {'profiles': 3, 'mara_listens': stats['listen_count'], 'history_page_2': 52, 'schema': 3, 'interest_totals': interest_totals, 'http_pages': pages, 'checks': ['scoping', 'API/HTML parity', 'invalid filters', 'interest intervals and bounds', 'Go management links and media', 'admin authentication', 'same-origin mutations', 'SQLite integrity']}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--work-dir', type=Path, required=True, help='Must be a new directory')
    parser.add_argument('--serve', action='store_true')
    parser.add_argument('--go-port', type=int)
    args = parser.parse_args()
    root = args.work_dir.resolve(); root.mkdir(parents=True, exist_ok=False); root.chmod(0o700)
    binary = args.binary.resolve()
    go = Server(binary, root / 'go', environment={'KOITO_WEB_UI': 'go', 'KOITO_FORCE_TZ': 'America/Sao_Paulo'}, port=args.go_port)
    try:
        # The React comparison server was removed with React itself (ISSUE-030).
        go.start(); key = seed(go); report = validate(go, key)
        report.update(go_url=go.origin)
        (root / 'report.json').write_text(json.dumps(report, indent=2) + '\n')
        print(json.dumps(report), flush=True)
        if args.serve:
            signal.signal(signal.SIGTERM, lambda *_: (_ for _ in ()).throw(KeyboardInterrupt()))
            while True: time.sleep(1)
    except KeyboardInterrupt:
        pass
    finally:
        go.stop()

if __name__ == '__main__': main()
