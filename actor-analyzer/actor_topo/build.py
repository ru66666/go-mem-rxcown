"""图构建：从 ParseResult 构建 NetworkX 有向图。

产出两类图：
- 拓扑图（父子关系）：parent → child，节点为 ActorType 或 PID 实例。
- 通信图（消息流）：sender → receiver，边聚合 count / total_bytes / msg_types。

粒度由 `by` 参数控制：`type`（按 ActorType 聚合）或 `pid`（按实例聚合）。
"""
from __future__ import annotations

from collections import Counter
from typing import Dict, Optional, Tuple

import networkx as nx

from . import metrics as M
from .model import PID
from .parser import ParseResult

# 合成节点标签
ROOT_NODE = "(root)"        # parent 为 nil
UNKNOWN_NODE = "(unknown)"  # 查不到类型的父/端点
SYSTEM_NODE = "(system)"    # sender 为 nil（如 *actor.Started）


def _node_key(by: str, pid: Optional[PID], pid_to_type: Dict[str, str]) -> Optional[str]:
    """把 PID（可能为 nil）映射为节点键；type 粒度需查类型映射。"""
    if pid is None:
        return None
    if by == "pid":
        return pid.identity()
    return pid_to_type.get(pid.identity())


def build_topology(result: ParseResult, by: str = "type") -> nx.DiGraph:
    """构建父子拓扑图。节点带 actor_size 统计、实例数与 actor_type。"""
    g = nx.DiGraph()
    node_meta: Dict[str, Dict] = {}

    def _touch(node: str, size: int, actor_type: str):
        meta = node_meta.setdefault(
            node,
            {"instances": 0, "size_sum": 0, "size_max": 0, "size_min": None,
             "actor_type": actor_type},
        )
        meta["instances"] += 1
        meta["size_sum"] += size
        meta["size_max"] = max(meta["size_max"], size)
        meta["size_min"] = size if meta["size_min"] is None else min(meta["size_min"], size)
        if not meta.get("actor_type"):
            meta["actor_type"] = actor_type

    for rec in result.actors:
        child = _node_key(by, rec.self, result.pid_to_type)
        child_type = rec.actor_type
        if child is None:
            child = UNKNOWN_NODE

        if rec.parent is None:
            parent = ROOT_NODE
            parent_type = ROOT_NODE
        else:
            parent = _node_key(by, rec.parent, result.pid_to_type)
            parent_type = result.pid_to_type.get(rec.parent.identity(), UNKNOWN_NODE)
            if parent is None:
                parent = UNKNOWN_NODE
                parent_type = UNKNOWN_NODE

        _touch(child, rec.actor_size, child_type)
        if parent not in g:
            g.add_node(parent, actor_type=parent_type)
        if child not in g:
            g.add_node(child)
        if g.has_edge(parent, child):
            g[parent][child]["count"] += 1
        else:
            g.add_edge(parent, child, count=1)

    # 写回节点属性（size 统计 + actor_type）
    for node, meta in node_meta.items():
        g.nodes[node].update(
            {
                "instances": meta["instances"],
                "size_sum": meta["size_sum"],
                "size_avg": meta["size_sum"] / meta["instances"],
                "size_max": meta["size_max"],
                "size_min": meta["size_min"],
                "actor_type": meta["actor_type"],
            }
        )
    return g


def build_communication(
    result: ParseResult,
    by: str = "type",
    exclude_nil_sender: bool = False,
) -> nx.DiGraph:
    """构建通信热点图。边聚合 count / total_bytes / msg_types，节点带 actor_type。"""

    def _resolve(pid: Optional[PID]) -> Tuple[Optional[str], Optional[str]]:
        """返回 (key, actor_type)。"""
        if pid is None:
            return None, None
        t = result.pid_to_type.get(pid.identity())
        if by == "pid":
            return pid.identity(), (t or UNKNOWN_NODE)
        return (t or UNKNOWN_NODE), (t or UNKNOWN_NODE)

    g = nx.DiGraph()
    for rec in result.messages:
        recv_key, recv_type = _resolve(rec.self)
        if recv_key is None:
            recv_key, recv_type = UNKNOWN_NODE, UNKNOWN_NODE

        if rec.sender is None:
            send_key, send_type = SYSTEM_NODE, SYSTEM_NODE
        else:
            send_key, send_type = _resolve(rec.sender)
            if send_key is None:
                send_key, send_type = UNKNOWN_NODE, UNKNOWN_NODE

        if send_key == SYSTEM_NODE and exclude_nil_sender:
            continue

        if send_key not in g:
            g.add_node(send_key, actor_type=send_type)
        if recv_key not in g:
            g.add_node(recv_key, actor_type=recv_type)

        if g.has_edge(send_key, recv_key):
            e = g[send_key][recv_key]
            e[M.COUNT] += 1
            e[M.TOTAL_BYTES] += rec.msg_size
            e[M.MSG_TYPES][rec.msg_type] += 1
        else:
            g.add_edge(
                send_key,
                recv_key,
                count=1,
                total_bytes=rec.msg_size,
                msg_types=Counter({rec.msg_type: 1}),
            )

    _annotate_nodes(g)
    return g


def _annotate_nodes(g: nx.DiGraph) -> None:
    """为通信图节点补充出/入度流量统计。"""
    for node in g.nodes:
        out_c = sum(g[node][t].get(M.COUNT, 0) for t in g.successors(node))
        in_c = sum(g[s][node].get(M.COUNT, 0) for s in g.predecessors(node))
        out_b = sum(g[node][t].get(M.TOTAL_BYTES, 0) for t in g.successors(node))
        in_b = sum(g[s][node].get(M.TOTAL_BYTES, 0) for s in g.predecessors(node))
        g.nodes[node].update(
            {
                "out_count": out_c,
                "in_count": in_c,
                "out_bytes": out_b,
                "in_bytes": in_b,
                "total_flow": out_c + in_c,
                "total_bytes": out_b + in_b,
            }
        )


def top_links(
    g: nx.DiGraph, metric: str, top: int = 20
) -> list[Tuple[str, str, float, dict]]:
    """按所选度量返回排序后的热点边列表：(sender, receiver, weight, attrs)。"""
    scored = []
    for u, v, data in g.edges(data=True):
        scored.append((u, v, M.edge_weight(data, metric), data))
    scored.sort(key=lambda t: t[2], reverse=True)
    return scored[:top]
