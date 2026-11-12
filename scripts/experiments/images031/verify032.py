#!/usr/bin/env python3
"""Compare issue 032 outputs with the independent 031 PNG oracle and baseline."""
import argparse
import json
import subprocess
from pathlib import Path


def main():
    """Check calibrated gates for the actual component and keep per-case evidence."""
    parser = argparse.ArgumentParser()
    parser.add_argument('--work-dir', type=Path, required=True)
    parser.add_argument('--outputs', type=Path, required=True)
    parser.add_argument('--report', type=Path, required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[3]
    data = json.loads((root / 'backlog/evidencias/transicao-imagens/results-031.json').read_text())
    results = []
    for i, row in enumerate(data['rows']):
        if row['mode'] != 'pure' or row['effort'] != -1 or 'quality' not in row or row.get('error'):
            continue
        name = f"{i:03}-pure-{Path(row['input']).stem}-{row['operation']}-e-1.webp"
        if row['operation'] == 'source':
            reference = args.work_dir / ('oracle-source-' + row['input'] + '.png')
        else:
            reference = args.work_dir / ('oracle-' + Path(name).stem + '.png')
        quality = json.loads(subprocess.check_output([str(args.work_dir / 'metrics'), str(reference), str(args.outputs / name)], text=True))
        baseline = next((x for x in data['rows'] if x['mode'] == 'vips' and x['input'] == row['input'] and x['operation'] == row['operation'] and x['size'] == row['size'] and 'quality' in x), None)
        if baseline is None:
            raise ValueError(f'missing baseline for row {i}')
        candidate_db = min(quality['psnr_black_db'], quality['psnr_white_db'])
        baseline_db = min(baseline['quality']['psnr_black_db'], baseline['quality']['psnr_white_db'])
        natural = 'natural' in row['input']
        tolerance = 1.5 if natural and row['operation'] == 'source' else 0.5
        absolute = (40 if row['operation'] == 'source' else 35) if natural else 0
        results.append({'row': i, 'input': row['input'], 'operation': row['operation'], 'quality': quality,
                        'baseline_db': baseline_db, 'relative_db': candidate_db - baseline_db,
                        'passed_calibrated': quality['dimension_match'] and candidate_db >= absolute and candidate_db >= baseline_db - tolerance})
    failures = [r for r in results if not r['passed_calibrated']]
    args.report.write_text(json.dumps({'cases': len(results), 'failures': failures, 'results': results}, indent=2) + '\n')
    print(f'{len(results)} cases; {len(failures)} calibrated quality failures')
    if failures or len(results) < 50:
        raise SystemExit(1)


if __name__ == '__main__':
    main()
