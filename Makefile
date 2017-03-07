SHELL := /bin/bash

install:
	go get -u github.com/kardianos/govendor
	govendor -version || true
	govendor sync
.PHONY: install

test:
	govendor test -v +local
.PHONY: test

test_short:
	govendor test -v -short +local
.PHONY: test_short
