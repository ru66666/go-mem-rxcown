#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Plot span counts/bytes over GC and per-size byte shares."""
import argparse
import re

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.ticker import MaxNLocator

HEAP_RE = re.compile(r"\[GC\.Heap\].*NumGC: (\d+)")
TOTAL_RE = re.compile(r"\[GC\.Total\].*PoolBytes: (\d+).*mallocSpan: (-?\d+), freeSpan: (-?\d+)")
SPAN_RE = re.compile(r"\[GC\.Span\] objectsize=(\d+), mallocSpan=(-?\d+), freeSpan=(-?\d+)")


def parse_snapshots(path):
    """Return (NumGC, PoolBytes, mallocSpan, freeSpan, per-size rows)."""
    snapshots, in_block, numgc, total, spans = [], False, None, None, {}

    def save():
        if total is not None:
            snapshots.append((numgc, total[0], total[1], total[2], dict(spans)))

    with open(path, encoding="utf-8", errors="replace") as f:
        for raw in f:
            line = raw
            i = line.find("]: ")
            if i != -1:
                line = line[i + 3:]
            if "memPool stats dump" in line:
                if in_block:
                    save()
                in_block, numgc, total, spans = True, None, None, {}
                continue
            m = HEAP_RE.search(line)
            if m:
                if in_block and total is not None:
                    save()
                    total, spans = None, {}
                in_block, numgc = True, int(m.group(1))
                continue
            if not in_block:
                continue
            m = TOTAL_RE.search(line)
            if m:
                total = tuple(map(int, m.groups()))
                continue
            m = SPAN_RE.search(line)
            if m:
                spans[int(m.group(1))] = (int(m.group(2)), int(m.group(3)))
                continue
            if "===============================" in line:
                save()
                in_block, numgc, total, spans = False, None, None, {}
    if in_block:
        save()
    if not snapshots:
        raise SystemExit("日志中没有找到 [GC.Total] 快照")
    return snapshots


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("log")
    p.add_argument("out", nargs="?", default="spans.png")
    p.add_argument("--span-size", type=int, default=16)
    args = p.parse_args()
    if args.span_size <= 0:
        p.error("--span-size must be positive")

    snaps = parse_snapshots(args.log)
    x = [r[0] if r[0] is not None else i + 1 for i, r in enumerate(snaps)]
    malloc_counts, free_counts = [r[2] for r in snaps], [r[3] for r in snaps]
    malloc_bytes = [sum(max(0, a) * args.span_size * (s + 12) for s, (a, _) in r[4].items()) for r in snaps]
    free_bytes = [sum(max(0, f) * args.span_size * (s + 12) for s, (_, f) in r[4].items()) for r in snaps]

    peak = max(snaps, key=lambda r: r[1])
    peak_gc, peak_pool, _, _, rows = peak
    sizes = sorted(rows)
    mraw = [max(0, rows[s][0]) * args.span_size * (s + 12) for s in sizes]
    fraw = [max(0, rows[s][1]) * args.span_size * (s + 12) for s in sizes]
    mscale = peak[2] / sum(mraw) if sum(mraw) else 1
    fscale = peak[3] / sum(fraw) if sum(fraw) else 1
    mbytes, fbytes = [v * mscale for v in mraw], [v * fscale for v in fraw]

    fig, (ax, malloc_ax, free_ax) = plt.subplots(
        3, 1, figsize=(max(14, len(sizes) * .9), 14),
        gridspec_kw={"height_ratios": [1, 1.2, 1.2]})
    axb = ax.twinx()
    l1, = ax.plot(x, malloc_counts, "o-", ms=3, lw=1.7, color="#6366f1", label="mallocSpan count")
    l2, = ax.plot(x, free_counts, "s-", ms=3, lw=1.7, color="#ef4444", label="freeSpan count")
    l3, = axb.plot(x, [v / 2**30 for v in malloc_bytes], "--", lw=1.4, color="#6366f1", label="mallocSpan bytes")
    l4, = axb.plot(x, [v / 2**30 for v in free_bytes], "--", lw=1.4, color="#ef4444", label="freeSpan bytes")
    ax.set_title("mallocSpan / freeSpan counts and bytes over GC")
    ax.set_xlabel("NumGC (GC count)"); ax.set_ylabel("span count"); axb.set_ylabel("estimated bytes (GiB)")
    ax.xaxis.set_major_locator(MaxNLocator(integer=True)); ax.grid(linestyle="--", alpha=.35)
    ax.legend([l1, l2, l3, l4], [h.get_label() for h in [l1, l2, l3, l4]], loc="upper left", ncol=2, fontsize=8)

    # Middle/lower panels: split malloc and free bytes so each has one clear
    # vertical axis. The color identifies objectSize.
    colors = plt.cm.tab20.colors
    handles = []
    for i, size in enumerate(sizes):
        color = colors[i % len(colors)]
        malloc_series = []
        free_series = []
        for _, _, _, _, rows in snaps:
            allocated, freed = rows.get(size, (0, 0))
            malloc_series.append(max(0, allocated) * args.span_size * (size + 12) / 2**20)
            free_series.append(max(0, freed) * args.span_size * (size + 12) / 2**20)
        line, = malloc_ax.plot(x, malloc_series, color=color, lw=1.3,
                               marker="o", ms=2.2, label="{}B".format(size))
        free_ax.plot(x, free_series, color=color, lw=1.2,
                     marker="s", ms=2.0)
        handles.append(line)
    malloc_ax.set_title("mallocSpan bytes by objectSize over GC")
    malloc_ax.set_ylabel("mallocSpan bytes (MiB)")
    malloc_ax.set_yscale("symlog", linthresh=0.1)
    malloc_ax.xaxis.set_major_locator(MaxNLocator(integer=True))
    malloc_ax.grid(linestyle="--", alpha=.35)
    malloc_ax.legend(handles=handles, ncol=4, fontsize=7, loc="upper left",
                     title="objectSize")
    free_ax.set_title("freeSpan bytes by objectSize over GC")
    free_ax.set_xlabel("NumGC (GC count)")
    free_ax.set_ylabel("freeSpan bytes (MiB)")
    free_ax.set_yscale("symlog", linthresh=0.1)
    free_ax.xaxis.set_major_locator(MaxNLocator(integer=True))
    free_ax.grid(linestyle="--", alpha=.35)
    free_ax.legend(handles=handles, ncol=4, fontsize=7, loc="upper left",
                   title="objectSize")
    fig.tight_layout(); plt.savefig(args.out, dpi=150)
    print("saved: {} ({} snapshots, peak NumGC={}, peak PoolBytes={:,})".format(args.out, len(snaps), peak_gc if peak_gc is not None else "?", peak_pool))


if __name__ == "__main__":
    main()
