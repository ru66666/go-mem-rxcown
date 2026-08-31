"""报告生成：CSV 文件 + 文本摘要（stdout）。

针对通信图生成：nodes.csv / edges.csv / top_links.csv / actor_stats.csv / msg_stats.csv。
拓扑图生成：topo_nodes.csv / topo_edges.csv。
"""
from __future__ import annotations

import csv
from collections import Counter
from pathlib import Path
from typing import Dict, Optional

import networkx as nx

from . import build as B
from . import metrics as M


def write_reports(
    topo: nx.DiGraph,
    comm: nx.DiGraph,
    out_dir: Path,
    metric: str,
    top: int = 20,
) -> None:
    out_dir.mkdir(parents=True, exist_ok=True)

    _write_nodes_edges(comm, out_dir / "nodes.csv", out_dir / "edges.csv", metric)
    _write_top_links(comm, out_dir / "top_links.csv", metric, top)
    _write_msg_stats(comm, out_dir / "msg_stats.csv")
    _write_actor_stats(topo, comm, out_dir / "actor_stats.csv")
    _write_topo(topo, out_dir / "topo_nodes.csv", out_dir / "topo_edges.csv")


def _write_nodes_edges(
    g: nx.DiGraph, nodes_path: Path, edges_path: Path, metric: str
) -> None:
    with nodes_path.open("w", newline="", encoding="utf-8") as f:
        w = csv.writer(f)
        w.writerow(
            ["node", "out_count", "in_count", "out_bytes", "in_bytes",
             "total_flow", "total_bytes"]
        )
        for n, d in g.nodes(data=True):
            w.writerow(
                [
                    n,
                    d.get("out_count", 0),
                    d.get("in_count", 0),
                    d.get("out_bytes", 0),
                    d.get("in_bytes", 0),
                    d.get("total_flow", 0),
                    d.get("total_bytes", 0),
                ]
            )

    with edges_path.open("w", newline="", encoding="utf-8") as f:
        w = csv.writer(f)
        w.writerow(
            ["sender", "receiver", "count", "total_bytes", "avg_bytes",
             "weight", "top_msg_types"]
        )
        for u, v, d in g.edges(data=True):
            w.writerow(
                [
                    u,
                    v,
                    d.get(M.COUNT, 0),
                    d.get(M.TOTAL_BYTES, 0),
                    M.edge_weight(d, "avg_bytes"),
                    M.edge_weight(d, metric),
                    M.summarize_msg_types(d),
                ]
            )


def _write_top_links(
    g: nx.DiGraph, path: Path, metric: str, top: int
) -> None:
    with path.open("w", newline="", encoding="utf-8") as f:
        w = csv.writer(f)
        w.writerow(["rank", "sender", "receiver", "weight", "count",
                    "total_bytes", "top_msg_types"])
        for i, (u, v, weight, d) in enumerate(B.top_links(g, metric, top), 1):
            w.writerow(
                [i, u, v, weight, d.get(M.COUNT, 0),
                 d.get(M.TOTAL_BYTES, 0), M.summarize_msg_types(d)]
            )


def _write_msg_stats(g: nx.DiGraph, path: Path) -> None:
    counter: Counter = Counter()
    bytes_counter: Counter = Counter()
    for _, _, d in g.edges(data=True):
        for mt, cnt in d.get(M.MSG_TYPES, {}).items():
            counter[mt] += cnt
    # 字节按消息类型无法从边聚合精确还原（边只存了总量），这里按类型计数即可
    with path.open("w", newline="", encoding="utf-8") as f:
        w = csv.writer(f)
        w.writerow(["msg_type", "count"])
        for mt, cnt in counter.most_common():
            w.writerow([mt, cnt])


