import io

from actor_topo import metrics as M
from actor_topo.build import (
    ROOT_NODE,
    SYSTEM_NODE,
    UNKNOWN_NODE,
    build_communication,
    build_topology,
    top_links,
)
from actor_topo.parser import parse_log
from actor_topo.schema import load_schema


SAMPLE = """\
[ACTOR] INFO Actor statistics统计 ActorType="*" ActorSize=8  Self="Address:nonhost, Id:sm, GroupID:0, Master:{nil}" Parent="nil"
[ACTOR] INFO Actor statistics统计 ActorType="*" ActorSize=16 Self="Address:nonhost, Id:sm/cellwall, GroupID:0, Master:{nil}" Parent="Address:nonhost, Id:sm, GroupID:0, Master:{nil}"
[ACTOR] INFO Actor statistics统计 ActorType="*" ActorSize=24 Self="Address:nonhost, Id:sm/mover, GroupID:0, Master:{nil}" Parent="Address:nonhost, Id:sm, GroupID:0, Master:{nil}"
[ACTOR] INFO Receive统计 MsgType="*actor.Started" MsgSize=8 Sender="nil" Self="Address:nonhost, Id:sm, GroupID:0, Master:{nil}" Parent="nil"
[ACTOR] INFO Receive统计 MsgType="*actor.Started" MsgSize=8 Sender="nil" Self="Address:nonhost, Id:sm/cellwall, GroupID:0, Master:{nil}" Parent="Address:nonhost, Id:sm, GroupID:0, Master:{nil}"
[ACTOR] INFO Receive统计 MsgType="*msg.Move" MsgSize=64 Sender="Address:nonhost, Id:sm/cellwall, GroupID:0, Master:{nil}" Self="Address:nonhost, Id:sm/mover, GroupID:0, Master:{nil}" Parent="Address:nonhost, Id:sm, GroupID:0, Master:{nil}"
[ACTOR] INFO Receive统计 MsgType="*msg.Move" MsgSize=64 Sender="Address:nonhost, Id:sm/cellwall, GroupID:0, Master:{nil}" Self="Address:nonhost, Id:sm/mover, GroupID:0, Master:{nil}" Parent="Address:nonhost, Id:sm, GroupID:0, Master:{nil}"
[ACTOR] INFO Receive统计 MsgType="*msg.Tick" MsgSize=32 Sender="Address:nonhost, Id:sm, GroupID:0, Master:{nil}" Self="Address:nonhost, Id:sm/cellwall, GroupID:0, Master:{nil}" Parent="Address:nonhost, Id:sm, GroupID:0, Master:{nil}"
[ACTOR] INFO Receive统计 MsgType="*msg.Tick" MsgSize=32 Sender="Address:nonhost, Id:sm, GroupID:0, Master:{nil}" Self="Address:nonhost, Id:sm/cellwall, GroupID:0, Master:{nil}" Parent="Address:nonhost, Id:sm, GroupID:0, Master:{nil}"
[ACTOR] INFO Receive统计 MsgType="*msg.Move" MsgSize=64 Sender="Address:nonhost, Id:sm/mover, GroupID:0, Master:{nil}" Self="Address:nonhost, Id:sm/cellwall, GroupID:0, Master:{nil}" Parent="Address:nonhost, Id:sm, GroupID:0, Master:{nil}"
"""


def _result():
    return parse_log(load_schema("schema.yaml"), io.StringIO(SAMPLE))


def test_topology_type_granularity():
    topo = build_topology(_result(), by="type")
    # 三个 Actor 都是 "*" 类型；parent 有 nil 和 "*" 两种
    assert "*" in topo.nodes
    assert ROOT_NODE in topo.nodes
    # sm 实例 parent=nil → root → *
    assert topo.has_edge(ROOT_NODE, "*")
    # 两个子实例 parent=sm(类型 *) → * → *
    assert topo.has_edge("*", "*")
    # 实例数 = 3
    assert topo.nodes["*"]["instances"] == 3
    assert topo.nodes["*"]["size_max"] == 24
    assert topo.nodes["*"]["size_sum"] == 48


def test_topology_pid_granularity():
    topo = build_topology(_result(), by="pid")
    assert "nonhost/sm" in topo.nodes
    assert "nonhost/sm/cellwall" in topo.nodes
    assert "nonhost/sm/mover" in topo.nodes
    assert topo.has_edge(ROOT_NODE, "nonhost/sm")
    assert topo.has_edge("nonhost/sm", "nonhost/sm/cellwall")
    # 每个实例 instances=1
    assert topo.nodes["nonhost/sm"]["instances"] == 1


def test_communication_type_granularity():
    comm = build_communication(_result(), by="type")
    # 所有 actor 都是 "*"，所以通信图里 "*" 自环 + system 节点
    assert "*" in comm.nodes
    assert SYSTEM_NODE in comm.nodes
    # system → *（两条 Started 消息）
    assert comm.has_edge(SYSTEM_NODE, "*")
    e = comm[SYSTEM_NODE]["*"]
    assert e[M.COUNT] == 2
    assert e[M.TOTAL_BYTES] == 16


def test_communication_pid_granularity():
    comm = build_communication(_result(), by="pid")
    assert comm.has_edge("nonhost/sm/cellwall", "nonhost/sm/mover")
    e = comm["nonhost/sm/cellwall"]["nonhost/sm/mover"]
    assert e[M.COUNT] == 2
    assert e[M.TOTAL_BYTES] == 128
    # system 节点存在（Started 消息 sender=nil）
    assert SYSTEM_NODE in comm.nodes


def test_exclude_nil_sender():
    comm = build_communication(_result(), by="type", exclude_nil_sender=True)
    assert SYSTEM_NODE not in comm.nodes


def test_top_links_ordering():
    comm = build_communication(_result(), by="pid")
    links = top_links(comm, "count", top=10)
    # 最高频：cellwall → mover 2 次；sm → cellwall 2 次
    assert links[0][2] == 2
    assert links[0][0] in ("nonhost/sm/cellwall", "nonhost/sm")


def test_unknown_node_when_type_unresolved():
    # 消息接收者/发送者未在 Actor 记录中出现 → (unknown)
    text = (
        '[ACTOR] INFO Receive统计 MsgType="*msg.X" MsgSize=10 Sender="Address:nonhost, Id:ghost, GroupID:0, Master:{nil}" '
        'Self="Address:nonhost, Id:sm, GroupID:0, Master:{nil}" Parent="nil"\n'
    )
    result = parse_log(load_schema("schema.yaml"), io.StringIO(text))
    comm = build_communication(result, by="type")
    assert UNKNOWN_NODE in comm.nodes
    assert comm.has_edge(UNKNOWN_NODE, UNKNOWN_NODE) or any(
        UNKNOWN_NODE in (u, v) for u, v in comm.edges
    )
