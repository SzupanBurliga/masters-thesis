# Raport stanu repozytorium

Data: 2026-11-17. Commit: `8934261` (gałąź `main`, drzewo czyste). Raport powstał wyłącznie z czytania plików i uruchomienia poleceń, które niczego nie zmieniają (`go build`, `go vet`, `go test`, `git log`, `ls`, `jq`, `grep`). Kod nie był modyfikowany.

---

## 1. ŚRODOWISKO

### 1.1 Wersje narzędzi (stan na 2026-11-17)

| Polecenie | Wynik |
|---|---|
| `go version` | go version go1.25.7 linux/amd64 |
| `docker --version` | Docker version 29.1.3, build 29.1.3-0ubuntu3~24.04.2 |
| `docker compose version` | Docker Compose version 2.40.3+ds1-0ubuntu1~24.04.1 |
| `k6 version` | k6 v2.3.0 (commit/e088784614, go1.26.8, linux/amd64) |
| `ghz --version` | `dev` (zainstalowana wersja nie podaje numeru) |
| `python3 --version` | Python 3.12.3 |

Uwaga: `app/go.mod:3` wymaga `go 1.26.0`, a lokalnie jest 1.25.7. Kompilacja przechodzi, bo Go sam pobiera właściwy toolchain. Obraz Dockera używa `golang:1.26` (wg `LOG.md`).

### 1.2 `results/meta/env_local.txt`

```
# Środowisko lokalne, 2026-10-10T18:21:37+02:00
## System
Description:	Ubuntu 24.04.4 LTS
6.18.40.1-microsoft-standard-WSL2
## Sprzęt
CPU(s):                                  8
Model name:                              11th Gen Intel(R) Core(TM) i5-1145G7 @ 2.60GHz
Thread(s) per core:                      2
Core(s) per socket:                      4
               total        used        free      shared  buff/cache   available
Mem:            15Gi       1.4Gi        12Gi       4.4Mi       2.1Gi        13Gi
## Narzędzia
go version go1.25.7 linux/amd64
Docker version 29.1.3, build 29.1.3-0ubuntu3~24.04.2
Docker Compose version 2.40.3+ds1-0ubuntu1~24.04.1
k6 v2.3.0 (commit/e088784614, go1.26.8, linux/amd64)
dev
libprotoc 3.21.12
protoc-gen-go v1.36.12
protoc-gen-go-grpc 1.6.2
Python 3.12.3
psql (PostgreSQL) 16.15 (Ubuntu 16.15-0ubuntu0.24.04.1)
jq-1.7
OpenSSL 3.0.13 30 Jan 2024 (Library: OpenSSL 3.0.13 30 Jan 2024)
```

Rozbieżność: plik podaje jądro `6.18.40.1`, obecnie działa `6.18.58.1-microsoft-standard-WSL2`, czyli jądro WSL zostało zaktualizowane po 10 X. Pomiary A1 (20–21 X) nie mają zapisanej wersji jądra, więc nie wiem, na którym jądrze się odbyły.

---

## 2. STRUKTURA I HISTORIA

### 2.1 Drzewo katalogów (3 poziomy, bez `.git`, `vendor`, `node_modules`, `__pycache__`)

```
.
├── .gitignore
├── LOG.md
├── Makefile
├── README.md
├── analysis/
│   ├── load.py
│   ├── requirements.txt
│   ├── saturation.py
│   └── stats.py
├── app/
│   ├── .dockerignore
│   ├── Dockerfile
│   ├── api/
│   │   ├── schema.graphql
│   │   └── service.proto
│   ├── cmd/
│   │   ├── configcheck/
│   │   ├── graphql/
│   │   ├── grpc/
│   │   ├── rest/
│   │   ├── seed/
│   │   └── tokens/
│   ├── go.mod
│   ├── go.sum
│   ├── gqlgen.yml
│   ├── internal/
│   │   ├── config/
│   │   ├── domain/
│   │   ├── equiv/
│   │   ├── graphql/
│   │   ├── grpcsrv/
│   │   ├── metrics/
│   │   ├── rest/
│   │   ├── security/
│   │   ├── serve/
│   │   └── store/
│   ├── migrations/
│   │   └── 001_init.sql
│   └── tools.go
├── bin/
│   ├── configcheck
│   └── seed
├── infra/
│   ├── aws/
│   ├── certs/
│   │   ├── gen.sh
│   │   ├── server.crt
│   │   └── server.key
│   ├── config_check.sh
│   ├── docker-compose.yml
│   ├── grafana/
│   │   ├── dashboards/
│   │   └── provisioning/
│   └── prometheus.yml
├── loadtests/
│   ├── check_complete.sh
│   ├── export_resources.sh
│   ├── lib/
│   │   ├── common.js
│   │   ├── ops.js
│   │   └── service.proto
│   ├── rates.env
│   ├── run_a1_calibration.sh
│   ├── run_experiment.sh
│   ├── run_matrix.sh
│   ├── seria_a.js
│   ├── seria_b.js
│   ├── seria_c.js
│   ├── seria_d.js
│   ├── tokens.json
│   └── wait_ready.sh
├── notatki/
│   ├── cli.md
│   ├── fix.md
│   ├── plan.md
│   ├── praktyczna.md
│   ├── stan_projektu.md
│   └── tu będą notatki.txt
├── results/
│   ├── aws/
│   │   ├── seria_a/
│   │   ├── seria_b/
│   │   ├── seria_c/
│   │   └── seria_d/
│   ├── local/
│   │   ├── seria_a/
│   │   ├── seria_a_A1_kalibracja.md
│   │   ├── seria_a_saturation_O1_ramp6k_summary.csv
│   │   ├── seria_a_saturation_O1_ramp6k_windows.csv
│   │   ├── seria_a_saturation_O1_ramp_summary.csv
│   │   ├── seria_a_saturation_O1_ramp_windows.csv
│   │   ├── seria_a_saturation_O3_ramp_summary.csv
│   │   ├── seria_a_saturation_O3_ramp_windows.csv
│   │   ├── seria_b/
│   │   ├── seria_c/
│   │   └── seria_d/
│   ├── local_db3cpu/
│   │   ├── config.txt
│   │   ├── seria_a/
│   │   ├── seria_a_A1_kalibracja.md
│   │   ├── seria_a_saturation_O1_ramp_summary.csv
│   │   ├── seria_a_saturation_O1_ramp_windows.csv
│   │   ├── seria_a_saturation_O3_ramp_summary.csv
│   │   └── seria_a_saturation_O3_ramp_windows.csv
│   └── meta/
│       └── env_local.txt
└── terminologia.md
```

Katalog `docs/` nie istniał przed tym raportem. Katalogi `infra/aws/`, `results/aws/seria_{a,b,c,d}/`, `results/local/seria_{b,c,d}/` są puste.

### 2.2 Ostatnie commity (`git log --oneline -20`)

```
8934261 feat: kalibracja A1 (C0), punkty nasycenia i spójna macierz przebiegów
d22c888 feat: Add Grafana provisioning for dashboards and Prometheus datasource
b2df1af Initial commit
```

W repozytorium są tylko 3 commity. Tagi: `v0.1-baseline` i `v0.2-security`. Według `LOG.md` (wpis z 2 XI) oba dodano wstecznie i wskazują `8934261`. W git śledzonych jest 74 pliki. Surowe wyniki (`results/local/**`, `results/local_*/**` poza podsumowaniami) są w `.gitignore`, więc istnieją tylko lokalnie.

### 2.3 Kompilacja i testy

| Polecenie (w `app/`) | Wynik |
|---|---|
| `go build ./...` | OK, kod 0, brak komunikatów |
| `go vet ./...` | OK, brak ostrzeżeń |
| `go test ./...` | OK: `internal/equiv`, `internal/graphql`, `internal/security`; pozostałe 16 pakietów nie ma testów |

Zastrzeżenie: testy równoważności w `internal/equiv` (`TestEquivO3`, `TestEquivO1O2`, `TestEquivNotFound`) zostały **pominięte (SKIP)**, bo nie działała baza (`equiv_test.go:64`, `t.Skipf("brak bazy: ...")`). Wynik `ok` dla tego pakietu nie potwierdza więc równoważności. Faktycznie wykonały się tylko testy jednostkowe: `TestJWT`, `TestMaxBody`, `TestRateLimit`, `TestCheckJSON` (`internal/security/security_test.go`) oraz `TestLimits`, `TestCalibration` (`internal/graphql/complexity_test.go`).

---

## 3. STAN IMPLEMENTACJI

Ocena tylko na podstawie kodu. Ścieżki względem katalogu głównego repozytorium.

