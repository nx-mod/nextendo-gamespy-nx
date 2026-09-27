#!/usr/bin/env sh
# nextendo-gamespy-nx — home-lab launch (NWFC/GameSpy NAS). TLS is terminated by the sni-router in front, so
# this serves plain HTTP and runs straight from the repo with no setup.
set -e
echo "[nextendo-gamespy-nx] starting on default ports (see main.go / README to override)"
exec go run .
