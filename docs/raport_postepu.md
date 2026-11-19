# Raport postępu, 19.11.2026

Stan na koniec dnia. Wszystkie liczby pochodzą z plików wyników wskazanych przy tabelach. Nic nie zostało wypchnięte na
GitHub ani utworzone w AWS.

---

## 1. Co zostało zrobione dziś

| commit | opis |
|---|---|
| `f87944a` | `run_experiment.sh`: przerwanie przebiegu (kod 3), gdy tokeny JWT wygasają w ciągu 48 h |
| `8f5aeee` | `Makefile`: `TOKENS_TTL ?= 2160h` przekazywane do `cmd/tokens` (wcześniej domyślne 24 h) |
| `d8fa4aa` | `docs/raport_stanu.md` (raport stanu repozytorium), punkt 6.1.1 zaktualizowany: tokeny odnowione |
| `08d2bfc` | `loadtests/run_pilot.sh`: pilotaż A2 z losową kolejnością (stałe ziarno) i zapisem stanu hosta; `.gitignore` dla surowych danych pilotażu |
| `43677ef` | `infra/aws/`: szkielet Terraform + `start.sh`, `teardown.sh`, `README.md` z planem i kosztami (nic nie utworzono) |
| `dea06f7` | `analysis/pilot_table.py`, `plot_pilot.py`, `plot_a1.py`, wykresy PNG, podsumowania pilotażu |
| (ten plik) | `docs/raport_postepu.md` |

Stan względem `origin/main`: lokalny `main` jest 8 commitów do przodu (1 sprzed dziś, `8934261`, i 7 dzisiejszych).
Sprawdziłem niewypchnięte zmiany: nie ma w nich kluczy prywatnych, tokenów ani plików `*.csv.gz`
(`infra/certs/*.key`, `loadtests/tokens.json` i surowe wyniki są w `.gitignore`, co potwierdził `git check-ignore`).
Największy plik ma 37 KB (`*_windows.csv`), a wykresy PNG łącznie 448 KB. **Push jest bezpieczny, ale go nie wykonałem.**

Pozostałe czynności (bez zmian w kodzie): `make up`, `make seed`, `make test`, `make build`, `infra/config_check.sh`,
3 przebiegi testu dymnego k6 (`results/local_smoke/`, poza git), 12 przebiegów pilotażu (`results/pilot/`),
1 przebieg diagnostyczny (`results/local_diag/`, poza git).

## 2. Tokeny JWT

- `loadtests/tokens.json`: 100 tokenów legalnych + 1 token napastnika, sekret `thesis-secret`,
  `exp` = 1801436833 (2027-02-01 00:07 CET). Nie były generowane ponownie.
- **Test curl, REST w C2** (`TLS_ENABLED=true AUTH_ENABLED=true`, `https://localhost:8443/users/1`):

| żądanie | kod HTTP |
|---|---|
| z tokenem legalnym (`legit[0]`) | **200** (`{"id":1,"name":"User 1","email":"user1@example.com"}`, TLS 1.3, HTTP/1.1) |
| z tokenem napastnika | 200 (poprawny JWT, to oczekiwane) |
| bez tokenu | **401** |
| z uszkodzonym podpisem | 401 |
| zwykłe HTTP na porcie TLS | 400 |

- **Sprawdzanie ważności w `run_experiment.sh`:** po kroku, który tworzy `tokens.json`, gdy go brak, skrypt odczytuje
  `exp` pierwszego tokenu legalnego (`jq`: dekodowanie base64url części payload). Jeśli do wygaśnięcia zostało mniej
  niż `TOKEN_MIN_LEFT` (domyślnie 172 800 s = 48 h) albo pliku nie da się odczytać, przebieg kończy się **kodem 3**
  z komunikatem, kiedy tokeny wygasają, i poleceniem `make tokens TOKENS_TTL=...`. Tokenów nie generuje sam.
  Sprawdzenie działa w każdej konfiguracji (także C0/C1), żeby macierz zatrzymała się od razu, a nie na pierwszym C2.
  Test na sztucznych plikach: token wygasający za 1 h daje kod 3, ważny 10 dni przechodzi, uszkodzony plik daje kod 3.
  Podczas pilotażu (12 przebiegów) i testu dymnego sprawdzenie przeszło.

