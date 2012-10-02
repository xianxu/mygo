#!/bin/sh

ssh -f -L 8001:smf1-adz-03-sr3:8000 smf1-adz-03-sr3 sleep 1h
ssh -f -L 8002:smf1-adj-27-sr4:8000 smf1-adj-27-sr4 sleep 1h
ssh -f -L 8003:smf1-afo-35-sr4:8000 smf1-afo-35-sr4 sleep 1h
ssh -f -L 8004:smf1-adz-19-sr2:8000 smf1-adz-19-sr2 sleep 1h
ssh -f -L 8005:smf1-adb-23-sr3:8000 smf1-adb-23-sr3 sleep 1h
ssh -f -L 8006:smf1-adz-27-sr1:8000 smf1-adz-27-sr1 sleep 1h
ssh -f -L 8007:smf1-afe-15-sr3:8000 smf1-afe-15-sr3 sleep 1h
ssh -f -L 8008:smf1-aer-19-sr4:8000 smf1-aer-19-sr4 sleep 1h
