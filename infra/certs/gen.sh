#!/usr/bin/env bash
# Certyfikat samopodpisany (SAN: localhost, 127.0.0.1). Klucz jest w .gitignore.
set -euo pipefail
cd "$(dirname "$0")"
openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
  -keyout server.key -out server.crt \
  -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
chmod 644 server.key
