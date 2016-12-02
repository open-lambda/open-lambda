#!/bin/bash
protoc --go_out=plugins=grpc:. registry.proto
