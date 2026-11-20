# Raport z nocy 19/20.11.2026 (część 2)

`loadtests/run_blocks.sh --max-hours 8 --pause 60 variability pilot_a2_rep2 d_rate_cal pilot_bcd2 a1_tls variability_extra`,
start 23:12:17, koniec 05:27:56 (6 h 16 min; limit do 07:12:17). Kod: commit `df68ba7`, czyste drzewo. Liczby pochodzą z
`_meta.json`, `*_points.csv.gz`, `*_resources.csv`, `*_hostcpu.txt`, a wyliczenia robią `analysis/pilot_table.py`,
`analysis/load.py`, `analysis/saturation.py` i `analysis/stats.py: cv`. Laptop: WSL2, k6 i serwer na jednej maszynie.

## 1. Co się wykonało

| blok | przebiegi | status | uwagi |
|---|---|---|---|
| 1. `variability` (REST O1 C0, 4900 RPS, powt. 1–5) | 5 | 5 OK | powt. 1 i 4 odstają (p. 3) |
| 2. `pilot_a2_rep2` | 12 | 12 OK | kolejność: `results/pilot/order_rep2.txt` (ziarno 20261120) |
| 3. `d_rate_cal` (GraphQL D, C0, 10/50/100/200 zapytań/s) | 12 | 12 OK | |
| 4. `pilot_bcd2` (B, C, D po poprawkach P1–P4) | 24 | 24 OK | |
| 5. `a1_tls` (A1 w C2) | 18 | 18 OK | REST O1 powt. 1 odstaje (p. 6) |
| 6. `variability_extra` (powt. 6–10) | 5 | 5 OK | |
| **razem** | **76** | **76 OK** | brak pominięć z braku czasu, błędów, OOM i restartów |

Wszystko, co zaplanowano, zostało wykonane.

## 2. CPU hosta (vmstat, `host_cpu_busy_mean/max` z `_meta.json`)

Na laptopie host = generator k6 + kontenery, więc to jest też CPU generatora.

| blok | n | średnia przebiegu: min / mediana / max [%] | maks. chwilowe: min / max [%] |
|---|---|---|---|
| variability | 5 | 70,4 / 71,8 / 81,6 | 77 / 90 |
| variability_extra | 5 | 70,6 / 71,1 / 71,5 | 77 / 80 |
| pilot_a2_rep2 | 12 | 47,2 / 82,3 / 94,9 | 53 / 99 |
| d_rate_cal | 12 | 25,4 / 38,3 / 45,4 | 30 / 73 |
| pilot_bcd2 | 24 | 25,6 / 48,3 / 76,1 | 30 / 94 |
| a1_tls | 18 | 63,1 / 77,95 / 81,5 | 80 / 99 |

W pilotażu A2 (powt. 2) średnie CPU hosta: REST C2/C4/C5 94,4–94,9% (maks. 98–99), gRPC C2/C4/C5 89,5–90,0% (maks. 94),
GraphQL C2/C4/C5 74,1–75,1% (maks. 84–85). W C0: REST 71,6, GraphQL 47,2, gRPC 67,0%.

## 3. Zmienność p95 (REST O1 C0, stały ruch 4900 RPS, 10 powtórzeń; `results/pilot_var/pilot_summary.md`)

| powt. | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 |
|---|---|---|---|---|---|---|---|---|---|---|
| p95 [ms] | **181,05** | 12,08 | 10,51 | **162,51** | 11,26 | 12,83 | 11,93 | 11,74 | 11,71 | 12,79 |
| osiągnięty RPS | **3997,5** | 4854,6 | 4887,0 | **4201,5** | 4883,9 | 4871,7 | 4875,9 | 4852,5 | 4849,9 | 4842,0 |
| CPU hosta śr. [%] | 81,4 | 70,6 | 71,8 | 81,6 | 70,4 | 71,5 | 70,9 | 70,6 | 71,3 | 71,1 |

**Współczynnik zmienności (CV) p95:**

| zbiór | n | CV |
|---|---|---|
| wszystkie | 10 | 154,14% |
| bez powt. 1 i 4 | 8 | 6,42% (p95 10,51–12,83 ms, mediana 11,84) |
| powt. 6–10 | 5 | 4,60% |

Cel pilotażu (CV < 5–10%) jest spełniony, ale **tylko po odrzuceniu dwóch przebiegów z zakłóceniami**.

