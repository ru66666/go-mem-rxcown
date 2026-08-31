"""PID 解析器：递归解析 `Address:..., Id:..., GroupID:..., Master:{...}` 结构体字面量。

由 Schema 的 `pid` 段驱动，字段名与类型均可配置，天然支持 `Master:{nil}` 的嵌套。
"""
from __future__ import annotations

import re
from typing import Any, Dict, List, Optional, Tuple

from .model import PID


def _snake_case(name: str) -> str:
    """PascalCase → snake_case（Go 字段名的常见默认映射）。

    Address → address, Id → id, GroupID → group_id, Master → master。
    若需要自定义模型字段名，用 `{type: ..., name: ...}` 显式指定即可。
    """
    s = re.sub(r"(.)([A-Z][a-z]+)", r"\1_\2", name)
    s = re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", s)
    return s.lower()


class PidParser:
    def __init__(self, pid_spec: Optional[Dict[str, Any]] = None):
        pid_spec = pid_spec or {}
        self.null_literal: str = pid_spec.get("null_literal", "nil")
        # fields: 线上字段名 -> {"type": string|int|pid, "name": 模型字段名}
        self.fields: Dict[str, Dict[str, str]] = self._normalize_fields(
            pid_spec.get("fields", {})
        )

    @staticmethod
    def _normalize_fields(raw: Dict[str, Any]) -> Dict[str, Dict[str, str]]:
        out: Dict[str, Dict[str, str]] = {}
        for wire_name, spec in raw.items():
            if isinstance(spec, str):
                out[wire_name] = {"type": spec, "name": _snake_case(wire_name)}
            elif isinstance(spec, dict):
                out[wire_name] = {
                    "type": spec.get("type", "string"),
                    "name": spec.get("name", _snake_case(wire_name)),
                }
            else:
                raise ValueError(f"invalid pid field spec for {wire_name!r}: {spec!r}")
        return out

    def parse(self, value: Optional[str]) -> Optional[PID]:
        if value is None:
            return None
        value = value.strip()
        if value == "" or value == self.null_literal:
            return None
        # 允许整体被 `{...}` 包裹（嵌套场景）
        if value.startswith("{") and value.endswith("}"):
            value = value[1:-1].strip()
            if value == self.null_literal:
                return None

        data: Dict[str, Any] = {}
        for wire_name, val in self._split_top_level(value):
            spec = self.fields.get(wire_name)
            if spec is None:
                # 未知字段：前向兼容，忽略
                continue
            data[spec["name"]] = self._coerce(spec["type"], val)
        return PID(**data)

    def _coerce(self, ftype: str, val: str) -> Any:
        if ftype == "string":
            return val.strip()
        if ftype == "int":
            try:
                return int(val.strip())
            except ValueError:
                return 0
        if ftype == "pid":
            return self.parse(val.strip())
        raise ValueError(f"unknown pid field type: {ftype!r}")

    @staticmethod
    def _split_top_level(s: str) -> List[Tuple[str, str]]:
        """按顶层逗号切分为 Key:Value 对，跟踪 `{}` 深度，避免切碎嵌套结构。"""
        pairs: List[str] = []
        depth = 0
        cur: List[str] = []
        for ch in s:
            if ch == "{":
                depth += 1
                cur.append(ch)
            elif ch == "}":
                depth -= 1
                cur.append(ch)
            elif ch == "," and depth == 0:
                pairs.append("".join(cur))
                cur = []
            else:
                cur.append(ch)
        if "".join(cur).strip():
            pairs.append("".join(cur))

        result: List[Tuple[str, str]] = []
        for pair in pairs:
            if ":" not in pair:
                continue
            key, val = pair.split(":", 1)
            result.append((key.strip(), val.strip()))
        return result
