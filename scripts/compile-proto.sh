#!/bin/sh

protoc -I="./internal/work" --go_out="./internal" "./internal/work/work.proto"