## 3. Wyniki etapu 2

### 3.1 Testy z działającą bazą
`make seed`: users 10 000, categories 50, products 5000, orders 50 000, order_items 200 000, md5 zamówień
`572c8a0c0d22d2e4e54af83daeb10f2f`. `go test -count=1 -v ./internal/equiv/`:

| test | wynik | zakres |
|---|---|---|
| `TestEquivO3` | **PASS** (0,82 s) | 100 losowych użytkowników, REST = GraphQL = gRPC |
| `TestEquivO1O2` | **PASS** (0,23 s) | 100 losowych użytkowników (O1) + O2 |
| `TestEquivNotFound` | **PASS** (0,02 s) | spójne błędy „nie znaleziono” |

Testy nie były pominięte (żadnego `SKIP`). `internal/graphql` i `internal/security` też przechodzą. Testu O4 brak.

### 3.2 Macierz `infra/config_check.sh` (po `make build`)
Kolumny zachowania: odrzucenie zwykłego HTTP (TLS) | 401 bez tokenu (JWT) | odrzucenie dużego ładunku (walidacja) |
429 przy przekroczeniu limitu (rate limit) | odrzucenie zbyt złożonego zapytania (złożoność).

| konf. | oczekiwane | REST | GraphQL | gRPC |
|---|---|---|---|---|
| C0 | – – – – – | zgodne | zgodne | zgodne |
| C1 | T – – – – | zgodne | zgodne | zgodne |
| C2 | T T – – – | zgodne | zgodne | zgodne |
| C3 | T T T – – | zgodne | zgodne | zgodne |
| C4 | T T T T – | zgodne | zgodne | zgodne |
| C5 | T T T T T | zgodne | zgodne | zgodne |

**18/18 OK** (T = mechanizm zadziałał, – = nie powinien i nie zadziałał). Dla każdej konfiguracji poprawne żądanie
bazowe też przeszło (`baseline_ok=true`).

### 3.3 k6 z TLS i tokenami
Przebieg `run_experiment.sh local_smoke seria_a <iface> O1 C5 1 50` (C5 = TLS + JWT + walidacja + rate limit + złożoność;
50 RPS; 5 s rozgrzewki, 20 s pomiaru):

| interfejs | iteracje | poprawne (checks) | błędne | uchwyt TLS max |
|---|---|---|---|---|
| REST | 1302 | 1302 | 0 | 26,58 ms |
| GraphQL | 1302 | 1302 | 0 | 21,79 ms |
| gRPC | 1302 | 1302 | 0 | n/d (k6 nie raportuje dla gRPC) |

**gRPC w k6 działa z certyfikatem self-signed (`plaintext:false`), poprawka nie jest potrzebna.** Tokeny są
akceptowane przez wszystkie trzy interfejsy. Moja pomyłka po drodze: pierwsza próba z `COOLDOWN=0s` odpadła
na walidacji k6 (faza musi trwać co najmniej 1 s).

## 4. Pilotaż A2

> **To pilotaż:** 1 powtórzenie, 120 s pomiaru, laptop (k6, aplikacja i baza na jednym hoście). Wyniki są orientacyjne
> i nie nadają się do wniosków o narzucie mechanizmów (patrz 4.2).

Ustawienia: env `pilot`, O1, C0/C2/C4/C5, WARMUP 30 s, DURATION 120 s, COOLDOWN 30 s, pauza 60 s, RATE = 70% SAT
z `loadtests/rates.env` (REST 4900, GraphQL 2485, gRPC 3045). Kolejność losowa (ziarno 20261119, `results/pilot/order.txt`).
Wszystkie przebiegi: `k6_exit` 0, brak OOM i restartów, w CSV 61–62 próbki `docker stats` w oknie pomiaru.

### 4.1 Tabela (`results/pilot/pilot_summary.md`, z `analysis/pilot_table.py`)

Latencje z fazy `measure` (tylko poprawne odpowiedzi), osiągnięty RPS = żądania w oknie / 120 s, CPU/RAM kontenera
aplikacji w oknie pomiaru, narzut = (wartość / wartość C0 tego interfejsu − 1) × 100%.

