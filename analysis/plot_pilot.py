"""Wykresy pilotażu A2: narzut p95 i średniego CPU aplikacji względem C0 (C2/C4/C5 x interfejs).

Użycie: ~/venv-thesis/bin/python analysis/plot_pilot.py [env=pilot]
Wejście: results/<env>/pilot_summary.csv (z analysis/pilot_table.py)
Wyjście: analysis/plots/pilot_p95_overhead.png, analysis/plots/pilot_cpu_overhead.png
"""
import sys
from pathlib import Path

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt  # noqa: E402
import numpy as np  # noqa: E402
import pandas as pd  # noqa: E402

ROOT = Path(__file__).resolve().parent.parent
OUT = ROOT / "analysis" / "plots"
IFACES = [("rest", "REST"), ("graphql", "GraphQL"), ("grpc", "gRPC")]
CFGS = ["C2", "C4", "C5"]
CFG_LABEL = {"C2": "C2\nTLS + JWT", "C4": "C4\n+ walidacja + rate limit", "C5": "C5\n+ limity złożoności"}
# paleta kategoryczna (kolejność stała: niebieski, pomarańczowy, morski), tryb jasny
COLORS = {"rest": "#2a78d6", "graphql": "#eb6834", "grpc": "#1baf7a"}
INK, INK2, GRID, SURFACE = "#1f1f1e", "#5f5e58", "#e4e3df", "#fcfcfb"


def grouped(df, col, ylabel, title, fname, env, note=""):
    fig, ax = plt.subplots(figsize=(8, 5.2), dpi=150)
    fig.patch.set_facecolor(SURFACE)
    ax.set_facecolor(SURFACE)
    x = np.arange(len(CFGS))
    width = 0.26
    for k, (iface, label) in enumerate(IFACES):
        # słupek = mediana z powtórzeń, wąsy = min-max (przy 1 powtórzeniu bez wąsów)
        sel = [df.loc[(df.iface == iface) & (df.config == c), col].dropna() for c in CFGS]
        vals = [v.median() if len(v) else np.nan for v in sel]
        lo = [m - v.min() if len(v) > 1 else 0 for m, v in zip(vals, sel)]
        hi = [v.max() - m if len(v) > 1 else 0 for m, v in zip(vals, sel)]
        xs = x + (k - 1) * width
        bars = ax.bar(xs, vals, width * 0.92, color=COLORS[iface], label=label, zorder=3)
        if any(lo) or any(hi):
            ax.errorbar(xs, vals, yerr=[lo, hi], fmt="none", ecolor=INK2, elinewidth=1, capsize=3, zorder=4)
        for b, v, h, l in zip(bars, vals, hi, lo):
            if np.isfinite(v):
                y = v + h if v >= 0 else v - l  # etykieta nad wąsem, żeby się nie nakładały
                ax.annotate(f"{v:+.0f}%", (b.get_x() + b.get_width() / 2, y), xytext=(0, 3 if v >= 0 else -3),
                            textcoords="offset points", ha="center", va="bottom" if v >= 0 else "top", fontsize=8, color=INK2)
    ax.axhline(0, color=INK2, linewidth=1, zorder=4)
    ax.set_xticks(x, [CFG_LABEL[c] for c in CFGS], fontsize=9, color=INK)
    ax.set_ylabel(ylabel, color=INK)
    ax.yaxis.grid(True, color=GRID, linewidth=0.8, zorder=0)
    ax.set_axisbelow(True)
    for s in ("top", "right", "left"):
        ax.spines[s].set_visible(False)
    ax.spines["bottom"].set_color(GRID)
    ax.tick_params(axis="y", colors=INK2, length=0)
    ax.tick_params(axis="x", length=0)
    ax.legend(frameon=False, ncol=3, loc="upper left", fontsize=9)
    ax.set_title(title, loc="left", fontsize=12, color=INK, pad=22)
    nrep = int(df.groupby(["iface", "config"]).size().max())
    reps_txt = "1 powtórzenie" if nrep == 1 else f"{nrep} powtórzenia (słupek = mediana, wąsy = min–max)"
    ax.text(0, 1.02, f"pilotaż, {reps_txt}; O1, 70% SAT C0, pomiar 120 s, laptop; wyniki orientacyjne",
            transform=ax.transAxes, fontsize=8, color=INK2)
    if note:
        fig.text(0.01, 0.01, note, fontsize=7.5, color=INK2, ha="left", va="bottom", wrap=True)
    fig.tight_layout(rect=(0, 0.07 if note else 0, 1, 1))
    OUT.mkdir(parents=True, exist_ok=True)
    fig.savefig(OUT / fname, facecolor=SURFACE)
    print(OUT / fname)


def shortfall_note(df):
    """Przypis o przebiegach, w których osiągnięty RPS < 95% zadanego (wartości z pilot_summary.csv)."""
    low = df[df.rps < 0.95 * df.rate]
    if low.empty:
        return ""
    parts = [f"{i} {c} " + ", ".join(f"{v:.0f}" for v in g.rps) + f"/{g.rate.iloc[0]}"
             for (i, c), g in low.groupby(["iface", "config"], sort=False)]
    return ("UWAGA: osiągnięty < 95% zadanego RPS (osiągnięty/zadany): " + "; ".join(parts) + ". Host (k6 + serwer na jednym "
            "laptopie) był nasycony przy TLS, więc te słupki mierzą ograniczenie generatora, a nie samego mechanizmu.")


def main(env="pilot"):
    df = pd.read_csv(ROOT / "results" / env / "pilot_summary.csv")
    note = shortfall_note(df)
    grouped(df, "p95_vs_C0_pct", "zmiana p95 względem C0 [%]", "Narzut p95 latencji względem C0", "pilot_p95_overhead.png", env, note)
    grouped(df, "cpu_vs_C0_pct", "zmiana średniego CPU aplikacji względem C0 [%]", "Narzut CPU aplikacji względem C0",
            "pilot_cpu_overhead.png", env, note)


if __name__ == "__main__":
    main(*(sys.argv[1:2] or ["pilot"]))