| Element | Status | Gdzie w kodzie (plik:linia) | Uwagi |
|---|---|---|---|
| Schemat bazy | zrobione | `app/migrations/001_init.sql:1-38` | 5 tabel (users, categories, products, orders, order_items) i 4 indeksy. Ładowany przez `docker-entrypoint-initdb.d` (`infra/docker-compose.yml`). |
| Dane testowe (seed) | zrobione | `app/cmd/seed/main.go:15-21`, `:40` | Deterministyczny (ziarno 42), `CopyFrom`: 10 000 użytkowników, 50 kategorii, 5000 produktów, 50 000 zamówień, 200 000 pozycji. Na końcu wypisuje md5 zamówień, który zapisuje się w `*_seed.txt` każdego przebiegu. |
| O1 (użytkownik) | zrobione | store `app/internal/store/store.go:48`; REST `app/internal/rest/rest.go:57,101`; GraphQL `app/internal/graphql/schema.resolvers.go:66`; gRPC `app/internal/grpcsrv/server.go:50` | `GET /users/{id}`, `user(id)`, `Shop/GetUser`. |
| O2 (lista zamówień) | zrobione | store `store.go:57`; REST `rest.go:59,138`; GraphQL `schema.resolvers.go:77`; gRPC `server.go:58` | Filtr `status`, `limit`, `offset`. W seriach A–D nie jest mierzona: `run_matrix.sh` używa tylko O1 i O3. |
| O3 (użytkownik→zamówienia→pozycje→produkt) | zrobione | store `store.go:86-118` (3 zapytania: user, orders, items+JOIN produktu); REST `rest.go:58,115`; GraphQL `schema.resolvers.go:103` + dataloadery; gRPC `server.go:70` | REST przyjmuje `?expand=items.product`, ale niezależnie od parametru zawsze zwraca pełną strukturę. Parametr jest sprawdzany tylko w C5 (`rest.go:126`). |
| O4 (nowe zamówienie) | zrobione | store `store.go:217`; REST `rest.go:60,161`; GraphQL `schema.resolvers.go:33`; gRPC `server.go:82` | Używana jako nośnik ataków w seriach C i D. Jako osobna operacja legalnego ruchu nie występuje w `run_matrix.sh`. |
| Interfejs REST | zrobione | `app/internal/rest/rest.go:38-62`, `app/cmd/rest/main.go` | chi v5, port 8443. |
| Interfejs GraphQL | zrobione | `app/cmd/graphql/main.go`, `app/api/schema.graphql`, `app/internal/graphql/` | gqlgen, `handler.NewDefaultServer` (introspekcja i APQ włączone), port 8444. |
| Dataloader w GraphQL | **jest** | `app/internal/graphql/loaders.go:19` (typ `Loaders`), `:28` (`newLoaders`), `:87` (`LoaderMiddleware`) | `dataloadgen`, osobne loadery na żądanie dla Orders, Items, Categories i Users; okno 1 ms (`loaders.go:30`). Brak loadera produktów: produkt dochodzi JOIN-em w `ItemsByOrders`. Stałej liczby zapytań SQL nic w kodzie nie weryfikuje. |
| Interfejs gRPC | zrobione | `app/api/service.proto`, `app/internal/grpcsrv/server.go`, `app/cmd/grpc/main.go` | Port 8445, health service i reflection zarejestrowane. |
| Test równoważności odpowiedzi | częściowo | `app/internal/equiv/equiv_test.go:149` (O3), `:241` (O1, O2), `:294` (not found) | Obejmuje O1, O2, O3 i błąd 404. **Brak O4.** Wymaga uruchomionej bazy i serwerów, bez nich test jest pomijany (patrz 2.3). |
| TLS: REST | zrobione | `app/internal/serve/serve.go:15-25` | TLS 1.3 (`:18`), domyślnie HTTP/1.1 także po TLS (`HTTP2_ENABLED=false`). |
| TLS: GraphQL | zrobione | jak REST (`serve.HTTP`) | to samo. |
| TLS: gRPC | zrobione | `app/cmd/grpc/main.go:31-37` | `credentials.NewServerTLSFromFile`. Minimalnej wersji TLS kod nie ustawia jawnie (domyślna z biblioteki). |
| JWT: REST | zrobione | `app/internal/security/http.go:37`, `jwt.go:22`; podpięcie `rest.go:47-49` | HS256, wymagane `exp` i `sub`. `/healthz` pominięte. |
| JWT: GraphQL | zrobione | `app/cmd/graphql/main.go:42-44` | ten sam middleware. |
| JWT: gRPC | zrobione | `app/internal/security/grpc.go:15`; podpięcie `server.go:104-106` | interceptor unarny. |
| Walidacja + limit rozmiaru: REST | zrobione | `http.go:23` (MaxBody); `rest.go:53-55`, `:93` (limit), `:103`, `:149`, `:183` (validator) | Włączane `VALIDATION_ENABLED` (C3). Domyślnie 64 KiB. |
| Walidacja + limit rozmiaru: GraphQL | zrobione | `app/cmd/graphql/main.go:36-38` (MaxBody); `schema.resolvers.go:67,78,104` (`r.Validate`) | Walidacja zakresów w resolverach; typy pilnuje schemat. |
| Walidacja + limit rozmiaru: gRPC | zrobione | `app/internal/grpcsrv/validate.go:25`; `server.go:114-118` (`MaxRecvMsgSize`) | Limit `limit=0` w gRPC jest akceptowany (różnica opisana w `validate.go:15-16`). |
| Rate limiting: REST | zrobione | `http.go:58-96`; `rest.go:50-52` | Token bucket na `sub` z JWT, a bez JWT na IP. Domyślnie 100 RPS / burst 200. Mapa limiterów nie jest czyszczona (`http.go:57`, komentarz). |
| Rate limiting: GraphQL | zrobione | `app/cmd/graphql/main.go:39-41` | ten sam middleware. |
| Rate limiting: gRPC | zrobione | `grpc.go:30`; `server.go:107-109` | klucz `sub` albo IP rozmówcy. |
| Limity złożoności: REST | zrobione | `app/internal/security/jsoncheck.go:14`; `rest.go:172-177` (O4), `rest.go:126` (`expand`) | Głębokość JSON (domyślnie 8) i długość tablic (domyślnie 100). Działa tylko na `POST /orders`. |
| Limity złożoności: GraphQL | zrobione | `app/internal/graphql/complexity.go:17-45`; podpięcie `app/cmd/graphql/main.go:30-32` | Analiza AST przed resolverami: głębokość (6), koszt (400), aliasy (10). Odrzucenie zwraca kod 422 (`complexity.go:47-52`). |
| Limity złożoności: gRPC | częściowo | `validate.go:47-49` | Jest tylko długość `repeated items` w CreateOrder. Brak limitu głębokości zagnieżdżenia wiadomości (tak też w `notatki/fix.md`, p. 4). |
| Przełączanie C0–C5 | zrobione | `app/internal/config/config.go:38-64` (zmienne środowiskowe); `loadtests/run_experiment.sh:16-25` (mapowanie C→flagi); `infra/docker-compose.yml` (przekazanie zmiennych); `infra/config_check.sh:9-21` (weryfikacja) | Zmienne środowiskowe `TLS_ENABLED`, `AUTH_ENABLED`, `VALIDATION_ENABLED`, `RATELIMIT_ENABLED`, `COMPLEXITY_ENABLED` (+ parametry `RATELIMIT_RPS`, `MAX_DEPTH`, `MAX_COST` itd.). Konfiguracje są kumulatywne: C1=TLS, C2=+JWT, C3=+walidacja, C4=+rate limit, C5=+złożoność. Ani flagi CLI, ani osobne profile compose nie są do tego używane. |
| docker-compose | zrobione | `infra/docker-compose.yml` | Usługi: `postgres` (bez profilu, `PG_CPUS` domyślnie 3, 2 GB), `app-rest`/`app-graphql`/`app-grpc` (profile `rest`, `graphql`, `grpc`; 2 CPU, 2 GB każda), `prometheus` i `grafana` (profil `monitoring`). |
| Makefile | zrobione | `Makefile` | Cele: `up`, `down`, `build`, `seed`, `test`, `proto`, `gql`, `certs`, `tokens`, `fmt`, `psql`, `monitoring`, `monitoring-down`. Celów do uruchamiania serii pomiarowych nie ma, to robią skrypty w `loadtests/`. |
| Skrypty k6: seria A | zrobione | `loadtests/seria_a.js` | Tryb `ramp` (A1) i `const` (A2: warmup, measure, cooldown). A1 wykonana; A2 nie uruchomiona ani razu. |
| Skrypty k6: seria B | zrobione (nieprzetestowane) | `loadtests/seria_b.js` | Ruch legalny i napastnik (jedna tożsamość). Wyników brak. |
| Skrypty k6: seria C | zrobione (nieprzetestowane) | `loadtests/seria_c.js`, generatory `lib/common.js:91-127` | Duże ładunki 10 KB–5 MB + tło O1. Wyników brak. |
| Skrypty k6: seria D | zrobione (nieprzetestowane) | `loadtests/seria_d.js`, generatory `lib/common.js:60-89` | GraphQL depth/alias/limit, REST json/array, gRPC array. Wyników brak. |
| Skrypty ghz | brak | — | W repozytorium nie ma skryptu ghz. Jest tylko komenda w `notatki/cli.md:68-69`, według `notatki/fix.md` nietestowana. gRPC jest obciążany przez `k6/net/grpc`. |
| Orkiestracja przebiegów | zrobione | `loadtests/run_experiment.sh`, `run_matrix.sh`, `run_a1_calibration.sh`, `check_complete.sh`, `wait_ready.sh` | `run_matrix.sh` pomija przebiegi z istniejącym wynikiem, więc można go wznawiać. |
| Zbieranie CPU/RAM | zrobione | `loadtests/export_resources.sh:6-19` | Pętla `docker stats --no-stream` (~co 1 s) do CSV: `ts,cpu_perc,mem_used,mem_limit`. A1 zbiera dodatkowo CPU/RAM Postgresa i `vmstat` hosta (`run_a1_calibration.sh:22-23`). |
| Metryki Prometheus | zrobione | `app/internal/metrics/metrics.go`, `infra/prometheus.yml`, `infra/grafana/` | Tylko podgląd na żywo, domyślnie wyłączony (`MONITORING=0`). |
| Analiza w Pythonie | częściowo | `analysis/load.py` (wczytywanie, okna, `recovery_time`), `analysis/stats.py` (bootstrap CI, narzut, Mann-Whitney, CV), `analysis/saturation.py` (punkt nasycenia A1) | Użyty był tylko `saturation.py`. `load.py`/`stats.py` nie dostały jeszcze żadnych danych z A2–D. Brak kodu generującego wykresy (matplotlib jest w `requirements.txt`, ale nic go nie importuje). |
| Kontrakt REST (OpenAPI) | brak | — | Brak `app/api/openapi.yaml` (odnotowane też w `notatki/fix.md`). |

