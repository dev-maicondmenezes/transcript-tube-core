#!/usr/bin/env bash
set -euo pipefail

go version
node --version

for var in YOUTUBE_API_KEY DEEPSEEK_API_KEY; do
  if [[ -z "${!var:-}" ]]; then
    echo "AVISO: $var não está definida" >&2
  else
    echo "OK: $var definida"
  fi
done
