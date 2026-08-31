"""规范化数据模型。

与解析 Schema 解耦：无论日志格式如何变化，下游（图构建、报告、导出）只依赖这里的
稳定结构。Schema 的字段契约（见 plan.md / README）负责把"线上字段名"映射到这些规范名。
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Optional


@dataclass
class PID:
    """Actor 的进程标识（protoactor-go 风格，含自定义 Master 嵌套）。"""

    address: str = ""
    id: str = ""
    group_id: int = 0
    master: Optional["PID"] = None

    def identity(self) -> str:
        """实例级唯一键：address/id。"""
        return f"{self.address}/{self.id}"

    def to_dict(self) -> dict:
        return {
            "address": self.address,
            "id": self.id,
            "group_id": self.group_id,
            "master": self.master.to_dict() if self.master else None,
        }


@dataclass
class ActorRecord:
    actor_type: str
    actor_size: int
    self: Optional[PID]
    parent: Optional[PID]

    def to_dict(self) -> dict:
        return {
            "actor_type": self.actor_type,
            "actor_size": self.actor_size,
            "self": self.self.to_dict() if self.self else None,
            "parent": self.parent.to_dict() if self.parent else None,
        }


@dataclass
class MessageRecord:
    msg_type: str
    msg_size: int
    sender: Optional[PID]
    self: Optional[PID]
    parent: Optional[PID]

    def to_dict(self) -> dict:
        return {
            "msg_type": self.msg_type,
            "msg_size": self.msg_size,
            "sender": self.sender.to_dict() if self.sender else None,
            "self": self.self.to_dict() if self.self else None,
            "parent": self.parent.to_dict() if self.parent else None,
        }
