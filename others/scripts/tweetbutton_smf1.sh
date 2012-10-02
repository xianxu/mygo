#!/bin/sh

for i in {0..19}
do
  ./tfe_server -rules=tweetbutton-smf1 -port_offset=$i &
done
