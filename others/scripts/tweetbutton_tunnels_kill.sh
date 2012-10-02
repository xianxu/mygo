#!/bin/sh

ps auxwww | grep ssh | grep 800 |  awk '{print $2}' | xargs kill
