#!/usr/bin/env python3

import sys
from pathlib import Path

import matplotlib.colors as mcolors
import matplotlib.patches as mpatches
import matplotlib.pyplot as plt
import numpy as np
import pandas as pd


SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = SCRIPT_DIR.parent
DATA_DIR = REPO_ROOT / "experiments_data"
FIG_DIR = DATA_DIR / "figures"
FIG_DIR.mkdir(parents=True, exist_ok=True)


ARCH_COLORS = {
    "computeToData":              "#1f3b66",
    "computeToData-cached":       "#3d7ab8",
    "computeToData-compressed":   "#8db4e2",
    "dataThroughTtp":             "#8b1a1a",
    "dataThroughTtp-cached":      "#d35400",
    "dataThroughTtp-compressed":  "#f4b183",
}


ARCH_LINESTYLE = {
    "computeToData":             "-",
    "dataThroughTtp":            "-",
    "computeToData-cached":      "--",
    "dataThroughTtp-cached":     "--",
    "computeToData-compressed":  ":",
    "dataThroughTtp-compressed": ":",
}


ARCH_ORDER = [
    "computeToData",
    "computeToData-cached",
    "computeToData-compressed",
    "dataThroughTtp",
    "dataThroughTtp-cached",
    "dataThroughTtp-compressed",
]

plt.rcParams.update({
    "figure.dpi": 100,
    "savefig.dpi": 200,
    "savefig.bbox": "tight",
    "savefig.pad_inches": 0.25,
    "font.size": 11,
    "axes.titlesize": 12,
    "axes.titleweight": "bold",
    "axes.labelsize": 11,
    "xtick.labelsize": 10,
    "ytick.labelsize": 10,
    "legend.fontsize": 9,
    "legend.frameon": False,
    "axes.grid": True,
    "axes.axisbelow": True,
    "grid.alpha": 0.3,
    "grid.linewidth": 0.5,
    "lines.linewidth": 2.0,
})


def _save(fig, name):
    out = FIG_DIR / f"{name}.png"
    fig.savefig(out)
    print(f"  wrote {out.relative_to(REPO_ROOT)}")
    plt.close(fig)


def _arch_order_present(df):
    present = set(df["archetype"].unique())
    return [a for a in ARCH_ORDER if a in present]


def _plot_archetype_line(ax, df_arch, arch, **kwargs):
    ax.plot(
        df_arch.iloc[:, 0], df_arch.iloc[:, 1],
        label=arch,
        color=ARCH_COLORS[arch],
        linestyle=ARCH_LINESTYLE[arch],
        **kwargs,
    )


def plot_weight_sweep():
    print("Experiment 1: weight_sweep")
    df = pd.read_csv(DATA_DIR / "weight_sweep.csv")
    varied_criteria = ["energy", "latency", "cost", "privacy"]
    arches = _arch_order_present(df)
    sql_limits = sorted(df["sql_limit"].unique()) if "sql_limit" in df.columns else [20000]

    for limit in sql_limits:
        sub_limit = df[df["sql_limit"] == limit] if "sql_limit" in df.columns else df

        fig, axes = plt.subplots(2, 2, figsize=(11, 7.5),
                                 sharex=True, sharey=True)
        axes = axes.flatten()
        for ax, varied in zip(axes, varied_criteria):
            sub = sub_limit[sub_limit["varied"] == varied]
            for arch in arches:
                arch_data = sub[sub["archetype"] == arch].sort_values("weight")
                _plot_archetype_line(
                    ax, arch_data[["weight", "score"]], arch,
                )
            ax.set_title(f"Varying weight of '{varied}'")
            ax.set_xlim(0, 1)
            ax.set_ylim(0, 1.0)
            if ax in (axes[2], axes[3]):
                ax.set_xlabel("weight of varied criterion (rest split equally)")
            if ax in (axes[0], axes[2]):
                ax.set_ylabel("TOPSIS closeness coefficient")

        fig.suptitle(
            f"Weight sensitivity sweep — {limit:,} SQL limit, 2 providers, warm cache".replace(",", " "),
            fontsize=13, y=1.02,
        )
        handles, labels = axes[0].get_legend_handles_labels()
        fig.legend(
            handles, labels,
            loc="lower center",
            ncol=3,
            bbox_to_anchor=(0.5, -0.05),
            title="Archetype",
        )
        suffix = f"_{int(limit) // 1000}k"
        _save(fig, f"weight_sweep{suffix}")


