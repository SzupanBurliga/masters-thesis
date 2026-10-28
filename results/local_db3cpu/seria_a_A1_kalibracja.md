# A1 — kalibracja nasycenia C0, środowisko local_db3cpu

Pomiary: 2026-10-21, commit z `config.txt`. WSL2, 8 rdzeni, 15,7 GB. Skrypt: `loadtests/run_a1_calibration.sh`.
Limity: aplikacja 2 CPU / 2 GB, **Postgres 3 CPU** (`PG_CPUS=3`; wcześniej 2) / 2 GB, k6 na hoście bez limitu.
18 przebiegów (3 interfejsy × O1/O3 × 3 powtórzenia), C0, wszystkie k6 exit 0, brak OOM/restartów/błędów (min_ok_frac = 1.0).
Rampy o tym samym nachyleniu (~667 RPS/min): O1 10→8000 RPS / 12 min, O3 10→4000 RPS / 6 min. Kolejność: rep → op → iface.

Kryterium (`analysis/saturation.py local_db3cpu seria_a <O1_ramp|O3_ramp>`): okna 10 s, nasycenie = początek 3 kolejnych okien
z p50 > max(5 × baza_p50, 10 ms) lub poprawnych < 99%. Dane okien: `results/local_db3cpu/seria_a_saturation_*_windows.csv`.

## Punkty nasycenia [RPS]
| iface | op | rep 1 | rep 2 | rep 3 | średnia | rozrzut |
|---|---|---|---|---|---|---|
| rest | O1 | 6958 | 7302 | 7065 | 7108 | 5% |
| grpc | O1 | 4497 | 4355 | 4709 | 4520 | 8% |
| graphql | O1 | 3773 | 3551 | 3771 | 3698 | 6% |
| rest | O3 | 2385 | 2330 | 2231 | 2315 | 7% |
| grpc | O3 | 1657 | 1534 | 1727 | 1639 | 12% |
| graphql | O3 | 1071 | 1101 | 1073 | 1082 | 3% |

Sufity rampy (peak 7652–8149 dla O1, 2618–2654 dla O3 w REST) leżą powyżej punktu nasycenia — każdy przypadek się nasycił.
Brak trendu spadkowego między rep 1→3 (np. REST O1 rośnie), więc dławienia termicznego nie widać.

## Co ogranicza przepustowość (CPU: aplikacja z 200%, Postgres z 300%, maszyna z 800%; mediana z ostatnich 20% rampy)
| przypadek | app | Postgres | maszyna (max) | interpretacja |
|---|---|---|---|---|
| graphql O1 / O3 | 192 / 195 | 66 / 149 | 78 / 70% | **CPU aplikacji** (limit 2) — nasycenie realne |
| rest O3 | 135 | 288 | 86% | **CPU bazy** (288/300) |
| grpc O3 | 126 | 206–227 | 91% | baza blisko limitu + maszyna ~91% |
| rest O1 | 129 | 101 | 91% | nic z limitów; **maszyna/k6** |
| grpc O1 | 155 | 81 | 88% | nic z limitów; **maszyna/k6** |

Wniosek: dla GraphQL nasycenie dotyczy aplikacji. Dla REST/gRPC O1 (i częściowo gRPC O3) mierzy się raczej limit generatora/hosta
(k6 + kontenery na jednym 8-rdzeniowym laptopie, CPU maszyny ~90%). Takie wartości są prawidłowym punktem odniesienia
dla tej maszyny, ale nie wydajnością samej aplikacji. Do pracy: opisać to ograniczenie; docelowe pomiary na AWS (osobny host dla k6).

Wpływ limitu bazy: przy Postgresie 2 CPU REST nasycał się wyraźnie niżej lub wcale (O1 ≥3500, O3 >1850); przy 3 CPU: 7100 i 2300.

## Wartości dla loadtests/rates.env (propozycja = minimum z 3 powtórzeń, zaokrąglone w dół — konserwatywnie)
SAT_REST_O1=7000  SAT_REST_O3=2230
SAT_GRAPHQL_O1=3550  SAT_GRAPHQL_O3=1070
SAT_GRPC_O1=4350  SAT_GRPC_O3=1530

UWAGA: te wartości dotyczą PG_CPUS=3. Kolejne serie (A2, B, C, D) muszą być uruchamiane z tym samym limitem
(`PG_CPUS=3`; domyślnie compose nadal daje 2), inaczej kalibracja nie ma zastosowania.