**Powt. 1 i 4 to epizody, nie przesunięcie:** w oknach 10 s fazy pomiaru powt. 1 ma RPS 4667 i 4891, potem spadek
do 2287 i 1475 oraz p95 195,6–306,0 ms, a następnie częściowy powrót. Powt. 4 ma p95 rosnące od 41,4 ms do 839,3 ms
i RPS spadający do 2327. W powt. 4 `vmstat` pokazuje us 58–60% przy 43–46% w powt. 2 przy tym samym ruchu, czyli coś
zużywało więcej CPU. **Przyczyny nie znam:** w plikach nie ma śladu innego procesu. Oba epizody wypadły w pierwszych
17 minutach nocy (23:12–23:29). Wniosek dla metody jest w p. 7.

## 4. Porównanie obu powtórzeń pilotażu A2 (`results/pilot/pilot_summary.md`)

| interfejs | konf. | osiągnięty RPS powt. 1 / 2 | p95 [ms] powt. 1 / 2 | narzut p95 vs C0 [%] powt. 1 / 2 | CPU aplikacji śr. [%] powt. 1 / 2 |
|---|---|---|---|---|---|
| REST | C0 | 4840,2 / 4839,5 | 18,17 / 16,02 | — | 111,3 / 118,5 |
| REST | C2 | 3244,5 / 3473,0 | 204,65 / 169,69 | 1026,4 / 959,5 | 100,0 / 102,6 |
| REST | C4 | 3535,6 / 3273,3 | 173,53 / 178,43 | 855,1 / 1014,1 | 99,6 / 105,3 |
| REST | C5 | 3441,2 / 3480,9 | 196,99 / 172,32 | 984,2 / 976,0 | 99,0 / 101,1 |
| GraphQL | C0 | 2474,5 / 2477,0 | 5,49 / 3,01 | — | 112,8 / 117,2 |
| GraphQL | C2 | 2438,4 / 2447,5 | 67,96 / 26,39 | 1137,0 / 777,5 | 117,0 / 121,8 |
| GraphQL | C4 | 2441,3 / 2457,9 | 42,34 / 27,22 | 670,6 / 804,9 | 118,5 / 120,7 |
| GraphQL | C5 | 2457,3 / 2443,1 | 31,18 / 38,59 | 467,6 / 1182,9 | 116,9 / 122,2 |
| gRPC | C0 | 3013,7 / 3028,2 | 8,76 / 7,86 | — | 123,1 / 127,9 |
| gRPC | C2 | 2879,2 / 2869,8 | 218,28 / 131,89 | 2393,0 / 1577,6 | 118,5 / 118,6 |
| gRPC | C4 | 2874,5 / 2853,1 | 171,75 / 128,08 | 1861,6 / 1529,1 | 115,6 / 120,8 |
| gRPC | C5 | 2814,2 / 2861,2 | 259,70 / 129,26 | 2866,1 / 1544,1 | 119,9 / 119,7 |

Błędy: 0,000% we wszystkich 24 przebiegach.

- **Obraz z pierwszego pilotażu się powtarza:** w REST i gRPC z TLS osiągnięty RPS jest poniżej zadanego, a CPU hosta
  wynosi ok. 90–95%. To nadal pomiar generatora, a nie mechanizmów.
- **GraphQL** (98–99% zadanego RPS, CPU hosta 74–85%) jest najmniej zniekształcony, ale p95 tej samej konfiguracji
  różni się między powtórzeniami nawet 2,6× (C2: 67,96 vs 26,39 ms). Kolejność C2 > C4 > C5 z pierwszego powtórzenia
  się nie potwierdziła (w drugim C5 ma najwyższe p95). Przy p95 rzędu kilkudziesięciu ms i dwóch powtórzeniach różnic
  między C2, C4 i C5 nie da się rozstrzygnąć.
- Wykres `analysis/plots/pilot_p95_overhead.png` pokazuje teraz medianę z 2 powtórzeń z zakresem min–max.

## 5. Kalibracja siły ataku GraphQL w serii D (P2; C0, legalne tło 1065 RPS; `results/pilot_drate/`)

p95 legalnych [ms]: baza → faza ataku. Napastnik: zrealizowane żądania w 60 s (zaplanowane = siła × 60).

| wariant | 10/s | 50/s | 100/s | 200/s |
|---|---|---|---|---|
| aliasy 50 | 1,19 → 5,49 (596 z 600) | 1,35 → **487,63** (2892 z 3000) | 1,24 → 3134,47 (3193 z 6000) | 1,26 → 3204,11 (3353 z 12 000) |
| `limit` 50 | 1,28 → 5,79 (599 z 600) | 1,24 → **1590,43** (2445 z 3000) | 1,24 → 3125,54 (2524 z 6000) | 1,31 → 3181,33 (2587 z 12 000) |
| głębokość 10 | 1,25 → 1,32 (596) | 1,29 → 1,34 (2983) | 1,23 → 1,59 (5947) | 1,22 → 3,31 (11 975) |

