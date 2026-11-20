# Raport z nocy 19/20.11.2026 (część 2)

Polecenie: `loadtests/run_blocks.sh --max-hours 8 --pause 60 variability pilot_a2_rep2 d_rate_cal pilot_bcd2 a1_tls variability_extra`,
start 23:12:17, koniec 05:27:56, limit do 07:12:17. Kod: commit `df68ba7`, czyste drzewo. Laptop (WSL2, 8 wątków):
k6, aplikacja i baza na jednej maszynie.

Źródła liczb: `results/blocks_logs/*.{log,tsv}`, `_meta.json`, `*_points.csv.gz`, `*_resources.csv`, `*_hostcpu.txt`
(vmstat). Wyliczenia z plików robią `analysis/pilot_table.py`, `analysis/load.py`, `analysis/saturation.py` i
`analysis/stats.py`. Wszystko, czego nie dało się ustalić z plików, jest oznaczone „nie wiem”.

---

## 1. Co się wykonało

| blok | start | koniec ostatniego przebiegu | OK | błędne | pominięte przez limit | suma czasu przebiegów / szacunek [s] |
|---|---|---|---|---|---|---|
| variability | 23:12:17 | 23:32:11 | 5 | 0 | 0 | 953 / 1050 |
| pilot_a2_rep2 | 23:33:11 | 00:22:23 | 12 | 0 | 0 | 2292 / 2520 |
| d_rate_cal | 00:23:23 | 00:54:13 | 12 | 0 | 0 | 1190 / 1440 |
| pilot_bcd2 | 00:55:13 | 02:01:52 | 24 | 0 | 0 | 2619 / 3060 |
| a1_tls | 02:02:52 | 05:06:06 | 18 | 0 | 0 | 9974 / 10 260 |
| variability_extra | 05:07:06 | 05:26:56 | 5 | 0 | 0 | 950 / 1050 |
| **razem** | | | **76** | **0** | **0** | |

Każdy blok miał jedną sesję, bez przerwań i wznowień. We wszystkich 76 `_meta.json`: `k6_exit` 0, `oom_killed` false,
`restart_count` 0. Każdy przebieg trwał krócej niż jego szacunek.

## 2. CPU generatora (vmstat; na laptopie host = generator k6 + kontenery)

`host_cpu_busy_mean` / `host_cpu_busy_max` z `_meta.json` (100 − `id` z vmstat co 5 s, faza k6).

| blok | n | średnia przebiegu: min / mediana / max [%] | maks. chwilowe: min / max [%] | przebiegi z maks. ≥ 95% |
|---|---|---|---|---|
| variability | 5 | 70,4 / 71,8 / 81,6 | 77 / 90 | 0 |
| variability_extra | 5 | 70,6 / 71,1 / 71,5 | 77 / 80 | 0 |
| pilot_a2_rep2 | 12 | 47,2 / 82,3 / 94,9 | 53 / 99 | 3 (REST C2, C4, C5) |
| d_rate_cal | 12 | 25,4 / 38,3 / 45,4 | 30 / 73 | 0 |
| pilot_bcd2 | 24 | 25,6 / 48,3 / 76,1 | 30 / 94 | 0 |
| a1_tls | 18 | 63,1 / 77,95 / 81,5 | 80 / 99 | 13 |

**Gdzie generator był wąskim gardłem.** Kryterium: osiągnięty ruch wyraźnie poniżej zadanego przy CPU aplikacji
poniżej jej limitu 200% i CPU hosta bliskim 100%.
- **pilot_a2_rep2, REST C2/C4/C5:** CPU hosta średnio 94,4–94,9%, maks. 98–99%; osiągnięte 66,8–71,0% zadanego RPS;
  CPU aplikacji średnio 101,1–105,3%. **Tak.**
- **pilot_a2_rep2, gRPC C2/C4/C5:** CPU hosta średnio 89,5–90,0%, maks. 94%; osiągnięte 93,7–94,2% zadanego RPS;
  CPU aplikacji średnio 118,6–120,8%. **Prawdopodobnie tak**, ale słabiej niż w REST. Pewności nie mam.
