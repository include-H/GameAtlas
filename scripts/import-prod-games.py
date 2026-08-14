#!/usr/bin/env python3
"""从生产库迁移随机游戏到本地开发库（含素材与 Wiki）。

用法（先启动本地后端，再运行）：
    cd backend && go run ./cmd/server &            # 本地 :3000
    python3 scripts/import-prod-games.py --count 40 --random
环境变量：
    GA_URL            生产库地址（必填）
    GA_PASSWORD       生产库管理员密码（必填）
    LOCAL_URL         本地库地址（默认 http://127.0.0.1:3000）
    LOCAL_PASSWORD    本地库管理员密码（必填）
"""
import argparse
import json
import mimetypes
import os
import random
import sys
import urllib.parse
import urllib.request
import uuid

PROD_URL = os.environ.get("GA_URL", "")
PROD_PASSWORD = os.environ.get("GA_PASSWORD", "")
LOCAL_URL = os.environ.get("LOCAL_URL", "http://127.0.0.1:3000")
LOCAL_PASSWORD = os.environ.get("LOCAL_PASSWORD", "")

# 素材类型 -> 本地上传端点后缀；与生产详情返回的素材分组字段一一对应
ASSET_ENDPOINTS = {
    "screenshots": "screenshot",
    "covers": "cover",
    "banners": "banner",
    "logos": "logo",
    "preview_videos": "video",
}

ASSET_TYPES = {  # 聚合写入用的 asset_type（与上传端点同义）
    "screenshots": "screenshot",
    "covers": "cover",
    "banners": "banner",
    "logos": "logo",
    "preview_videos": "video",
}


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


def download_bytes(opener, url: str):
    req = urllib.request.Request(url)
    with opener.open(req, timeout=60) as resp:
        return resp.read(), resp.headers.get("Content-Type", "")


def upload_asset(opener, base_url: str, game_id: int, asset_type: str, filename: str, data: bytes, content_type: str):
    """上传素材到本地，返回 {path, asset_uid}。"""
    if not content_type or content_type == "application/octet-stream":
        content_type = mimetypes.guess_type(filename)[0] or "application/octet-stream"
    boundary = uuid.uuid4().hex
    parts = [
        f"--{boundary}\r\n"
        f'Content-Disposition: form-data; name="game_id"\r\n\r\n{game_id}\r\n',
        f"--{boundary}\r\n"
        f'Content-Disposition: form-data; name="file"; filename="{filename}"\r\n'
        f"Content-Type: {content_type}\r\n\r\n",
    ]
    body = "".join(parts).encode() + data + f"\r\n--{boundary}--\r\n".encode()
    req = urllib.request.Request(
        f"{base_url.rstrip('/')}/api/assets/{asset_type}",
        data=body,
        method="POST",
    )
    req.add_header("Content-Type", f"multipart/form-data; boundary={boundary}")
    try:
        with opener.open(req, timeout=120) as resp:
            result = json.load(resp)
            return (resp.status, result.get("data") or {})
    except urllib.error.HTTPError as exc:
        return exc.code, {"error": exc.read().decode(errors="replace")[:300]}