- Aliasy i `limit` przy 50/s nasycają aplikację: CPU śr. 149,18% i 155,41%, maks. 199,03% i 202,12% przy limicie 200%.
  Powyżej 50/s generator nie wysyła już zaplanowanych żądań (aplikacja nie nadąża, VU się kończą), a p95 legalnych
  zatrzymuje się na ok. 3,1–3,2 s.
- **Głębokość 10 jest tania nawet przy 200/s** (p95 napastnika 21,26 ms, legalnych 3,31 ms). Głębokie zapytanie po
  łańcuchu kategorii jest obsługiwane przez dataloadery stałą liczbą zapytań SQL. To wynik sam w sobie.
- **Propozycja: `ATTACK_RATE_D_GRAPHQL=50`**, najniższa z badanych wartości, przy której aliasy i `limit` wyraźnie
  szkodzą (p95 legalnych 488 i 1590 ms), a generator jeszcze realizuje 82–96% zaplanowanych żądań. Wartości między 10
  a 50 nie mierzyłem. **Nie zmieniałem `rates.env`, to czeka na Twoją decyzję.** Dla głębokości 10 żadna z badanych
  sił nie daje dużego efektu.

## 6. Kalibracja A1: z TLS (C2) i bez (C0) (`results/local_db3cpu/seria_a_saturation_*_ramp[_C2]_summary.csv`)

Punkt nasycenia [RPS], 3 powtórzenia; zmiana = mediana C2 względem mediany C0.

| iface | op | C0 (powt. 1/2/3) | C2 (powt. 1/2/3) | mediana C0 → C2 | zmiana | CPU hosta maks. C0 / C2 [%] |
|---|---|---|---|---|---|---|
| REST | O1 | 6958 / 7302 / 7065 | **979** / 3587 / 3613 | 7065 → 3587 | −49,2% | 90–92 / 98–99 |
| gRPC | O1 | 4497 / 4355 / 4709 | 2916 / 2927 / 2781 | 4497 → 2916 | −35,2% | 87–88 / 99 |
| GraphQL | O1 | 3773 / 3551 / 3771 | 3007 / 2984 / 3091 | 3771 → 3007 | −20,3% | 76–78 / 99 |
| REST | O3 | 2385 / 2330 / 2231 | 1805 / 1739 / 1614 | 2330 → 1739 | −25,4% | 85–86 / 93–95 |
| gRPC | O3 | 1657 / 1534 / 1727 | 1279 / 1281 / 1171 | 1657 → 1279 | −22,8% | 91–92 / 96–97 |
| GraphQL | O3 | 1071 / 1101 / 1073 | 1048 / 1033 / 1039 | 1073 → 1039 | **−3,2%** | 69–70 / 80–81 |

- **O1 z TLS:** wszystkie trzy interfejsy nasycają się w okolicy 2800–3600 RPS przy CPU hosta 98–99%. To limit laptopa
  szyfrującego po obu stronach, a nie interfejsów. Ranking REST > gRPC > GraphQL z C0 tu nie jest pomiarem interfejsów.
- **GraphQL O3** to jedyny przypadek, w którym host nie jest nasycony (80–81%) i, jak w C0, wąskim gardłem jest CPU
  aplikacji. Tu −3,2% (mediana) można traktować jako lokalny pomiar narzutu TLS + JWT na przepustowość.
- REST O3 i gRPC O3 (CPU hosta 93–97%): nie da się oddzielić narzutu mechanizmu od limitu hosta.
- **REST O1 C2, powt. 1 (979 RPS) to zakłócenie, a nie nasycenie:** do okna 160 s wszystko w normie (1798,7 RPS, p50
  1,15 ms), w oknie 170 s RPS spada do 978,6, a p50 do 68,13 ms, i do końca rampy 56 z 73 okien jest „złych”, choć
  szczyt RPS później wynosi 3696. To ten sam wzorzec co w powt. 1 i 4 zmienności. Przyczyna: nie wiem.

## 7. Pilotaż B, C, D po poprawkach (`results/pilot_bcd2/`)

24/24 OK. Kody odpowiedzi napastnika (z `*_points.csv.gz`):

| seria | C0 | z ochroną |
|---|---|---|
| B (1000 RPS, 1 token) | 100% sukcesów | C4: REST 6194 × 200 + 53 382 × 429; GraphQL 6196 + 53 493 × 429; gRPC 6144 + 52 925 × kod 8 |
| C (3314 pozycji) | REST 922 × 201, GraphQL 903 × 200, gRPC 724 × OK | C5: REST 1201 × 413, GraphQL 1200 × 413, **gRPC 1200 × kod 3** |
| D GraphQL głęb./aliasy/`limit` (10/s) | 200 | C5: 422 (601/600/601) |
| D REST JSON 64 | 201 | C5: 601 × 400 |
| D tablica 2000 | REST 601 × 201, gRPC 601 × OK | C5: **REST 601 × 400, gRPC 600 × kod 3** |