def plot_workload_scaling():
    print("Experiment 2: workload_scaling")
    df = pd.read_csv(DATA_DIR / "workload_scaling.csv")
    arches = _arch_order_present(df)

    fig, axes = plt.subplots(1, 2, figsize=(12, 5),
                             sharey=True)
    titles = {
        "warm": "Warm cache",
        "cold": "Cold cache",
    }
    for ax, state in zip(axes, ("warm", "cold")):
        sub = df[df["cache_state"] == state]
        for arch in arches:
            arch_data = sub[sub["archetype"] == arch].sort_values("expected_rows")
            ax.plot(
                arch_data["expected_rows"], arch_data["score"],
                label=arch,
                color=ARCH_COLORS[arch],
                linestyle=ARCH_LINESTYLE[arch],
                marker="o",
                markersize=4,
            )
        ax.set_xlabel("SQL limit")
        ax.set_title(titles[state])
        ax.set_xticks([3000, 6000, 9000, 12000, 15000, 18000,
                       21000, 24000, 27000, 30000])
        ax.tick_params(axis="x", rotation=45)
        ax.set_ylim(0, 1.0)
    axes[0].set_ylabel("TOPSIS closeness coefficient")

    fig.suptitle(
        "Workload scaling: energy-heavy weighting, 2 providers",
        fontsize=13, y=1.04,
    )
    handles, labels = axes[0].get_legend_handles_labels()
    fig.legend(
        handles, labels,
        loc="lower center",
        ncol=3,
        bbox_to_anchor=(0.5, -0.22),
        title="Archetype",
    )
    _save(fig, "workload_scaling")


def plot_rank_reversal():
    print("Experiment 5: rank_reversal")
    df = pd.read_csv(DATA_DIR / "rank_reversal.csv")
    base_archs = ["computeToData", "dataThroughTtp"]
    base = df[df["archetype"].isin(base_archs)]

    fig, (ax, ax_ratio) = plt.subplots(
        1, 2, figsize=(11, 4.5),
        gridspec_kw={"width_ratios": [2.4, 1]},
    )
    x = range(len(base_archs))
    width = 0.38
    run_labels = {
        "full_catalogue": "Full catalogue (6 archetypes)",
        "base_only":      "Base archetypes only (2 archetypes)",
    }
    run_colors = {
        "full_catalogue": "#1f3b66",
        "base_only":      "#d35400",
    }
    run_offsets = {
        "full_catalogue": -width / 2,
        "base_only":      width / 2,
    }
    scores_by_run = {}
    for run in ("full_catalogue", "base_only"):
        sub = base[base["run"] == run].set_index("archetype").reindex(base_archs)
        bars = ax.bar(
            [i + run_offsets[run] for i in x], sub["score"],
            width,
            label=run_labels[run],
            color=run_colors[run],
            edgecolor="black", linewidth=0.5,
        )
        for bar, score in zip(bars, sub["score"]):
            ax.text(
                bar.get_x() + bar.get_width() / 2,
                bar.get_height() + 0.02,
                f"{score:.3f}",
                ha="center", fontsize=9,
            )
        scores_by_run[run] = sub["score"]

    ax.set_xticks(list(x))
    ax.set_xticklabels(base_archs)
    ax.set_ylabel("TOPSIS closeness coefficient")
    ax.set_ylim(0, 1.10)
    ax.set_title("Scores per archetype")
    ax.legend(loc="upper right")

    rescale = {
        arch: scores_by_run["base_only"][arch] / scores_by_run["full_catalogue"][arch]
        for arch in base_archs
    }
    ratio_x = range(len(base_archs))
    ratio_bars = ax_ratio.bar(
        ratio_x,
        [rescale[a] for a in base_archs],
        width=0.55,
        color=["#5c7aa6", "#5c7aa6"],
        edgecolor="black", linewidth=0.5,
    )
    for bar, arch in zip(ratio_bars, base_archs):
        ax_ratio.text(
            bar.get_x() + bar.get_width() / 2,
            bar.get_height() + 0.05,
            f"{rescale[arch]:.2f}×",
            ha="center", fontsize=9,
        )
    ax_ratio.set_xticks(list(ratio_x))
    ax_ratio.set_xticklabels(base_archs, fontsize=9)
    ax_ratio.set_ylabel("base score vs full-catalogue score")
    ax_ratio.set_ylim(0, max(rescale.values()) * 1.35)
    ax_ratio.set_title("Rescaling factor")

    rel_diff = abs(rescale["computeToData"] - rescale["dataThroughTtp"]) / rescale["computeToData"] * 100
    ax_ratio.text(
        0.5, max(rescale.values()) * 1.22,
        f"Δ = {rel_diff:.1f}%",
        ha="center", fontsize=9, fontweight="bold", color="#333333",
        transform=ax_ratio.transData,
    )

    fig.suptitle("Rank stability of the archetypes")
    fig.tight_layout(rect=[0, 0, 1, 0.95])
    _save(fig, "rank_reversal")


    df.to_csv(FIG_DIR / "rank_reversal_table.csv", index=False)


