#!/usr/bin/env python3
"""清理本地库的重复标题游戏（导入脚本幂等保护补上之前产生的重复）。

策略：按标题分组，每组保留"内容最完整"的一份（素材数 + 文件数 + 简介 +
Wiki + 开发商 + 发行商加权分最高，并列取 id 最小），其余走 DELETE 接口删除。

用法（先启动本地后端）：
    cd backend && go run ./cmd/server &
    python3 scripts/dedupe-games.py                 # dry-run，只打印计划
    python3 scripts/dedupe-games.py --apply         # 真正执行删除
环境变量：
    LOCAL_URL         目标库地址（默认 http://127.0.0.1:3000）
    LOCAL_PASSWORD    目标库管理员密码（默认 1234）
"""
import argparse
import json
import os
import sys
import urllib.request

LOCAL_URL = os.environ.get("LOCAL_URL", "http://127.0.0.1:3000")
LOCAL_PASSWORD = os.environ.get("LOCAL_PASSWORD", "1234")


def login(base_url: str, password: str):
    opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor())
    req = urllib.request.Request(
        base_url.rstrip("/") + "/api/auth/login",
        data=json.dumps({"password": password}).encode(),
        method="POST",
    )
    req.add_header("Content-Type", "application/json")
    try:
        with opener.open(req, timeout=15) as resp:
            if resp.status != 200:
                sys.exit(f"登录 {base_url} 失败: HTTP {resp.status}")
    except urllib.error.HTTPError as exc:
        sys.exit(f"登录 {base_url} 失败: HTTP {exc.code}")
    return opener


def get_json(opener, url: str):
    req = urllib.request.Request(url)
    with opener.open(req, timeout=30) as resp:
        return json.load(resp)


def fetch_all_games(opener, base_url: str):
    games = []
    page = 1
    while True:
        data = get_json(opener, f"{base_url}/api/games?page={page}&page_size=100&include_all=true")
        batch = data.get("data") or []
        games.extend(batch)
        pagination = data.get("pagination") or {}
        if page >= pagination.get("totalPages", 1) or not batch:
            break
        page += 1
    return games


def delete_game(opener, url: str) -> int:
    req = urllib.request.Request(url, method="DELETE")
    try:
        with opener.open(req, timeout=60) as resp:
            return resp.status
    except urllib.error.HTTPError as exc:
        return exc.code


def completeness_score(item: dict) -> tuple:
    cover = 1 if (item.get("cover_image") or "").strip() else 0
    banner = 1 if (item.get("banner_image") or "").strip() else 0
    assets = cover + banner + int(item.get("screenshot_count") or 0) + int(item.get("logo_count") or 0) + int(item.get("video_count") or 0)
    files = int(item.get("file_count") or 0)
    summary = 1 if (item.get("summary") or "").strip() else 0
    wiki = 1 if (item.get("wiki_content") or "").strip() else 0
    return (assets, files, summary, wiki)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--apply", action="store_true", help="真正执行删除（默认只打印计划）")
    args = parser.parse_args()

    base = LOCAL_URL.rstrip("/")
    opener = login(base, LOCAL_PASSWORD)

    games = fetch_all_games(opener, base)
    items = games
    print(f"库中共 {len(items)} 款游戏")

    groups: dict[str, list] = {}
    for item in items:
        groups.setdefault(item.get("title", ""), []).append(item)

    to_delete: list = []
    for title, group in groups.items():
        if len(group) <= 1:
            continue
        group.sort(key=lambda g: (completeness_score(g), -g.get("id", 0)), reverse=True)
        keep, rest = group[0], group[1:]
        print(f"[重复] {title!r} × {len(group)}：保留 id={keep.get('id')} (public_id={keep.get('public_id')} 完整度={completeness_score(keep)})，删除 {len(rest)} 份")
        for item in rest:
            to_delete.append(item)

    print(f"\n共 {len(to_delete)} 个重复项待删除")
    if not to_delete:
        return
    if not args.apply:
        print("dry-run：未执行删除。确认无误后加 --apply 重新运行。")
        return

    deleted = failed = 0
    for item in to_delete:
        status = delete_game(opener, f"{base}/api/games/{item.get('public_id')}")
        if status == 200:
            deleted += 1
        else:
            failed += 1
            print(f"[删除失败 {status}] {item.get('title')} (id={item.get('id')})")
    print(f"完成：删除 {deleted}，失败 {failed}")


if __name__ == "__main__":
    main()