def copy_game_assets(prod, local, base_url, prod_url, game_id, detail):
    """把生产的素材下载并上传到本地，返回聚合写入所需的素材结构。

    返回 (new_assets, order_lists, logo_positions, prod_uid_to_local_uid)
    """
    new_assets = []
    order_lists = {
        "screenshot_order_asset_uids": [],
        "video_order_asset_uids": [],
        "cover_order_asset_uids": [],
        "logo_order_asset_uids": [],
        "banner_order_asset_uids": [],
    }
    logo_positions = []
    uid_map = {}

    for field, endpoint in ASSET_ENDPOINTS.items():
        items = detail.get(field) or []
        asset_type = ASSET_TYPES[field]
        for item in items:
            path = item.get("path") or ""
            if not path:
                continue
            prod_uid = item.get("asset_uid") or ""
            try:
                data, content_type = download_bytes(prod, prod_url.rstrip("/") + path)
            except Exception as exc:
                print(f"    [素材下载失败] {path}: {exc}")
                continue
            filename = path.rsplit("/", 1)[-1] or f"{prod_uid}.bin"
            status, resp = upload_asset(local, base_url, game_id, endpoint, filename, data, content_type)
            if status not in (200, 201):
                print(f"    [素材上传失败 {status}] {filename}: {resp.get('error')}")
                continue
            local_uid = resp.get("asset_uid") or ""
            local_path = resp.get("path") or ""
            if prod_uid:
                uid_map[prod_uid] = local_uid
            entry = {"asset_uid": local_uid, "asset_type": asset_type, "path": local_path}

            poster_path = item.get("poster_path")
            if endpoint == "video" and poster_path:
                try:
                    poster_data, poster_type = download_bytes(prod, prod_url.rstrip("/") + poster_path)
                    poster_name = poster_path.rsplit("/", 1)[-1] or "poster.jpg"
                    _, poster_resp = upload_asset(local, base_url, game_id, "poster", poster_name, poster_data, poster_type)
                    if poster_resp.get("path"):
                        entry["poster_path"] = poster_resp["path"]
                except Exception as exc:
                    print(f"    [海报下载失败] {poster_path}: {exc}")
            new_assets.append(entry)

    def order_key(field):
        return order_lists[field + "_order_asset_uids"]

    for field in ("screenshots", "preview_videos", "covers", "logos", "banners"):
        key = {"screenshots": "screenshot", "preview_videos": "video", "covers": "cover", "logos": "logo", "banners": "banner"}[field]
        target = order_lists[key + "_order_asset_uids"]
        for item in detail.get(field) or []:
            local_uid = uid_map.get(item.get("asset_uid") or "")
            if local_uid:
                target.append(local_uid)

    for item in detail.get("logos") or []:
        local_uid = uid_map.get(item.get("asset_uid") or "")
        if local_uid:
            logo_positions.append({
                "asset_uid": local_uid,
                "position_x": item.get("position_x"),
                "position_y": item.get("position_y"),
                "width_pct": item.get("width_pct"),
            })

    return new_assets, order_lists, logo_positions, uid_map


def ensure_relations(local, base_url: str, detail: dict):
    """按名称在本地幂等创建系列/开发商/发行商，返回可用的 ID 列表。

    全部 best-effort：失败只警告，不中断导入。
    """
    result = {"series_id": None, "developer_ids": [], "publisher_ids": []}
    try:
        local_series = {s.get("slug"): s.get("id") for s in ((get_json(local, f"{base_url.rstrip('/')}/api/series") or {}).get("data") or [])}
        series = detail.get("series")
        if series and series.get("slug"):
            sid = local_series.get(series["slug"])
            if sid is None:
                _, resp = post_json(local, f"{base_url.rstrip('/')}/api/series", {"name": series.get("name"), "slug": series.get("slug")})
                sid = (resp.get("data") or {}).get("id")
            if sid:
                result["series_id"] = sid
    except Exception as exc:
        print(f"    [系列映射失败] {exc}")

    for resource, field, target in (("developers", "developers", "developer_ids"), ("publishers", "publishers", "publisher_ids")):
        try:
            local_items = {s.get("slug"): s.get("id") for s in ((get_json(local, f"{base_url.rstrip('/')}/api/{resource}") or {}).get("data") or [])}
            for item in detail.get(field) or []:
                slug = item.get("slug")
                if not slug:
                    continue
                mid = local_items.get(slug)
                if mid is None:
                    _, resp = post_json(local, f"{base_url.rstrip('/')}/api/{resource}", {"name": item.get("name"), "slug": slug})
                    mid = (resp.get("data") or {}).get("id")
                if mid:
                    result[target].append(mid)
        except Exception as exc:
            print(f"    [{resource} 映射失败] {exc}")
    return result


