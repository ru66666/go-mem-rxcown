#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
从完整版 memPool dump 日志（mode=full）提取逐 pool 分级明细，
生成 Neo4j admin-import CSV（图数据库格式）。

只入库明细段，聚合段（[GC.*] / ===== GC aggregate ===== / === full detail ===）不入库；
mode=aggregate 的快照整块跳过。

图模型（每个快照一张子图，Span 作叶子）：
    (Snapshot)-[:HAS_POOL]->(Pool)-[:HAS_ARENA]->(Arena)-[:HAS_SPAN]->(Span)
    (Pool)-[:HAS_ARENA_OB]->(ArenaOB)

用法:
    python3 log2graph.py 1020-full.log              # 输出到 ./graph_import/
    python3 log2graph.py a.log b.log -o my_import   # 多份日志 + 指定输出目录

导入（Neo4j 5.x，需先停库或用空库）:
    neo4j-admin database import full neo4j \\
      --nodes=graph_import/snapshots.csv \\
      --nodes=graph_import/pools.csv \\
      --nodes=graph_import/arenas.csv \\
      --nodes=graph_import/spans.csv \\
      --nodes=graph_import/arenaob.csv \\
      --relationships=graph_import/rels.csv
"""
import csv
import os
import re
import sys

DUMP_RE = re.compile(
    r"memPool stats dump \(reason=([^,]+), mode=(full|aggregate)\), time: (\S+)")
POOL_RE = re.compile(r"^Pool: (.+?) \[(.+?)\]( \(idle\))?(, .*)?$")
POOL_STAT_RE = re.compile(
    r"Allocated: (\d+), Freed: (\d+), InUsed: (-?\d+), Usage: (\d+)%")
ARENA_RE = re.compile(
    r"\[Arena\] ObjectSize: (\d+), SpanNum: (\d+), SpanAlloc: (\d+), "
    r"SpanFreed: (\d+), FreedBytes: (\S+), FreedObjs: (\d+)")
SPAN_RE = re.compile(r"\[Span\] TotalSlots: (\d+), InUsed: (\d+)")
ARENAOB_RE = re.compile(
    r"\[ArenaOB\] InUsed: (-?\d+), ObjAlloc: (\d+), ObjFree: (\d+), "
    r"FreedBytes: (\d+), ObjectsCap: (\d+), IdxPoolCap: (\d+)")
FMT_BYTES_RE = re.compile(r"^([\d.]+)(B|KB|MB|GB)$")

UNIT_MULT = {"B": 1, "KB": 1024, "MB": 1024 ** 2, "GB": 1024 ** 3}


def parse_fmt_bytes(s):
    """把 fmtBytes 输出（如 1.50MB / 512B / 0B）还原成字节数。"""
    m = FMT_BYTES_RE.match(s)
    if not m:
        return 0
    return int(round(float(m.group(1)) * UNIT_MULT[m.group(2)]))


def strip_syslog_prefix(raw):
    """去掉 syslog 前缀，只留业务内容（无前缀则原样返回）。"""
    i = raw.find("]: ")
    return raw[i + 3:] if i != -1 else raw


def parse_log(path):
    """解析一份日志，返回快照列表。

    每个快照: {time, reason, pools: [{name, kind, idle, allocated, freed,
    inUsed, usage, arenas: [...], arena_ob: {...}|None}]}
    """
    snapshots = []
    snap = None      # 当前 full 快照
    pool = None      # 当前 pool

    def finish():
        nonlocal snap, pool
        if snap is not None and snap["pools"]:
            snapshots.append(snap)
        snap, pool = None, None

    with open(path, encoding="utf-8", errors="replace") as f:
        for raw in f:
            line = strip_syslog_prefix(raw).rstrip("\n")

            m = DUMP_RE.search(line)
            if m:
                finish()
                if m.group(2) == "full":
                    snap = {"time": m.group(3), "reason": m.group(1),
                            "pools": []}
                continue
            if snap is None:
                continue

            m = POOL_RE.match(line.strip())
            if m is None:  # 容错：明细行带 [prefix] 标签时去掉再匹配
                s = line.strip()
                if s.startswith("["):
                    k = s.find("] ")
                    if k != -1:
                        m = POOL_RE.match(s[k + 2:])
            if m:
                pool = {"name": m.group(1), "kind": m.group(2),
                        "idle": m.group(3) is not None,
                        "allocated": 0, "freed": 0, "inUsed": 0, "usage": 0,
                        "arenas": [], "arena_ob": None}
                snap["pools"].append(pool)
                ms = POOL_STAT_RE.search(line)
                if ms:
                    pool["allocated"] = int(ms.group(1))
                    pool["freed"] = int(ms.group(2))
                    pool["inUsed"] = int(ms.group(3))
                    pool["usage"] = int(ms.group(4))
                continue

            if pool is None:
                continue

            m = ARENA_RE.search(line)
            if m:
                pool["arenas"].append({
                    "objectSize": int(m.group(1)),
                    "spanNum": int(m.group(2)),
                    "spanAlloc": int(m.group(3)),
                    "spanFreed": int(m.group(4)),
                    "freedBytes": parse_fmt_bytes(m.group(5)),
                    "freedObjs": int(m.group(6)),
                    "span": None})
                continue

            m = SPAN_RE.search(line)
            if m and pool["arenas"]:
                last = pool["arenas"][-1]
                last["span"] = {"totalSlots": int(m.group(1)),
                                "inUsed": int(m.group(2))}
                continue

            m = ARENAOB_RE.search(line)
            if m:
                pool["arena_ob"] = {
                    "inUsed": int(m.group(1)),
                    "objAlloc": int(m.group(2)),
                    "objFree": int(m.group(3)),
                    "freedBytes": int(m.group(4)),
                    "objectsCap": int(m.group(5)),
                    "idxPoolCap": int(m.group(6))}
                continue

            if "=== full detail:" in line:
                finish()
                continue
    finish()
    return snapshots


def write_csv(path, header, rows):
    with open(path, "w", newline="", encoding="utf-8") as f:
        w = csv.writer(f, lineterminator="\n")
        w.writerow(header)
        w.writerows(rows)


def export(snapshots, out_dir):
    """把快照列表写成 Neo4j admin-import CSV，返回各类节点计数。"""
    os.makedirs(out_dir, exist_ok=True)

    snap_rows, pool_rows, arena_rows = [], [], []
    span_rows, ob_rows, rel_rows = [], [], []

    for i, snap in enumerate(snapshots, 1):
        sid = "snap-{:04d}".format(i)
        snap_rows.append([sid, snap["time"], snap["reason"], "full",
                          "Snapshot"])
        for j, p in enumerate(snap["pools"], 1):
            pid = "{}-pool-{:04d}".format(sid, j)
            pool_rows.append([pid, sid, p["name"], p["kind"],
                              p["allocated"], p["freed"], p["inUsed"],
                              p["usage"], "true" if p["idle"] else "false",
                              "Pool"])
            rel_rows.append([sid, pid, "HAS_POOL"])
            for k, a in enumerate(p["arenas"], 1):
                aid = "{}-arena-{:04d}".format(pid, k)
                arena_rows.append([aid, pid, a["objectSize"], a["spanNum"],
                                   a["spanAlloc"], a["spanFreed"],
                                   a["freedBytes"], a["freedObjs"], "Arena"])
                rel_rows.append([pid, aid, "HAS_ARENA"])
                if a["span"] is not None:
                    span_rows.append([aid + "-span", aid,
                                      a["span"]["totalSlots"],
                                      a["span"]["inUsed"], "Span"])
                    rel_rows.append([aid, aid + "-span", "HAS_SPAN"])
            if p["arena_ob"] is not None:
                ob = p["arena_ob"]
                ob_rows.append([pid + "-arenaob", pid, ob["inUsed"],
                                ob["objAlloc"], ob["objFree"],
                                ob["freedBytes"], ob["objectsCap"],
                                ob["idxPoolCap"], "ArenaOB"])
                rel_rows.append([pid, pid + "-arenaob", "HAS_ARENA_OB"])

    write_csv(os.path.join(out_dir, "snapshots.csv"),
              ["snapshotId:ID(Snapshot-ID)", "time:STRING", "reason:STRING",
               "mode:STRING", ":LABEL"], snap_rows)
    write_csv(os.path.join(out_dir, "pools.csv"),
              ["poolId:ID(Pool-ID)", "snapshotId:STRING", "name:STRING",
               "kind:STRING", "allocated:long", "freed:long",
               "inUsed:long", "usage:int", "idle:boolean", ":LABEL"],
              pool_rows)
    write_csv(os.path.join(out_dir, "arenas.csv"),
              ["arenaId:ID(Arena-ID)", "poolId:STRING", "objectSize:int",
               "spanNum:int", "spanAlloc:long", "spanFreed:long",
               "freedBytes:long", "freedObjs:long", ":LABEL"], arena_rows)
    write_csv(os.path.join(out_dir, "spans.csv"),
              ["spanId:ID(Span-ID)", "arenaId:STRING", "totalSlots:long",
               "inUsed:long", ":LABEL"], span_rows)
    write_csv(os.path.join(out_dir, "arenaob.csv"),
              ["arenaObId:ID(ArenaOB-ID)", "poolId:STRING", "inUsed:long",
               "objAlloc:long", "objFree:long", "freedBytes:long",
               "objectsCap:long", "idxPoolCap:long", ":LABEL"], ob_rows)
    write_csv(os.path.join(out_dir, "rels.csv"),
              [":START_ID", ":END_ID", ":TYPE"], rel_rows)

    return (len(snap_rows), len(pool_rows), len(arena_rows),
            len(span_rows), len(ob_rows), len(rel_rows))


def main():
    args = sys.argv[1:]
    out_dir = "graph_import"
    if "-o" in args:
        i = args.index("-o")
        if i + 1 >= len(args):
            raise SystemExit("用法: python3 log2graph.py <log>... [-o outdir]")
        out_dir = args[i + 1]
        del args[i:i + 2]
    if not args:
        raise SystemExit("用法: python3 log2graph.py <log>... [-o outdir]")

    snapshots = []
    for log_path in args:
        snaps = parse_log(log_path)
        print("{}: {} 个 full 快照".format(log_path, len(snaps)))
        snapshots.extend(snaps)
    if not snapshots:
        raise SystemExit("日志里没有找到 mode=full 的快照明细，"
                         "请用 DumpAllPoolsGCStatsWithReason（完整版）重新采样")

    counts = export(snapshots, out_dir)
    print("saved: {} -> snapshots={} pools={} arenas={} spans={} "
          "arenaOB={} rels={}".format(out_dir, *counts))
    print("导入示例:\n"
          "  neo4j-admin database import full neo4j \\\n"
          "    --nodes={}/snapshots.csv --nodes={}/pools.csv \\\n"
          "    --nodes={}/arenas.csv --nodes={}/spans.csv \\\n"
          "    --nodes={}/arenaob.csv --relationships={}/rels.csv"
          .format(out_dir, out_dir, out_dir, out_dir, out_dir, out_dir))


if __name__ == "__main__":
    main()