SCENARIOS_ORDER = ["balanced", "energy", "latency", "monetary", "privacy"]


def _short_arch(name):
    return (
        name.replace("computeToData", "CtD")
            .replace("dataThroughTtp", "DtTTP")
            .replace("-compressed", "-compress")
    )


def _draw_winner_heatmap(ax, df_state, scenarios, sql_limits, title):
    arch_to_idx = {a: i for i, a in enumerate(ARCH_ORDER)}

    winners = (
        df_state.loc[df_state.groupby(["scenario", "sql_limit"])["score"].idxmax()]
                .reset_index(drop=True)
    )

    grid_names = np.empty((len(scenarios), len(sql_limits)), dtype=object)
    grid_idx = np.zeros((len(scenarios), len(sql_limits)), dtype=int)
    for i, scen in enumerate(scenarios):
        for j, lim in enumerate(sql_limits):
            match = winners[(winners["scenario"] == scen) & (winners["sql_limit"] == lim)]
            if len(match) == 0:
                grid_names[i, j] = "?"
                grid_idx[i, j] = 0
                continue
            arch = match["archetype"].values[0]
            grid_names[i, j] = arch
            grid_idx[i, j] = arch_to_idx[arch]

    cmap = mcolors.ListedColormap([ARCH_COLORS[a] for a in ARCH_ORDER])
    bounds = list(range(len(ARCH_ORDER) + 1))
    norm = mcolors.BoundaryNorm(bounds, cmap.N)

    ax.imshow(grid_idx, cmap=cmap, norm=norm, aspect="auto",
              interpolation="nearest")


    for j in range(1, len(sql_limits)):
        ax.axvline(j - 0.5, color="white", linewidth=1.0)
    for i in range(1, len(scenarios)):
        ax.axhline(i - 0.5, color="white", linewidth=1.0)

    ax.set_xticks(range(len(sql_limits)))
    ax.set_xticklabels(sql_limits, rotation=0, fontsize=9)
    ax.set_yticks(range(len(scenarios)))
    ax.set_yticklabels(scenarios)
    ax.set_ylabel("Scenario")
    ax.set_title(title, pad=8)
    ax.grid(False)


    for i in range(len(scenarios)):
        for j in range(len(sql_limits)):
            ax.text(
                j, i,
                _short_arch(grid_names[i, j]),
                ha="center", va="center",
                fontsize=7, color="white", fontweight="bold",
            )

    return sorted(set(grid_names.flatten()), key=lambda a: arch_to_idx.get(a, 99))


