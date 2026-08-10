#!/usr/bin/env python3
"""从生产库导入游戏数据到本地开发库（用于本地性能/数量级测试）。

用法（先启动本地后端，再运行）：
    cd backend && go run ./cmd/server &            # 本地 :3000
    python3 scripts/import-prod-games.py --count 60
环境变量：
    GA_URL            生产库地址（必填）
    GA_PASSWORD       生产库管理员密码（必填）
    LOCAL_URL         本地库地址（默认 http://127.0.0.1:3000）
    LOCAL_PASSWORD    本地库管理员密码（必填）
"""
import argparse
import json
import os
import sys
import urllib.parse
import urllib.request

PROD_URL = os.environ.get("GA_URL", "")
PROD_PASSWORD = os.environ.get("GA_PASSWORD", "")
LOCAL_URL = os.environ.get("LOCAL_URL", "http://127.0.0.1:3000")
LOCAL_PASSWORD = os.environ.get("LOCAL_PASSWORD", "")


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


def post_json(opener, url: str, payload: dict, method="POST"):
    req = urllib.request.Request(url, data=json.dumps(payload).encode(), method=method)
    req.add_header("Content-Type", "application/json")
    try:
        with opener.open(req, timeout=60) as resp:
            return resp.status, json.load(resp)
    except urllib.error.HTTPError as exc:
        body = exc.read().decode(errors="replace")[:300]
        return exc.code, {"error": body}


def fetch_all_games(opener, base_url):
    games = []
    page = 1
    while True:
        data = get_json(opener, f"{base_url.rstrip('/')}/api/games?page={page}&limit=50")
        batch = data.get("data") or []
        games.extend(batch)
        pagination = data.get("pagination") or {}
        if page >= pagination.get("totalPages", 1) or not batch:
            break
        page += 1
    return games


def build_aggregate_payload(detail: dict) -> dict:
    game = detail
    logos = game.get("logos") or []
    # 本地库没有生产库的系列/开发商/发行商 ID，置空避免 FK 违反（本地仅做数量级测试）
    return {
        "game": {
            "title": game.get("title") or "",
            "title_alt": game.get("title_alt"),
            "visibility": game.get("visibility") or "public",
            "release_date": game.get("release_date"),
            "series_id": None,
            "developer_ids": [],
            "publisher_ids": [],
            "summary": game.get("summary"),
            "logo_visible": game.get("logo_visible", True),
        },
        "assets": {
            "files": [],
            "screenshot_order_asset_uids": [s.get("asset_uid") for s in game.get("screenshots") or [] if s.get("asset_uid")],
            "video_order_asset_uids": [v.get("asset_uid") for v in game.get("preview_videos") or [] if v.get("asset_uid")],
            "cover_order_asset_uids": [c.get("asset_uid") for c in game.get("covers") or [] if c.get("asset_uid")],
            "logo_order_asset_uids": [l.get("asset_uid") for l in logos if l.get("asset_uid")],
            "banner_order_asset_uids": [b.get("asset_uid") for b in game.get("banners") or [] if b.get("asset_uid")],
            "logo_positions": [
                {
                    "asset_uid": l["asset_uid"],
                    "position_x": l.get("position_x"),
                    "position_y": l.get("position_y"),
                    "width_pct": l.get("width_pct"),
                }
                for l in logos if l.get("asset_uid")
            ],
            "new_assets": [],
        },
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--count", type=int, default=60, help="导入数量（默认 60）")
    parser.add_argument("--start", type=int, default=0, help="跳过前 N 款")
    parser.add_argument("--prod-url", default=PROD_URL, help="生产库地址")
    parser.add_argument("--prod-password", default=PROD_PASSWORD, help="生产库管理员密码")
    parser.add_argument("--local-url", default=LOCAL_URL, help="本地库地址")
    parser.add_argument("--local-password", default=LOCAL_PASSWORD, help="本地库管理员密码")
    args = parser.parse_args()

    if not args.prod_url:
        parser.error("缺少生产库地址：--prod-url 或环境变量 GA_URL")
    if not args.prod_password:
        parser.error("缺少生产库密码：--prod-password 或环境变量 GA_PASSWORD")
    if not args.local_password:
        parser.error("缺少本地库密码：--local-password 或环境变量 LOCAL_PASSWORD")

    prod = login(args.prod_url, args.prod_password)
    local = login(args.local_url, args.local_password)

    games = fetch_all_games(prod, args.prod_url)
    print(f"生产库共 {len(games)} 款，本次导入 {min(args.count, len(games) - args.start)} 款")

    # 幂等保护：导入前拉取本地全部标题，已存在则跳过创建（脚本可重复执行）
    local_games = fetch_all_games(local, args.local_url)
    local_titles = {g.get("title", "").strip().lower() for g in local_games if g.get("title")}
    print(f"本地库共 {len(local_games)} 款，其中 {len(local_titles)} 个不重复标题")

    created = failed = skipped = 0
    for item in games[args.start:args.start + args.count]:
        pid = item.get("public_id")
        title = item.get("title", "")
        if title.strip().lower() in local_titles:
            print(f"[跳过] {title}：本地已存在同名游戏")
            skipped += 1
            continue
        detail = get_json(prod, f"{args.prod_url.rstrip('/')}/api/games/{pid}").get("data") or {}

        status, resp = post_json(local, f"{args.local_url.rstrip('/')}/api/games", {"title": title, "visibility": item.get("visibility") or "public"})
        if status not in (200, 201):
            print(f"[创建失败 {status}] {title}: {resp}")
            failed += 1
            continue
        new_pid = (resp.get("data") or {}).get("public_id") or pid

        payload = build_aggregate_payload(detail)
        status, resp = post_json(local, f"{args.local_url.rstrip('/')}/api/games/{new_pid}/aggregate", payload, method="PUT")
        if status not in (200, 201):
            print(f"[写入失败 {status}] {title}: {resp}")
            failed += 1
            continue
        created += 1
        if created % 10 == 0:
            print(f"进度 {created}/{args.count}")

    print(f"完成：成功 {created}，失败 {failed}，跳过重复 {skipped}")
    if failed:
        print("提示：系列/开发商/发行商引用不存在的本地 ID 会跳过关系但保留游戏，属预期。")


if __name__ == "__main__":
    main()