- **a1_tls, O1 (wszystkie interfejsy):** maks. CPU hosta 98–99% w punkcie nasycenia. **Tak** (patrz p. 5).
- **a1_tls, REST O3 i gRPC O3:** maks. 93–97%. Nie wiem, czy limitem był host, czy aplikacja z bazą.
- **a1_tls, GraphQL O3:** maks. 80–81%. **Nie**, limitem była aplikacja.
- variability, d_rate_cal, pilot_bcd2: maks. ≤ 94% i (poza zakłóceniami, p. 3) osiągany zadany ruch. Nie.

## 3. Zmienność: REST O1 C0, stały ruch 4900 RPS, 10 powtórzeń

`results/pilot_var/pilot_summary.md`. Faza pomiaru 120 s, percentyle z poprawnych odpowiedzi.

| powt. | osiągnięty RPS | p50 [ms] | p95 [ms] | p99 [ms] | CPU hosta śr./maks. [%] |
|---|---|---|---|---|---|
| 1 | **3997,48** | 8,78 | **181,05** | 271,14 | 81,4 / 90 |
| 2 | 4854,63 | 1,48 | 12,08 | 24,12 | 70,6 / 77 |
| 3 | 4887,00 | 1,46 | 10,51 | 20,59 | 71,8 / 86 |
| 4 | **4201,50** | 11,23 | **162,51** | 466,58 | 81,6 / 90 |
| 5 | 4883,88 | 1,46 | 11,26 | 22,09 | 70,4 / 77 |
| 6 | 4871,73 | 1,53 | 12,83 | 25,34 | 71,5 / 79 |
| 7 | 4875,88 | 1,52 | 11,93 | 24,68 | 70,9 / 77 |
| 8 | 4852,50 | 1,48 | 11,74 | 22,38 | 70,6 / 77 |
| 9 | 4849,92 | 1,49 | 11,71 | 22,57 | 71,3 / 80 |
| 10 | 4842,02 | 1,52 | 12,79 | 26,28 | 71,1 / 78 |

Błędy: 0 we wszystkich powtórzeniach.

**Współczynnik zmienności p95** (`stats.cv`: odchylenie standardowe z próby / średnia) i **przedział ufności** średniego
p95 (95%, rozkład t-Studenta; dla porównania bootstrap, 10 000 losowań, ziarno 42):

| zbiór | n | średnia p95 [ms] | CV | półszerokość CI (t) | CI bootstrap [ms] | n potrzebne do ±5% (t) |
|---|---|---|---|---|---|---|
| wszystkie | 10 | 43,84 | 154,14% | ±110,26% | 11,70–89,26 (±88,46%) | nie do osiągnięcia rozsądną liczbą |
| bez powt. 1 i 4 | 8 | 11,86 | 6,42% | ±5,37% | 11,35–12,33 (±4,15%) | **9** |
| powt. 6–10 | 5 | 12,20 | 4,60% | ±5,72% | 11,77–12,63 (±3,55%) | **6** |

„n potrzebne” = najmniejsze n, dla którego t(0,975; n−1) × CV / √n ≤ 5%, przy założeniu, że CV jest taki jak w danym
zbiorze. **Odpowiedź: 6–9 powtórzeń**, ale tylko wtedy, gdy przebiegi z zakłóceniami są wykrywane i powtarzane.
Z nimi żadna rozsądna liczba powtórzeń nie da ±5% dla średniej. Mediana p95 z 10 powtórzeń wynosi 12,01 ms, więc
mediana jest na takie zakłócenia odporna.

**Powtórzenia 1 i 4 to epizody, nie stałe przesunięcie.** W oknach 10 s fazy pomiaru:
- powt. 1: RPS 4667 i 4891, potem 2287 i 1475, a p95 195,6–306,0 ms;
- powt. 4: p95 rośnie od 41,4 do 839,3 ms, RPS spada do 2327.

