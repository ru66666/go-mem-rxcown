"""可选 Neo4j 后端：导入拓扑图 + 通信图，并提供若干 Cypher 查询。

仅在提供连接参数时启用；`neo4j` 驱动 lazy import，未安装时给出提示而不报错。
"""
from __future__ import annotations

from typing import Dict, List, Optional

import networkx as nx

from .. import metrics as M


class Neo4jExporter:
    def __init__(self, uri: str, user: str, password: str):
        self.uri = uri
        self.user = user
        self.password = password
        self.driver = None

    def _connect(self):
        if self.driver is not None:
            return self.driver
        try:
            from neo4j import GraphDatabase
        except ImportError as e:  # pragma: no cover
            raise RuntimeError(
                "使用 Neo4j 后端需要安装驱动：`pip install neo4j`"
            ) from e
        self.driver = GraphDatabase.driver(
            self.uri, auth=(self.user, self.password)
        )
        return self.driver

    def import_graphs(self, topo: nx.DiGraph, comm: nx.DiGraph) -> None:
        driver = self._connect()
        with driver.session() as session:
            session.execute_write(_merge_topo, topo)
            session.execute_write(_merge_comm, comm)

    def close(self):
        if self.driver is not None:
            self.driver.close()
            self.driver = None


def _merge_topo(tx, topo: nx.DiGraph):
    for n, d in topo.nodes(data=True):
        tx.run(
            """
            MERGE (a:Actor {id: $id})
            SET a.instances = coalesce(a.instances, 0) + 1,
                a.size_sum = $size_sum,
                a.size_avg = $size_avg,
                a.size_max = $size_max
            """,
            id=n,
            size_sum=d.get("size_sum", 0),
            size_avg=d.get("size_avg", 0),
            size_max=d.get("size_max", 0),
        )
    for u, v, d in topo.edges(data=True):
        tx.run(
            """
            MATCH (p:Actor {id: $parent}), (c:Actor {id: $child})
            MERGE (p)-[r:PARENT_OF]->(c)
            SET r.count = coalesce(r.count, 0) + $count
            """,
            parent=u, child=v, count=d.get("count", 0),
        )


def _merge_comm(tx, comm: nx.DiGraph):
    for u, v, d in comm.edges(data=True):
        tx.run(
            """
            MERGE (s:Actor {id: $sender})
            MERGE (r:Actor {id: $receiver})
            MERGE (s)-[e:SENDS_TO]->(r)
            SET e.count = $count,
                e.total_bytes = $total_bytes,
                e.avg_bytes = $avg_bytes,
                e.msg_types = $msg_types
            """,
            sender=u,
            receiver=v,
            count=d.get(M.COUNT, 0),
            total_bytes=d.get(M.TOTAL_BYTES, 0),
            avg_bytes=M.edge_weight(d, "avg_bytes"),
            msg_types=dict(d.get(M.MSG_TYPES, {})),
        )


# 内置 Cypher 查询（供用户直接粘贴到 Neo4j Browser 使用）
QUERIES: Dict[str, str] = {
    "top_links": """
// Top 通信链路（按累计字节）
MATCH (s)-[e:SENDS_TO]->(r)
RETURN s.id, r.id, e.count, e.total_bytes, e.avg_bytes
ORDER BY e.total_bytes DESC LIMIT 20
""",
    "hottest_actors": """
// 最热 Actor（按出入流量）
MATCH (a:Actor)-[e:SENDS_TO]-()
RETURN a.id, count(e) AS flow, sum(e.total_bytes) AS bytes
ORDER BY flow DESC LIMIT 20
""",
    "biggest_actors": """
// 最大 Actor（按 size_max）
MATCH (a:Actor)
RETURN a.id, a.size_max, a.size_avg, a.instances
ORDER BY a.size_max DESC LIMIT 20
""",
    "parent_chain": """
// 某 Actor 的父链（替换 {id}）
MATCH path = (root:Actor)-[:PARENT_OF*]->(a:Actor {id: 'sm/cellwall'})
RETURN path
""",
}
