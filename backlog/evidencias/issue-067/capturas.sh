#!/bin/bash
# Screenshot every reference HTML page of two trees at six widths and compare bytes.
S=/private/tmp/claude-501/-Users-evertonberger-Projects-TuneTrail/69aa52b5-96f9-4650-b1c9-bdea99cba976/scratchpad
C="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
BASE=$1; NEW=$2; OUT=$S/shots; rm -rf $OUT; mkdir -p $OUT/base $OUT/new
(cd $BASE && exec python3 -m http.server 8771 >/dev/null 2>&1) & P1=$!
(cd $NEW && exec python3 -m http.server 8772 >/dev/null 2>&1) & P2=$!
sleep 1
same=0; diff=0
for page in $(cd $BASE && find . -name '*.html' | sed 's#^\./##' | sort); do
  for w in 320 360 390 768 1024 1440; do
    name=$(echo "$page" | tr '/' '_')-$w.png
    for side in base new; do
      port=8771; [ $side = new ] && port=8772
      "$C" --headless=new --disable-gpu --hide-scrollbars --force-device-scale-factor=1 --user-data-dir=$S/chr-$side \
        --window-size=$w,2400 --virtual-time-budget=3000 --screenshot=$OUT/$side/$name "http://127.0.0.1:$port/$page" >/dev/null 2>&1
    done
    if cmp -s $OUT/base/$name $OUT/new/$name; then same=$((same+1)); else diff=$((diff+1)); echo "DIFF $name"; fi
  done
done
kill $P1 $P2
echo "same=$same diff=$diff"
