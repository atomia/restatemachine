#!/bin/sh

restatemachine_write_stderr() {
	echo "$1" 1>&2
}

restatemachine_transition_to() {
	restatemachine_write_stderr "$1"
}

restatemachine_transition_after_seconds() {
	restatemachine_write_stderr "$1"
}

restatemachine_status_message() {
	restatemachine_write_stderr "$1"
}
