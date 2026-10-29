#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Plot current span distribution and per-size footprint over GC snapshots.

Creates a distribution chart and a separate GC trend chart. The distribution
chart has two bars per objectSize with count/byte axes; each bar is labeled
with its value and its share of that measure's total.

Usage:
    python3 plot_span_dist.py 10.22.log [distribution.png] [--trend-out trend.png]
"""
import argparse
import re

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.ticker import MaxNLocator

HEAP_RE = re.compile(r"\[GC\.Heap\].*NumGC: (\d+)")
SPAN_RE = re.compile(
    r"\[GC\.Span\] objectsize=(\d+), mallocSpan=(\d+), freeSpan=(-?\d+)")
TOTAL_RE = re.compile(r"\[GC\.Total\].*PoolBytes: (\d+)")


def parse_snapshots(path):
    """Return ordered snapshots: (NumGC, span rows, PoolBytes)."""
    snapshots = []
    in_block = False
    numgc = None
    spans = {}
    pool_bytes = None

    def save_snapshot():
        if spans:
            snapshots.append((numgc, dict(spans), pool_bytes))

    with open(path, encoding="utf-8", errors="replace") as f:
        for raw in f:
            line = raw
            i = line.find("]: ")
            if i != -1:
                line = line[i + 3:]

            if "memPool stats dump" in line:
                if in_block:
                    save_snapshot()
                in_block, numgc, spans, pool_bytes = True, None, {}, None
                continue

            m = HEAP_RE.search(line)
            if m:
                if not in_block:
                    in_block, spans, pool_bytes = True, {}, None
                elif spans:
                    # A new heap header before the separator starts a new snapshot.
                    save_snapshot()
                    spans, pool_bytes = {}, None
                numgc = int(m.group(1))
                continue

            if not in_block:
                continue

            m = TOTAL_RE.search(line)
            if m:
                pool_bytes = int(m.group(1))
                continue

            m = SPAN_RE.search(line)
            if m:
                spans[int(m.group(1))] = (int(m.group(2)), int(m.group(3)))
                continue

            if "===============================" in line:
                save_snapshot()
                in_block, numgc, spans, pool_bytes = False, None, {}, None

    if in_block:
        save_snapshot()
    if not snapshots:
        raise SystemExit("日志中没有找到 [GC.Span] 快照")
    return snapshots


def live_span_count(rows, size):
    allocated, freed = rows.get(size, (0, 0))
    # The log reports cumulative span allocation/free counts.
    return max(0, allocated - freed)


def footprint_by_size(rows, sizes, span_size):
    """Return live bytes per size using the exact pool layout.

    mallocSpan and freeSpan are cumulative counts. The live span count is
    their difference, and each span owns ``span_size`` objects of
    ``objectSize + 12`` bytes (the pool header/tail metadata).
    """
    return [live_span_count(rows, size) * span_size * (size + 12)
            for size in sizes]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("log", help="input aggregate log")
    parser.add_argument("out", nargs="?", default="span-dist.png",
                        help="distribution chart (default: span-dist.png)")
    parser.add_argument("--trend-out", help="GC trend chart output path")
    parser.add_argument("--span-size", type=int, default=16,
                        help="objects per span; defaults to pool DefaultSpanSize (16)")
    args = parser.parse_args()
    if args.span_size <= 0:
        parser.error("--span-size must be positive")

    snapshots = parse_snapshots(args.log)
    # Use the snapshot with the largest PoolBytes for the distribution chart.
    # The trend chart still uses every snapshot.
    _, peak_snapshot = max(
        enumerate(snapshots),
        key=lambda item: item[1][2] if item[1][2] is not None else -1)
    peak_gc, latest, peak_pool_bytes = peak_snapshot
    sizes = sorted({size for _, rows, _ in snapshots for size in rows})

    # mpool span footprint is (objectSize + reservedLen) * spanSize;
    # reservedLen is 12 bytes in mpool.go.
    counts = [live_span_count(latest, size) for size in sizes]
    byte_counts = footprint_by_size(latest, sizes, args.span_size)
    trend_out = args.trend_out
    if not trend_out:
        if args.out.lower().endswith(".png"):
            trend_out = args.out[:-4] + "-trend.png"
        else:
            trend_out = args.out + "-trend.png"

    # Distribution chart: grouped bars with separate count and footprint axes.
    positions = list(range(len(sizes)))
    width = 0.38
    fig, ax_count = plt.subplots(figsize=(max(14, len(sizes) * 0.9), 6.5))
    count_bars = ax_count.bar(
        [x - width / 2 for x in positions], counts, width,
        color="#6366f1", label="live span count")
    ax_bytes = ax_count.twinx()
    byte_bars = ax_bytes.bar(
        [x + width / 2 for x in positions],
        [value / 2**20 for value in byte_counts], width,
        color="#f59e0b", alpha=0.88, label="live footprint")
    ax_count.set_yscale("log")
    ax_bytes.set_yscale("log")
    ax_count.set_ylabel("live span count")
    ax_bytes.set_ylabel("live footprint (MiB)")
    ax_count.set_title(
        "Peak PoolBytes span distribution (NumGC={}, PoolBytes={}, spanSize={})".format(
            peak_gc if peak_gc is not None else "?",
            "{:,}".format(peak_pool_bytes) if peak_pool_bytes is not None else "?",
            args.span_size))
    ax_count.set_xticks(positions)
    ax_count.set_xticklabels(["{}B".format(size) for size in sizes],
                             rotation=35, ha="right")
    ax_count.grid(axis="y", linestyle="--", alpha=0.35)
    ax_count.legend([count_bars, byte_bars],
                    ["live span count", "live footprint"], loc="upper right")

    total_count = sum(counts)
    total_bytes = sum(byte_counts)
    for bar, value in zip(count_bars, counts):
        if value <= 0:
            continue
        pct = 100 * value / total_count if total_count else 0
        ax_count.annotate(
            "{:,}\n{:.1f}%".format(value, pct),
            xy=(bar.get_x() + bar.get_width() / 2, value),
            xytext=(0, 3), textcoords="offset points",
            ha="center", va="bottom", fontsize=7, color="#3730a3")
    for bar, value in zip(byte_bars, byte_counts):
        if value <= 0:
            continue
        pct = 100 * value / total_bytes if total_bytes else 0
        mib = value / 2**20
        ax_bytes.annotate(
            "{:.1f} MiB\n{:.1f}%".format(mib, pct),
            xy=(bar.get_x() + bar.get_width() / 2, mib),
            xytext=(0, 3), textcoords="offset points",
            ha="center", va="bottom", fontsize=7, color="#92400e")

    fig.tight_layout()
    plt.savefig(args.out, dpi=150)
    plt.close(fig)

    # Separate trend chart: current bytes by size class at every GC snapshot.
    fig, ax_trend = plt.subplots(figsize=(14, 7.5))
    gc_values = [gc if gc is not None else i + 1
                 for i, (gc, _, _) in enumerate(snapshots)]
    for size in sizes:
        values_mib = []
        for _, rows, _ in snapshots:
            distribution = footprint_by_size(rows, sizes, args.span_size)
            values_mib.append(distribution[sizes.index(size)] / 2**20)
        line, = ax_trend.plot(gc_values, values_mib, marker="o", ms=2.8,
                              lw=2.2 if size == 3072 else 1.2,
                              label="{}B".format(size))
        if size == 3072 and values_mib:
            peak_i = max(range(len(values_mib)), key=values_mib.__getitem__)
            final_i = len(values_mib) - 1
            ax_trend.annotate(
                "3072B peak: {:.1f} MiB (GC {})".format(
                    values_mib[peak_i], gc_values[peak_i]),
                (gc_values[peak_i], values_mib[peak_i]),
                xytext=(8, 8), textcoords="offset points", fontsize=8,
                color=line.get_color(),
                arrowprops={"arrowstyle": "->", "color": line.get_color()})
            ax_trend.annotate(
                "final: {:.1f} MiB (GC {})".format(
                    values_mib[final_i], gc_values[final_i]),
                (gc_values[final_i], values_mib[final_i]),
                xytext=(-95, -18), textcoords="offset points", fontsize=8,
                color=line.get_color(),
                arrowprops={"arrowstyle": "->", "color": line.get_color()})
    ax_trend.set_title("Live footprint by objectSize over GC snapshots")
    ax_trend.set_xlabel("NumGC (GC count)")
    ax_trend.set_ylabel("live footprint (MiB)")
    ax_trend.set_yscale("symlog", linthresh=0.1)
    ax_trend.xaxis.set_major_locator(MaxNLocator(integer=True))
    ax_trend.grid(linestyle="--", alpha=0.35)
    ax_trend.legend(ncol=4, fontsize=8, loc="upper left")

    fig.tight_layout()
    plt.savefig(trend_out, dpi=150)
    plt.close(fig)
    print("saved: {} (peak NumGC={}, PoolBytes={}, {} object sizes, {:.1f} MiB span footprint)".format(
        args.out, peak_gc if peak_gc is not None else "?",
        peak_pool_bytes if peak_pool_bytes is not None else "?",
        len(sizes), total_bytes / 2**20))
    print("saved: {} ({} snapshots, {} object sizes)".format(
        trend_out, len(snapshots), len(sizes)))


if __name__ == "__main__":
    main()
