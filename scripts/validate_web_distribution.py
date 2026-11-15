#!/usr/bin/env python3
"""Exercise external asset distribution with fictional data and disposable copies only."""
import argparse
import os
from pathlib import Path
import shutil
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
from validate_multiuser import ROOT, clean_environment


def environment(data):
    """Do not inherit operational configuration or checkout resource overrides."""
    env = clean_environment()
    env.pop('TUNETRAIL_LANGS_DIR', None)
    env.pop('TUNETRAIL_WEB_DIR', None)
    env.update(TUNETRAIL_CONFIG_DIR=str(data), TUNETRAIL_DEFAULT_PASSWORD='fixture-password',
               TUNETRAIL_BIND_ADDR='127.0.0.1', TUNETRAIL_SKIP_IMPORT='true',
               TUNETRAIL_DISABLE_MUSICBRAINZ='true', TUNETRAIL_DISABLE_DEEZER='true',
               TUNETRAIL_DISABLE_COVER_ART_ARCHIVE='true', TUNETRAIL_LOG_LEVEL='error')
    return env


def main():
    """Validate adjacent resources through an executable symlink, startup guards, and snapshot isolation."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    args = parser.parse_args()
    binary = args.binary.resolve(strict=True)
    with tempfile.TemporaryDirectory(prefix='tunetrail-web-distribution-') as tmp:
        root = Path(tmp)
        package = root/'release'
        package.mkdir()
        target = package/'tunetrail'
        shutil.copy2(binary, target)
        shutil.copytree(ROOT/'assets', package/'assets')
        shutil.copytree(ROOT/'langs', package/'langs')
        link = root/'server-link'
        link.symlink_to(target)
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0))
            port = sock.getsockname()[1]
        env = environment(root/'data')
        env['TUNETRAIL_LISTEN_PORT'] = str(port)
        origin = f'http://127.0.0.1:{port}'
        with (root/'server.log').open('w') as log:
            process = subprocess.Popen([str(link)], cwd='/', env=env, stdout=log, stderr=log)
            try:
                for _ in range(100):
                    if process.poll() is not None:
                        raise AssertionError('distribution server exited before readiness')
                    try:
                        with urllib.request.urlopen(origin+'/', timeout=1) as response:
                            assert response.status == 200
                        break
                    except OSError:
                        time.sleep(.1)
                else:
                    raise AssertionError('distribution readiness timeout')
                with urllib.request.urlopen(origin+'/web-assets/ui.css') as response:
                    before = response.read()
                    old_etag = response.headers['ETag']
                    assert response.headers['X-Content-Type-Options'] == 'nosniff'
                    assert response.headers.get('Last-Modified') is None
                    assert response.headers.get('Cache-Control') == 'no-cache'
                    assert response.headers.get('ETag')
                with urllib.request.urlopen(origin+'/') as response:
                    old_generation = response.headers['X-TuneTrail-UI-Generation']
                    assert response.headers['Cache-Control'] == 'private, no-cache'
                css = package/'assets/css/ui/base.css'
                original_css = css.read_bytes()
                css.write_text('changed after startup')
                with urllib.request.urlopen(origin+'/web-assets/ui.css') as response:
                    assert response.read() == before, 'mixed asset generation'
                css.write_bytes(original_css)
                for route in ['/web-assets/', '/web-assets/icons/LICENSE', '/web-assets/templates/public/live.html', '/web-assets/brand/icon.png']:
                    try:
                        urllib.request.urlopen(origin+route)
                    except urllib.error.HTTPError as error:
                        assert error.code == 404, route
                    else:
                        raise AssertionError('exposed '+route)
            finally:
                process.terminate()
                try:
                    process.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
        # A fresh process must reject an old URL instead of serving new bytes as immutable.
        css.write_bytes(original_css+b'\n/* fictional next release */\n')
        with (root/'restart.log').open('w') as log:
            process = subprocess.Popen([str(link)], cwd='/', env=env, stdout=log, stderr=log)
            try:
                for _ in range(100):
                    try:
                        with urllib.request.urlopen(origin+'/', timeout=1) as response:
                            assert response.headers['X-TuneTrail-UI-Generation'] != old_generation
                        break
                    except OSError:
                        time.sleep(.1)
                else:
                    raise AssertionError('restart readiness timeout')
                try:
                    urllib.request.urlopen(origin+'/web-assets/ui.css?v='+old_etag.strip('"'))
                except urllib.error.HTTPError as error:
                    assert error.code == 404 and error.headers['Cache-Control'] == 'no-store'
                else:
                    raise AssertionError('old hash served after restart')
                with urllib.request.urlopen(origin+'/web-assets/ui.css') as response:
                    assert response.headers['ETag'] != old_etag
                    assert b'fictional next release' in response.read()
            finally:
                process.terminate()
                try:
                    process.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
        css.write_bytes(original_css)
        # Every following failure runs in a fresh process and cannot reuse a cached snapshot.
        isolated = root/'isolated'
        isolated.mkdir()
        alone = isolated/'tunetrail'
        shutil.copy2(binary, alone)
        checks = [(alone, environment(root/'alone-data'))]
        override = environment(root/'invalid-root-data')
        override['TUNETRAIL_WEB_DIR'] = str(root/'absent-root')
        checks.append((target, override))
        for executable, settings in checks:
            result = subprocess.run([str(executable)], cwd='/', env=settings, capture_output=True, timeout=10)
            assert result.returncode != 0 and b'assets' in result.stderr
            assert not Path(settings['TUNETRAIL_CONFIG_DIR']).exists(), 'assets failure touched data directory'
        template = package/'assets/templates/public/live.html'
        original_template = template.read_bytes()
        template.write_text('{{invalid-template')
        result = subprocess.run([str(target)], cwd='/', env=environment(root/'bad-template-data'), capture_output=True, timeout=10)
        assert result.returncode != 0, 'invalid template accepted'
        template.write_bytes(original_template)
        css.write_text('@import "ui.css";')
        settings = environment(root/'bad-css-data')
        result = subprocess.run([str(target)], cwd='/', env=settings, capture_output=True, timeout=10)
        assert result.returncode != 0 and not Path(settings['TUNETRAIL_CONFIG_DIR']).exists()
        css.write_bytes(original_css)
        template.write_bytes(b'{{define "invalid-asset"}}{{if false}}{{asset "missing.css"}}{{end}}{{end}}'+original_template)
        result = subprocess.run([str(target)], cwd='/', env=environment(root/'bad-asset-data'), capture_output=True, timeout=10)
        assert result.returncode != 0 and b'unknown public asset' in result.stderr
    print('PASS external web distribution: adjacent root via symlink, cwd /, snapshot, exposure, missing/invalid roots and template/CSS/asset failure, hash rejection across restarts')


if __name__ == '__main__':
    main()