def _write_actor_stats(
    topo: nx.DiGraph, comm: nx.DiGraph, path: Path
) -> None:
    """合并拓扑图（类型→大小）与通信图（类型→流量）的节点统计。"""
    rows = []
    for n, d in topo.nodes(data=True):
        flow = comm.nodes[n] if comm.has_node(n) else {}
        rows.append(
            {
                "node": n,
                "instances": d.get("instances", 0),
                "size_sum": d.get("size_sum", 0),
                "size_avg": round(d.get("size_avg", 0), 2),
                "size_max": d.get("size_max", 0),
                "out_count": flow.get("out_count", 0),
                "in_count": flow.get("in_count", 0),
                "total_bytes": flow.get("total_bytes", 0),
            }
        )
    # 拓扑图可能没有、但通信图有的节点（如 system/unknown）
    for n in comm.nodes:
        if n in topo:
            continue
        d = comm.nodes[n]
        rows.append(
            {
                "node": n,
                "instances": 0,
                "size_sum": 0,
                "size_avg": 0,
                "size_max": 0,
                "out_count": d.get("out_count", 0),
                "in_count": d.get("in_count", 0),
                "total_bytes": d.get("total_bytes", 0),
            }
        )
    rows.sort(key=lambda r: r["total_bytes"], reverse=True)

    with path.open("w", newline="", encoding="utf-8") as f:
        w = csv.writer(f)
        w.writerow(
            ["node", "instances", "size_sum", "size_avg", "size_max",
             "out_count", "in_count", "total_bytes"]
        )
        for r in rows:
            w.writerow(
                [r["node"], r["instances"], r["size_sum"], r["size_avg"],
                 r["size_max"], r["out_count"], r["in_count"], r["total_bytes"]]
            )


def _write_topo(topo: nx.DiGraph, nodes_path: Path, edges_path: Path) -> None:
    with nodes_path.open("w", newline="", encoding="utf-8") as f:
        w = csv.writer(f)
        w.writerow(["node", "instances", "size_sum", "size_avg", "size_max", "size_min"])
        for n, d in topo.nodes(data=True):
            w.writerow(
                [n, d.get("instances", 0), d.get("size_sum", 0),
                 round(d.get("size_avg", 0), 2), d.get("size_max", 0),
                 d.get("size_min", 0)]
            )
    with edges_path.open("w", newline="", encoding="utf-8") as f:
        w = csv.writer(f)
        w.writerow(["parent", "child", "count"])
        for u, v, d in topo.edges(data=True):
            w.writerow([u, v, d.get("count", 0)])


def text_summary(
    topo: nx.DiGraph,
    comm: nx.DiGraph,
    metric: str,
    top: int = 20,
    skipped: int = 0,
    total: int = 0,
) -> str:
    """生成面向 stdout 的文本摘要。"""
    lines: list[str] = []
    lines.append("=" * 60)
    lines.append("Actor 通信热点分析报告")
    lines.append("=" * 60)
    if total:
        lines.append(f"日志总行数: {total}   跳过(未匹配)行数: {skipped}")
    lines.append(f"热点度量: {metric} ({M.METRIC_LABELS.get(metric, metric)})")
    lines.append("")

    lines.append(f"--- Top {top} 热点通信链路 (sender → receiver) ---")
    for i, (u, v, weight, d) in enumerate(B.top_links(comm, metric, top), 1):
        lines.append(
            f"  {i:>3}. {u} → {v}   weight={weight:,.1f}  "
            f"count={d.get(M.COUNT, 0):,}  bytes={d.get(M.TOTAL_BYTES, 0):,}  "
            f"[{M.summarize_msg_types(d)}]"
        )

    lines.append("")
    lines.append(f"--- Top {top} 热点消息类型 ---")
    counter: Counter = Counter()
    for _, _, d in comm.edges(data=True):
        for mt, cnt in d.get(M.MSG_TYPES, {}).items():
            counter[mt] += cnt
    for i, (mt, cnt) in enumerate(counter.most_common(top), 1):
        lines.append(f"  {i:>3}. {mt}  × {cnt:,}")

    lines.append("")
    lines.append(f"--- Top {top} 最热 Actor（按出入流量） ---")
    nodes = sorted(
        comm.nodes(data=True), key=lambda nd: nd[1].get("total_flow", 0), reverse=True
    )
    for i, (n, d) in enumerate(nodes[:top], 1):
        lines.append(
            f"  {i:>3}. {n}   流量={d.get('total_flow', 0):,}  "
            f"字节={d.get('total_bytes', 0):,}"
        )

    lines.append("")
    lines.append("--- 最大 Actor（按 size） ---")
    topo_nodes = sorted(
        topo.nodes(data=True), key=lambda nd: nd[1].get("size_max", 0), reverse=True
    )
    for i, (n, d) in enumerate(topo_nodes[:top], 1):
        lines.append(
            f"  {i:>3}. {n}   size_max={d.get('size_max', 0):,}  "
            f"size_avg={d.get('size_avg', 0):,.1f}  instances={d.get('instances', 0):,}"
        )

    lines.append("")
    lines.append(f"节点数: 拓扑 {topo.number_of_nodes()} / 通信 {comm.number_of_nodes()}")
    lines.append(f"边数:   拓扑 {topo.number_of_edges()} / 通信 {comm.number_of_edges()}")
    return "\n".join(lines)