---

## 4. FRAGMENTY KODU DO PREZENTACJI

Skopiowane dosłownie z repozytorium.

### 4.1 Middleware zabezpieczeń w REST, `app/internal/rest/rest.go:47-55`

```go
	if cfg.Auth {
		r.Use(security.Auth(security.NewJWT(cfg.JWTSecret), "/healthz"))
	}
	if cfg.RateLimit {
		r.Use(security.RateLimit(security.NewLimiters(cfg.RateLimitRPS, cfg.RateLimitBurst), "/healthz"))
	}
	if cfg.Validation {
		r.Use(security.MaxBody(cfg.MaxBodyBytes))
	}
```

### 4.2 Limit głębokości, aliasów i kosztu w GraphQL, `app/internal/graphql/complexity.go:33-42`

```go
	aliases := 0
	depth := l.depth(oc.Operation.SelectionSet, oc.Doc.Fragments, 0, &aliases)
	if depth > l.MaxDepth {
		return rejected("query depth %d exceeds limit %d", depth, l.MaxDepth)
	}
	if aliases > l.MaxAliases {
		return rejected("alias count %d exceeds limit %d", aliases, l.MaxAliases)
	}
	if cost := l.cost(oc.Operation.SelectionSet, oc.Doc.Fragments, oc.Variables, 0); cost > l.MaxCost {
		return rejected("query cost %d exceeds limit %d", cost, l.MaxCost)
```

### 4.3 Łańcuch interceptorów w gRPC, `app/internal/grpcsrv/server.go:102-111`

```go
func Options(cfg config.Config) []grpc.ServerOption {
	chain := []grpc.UnaryServerInterceptor{metricsInterceptor}
	if cfg.Auth {
		chain = append(chain, security.AuthInterceptor(security.NewJWT(cfg.JWTSecret)))
	}
	if cfg.RateLimit {
		chain = append(chain, security.RateLimitInterceptor(security.NewLimiters(cfg.RateLimitRPS, cfg.RateLimitBurst)))
	}
	if cfg.Validation || cfg.Complexity {
		chain = append(chain, security.ValidationInterceptor(Validator(cfg.Validation, cfg.Complexity, cfg.MaxArrayLen)))
```

### 4.4 Konfiguracje C0–C5, `loadtests/run_experiment.sh:16-25`

```bash
case "$CFG" in
  C0) FL=(false false false false false) ;;
  C1) FL=(true  false false false false) ;;
  C2) FL=(true  true  false false false) ;;
  C3) FL=(true  true  true  false false) ;;
  C4) FL=(true  true  true  true  false) ;;
  C5) FL=(true  true  true  true  true)  ;;
  *) echo "nieznana konfiguracja $CFG" >&2; exit 2 ;;
esac
export TLS_ENABLED=${FL[0]} AUTH_ENABLED=${FL[1]} VALIDATION_ENABLED=${FL[2]} RATELIMIT_ENABLED=${FL[3]} COMPLEXITY_ENABLED=${FL[4]}
```

### 4.5 Scenariusz k6: ruch legalny i napastnik, `loadtests/seria_b.js:14-23`

```js
    legit: {
      executor: "constant-arrival-rate", exec: "legit",
      rate: LEGIT_RATE, timeUnit: "1s", duration: `${LEGIT_ONLY + ATTACK_DUR + RECOVER}s`,
      preAllocatedVUs: 100, maxVUs: 1000, tags: { who: "legit" },
    },
    attacker: {
      executor: "constant-arrival-rate", exec: "attacker",
      rate: ATTACK_RATE, timeUnit: "1s", duration: `${ATTACK_DUR}s`, startTime: `${LEGIT_ONLY}s`,
      preAllocatedVUs: 300, maxVUs: 5000, tags: { who: "attacker" },
    },
```

---

## 5. WYNIKI POMIARÓW

### 5.1 Co istnieje

- W `results/` jest 272 pliki, łącznie 1 738 612 911 B (`du`: `results/local` 468 MB, `results/local_db3cpu` 1,2 GB, `results/aws` i `results/meta` prawie puste).
- Wszystkie przebiegi to **seria A, etap A1 (kalibracja nasycenia, rampa), konfiguracja C0, operacje O1 i O3**. Nie ma ani jednego przebiegu A2, B, C, D, ani żadnej konfiguracji poza C0.
- `results/local/`: pierwsza próba z 20 X, w `LOG.md` i `results/local/seria_a_A1_kalibracja.md` opisana jako **odrzucona**.
- `results/local_db3cpu/`: druga próba z 21 X (Postgres 3 CPU), **przyjęta**, z niej pochodzą wartości w `loadtests/rates.env`.
- Wyników ghz nie ma. Poza `results/` nie ma żadnych plików z wynikami k6, ghz ani docker stats.
- **Wykresów (`*.png`, `*.svg`) w repozytorium nie ma.**

### 5.2 Wartości z plików k6 (`--summary-export`, `<przebieg>.json`)

Sposób odczytu: RPS = `metrics.iterations.rate` (1 iteracja = 1 żądanie). p50/p95/p99 = `http_req_duration` (REST, GraphQL) lub `grpc_req_duration` (gRPC), pola `med`, `p(95)`, `p(99)`, w ms. Błędy HTTP = `http_req_failed.value`. Poprawne = `checks.value`. `dropped_iterations.count` to iteracje, których k6 nie zdołał wysłać. Liczby zaokrąglono do 1–2 miejsc po przecinku, niczego nie przeliczano.

**Ważne przy interpretacji:** to przebiegi typu rampa (10 → 8000 lub 4000 RPS). RPS to średnia z całej rampy, a percentyle są liczone z całego przebiegu łącznie z fazą przeciążenia. Nie są to wartości w punkcie nasycenia ani wartości przy stałym obciążeniu. Punkty nasycenia są w tabeli 5.3.