| interfejs | konf. | zadany RPS | osiągnięty RPS | p50 [ms] | p95 [ms] | p99 [ms] | błędy [%] | CPU śr. [%] | CPU max [%] | RAM śr. [MiB] | RAM max [MiB] | p95 vs C0 [%] | CPU vs C0 [%] |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| rest | C0 | 4900 | 4840.2 | 1.65 | 18.17 | 38.05 | 0.000 | 111.3 | 121.2 | 24.4 | 27.4 | — | — |
| rest | C2 | 4900 | 3244.5 | 68.79 | 204.65 | 287.20 | 0.000 | 100.0 | 150.1 | 79.6 | 96.4 | 1026.4 | -10.1 |
| rest | C4 | 4900 | 3535.6 | 58.20 | 173.53 | 256.33 | 0.000 | 99.6 | 139.8 | 74.1 | 90.3 | 855.1 | -10.5 |
| rest | C5 | 4900 | 3441.2 | 66.22 | 196.99 | 291.95 | 0.000 | 99.0 | 126.5 | 76.2 | 94.7 | 984.2 | -11.0 |
| graphql | C0 | 2485 | 2474.5 | 1.09 | 5.49 | 21.36 | 0.000 | 112.8 | 147.2 | 20.2 | 21.6 | — | — |
| graphql | C2 | 2485 | 2438.4 | 4.87 | 67.96 | 126.46 | 0.000 | 117.0 | 143.6 | 43.5 | 50.4 | 1137.0 | 3.8 |
| graphql | C4 | 2485 | 2441.3 | 2.56 | 42.34 | 89.96 | 0.000 | 118.5 | 142.6 | 45.7 | 49.7 | 670.6 | 5.1 |
| graphql | C5 | 2485 | 2457.3 | 1.78 | 31.18 | 82.04 | 0.000 | 116.9 | 142.6 | 35.3 | 39.9 | 467.6 | 3.7 |
| grpc | C0 | 3045 | 3013.7 | 1.51 | 8.76 | 23.30 | 0.000 | 123.1 | 132.0 | 26.9 | 30.0 | — | — |
| grpc | C2 | 3045 | 2879.2 | 35.32 | 218.28 | 325.51 | 0.000 | 118.5 | 150.6 | 154.0 | 225.0 | 2393.0 | -3.8 |
| grpc | C4 | 3045 | 2874.5 | 34.23 | 171.75 | 265.93 | 0.000 | 115.6 | 159.4 | 179.0 | 210.9 | 1861.6 | -6.1 |
| grpc | C5 | 3045 | 2814.2 | 78.29 | 259.70 | 356.94 | 0.000 | 119.9 | 158.2 | 182.6 | 219.3 | 2866.1 | -2.6 |

`dropped_iterations` z podsumowań k6 (iteracje, których k6 nie zdołał wysłać): REST C0 6270, C2 321 921, C4 257 993,
C5 243 736; GraphQL C0 448, C2 10 401, C4 10 422, C5 3980; gRPC C0 1057, C2 33 163, C4 49 082, C5 41 168.
Liczba VU nie przekroczyła 967 (limit `maxVUs` 2000). W C4/C5 nie było odpowiedzi 429 (błędy 0,000%), więc limit
100 RPS na token nie dławił ruchu legalnego, przy czym w REST osiągnięty ruch był niższy od zadanego.

### 4.2 Interpretacja: w C2–C5 pilotaż mierzy laptop, a nie mechanizmy
- W REST i gRPC z TLS osiągnięty RPS był niższy od zadanego (REST 3244–3536 / 4900, gRPC 2814–2879 / 3045), a CPU
  aplikacji (śr. 99–120%) było daleko od limitu 200%. „Narzut CPU” REST wyszedł ujemny, bo aplikacja obsłużyła mniej żądań.
