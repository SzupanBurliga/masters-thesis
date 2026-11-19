# Część 1: pilotaż serii B, C, D (test działania), 19.11.2026

Blok `pilot_bcd` (`loadtests/run_blocks.sh --max-hours 2.5 --pause 15 pilot_bcd`), 15:07:41–15:58, wyniki w
`results/pilot_bcd/` (117 MB, poza git), status w `results/blocks_logs/pilot_bcd.tsv`. W tym czasie laptop był
używany, więc liczby służą tylko do sprawdzenia działania, a nie do porównań. Liczby pochodzą z `_meta.json`,
`*_points.csv.gz` i `analysis/load.py`.

## 1. Co działa od początku do końca

- **24/24 przebiegi OK:** `k6_exit` 0, brak OOM i restartów, czas 94–127 s na przebieg (szacunek 120–150 s).
  `run_blocks.sh` nie musiał niczego pomijać ani wznawiać.
- **`analysis/load.py` przetwarza wszystkie trzy serie bez błędu.** Daje metryki legalnych w oknach baza / atak /
  powrót, metryki napastnika, `recovery_s` oraz CPU i RAM.
- **`vmstat` w każdym przebiegu:** `host_cpu_busy_mean/max` zapisane w 24/24 `_meta.json`.
- **Każdy mechanizm odrzuca swój atak, a ruch legalny przechodzi w 100%** (`base/attack_ok_share` = 1.0 we
  wszystkich 24 przebiegach). Rozkład kodów odpowiedzi napastnika (z `*_points.csv.gz`):

| seria | atak | C0 (bez ochrony) | z ochroną | kod odrzucenia |
|---|---|---|---|---|
| B | 1000 RPS z 1 tokenu, 60 s | 100% sukcesów (REST 60 000/60 000) | **C4:** REST 6187 × 200 + 53 166 × 429; GraphQL 6198 + 53 441 × 429; gRPC 6197 + 52 771 × kod 8 | 429 / `RESOURCE_EXHAUSTED` |
| C | zamówienie 100 KB, 20/s | przyjęte (REST 201, GraphQL 200, gRPC OK) | **C5:** REST 1201/1201, GraphQL 1201/1201 → 413; gRPC 929/929 → kod 8 | 413 / `RESOURCE_EXHAUSTED` |
| D | GraphQL głębokość 10, aliasy 50, `limit` 50 | 200 | **C5:** 601/601 każdy wariant → 422 | 422 |
| D | REST JSON zagnieżdżony 64 | 201 | **C5:** 600/600 → 400 | 400 |
| D | REST tablica 10 000, gRPC tablica 10 000 | 201 / OK | **C5:** REST 600/600 → 413, gRPC 601/601 → kod 8 | 413 / `RESOURCE_EXHAUSTED` |

  Rate limit przepuścił napastnikowi ok. 6190–6200 żądań w 60 s. Zgadza się to z ustawieniem kubełka
  (100 RPS × 60 s + 200 burst).
- **Ataki bez ochrony (C0) faktycznie szkodzą**, przynajmniej w serii C i w D-tablicy: p95 legalnych w fazie ataku
  REST C 16 323 ms, gRPC C 12 067 ms, REST D-tablica 26 773 ms, gRPC D-tablica 34 798 ms (bazowo 1,9–3,1 ms).
  Z ochroną (C5) p95 legalnych w fazie ataku: REST C 6,09 ms, GraphQL C 1,77 ms, REST D-tablica 11,30 ms.

## 2. Co się wysypało albo nie mierzy tego, co powinno

Żaden przebieg nie zakończył się błędem. Są natomiast problemy z **projektem scenariuszy i analizy**:

