#!/usr/bin/env python3
"""Compare local graph endpoints to literal fictional listen instants."""
import argparse
import json
from datetime import datetime
from html.parser import HTMLParser
from urllib.request import urlopen

# Fixed by fixture v1, not derived from the application or its query code.
BASE = ["2018-11-04T02:59:00Z", "2018-11-04T03:01:00Z", "2019-02-17T01:30:00Z", "2019-02-17T02:30:00Z", "2024-05-01T00:00:00Z", "2024-05-01T00:00:01Z"]
ALPHA = {"track": BASE, "artist": BASE + ["2024-05-01T00:00:02Z"], "album": BASE + ["2024-05-01T00:00:02Z"]}
BETA = {kind: ["2024-05-01T00:00:00Z"] for kind in ("track", "artist", "album")}


def unix(value):
    """Interpret an RFC3339 timestamp as a UTC Unix second."""
    return int(datetime.fromisoformat(value.replace("Z", "+00:00")).timestamp())


class InterestTable(HTMLParser):
    """Extract each time interval and count from the rendered interest table."""
    def __init__(self):
        super().__init__(); self.inside=False; self.summary=False; self.name=""; self.row=None; self.cell=False; self.rows=[]
    def handle_starttag(self, tag, attrs):
        attrs=dict(attrs)
        if tag == 'details': self.inside=False; self.name=""
        if tag == 'summary': self.summary=True
        if self.inside and tag == 'tr': self.row=[]
        if self.inside and self.row is not None and tag == 'time': self.row.append(attrs.get('datetime'))
        if self.inside and self.row is not None and tag == 'td': self.cell=True
    def handle_endtag(self, tag):
        if tag == 'summary': self.summary=False; self.inside=self.name.strip()=='Interest values'
        if self.inside and tag == 'tr' and self.row and len(self.row)==3: self.rows.append(self.row); self.row=None
        if tag == 'td': self.cell=False
        if tag == 'details': self.inside=False
    def handle_data(self,data):
        if self.summary: self.name+=data
        if self.inside and self.row is not None and self.cell and len(self.row)==2 and data.strip().isdigit(): self.row.append(int(data.strip()))


def get(origin, suffix, json_response=True):
    """Read a URL only from an explicitly supplied loopback server."""
    with urlopen(origin+suffix, timeout=10) as response:
        data=response.read().decode()
        return json.loads(data) if json_response else data


def main():
    """Validate 16 buckets and historical activity for two fictional profiles."""
    parser=argparse.ArgumentParser(); parser.add_argument('--go',required=True);parser.add_argument('--react',required=True);args=parser.parse_args()
    import re
    for origin in (args.go,args.react):
        if not re.fullmatch(r'http://127\.0\.0\.1:\d+', origin): raise ValueError('loopback only')
    checked=0
    for origin in (args.go,args.react):
        for slug,entities in [('alpha',ALPHA),('beta',BETA)]:
            for kind,items in entities.items():
                buckets=get(origin,f'/apis/web/v1/profiles/{slug}/{kind}/1/interest?buckets=16')
                assert len(buckets)==16
                stamps=[unix(item) for item in items]
                start,end=min(stamps),unix(buckets[-1]['bucket_end'])
                assert end>=max(stamps)
                bounds=[start+((end-start)*i)//16 for i in range(17)]
                expected=[sum(left<=x and (x<right or (i==15 and x==right)) for x in stamps) for i,(left,right) in enumerate(zip(bounds,bounds[1:]))]
                assert [unix(b['bucket_start']) for b in buckets]==bounds[:-1]
                assert [unix(b['bucket_end']) for b in buckets]==bounds[1:]
                assert [b['listen_count'] for b in buckets]==expected,(slug,kind,expected)
                if origin==args.go:
                    table=InterestTable();table.feed(get(origin,f'/{slug}/{kind}/1?tz=America/Sao_Paulo',False))
                    assert len(table.rows)==16,(slug,kind,len(table.rows))
                    # Requests can cross a second: recompute from the HTML's own endpoint.
                    html_end=unix(table.rows[-1][1]); html_bounds=[start+(html_end-start)*i//16 for i in range(17)]
                    html_expected=[sum(left<=x and (x<right or (i==15 and x==right)) for x in stamps) for i,(left,right) in enumerate(zip(html_bounds,html_bounds[1:]))]
                    assert [unix(r[0]) for r in table.rows]==html_bounds[:-1]
                    assert [unix(r[1]) for r in table.rows]==html_bounds[1:]
                    assert [r[2] for r in table.rows]==html_expected
                checked+=16
        activity=get(origin,'/apis/web/v1/profiles/alpha/listen-activity?year=2018&month=11&tz=America%2FSao_Paulo')['activity']
        nonzero={x['start_time'][:10]:x['listens'] for x in activity if x['listens']}
        assert nonzero=={'2018-11-03':1,'2018-11-04':1},nonzero
    print(json.dumps({'api_buckets_checked':checked,'html_buckets_checked':96,'dst_activity':'2018-11-03=1; 2018-11-04=1','clocks':'server real; interval end taken from response'},ensure_ascii=False))


if __name__=='__main__': main()
