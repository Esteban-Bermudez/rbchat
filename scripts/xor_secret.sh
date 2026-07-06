#!/bin/sh
# XOR a secret with a fixed key and output the hex-encoded result.
# Used by CI to produce the obfuscated value for ldflags injection.
#
# Usage: RBCHAT_SECRET=secretvalue ./scripts/xor_secret.sh

XOR_KEY_HEX="fba2917c3d5e6f801234567890abcdef"

if [ -z "$RBCHAT_SECRET" ]; then
	exit 0
fi

python3 -c "
import sys
secret = '$RBCHAT_SECRET'.encode()
key = bytes.fromhex('$XOR_KEY_HEX')
result = bytes([secret[i] ^ key[i % len(key)] for i in range(len(secret))])
sys.stdout.write(result.hex())
"
