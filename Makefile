SHELL := /bin/bash

.phony: install test

install:
	go get -u github.com/kardianos/govendor
	govendor -version || true
	govendor sync

test:
	govendor test +local -v -short
