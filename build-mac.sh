#!/bin/bash

go get main
env GOOS=darwin GOARCH=arm64 go build -o grenis-rss-macos