| środowisko | iface | op_tag | C | rep | RPS (śr. z rampy) | p50 [ms] | p95 [ms] | p99 [ms] | http_req_failed | checks (odsetek poprawnych) | dropped_iterations |
|---|---|---|---|---|---|---|---|---|---|---|---|
| local_db3cpu | graphql | O1_ramp | C0 | 1 | 2961.3 | 629.08 | 882.48 | 1010.05 | 0 | 1 (0 nieudanych) | 782570 |
| local_db3cpu | graphql | O1_ramp | C0 | 2 | 3057.9 | 604.99 | 859.94 | 1013.7 | 0 | 1 (0 nieudanych) | 739945 |
| local_db3cpu | graphql | O1_ramp | C0 | 3 | 3073.2 | 605.49 | 843.83 | 992.07 | 0 | 1 (0 nieudanych) | 725226 |
| local_db3cpu | graphql | O3_ramp | C0 | 1 | 966.5 | 2571.89 | 3068.97 | 3513.55 | 0 | 1 (0 nieudanych) | 380835 |
| local_db3cpu | graphql | O3_ramp | C0 | 2 | 967.3 | 2544.76 | 3029.85 | 3351.13 | 0 | 1 (0 nieudanych) | 376495 |
| local_db3cpu | graphql | O3_ramp | C0 | 3 | 935.2 | 2592.38 | 3681.77 | 4253.64 | 0 | 1 (0 nieudanych) | 392409 |
| local_db3cpu | grpc | O1_ramp | C0 | 1 | 3355.7 | 267 | 696.97 | 844.45 | n/d (gRPC) | 1 (0 nieudanych) | 515983 |
| local_db3cpu | grpc | O1_ramp | C0 | 2 | 3485.3 | 136.4 | 652.75 | 765.86 | n/d (gRPC) | 1 (0 nieudanych) | 438801 |
| local_db3cpu | grpc | O1_ramp | C0 | 3 | 3476.7 | 171.02 | 662.36 | 767.34 | n/d (gRPC) | 1 (0 nieudanych) | 442999 |
| local_db3cpu | grpc | O3_ramp | C0 | 1 | 1331.2 | 1526.87 | 2120.35 | 2440.67 | n/d (gRPC) | 1 (0 nieudanych) | 252080 |
| local_db3cpu | grpc | O3_ramp | C0 | 2 | 1326.8 | 1574.06 | 2084.78 | 2343.38 | n/d (gRPC) | 1 (0 nieudanych) | 253065 |
| local_db3cpu | grpc | O3_ramp | C0 | 3 | 1162.5 | 1336.95 | 3513.04 | 5397.02 | n/d (gRPC) | 1 (0 nieudanych) | 308707 |
| local_db3cpu | rest | O1_ramp | C0 | 1 | 3991.2 | 2.18 | 114.02 | 180.75 | 0 | 1 (0 nieudanych) | 59892 |
| local_db3cpu | rest | O1_ramp | C0 | 2 | 4017.5 | 2.19 | 106.53 | 198.87 | 0 | 1 (0 nieudanych) | 63737 |
| local_db3cpu | rest | O1_ramp | C0 | 3 | 4010 | 2.2 | 107.1 | 177.69 | 0 | 1 (0 nieudanych) | 64796 |
| local_db3cpu | rest | O3_ramp | C0 | 1 | 1680.9 | 121.73 | 1478.43 | 1644.27 | 0 | 1 (0 nieudanych) | 130152 |
| local_db3cpu | rest | O3_ramp | C0 | 2 | 1682.4 | 187.13 | 1393.09 | 1656.87 | 0 | 1 (0 nieudanych) | 122758 |
| local_db3cpu | rest | O3_ramp | C0 | 3 | 1690.9 | 232.98 | 1475.98 | 1638.25 | 0 | 1 (0 nieudanych) | 127919 |
| local | graphql | O1_ramp6k | C0 | 1 | 2562 | 309.13 | 944.54 | 1215.74 | 0 | 1 (0 nieudanych) | 262559 |
| local | graphql | O1_ramp6k | C0 | 2 | 2448.3 | 592.49 | 1052.06 | 1349.1 | 0 | 1 (0 nieudanych) | 325315 |
| local | graphql | O1_ramp | C0 | 1 | 1024 | 0.99 | 3.26 | 28.38 | 0 | 1 (0 nieudanych) | 43 |
| local | graphql | O1_ramp | C0 | 2 | 1014.3 | 1.22 | 100.35 | 207.42 | 0 | 1 (0 nieudanych) | 2325 |
| local | graphql | O3_ramp | C0 | 1 | 792.9 | 1238.91 | 2912.72 | 3035.58 | 0 | 1 (0 nieudanych) | 39185 |
| local | graphql | O3_ramp | C0 | 2 | 678.4 | 1508.58 | 4227.77 | 4541.6 | 0 | 1 (0 nieudanych) | 59292 |
| local | grpc | O1_ramp6k | C0 | 1 | 2638.4 | 56.36 | 869.57 | 1227.63 | n/d (gRPC) | 1 (0 nieudanych) | 224287 |
| local | grpc | O1_ramp6k | C0 | 2 | 143.9 | 12.13 | 556.22 | 1138.6 | n/d (gRPC) | 1 (0 nieudanych) | 129067 |
| local | grpc | O1_ramp | C0 | 1 | 1026.1 | 1.15 | 4.13 | 29.64 | n/d (gRPC) | 1 (0 nieudanych) | 116 |
| local | grpc | O1_ramp | C0 | 2 | 1021.3 | 1.2 | 6.82 | 46.66 | n/d (gRPC) | 1 (0 nieudanych) | 264 |
| local | grpc | O3_ramp | C0 | 1 | 991.7 | 4.45 | 443.27 | 595.52 | n/d (gRPC) | 1 (0 nieudanych) | 5114 |
| local | grpc | O3_ramp | C0 | 2 | 982.1 | 4.5 | 533.43 | 670.57 | n/d (gRPC) | 1 (0 nieudanych) | 6785 |
| local | rest | O1_ramp6k | C0 | 1 | 2996 | 1.38 | 66.35 | 189.39 | 0 | 1 (0 nieudanych) | 34560 |
| local | rest | O1_ramp6k | C0 | 2 | 3004.9 | 1.26 | 53.53 | 129.87 | 0 | 1 (0 nieudanych) | 30391 |
| local | rest | O1_ramp | C0 | 1 | 1028.7 | 0.77 | 2.02 | 10.83 | 0 | 1 (0 nieudanych) | 0 |
| local | rest | O1_ramp | C0 | 2 | 1028.8 | 0.72 | 1.03 | 1.91 | 0 | 1 (0 nieudanych) | 0 |
| local | rest | O3_ramp | C0 | 1 | 1018.3 | 2.25 | 37.76 | 132.51 | 0 | 1 (0 nieudanych) | 1881 |
| local | rest | O3_ramp | C0 | 2 | 1022.4 | 2.16 | 51.8 | 120.71 | 0 | 1 (0 nieudanych) | 479 |

Z plików `*_meta.json`: we wszystkich 36 przebiegach `k6_exit` = 0, `oom_killed` = false, `restart_count` = 0. Pliku `failed.txt` nie ma.

### 5.3 Punkty nasycenia (odczytane z `*_saturation_*_summary.csv` i `seria_a_A1_kalibracja.md`)

**Przyjęte: `results/local_db3cpu/`** (Postgres 3 CPU, 21 X 2026, `config.txt`: `PG_CPUS=3 app_cpus=2 commit=d22c888`)

| iface | op | rep 1 | rep 2 | rep 3 | średnia (z .md) | rozrzut (z .md) | w `rates.env` |
|---|---|---|---|---|---|---|---|
| rest | O1 | 6958 | 7302 | 7065 | 7108 | 5% | 7000 |
| grpc | O1 | 4497 | 4355 | 4709 | 4520 | 8% | 4350 |
| graphql | O1 | 3773 | 3551 | 3771 | 3698 | 6% | 3550 |
| rest | O3 | 2385 | 2330 | 2231 | 2315 | 7% | 2230 |
| grpc | O3 | 1657 | 1534 | 1727 | 1639 | 12% | 1530 |
| graphql | O3 | 1071 | 1101 | 1073 | 1082 | 3% | 1070 |

We wszystkich wierszach `min_ok_frac` = 1.0. `host_cpu_max_pct` (CPU hosta z vmstat): O1 76–92%, O3 69–92%.

**Odrzucone: `results/local/`** (Postgres 2 CPU, 20 X), `sat_rps`, puste pole = brak nasycenia do sufitu rampy

| tag | iface | rep 1 | rep 2 |
|---|---|---|---|
| O1_ramp (do 2000) | rest / graphql / grpc | — / — / — | — / — / — |
| O1_ramp6k (do 6000) | rest | 3557 | — (max_healthy 5823) |
| O1_ramp6k | graphql | 3385 | 2621 |
| O1_ramp6k | grpc | 4002 | 3442 |
| O3_ramp (do 2000) | rest | — (1868) | — (1820) |
| O3_ramp | graphql | 1000 | 630 |
| O3_ramp | grpc | 1610 | 1686 |

### 5.4 CPU i RAM

- Surowe próbki `docker stats` są w `*_resources.csv` (aplikacja, wszystkie 36 przebiegów) i `*_pg_resources.csv` (Postgres, tylko `local_db3cpu`). CPU hosta jest w `*_hostcpu.txt` (`local_db3cpu` i `local/*_O1_ramp6k_*`).
- **W plikach nie ma gotowych wartości średniego ani maksymalnego CPU/RAM na przebieg.** Zgodnie z poleceniem nie liczyłem ich od nowa. Do policzenia potrzebny jest skrypt, np. rozszerzenie `analysis/load.py`, gdzie funkcja `_resources` już czyta te pliki.
- Jedyne gotowe wartości CPU są w `results/local_db3cpu/seria_a_A1_kalibracja.md`: mediana z ostatnich 20% rampy, w % jednego rdzenia:

