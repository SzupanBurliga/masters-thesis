#!/usr/bin/env bash
# run_blocks.sh [--max-hours H] [--pause S] [--dry-run] <blok> [<blok> ...]
#
# Uruchamia bloki przebiegów po kolei, w limicie czasu. Zasady:
# - przebieg gotowy = <wynik>_meta.json z "k6_exit":0 -> pomijany (wznawianie po Ctrl+C / restarcie: to samo polecenie);
# - przebieg, który nie zmieści się przed końcem limitu (szacunek = fazy k6 + OVERHEAD_S), nie jest zaczynany;
#   kolejne (krótsze) przebiegi są dalej sprawdzane;
# - błąd przebiegu nie przerywa całości (wyjątek: kod 3 = błąd przygotowania, np. wygasłe tokeny -> stop);
# - każdy blok ma osobny log results/blocks_logs/<blok>.log i status <blok>.tsv (dopisywane przy każdej sesji).
#
# Bloki: smoke, pilot_bcd, variability, variability_extra, pilot_a2_rep2, a1_tls
# Przykład: loadtests/run_blocks.sh --max-hours 2.5 --pause 15 pilot_bcd
set -uo pipefail
cd "$(dirname "$0")/.."
source loadtests/rates.env

MAX_HOURS=24; PAUSE=60; DRY=0; BLOCKS=()
while [ $# -gt 0 ]; do
  case "$1" in
    --max-hours) MAX_HOURS=$2; shift 2 ;;
    --pause) PAUSE=$2; shift 2 ;;
    --dry-run) DRY=1; shift ;;
    -h|--help) sed -n 2,14p "$0"; exit 0 ;;
    *) BLOCKS+=("$1"); shift ;;
  esac
