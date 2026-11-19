"""Tabela pilotażu A2: RPS, p50/p95/p99, błędy, CPU/RAM aplikacji w fazie pomiaru, narzut względem C0.

Użycie: ~/venv-thesis/bin/python analysis/pilot_table.py [env=pilot]
Wynik: results/<env>/pilot_summary.csv i results/<env>/pilot_summary.md

Źródła (nic nie jest szacowane):
- latencje i liczba żądań: *_points.csv.gz (k6), tylko scenariusz "measure" i okno [WARMUP, WARMUP+DURATION];
  percentyle z odpowiedzi poprawnych (jak w load.py), błędy = 1 - udział poprawnych;
- osiągnięty RPS = liczba żądań w oknie / DURATION;
- CPU/RAM: próbki docker stats z *_resources.csv w oknie [start+WARMUP, start+WARMUP+DURATION],
  gdzie start = pole "start" z *_meta.json (zapisywane tuż przed uruchomieniem k6).
"""
import json
import sys
from datetime import datetime
from pathlib import Path

import numpy as np
import pandas as pd

sys.path.insert(0, str(Path(__file__).resolve().parent))
from load import RESULTS, _secs, _stats, _window, load_points  # noqa: E402

IFACES = ["rest", "graphql", "grpc"]
CFGS = ["C0", "C2", "C4", "C5"]


def _mib(s):
    s = str(s)
    for unit, f in (("GiB", 1024), ("MiB", 1), ("KiB", 1 / 1024), ("B", 1 / 1024 / 1024)):
        if s.endswith(unit):
            return float(s[: -len(unit)]) * f
    return np.nan


def resources_window(path, t0, t1):
    df = pd.read_csv(path)
    df = df[(df["ts"] >= t0) & (df["ts"] < t1)]
    cpu = df["cpu_perc"].str.rstrip("%").astype(float)
    mem = df["mem_used"].map(_mib)
    return {"res_samples": len(df), "cpu_mean": cpu.mean(), "cpu_max": cpu.max(),
            "mem_mean_mib": mem.mean(), "mem_max_mib": mem.max()}


def main(env="pilot"):
    rows = []
    for meta_path in sorted(Path(RESULTS, env, "seria_a").glob("*_meta.json")):
        base = str(meta_path)[: -len("_meta.json")]
        meta = json.loads(meta_path.read_text())
        W, D = _secs(meta["params"]["WARMUP"]), _secs(meta["params"]["DURATION"])
        pts = load_points(base + "_points.csv.gz")
        st = _stats(_window(pts, W, W + D, "measure"), "m")
        start = datetime.fromisoformat(meta["start"]).timestamp()
        row = {"iface": meta["iface"], "config": meta["config"], "rate": meta["rate"], "k6_exit": meta["k6_exit"],
               "oom": meta["oom_killed"], "rps": st["m_n"] / D, "p50_ms": st["m_p50"], "p95_ms": st["m_p95"],
               "p99_ms": st["m_p99"], "err_pct": (1 - st["m_ok_share"]) * 100, "n": st["m_n"]}
        row.update(resources_window(Path(base + "_resources.csv"), start + W, start + W + D))
        rows.append(row)
    df = pd.DataFrame(rows)
    for col, out in (("p95_ms", "p95_vs_C0_pct"), ("cpu_mean", "cpu_vs_C0_pct")):
        c0 = df[df["config"] == "C0"].set_index("iface")[col]
        df[out] = df.apply(lambda r: (r[col] / c0.get(r["iface"], np.nan) - 1) * 100, axis=1)
    df["iface"] = pd.Categorical(df["iface"], IFACES, ordered=True)
    df["config"] = pd.Categorical(df["config"], CFGS, ordered=True)
    df = df.sort_values(["iface", "config"]).reset_index(drop=True)
    out = Path(RESULTS, env)
    df.to_csv(out / "pilot_summary.csv", index=False)

    fmt = lambda v, d=2: "—" if pd.isna(v) else f"{v:.{d}f}"  # noqa: E731
    lines = ["| interfejs | konf. | zadany RPS | osiągnięty RPS | p50 [ms] | p95 [ms] | p99 [ms] | błędy [%] | CPU śr. [%] | CPU max [%] | RAM śr. [MiB] | RAM max [MiB] | p95 vs C0 [%] | CPU vs C0 [%] |",
             "|---|---|---|---|---|---|---|---|---|---|---|---|---|---|"]
    for _, r in df.iterrows():
        lines.append(f"| {r.iface} | {r.config} | {r.rate} | {fmt(r.rps, 1)} | {fmt(r.p50_ms)} | {fmt(r.p95_ms)} | {fmt(r.p99_ms)} | "
                     f"{fmt(r.err_pct, 3)} | {fmt(r.cpu_mean, 1)} | {fmt(r.cpu_max, 1)} | {fmt(r.mem_mean_mib, 1)} | {fmt(r.mem_max_mib, 1)} | "
                     f"{'—' if r.config == 'C0' else fmt(r.p95_vs_C0_pct, 1)} | {'—' if r.config == 'C0' else fmt(r.cpu_vs_C0_pct, 1)} |")
    (out / "pilot_summary.md").write_text("\n".join(lines) + "\n")
    print("\n".join(lines))
    print(f"\nPróbki docker stats w oknie pomiaru: min {df.res_samples.min()}, max {df.res_samples.max()}; "
          f"k6_exit: {sorted(df.k6_exit.unique())}; OOM: {sorted(df.oom.unique())}")


if __name__ == "__main__":
    main(*(sys.argv[1:2] or ["pilot"]))
