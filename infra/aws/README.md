# Środowisko pomiarowe AWS: plan do akceptacji

> **Nic z tego katalogu nie zostało uruchomione.** `start.sh` tworzy płatne zasoby. Uruchamiać dopiero po akceptacji
> kosztów. Na tym komputerze nie ma jeszcze AWS CLI ani Terraform (stan 19.11.2026).

## 1. Dlaczego AWS

Kalibracja A1 (`results/local_db3cpu/seria_a_A1_kalibracja.md`) wykazała, że dla REST O1 i gRPC O1 CPU całej maszyny
dochodziło do 88–92%, a kontenery aplikacji (129% / 155% z 200%) i bazy (101% / 81% z 300%) nie były u swoich limitów.
Wąskim gardłem był więc wspólny host z generatorem k6. Na AWS generator i serwer stoją na osobnych maszynach.

## 2. Architektura

```
            Twoje IP (tylko SSH :22)
                   │
   ┌───────────────┴──────────────── eu-central-1a, placement group "cluster" ───────────┐
   │                                                                                     │
   │  generator: c7i.4xlarge (16 vCPU, 32 GiB)      serwer: c7i.2xlarge (8 vCPU, 16 GiB)  │
   │  k6, *_points.csv.gz, analiza      ──HTTPS/gRPC──▶  Docker: app (2 CPU, 2 GB)        │
   │  gp3 120 GB                        prywatny IP      Postgres 16 (3 CPU, 2 GB)       │
   │                                                     docker stats + vmstat, gp3 40 GB │
   └─────────────────────────────────────────────────────────────────────────────────────┘
```

- **Dwie instancje, nie trzy.** Aplikacja i baza na jednej maszynie z tymi samymi limitami cgroup co lokalnie
  (app 2 CPU / 2 GB, Postgres 3 CPU / 2 GB). Dzięki temu wyniki dają się porównać z kalibracją lokalną i nie
  dochodzi przeskok sieciowy przy każdym zapytaniu SQL (O3 to 3 zapytania). Trzecia maszyna dla bazy miałaby sens, gdyby
  pomiary pokazały, że app i Postgres konkurują o pamięć podręczną/przepustowość pamięci mimo limitów CPU. To wariant
  rezerwowy (+0,4074 $/h za drugą c7i.2xlarge).
- **Ta sama strefa i placement group `cluster`**: najniższe i najstabilniejsze opóźnienie sieciowe między generatorem a
  serwerem, więc sieć dodaje jak najmniej szumu do p95.
- **Rozmiar serwera, c7i.2xlarge (8 vCPU):** limity kontenerów sumują się do 5 vCPU. Pozostałe 3 vCPU są na system,
  Dockera, pętlę `docker stats` i `vmstat`, żeby pomiar zasobów nie zabierał CPU mierzonym kontenerom.
- **Rozmiar generatora, c7i.4xlarge (16 vCPU):** z A1 (REST O1, ok. 7000 RPS, bez TLS) wynika, że CPU hosta
  wynosiło 90–92% z 8 wątków, z czego aplikacja zajmowała 129%, a Postgres 101% (dane z pliku kalibracji). Na k6 i system
  zostawało więc rzędu 4,9 wątku, czyli ok. 0,7 wątku na 1000 RPS. **To przybliżenie z jednego pomiaru, a nie
  zmierzona wydajność k6.** Przy ~10 tys. RPS i TLS daje to ponad 7 wątków, więc 16 vCPU zostawia ok. 2× zapasu.
  Pierwszy krok na AWS: przebieg z `vmstat` na generatorze. Jeśli k6 przekracza ~60% CPU, potrzebna jest większa
  instancja (c7i.8xlarge). Mniejsza (c7i.2xlarge) wystarczy tylko wtedy, gdy k6 nie przekracza ~40%.
- **Uwaga do serii B:** nominalne 10× nasycenia (np. REST O1, 70 000 RPS) przekracza także ten generator. Decyzja o
  sposobie liczenia siły ataku obowiązuje również na AWS.