done
[ ${#BLOCKS[@]} -gt 0 ] || { echo "podaj co najmniej jeden blok (--help)" >&2; exit 2; }
OVERHEAD_S=${OVERHEAD_S:-30}
T0=$(date +%s)
DEADLINE=$(( T0 + $(awk -v h="$MAX_HOURS" 'BEGIN{printf "%d", h*3600}') ))
LOGDIR=results/blocks_logs; mkdir -p "$LOGDIR"

sat() { local v="SAT_${1^^}_${2}"; echo "${!v}"; }
pct() { echo $(( $1 * $2 / 100 )); }

# ---- definicje bloków: każdy przebieg = add <plik_wyniku_bez_rozszerzenia> <szacunek_s> <polecenie> ----
RUN_OUT=(); RUN_EST=(); RUN_CMD=()
add() { RUN_OUT+=("$1"); RUN_EST+=("$2"); RUN_CMD+=("$3"); }
exp_cmd() { # env series iface op cfg rep rate [ZMIENNA=wartość ...] -> polecenie run_experiment.sh
  local e=$1 s=$2 i=$3 o=$4 c=$5 r=$6 rt=$7; shift 7
  echo "env $* loadtests/run_experiment.sh $e $s $i $o $c $r $rt"
}

define_block() {
  RUN_OUT=(); RUN_EST=(); RUN_CMD=()
  case "$1" in
    smoke) # test działania run_blocks.sh: 2 krótkie przebiegi
      local T="WARMUP=10s DURATION=30s COOLDOWN=5s MODE=const"
      add results/blocks_smoke/seria_a/rest_O1_C0_1 $((45 + OVERHEAD_S)) "$(exp_cmd blocks_smoke seria_a rest O1 C0 1 200 $T)"
      add results/blocks_smoke/seria_a/grpc_O1_C5_1 $((45 + OVERHEAD_S)) "$(exp_cmd blocks_smoke seria_a grpc O1 C5 1 200 $T)" ;;

    pilot_bcd) # test działania serii B, C, D: po 1 przebiegu, krótkie czasy, C0 (bez ochrony) i konfiguracja z mechanizmem
      local i c
      for i in rest graphql grpc; do  # B: legalni 30% SAT O1 + napastnik 1000 RPS (jeden token); C4 = rate limit
        for c in C0 C4; do
          add "results/pilot_bcd/seria_b/${i}_O1_atk1000_${c}_1" $((30 + 60 + 30 + OVERHEAD_S)) \
            "$(exp_cmd pilot_bcd seria_b "$i" O1 "$c" 1 "$(pct "$(sat "$i" O1)" 30)" TAG=O1_atk1000 LEGIT_RATE="$(pct "$(sat "$i" O1)" 30)" ATTACK_RATE=1000 LEGIT_ONLY=30 ATTACK_DUR=60 RECOVER=30)"
        done
      done
      local T="WARMUP=20s DURATION=60s COOLDOWN=10s"
      for i in rest graphql grpc; do  # C: ładunek 100 KB (> limit 64 KiB), 20/s + tło O1 30% SAT; C5 = limity
        for c in C0 C5; do
          add "results/pilot_bcd/seria_c/${i}_s102400_${c}_1" $((90 + OVERHEAD_S)) \
            "$(exp_cmd pilot_bcd seria_c "$i" O1 "$c" 1 "$(pct "$(sat "$i" O1)" 30)" $T TAG=s102400 SIZE=102400 BG_RATE="$(pct "$(sat "$i" O1)" 30)")"
        done
      done
      local spec v lvl var
      for spec in graphql:depth:10 graphql:alias:50 graphql:limit:50 rest:json:64 rest:array:10000 grpc:array:10000; do
        IFS=: read -r i var lvl <<<"$spec"; v=$var; [ "$v" = json ] && v=depth
        for c in C0 C5; do  # D: najwyższy poziom każdego wariantu, 10/s + tło O1 30% SAT
          add "results/pilot_bcd/seria_d/${i}_${var}${lvl}_${c}_1" $((90 + OVERHEAD_S)) \
            "$(exp_cmd pilot_bcd seria_d "$i" O1 "$c" 1 "$(pct "$(sat "$i" O1)" 30)" $T TAG="${var}${lvl}" VARIANT="$v" LEVEL="$lvl" BG_RATE="$(pct "$(sat "$i" O1)" 30)")"
        done
      done ;;

    variability|variability_extra) # zmienność: REST O1 C0, stały ruch 70% SAT, jak pilotaż A2
      local r reps; [ "$1" = variability ] && reps="1 2 3 4 5" || reps="6 7 8 9 10"
      for r in $reps; do
        add "results/pilot_var/seria_a/rest_O1_C0_${r}" $((180 + OVERHEAD_S)) \
          "$(exp_cmd pilot_var seria_a rest O1 C0 "$r" "$(pct "$SAT_REST_O1" 70)" WARMUP=30s DURATION=120s COOLDOWN=30s MODE=const)"
      done ;;

    pilot_a2_rep2) # drugie powtórzenie pilotażu A2, kolejność losowana z innym ziarnem, zapisana w results/pilot/order_rep2.txt
      local ord=results/pilot/order_rep2.txt n i c
      [ -s "$ord" ] || python3 -I -c "
import random
runs=[(i,c) for i in ('rest','graphql','grpc') for c in ('C0','C2','C4','C5')]
random.Random(20261120).shuffle(runs)
print('# seed=20261120; kolejność: nr iface cfg')
for n,(i,c) in enumerate(runs,1): print(n,i,c)" > "$ord"
      while read -r n i c; do
        add "results/pilot/seria_a/${i}_O1_${c}_2" $((180 + OVERHEAD_S)) \
          "$(exp_cmd pilot seria_a "$i" O1 "$c" 2 "$(pct "$(sat "$i" O1)" 70)" WARMUP=30s DURATION=120s COOLDOWN=30s MODE=const)"
      done < <(grep -v '^#' "$ord") ;;

    a1_tls) # kalibracja A1 z TLS + JWT (C2), kolejność jak w A1 C0: rep -> op -> iface
      local r o i dur
      for r in 1 2 3; do for o in O1 O3; do for i in rest graphql grpc; do
        [ "$o" = O1 ] && dur=720 || dur=360
        add "results/local_db3cpu/seria_a/${i}_${o}_ramp_C2_${r}" $((dur + OVERHEAD_S)) \
          "env CFG=C2 IFACES=$i OPS=$o REPS=$r PAUSE=0 loadtests/run_a1_calibration.sh local_db3cpu"
      done; done; done ;;

    *) echo "nieznany blok: $1" >&2; return 1 ;;
  esac
}

