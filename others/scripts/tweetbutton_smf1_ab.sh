#!/bin/sh

for i in {0..1}
do
  port="$((i + 8888))"
  ab -n 1000000 -c 5 -r -k http://smf1-aea-35-sr2:${port}/1/urls/count.json?url=11
done
