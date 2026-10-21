#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
从 mpool 聚合日志中提取所有快照的 [GC.Total] 数据，
画槽位构成折线图：全部槽位 PoolBytes / inUse / empty 三条线（MB）。

用法:
    python3 plot_slots.py 1020.log            # 输出 slots.png
    python3 plot_slots.py 1020.log out.png    # 指定输出文件

依赖: matplotlib (pip install matplotlib)
"""
import re
import sys

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt

TOTAL_RE = re.compile(
    r"\[GC\.Total\] Pools: (\d+), PoolBytes: (\d+) "
    r"\(inUse (\d+) \+ empty (\d+) \+ OB (\d+)\), InUsed: (-?\d+)")


def parse_snapshots(path):
    """读取日志，按聚合快照顺序返回 [(poolBytes, inUse, empty)]（字节）。"""
    snaps = []
    cur = None
    in_block = False
    with open(path, encoding="utf-8", errors="replace") as f:
        for raw in f:
            line = raw
            i = line.find("]: ")
            if i != -1:
                line = line[i + 3:]
            if "memPool stats dump" in line:
                in_block = True
                cur = None
                continue
            if not in_block:
                continue
            m = TOTAL_RE.search(line)
            if m:
                cur = (int(m.group(2)), int(m.group(3)), int(m.group(4)))
                continue
            if "===============================" in line:
                if cur is not None:
                    snaps.append(cur)
                in_block = False
    if not snaps:
        raise SystemExit("日志中没有找到 [GC.Total] 快照")
    return snaps


def main():
    if len(sys.argv) < 2:
        raise SystemExit("用法: python3 plot_slots.py <log> [out.png]")
    log_path = sys.argv[1]
    out_path = sys.argv[2] if len(sys.argv) > 2 else "slots.png"

    snaps = parse_snapshots(log_path)
    x = list(range(1, len(snaps) + 1))
    mb = lambda v: v / 2 ** 20
    pb = [mb(s[0]) for s in snaps]
    inuse = [mb(s[1]) for s in snaps]
    empty = [mb(s[2]) for s in snaps]

    fig, ax = plt.subplots(figsize=(11, 5.5))
    ax.plot(x, pb, color="#6366f1", marker="o", ms=3, lw=1.8,
            label="PoolBytes (total slots)")
    ax.plot(x, inuse, color="#10b981", marker="o", ms=3, lw=1.8,
            label="inUse")
    ax.plot(x, empty, color="#ef4444", marker="o", ms=3, lw=1.8,
            label="empty (retained)")

    ax.fill_between(x, empty, color="#ef4444", alpha=0.12)
    ax.fill_between(x, inuse, color="#10b981", alpha=0.10)

    ax.set_title("Pool slot usage: total = inUse + empty (per GC snapshot)")
    ax.set_xlabel("GC snapshot #")
    ax.set_ylabel("MB")
    ax.grid(linestyle="--", alpha=0.4)
    ax.legend()
    plt.tight_layout()
    plt.savefig(out_path, dpi=150)
    print("saved: {} ({} snapshots)".format(out_path, len(snaps)))


if __name__ == "__main__":
    main()
