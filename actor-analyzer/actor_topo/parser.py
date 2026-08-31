"""日志解析：日志行 → 规范化记录（ActorRecord / MessageRecord）。

两遍、流式：第一遍收集 Actor 记录以建立 pid→type 映射；第二遍处理消息记录。
逐条消息不落内存（图构建阶段边聚合），仅保留记录对象供后续使用。
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Dict, List, Optional, TextIO

from .model import ActorRecord, MessageRecord, PID
from .schema import Schema


@dataclass
class ParseResult:
    actors: List[ActorRecord] = field(default_factory=list)
    messages: List[MessageRecord] = field(default_factory=list)
    skipped_lines: int = 0
    total_lines: int = 0
    # pid identity -> actor_type（第一遍建立，供 type 粒度聚合与未知类型回退）
    pid_to_type: Dict[str, str] = field(default_factory=dict)

    def actor_type_of(self, pid: Optional[PID]) -> Optional[str]:
        if pid is None:
            return None
        return self.pid_to_type.get(pid.identity())


def parse_log(schema: Schema, stream: TextIO) -> ParseResult:
    """从文本流解析日志。返回 ParseResult。"""
    result = ParseResult()
    actor_spec = schema.records.get("actor")
    message_spec = schema.records.get("message")

    # 第一遍：Actor 记录 + 建立 pid→type 映射
    for raw in stream:
        result.total_lines += 1
        line = raw.rstrip("\n").rstrip("\r")
        if actor_spec is not None and (m := actor_spec.pattern.match(line)):
            rec = _parse_actor(schema, actor_spec, m)
            result.actors.append(rec)
            if rec.self is not None:
                result.pid_to_type[rec.self.identity()] = rec.actor_type
            continue
        if message_spec is not None and (m := message_spec.pattern.match(line)):
            result.messages.append(_parse_message(schema, message_spec, m))
            continue
        result.skipped_lines += 1

    return result


def _parse_actor(schema: Schema, spec, m) -> ActorRecord:
    f = spec.fields
    return ActorRecord(
        actor_type=_string(m, f, "actor_type"),
        actor_size=_int(m, f, "actor_size"),
        self=_pid(schema, m, f, "self"),
        parent=_pid(schema, m, f, "parent"),
    )


def _parse_message(schema: Schema, spec, m) -> MessageRecord:
    f = spec.fields
    return MessageRecord(
        msg_type=_string(m, f, "msg_type"),
        msg_size=_int(m, f, "msg_size"),
        sender=_pid(schema, m, f, "sender"),
        self=_pid(schema, m, f, "self"),
        parent=_pid(schema, m, f, "parent"),
    )


def _string(m, fields: Dict[str, str], name: str) -> str:
    return m.group(name) if name in fields and m.groupdict().get(name) is not None else ""


def _int(m, fields: Dict[str, str], name: str) -> int:
    raw = m.group(name) if name in fields else None
    if raw is None:
        return 0
    try:
        return int(raw)
    except (TypeError, ValueError):
        return 0


def _pid(schema: Schema, m, fields: Dict[str, str], name: str) -> Optional[PID]:
    raw = m.groupdict().get(name) if name in fields else None
    if raw is None:
        return None
    return schema.pid_parser.parse(raw)