- **Przebieg diagnostyczny** (REST C2, 4900 RPS, 60 s, `results/pilot/diag_rest_C2_cpu.txt`, próbki co 5–7 s):
  CPU całego hosta 91,8–100% (`top`), proces k6 ok. 500% (`ps %cpu`, średnia od startu procesu), aplikacja
  77,06–110,94%, Postgres 42,30–73,58%. Osiągnięto 3055 RPS, `dropped_iterations` 145 092.
  **Wniosek: z TLS wąskim gardłem jest generator k6 na wspólnym laptopie.** To ten sam problem, co w A1 dla REST/gRPC O1,
  tylko silniejszy, bo k6 szyfruje każde połączenie.
- GraphQL (najniższy zadany RPS) osiągnął 98–99% zadanego. Jego liczby są najmniej zniekształcone, ale p95 maleje
  C2 > C4 > C5, czego mechanizmy nie tłumaczą (C5 ma ich najwięcej). Przy 1 powtórzeniu jest to szum lub efekt generatora.
  Nie wiem, który z nich.
- k6 ostrzegał o 100 000–200 000 unikalnych szeregach metryk: tag URL zawiera `/users/{id}` z 10 000 różnych id.
  To dodatkowo obciąża generator i pamięć k6.

### 4.3 Czy któryś przebieg odstaje (`results/pilot/host_pre.csv`)
Przed każdym przebiegiem: `loadavg`, % bezczynności CPU (`vmstat`), „cpu MHz” z `/proc/cpuinfo` oraz czas stałego,
jednowątkowego testu obliczeniowego (3 próby, mediana). `/proc/cpuinfo` w WSL2 pokazuje stałe 1497,6 MHz we wszystkich
przebiegach, więc nie odzwierciedla taktowania. Miarą jest czas testu.

| nr | interfejs | konf. | load1 | bezczynność [%] | mediana testu [ms] | vs mediana wszystkich [%] |
|---|---|---|---|---|---|---|
| 1 | graphql | C2 | 0.3 | 96 | 253.9 | -2.1 |
| 2 | rest | C2 | 3.6 | 99 | 262.5 | +1.2 |
| 3 | grpc | C5 | 5.5 | 98 | 264.3 | +1.9 |
| 4 | graphql | C4 | 5.4 | 99 | 239.0 | -7.9 |
| 5 | grpc | C4 | 4.9 | 100 | 268.2 | +3.4 |
| 6 | graphql | C0 | 4.9 | 99 | 261.2 | +0.7 |
| 7 | rest | C4 | 2.1 | 100 | 260.1 | +0.3 |
| 8 | rest | C0 | 4.4 | 100 | 249.6 | -3.8 |
| 9 | rest | C5 | 4.4 | 100 | 251.2 | -3.2 |
| 10 | grpc | C2 | 4.8 | 99 | 253.0 | -2.5 |
| 11 | grpc | C0 | 5.9 | 100 | 258.7 | -0.3 |
| 12 | graphql | C5 | 4.1 | 100 | 266.3 | +2.7 |

Mediana 259,4 ms, zakres 239,0–268,2 ms. **Żaden przebieg wyraźnie nie odstaje.** Najbardziej odbiega nr 4 (−7,9%,
procesor szybszy, a nie dławiony). `load1` 2–6 to resztka po poprzednim przebiegu (średnia krocząca nie opada przez 60 s),
a chwilowa bezczynność wynosiła 96–100%. Kolejność potwierdza, że degradacja zależy od konfiguracji, a nie od temperatury:
REST C0 (nr 8) jest w normie, choć stał między REST C4 (nr 7) i REST C5 (nr 9), które spadły.

### 4.4 Wykresy
- `analysis/plots/pilot_p95_overhead.png`: narzut p95 względem C0 (C2/C4/C5 × interfejs), z przypisem o przebiegach,
  w których osiągnięty RPS był niższy niż 95% zadanego
- `analysis/plots/pilot_cpu_overhead.png`: to samo dla średniego CPU aplikacji
- `analysis/plots/a1_offered_vs_achieved.png`: kalibracja A1, osiągnięty vs zadany RPS dla O1 i O3 z punktami
  nasycenia (3 powtórzenia). Zadany RPS wyliczony z definicji rampy k6 (liniowo 10→8000 RPS / 720 s dla O1,
  10→4000 / 360 s dla O3). Ostatnie, niepełne okno każdego przebiegu pominięte.