W powt. 4 vmstat pokazuje `us` 58–60% przy 43–46% w powt. 2 przy tym samym ruchu. Coś zużywało więcej CPU, ale
**nie wiem co**: w plikach nie ma śladu innego procesu. Oba epizody wypadły w pierwszych 17 minutach nocy (23:12–23:29).

## 4. Porównanie obu pilotaży A2

`analysis/pilot_table.py` obsługuje teraz wiele powtórzeń: kolumna `rep`, narzut względem C0 tego samego powtórzenia.
Wynik: `results/pilot/pilot_summary.md`. Powt. 1 z 19.11 (13:27–14:16, laptop wolny), powt. 2 z tej nocy.
% zadanego = osiągnięty RPS / zadany RPS.

| interfejs | konf. | % zadanego RPS (1 / 2) | p95 [ms] (1 / 2) | CPU aplikacji śr. [%] (1 / 2) | CPU hosta śr./maks. powt. 2 [%] |
|---|---|---|---|---|---|
| REST | C0 | 98,8 / 98,8 | 18,17 / 16,02 | 111,3 / 118,5 | 71,6 / 79 |
| REST | C2 | 66,2 / 70,9 | 204,65 / 169,69 | 100,0 / 102,6 | 94,9 / 99 |
| REST | C4 | 72,2 / 66,8 | 173,53 / 178,43 | 99,6 / 105,3 | 94,4 / 98 |
| REST | C5 | 70,2 / 71,0 | 196,99 / 172,32 | 99,0 / 101,1 | 94,9 / 99 |
| GraphQL | C0 | 99,6 / 99,7 | 5,49 / 3,01 | 112,8 / 117,2 | 47,2 / 53 |
| GraphQL | C2 | 98,1 / 98,5 | 67,96 / 26,39 | 117,0 / 121,8 | 74,7 / 84 |
| GraphQL | C4 | 98,2 / 98,9 | 42,34 / 27,22 | 118,5 / 120,7 | 74,1 / 84 |
| GraphQL | C5 | 98,9 / 98,3 | 31,18 / 38,59 | 116,9 / 122,2 | 75,1 / 85 |
| gRPC | C0 | 99,0 / 99,4 | 8,76 / 7,86 | 123,1 / 127,9 | 67,0 / 74 |
| gRPC | C2 | 94,6 / 94,2 | 218,28 / 131,89 | 118,5 / 118,6 | 89,5 / 94 |
| gRPC | C4 | 94,4 / 93,7 | 171,75 / 128,08 | 115,6 / 120,8 | 89,5 / 94 |
| gRPC | C5 | 92,4 / 94,0 | 259,70 / 129,26 | 119,9 / 119,7 | 90,0 / 94 |

Błędy: 0,000% we wszystkich 24 przebiegach. Pierwszy pilotaż nie zapisywał vmstat, więc CPU hosta jest tylko dla
powtórzenia 2.

**Czy drugie powtórzenie potwierdza pierwsze?**
- **Tak, co do głównego wniosku:** C0 osiąga 98,8–99,7% zadanego ruchu w obu powtórzeniach. Z TLS REST osiąga tylko
  66–72%, gRPC 92–95%, a host jest bliski 100%. W obu powtórzeniach mierzymy generator, nie mechanizmy.
- **Nie, co do wielkości narzutu:** p95 tej samej konfiguracji różni się między powtórzeniami nawet 2,6× (GraphQL C2:
  67,96 vs 26,39 ms; gRPC C2: 218,28 vs 131,89 ms). Kolejność GraphQL C2 > C4 > C5 z pierwszego powtórzenia się nie
  potwierdziła: w drugim najwyższe p95 ma C5. Przy dwóch powtórzeniach różnic C2/C4/C5 nie da się rozstrzygnąć.
- Wykres `analysis/plots/pilot_p95_overhead.png`: słupek = mediana z 2 powtórzeń, wąsy = min–max.

