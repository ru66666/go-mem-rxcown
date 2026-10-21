#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
从 mpool 聚合日志中提取最后一个快照的 [GC.Span] 数据，
按 mallocSpan 数量画 objectSize 分布柱状图。

用法:
    python3 plot_span_dist.py 1020.log            # 输出 span_dist.png
    python3 plot_span_dist.py 1020.log out.png    # 指定输出文件

依赖: matplotlib (pip install matplotlib)
"""
import re
import sys

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt

PREFIX_RE = re.compile(r"\[22132\]: ")  # 日志前缀（进程号），按需可放宽


def parse_last_snapshot(path):
    """读取日志，返回最后一个聚合快照的 [(objectSize, mallocSpan)]，按 size 升序。"""
    span = {}
    in_block = False
    found = False
    with open(path, encoding="utf-8", errors="replace") as f:
        for raw in f:
            line = raw
            i = line.find("]: ")
            if i != -1:
                line = line[i + 3:]
            if "memPool stats dump" in line:
                in_block = True
                span = {}
                continue
            if not in_block:
                continue
            m = re.search(r"\[GC\.Span\] objectsize=(\d+), mallocSpan=(\d+)", line)
            if m:
                span[int(m.group(1))] = int(m.group(2))
                continue
            if "===============================" in line:
                if span:
                    found = True  # 记住最后一个完整快照
                in_block = False
    if not found:
        raise SystemExit("日志中没有找到完整的聚合快照")
    return sorted(span.items())


def main():
    if len(sys.argv) < 2:
        raise SystemExit("用法: python3 plot_span_dist.py <log> [out.png]")
    log_path = sys.argv[1]
    out_path = sys.argv[2] if len(sys.argv) > 2 else "span_dist.png"

    data = parse_last_snapshot(log_path)
    sizes = ["{}B".format(s) for s, _ in data]
    counts = [ms for _, ms in data]
    total = sum(counts)

    fig, ax = plt.subplots(figsize=(max(9, len(sizes) * 0.75), 5.5))
    bars = ax.bar(sizes, counts, color="#6366f1", edgecolor="white")

    # 柱顶标注数量与占比
    for bar, c in zip(bars, counts):
        pct = c / total * 100 if total else 0
        ax.annotate("{}\n{:.1f}%".format(c, pct),
                    xy=(bar.get_x() + bar.get_width() / 2, bar.get_height()),
                    xytext=(0, 3), textcoords="offset points",
                    ha="center", va="bottom", fontsize=8)

    ax.set_title("mallocSpan distribution by objectSize (last GC snapshot, total={:,})".format(total))
    ax.set_xlabel("objectSize")
    ax.set_ylabel("mallocSpan count")
    ax.set_yscale("log")  # 数量差距大（10 万 vs 1），对数轴
    ax.grid(axis="y", linestyle="--", alpha=0.4)
    plt.xticks(rotation=30, ha="right")
    plt.tight_layout()
    plt.savefig(out_path, dpi=150)
    print("saved: {} (total mallocSpan={})".format(out_path, total))


if __name__ == "__main__":
    main()
