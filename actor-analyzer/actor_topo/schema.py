"""Schema 加载与校验：驱动日志解析的可配置层。

Schema 定义两类记录的解析方式（正则 + 字段类型映射）以及 PID 结构体的字段契约。
支持 YAML / JSON（按扩展名嗅探）。
"""
from __future__ import annotations

import json
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, Optional

from .pid import PidParser


class SchemaError(Exception):
    pass


@dataclass
class RecordSpec:
    name: str
    pattern: re.Pattern
    fields: Dict[str, str]  # 线上捕获组名 -> 类型 (string|int|pid)


@dataclass
class Schema:
    path: Optional[Path]
    records: Dict[str, RecordSpec]
    pid_parser: PidParser

    def record_fields(self, name: str) -> Dict[str, str]:
        return self.records[name].fields


def load_schema(path: str) -> Schema:
    p = Path(path)
    raw = _read_raw(p)

    records_raw = raw.get("records", {})
    if not records_raw:
        raise SchemaError("schema 缺少 'records' 段")

    records: Dict[str, RecordSpec] = {}
    for name, spec in records_raw.items():
        pattern = spec.get("pattern")
        if not pattern:
            raise SchemaError(f"record {name!r} 缺少 'pattern'")
        fields = spec.get("fields", {})
        if not fields:
            raise SchemaError(f"record {name!r} 缺少 'fields'")
        try:
            compiled = re.compile(pattern)
        except re.error as e:
            raise SchemaError(f"record {name!r} 的正则无法编译: {e}") from e
        records[name] = RecordSpec(name=name, pattern=compiled, fields=fields)

    pid_parser = PidParser(raw.get("pid", {}))
    return Schema(path=p, records=records, pid_parser=pid_parser)


def _read_raw(p: Path) -> Dict[str, Any]:
    if not p.exists():
        raise SchemaError(f"schema 文件不存在: {p}")
    text = p.read_text(encoding="utf-8")
    if p.suffix.lower() == ".json":
        return json.loads(text)
    try:
        import yaml
    except ImportError as e:  # pragma: no cover
        raise SchemaError("解析 YAML 需要 PyYAML，请 `pip install pyyaml`") from e
    return yaml.safe_load(text)