- **P3 działa:** we wszystkich interfejsach 3314 pozycji. Rozmiar: REST 104 963 B, GraphQL 105 051 B, gRPC 23 073 B.
  Ten sam ładunek trafia w REST/GraphQL w limit rozmiaru (413, już w C3), a w gRPC dopiero w limit długości tablicy
  (kod 3, C5), co trzeba opisać w pracy. Napastnik gRPC w C0 zrealizował teraz 724 z 1200 żądań (przed P3 było 114).
- **P4 działa:** tablica 2000 (REST 63 600 B, gRPC 13 955 B) jest odrzucana przez limit złożoności (REST 400, gRPC kod 3).
  W C0 jest jednak dużo łagodniejsza niż 10 000: p95 legalnych w fazie ataku REST 18,72 ms, gRPC 62,83 ms
  (przy 10 000: 26 773 i 34 798 ms).
- **P1 działa:** `limit` 50 w C0 daje p50 napastnika 23,70 ms (w pierwszym pilotażu 10,83 ms przy tym samym 10/s).
- **P5/P6:** `recovery_s` jest teraz liczbą (0–10 s) wszędzie poza przebiegami, w których system nie wrócił w 10 s
  wystygnięcia (C w C0, gRPC tablica 2000 w C0). Czas odrzucenia (p50 wszystkich odpowiedzi napastnika) wynosi
  0,75–2,77 ms, a dla gRPC C5 w serii C 10,11 ms.
- Seria C w C0 nadal jest bardzo szkodliwa: p95 legalnych w fazie ataku 17 501–21 510 ms, po ataku 25 299–33 783 ms.
  10 s wystygnięcia w pilotażu to za mało, żeby zobaczyć powrót. W macierzy jest 120 s, ale nie wiem, czy wystarczy.
- Niewyjaśnione: REST tablica 2000 C5, `base_p95` 34,24 ms (w C0 1,56 ms). Pojedynczy przebieg, nie wiem dlaczego.

## 8. Wnioski i propozycje (do decyzji)

1. **Zakłócenia na laptopie występują i trzeba je obsłużyć w metodzie:** 3 z 16 przebiegów REST tej nocy, w których
   da się to ocenić (zmienność 2/10, A1 C2 1/6; w pilotażu A2 REST C2–C5 host był nasycony, więc nie da się), miały epizod, w którym RPS spada, a latencja rośnie przez kilka
   minut. Propozycja: (a) automatyczna flaga w analizie, gdy osiągnięty RPS w fazie pomiaru jest < 95% zadanego
   w konfiguracji, w której host nie jest nasycony (C0), i ponowienie takiego przebiegu; (b) raportowanie mediany
   z ≥ 3 powtórzeń zamiast średniej; (c) przed nocą zamknąć Windows Update, OneDrive i Defender (przyczyny nie znam,
   to tylko najczęstsi podejrzani).
2. **Lokalnie z TLS mierzymy generator**, co pokazały teraz oba pilotaże A2 i kalibracja A1 C2. Lokalna macierz C0–C5
   ma sens tylko przy ruchu, przy którym host nie jest nasycony z TLS, a do tego trzeba wybrać wspólny niższy RPS
   (np. na podstawie A1 C2: GraphQL O3 nasyca się na aplikacji; O1 C2 2800–3600 RPS to limit hosta). Albo AWS.
   To decyzja, którą warto podjąć z promotorem.
3. **`ATTACK_RATE_D_GRAPHQL=50`** (p. 5) — czeka na zgodę.
4. **Wystygnięcie w C/D:** sprawdzić w pierwszym przebiegu macierzy, czy 120 s wystarcza do powrotu po ataku w C0.
5. `plot_pilot.py` i `pilot_table.py` obsługują już wiele powtórzeń (commit `d445232`).

## 9. Pliki

- Podsumowania: `results/pilot/pilot_summary.md`, `results/pilot_var/pilot_summary.md`,
  `results/local_db3cpu/seria_a_saturation_O{1,3}_ramp_C2_{summary,windows}.csv`, `results/local_db3cpu/config_C2.txt`
- Statusy bloków: `results/blocks_logs/*.tsv`
- Wykresy: `analysis/plots/pilot_p95_overhead.png`, `analysis/plots/pilot_cpu_overhead.png` (2 powtórzenia)
- Surowe dane (poza git): `results/pilot_var/`, `results/pilot/`, `results/pilot_drate/`, `results/pilot_bcd2/`,
  `results/local_db3cpu/seria_a/*_C2_*`