| przypadek | app (limit 200) | Postgres (limit 300) | maszyna (max) | interpretacja (z pliku) |
|---|---|---|---|---|
| graphql O1 / O3 | 192 / 195 | 66 / 149 | 78 / 70% | CPU aplikacji |
| rest O3 | 135 | 288 | 86% | CPU bazy |
| grpc O3 | 126 | 206–227 | 91% | baza blisko limitu + maszyna ~91% |
| rest O1 | 129 | 101 | 91% | maszyna/k6 |
| grpc O1 | 155 | 81 | 88% | maszyna/k6 |

- `results/local/seria_a_A1_kalibracja.md` podaje tylko: „CPU kontenera aplikacji w końcówce ~110–123%” (REST O1 ramp6k rep 1) i „CPU maszyny max 79–89%”.
- Wartości RAM nie ma w żadnym pliku podsumowania.

### 5.5 Anomalie widoczne w plikach

- `results/local/seria_a/grpc_O1_ramp6k_C0_2`: w `_meta.json` start 17:47:59, koniec 20:41:02. W logu k6: `running (2h53m02.8s)`, choć rampa trwa 9 min. RPS z summary to 143,85, a w innych przebiegach O1 ramp6k ok. 2500–3000. Przyczyny z plików nie da się ustalić. Przebieg należy do odrzuconej próby.
- Wszystkie przebiegi `local_db3cpu` mają w `_meta.json` `"commit":"d22c888…","dirty":"4"`, czyli wykonano je na drzewie z 4 niezacommitowanymi zmianami. Według `git diff --stat d22c888 8934261` późniejszy commit zmienił tylko `infra/` i `loadtests/`, a nie `app/`. Z `results/local/` 12 przebiegów ma `dirty` 0, a 6 ma `dirty` 1.
- `results/local_db3cpu/seria_a_A1_kalibracja.md` mówi, że compose „domyślnie nadal daje 2” CPU dla Postgresa. Obecny `infra/docker-compose.yml` ma `${PG_CPUS:-3}`, więc ta uwaga jest nieaktualna.

### 5.6 Lista wszystkich plików w `results/` (ścieżka, data modyfikacji, rozmiar w bajtach)

<details>
<summary>272 pliki, rozwiń</summary>

