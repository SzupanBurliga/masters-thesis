"""Kalibracja A1: osiągnięta przepustowość w funkcji zadanej (offered vs achieved RPS), O1 i O3,
z zaznaczonym punktem nasycenia każdego przebiegu.

Użycie: ~/venv-thesis/bin/python analysis/plot_a1.py [env=local_db3cpu]
Wejście: results/<env>/seria_a_saturation_{O1,O3}_ramp_{windows,summary}.csv (z analysis/saturation.py)
Wyjście: analysis/plots/a1_offered_vs_achieved.png

Zadany RPS nie jest zapisany w plikach okien; wynika z definicji rampy k6 (ramping-arrival-rate, liniowo
od startRate=10 do MAXRATE w czasie RAMP; loadtests/seria_a.js i loadtests/run_a1_calibration.sh):
  O1: 10 -> 8000 RPS w 720 s, O3: 10 -> 4000 RPS w 360 s. Okno w (10 s) dostaje wartość ze środka, w + 5 s.
Osiągnięty RPS = liczba żądań zakończonych w oknie / 10 s (kolumna rps). Punkt nasycenia = okno, którego
rps równa się first_bad_rps z pliku summary (początek 3 kolejnych "złych" okien, kryterium z saturation.py).
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
RAMPS = {"O1": (10, 8000, 720), "O3": (10, 4000, 360)}  # start, max, czas [s]
WIN = 10
IFACES = [("rest", "REST"), ("graphql", "GraphQL"), ("grpc", "gRPC")]
COLORS = {"rest": "#2a78d6", "graphql": "#eb6834", "grpc": "#1baf7a"}
INK, INK2, GRID, SURFACE = "#1f1f1e", "#5f5e58", "#e4e3df", "#fcfcfb"


def offered(op, w):
    start, top, dur = RAMPS[op]
    t = np.minimum(w + WIN / 2, dur)
    return start + (top - start) * t / dur


def main(env="local_db3cpu"):
    fig, axes = plt.subplots(1, 2, figsize=(11, 4.8), dpi=150)
    fig.patch.set_facecolor(SURFACE)
    for ax, op in zip(axes, ["O1", "O3"]):
        ax.set_facecolor(SURFACE)
        w = pd.read_csv(ROOT / "results" / env / f"seria_a_saturation_{op}_ramp_windows.csv")
        s = pd.read_csv(ROOT / "results" / env / f"seria_a_saturation_{op}_ramp_summary.csv")
        top = RAMPS[op][1]
        ax.plot([0, top], [0, top], color=INK2, linewidth=1, linestyle=(0, (4, 3)), zorder=2, label="osiągnięty = zadany")
        # pomijamy ostatnie okno każdego przebiegu: k6 kończy rampę w jego trakcie, więc okno jest niepełne
        # (okna liczone od pierwszego znacznika czasu, nie od startu rampy) i daje sztuczny spadek RPS
        w = w[w.w < w.groupby(["iface", "rep"]).w.transform("max")]
        for iface, label in IFACES:
            for rep in sorted(w[(w.iface == iface)].rep.unique()):
                g = w[(w.iface == iface) & (w.rep == rep)].sort_values("w")
                ax.plot(offered(op, g.w), g.rps, color=COLORS[iface], linewidth=1.4, alpha=0.9, zorder=3,
                        label=label if rep == 1 else None)
                row = s[(s.iface == iface) & (s.rep == rep)]
                if len(row) and pd.notna(row.first_bad_rps.iloc[0]):
                    hit = g.iloc[(g.rps - row.first_bad_rps.iloc[0]).abs().argmin()]
                    ax.scatter(offered(op, hit.w), hit.rps, s=42, color=COLORS[iface], edgecolor=SURFACE, linewidth=1.5, zorder=5)
        ax.set_title(f"{op} ({'użytkownik' if op == 'O1' else 'użytkownik → zamówienia → pozycje → produkt'})",
                     loc="left", fontsize=11, color=INK)
        ax.set_xlabel("zadany RPS (rampa k6)", color=INK)
        ax.set_ylabel("osiągnięty RPS (okna 10 s)", color=INK)
        ax.set_xlim(0, top)
        ax.set_ylim(0, top * 1.05)
        ax.grid(True, color=GRID, linewidth=0.8, zorder=0)
        for sp in ("top", "right"):
            ax.spines[sp].set_visible(False)
        for sp in ("left", "bottom"):
            ax.spines[sp].set_color(GRID)
        ax.tick_params(colors=INK2, length=0)
    axes[0].legend(frameon=False, fontsize=9, loc="upper left", title="kropka = punkt nasycenia\n(każde z 3 powtórzeń)",
                   title_fontsize=8, alignment="left")
    fig.suptitle("Kalibracja A1 (C0): przepustowość osiągnięta vs zadana", x=0.01, ha="left", fontsize=12, color=INK)
    fig.text(0.01, 0.925, f"env {env}: aplikacja 2 CPU, Postgres 3 CPU, k6 na tym samym hoście; 3 powtórzenia",
             fontsize=8, color=INK2)
    fig.tight_layout(rect=(0, 0, 1, 0.93))
    OUT.mkdir(parents=True, exist_ok=True)
    fig.savefig(OUT / "a1_offered_vs_achieved.png", facecolor=SURFACE)
    print(OUT / "a1_offered_vs_achieved.png")


if __name__ == "__main__":
    main(*(sys.argv[1:2] or ["local_db3cpu"]))
