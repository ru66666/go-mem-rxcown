"""验证 Schema 可配置性：格式变化（字段重排 / 别名）只需改 Schema，不改代码。"""
import io

import pytest

from actor_topo.parser import parse_log
from actor_topo.schema import SchemaError, load_schema

# 字段重排后的日志行（Same fields, different order）
REORDERED_ACTOR = (
    '[ACTOR] INFO Actor statistics统计 Parent="Address:nonhost, Id:sm, GroupID:0, Master:{nil}" '
    'Self="Address:nonhost, Id:sm/cellwall, GroupID:0, Master:{nil}" ActorSize=8 ActorType="*"'
)


def test_json_schema_and_reordered_fields(tmp_path):
    schema = {
        "version": 1,
        "records": {
            "actor": {
                # 捕获组顺序与默认不同，但规范名一致
                "pattern": (
                    r'^\[ACTOR\]\s+INFO\s+Actor\s+statistics统计\s+'
                    r'Parent="(?P<parent>[^"]*)"\s+Self="(?P<self>[^"]*)"\s+'
                    r'ActorSize=(?P<actor_size>\d+)\s+ActorType="(?P<actor_type>[^"]*)"\s*$'
                ),
                "fields": {
                    "actor_type": "string",
                    "actor_size": "int",
                    "self": "pid",
                    "parent": "pid",
                },
            }
        },
        "pid": {"null_literal": "nil",
                "fields": {"Address": "string", "Id": "string",
                           "GroupID": "int", "Master": "pid"}},
    }
    import json
    p = tmp_path / "schema.json"
    p.write_text(json.dumps(schema), encoding="utf-8")

    s = load_schema(str(p))
    result = parse_log(s, io.StringIO(REORDERED_ACTOR + "\n"))
    assert len(result.actors) == 1
    a = result.actors[0]
    assert a.actor_type == "*"
    assert a.actor_size == 8
    assert a.self.id == "sm/cellwall"
    assert a.parent.id == "sm"


def test_missing_pattern_raises(tmp_path):
    import json
    p = tmp_path / "bad.json"
    p.write_text(json.dumps({"records": {"actor": {"fields": {}}}}), encoding="utf-8")
    with pytest.raises(SchemaError):
        load_schema(str(p))


def test_pid_alias_via_explicit_name(tmp_path):
    """PID 字段名别名：wire 名 `Addr` 显式映射到模型字段 `address`。"""
    import json
    schema = {
        "version": 1,
        "records": {
            "actor": {
                "pattern": r'^A="(?P<actor_type>[^"]*)"\s+Z=(?P<actor_size>\d+)\s+S="(?P<self>[^"]*)"\s+P="(?P<parent>[^"]*)"$',
                "fields": {"actor_type": "string", "actor_size": "int",
                           "self": "pid", "parent": "pid"},
            }
        },
        "pid": {"null_literal": "nil",
                "fields": {"Addr": {"type": "string", "name": "address"},
                           "Id": "string", "GroupID": "int", "Master": "pid"}},
    }
    p = tmp_path / "alias.json"
    p.write_text(json.dumps(schema), encoding="utf-8")

    s = load_schema(str(p))
    line = 'A="*" Z=8 S="Addr:nonhost, Id:x, GroupID:0, Master:nil" P="nil"\n'
    result = parse_log(s, io.StringIO(line))
    assert result.actors[0].self.address == "nonhost"
    assert result.actors[0].self.id == "x"
