#!/usr/bin/env bash
# cloud-init dla obu ról (role=${role}): Docker, Go, k6, narzędzia, klon repozytorium.
# Wersje przypięte tak jak lokalnie (results/meta/env_local.txt; go.mod wymaga Go 1.26).
set -euxo pipefail
export DEBIAN_FRONTEND=noninteractive
GO_VERSION=1.26.0
K6_VERSION=v2.3.0

apt-get update
apt-get install -y ca-certificates curl git jq make sysstat python3-venv postgresql-client tmux chrony

# Docker z oficjalnego repozytorium
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu noble stable" > /etc/apt/sources.list.d/docker.list
apt-get update
apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
usermod -aG docker ubuntu

# Go
curl -fsSL "https://go.dev/dl/go$${GO_VERSION}.linux-amd64.tar.gz" | tar -C /usr/local -xz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' > /etc/profile.d/go.sh

# k6 (potrzebny na generatorze; na serwerze nie szkodzi, przydaje się do testu dymnego)
curl -fsSL "https://github.com/grafana/k6/releases/download/$${K6_VERSION}/k6-$${K6_VERSION}-linux-amd64.tar.gz" \
  | tar -xz --strip-components=1 -C /usr/local/bin "k6-$${K6_VERSION}-linux-amd64/k6"

# stabilność pomiaru: governor wydajności, jeśli dostępny (na c7i zwykle zarządza hiperwizor)
for g in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do [ -w "$g" ] && echo performance > "$g" || true; done

sudo -u ubuntu git clone "${repo_url}" /home/ubuntu/masters-thesis
sudo -u ubuntu git -C /home/ubuntu/masters-thesis checkout "${repo_ref}"
echo "${role}" > /etc/thesis-role
touch /var/lib/cloud/thesis-ready
