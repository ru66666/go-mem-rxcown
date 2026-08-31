from collections import Counter

import pytest

from actor_topo import metrics as M


def _attrs(count=0, total_bytes=0, msg_types=None):
    return {
        M.COUNT: count,
        M.TOTAL_BYTES: total_bytes,
        M.MSG_TYPES: msg_types or Counter(),
    }


def test_count():
    assert M.edge_weight(_attrs(count=5, total_bytes=100), "count") == 5


def test_total_bytes():
    assert M.edge_weight(_attrs(count=5, total_bytes=100), "total_bytes") == 100


def test_avg_bytes():
    assert M.edge_weight(_attrs(count=5, total_bytes=100), "avg_bytes") == 20


def test_avg_bytes_zero_count():
    assert M.edge_weight(_attrs(count=0, total_bytes=0), "avg_bytes") == 0


def test_volume_equals_total_bytes():
    assert M.edge_weight(_attrs(count=5, total_bytes=100), "volume") == 100


def test_unknown_metric_raises():
    with pytest.raises(KeyError):
        M.edge_weight(_attrs(), "nope")


def test_register_custom_metric():
    M.register_metric("my_metric", lambda a: a.get(M.COUNT, 0) * 2, "自定义")
    assert M.edge_weight(_attrs(count=3), "my_metric") == 6
    assert "my_metric" in M.available_metrics()


def test_summarize_msg_types():
    d = _attrs(msg_types=Counter({"*msg.A": 5, "*msg.B": 2, "*msg.C": 1}))
    assert M.summarize_msg_types(d) == "*msg.A×5, *msg.B×2, *msg.C×1"
