"""热点度量注册表：把边聚合出的原始量映射为"热度"权重。

边在构建阶段累积三件事：count（频次）、total_bytes（累计字节）、msg_types（Counter）。
度量函数从这三者派生一个标量权重。用户可注册自定义度量（见 register_metric）。
"""
from __future__ import annotations

from collections import Counter
from typing import Callable, Dict, List

# 边属性名（与 build.py 保持一致）
COUNT = "count"
TOTAL_BYTES = "total_bytes"
MSG_TYPES = "msg_types"

EdgeAttrs = Dict[str, object]
MetricFn = Callable[[EdgeAttrs], float]


def _count(attrs: EdgeAttrs) -> float:
    return float(attrs.get(COUNT, 0) or 0)


def _total_bytes(attrs: EdgeAttrs) -> float:
    return float(attrs.get(TOTAL_BYTES, 0) or 0)


def _avg_bytes(attrs: EdgeAttrs) -> float:
    c = _count(attrs)
    if c == 0:
        return 0.0
    return _total_bytes(attrs) / c


def _volume(attrs: EdgeAttrs) -> float:
    # 频次 × 每条消息大小 == 累计字节
    return _total_bytes(attrs)


METRICS: Dict[str, MetricFn] = {
    "count": _count,
    "total_bytes": _total_bytes,
    "avg_bytes": _avg_bytes,
    "volume": _volume,
}

METRIC_LABELS: Dict[str, str] = {
    "count": "消息频次",
    "total_bytes": "累计字节",
    "avg_bytes": "平均每条字节",
    "volume": "频次×大小（累计字节）",
}


def register_metric(name: str, fn: MetricFn, label: str = "") -> None:
    """注册一个自定义度量。fn 接收边属性 dict，返回 float 权重。"""
    METRICS[name] = fn
    if label:
        METRIC_LABELS[name] = label


def available_metrics() -> List[str]:
    return sorted(METRICS.keys())


def edge_weight(attrs: EdgeAttrs, metric: str) -> float:
    if metric not in METRICS:
        raise KeyError(
            f"未知度量 {metric!r}，可用: {', '.join(available_metrics())}"
        )
    return METRICS[metric](attrs)


def edge_stats(attrs: EdgeAttrs) -> Dict[str, float]:
    """返回该边在所有度量下的取值（用于报告/导出）。"""
    return {name: fn(attrs) for name, fn in METRICS.items()}


def summarize_msg_types(attrs: EdgeAttrs, top: int = 5) -> str:
    c: Counter = attrs.get(MSG_TYPES)
    if not c:
        return ""
    return ", ".join(f"{t}×{n}" for t, n in c.most_common(top))
