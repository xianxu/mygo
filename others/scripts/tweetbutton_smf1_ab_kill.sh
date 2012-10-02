#!/bin/sh

ps auxwww | grep ab | grep smf1-aea-35-sr2 | grep -v grep |  awk '{print $2}' | xargs kill
