"""Wczytywanie wyników przebiegów z results/<env>/<seria>/ do jednej ramki danych."""
import json
import sys
from pathlib import Path

import numpy as np
import pandas as pd

RESULTS = Path(__file__).resolve().parent.parent / "results"


def _trend(metrics, name):
    m = metrics.get(name) or {}
    return {k: m.get(k) for k in ("avg", "med", "p(90)", "p(95)", "p(99)", "max")}


def _resources(path):
    if not path.exists():
        return {"cpu_mean": None, "cpu_max": None, "mem_max_mib": None}
    df = pd.read_csv(path)
    if df.empty:
        return {"cpu_mean": None, "cpu_max": None, "mem_max_mib": None}
    cpu = df["cpu_perc"].str.rstrip("%").astype(float)

    def mib(s):
        s = str(s)
        for unit, f in (("GiB", 1024), ("MiB", 1), ("KiB", 1 / 1024)):
            if s.endswith(unit):
                return float(s[: -len(unit)]) * f
        return float("nan")

    return {"cpu_mean": cpu.mean(), "cpu_max": cpu.max(), "mem_max_mib": df["mem_used"].map(mib).max()}


LAT = ("http_req_duration", "grpc_req_duration")
# domyślne osie czasu serii B (sekundy), gdy meta nie zawiera nadpisania
B_DEFAULTS = {"LEGIT_ONLY": 120, "ATTACK_DUR": 300, "RECOVER": 180}


def _secs(v, default=None):
    """'90s' | '10m' | '1h' | '120' -> sekundy."""
    if v in (None, ""):
        return default
    v = str(v)
    mult = {"s": 1, "m": 60, "h": 3600}
    return int(v[:-1]) * mult[v[-1]] if v[-1] in mult else int(v)


def load_points(path):
    """Czyta *_points.csv.gz (k6): tylko latencje; zwraca t (s od startu), ms, ok, scenario.

    ok = odpowiedź poprawna: expected_response=true (HTTP) lub status=0 (gRPC). Błędy, 4xx (w tym 429)
    i odrzucenia nie wchodzą do percentyli (zasada 11.3 planu).
    """
    path = Path(path)
    if not path.exists():
        return None
    parts = []
    cols = ["metric_name", "timestamp", "metric_value", "expected_response", "scenario", "status"]
    for chunk in pd.read_csv(path, usecols=cols, dtype=str, chunksize=500_000):
        parts.append(chunk[chunk["metric_name"].isin(LAT)])
    if not parts:
        return None
    df = pd.concat(parts, ignore_index=True)
    if df.empty:
        return None
    t = df["timestamp"].astype(float)
    ok = df["expected_response"].map({"true": True, "false": False})
    ok = ok.where(ok.notna(), df["status"] == "0").astype(bool)
    out = pd.DataFrame({"t": t - t.min(), "ms": df["metric_value"].astype(float), "ok": ok, "scenario": df["scenario"]})
    return out


def _window(pts, t0, t1, scenario=None):
    m = (pts["t"] >= t0) & (pts["t"] < t1)
    if scenario is not None:
        m &= pts["scenario"] == scenario
    return pts[m]


def _stats(w, prefix):
    good = w[w["ok"]]["ms"]
    row = {f"{prefix}_n": len(w), f"{prefix}_ok_share": float(w["ok"].mean()) if len(w) else np.nan}
    for q in (50, 95, 99):
        row[f"{prefix}_p{q}"] = float(np.percentile(good, q)) if len(good) else np.nan
    return row


def recovery_time(pts, scenario, t_end_attack, baseline_p95, step=10, factor=1.10):
    """Czas (s) od końca ataku do pierwszego okna `step` s, w którym p95 <= baza * factor. NaN gdy nigdy."""
    if not np.isfinite(baseline_p95):
        return np.nan
    t = t_end_attack
    tmax = pts["t"].max()
    while t < tmax:
        w = _window(pts, t, t + step, scenario)
        good = w[w["ok"]]["ms"]
        if len(good) and np.percentile(good, 95) <= baseline_p95 * factor:
            return t - t_end_attack
        t += step
    return np.nan