| plik | data | rozmiar [B] |
|---|---|---|
| results/local/seria_a/graphql_O1_ramp6k_C0_1.json | 2026-10-20 17:28 | 4348 |
| results/local/seria_a/graphql_O1_ramp6k_C0_1_hostcpu.txt | 2026-10-20 17:28 | 13340 |
| results/local/seria_a/graphql_O1_ramp6k_C0_1_k6.log | 2026-10-20 17:28 | 78563 |
| results/local/seria_a/graphql_O1_ramp6k_C0_1_meta.json | 2026-10-20 17:28 | 636 |
| results/local/seria_a/graphql_O1_ramp6k_C0_1_points.csv.gz | 2026-10-20 17:28 | 60865689 |
| results/local/seria_a/graphql_O1_ramp6k_C0_1_resources.csv | 2026-10-20 17:28 | 11489 |
| results/local/seria_a/graphql_O1_ramp6k_C0_1_seed.txt | 2026-10-20 17:19 | 139 |
| results/local/seria_a/graphql_O1_ramp6k_C0_2.json | 2026-10-20 17:37 | 4328 |
| results/local/seria_a/graphql_O1_ramp6k_C0_2_hostcpu.txt | 2026-10-20 17:37 | 13340 |
| results/local/seria_a/graphql_O1_ramp6k_C0_2_k6.log | 2026-10-20 17:37 | 78531 |
| results/local/seria_a/graphql_O1_ramp6k_C0_2_meta.json | 2026-10-20 17:37 | 636 |
| results/local/seria_a/graphql_O1_ramp6k_C0_2_points.csv.gz | 2026-10-20 17:37 | 58585128 |
| results/local/seria_a/graphql_O1_ramp6k_C0_2_resources.csv | 2026-10-20 17:37 | 11465 |
| results/local/seria_a/graphql_O1_ramp6k_C0_2_seed.txt | 2026-10-20 17:28 | 139 |
| results/local/seria_a/graphql_O1_ramp_C0_1.json | 2026-10-20 14:32 | 4341 |
| results/local/seria_a/graphql_O1_ramp_C0_1_k6.log | 2026-10-20 14:32 | 27380 |
| results/local/seria_a/graphql_O1_ramp_C0_1_meta.json | 2026-10-20 14:32 | 634 |
| results/local/seria_a/graphql_O1_ramp_C0_1_points.csv.gz | 2026-10-20 14:32 | 7465188 |
| results/local/seria_a/graphql_O1_ramp_C0_1_resources.csv | 2026-10-20 14:32 | 3810 |
| results/local/seria_a/graphql_O1_ramp_C0_1_seed.txt | 2026-10-20 14:29 | 139 |
| results/local/seria_a/graphql_O1_ramp_C0_2.json | 2026-10-20 14:35 | 4342 |
| results/local/seria_a/graphql_O1_ramp_C0_2_k6.log | 2026-10-20 14:35 | 27400 |
| results/local/seria_a/graphql_O1_ramp_C0_2_meta.json | 2026-10-20 14:35 | 634 |
| results/local/seria_a/graphql_O1_ramp_C0_2_points.csv.gz | 2026-10-20 14:35 | 7605757 |
| results/local/seria_a/graphql_O1_ramp_C0_2_resources.csv | 2026-10-20 14:35 | 3820 |
| results/local/seria_a/graphql_O1_ramp_C0_2_seed.txt | 2026-10-20 14:32 | 139 |
| results/local/seria_a/graphql_O3_ramp_C0_1.json | 2026-10-20 14:39 | 4278 |
| results/local/seria_a/graphql_O3_ramp_C0_1_k6.log | 2026-10-20 14:39 | 27743 |
| results/local/seria_a/graphql_O3_ramp_C0_1_meta.json | 2026-10-20 14:39 | 634 |
| results/local/seria_a/graphql_O3_ramp_C0_1_points.csv.gz | 2026-10-20 14:39 | 6704726 |
| results/local/seria_a/graphql_O3_ramp_C0_1_resources.csv | 2026-10-20 14:39 | 3911 |
| results/local/seria_a/graphql_O3_ramp_C0_1_seed.txt | 2026-10-20 14:36 | 139 |
| results/local/seria_a/graphql_O3_ramp_C0_2.json | 2026-10-20 14:42 | 4367 |
| results/local/seria_a/graphql_O3_ramp_C0_2_k6.log | 2026-10-20 14:42 | 27866 |
| results/local/seria_a/graphql_O3_ramp_C0_2_meta.json | 2026-10-20 14:42 | 634 |
| results/local/seria_a/graphql_O3_ramp_C0_2_points.csv.gz | 2026-10-20 14:42 | 5860866 |
| results/local/seria_a/graphql_O3_ramp_C0_2_resources.csv | 2026-10-20 14:42 | 3902 |
| results/local/seria_a/graphql_O3_ramp_C0_2_seed.txt | 2026-10-20 14:39 | 139 |
| results/local/seria_a/grpc_O1_ramp6k_C0_1.json | 2026-10-20 17:47 | 2401 |
| results/local/seria_a/grpc_O1_ramp6k_C0_1_hostcpu.txt | 2026-10-20 17:47 | 13331 |
| results/local/seria_a/grpc_O1_ramp6k_C0_1_k6.log | 2026-10-20 17:47 | 78227 |
| results/local/seria_a/grpc_O1_ramp6k_C0_1_meta.json | 2026-10-20 17:47 | 633 |
| results/local/seria_a/grpc_O1_ramp6k_C0_1_points.csv.gz | 2026-10-20 17:47 | 21545975 |
| results/local/seria_a/grpc_O1_ramp6k_C0_1_resources.csv | 2026-10-20 17:47 | 11496 |
| results/local/seria_a/grpc_O1_ramp6k_C0_1_seed.txt | 2026-10-20 17:38 | 139 |
| results/local/seria_a/grpc_O1_ramp6k_C0_2.json | 2026-10-20 20:41 | 2424 |
| results/local/seria_a/grpc_O1_ramp6k_C0_2_hostcpu.txt | 2026-10-20 20:41 | 13225 |
| results/local/seria_a/grpc_O1_ramp6k_C0_2_k6.log | 2026-10-20 20:41 | 78057 |
| results/local/seria_a/grpc_O1_ramp6k_C0_2_meta.json | 2026-10-20 20:41 | 633 |
| results/local/seria_a/grpc_O1_ramp6k_C0_2_points.csv.gz | 2026-10-20 20:41 | 22858848 |
| results/local/seria_a/grpc_O1_ramp6k_C0_2_resources.csv | 2026-10-20 20:41 | 11458 |
| results/local/seria_a/grpc_O1_ramp6k_C0_2_seed.txt | 2026-10-20 17:47 | 139 |
| results/local/seria_a/grpc_O1_ramp_C0_1.json | 2026-10-20 14:46 | 2411 |
| results/local/seria_a/grpc_O1_ramp_C0_1_k6.log | 2026-10-20 14:46 | 27035 |
| results/local/seria_a/grpc_O1_ramp_C0_1_meta.json | 2026-10-20 14:46 | 631 |
| results/local/seria_a/grpc_O1_ramp_C0_1_points.csv.gz | 2026-10-20 14:46 | 2428063 |
| results/local/seria_a/grpc_O1_ramp_C0_1_resources.csv | 2026-10-20 14:46 | 3834 |
| results/local/seria_a/grpc_O1_ramp_C0_1_seed.txt | 2026-10-20 14:43 | 139 |
| results/local/seria_a/grpc_O1_ramp_C0_2.json | 2026-10-20 14:50 | 2399 |
| results/local/seria_a/grpc_O1_ramp_C0_2_k6.log | 2026-10-20 14:50 | 27026 |
| results/local/seria_a/grpc_O1_ramp_C0_2_meta.json | 2026-10-20 14:50 | 631 |
| results/local/seria_a/grpc_O1_ramp_C0_2_points.csv.gz | 2026-10-20 14:50 | 2449070 |
| results/local/seria_a/grpc_O1_ramp_C0_2_resources.csv | 2026-10-20 14:50 | 3829 |
| results/local/seria_a/grpc_O1_ramp_C0_2_seed.txt | 2026-10-20 14:47 | 139 |
| results/local/seria_a/grpc_O3_ramp_C0_1.json | 2026-10-20 14:53 | 2406 |
| results/local/seria_a/grpc_O3_ramp_C0_1_k6.log | 2026-10-20 14:53 | 27064 |
| results/local/seria_a/grpc_O3_ramp_C0_1_meta.json | 2026-10-20 14:53 | 631 |
| results/local/seria_a/grpc_O3_ramp_C0_1_points.csv.gz | 2026-10-20 14:53 | 2870162 |
| results/local/seria_a/grpc_O3_ramp_C0_1_resources.csv | 2026-10-20 14:53 | 3846 |
| results/local/seria_a/grpc_O3_ramp_C0_1_seed.txt | 2026-10-20 14:50 | 139 |
| results/local/seria_a/grpc_O3_ramp_C0_2.json | 2026-10-20 14:57 | 2397 |
| results/local/seria_a/grpc_O3_ramp_C0_2_k6.log | 2026-10-20 14:57 | 27064 |
| results/local/seria_a/grpc_O3_ramp_C0_2_meta.json | 2026-10-20 14:57 | 631 |
| results/local/seria_a/grpc_O3_ramp_C0_2_points.csv.gz | 2026-10-20 14:57 | 2858466 |
| results/local/seria_a/grpc_O3_ramp_C0_2_resources.csv | 2026-10-20 14:57 | 3859 |
| results/local/seria_a/grpc_O3_ramp_C0_2_seed.txt | 2026-10-20 14:54 | 139 |
| results/local/seria_a/rest_O1_ramp6k_C0_1.json | 2026-10-20 17:09 | 4260 |
| results/local/seria_a/rest_O1_ramp6k_C0_1_hostcpu.txt | 2026-10-20 17:09 | 13337 |
| results/local/seria_a/rest_O1_ramp6k_C0_1_k6.log | 2026-10-20 17:09 | 78417 |
| results/local/seria_a/rest_O1_ramp6k_C0_1_meta.json | 2026-10-20 17:09 | 633 |
| results/local/seria_a/rest_O1_ramp6k_C0_1_points.csv.gz | 2026-10-20 17:09 | 116578707 |
| results/local/seria_a/rest_O1_ramp6k_C0_1_resources.csv | 2026-10-20 17:09 | 11447 |
| results/local/seria_a/rest_O1_ramp6k_C0_1_seed.txt | 2026-10-20 17:00 | 139 |
| results/local/seria_a/rest_O1_ramp6k_C0_2.json | 2026-10-20 17:18 | 4258 |
| results/local/seria_a/rest_O1_ramp6k_C0_2_hostcpu.txt | 2026-10-20 17:18 | 13339 |
| results/local/seria_a/rest_O1_ramp6k_C0_2_k6.log | 2026-10-20 17:18 | 78413 |
| results/local/seria_a/rest_O1_ramp6k_C0_2_meta.json | 2026-10-20 17:18 | 633 |
| results/local/seria_a/rest_O1_ramp6k_C0_2_points.csv.gz | 2026-10-20 17:18 | 116650148 |
| results/local/seria_a/rest_O1_ramp6k_C0_2_resources.csv | 2026-10-20 17:18 | 11479 |
| results/local/seria_a/rest_O1_ramp6k_C0_2_seed.txt | 2026-10-20 17:09 | 139 |
| results/local/seria_a/rest_O1_ramp_C0_1.json | 2026-10-20 14:17 | 4229 |
| results/local/seria_a/rest_O1_ramp_C0_1_k6.log | 2026-10-20 14:17 | 27329 |
| results/local/seria_a/rest_O1_ramp_C0_1_meta.json | 2026-10-20 14:17 | 631 |
| results/local/seria_a/rest_O1_ramp_C0_1_points.csv.gz | 2026-10-20 14:17 | 12846832 |
| results/local/seria_a/rest_O1_ramp_C0_1_resources.csv | 2026-10-20 14:17 | 3787 |
| results/local/seria_a/rest_O1_ramp_C0_1_seed.txt | 2026-10-20 14:14 | 139 |
| results/local/seria_a/rest_O1_ramp_C0_2.json | 2026-10-20 14:21 | 4183 |
| results/local/seria_a/rest_O1_ramp_C0_2_k6.log | 2026-10-20 14:21 | 27340 |
| results/local/seria_a/rest_O1_ramp_C0_2_meta.json | 2026-10-20 14:21 | 631 |
| results/local/seria_a/rest_O1_ramp_C0_2_points.csv.gz | 2026-10-20 14:21 | 12723133 |
| results/local/seria_a/rest_O1_ramp_C0_2_resources.csv | 2026-10-20 14:21 | 3780 |
| results/local/seria_a/rest_O1_ramp_C0_2_seed.txt | 2026-10-20 14:18 | 139 |
| results/local/seria_a/rest_O3_ramp_C0_1.json | 2026-10-20 14:24 | 4306 |
| results/local/seria_a/rest_O3_ramp_C0_1_k6.log | 2026-10-20 14:24 | 27388 |
| results/local/seria_a/rest_O3_ramp_C0_1_meta.json | 2026-10-20 14:24 | 631 |
| results/local/seria_a/rest_O3_ramp_C0_1_points.csv.gz | 2026-10-20 14:24 | 14021790 |
| results/local/seria_a/rest_O3_ramp_C0_1_resources.csv | 2026-10-20 14:24 | 3829 |
| results/local/seria_a/rest_O3_ramp_C0_1_seed.txt | 2026-10-20 14:21 | 139 |
| results/local/seria_a/rest_O3_ramp_C0_2.json | 2026-10-20 14:28 | 4345 |
| results/local/seria_a/rest_O3_ramp_C0_2_k6.log | 2026-10-20 14:28 | 27387 |
| results/local/seria_a/rest_O3_ramp_C0_2_meta.json | 2026-10-20 14:28 | 631 |
| results/local/seria_a/rest_O3_ramp_C0_2_points.csv.gz | 2026-10-20 14:28 | 14111026 |
| results/local/seria_a/rest_O3_ramp_C0_2_resources.csv | 2026-10-20 14:28 | 3821 |
| results/local/seria_a/rest_O3_ramp_C0_2_seed.txt | 2026-10-20 14:25 | 139 |
| results/local/seria_a_A1_kalibracja.md | 2026-10-20 21:03 | 3327 |
| results/local/seria_a_saturation_O1_ramp6k_summary.csv | 2026-10-20 21:03 | 401 |
| results/local/seria_a_saturation_O1_ramp6k_windows.csv | 2026-10-20 21:03 | 17877 |
| results/local/seria_a_saturation_O1_ramp_summary.csv | 2026-10-20 21:03 | 330 |
| results/local/seria_a_saturation_O1_ramp_windows.csv | 2026-10-20 21:03 | 6164 |
| results/local/seria_a_saturation_O3_ramp_summary.csv | 2026-10-20 21:03 | 384 |
| results/local/seria_a_saturation_O3_ramp_windows.csv | 2026-10-20 21:03 | 6157 |
| results/local_db3cpu/config.txt | 2026-10-21 12:22 | 67 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_1.json | 2026-10-21 12:47 | 4381 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_1_hostcpu.txt | 2026-10-21 12:47 | 17471 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_1_k6.log | 2026-10-21 12:47 | 106441 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_1_meta.json | 2026-10-21 12:47 | 641 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_1_pg_resources.csv | 2026-10-21 12:47 | 15451 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_1_points.csv.gz | 2026-10-21 12:47 | 95633382 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_1_resources.csv | 2026-10-21 12:47 | 15273 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_1_seed.txt | 2026-10-21 12:35 | 139 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_2.json | 2026-10-21 13:46 | 4309 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_2_hostcpu.txt | 2026-10-21 13:46 | 17429 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_2_k6.log | 2026-10-21 13:46 | 106450 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_2_meta.json | 2026-10-21 13:46 | 641 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_2_pg_resources.csv | 2026-10-21 13:46 | 15443 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_2_points.csv.gz | 2026-10-21 13:46 | 97322295 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_2_resources.csv | 2026-10-21 13:46 | 15250 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_2_seed.txt | 2026-10-21 13:34 | 139 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_3.json | 2026-10-21 14:46 | 4280 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_3_hostcpu.txt | 2026-10-21 14:46 | 17410 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_3_k6.log | 2026-10-21 14:46 | 106469 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_3_meta.json | 2026-10-21 14:46 | 641 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_3_pg_resources.csv | 2026-10-21 14:46 | 15454 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_3_points.csv.gz | 2026-10-21 14:46 | 97917056 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_3_resources.csv | 2026-10-21 14:46 | 15215 |
| results/local_db3cpu/seria_a/graphql_O1_ramp_C0_3_seed.txt | 2026-10-21 14:34 | 139 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_1.json | 2026-10-21 13:13 | 4313 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_1_hostcpu.txt | 2026-10-21 13:13 | 9001 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_1_k6.log | 2026-10-21 13:13 | 53113 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_1_meta.json | 2026-10-21 13:13 | 641 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_1_pg_resources.csv | 2026-10-21 13:13 | 8133 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_1_points.csv.gz | 2026-10-21 13:13 | 16562220 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_1_resources.csv | 2026-10-21 13:13 | 7698 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_1_seed.txt | 2026-10-21 13:08 | 139 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_2.json | 2026-10-21 14:13 | 4297 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_2_hostcpu.txt | 2026-10-21 14:13 | 8978 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_2_k6.log | 2026-10-21 14:13 | 53134 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_2_meta.json | 2026-10-21 14:13 | 641 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_2_pg_resources.csv | 2026-10-21 14:13 | 8155 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_2_points.csv.gz | 2026-10-21 14:13 | 16752225 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_2_resources.csv | 2026-10-21 14:13 | 7670 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_2_seed.txt | 2026-10-21 14:07 | 139 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_3.json | 2026-10-21 15:13 | 4273 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_3_hostcpu.txt | 2026-10-21 15:13 | 8958 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_3_k6.log | 2026-10-21 15:13 | 53127 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_3_meta.json | 2026-10-21 15:13 | 641 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_3_pg_resources.csv | 2026-10-21 15:13 | 8155 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_3_points.csv.gz | 2026-10-21 15:13 | 16018236 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_3_resources.csv | 2026-10-21 15:13 | 7735 |
| results/local_db3cpu/seria_a/graphql_O3_ramp_C0_3_seed.txt | 2026-10-21 15:07 | 139 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_1.json | 2026-10-21 12:59 | 2422 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_1_hostcpu.txt | 2026-10-21 12:59 | 17473 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_1_k6.log | 2026-10-21 12:59 | 105990 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_1_meta.json | 2026-10-21 12:59 | 638 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_1_pg_resources.csv | 2026-10-21 12:59 | 15396 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_1_points.csv.gz | 2026-10-21 12:59 | 36900567 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_1_resources.csv | 2026-10-21 12:59 | 15230 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_1_seed.txt | 2026-10-21 12:48 | 139 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_2.json | 2026-10-21 13:59 | 2418 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_2_hostcpu.txt | 2026-10-21 13:59 | 17330 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_2_k6.log | 2026-10-21 13:59 | 105991 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_2_meta.json | 2026-10-21 13:59 | 638 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_2_pg_resources.csv | 2026-10-21 13:59 | 15443 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_2_points.csv.gz | 2026-10-21 13:59 | 37888880 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_2_resources.csv | 2026-10-21 13:59 | 15226 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_2_seed.txt | 2026-10-21 13:47 | 139 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_3.json | 2026-10-21 14:59 | 2414 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_3_hostcpu.txt | 2026-10-21 14:59 | 17325 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_3_k6.log | 2026-10-21 14:59 | 105991 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_3_meta.json | 2026-10-21 14:59 | 638 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_3_pg_resources.csv | 2026-10-21 14:59 | 15414 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_3_points.csv.gz | 2026-10-21 14:59 | 37810198 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_3_resources.csv | 2026-10-21 14:59 | 15217 |
| results/local_db3cpu/seria_a/grpc_O1_ramp_C0_3_seed.txt | 2026-10-21 14:47 | 139 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_1.json | 2026-10-21 13:20 | 2398 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_1_hostcpu.txt | 2026-10-21 13:21 | 9001 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_1_k6.log | 2026-10-21 13:20 | 52542 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_1_meta.json | 2026-10-21 13:20 | 638 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_1_pg_resources.csv | 2026-10-21 13:21 | 8122 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_1_points.csv.gz | 2026-10-21 13:20 | 8441032 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_1_resources.csv | 2026-10-21 13:20 | 7617 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_1_seed.txt | 2026-10-21 13:15 | 139 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_2.json | 2026-10-21 14:20 | 2378 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_2_hostcpu.txt | 2026-10-21 14:20 | 8972 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_2_k6.log | 2026-10-21 14:20 | 52668 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_2_meta.json | 2026-10-21 14:20 | 638 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_2_pg_resources.csv | 2026-10-21 14:20 | 8133 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_2_points.csv.gz | 2026-10-21 14:20 | 8431475 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_2_resources.csv | 2026-10-21 14:20 | 7661 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_2_seed.txt | 2026-10-21 14:14 | 139 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_3.json | 2026-10-21 15:20 | 2412 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_3_hostcpu.txt | 2026-10-21 15:20 | 9067 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_3_k6.log | 2026-10-21 15:20 | 52952 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_3_meta.json | 2026-10-21 15:20 | 638 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_3_pg_resources.csv | 2026-10-21 15:20 | 8028 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_3_points.csv.gz | 2026-10-21 15:20 | 7473151 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_3_resources.csv | 2026-10-21 15:20 | 7606 |
| results/local_db3cpu/seria_a/grpc_O3_ramp_C0_3_seed.txt | 2026-10-21 15:14 | 139 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_1.json | 2026-10-21 12:34 | 4316 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_1_hostcpu.txt | 2026-10-21 12:34 | 17471 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_1_k6.log | 2026-10-21 12:34 | 106300 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_1_meta.json | 2026-10-21 12:34 | 638 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_1_pg_resources.csv | 2026-10-21 12:34 | 15522 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_1_points.csv.gz | 2026-10-21 12:34 | 208393769 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_1_resources.csv | 2026-10-21 12:34 | 15257 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_1_seed.txt | 2026-10-21 12:22 | 139 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_2.json | 2026-10-21 13:33 | 4319 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_2_hostcpu.txt | 2026-10-21 13:33 | 17469 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_2_k6.log | 2026-10-21 13:33 | 106295 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_2_meta.json | 2026-10-21 13:33 | 638 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_2_pg_resources.csv | 2026-10-21 13:33 | 15478 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_2_points.csv.gz | 2026-10-21 13:33 | 208323479 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_2_resources.csv | 2026-10-21 13:33 | 15240 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_2_seed.txt | 2026-10-21 13:22 | 139 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_3.json | 2026-10-21 14:33 | 4337 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_3_hostcpu.txt | 2026-10-21 14:33 | 17469 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_3_k6.log | 2026-10-21 14:33 | 106301 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_3_meta.json | 2026-10-21 14:33 | 638 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_3_pg_resources.csv | 2026-10-21 14:33 | 15473 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_3_points.csv.gz | 2026-10-21 14:33 | 208277328 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_3_resources.csv | 2026-10-21 14:33 | 15226 |
| results/local_db3cpu/seria_a/rest_O1_ramp_C0_3_seed.txt | 2026-10-21 14:21 | 139 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_1.json | 2026-10-21 13:06 | 4354 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_1_hostcpu.txt | 2026-10-21 13:06 | 8999 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_1_k6.log | 2026-10-21 13:06 | 53043 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_1_meta.json | 2026-10-21 13:06 | 638 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_1_pg_resources.csv | 2026-10-21 13:06 | 8123 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_1_points.csv.gz | 2026-10-21 13:06 | 47796610 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_1_resources.csv | 2026-10-21 13:06 | 7631 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_1_seed.txt | 2026-10-21 13:01 | 139 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_2.json | 2026-10-21 14:06 | 4317 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_2_hostcpu.txt | 2026-10-21 14:06 | 9000 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_2_k6.log | 2026-10-21 14:06 | 53042 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_2_meta.json | 2026-10-21 14:06 | 638 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_2_pg_resources.csv | 2026-10-21 14:06 | 8119 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_2_points.csv.gz | 2026-10-21 14:06 | 48322468 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_2_resources.csv | 2026-10-21 14:06 | 7643 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_2_seed.txt | 2026-10-21 14:00 | 139 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_3.json | 2026-10-21 15:06 | 4302 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_3_hostcpu.txt | 2026-10-21 15:06 | 8984 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_3_k6.log | 2026-10-21 15:06 | 53038 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_3_meta.json | 2026-10-21 15:06 | 638 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_3_pg_resources.csv | 2026-10-21 15:06 | 8080 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_3_points.csv.gz | 2026-10-21 15:06 | 47981538 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_3_resources.csv | 2026-10-21 15:06 | 7643 |
| results/local_db3cpu/seria_a/rest_O3_ramp_C0_3_seed.txt | 2026-10-21 15:00 | 139 |
| results/local_db3cpu/seria_a_A1_kalibracja.md | 2026-10-21 15:32 | 2976 |
| results/local_db3cpu/seria_a_saturation_O1_ramp_summary.csv | 2026-10-21 15:31 | 545 |
| results/local_db3cpu/seria_a_saturation_O1_ramp_windows.csv | 2026-10-21 15:31 | 37012 |
| results/local_db3cpu/seria_a_saturation_O3_ramp_summary.csv | 2026-10-21 15:32 | 561 |
| results/local_db3cpu/seria_a_saturation_O3_ramp_windows.csv | 2026-10-21 15:32 | 18682 |
| results/meta/env_local.txt | 2026-10-13 10:57 | 930 |

