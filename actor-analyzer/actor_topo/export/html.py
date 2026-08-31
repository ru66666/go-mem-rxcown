"""pyvis 交互式 HTML 导出。

节点大小 ∝ 节点总出入权重，边粗细 ∝ 所选度量，节点按类型着色，
hover 显示 count / bytes / top msg_type。
"""
from __future__ import annotations

import hashlib
import json
from pathlib import Path
from typing import Optional

import networkx as nx

from .. import metrics as M

# 色盲友好、浅底可读的离散调色板（节点按类型着色）
_PALETTE = [
    "#4e79a7", "#f28e2b", "#e15759", "#76b7b2", "#59a14f",
    "#edc948", "#b07aa1", "#ff9da7", "#9c755f", "#bab0ac",
    "#86bcb6", "#d37295", "#8cd17d", "#f1ce63", "#499894", "#e8a838",
]


def _color_for_type(t: str) -> str:
    # 确定性哈希，跨运行颜色稳定
    idx = int(hashlib.md5(t.encode("utf-8")).hexdigest(), 16) % len(_PALETTE)
    return _PALETTE[idx]


def export_html(
    g: nx.DiGraph,
    path: Path,
    metric: str,
    title: str = "Actor graph",
    height: str = "750px",
    directed: bool = True,
    node_size_field: str = "total_flow",
) -> None:
    """将 NetworkX 图导出为 pyvis 交互式 HTML。"""
    try:
        from pyvis.network import Network
    except ImportError as e:  # pragma: no cover
        raise RuntimeError(
            "导出 HTML 需要 pyvis，请 `pip install pyvis`"
        ) from e

    net = Network(
        height=height,
        width="100%",
        directed=directed,
        bgcolor="#ffffff",
        font_color="#333333",
        heading=title,
    )

    # 节点：大小 ∝ 指定字段（默认出入流量），颜色 ∝ actor_type
    sizes = _scale([g.nodes[n].get(node_size_field, 0) for n in g.nodes])
    for n, size in zip(g.nodes, sizes):
        label = _short(n)
        title_text = _node_title(g, n)
        node_type = g.nodes[n].get("actor_type", n)
        net.add_node(
            n,
            label=label,
            title=title_text,
            size=size,
            color=_color_for_type(str(node_type)),
        )

    # 边：粗细 ∝ 所选度量
    weights = [M.edge_weight(d, metric) for _, _, d in g.edges(data=True)]
    scaled = _scale(weights, min_w=1, max_w=12)
    for (u, v, d), w in zip(g.edges(data=True), scaled):
        title_text = _edge_title(d)
        net.add_edge(u, v, value=w, title=title_text)

    net.set_options(json.dumps(_options()))
    net.save_graph(str(path))


def _scale(values, min_w: float = 6, max_w: float = 40) -> list[float]:
    if not values:
        return []
    lo, hi = min(values), max(values)
    if hi == lo:
        return [(min_w + max_w) / 2 for _ in values]
    return [
        min_w + (v - lo) / (hi - lo) * (max_w - min_w) for v in values
    ]


def _short(n: str) -> str:
    # 长 id（如 nonhost/sm/cellwall）截断展示
    return n if len(n) <= 40 else "…" + n[-39:]


def _node_title(g: nx.DiGraph, n: str) -> str:
    d = g.nodes[n]
    parts = [f"<b>{n}</b>"]
    if d.get("actor_type"):
        parts.append(f"type: {d['actor_type']}")
    if "instances" in d:
        parts.append(f"instances: {d['instances']}")
    if "size_avg" in d:
        parts.append(f"size_avg: {d['size_avg']:.1f}")
    if "size_max" in d:
        parts.append(f"size_max: {d['size_max']}")
    if "out_count" in d:
        parts.append(f"out: {d['out_count']:,} / in: {d['in_count']:,}")
    if "total_bytes" in d:
        parts.append(f"bytes: {d['total_bytes']:,}")
    return "<br>".join(parts)


def _edge_title(d: dict) -> str:
    parts = [
        f"count: {d.get(M.COUNT, 0):,}",
        f"bytes: {d.get(M.TOTAL_BYTES, 0):,}",
        f"avg: {M.edge_weight(d, 'avg_bytes'):.1f}",
    ]
    mt = M.summarize_msg_types(d)
    if mt:
        parts.append(f"msg: {mt}")
    return "<br>".join(parts)


def _options() -> dict:
    return {
        "physics": {
            "enabled": True,
            "barnesHut": {
                "gravitationalConstant": -8000,
                "springLength": 120,
                "springConstant": 0.04,
            },
        },
        "interaction": {"hover": True, "tooltipDelay": 100},
    }
