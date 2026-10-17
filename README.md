# Analiza porównawcza narzutu mechanizmów bezpieczeństwa: REST, GraphQL, gRPC

Jedna aplikacja w Go (wspólna warstwa domenowa + PostgreSQL) wystawiona jako REST, GraphQL i gRPC.
Mechanizmy bezpieczeństwa włączane flagami (konfiguracje C0–C5): TLS 1.3, JWT (HS256), walidacja i limity rozmiaru,
rate limiting (token bucket per `sub`), limity złożoności.

## Wymagania
Go (go.mod wymaga 1.26; starszy Go pobierze toolchain), Docker + Compose v2, k6, protoc + `protoc-gen-go(-grpc)`, jq, openssl, Python 3.11+.

## Szybki start
```bash
make up          # PostgreSQL 16 (migracja z app/migrations)
make seed        # deterministyczne dane (ziarno 42): 10k users, 50 categories, 5k products, 50k orders, 200k items
make certs       # certyfikat samopodpisany (infra/certs)
make tokens      # tokeny JWT dla k6 (loadtests/tokens.json)
make test        # testy jednostkowe + test równoważności (wymaga działającej bazy)
make build       # obrazy Dockera dla trzech interfejsów

# jeden interfejs naraz, konfiguracja flagami:
TLS_ENABLED=true AUTH_ENABLED=true docker compose -f infra/docker-compose.yml --profile rest up -d app-rest
curl -sk -H "Authorization: Bearer $(jq -r '.legit[0]' loadtests/tokens.json)" https://localhost:8443/users/1
```
Porty: REST 8443, GraphQL 8444 (`/graphql`), gRPC 8445; metryki 9101/9102/9103.

## Podgląd na żywo (opcjonalny)
Grafana + Prometheus jako osobny profil `monitoring`, domyślnie wyłączony (obciąża tę samą maszynę co pomiar).
```bash
make monitoring                              # start: Grafana http://localhost:3000 (bez logowania), Prometheus :9090
make monitoring-down                         # stop
MONITORING=1 loadtests/run_experiment.sh ... # przebieg z podglądem (stos zostaje włączony po przebiegu)
MONITORING=1 loadtests/run_matrix.sh local seria_b
```
Dashboard „Eksperymenty: podgląd na żywo”: RPS i kody odpowiedzi, p50/p95/p99 po stronie serwera, CPU i RAM procesu
aplikacji, goroutines/sterta, restarty i dostępność. To tylko podgląd — dane do pracy pochodzą z k6 i CSV.
Flaga `monitoring` jest zapisywana w `*_meta.json`, więc w analizie widać, które przebiegi miały włączony podgląd.
cAdvisor pominięty (nie działa z Dockerem 29 i containerd snapshotterem), CPU/RAM biorą się z metryk `process_*`.

## Weryfikacja konfiguracji C0–C5
`infra/config_check.sh [rest|graphql|grpc]` uruchamia każdą konfigurację i porównuje zachowanie z macierzą z praktyczna.md §3.6.

## Eksperymenty
```bash
# pojedynczy przebieg: <env> <seria> <iface> <op> <cfg> <rep> <rate>
loadtests/run_experiment.sh local seria_a rest O1 C0 1 200
# cała seria (wznawia po przerwie, pomija wykonane); poziomy RPS w loadtests/rates.env (SAT_REST_O1=...)
loadtests/run_matrix.sh local seria_a1   # nasycenie C0
loadtests/run_matrix.sh local seria_a2   # obciążenie stałe 70%
loadtests/run_matrix.sh local seria_b | seria_c | seria_d
loadtests/check_complete.sh local seria_a
```
Wyniki: `results/<env>/<seria>/` (k6 JSON, CSV zasobów z `docker stats`, metadane, log k6).

## Analiza
```bash
pip install -r analysis/requirements.txt
python analysis/load.py local seria_a
python analysis/stats.py local seria_a "measure_p(95)"
```

## Struktura
`app/` (kod: `cmd/`, `internal/`, `api/`, `migrations/`), `infra/` (compose, certy, config_check), `loadtests/` (k6 i skrypty uruchomieniowe), `analysis/` (Python), `results/`.