1. **Seria D, wariant GraphQL `limit` nie wzmacnia ataku.** Seed daje każdemu użytkownikowi dokładnie 5 zamówień
   (`app/cmd/seed/main.go:79`: `user_id = 1 + i % 10000`, 50 000 zamówień), więc `orders(limit:50)` zwraca 5,
   tak samo jak `limit:10`. W C0: p95 napastnika `limit50` 10,83 ms, a legalnych w fazie ataku 1,26 ms przy bazie
   1,40 ms. Model kosztu w `complexity.go` liczy ten atak jako 7601 i go odrzuca, ale realny koszt jest mały.
2. **Seria D, GraphQL przy 10 zapytaniach/s nie szkodzi nawet bez ochrony.** W C0 p95 legalnych w fazie ataku:
   głębokość 10 → 1,27 ms (baza 1,49), aliasy 50 → 5,52 ms (baza 1,33), `limit` 50 → 1,26 ms (baza 1,40).
   Nie da się więc pokazać, przed czym chronią limity złożoności.
3. **Seria C wyrównuje bajty, nie pracę.** 100 KB to w JSON 3314 pozycji zamówienia (30,894 B na pozycję), a w
   protobufie 14 628 pozycji (`GRPC_BYTES_PER_ITEM = 7`, `loadtests/lib/common.js`). W C0 gRPC wstawia więc do bazy
   ok. 4,4× więcej wierszy na żądanie niż REST/GraphQL, a komentarz w `common.js` twierdzi, że „wszystkie interfejsy
   wykonują N wstawień”. Do tego k6 nie nadąża z budową tak dużych wiadomości gRPC: napastnik gRPC w C0 wysłał
   **114 z 1200** zaplanowanych żądań, a CPU hosta w obu przebiegach gRPC C sięgało 99% (średnio 83,2 i 89,8%).
   Te przebiegi mierzą generator.
4. **Na najwyższym poziomie serii D (tablica 10 000) i w serii C odrzuca limit rozmiaru (C3), a nie limit złożoności (C5).**
   Tablica 10 000 pozycji to 317 808 B, czyli więcej niż limit 64 KiB, stąd 413 / kod 8 zamiast 400 / `InvalidArgument`
   z limitu długości tablicy. Poziomy 100 (3184 B) i 1000 (31 771 B) mieszczą się w 64 KiB, więc tam zadziała limit
   złożoności. W macierzy D na poziomie 10 000 konfiguracje C3 i C5 dadzą to samo. To nie błąd, ale trzeba to opisać.
5. **`recovery_s` często wychodzi NaN:**
   - seria B, REST C0 i C4: kryterium „p95 ≤ 1,1 × baza” przy bazie poniżej 1 ms (0,78 ms) jest węższe niż szum
     (powrót 1,025 ms > 0,858 ms), więc system „nigdy nie wraca”;
   - C i D w C0: `COOLDOWN` w pilotażu wynosił 10 s, a system po ataku jeszcze się nie podniósł (p95 w oknie powrotu
     REST C 23 009 ms, gRPC C 61 093 ms). W macierzy `COOLDOWN` to 120 s, więc to głównie skutek krótkiego pilotażu.
     Nie wiem jednak, czy 120 s wystarczy.
6. **Latencja napastnika liczona tylko z poprawnych odpowiedzi** (`load.py`, `_stats`): gdy mechanizm odrzuca 100%,
   `attacker_p95` = NaN. Nie widać więc, jak szybko mechanizm odrzuca, a to jest koszt ochrony.
7. **Okno bazowe w C i D to cała rozgrzewka od t = 0**, razem ze startem połączeń (np. gRPC C5: `base_p95` 35,23 ms).
8. `run_matrix.sh` nadal uznaje przebieg za gotowy po samym `.json`, który k6 zapisuje także po przerwaniu
   (wykazane wcześniej). `run_blocks.sh` i `run_a1_calibration.sh` używają już `_meta.json` z `k6_exit` 0.

## 3. Propozycje poprawek (do Twojej zgody, przed nocą)