Odtworzenie: `~/venv-thesis/bin/python analysis/pilot_table.py pilot && ~/venv-thesis/bin/python analysis/plot_pilot.py pilot && ~/venv-thesis/bin/python analysis/plot_a1.py`.

## 5. Plan AWS (szczegóły: `infra/aws/README.md`)

**Architektura:** 2 instancje w `eu-central-1a`, w placement group `cluster`.
- Generator **c7i.4xlarge** (16 vCPU, 32 GiB, gp3 120 GB): k6, zapisy `*_points.csv.gz`.
- Serwer **c7i.2xlarge** (8 vCPU, 16 GiB, gp3 40 GB): aplikacja (2 CPU / 2 GB) i Postgres (3 CPU / 2 GB) w Dockerze,
  z tymi samymi limitami co lokalnie. 3 wolne vCPU są na system i pomiar zasobów.
- Bez trzeciej maszyny dla bazy: zachowujemy zgodność z kalibracją lokalną i unikamy przeskoku sieciowego przy każdym
  zapytaniu SQL. To wariant rezerwowy.
- Rozmiar generatora: z A1 wynika rzędu 0,7 wątku hosta na 1000 RPS dla k6 bez TLS. To przybliżenie z jednego pomiaru,
  a nie zmierzona wydajność k6. Pilotaż pokazał, że TLS znacznie zwiększa koszt po stronie k6. Na AWS pierwszy krok to
  przebieg z `vmstat` na generatorze. Jeśli k6 przekroczy ok. 60% CPU, trzeba wziąć c7i.8xlarge.
- Security group: SSH tylko z bieżącego IP, ruch testowy tylko wewnątrz grupy.

**Koszt instancji** (ceny on-demand Linux z oficjalnych danych kalkulatora AWS, publikacja 2026-11-18T22:02:54Z):
c7i.2xlarge 0,4074 $/h, c7i.4xlarge 0,8148 $/h (Frankfurt). Razem 1,2222 $/h. W Sztokholmie 0,3822 + 0,7644 = 1,1466 $/h.

| wariant | godziny | Frankfurt | Sztokholm |
|---|---|---|---|
| macierz jak lokalnie (351 × ~25 min) | 150 | 183,33 $ | 171,99 $ |
| + pilotaż i A1 na AWS | 170 | 207,77 $ | 194,92 $ |
| pauza 60 s zamiast 600 s (351 × ~16 min) | 94 | 114,89 $ | 107,78 $ |
| jw. + pilotaż i A1 | 110 | 134,44 $ | 126,13 $ |

Ceny dysków gp3 i spot: **nie wiem** (nie udało się ich odczytać z oficjalnego źródła). Trzeba doliczyć 160 GB gp3.
Jak ograniczyć koszt: skrócić pauzę (na serwerach w chmurze nie chodzi o temperaturę laptopa), zatrzymywać instancje
między seriami, wybrać Sztokholm, ewentualnie `REPS_D=1`. Spotu nie polecam do pomiarów, bo przerwanie niszczy przebieg.
Budżet: `aws_budgets_budget` w `main.tf` (domyślnie 150 $/mies.; alarmy e-mail przy 50/80/100% rzeczywistych kosztów
i 100% prognozy).

**Pliki w `infra/aws/`:** `main.tf`, `variables.tf`, `outputs.tf`, `terraform.tfvars.example`, `user_data.sh`
(Docker, Go 1.26.0, k6 v2.3.0, jq, sysstat, chrony, klon repozytorium), `start.sh`, `teardown.sh`, `README.md`.
Wybrałem Terraform zamiast skryptów CLI, bo zapisuje stan, `plan` pokazuje zmiany przed utworzeniem, a `destroy` usuwa
dokładnie to, co powstało. `teardown.sh` dodatkowo sprząta po tagu `Project=thesis-bench`.
**Szkielet nie był walidowany (`terraform validate`), bo Terraform nie jest zainstalowany.**

**Kroki uruchomienia (po Twojej akceptacji):** instalacja Terraform i AWS CLI + `aws configure`; `terraform.tfvars`;
`./start.sh` (pokazuje plan, pyta o „tak”); po ok. 5 min `make build && make up` na serwerze, tokeny na generatorze;
test dymny C5; A1 na AWS i nowe `rates.env`; macierz; po każdej serii `rsync` wyników; na końcu `./teardown.sh`.

