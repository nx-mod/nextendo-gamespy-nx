#!/usr/bin/env sh
# Example: a NAS (Nintendo Wi-Fi Connection) login. NAS form values are base64
# with '=' replaced by '*'; encode a few fields and post them to /ac.
set -e
BASE="${1:-http://localhost:8475}"
enc(){ printf '%s' "$1" | base64 | tr '=' '*'; }
BODY="action=$(enc login)&gamecd=$(enc ADAE)&userid=$(enc 1234567890123)"
echo "== POST /ac (login) =="
curl -s -X POST "$BASE/ac" -d "$BODY" ; echo
echo "(returncd=001 + a token/challenge means the NAS auth succeeded)"