| # | problem | propozycja | zmiana |
|---|---|---|---|
| P1 | D: `limit` bez efektu | Zapytanie od góry: `orders(limit:L){ user{ orders(limit:L){ items{…} } } }`, czyli L zamówień × 5 zamówień użytkownika × 4 pozycje. Realna praca rośnie liniowo z L, a dane zostają bez zmian. Alternatywa (nowy seed z „ciężkimi” użytkownikami) zmienia dane wszystkich serii i A1, więc jej nie polecam. | `lib/common.js` |
| P2 | D: GraphQL za słaby | Napastnik w D jako parametr `ATTACK_RATE` z domyślną wartością zależną od wariantu. Wartość trzeba wyznaczyć: proponuję w nocy krótki blok kalibracyjny (np. 10/50/100/200 zapytań/s dla C0) i wybrać najniższą, przy której p95 legalnych wyraźnie rośnie. | `run_blocks.sh`, `run_matrix.sh` |
| P3 | C: różna praca w interfejsach | **Decyzja do podjęcia** (najlepiej z promotorem): (a) ta sama liczba pozycji we wszystkich interfejsach, a rozmiar na drucie raportowany obok (polecam, bo pytanie dotyczy obciążenia serwera); (b) zostawić równe bajty i opisać różnicę. W obu przypadkach gRPC na laptopie jest ograniczony przez k6, więc seria C dla gRPC wymaga AWS albo niższego `BIG_RATE`. | `lib/common.js` |
| P4 | D/C: rozmiar zamiast złożoności | Opisać w pracy, że na poziomie 10 000 / 100 KB odrzuca C3. Opcjonalnie zmienić najwyższy poziom tablicy na 2000 (ok. 63,5 KB, nadal poniżej 64 KiB), żeby C5 był testowany na trzech poziomach. | opis albo `run_matrix.sh` |
| P5 | `recovery_s` przy bazie < 1 ms | Próg `max(1,1 × baza, baza + 1 ms)` w `recovery_time`. Wartość 1 ms trzeba opisać w metodzie jako arbitralny próg szumu. | `analysis/load.py` |
| P6 | brak latencji odrzuceń | Dodatkowe kolumny `*_all_p50/p95/p99` ze wszystkich odpowiedzi (też odrzuconych) obok obecnych. | `analysis/load.py` |
| P7 | okno bazowe od t = 0 | Baza w C/D = druga połowa rozgrzewki. | `analysis/load.py` |
| P8 | `run_matrix.sh` | Znacznik ukończenia jak w `run_blocks.sh` (`_meta.json` z `k6_exit` 0). | `loadtests/run_matrix.sh` |

P5–P8 to małe, mechaniczne zmiany bez wpływu na pomiar. P1 i P2 zmieniają scenariusz D, a P3 to decyzja metodyczna.
Po zmianach P1–P3 blok `pilot_bcd` z części 2 (pkt 4) warto puścić ponownie.

## 4. CPU hosta (vmstat, średnie / maks. % zajętości, laptop używany równolegle)

| seria | REST C0 / ochrona | GraphQL C0 / ochrona | gRPC C0 / ochrona |
|---|---|---|---|
| B (C0 / C4) | 34,4/47 · 67,4/89 | 30,4/41 · 47,2/71 | 52,0/63 · 63,8/82 |
| C (C0 / C5) | 49,2/77 · 60,2/69 | 48,0/76 · 32,2/38 | 83,2/99 · 89,8/99 |
| D tablica (C0 / C5) | 50,7/82 · 70,8/78 | — | 70,2/98 · 80,8/97 |
| D JSON 64 (C0 / C5) | 38,6/45 · 61,2/69 | — | — |
| D GraphQL głęb./aliasy/limit (C0) | — | 25,3/29 · 26,4/30 · 23,9/27 | — |
| D GraphQL głęb./aliasy/limit (C5) | — | 34,2/64 · 32,2/41 · 31,6/40 | — |