- Inne procesory (Sapphire Rapids zamiast i5-1145G7) oznaczają, że **kalibrację A1 trzeba powtórzyć na AWS**
  (`run_a1_calibration.sh aws`). Punkty nasycenia z laptopa nie przenoszą się na nowe maszyny.

## 3. Pliki

| plik | rola |
|---|---|
| `main.tf` | placement group, security group (SSH tylko z Twojego IP, reszta ruchu tylko wewnątrz grupy), 2 instancje Ubuntu 24.04, budżet z alarmami |
| `variables.tf`, `outputs.tf` | parametry (typy instancji, region, repozytorium/tag) i adresy po utworzeniu |
| `terraform.tfvars.example` | wzór `terraform.tfvars` (ten plik jest w `.gitignore`) |
| `user_data.sh` | cloud-init: Docker + compose, Go 1.26.0, k6 v2.3.0, jq, sysstat, klon repozytorium na wskazanym tagu |
| `start.sh` | `terraform init/plan/apply` z bieżącym IP (`checkip.amazonaws.com`); pyta o potwierdzenie |
| `teardown.sh` | `terraform destroy` + kontrola i awaryjne usunięcie po tagu `Project=thesis-bench` (AWS CLI) |

**Dlaczego Terraform, a nie skrypty AWS CLI:** stan zasobów jest zapisany, więc `terraform destroy` usuwa dokładnie to,
co powstało (instancje, dyski, SG, placement group, klucz, budżet). `plan` pokazuje przed utworzeniem, co powstanie.
Skrypty CLI musiałyby same pilnować kolejności i identyfikatorów. CLI zostaje tylko jako zabezpieczenie w `teardown.sh`.

## 4. Koszt

Ceny on-demand, Linux, $/h, z oficjalnych danych kalkulatora AWS
(`https://b0.p.awsstatic.com/pricing/2.0/meteredUnitMaps/ec2/USD/current/ec2-ondemand-without-sec-sel/<region>/Linux/index.json`,
publikacja 2026-11-18T22:02:54Z):

| typ | eu-central-1 (Frankfurt) | eu-north-1 (Sztokholm) |
|---|---|---|
| c7i.xlarge | 0,2037 | 0,1911 |
| c7i.2xlarge | 0,4074 | 0,3822 |
| c7i.4xlarge | 0,8148 | 0,7644 |
| c7i.8xlarge | 1,6296 | 1,5288 |

Ceny gp3 (EBS) i spot dla regionu: **nie wiem** (nie udało się ich odczytać z oficjalnego źródła).
Dyski (160 GB łącznie) trzeba doliczyć osobno według cennika EBS.

Koszt instancji (cena × godziny, obie maszyny włączone przez cały czas):

| wariant | godziny | Frankfurt: 2xl + 4xl (1,2222 $/h) | Sztokholm: 2xl + 4xl (1,1466 $/h) | Frankfurt: 2xl + 2xl (0,8148 $/h) |
|---|---|---|---|---|
| macierz jak lokalnie (351 × ~25 min) | 150 | 183,33 $ | 171,99 $ | 122,22 $ |
| + pilotaż i kalibracja A1 na AWS | 170 | 207,77 $ | 194,92 $ | 138,52 $ |
| pauza 60 s zamiast 600 s (351 × ~16 min) | 94 | 114,89 $ | 107,78 $ | 76,59 $ |
| jw. + pilotaż i A1 | 110 | 134,44 $ | 126,13 $ | 89,63 $ |

(16 min = 2 + 10 + 2 min faz k6 + ok. 1 min startu i seedu + 1 min pauzy.)

### Jak ograniczyć koszt
1. **Pauza 600 s → 60 s.** Pauza służyła stabilizacji temperatury laptopa. Na serwerach w chmurze ten powód praktycznie
   znika, a pauza stanowi ok. 40% czasu przebiegu, więc oszczędność to ok. 37% (tabela wyżej). Kolejność przebiegów i tak
   warto losować (jak w `run_pilot.sh`).
