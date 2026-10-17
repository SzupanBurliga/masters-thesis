"""Statystyki: przedziały ufności (bootstrap), narzut względem C0 / poprzedniej konfiguracji, testy istotności."""
import numpy as np
import pandas as pd
from scipy import stats

ORDER = ["C0", "C1", "C2", "C3", "C4", "C5"]


def bootstrap_ci(x, n=10000, alpha=0.05, seed=42):
    rng = np.random.default_rng(seed)
    x = np.asarray(x, dtype=float)
    means = rng.choice(x, size=(n, len(x)), replace=True).mean(axis=1)
    return np.percentile(means, [100 * alpha / 2, 100 * (1 - alpha / 2)])


def summarize(df, metric):
    """Średnia, odchylenie i 95% CI metryki dla każdej kombinacji (iface, op, config)."""
    rows = []
    for (iface, op, cfg), g in df.groupby(["iface", "op", "config"]):
        v = g[metric].dropna()
        if v.empty:
            continue
        lo, hi = bootstrap_ci(v) if len(v) > 1 else (v.iloc[0], v.iloc[0])
        rows.append({"iface": iface, "op": op, "config": cfg, "n": len(v), "mean": v.mean(), "std": v.std(ddof=1) if len(v) > 1 else 0.0, "ci_lo": lo, "ci_hi": hi})
    return pd.DataFrame(rows)


def overhead(df, metric):
    """Narzut % względem C0 oraz względem poprzedniej konfiguracji (kolejność C0..C5)."""
    s = summarize(df, metric)
    out = []
    for (iface, op), g in s.groupby(["iface", "op"]):
        g = g.set_index("config").reindex([c for c in ORDER if c in set(g["config"])])
        base = g.loc["C0", "mean"] if "C0" in g.index else np.nan
        prev = None
        for cfg, r in g.iterrows():
            out.append({"iface": iface, "op": op, "config": cfg, "mean": r["mean"],
                        "vs_C0_pct": (r["mean"] / base - 1) * 100,
                        "vs_prev_pct": (r["mean"] / prev - 1) * 100 if prev else np.nan})
            prev = r["mean"]
    return pd.DataFrame(out)


def significant(a, b, alpha=0.05):
    """Test Manna-Whitneya (dwustronny)."""
    _, p = stats.mannwhitneyu(a, b, alternative="two-sided")
    return p < alpha, p


def cv(x):
    """Współczynnik zmienności (cel pilotażu: poniżej ~5-10%)."""
    x = np.asarray(x, dtype=float)
    return x.std(ddof=1) / x.mean() * 100


if __name__ == "__main__":
    import sys
    from load import RESULTS, load_series
    env, series, metric = (sys.argv[1:4] + ["local", "seria_a", "measure_p95"])[:3] if len(sys.argv) < 4 else sys.argv[1:4]
    df = load_series(RESULTS, env, series)
    if df.empty:
        sys.exit("brak wyników")
    print(overhead(df, metric).to_string())
