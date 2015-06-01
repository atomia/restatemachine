#!/bin/sh
if status restatemachine | grep "process " > /dev/null; then
	restart restatemachine
else
	start restatemachine
fi