</details>

---

## 6. BRAKI I RYZYKA

### 6.1 Czego brakuje, żeby uruchomić serie A–D

Seria A (A2), B, C, D:

1. ~~Tokeny JWT wygasają 19 XI 2026.~~ **Rozwiązane 19 XI 2026.** `loadtests/tokens.json` wygenerowano ponownie (100 tokenów legalnych + napastnik, sekret `thesis-secret`), `exp` = 1801436833, czyli 2027-02-01 00:07 CET. Sprawdzone na serwerze REST w C2: z tokenem 200, bez tokenu 401. Dodatkowo `run_experiment.sh` przerywa przebieg (kod 3), gdy do wygaśnięcia pierwszego tokenu zostało mniej niż 48 h, a `make tokens` ma teraz `TOKENS_TTL ?= 2160h` zamiast domyślnych 24 h z `cmd/tokens`.
2. **Brak jakiegokolwiek przebiegu z konfiguracją C1–C5 przez k6.** Ścieżki k6 z TLS (HTTPS z certyfikatem self-signed, gRPC z `plaintext:false`, `lib/common.js:136`) i z tokenami nie zostały sprawdzone pomiarem. `infra/config_check.sh` testuje zachowanie klientem Go (`cmd/configcheck`), nie przez k6. Czy k6 gRPC zaakceptuje certyfikat self-signed: nie wiem.
3. **Pilotaż serii A2/B/C/D** (krótkie czasy) i przejście ich wyników przez `analysis/load.py` + `stats.py`: według `notatki/fix.md` (p. 3) nic z tego nie zostało zrobione. Wyników takich przebiegów w `results/` nie ma.
4. **Seria B: nominalna siła ataku jest nieosiągalna.** `run_matrix.sh:36` ustawia `ATTACK_RATE = SAT × 5` lub `× 10` (np. REST O1: 70 000 RPS), a A1 pokazała sufit generatora na tej maszynie ok. 7–8 tys. RPS. Decyzja jest otwarta (`LOG.md`, 28 X, pkt 2).
5. **Kalibracja progów** `RATELIMIT_RPS`/`BURST` (100/200), `MAX_COST` (w kodzie 400, w planie 300), `MAX_DEPTH`, `MAX_ALIASES`, limitu 64 KiB na O4: otwarte według `notatki/fix.md`. `complexity_test.go` zawiera `TestCalibration`, ale wyniku dla O3 nie ma w `LOG.md`.
6. **Zamrożenie kodu**: tagu `v1.0-experiments` nie ma (są tylko `v0.1-baseline`, `v0.2-security`).
7. **Miejsce na dysku**: przebieg rampy A1 daje ok. 95–200 MB `*_points.csv.gz` (`results/local_db3cpu` = 1,2 GB dla 18 przebiegów). Przy 351 planowanych przebiegach z 600 s pomiaru wolnego miejsca nie sprawdzałem.
8. **Generowanie wykresów**: w `analysis/` nie ma kodu rysującego (wymaganych rysunków 1, 2, 4, 7–10 z `notatki/stan_projektu.md`).
9. **ghz**: brak skryptu. Komenda w `notatki/cli.md` jest nietestowana.
10. **Środowisko AWS**: `infra/aws/` i `results/aws/*` są puste (w notatkach oznaczone jako opcjonalne).