def plot_scenarios_winner_heatmap():
    print("Experiment 6a: scenarios_winner_heatmap (warm + cold)")
    df = pd.read_csv(DATA_DIR / "scenarios_workload.csv")

    scenarios = SCENARIOS_ORDER
    sql_limits = sorted(df["sql_limit"].unique())

    fig, (ax_warm, ax_cold) = plt.subplots(
        2, 1,
        figsize=(13, 10),
        sharex=True,
        gridspec_kw={"hspace": 0.18},
    )

    archs_used_warm = _draw_winner_heatmap(
        ax_warm,
        df[df["cache_state"] == "warm"],
        scenarios, sql_limits,
        title="Warm cache",
    )
    archs_used_cold = _draw_winner_heatmap(
        ax_cold,
        df[df["cache_state"] == "cold"],
        scenarios, sql_limits,
        title="Cold cache",
    )
    ax_cold.set_xlabel("SQL limit, log scale)")


    archs_used = []
    for a in ARCH_ORDER:
        if a in archs_used_warm or a in archs_used_cold:
            archs_used.append(a)
    legend_handles = [
        mpatches.Patch(color=ARCH_COLORS[a], label=a) for a in archs_used
    ]
    fig.legend(
        handles=legend_handles,
        loc="lower center",
        bbox_to_anchor=(0.5, -0.02),
        ncol=min(len(legend_handles), 3),
        frameon=False,
        title="Winning archetype",
    )

    fig.suptitle(
        "Winning archetype per scenario × workload — 2 providers",
        fontsize=14, y=0.995,
    )
    _save(fig, "scenarios_winner_heatmap")


def plot_scenarios_score_lines():
    print("Experiment 6b: scenarios_score_lines (warm cache)")
    df = pd.read_csv(DATA_DIR / "scenarios_workload.csv")
    df = df[df["cache_state"] == "warm"]
    scenarios = SCENARIOS_ORDER
    arches = _arch_order_present(df)

    fig, axes = plt.subplots(
        2, 3,
        figsize=(15, 9),
        sharey=True, sharex=True,
        gridspec_kw={"hspace": 0.35, "wspace": 0.08},
    )
    axes_flat = axes.flatten()

    for ax, scen in zip(axes_flat, scenarios):
        sub = df[df["scenario"] == scen]
        for arch in arches:
            arch_data = sub[sub["archetype"] == arch].sort_values("sql_limit")
            ax.plot(
                arch_data["sql_limit"], arch_data["score"],
                label=arch,
                color=ARCH_COLORS[arch],
                linestyle=ARCH_LINESTYLE[arch],
                marker="o",
                markersize=4,
            )
        ax.set_xlabel("SQL limit")
        ax.set_title(scen, pad=8)
        ax.set_ylim(0, 1.0)
        ax.set_xticks([3000, 6000, 9000, 12000, 15000, 18000,
                       21000, 24000, 27000, 30000])
        ax.tick_params(axis="x", rotation=45)


    axes[0, 0].set_ylabel("TOPSIS closeness coefficient")
    axes[1, 0].set_ylabel("TOPSIS closeness coefficient")


    axes_flat[5].axis("off")
    handles, labels = axes_flat[0].get_legend_handles_labels()
    axes_flat[5].legend(
        handles, labels,
        loc="center",
        title="Archetype",
        frameon=False,
    )

    fig.suptitle(
        "TOPSIS scores per archetype across different weight scenarios - 2 providers, warm cache",
        fontsize=13, y=1.00,
    )
    _save(fig, "scenarios_score_lines")


OVERHEAD_CSV = REPO_ROOT / "experiments" / "solver-overhead" / "results.csv"