done_run() { grep -qs '"k6_exit":0' "$1_meta.json"; }
fmt_t() { date -d "@$1" '+%F %T'; }

STOP=0
trap 'echo "przerwano (Ctrl+C) — wznowienie: to samo polecenie" >&2; STOP=1' INT TERM

echo "start $(fmt_t "$T0"), limit ${MAX_HOURS} h (do $(fmt_t "$DEADLINE")), pauza ${PAUSE} s, bloki: ${BLOCKS[*]}"
for B in "${BLOCKS[@]}"; do
  define_block "$B" || exit 2
  LOG=$LOGDIR/$B.log; ST=$LOGDIR/$B.tsv
  [ -s "$ST" ] || printf "czas\tprzebieg\tstatus\tkod\tczas_s\tszac_s\n" > "$ST"
  total=0; todo=0
  for k in "${!RUN_OUT[@]}"; do done_run "${RUN_OUT[$k]}" || { total=$((total + RUN_EST[k])); todo=$((todo + 1)); }; done
  msg="== blok $B: ${#RUN_OUT[@]} przebiegów, do zrobienia $todo, szacunek $((total / 60)) min"
  echo "$msg"
  if [ "$DRY" = 1 ]; then
    for k in "${!RUN_OUT[@]}"; do
      done_run "${RUN_OUT[$k]}" && s=gotowy || s="$((RUN_EST[k]))s"
      printf "   %-60s %s\n" "${RUN_OUT[$k]#results/}" "$s"
    done
    continue
  fi
  echo "$(date '+%F %T') sesja start (limit do $(fmt_t "$DEADLINE")) $msg" >> "$LOG"
  for k in "${!RUN_OUT[@]}"; do
    [ "$STOP" = 1 ] && break 2
    out=${RUN_OUT[$k]}; est=${RUN_EST[$k]}; name=${out#results/}
    if done_run "$out"; then continue; fi
    now=$(date +%s)
    if (( now + est > DEADLINE )); then
      echo "$(date '+%F %T') POMINIĘTY (nie zmieści się: szac. ${est}s, zostało $((DEADLINE - now))s) $name" | tee -a "$LOG"
      printf "%s\t%s\tPOMINIĘTY_CZAS\t-\t-\t%s\n" "$(date -Iseconds)" "$name" "$est" >> "$ST"
      continue
    fi
    echo "$(date '+%F %T') START $name (szac. ${est}s)" | tee -a "$LOG"
    t=$(date +%s)
    bash -c "${RUN_CMD[$k]}" >> "$LOG" 2>&1
    rc=$?
    dt=$(( $(date +%s) - t ))
    if [ "$rc" = 0 ] && done_run "$out"; then status=OK; elif [ "$rc" = 130 ] || [ "$STOP" = 1 ]; then status=PRZERWANY; else status=BŁĄD; fi
    echo "$(date '+%F %T') $status (kod $rc, ${dt}s) $name" | tee -a "$LOG"
    printf "%s\t%s\t%s\t%s\t%s\t%s\n" "$(date -Iseconds)" "$name" "$status" "$rc" "$dt" "$est" >> "$ST"
    [ "$status" = PRZERWANY ] && break 2
    if [ "$rc" = 3 ]; then echo "kod 3 = błąd przygotowania (np. tokeny) — zatrzymuję wszystko" | tee -a "$LOG"; break 2; fi
    [ "$STOP" = 1 ] && break 2
    sleep "$PAUSE" & wait $!
  done
  echo "$(date '+%F %T') sesja koniec bloku $B" >> "$LOG"
done
echo "koniec $(date '+%F %T')"