def point_metrics(pts, meta):
    """Metryki z okien czasowych, zależnie od serii. Prefiksy: measure_ (A), base_/attack_/recover_ (B-D)."""
    series, params = meta["series"], meta["params"]
    W, D, C = (_secs(params.get(k)) for k in ("WARMUP", "DURATION", "COOLDOWN"))
    if series == "seria_a":
        if meta.get("tag", "").endswith("_ramp"):
            return {}
        return _stats(_window(pts, W, W + D, "measure"), "measure")
    row = {}
    if series == "seria_b":
        base_end = _secs(params.get("LEGIT_ONLY"), B_DEFAULTS["LEGIT_ONLY"])
        att_end = base_end + _secs(params.get("ATTACK_DUR"), B_DEFAULTS["ATTACK_DUR"])
    else:  # C, D: tło od początku, atak w fazie pomiarowej
        base_end, att_end = W, W + D
    # pierwsze 20% bazy odrzucamy jako rozgrzewkę legalnego ruchu (B); w C/D baza = rozgrzewka właściwa
    base_start = 0.25 * base_end if series == "seria_b" else 0
    row.update(_stats(_window(pts, base_start, base_end, "legit"), "base"))
    row.update(_stats(_window(pts, base_end, att_end, "legit"), "attack"))
    row.update(_stats(_window(pts, att_end, pts["t"].max() + 1, "legit"), "recover"))
    atk = [s for s in pts["scenario"].unique() if s not in ("legit",)]
    if atk:
        row.update(_stats(_window(pts, base_end, att_end, atk[0]), "attacker"))
    row["recovery_s"] = recovery_time(pts, "legit", att_end, row.get("base_p95", np.nan))
    return row


def load_series(results_dir, env, series):
    rows = []
    for meta_path in sorted(Path(results_dir, env, series).glob("*_meta.json")):
        base = str(meta_path)[: -len("_meta.json")]
        meta = json.loads(meta_path.read_text())
        summary_path = Path(base + ".json")
        if not summary_path.exists():
            continue
        metrics = json.loads(summary_path.read_text())["metrics"]
        row = {k: meta[k] for k in ("env", "series", "iface", "op", "tag", "config", "rep", "rate", "oom_killed", "restart_count", "k6_exit")}
        row.update({f"param_{k}": v for k, v in meta["params"].items()})
        # UWAGA: poniższe percentyle z podsumowania k6 obejmują także odpowiedzi błędne i wszystkie fazy;
        # do pracy używaj kolumn z load_points (measure_*, base_*, attack_*, recover_*).
        # opóźnienia: dla serii A faza pomiarowa; dla B-D osobno legalni i napastnik
        for label, suffixes in (("measure", ["{phase:measure}"]), ("legit", ["{who:legit}"]), ("attacker", ["{who:attacker}"])):
            for proto, mname in (("http", "http_req_duration"), ("grpc", "grpc_req_duration")):
                for suf in suffixes:
                    t = _trend(metrics, mname + suf)
                    if t["avg"] not in (None, 0):
                        row.update({f"{label}_{k}": v for k, v in t.items()})
        row["http_failed_rate"] = (metrics.get("http_req_failed") or {}).get("value")
        row["checks_rate"] = (metrics.get("checks") or {}).get("value")
        row["iterations"] = (metrics.get("iterations") or {}).get("count")
        pts = load_points(base + "_points.csv.gz")
        if pts is not None:
            row.update(point_metrics(pts, meta))
        row.update(_resources(Path(base + "_resources.csv")))
        rows.append(row)
    return pd.DataFrame(rows)


if __name__ == "__main__":
    env, series = (sys.argv[1:3] + ["local", "seria_a"])[:2] if len(sys.argv) < 3 else sys.argv[1:3]
    df = load_series(RESULTS, env, series)
    pd.set_option("display.width", 200)
    print(df.drop(columns=[c for c in df.columns if c.startswith("param_")]).to_string())
