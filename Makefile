.ONESHELL:
SHELL := /bin/bash

.PHONY: run
run:
	source .env
	go run ./cmd/beacon

.PHONY: deploy
deploy:
	go GOOS=linux GOARCH=amd64 go build -ldflags="-w -s -X 'main.Environment=prod'" ./cmd/beacon
	source .env.prod
	ansible-playbook -i deploy/ansible/inventory.yml deploy/ansible/deploy.yml -u root
