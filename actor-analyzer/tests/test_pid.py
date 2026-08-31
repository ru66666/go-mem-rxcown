from actor_topo.pid import PidParser


def _parser():
    return PidParser(
        {
            "null_literal": "nil",
            "fields": {
                "Address": "string",
                "Id": "string",
                "GroupID": "int",
                "Master": "pid",
            },
        }
    )


def test_null():
    p = _parser()
    assert p.parse("nil") is None
    assert p.parse("") is None
    assert p.parse(None) is None


def test_full_pid():
    p = _parser()
    pid = p.parse(
        "Address:nonhost, Id:sm/cellwall, GroupID:0, Master:{nil}"
    )
    assert pid is not None
    assert pid.address == "nonhost"
    assert pid.id == "sm/cellwall"
    assert pid.group_id == 0
    assert pid.master is None


def test_nested_master():
    p = _parser()
    pid = p.parse(
        "Address:nonhost, Id:sm, GroupID:3, Master:{Address:remote, Id:root, GroupID:1, Master:nil}"
    )
    assert pid is not None
    assert pid.group_id == 3
    assert pid.master is not None
    assert pid.master.address == "remote"
    assert pid.master.id == "root"
    assert pid.master.group_id == 1
    assert pid.master.master is None


def test_identity():
    p = _parser()
    pid = p.parse("Address:nonhost, Id:sm/cellwall, GroupID:0, Master:{nil}")
    assert pid.identity() == "nonhost/sm/cellwall"


def test_split_top_level_keeps_nested():
    # 顶层逗号切分不应切碎嵌套 Master 里的逗号
    pairs = PidParser._split_top_level(
        "Address:nonhost, Id:sm, GroupID:0, Master:{Address:r, Id:x, GroupID:1, Master:nil}"
    )
    assert len(pairs) == 4
    assert pairs[3] == ("Master", "{Address:r, Id:x, GroupID:1, Master:nil}")
