.ONESHELL:
SHELL := /bin/bash

.PHONY: run
run:
	source .env
	go run ./cmd/beacon

.PHONY: deploy
deploy:
	go build -ldflags="-w -s" ./cmd/beacon
	ansible-playbook deploy/ansible/ubuntu.yml -i $(IP), -u $(USER)
