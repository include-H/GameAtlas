#!/usr/bin/env python3
"""从孤儿隔离区恢复被误隔离的预告片封面文件。

背景：孤儿素材扫描未把 game_assets.poster_path 视为引用路径，
导致预告片封面文件被移动到 data/orphaned-assets/。本脚本以 DB
的 poster_path 为准，把缺失的封面文件从隔离区移回原位置。

用法（在服务端 data 目录所在处运行）：
    python3 scripts/restore-video-posters.py            # 恢复
    python3 scripts/restore-video-posters.py --dry-run  # 只打印计划
    python3 scripts/restore-video-posters.py --data-dir /path/to/data
"""
import argparse
import os
import shutil
import sqlite3


def find_in_quarantine(orphan_root: str, basename: str) -> str | None:
    for dirpath, _, files in os.walk(orphan_root):
        if basename in files:
            return os.path.join(dirpath, basename)
    return None


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--data-dir", default="data", help="data 目录（含 db.db / gamelist / orphaned-assets）")
    parser.add_argument("--dry-run", action="store_true", help="只打印计划，不移动文件")
    args = parser.parse_args()

    db_path = os.path.join(args.data_dir, "db.db")
    gamelist = os.path.join(args.data_dir, "gamelist")
    orphan_root = os.path.join(args.data_dir, "orphaned-assets")

    conn = sqlite3.connect(db_path)
    rows = conn.execute(
        "SELECT DISTINCT poster_path FROM game_assets "
        "WHERE COALESCE(TRIM(poster_path), '') != ''"
    ).fetchall()
    conn.close()

    restored: list[str] = []
    missing: list[str] = []
    for (poster_path,) in rows:
        rel = poster_path.removeprefix("/assets/")
        target = os.path.join(gamelist, rel)
        if os.path.isfile(target):
            continue
        source = find_in_quarantine(orphan_root, os.path.basename(rel))
        if not source:
            missing.append(poster_path)
            continue
        if args.dry_run:
            print(f"[dry-run] {source} -> {target}")
        else:
            os.makedirs(os.path.dirname(target), exist_ok=True)
            shutil.move(source, target)
            print(f"restored {target}")
        restored.append(poster_path)

    print(f"\nrestored: {len(restored)}, still missing: {len(missing)}")
    for p in missing:
        print(f"  MISSING {p}")


if __name__ == "__main__":
    main()
