.ONESHELL:
SHELL := /bin/bash

VERSION ?= $(shell git describe --tags --always --dirty)

.PHONY: run
run:
	source .env
	go run ./cmd/beacon

.PHONY: build
build:
	go build \
		-ldflags="-w -s -X main.Version=$(VERSION)" \
		-o beacon ./cmd/beacon

.PHONY: ansible-deploy
ansible-deploy: build
	ansible-playbook deploy/ansible/ubuntu.yml -i $(IP), -u $(USER)
