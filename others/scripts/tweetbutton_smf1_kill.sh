#!/bin/sh

ps auxwww | grep tfe_server | grep -v grep |  awk '{print $2}' | xargs kill
