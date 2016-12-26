SHELL := /bin/bash

.phony: install test

install:
	go get -u github.com/kardianos/govendor
	govendor -version || true
	govendor sync

test:
	govendor test -v +local

test_short:
	govendor test -v -short +local