def build_aggregate_payload(detail: dict, new_assets, order_lists, logo_positions, relations: dict) -> dict:
    game = detail
    return {
        "game": {
            "title": game.get("title") or "",
            "title_alt": game.get("title_alt"),
            "visibility": game.get("visibility") or "public",
            "release_date": game.get("release_date"),
            "series_id": relations["series_id"],
            "developer_ids": relations["developer_ids"],
            "publisher_ids": relations["publisher_ids"],
            "summary": game.get("summary"),
            "logo_visible": game.get("logo_visible", True),
        },
        "assets": {
            "files": [],
            "screenshot_order_asset_uids": order_lists["screenshot_order_asset_uids"],
            "video_order_asset_uids": order_lists["video_order_asset_uids"],
            "cover_order_asset_uids": order_lists["cover_order_asset_uids"],
            "logo_order_asset_uids": order_lists["logo_order_asset_uids"],
            "banner_order_asset_uids": order_lists["banner_order_asset_uids"],
            "logo_positions": logo_positions,
            "new_assets": new_assets,
        },
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--count", type=int, default=40, help="导入数量（默认 40）")
    parser.add_argument("--random", action="store_true", help="随机抽取（默认按顺序从 --start 开始）")
    parser.add_argument("--seed", type=int, default=42, help="随机种子，保证可复现（默认 42）")
    parser.add_argument("--start", type=int, default=0, help="顺序模式下跳过前 N 款")
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
    if args.random:
        selected = random.Random(args.seed).sample(games, min(args.count, len(games)))
        mode = f"随机抽取 {len(selected)} 款（seed={args.seed}）"
    else:
        selected = games[args.start:args.start + args.count]
        mode = f"顺序导入 {len(selected)} 款"
    print(f"生产库共 {len(games)} 款，{mode}")

    # 幂等保护：导入前拉取本地全部标题，已存在则跳过创建（脚本可重复执行）
    local_games = fetch_all_games(local, args.local_url)
    local_titles = {g.get("title", "").strip().lower() for g in local_games if g.get("title")}
    print(f"本地库共 {len(local_games)} 款，其中 {len(local_titles)} 个不重复标题")

    created = failed = skipped = 0
    for item in selected:
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
        created_data = resp.get("data") or {}
        new_pid = created_data.get("public_id") or pid
        game_id = created_data.get("id")
        if not game_id:
            print(f"[创建失败] {title}: 返回缺少游戏 id")
            failed += 1
            continue

        new_assets, order_lists, logo_positions, uid_map = copy_game_assets(prod, local, args.local_url, args.prod_url, game_id, detail)
        if len(uid_map) != len(new_assets):
            print(f"    [注意] 素材 {len(new_assets)} 个，UID 映射 {len(uid_map)} 个（无 UID 的素材只保留条目）")

        relations = ensure_relations(local, args.local_url, detail)
        payload = build_aggregate_payload(detail, new_assets, order_lists, logo_positions, relations)
        status, resp = post_json(local, f"{args.local_url.rstrip('/')}/api/games/{new_pid}/aggregate", payload, method="PUT")
        if status not in (200, 201):
            print(f"[写入失败 {status}] {title}: {resp}")
            failed += 1
            continue

        wiki_content = detail.get("wiki_content")
        if wiki_content:
            status, resp = post_json(local, f"{args.local_url.rstrip('/')}/api/games/{new_pid}/wiki", {"content": wiki_content, "change_summary": "从生产库迁移"}, method="PUT")
            if status not in (200, 201):
                print(f"[Wiki 写入失败 {status}] {title}: {resp}")

        created += 1
        print(f"[成功] {title}（素材 {len(new_assets)} 个）")
        if created % 10 == 0:
            print(f"进度 {created}/{len(selected)}")

    print(f"完成：成功 {created}，失败 {failed}，跳过重复 {skipped}")


if __name__ == "__main__":
    main()