**Skrypty `loadtests/` na dwóch maszynach (propozycja, niezaimplementowana):** zmienna `SERVER`. Gdy jest ustawiona,
`run_experiment.sh` wykonuje przez `ssh` na serwerze `docker compose down/up` (z flagami C0–C5), seed,
`export_resources.sh` i `vmstat`, a potem kopiuje CSV `scp` do `results/` na generatorze. k6 łączy się z
`HOST=$SERVER:<port>`. Na generatorze dodatkowy `vmstat` do `*_gencpu.txt`. `chrony` na obu maszynach do
synchronizacji zegarów.

## 6. Braki przed właściwymi pomiarami (od najważniejszego)

1. **Decyzja, gdzie mierzyć.** Pilotaż pokazał, że na laptopie przy 70% SAT z TLS (C2–C5) REST i gRPC są ograniczone
   przez generator, więc lokalna macierz A2–D w obecnej postaci zmierzyłaby laptopa, a nie mechanizmy. Opcje:
   (a) AWS (plan gotowy, koszt w p. 5), (b) lokalnie ze wspólnym, niższym RPS, który k6 utrzyma z TLS (trzeba go
   wyznaczyć), (c) jedno i drugie: lokalnie tylko jakościowo.
2. **Pomiar CPU generatora w każdym przebiegu** (`vmstat` w `run_experiment.sh`, dziś tylko w A1). Bez tego nie da się
   wykazać, że generator nie był wąskim gardłem.
3. **Tag `name` dla URL w k6** (`lib/ops.js`, np. `name: "/users/{id}"`), żeby usunąć 100–200 tys. szeregów metryk.
4. **Push 8 commitów** na GitHub (bezpieczne, zrobisz to sam).
5. **Decyzje z promotorem:** siła ataku w serii B (nominalne 5×/10× SAT jest nieosiągalne także na c7i.4xlarge
   dla REST O1), powtórzenia serii D, pauza między przebiegami.
6. **Kalibracja progów:** rate limit (w pilotażu brak 429 przy legalnym ruchu, ale przy obniżonym RPS),
   `MAX_COST`/`MAX_DEPTH`/`MAX_ALIASES` względem O3 i zapytań serii D, limit 64 KB dla O4.
7. **Pilotaż serii B, C, D** (po jednym przebiegu) i przejście ich przez `analysis/load.py`/`stats.py`.
8. **Zamrożenie kodu**: tag `v1.0-experiments`.
9. Mniej pilne: test równoważności O4, limit głębokości wiadomości w gRPC, `openapi.yaml`, skrypt ghz.

## 7. Podsumowanie na zajęcia

Aplikacja w trzech wariantach (REST, GraphQL, gRPC) jest gotowa: testy równoważności z prawdziwą bazą potwierdzają te
same odpowiedzi dla O1–O3, a automatyczna macierz potwierdza, że każdy z pięciu poziomów zabezpieczeń C0–C5 działa
zgodnie z założeniami we wszystkich interfejsach (18/18). Kalibracja A1 wyznaczyła punkty nasycenia bez zabezpieczeń:
REST 7000, gRPC 4350 i GraphQL 3550 żądań na sekundę dla prostego odczytu. Dzisiejszy pilotaż przy 70% tego obciążenia
pokazał, że po włączeniu TLS generator ruchu i serwer, działające na jednym laptopie, wysycają procesor (CPU hosta
92–100%), więc lokalnie mierzyłbym ograniczenie laptopa, a nie koszt zabezpieczeń. Dlatego pomiary docelowe planuję na
AWS: osobna maszyna dla generatora k6 i osobna dla aplikacji z bazą w tej samej strefie, z kosztem samych instancji od 114,89 do 207,77 $ (Frankfurt)
w zależności od długości pauz między przebiegami. Najbliższe kroki to decyzja o środowisku i sile ataku w serii B,
pomiar CPU generatora w każdym przebiegu, kalibracja progów i zamrożenie kodu przed startem właściwej macierzy.
