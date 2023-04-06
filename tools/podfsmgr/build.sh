#!/bin/bash

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/podfsmgr-linux-amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/podfsmgr-linux-arm64 .
