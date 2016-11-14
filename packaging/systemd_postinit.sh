#!/bin/sh
if service restatemachine status | grep "running" > /dev/null; then
	service restatemachine restart
else
	service restatemachine start
fi
