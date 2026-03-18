#!/usr/bin/env bash
set -euo pipefail

go build -o bin/downloader ./cmd/downloader
./bin/downloader "$@"
