#!/bin/bash
mkdir -p .tmp
docker run --rm -v "$PWD":/usr/src/myapp -w /usr/src/myapp golang:1.8.6-alpine go build -v main.go -o main
mv main .tmp/
docker build . -t registry.cn-hangzhou.aliyuncs.com/dmcloudv1/udp-proxy:latest