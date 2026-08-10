#!/usr/bin/env python3
"""预热 WebP 变体：登录管理员后对全部游戏素材触发 ?w= 懒生成。

用法：
    python3 scripts/warmup-webp-variants.py --url http://127.0.0.1:3000 --password xxxx
    python3 scripts/warmup-webp-variants.py --url https://x --password xxxx
    python3 scripts/warmup-webp-variants.py --dry-run           # 只统计不请求
环境变量 GA_URL / GA_PASSWORD 可替代 --url / --password。
"""
import argparse
import os
import sys
import urllib.parse
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed

WIDTHS = (480, 1280)
WORKERS = 6
TIMEOUT = 60


def build_opener(base_url: str, password: str):
    cookie_jar = urllib.request.HTTPCookieProcessor()
    opener = urllib.request.build_opener(cookie_jar)
    body = __import__("json").dumps({"password": password}).encode()
    req = urllib.request.Request(base_url.rstrip("/") + "/api/auth/login", data=body, method="POST")
    req.add_header("Content-Type", "application/json")
    try:
        with opener.open(req, timeout=15) as resp:
            if resp.status != 200:
                sys.exit(f"登录失败: HTTP {resp.status}")
    except urllib.error.HTTPError as exc:
        sys.exit(f"登录失败: HTTP {exc.code} {exc.reason}")
    return opener


def fetch_json(opener, url: str):
    req = urllib.request.Request(url)
    with opener.open(req, timeout=30) as resp:
        return __import__("json").load(resp)


def collect_assets(games):
    assets = []
    for game in games:
        pid = game.get("public_id")
        if not pid:
            continue
        paths = {
            "cover": game.get("cover_image"),
            "banner": game.get("banner_image"),
            "screenshot": [s["path"] for s in game.get("screenshots", [])],
            "logo": [l["path"] for l in game.get("logos", [])],
            "poster": [v.get("poster_path") for v in game.get("preview_videos", []) if v.get("poster_path")],
        }
        for kind, value in paths.items():
            if kind in ("screenshot", "logo", "poster"):
                for p in value:
                    if p:
                        assets.append((pid, kind, p))
            elif value:
                assets.append((pid, kind, value))
    return assets


def warm(opener, base_url: str, asset, width: int):
    path = asset[2]
    url = f"{base_url.rstrip('/')}{path}?w={width}"
    try:
        req = urllib.request.Request(url)
        with opener.open(req, timeout=TIMEOUT) as resp:
            return resp.status == 200, resp.headers.get("Content-Type", ""), resp.status
    except Exception:
        return False, "", 0


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--url", default=os.environ.get("GA_URL", ""))
    parser.add_argument("--password", default=os.environ.get("GA_PASSWORD", ""))
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    if not args.url:
        sys.exit("缺少目标地址：--url 或环境变量 GA_URL")
    if not args.password:
        sys.exit("缺少管理员密码：--password 或环境变量 GA_PASSWORD")

    opener = build_opener(args.url, args.password)
    base = args.url.rstrip("/")

    games = []
    page = 1
    while True:
        data = fetch_json(opener, f"{base}/api/games?page={page}&limit=50")
        batch = data.get("data") or []
        games.extend(batch)
        pagination = data.get("pagination") or {}
        if page >= pagination.get("totalPages", 1) or not batch:
            break
        page += 1

    assets = collect_assets(games)
    total = len(assets) * len(WIDTHS)
    print(f"游戏 {len(games)} 个，素材 {len(assets)} 条，待触发请求 {total} 个")
    if args.dry_run:
        sys.exit(0)

    done = ok = failed = 0
    jobs = [(asset, w) for asset in assets for w in WIDTHS]
    with ThreadPoolExecutor(max_workers=WORKERS) as pool:
        futures = {pool.submit(warm, opener, args.url, asset, w): (asset, w) for asset, w in jobs}
        for future in as_completed(futures):
            success, content_type, status = future.result()
            done += 1
            if success:
                ok += 1
            else:
                failed += 1
                print(f"失败 {status}: {futures[future][0][2]}?w={futures[future][1]}")
            if done % 200 == 0:
                print(f"进度 {done}/{total} (成功 {ok}，失败 {failed})")

    print(f"完成：成功 {ok}，失败 {failed}，共 {done} 请求")


if __name__ == "__main__":
    main()
