#!/usr/bin/env bash

go build -mod=vendor -o dist/ci-firewall cmd/ci-firewall/ci-firewall.go
