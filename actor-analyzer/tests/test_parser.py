import io

from actor_topo.parser import parse_log
from actor_topo.schema import load_schema


SCHEMA_PATH = "schema.yaml"

ACTOR_LINE = (
    '[ACTOR] INFO Actor statistics统计 ActorType="*" ActorSize=8  '
    'Self="Address:nonhost, Id:sm/cellwall, GroupID:0, Master:{nil}" '
    'Parent="Address:nonhost, Id:sm, GroupID:0, Master:{nil}"'
)

MESSAGE_LINE = (
    '[ACTOR] INFO Receive统计 MsgType="*actor.Started" MsgSize=8 Sender="nil" '
    'Self="Address:nonhost, Id:sm, GroupID:0, Master:{nil}" Parent="nil"'
)


def _schema():
    return load_schema(SCHEMA_PATH)


def test_parse_actor_line():
    result = parse_log(_schema(), io.StringIO(ACTOR_LINE + "\n"))
    assert result.skipped_lines == 0
    assert len(result.actors) == 1
    a = result.actors[0]
    assert a.actor_type == "*"
    assert a.actor_size == 8
    assert a.self.id == "sm/cellwall"
    assert a.parent.id == "sm"


def test_parse_message_line():
    result = parse_log(_schema(), io.StringIO(MESSAGE_LINE + "\n"))
    assert len(result.messages) == 1
    m = result.messages[0]
    assert m.msg_type == "*actor.Started"
    assert m.msg_size == 8
    assert m.sender is None
    assert m.self.id == "sm"
    assert m.parent is None


def test_skip_unmatched_lines():
    text = "some random line\n" + ACTOR_LINE + "\nanother noise\n"
    result = parse_log(_schema(), io.StringIO(text))
    assert result.total_lines == 3
    assert result.skipped_lines == 2
    assert len(result.actors) == 1


def test_pid_to_type_mapping():
    text = ACTOR_LINE + "\n" + MESSAGE_LINE + "\n"
    result = parse_log(_schema(), io.StringIO(text))
    # 映射只由 Self PID 建立（cellwall），父 PID（sm）不在其中
    assert result.pid_to_type["nonhost/sm/cellwall"] == "*"
    assert "nonhost/sm" not in result.pid_to_type
