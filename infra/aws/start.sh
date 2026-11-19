#!/usr/bin/env bash
# start.sh — tworzy środowisko (terraform apply) z bieżącym publicznym IP w security group.
# TWORZY PŁATNE ZASOBY. Uruchamiać dopiero po akceptacji kosztów (README.md).
set -euo pipefail
cd "$(dirname "$0")"
: "${TF_VAR_owner_email:?ustaw TF_VAR_owner_email (adres na alarmy budżetu)}"
export TF_VAR_my_ip_cidr="$(curl -fsS https://checkip.amazonaws.com | tr -d '[:space:]')/32"
echo "Moje IP: $TF_VAR_my_ip_cidr"
terraform init -upgrade
terraform plan -out=tfplan
read -r -p "Utworzyć powyższe zasoby (płatne)? Wpisz 'tak': " ok
[ "$ok" = "tak" ] || { echo "przerwano"; exit 1; }
terraform apply tfplan
terraform output
echo "Czekaj ok. 3-5 min na cloud-init (plik /var/lib/cloud/thesis-ready na obu maszynach)."