## 5. Kalibracja A1 z TLS (C2) i bez (C0)

`results/local_db3cpu/seria_a_saturation_O{1,3}_ramp[_C2]_summary.csv`, to samo kryterium nasycenia
(`analysis/saturation.py`), 3 powtórzenia. Zmiana = mediana C2 względem mediany C0. CPU hosta = `host_cpu_max_pct`
z vmstat w przebiegu.

| iface | op | C0 (powt. 1/2/3) | C2 (powt. 1/2/3) | mediana C0 → C2 | zmiana | min C0 → C2 | CPU hosta maks. C0 / C2 [%] |
|---|---|---|---|---|---|---|---|
| REST | O1 | 6958 / 7302 / 7065 | **979** / 3587 / 3613 | 7065 → 3587 | −49,2% | 6958 → 979 | 90–92 / 98–99 |
| gRPC | O1 | 4497 / 4355 / 4709 | 2916 / 2927 / 2781 | 4497 → 2916 | −35,2% | 4355 → 2781 | 87–88 / 99 |
| GraphQL | O1 | 3773 / 3551 / 3771 | 3007 / 2984 / 3091 | 3771 → 3007 | −20,3% | 3551 → 2984 | 76–78 / 99 |
| REST | O3 | 2385 / 2330 / 2231 | 1805 / 1739 / 1614 | 2330 → 1739 | −25,4% | 2231 → 1614 | 85–86 / 93–95 |
| gRPC | O3 | 1657 / 1534 / 1727 | 1279 / 1281 / 1171 | 1657 → 1279 | −22,8% | 1534 → 1171 | 91–92 / 96–97 |
| GraphQL | O3 | 1071 / 1101 / 1073 | 1048 / 1033 / 1039 | 1073 → 1039 | **−3,2%** | 1071 → 1033 | 69–70 / 80–81 |

**Czy przy C2 limitem był generator?**
- **O1: tak, dla wszystkich interfejsów.** CPU hosta 98–99%, a punkty nasycenia trzech interfejsów zbiegają się do
  2781–3613 RPS. To limit laptopa (szyfrowanie po stronie k6 i serwera), więc „zmiana” w O1 nie jest narzutem TLS na
  interfejs, a ranking REST > gRPC > GraphQL z C0 tu nic nie mówi.
- **GraphQL O3: nie.** CPU hosta 80–81%, a limitem jest CPU aplikacji, jak w C0. **−3,2% (mediana) to jedyny lokalny
  pomiar narzutu TLS + JWT na przepustowość, któremu można ufać.**
- **REST O3, gRPC O3:** CPU hosta 93–97%. Nie wiem, jaka część spadku wynika z mechanizmów, a jaka z hosta.
- **REST O1 C2, powt. 1 (979 RPS) to zakłócenie, nie nasycenie.** Do okna 160 s wszystko w normie (1798,7 RPS, p50
  1,15 ms), w oknie 170 s RPS spada do 978,6, a p50 do 68,13 ms, i do końca rampy 56 z 73 okien jest „złych”, choć
  szczyt RPS później wynosi 3696. To ten sam wzorzec co w powt. 1 i 4 zmienności. Przyczyna: nie wiem. Dlatego obok
  mediany podaję minimum, które jest nim zaburzone.

## 6. Siła ataku GraphQL (d_rate_cal; C0, najwyższe poziomy, legalne tło 1065 RPS O1)

`results/pilot_drate/seria_d/`, `analysis/load.py`. p95 legalnych [ms]: baza (2. połowa rozgrzewki) → faza ataku.
W nawiasie: zrealizowane / zaplanowane żądania napastnika w 60 s.

