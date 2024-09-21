#!/usr/bin/env bash

set -euo pipefail

if ! command -v gofumpt &> /dev/null; then
    echo "Error: gofumpt is not installed or not in your PATH" >&2
    exit 1
fi

output=$(gofumpt -l -w "$@")

if [[ -n "$output" ]]; then
    echo "$output"
fi

exit 0