def plot_solver_overhead():
    print("Experiment 7b: solver_overhead")
    if not OVERHEAD_CSV.exists():
        print(f"  skipped: {OVERHEAD_CSV.relative_to(REPO_ROOT)} not found "
              "(run experiments/solver-overhead/fire.sh and pull.py first)")
        return

    df = pd.read_csv(OVERHEAD_CSV)

    panels = [
        ("cpu_cores",     "CPU usage",    "millicores (m)",  1000.0),
        ("energy_joules", "Energy usage", "mJ per second",   1000.0),
    ]

    color_off = ARCH_COLORS["dataThroughTtp"]
    color_on  = ARCH_COLORS["computeToData"]

    fig, axes = plt.subplots(1, 2, figsize=(10, 4.6))

    for ax, (key, title, unit, scale) in zip(axes, panels):
        off = (df.loc[df["condition"] == "off", key] * scale).tolist()
        on  = (df.loc[df["condition"] == "on",  key] * scale).tolist()
        off_m, on_m = float(np.mean(off)), float(np.mean(on))
        off_sd = float(np.std(off, ddof=1)) if len(off) > 1 else 0.0
        on_sd  = float(np.std(on,  ddof=1)) if len(on)  > 1 else 0.0

        ax.bar([0, 1], [off_m, on_m],
               yerr=[off_sd, on_sd], capsize=8,
               color=[color_off, color_on],
               edgecolor="white", linewidth=1.2, width=0.62, zorder=2,
               error_kw={"elinewidth": 1.5, "ecolor": "#333333"})

        for x, ys in ((0, off), (1, on)):
            ax.scatter([x] * len(ys), ys, s=46, color="white",
                       edgecolor="black", linewidth=1.4, zorder=3)

        for i, v in enumerate([off_m, on_m]):
            ax.text(i, v / 2, f"{v:.2f}", ha="center", va="center",
                    fontsize=11, fontweight="bold", color="white", zorder=4)

        ax.set_xticks([0, 1])
        ax.set_xticklabels(["Solver OFF", "Solver ON"])
        ax.set_ylabel(unit)
        ax.set_title(title, pad=8)

        all_y = [off_m + off_sd, on_m + on_sd] + off + on
        top = max(all_y) * 1.25
        ax.set_ylim(0, top)

        if off_m > 0:
            pct = 100.0 * (off_m - on_m) / off_m
            sign = "less" if pct >= 0 else "more"
            ax.text(0.5, top * 0.92,
                    f"Solver ON uses {abs(pct):.0f}% {sign}",
                    ha="center", va="top",
                    fontsize=9.5, color="#333333", style="italic")

    n_rounds = len(df) // 2
    fig.suptitle("Orchestrator resource use: solver enabled vs disabled",
                 fontsize=13, y=1.00)
    fig.text(0.5, -0.02,
             f"Bars: mean across {n_rounds} rounds × 300 requests   ·   "
             "Whiskers: sample standard deviation   ·   "
             "Dots: each round's average",
             ha="center", fontsize=8.5, color="#666666")

    _save(fig, "solver_overhead")


MEASURED_DIR = REPO_ROOT / "energy-efficiency" / "experiments" / "data_today"

MEASURED_LATENCY_REPS = {
    "computeToData":  [
        "baseline_ComputeToData_260623-2007/exp_1",
        "baseline_ComputeToData_260623-2105/exp_1",
    ],
    "dataThroughTtp": [
        "baseline_DataThroughTTP_260623-2038/exp_1",
        "baseline_DataThroughTTP_260623-2053/exp_1",
    ],
}

MEASURED_ENERGY_REPS = {
    "computeToData":  ["baseline_ComputeToData_260623-2007/exp_1"],
    "dataThroughTtp": ["baseline_DataThroughTTP_260623-2038/exp_1"],
}


def _load_measured_latencies(rep_path):
    df = pd.read_csv(MEASURED_DIR / rep_path / "runs_results.csv")
    return df.loc[df["data_status_code"] == 200, "data_req_exec_time"].tolist()


def _load_measured_active_energy(rep_path):
    section = None
    active = {}
    with open(MEASURED_DIR / rep_path / "full_energy_values.txt") as f:
        for raw in f:
            line = raw.strip()
            if line == "Idle Energy:":
                section = "idle"
            elif line == "Active Energy:":
                section = "active"
            elif ":" in line and section == "active":
                k, v = line.split(":", 1)
                active[k.strip()] = float(v.strip())
    return active