| wariant | 10/s | 50/s | 100/s | 200/s |
|---|---|---|---|---|
| aliasy 50 | 1,19 → 5,49 (596/600) | 1,35 → **487,63** (2892/3000) | 1,24 → 3134,47 (3193/6000) | 1,26 → 3204,11 (3353/12 000) |
| `limit` 50 | 1,28 → 5,79 (599/600) | 1,24 → **1590,43** (2445/3000) | 1,24 → 3125,54 (2524/6000) | 1,31 → 3181,33 (2587/12 000) |
| głębokość 10 | 1,25 → 1,32 (596/600) | 1,29 → 1,34 (2983/3000) | 1,23 → 1,59 (5947/6000) | 1,22 → 3,31 (11 975/12 000) |

CPU aplikacji (śr. / maks. [%]): aliasy 74,83/103,30 → 149,18/199,03 → 150,64/195,88 → 151,04/194,76;
`limit` 76,76/110,53 → 155,41/202,12 → 159,83/201,32 → 158,66/196,79; głębokość 59,86/95,43 → 70,83/93,09 →
84,87/122,94 → 109,30/161,22. Błędy legalnych: 0 we wszystkich 12 przebiegach.

- **Ruch legalny wyraźnie cierpi od 50/s** przy aliasach i `limit`: p95 rośnie z ok. 1,3 ms do 488 i 1590 ms, a
  aplikacja dochodzi do limitu CPU. Przy 10/s p95 rośnie do 5,5–5,8 ms, czyli wyraźnie względnie, ale niewiele w ms.
  Powyżej 50/s aplikacja nie nadąża, generator realizuje coraz mniejszą część zaplanowanych żądań, a p95 legalnych
  zatrzymuje się na ok. 3,1–3,2 s.
- **Głębokość 10 nie szkodzi nawet przy 200/s** (p95 legalnych 3,31 ms): dataloadery obsługują łańcuch zagnieżdżeń stałą
  liczbą zapytań SQL. To wynik sam w sobie, nie błąd scenariusza.
- **Propozycja: `ATTACK_RATE_D_GRAPHQL=50`.** To najniższa z badanych wartości, przy której aliasy i `limit` wyraźnie
  szkodzą, a generator jeszcze realizuje 82–96% zaplanowanych żądań. Wartości między 10 a 50 nie mierzyłem.
  **`rates.env` jest bez zmian (nadal 10) i czeka na Twoją decyzję.**

## 7. Pilotaż B, C, D po poprawkach (pilot_bcd2)

`results/pilot_bcd2/`, 24/24 OK. Kody odpowiedzi napastnika z `*_points.csv.gz`.

| poprawka | czy zadziałała | dowód z plików |
|---|---|---|
| **P1** `limit` w D | **tak, zwiększa pracę** | `limit` 50, C0, 10/s: p50/p95 napastnika 23,70/32,18 ms (pilot_bcd: 7,92/10,83 ms); p95 legalnych w fazie ataku 5,71 ms przy bazie 1,25 (pilot_bcd: 1,26 przy bazie 1,50, przeliczone z P7). W d_rate_cal przy 50/s p95 legalnych 1590,43 ms. C5: 601/601 × 422. |
| **P2** siła ataku D | **parametr działa** | d_rate_cal użył 10/50/100/200 (p. 6); pilot_bcd2 GraphQL D celowo przy 10/s (`rates.env`). Wartość docelowa czeka na decyzję. |
| **P3** równe pozycje w C | **tak** | 3314 pozycji we wszystkich interfejsach; rozmiar: REST 104 963 B, GraphQL 105 051 B, gRPC 23 073 B. C5: REST 1201 × 413, GraphQL 1200 × 413, gRPC 1200 × kod 3. Napastnik gRPC w C0: 724 z 1200 żądań (pilot_bcd: 114). |
| **P4** tablica 2000 | **tak** | REST 63 600 B → C5 601 × **400**; gRPC 13 955 B → C5 600 × **kod 3** (limit złożoności, nie rozmiaru). |
| **P5** próg powrotu | **tak, czas powrotu już się liczy** | `recovery_s` = 0–10 s we wszystkich przebiegach B; w seriach C/D liczbą wszędzie poza tymi, gdzie system nie wrócił w 10 s wystygnięcia: C w C0 (p95 po ataku 25 299–33 783 ms) i gRPC tablica 2000 w C0 (196,90 ms przy bazie 2,94 ms). |
| **P6** latencja odrzuceń | **tak** | p50 wszystkich odpowiedzi napastnika przy odrzuceniu: 0,75–2,77 ms; gRPC C5 w serii C 10,11 ms. |
| **P7** okno bazowe | **działa, ale nie wyjaśnia wszystkiego** | baza = 2. połowa rozgrzewki. REST tablica 2000 C5 nadal ma `base_p95` 34,24 ms (C0: 1,56 ms). Dlaczego: nie wiem (1 przebieg). |
| **P8** znacznik ukończenia | **tak** (test 19.11) | przerwany Ctrl+C przebieg powtórzony przy wznowieniu (`results/blocks_logs/smoke2.tsv`). Tej nocy nic nie było przerywane. |

