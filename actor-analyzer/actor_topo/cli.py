"""命令行入口。

用法：
  python -m actor_topo parse   --log app.log --schema schema.yaml --out records.json
  python -m actor_topo analyze --log app.log --schema schema.yaml --out out/ \
          --by type --metric count --top 20 [--exclude-nil-sender] [--neo4j-uri ...]
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from . import metrics as M
from .build import build_communication, build_topology
from .parser import parse_log
from .report import text_summary, write_reports
from .schema import load_schema


def _arg_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        prog="actor_topo",
        description="解析 protoactor-go 插桩日志，构建 Actor 拓扑图与通信热点图",
    )
    sub = p.add_subparsers(dest="command", required=True)

    def add_common(sp):
        sp.add_argument("--log", required=True, help="插桩日志文件路径")
        sp.add_argument("--schema", required=True, help="Schema 文件（YAML/JSON）")

    parse_sp = sub.add_parser("parse", help="仅解析日志，导出规范化 JSON")
    add_common(parse_sp)
    parse_sp.add_argument("--out", default="records.json", help="输出 JSON 路径")

    ana_sp = sub.add_parser("analyze", help="解析并生成图 + 报告")
    add_common(ana_sp)
    ana_sp.add_argument("--out", default="out", help="输出目录")
    ana_sp.add_argument("--by", choices=["type", "pid"], default="type",
                        help="通信图聚合粒度：type（类型）/ pid（实例）")
    ana_sp.add_argument("--metric", default="count",
                        help="热点度量：" + ", ".join(M.available_metrics()))
    ana_sp.add_argument("--top", type=int, default=20, help="Top-N 数量")
    ana_sp.add_argument("--exclude-nil-sender", action="store_true",
                        help="丢弃 sender 为 nil 的消息（如系统消息）")
    ana_sp.add_argument("--no-html", action="store_true", help="不生成 HTML")
    ana_sp.add_argument("--neo4j-uri", default=None)
    ana_sp.add_argument("--neo4j-user", default=None)
    ana_sp.add_argument("--neo4j-password", default=None)
    return p


def cmd_parse(args) -> int:
    schema = load_schema(args.schema)
    with open(args.log, encoding="utf-8") as f:
        result = parse_log(schema, f)
    payload = {
        "skipped_lines": result.skipped_lines,
        "total_lines": result.total_lines,
        "actors": [a.to_dict() for a in result.actors],
        "messages": [m.to_dict() for m in result.messages],
    }
    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"解析完成: {len(result.actors)} Actor, {len(result.messages)} 消息, "
          f"跳过 {result.skipped_lines} 行 → {out}")
    return 0


def cmd_analyze(args) -> int:
    schema = load_schema(args.schema)
    with open(args.log, encoding="utf-8") as f:
        result = parse_log(schema, f)

    topo = build_topology(result, by=args.by)
    comm = build_communication(
        result, by=args.by, exclude_nil_sender=args.exclude_nil_sender
    )

    out_dir = Path(args.out)
    write_reports(topo, comm, out_dir, args.metric, args.top)

    if not args.no_html:
        from .export.html import export_html
        export_html(
            topo, out_dir / "topology.html", args.metric,
            title="Actor 父子拓扑图", node_size_field="instances",
        )
        export_html(
            comm, out_dir / "communication.html", args.metric,
            title="Actor 通信热点图",
        )

    neo4j_cfg = (args.neo4j_uri, args.neo4j_user, args.neo4j_password)
    if all(neo4j_cfg):
        from .export.neo4j import Neo4jExporter
        exporter = Neo4jExporter(*neo4j_cfg)
        try:
            exporter.import_graphs(topo, comm)
            print("Neo4j 导入完成")
        finally:
            exporter.close()
    elif any(neo4j_cfg):
        print("警告: Neo4j 参数不完整（需同时提供 uri/user/password），已跳过导入",
              file=sys.stderr)

    print(text_summary(topo, comm, args.metric, args.top,
                       skipped=result.skipped_lines, total=result.total_lines))
    return 0


def main(argv=None) -> int:
    args = _arg_parser().parse_args(argv)
    if args.command == "parse":
        return cmd_parse(args)
    if args.command == "analyze":
        return cmd_analyze(args)
    return 1


if __name__ == "__main__":
    sys.exit(main())
