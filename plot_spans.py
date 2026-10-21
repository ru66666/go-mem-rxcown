#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
从 mpool 聚合日志中提取每个快照的 NumGC 与 [GC.Total] 的 mallocSpan/freeSpan，
画折线图：X 轴为 GC 次数（NumGC），两条折线分别是 mallocSpan 和 freeSpan。

用法:
    python3 plot_spans.py 1020.log            # 输出 spans.png
    python3 plot_spans.py 1020.log out.png    # 指定输出文件

依赖: matplotlib (pip install matplotlib)
"""
import re
import sys

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt

NUMGC_RE = re.compile(r"\[GC\.Heap\].*NumGC: (\d+)")
TOTAL_RE = re.compile(
    r"\[GC\.Total\].*mallocSpan: (-?\d+), freeSpan: (-?\d+)")


def parse_snapshots(path):
    """读取日志，按快照顺序返回 [(numgc, mallocSpan, freeSpan)]。"""
    snaps = []
    in_block = False
    numgc = None
    total = None
    with open(path, encoding="utf-8", errors="replace") as f:
        for raw in f:
            line = raw
            i = line.find("]: ")
            if i != -1:
                line = line[i + 3:]
            if "memPool stats dump" in line:
                in_block = True
                numgc = None
                total = None
                continue
            if not in_block:
                continue
            m = NUMGC_RE.search(line)
            if m:
                numgc = int(m.group(1))
                continue
            m = TOTAL_RE.search(line)
            if m:
                total = (int(m.group(1)), int(m.group(2)))
                continue
            if "===============================" in line:
                if total is not None:
                    snaps.append((numgc, total[0], total[1]))
                in_block = False
    if not snaps:
        raise SystemExit("日志中没有找到 [GC.Total] 快照")
    return snaps


def main():
    if len(sys.argv) < 2:
        raise SystemExit("用法: python3 plot_spans.py <log> [out.png]")
    log_path = sys.argv[1]
    out_path = sys.argv[2] if len(sys.argv) > 2 else "spans.png"

    snaps = parse_snapshots(log_path)
    x = [s[0] if s[0] is not None else i + 1
         for i, s in enumerate(snaps)]
    malloc_span = [s[1] for s in snaps]
    free_span = [s[2] for s in snaps]

    fig, ax = plt.subplots(figsize=(11, 5.5))

    ax.plot(x, malloc_span, color="#6366f1", marker="o", ms=3,
            lw=1.8, label="mallocSpan")
    ax.plot(x, free_span, color="#ef4444", marker="s", ms=3,
            lw=1.8, label="freeSpan")

    ax.set_title("mallocSpan / freeSpan over GC count (per GC snapshot)")
    ax.set_xlabel("NumGC (GC count)")
    ax.set_ylabel("spans")
    ax.grid(linestyle="--", alpha=0.4)
    ax.legend(loc="upper left")
    plt.tight_layout()
    plt.savefig(out_path, dpi=150)
    print("saved: {} ({} snapshots)".format(out_path, len(snaps)))


if __name__ == "__main__":
    main()
