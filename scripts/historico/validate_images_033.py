#!/usr/bin/env python3
"""Compare legacy and Go derivatives in new fictional SQLite fixtures only."""
import argparse
import os
import shutil
import sqlite3
import subprocess
from pathlib import Path
from roadmap06_fixture import create
from validate_multiuser import Server


def main():
    """Start two isolated instances, capture the UI and verify database integrity."""
    parser=argparse.ArgumentParser()
    parser.add_argument('--binary',type=Path,required=True)
    parser.add_argument('--work-dir',type=Path,required=True)
    parser.add_argument('--legacy-source',type=Path,required=True)
    parser.add_argument('--legacy-worker',type=Path,required=True)
    parser.add_argument('--output',type=Path,required=True)
    args=parser.parse_args()
    create(args.work_dir)
    image_id='11111111-1111-4111-8111-111111111111'
    servers=[]
    try:
        for folder in ('go','react'):
            directory=args.work_dir/folder
            with sqlite3.connect(directory/'koito.db') as conn:
                conn.execute('UPDATE listening_profiles SET avatar_image_id=? WHERE id=1',(image_id,));conn.commit()
            cache=directory/'image_cache'/image_id[:2]/image_id
            shutil.copyfile(args.legacy_source,cache/'source.webp')
            if folder=='go':
                for n in (64,128,300,640,1000):
                    subprocess.run([str(args.legacy_worker),'-input',str(cache/'source.webp'),'-output',str(cache/f'{n}x{n}.webp'),'-size',str(n),'-workers','1','-reps','1'],check=True,stdout=subprocess.DEVNULL)
            server=Server(args.binary,directory);servers.append(server);server.start()
        env=dict(os.environ)
        subprocess.run(['node','scripts/capture_images_033.cjs',servers[0].origin,servers[1].origin,str(args.output)],env=env,check=True)
        for server in servers:
            with sqlite3.connect(server.directory/'koito.db') as conn:
                if conn.execute('PRAGMA integrity_check').fetchall()!=[('ok',)] or conn.execute('PRAGMA foreign_key_check').fetchall():raise AssertionError('fixture integrity')
        print('Both fictional SQLite fixtures retain integrity')
    finally:
        for server in servers:server.stop()


if __name__=='__main__':
    main()