2. **Zatrzymywanie instancji między seriami** (`aws ec2 stop-instances`): zatrzymana instancja nie płaci za CPU, tylko za
   dyski. Wyniki zostają na dysku generatora.
3. **Sztokholm zamiast Frankfurtu:** ok. 6% taniej, opóźnienie do Polski nie ma znaczenia (cały ruch testowy jest wewnątrz AWS).
4. **Spot: nie polecam na pomiary.** Przerwanie w trakcie 25-minutowego przebiegu niszczy przebieg, a inna maszyna
   po wznowieniu to inny sprzęt. Ewentualnie tylko na pilotaż/debugowanie.
5. **Mniej przebiegów:** np. `REPS_D=1` (54 zamiast 108 przebiegów serii D) skraca macierz o ok. 15%.

### AWS Budgets
`main.tf` tworzy budżet miesięczny `thesis-bench-monthly` (`budget_usd`, domyślnie 150 $) z e-mailami przy 50%, 80%
i 100% kosztów rzeczywistych oraz przy 100% prognozy. Pomiary przejdą najpewniej przez dwa miesiące kalendarzowe,
więc limit ustaw na ok. 60–70% kosztu całego wariantu. Dwa pierwsze budżety na koncie są bezpłatne. Dodatkowo warto
włączyć w konsoli **Billing → Free Tier/Cost Anomaly Detection** (darmowe).

## 5. Przeniesienie `loadtests/` na dwie maszyny (propozycja, jeszcze nie zaimplementowana)

Dziś `run_experiment.sh` robi wszystko lokalnie: `docker compose down/up`, seed, `export_resources.sh` (docker stats),
k6 na `localhost`. Proponowana zmiana (mała, zgodna wstecz):

- Nowa zmienna `SERVER` (adres prywatny serwera). Puste = obecne zachowanie lokalne.
- Gdy `SERVER` jest ustawione, polecenia serwerowe idą przez `ssh ubuntu@$SERVER`:
  `docker compose ... down/up` (z flagami C0–C5 przekazanymi w poleceniu), `./bin/seed` (lub seed z generatora po
  `DB_URL=postgres://...@$SERVER:5432/shop`), `export_resources.sh start/stop` i `vmstat` (CSV zapisywane na serwerze,
  na końcu przebiegu kopiowane `scp` do `results/` na generatorze), `docker inspect` (OOM, restarty) do `_meta.json`.
- k6 dostaje `HOST=$SERVER:<port>` (zmienna już istnieje w `lib/common.js`). Certyfikat jest self-signed i k6
  go akceptuje (`insecureSkipTLSVerify`, sprawdzone lokalnie dla HTTP i gRPC w C5).
- `wait_ready.sh` sprawdza `$SERVER` zamiast `localhost`.
- Zegar: `chrony` na obu maszynach (instalowany w `user_data.sh`), bo okna CPU są dopasowywane do znaczników czasu k6.
- Na generatorze `vmstat` co 5 s do `*_gencpu.txt`, żeby wykazać w pracy, że generator nie był wąskim gardłem.

## 6. Kroki uruchomienia (po akceptacji)

1. Zainstaluj Terraform i AWS CLI, skonfiguruj `aws configure` (użytkownik IAM z uprawnieniami EC2 + Budgets).
2. `cp terraform.tfvars.example terraform.tfvars`, uzupełnij e-mail i `repo_ref` (tag `v1.0-experiments`).
3. `TF_VAR_owner_email=... ./start.sh`, potwierdź „tak”.
4. Po ok. 5 min: `ssh` na serwer, `make build`, `make up`; na generatorze `make tokens TOKENS_TTL=2160h`
   i skopiuj `tokens.json` (z tym samym `JWT_SECRET`).
5. Test dymny (C5, 20 s) dla 3 interfejsów, potem A1 na AWS, potem `rates.env` dla AWS, potem macierz `run_matrix.sh aws ...`.
6. Po każdej serii: `rsync` wyników z generatora na laptopa, ewentualnie `stop-instances`.
7. Na koniec: `./teardown.sh` i sprawdzenie w konsoli, że nic z tagiem `Project=thesis-bench` nie zostało.
