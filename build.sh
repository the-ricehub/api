#!/bin/bash

mkdir -p build
go build -o build/api ./src
echo "API has been compiled to ./build/api"