Seria B (bez zmian względem pilot_bcd) nadal działa: C4 przepuszcza napastnikowi 6144–6196 żądań z ok. 59 000,
reszta dostaje 429 albo kod 8. Ruch legalny we wszystkich 24 przebiegach: 100% poprawnych odpowiedzi.

Uwaga do P4: tablica 2000 w C0 jest dużo łagodniejszym atakiem niż 10 000 (p95 legalnych w fazie ataku REST 18,72 ms,
gRPC 62,83 ms, wobec 26 773 i 34 798 ms przy 10 000).

## 8. Podsumowanie

Noc przeszła bez awarii: wszystkie 76 przebiegów się wykonało, a skrypty, analiza i poprawki scenariuszy działają.
Pomiary bez szyfrowania są powtarzalne (zmienność p95 4,6–6,4%), ale w 3 z 16 ocenialnych przebiegów REST coś na
laptopie zakłóciło pomiar i nie umiem wskazać co, więc takie przebiegi trzeba wykrywać i powtarzać. Po włączeniu TLS laptop
nie nadąża z generowaniem ruchu, więc lokalnie nie da się zmierzyć kosztu zabezpieczeń dla REST i gRPC. Jedyny czysty
lokalny wynik to GraphQL O3: TLS i JWT obniżają tam przepustowość o ok. 3%.

**Do poprawy przed pomiarami w chmurze:**
1. **Wykrywanie zakłóconych przebiegów:** flaga w analizie, gdy w C0 osiągnięty RPS w fazie pomiaru jest < 95% zadanego,
   i automatyczne ponowienie; raportowanie mediany z ≥ 3 powtórzeń; dla przedziału ±5% planować 6–9 czystych powtórzeń
   tam, gdzie potrzebna jest średnia.
2. **Zdalny `run_experiment.sh`** (zmienna `SERVER`, docker stats i vmstat na serwerze, vmstat osobno na generatorze).
   Propozycja jest w `infra/aws/README.md`, jeszcze niezaimplementowana.
3. **Kalibracja A1 od nowa na AWS**, w C0 i C2, z vmstat generatora (punkty z laptopa się nie przeniosą).
4. **Decyzje:** `ATTACK_RATE_D_GRAPHQL` (proponuję 50), siła ataku w serii B (5×/10× nasycenia), czas wystygnięcia
   w C/D (10 s w pilotażu było za mało; czy 120 s wystarczy, nie wiem).
5. **Opis w pracy:** ten sam ładunek trafia w różne mechanizmy zależnie od formatu (JSON 105 KB → limit rozmiaru w C3;
   protobuf 23 KB → limit tablicy w C5); głębokie zapytania GraphQL są tanie dzięki dataloaderom.
6. Do wyjaśnienia na jednym przebiegu: wysokie `base_p95` w REST tablica 2000 C5 (34,24 ms).
7. Tag `v1.0-experiments` po wprowadzeniu powyższego; push 24 lokalnych commitów (po Twojej decyzji).
