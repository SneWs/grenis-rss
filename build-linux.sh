#!/bin/bash

go get main
env GOOS=linux GOARCH=amd64 go build -o grenis-rss
