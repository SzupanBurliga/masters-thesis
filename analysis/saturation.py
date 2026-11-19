"""Punkt nasycenia z przebiegów rampowych (A1): python analysis/saturation.py <env> <seria> [tag_glob] [konfiguracja=C0]

Dla każdego przebiegu liczy w oknach 10 s: RPS, odsetek poprawnych odpowiedzi, p50 i p95 (z poprawnych).
Okno uznajemy za "złe", gdy poprawnych < 99% albo p50 > max(5 * baza_p50, 10 ms); baza = mediana p50 z okien 2-4.
Kryterium po medianie, bo pojedyncze skoki p95 (ogon) nie oznaczają nasycenia; kolejka = trwały wzrost p50.
Punkt nasycenia = RPS pierwszego z NSUS kolejnych złych okien; brak = "nie nasycony" (sufit rampy przy zdrowej medianie).
Okna po przerwie > 60 s (zbłąkane znaczniki czasu) są odrzucane.
Wypisuje tabele i zapisuje <results>/<env>/<seria>_saturation_<tag>_windows.csv (okna) oraz _summary.csv (przebiegi);
dla konfiguracji innej niż C0 nazwy mają dopisek _<konfiguracja> (np. ..._O1_ramp_C2_summary.csv).
"""
import sys
from pathlib import Path

import numpy as np
import pandas as pd

RESULTS = Path(__file__).resolve().parent.parent / "results"
WIN = 10
NSUS = 3


def windows(path):
    cols = ["metric_name", "timestamp", "metric_value", "expected_response", "status"]
    d = pd.read_csv(path, usecols=cols, dtype=str)
    d = d[d.metric_name.isin(["http_req_duration", "grpc_req_duration"])].copy()
    d["t"] = d.timestamp.astype(int)
    d["ms"] = d.metric_value.astype(float)
    d["ok"] = (d.expected_response == "true") | ((d.metric_name == "grpc_req_duration") & (d.status == "0"))
    d["w"] = (d.t - d.t.min()) // WIN * WIN
    g = d.groupby("w").apply(lambda x: pd.Series({
        "rps": len(x) / WIN, "ok_frac": x.ok.mean(),
        "p50": x.ms[x.ok].median(), "p95": np.percentile(x.ms[x.ok], 95) if x.ok.any() else np.nan,
    }), include_groups=False)
    gap = g.index.to_series().diff().fillna(0) > 60
    if gap.any():
        g = g.iloc[: int(np.argmax(gap.values))]
    return g


def host_cpu_max(path):
    """Maks. zajętość CPU maszyny (100 - id) z vmstat -t; pierwsza linia danych (średnia od startu) pominięta."""
    vals = []
    for line in path.read_text().splitlines()[3:]:
        f = line.split()
        if len(f) >= 15 and f[14].isdigit():
            vals.append(100 - int(f[14]))
    return max(vals) if vals else np.nan


def first_sustained(bad):
    run = bad.rolling(NSUS).sum() == NSUS
    return int(np.argmax(run.values)) - NSUS + 1 if run.any() else None


def main():
    env, seria = sys.argv[1], sys.argv[2]
    tag = sys.argv[3] if len(sys.argv) > 3 else "ramp"
    cfg = sys.argv[4] if len(sys.argv) > 4 else "C0"
    suffix = "" if cfg == "C0" else f"_{cfg}"
    d = RESULTS / env / seria
    summ, allw = [], []
    for f in sorted(d.glob(f"*_{tag}_{cfg}_*_points.csv.gz")):
        iface, op, *_ , rep = f.name.replace("_points.csv.gz", "").split("_")
        g = windows(f)
        base = g.p50.iloc[1:4].median()
        lim = max(5 * base, 10)
        bad = (g.ok_frac < 0.99) | (g.p50 > lim)
        i = first_sustained(bad)
        first_bad = g.rps.iloc[i] if i is not None else np.nan
        good = g.iloc[:i] if i is not None else g
        host = d / f.name.replace("_points.csv.gz", "_hostcpu.txt")
        hostcpu = np.nan
        if host.exists():
            hostcpu = host_cpu_max(host)
        summ.append(dict(iface=iface, op=op, config=cfg, rep=rep, base_p50_ms=round(base, 2), p50_limit_ms=round(lim, 1),
                         max_healthy_rps=round(good.rps.max()), sat_rps=round(first_bad) if i is not None else np.nan, first_bad_rps=first_bad,
                         peak_rps=round(g.rps.max()), min_ok_frac=round(g.ok_frac.min(), 4),
                         host_cpu_max_pct=hostcpu))
        g = g.assign(iface=iface, op=op, rep=rep, bad=bad)
        allw.append(g.reset_index())
    s = pd.DataFrame(summ)
    print(s.to_string(index=False))
    print(s.groupby(["iface", "op"])[["max_healthy_rps", "sat_rps", "peak_rps"]].mean().round())
    s.to_csv(RESULTS / env / f"{seria}_saturation_{tag}{suffix}_summary.csv", index=False)
    pd.concat(allw).to_csv(RESULTS / env / f"{seria}_saturation_{tag}{suffix}_windows.csv", index=False)


if __name__ == "__main__":
    main()