def plot_measured_archetype_comparison():
    print("Experiment 8: measured_archetype_comparison")
    if not MEASURED_DIR.exists():
        print(f"  skipped: {MEASURED_DIR.relative_to(REPO_ROOT)} not found "
              "(run energy-efficiency/experiments/execute_experiments.py first)")
        return


    latency = {}
    for arch, reps in MEASURED_LATENCY_REPS.items():
        all_times = []
        for rep in reps:
            all_times.extend(_load_measured_latencies(rep))
        latency[arch] = {
            "mean":      float(np.mean(all_times)),
            "min":       float(np.min(all_times)),
            "max":       float(np.max(all_times)),
            "n_actions": len(all_times),
            "n_reps":    len(reps),
        }


    energy = {}
    for arch, reps in MEASURED_ENERGY_REPS.items():
        e = _load_measured_active_energy(reps[0])
        energy[arch] = {
            "sql_query":     e.get("sql-query", 0.0),
            "sql_algorithm": e.get("sql-algorithm", 0.0),
        }

    archs = ["computeToData", "dataThroughTtp"]
    x = np.arange(len(archs))
    base_colors = [ARCH_COLORS[a] for a in archs]


    light_colors = [ARCH_COLORS[f"{a}-compressed"] for a in archs]

    fig, (ax_lat, ax_eng) = plt.subplots(1, 2, figsize=(11, 5))


    means = [latency[a]["mean"] for a in archs]
    lower = [latency[a]["mean"] - latency[a]["min"] for a in archs]
    upper = [latency[a]["max"]  - latency[a]["mean"] for a in archs]
    ax_lat.bar(
        x, means, color=base_colors,
        yerr=[lower, upper], capsize=8,
        edgecolor="black", linewidth=0.5,
    )
    for xi, m in zip(x, means):
        ax_lat.text(xi, m + 0.3, f"{m:.1f} s",
                    ha="center", fontsize=10, fontweight="bold")
    ax_lat.set_xticks(x)
    ax_lat.set_xticklabels(archs)
    ax_lat.set_ylabel("Data-request latency (seconds)")
    ax_lat.set_title(
        f"Measured per-request latency\n",
        pad=8,
    )


    sql_q = [energy[a]["sql_query"]     for a in archs]
    sql_a = [energy[a]["sql_algorithm"] for a in archs]
    totals = [q + al for q, al in zip(sql_q, sql_a)]
    ax_eng.bar(x, sql_q, color=base_colors,
               edgecolor="black", linewidth=0.5, label="sql-query")
    ax_eng.bar(x, sql_a, bottom=sql_q, color=light_colors,
               edgecolor="black", linewidth=0.5, label="sql-algorithm")
    for xi, t in zip(x, totals):
        ax_eng.text(xi, t + 2, f"{t:.0f} J",
                    ha="center", fontsize=10, fontweight="bold")
    ax_eng.set_xticks(x)
    ax_eng.set_xticklabels(archs)
    ax_eng.set_ylabel("Active energy (joules)")
    ax_eng.set_title(
        "Measured SQL-container active energy\n",
        pad=8,
    )
    ax_eng.legend(loc="upper left")

    fig.suptitle(
        "Measured archetype cost — Docker Desktop + Kepler",
        fontsize=13, y=1.02,
    )
    _save(fig, "measured_archetype_comparison")


def main():
    missing = [
        name for name in (
            "weight_sweep.csv",
            "workload_scaling.csv",
            "rank_reversal.csv",
            "scenarios_workload.csv",
        )
        if not (DATA_DIR / name).exists()
    ]
    if missing:
        sys.exit(
            f"Missing CSV files in {DATA_DIR}: {missing}\n"
            "Run `go test -run TestExperiment_ -v ./pkg/solver/...` from go/ first."
        )

    plot_weight_sweep()
    plot_workload_scaling()
    plot_rank_reversal()
    plot_scenarios_winner_heatmap()
    plot_scenarios_score_lines()
    plot_solver_overhead()
    plot_measured_archetype_comparison()
    print(f"\nAll figures in {FIG_DIR.relative_to(REPO_ROOT)}/")


if __name__ == "__main__":
    main()