### 6.2 TODO, FIXME i znane błędy w kodzie

- `grep -rn "TODO|FIXME|XXX|HACK|BUG"` (bez `.git`, `results`, plików generowanych) **nic nie znalazł**.
- Znane ograniczenia opisane w komentarzach kodu:
  - `app/internal/security/http.go:57`: mapa limiterów rośnie bez czyszczenia.
  - `app/internal/grpcsrv/validate.go:15-16`: gRPC akceptuje `limit=0` (proto3), REST/GraphQL nie. To znana różnica modeli.
  - `app/internal/graphql/loaders.go:29`: okno dataloadera 1 ms dodaje do ~1 ms latencji na poziom zagnieżdżenia.
  - `infra/docker-compose.yml`: cAdvisor pominięty (nie działa z Dockerem 29). Monitoring obciąża tę samą maszynę.
- Inne obserwacje z kodu:
  - `app/internal/graphql/complexity.go:154`: `var _ = fmt.Sprintf`, martwy import (kosmetyka).
  - `loadtests/seria_c.js:5`: import `check` nieużywany (kosmetyka).
  - REST O3 zawsze zwraca pełną strukturę niezależnie od `expand` (`rest.go:126-130`), więc parametr jest walidowany, ale nie zmienia odpowiedzi.
  - Brak testu równoważności dla O4. Brak testów `internal/store`, `internal/rest`, `internal/grpcsrv` (interceptory i walidator).
  - gRPC: brak limitu głębokości zagnieżdżenia wiadomości.

### 6.3 Co zajmie najwięcej czasu (ocena)

1. **Same pomiary**: według `LOG.md`/`notatki` ok. 351 przebiegów × ~25 min ≈ 150 h czystego czasu maszyny, w praktyce kilka tygodni nocy. Każdy błąd wykryty w trakcie (np. wygasłe tokeny, źle dobrany próg) oznacza powtórkę części serii. Dlatego największym ryzykiem czasowym jest pominięcie pilotażu.
2. **Pilotaż i kalibracja progów** (rate limit, koszt GraphQL, decyzja o sile ataku w B). Wymaga iteracji i uzgodnień z promotorem, a blokuje zamrożenie kodu.
3. **Analiza i wykresy**: kod statystyk jest szkieletem, a kodu wykresów nie ma. Trzeba go napisać i sprawdzić na prawdziwych danych.

---

## Podsumowanie

Aplikacja w trzech wariantach (REST, GraphQL z dataloaderem, gRPC) jest kompletna dla operacji O1–O4, kompiluje się, przechodzi `go vet`, a testy jednostkowe zabezpieczeń i złożoności GraphQL przechodzą (testy równoważności były pominięte z powodu braku bazy). Wszystkie pięć mechanizmów C1–C5 jest zaimplementowanych we wszystkich interfejsach i przełączanych zmiennymi środowiskowymi. Jedyną luką funkcjonalną jest brak limitu głębokości wiadomości w gRPC. Skrypty k6 dla serii A–D, orkiestracja przebiegów i zbieranie CPU/RAM przez `docker stats` istnieją, ale poza kalibracją A1 żaden z nich nie był jeszcze uruchomiony. Jedyne wyniki to 18 przyjętych przebiegów A1 (C0, O1/O3, `results/local_db3cpu/`), które dają punkty nasycenia REST 7000/2230, gRPC 4350/1530 i GraphQL 3550/1070 RPS, oraz 18 przebiegów odrzuconej pierwszej próby. Przed uruchomieniem serii A2–D trzeba co najmniej wygenerować nowe tokeny JWT (obecne wygasają 19 XI), przejść pilotaż z włączonymi zabezpieczeniami, skalibrować progi i rozstrzygnąć siłę ataku w serii B.
