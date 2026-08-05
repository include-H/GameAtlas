#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""临时一次性 helper：为已有预告片（video 资产）批量补生成封面帧 poster_path。

约定：本脚本是 v1.1.2 的临时工具，v1.1.2 发布后直接删除，不要继续维护。
它不修改前端，也不重传视频：直接登录后端 API，拉取缺封面的预告片文件，
用 ffmpeg 抽第一帧上传为 poster，再通过聚合更新接口把 poster_path 写回。

用法:
    ADMIN_PASSWORD=xxx python3 scripts/backfill-video-posters-api.py
    ADMIN_PASSWORD=xxx python3 scripts/backfill-video-posters-api.py --dry-run
    ADMIN_PASSWORD=xxx python3 scripts/backfill-video-posters-api.py <game-public-id> [...]

环境变量:
    API_BASE        后端 API 根地址，默认 http://127.0.0.1:3000/api
    ADMIN_PASSWORD  管理员密码，默认 1234（与后端默认值一致）
    FFMPEG          ffmpeg 可执行文件，默认 ffmpeg
    FFPROBE         ffprobe 可执行文件，默认 ffprobe

依赖: 仅 Python 3.8+ 标准库（urllib / http.cookiejar），无需安装任何包；
需要宿主机上可用的 ffmpeg/ffprobe。
"""

import argparse
import json
import os
import subprocess
import sys
import tempfile
import uuid
from http.cookiejar import CookieJar
from urllib import error as urllib_error
from urllib import request as urllib_request

DEFAULT_API_BASE = "http://127.0.0.1:3000/api"
DEFAULT_PASSWORD = "1234"
PAGE_LIMIT = 100
POSTER_CONTENT_TYPE = "image/jpeg"


def log(msg):
    print(msg, flush=True)


class ApiClient:
    """极简 HTTP 客户端：维护登录 session cookie，标准库实现。"""

    def __init__(self, base):
        self.base = base.rstrip("/")
        self.opener = urllib_request.build_opener(urllib_request.HTTPCookieProcessor(CookieJar()))

    def origin(self):
        base = self.base
        for suffix in ("/api", "/api/"):
            if base.endswith(suffix):
                return base[: -len(suffix)]
        return base

    def request(self, method, path, data=None, headers=None):
        return self.url_request(self.base + path, method=method, data=data, headers=headers)

    def url_request(self, url, method, data=None, headers=None):
        body = None
        if data is not None:
            body = data if isinstance(data, bytes) else json.dumps(data).encode("utf-8")
        req = urllib_request.Request(url, data=body, method=method, headers=headers or {})
        try:
            with self.opener.open(req, timeout=120) as resp:
                return resp.status, resp.read()
        except urllib_error.HTTPError as exc:
            return exc.code, exc.read()

    def json_request(self, method, path, data=None, headers=None):
        status, body = self.request(method, path, data=data, headers=headers)
        if not (200 <= status < 300):
            detail = ""
            try:
                parsed = json.loads(body.decode("utf-8"))
                detail = parsed.get("error") or parsed.get("data") or body.decode("utf-8", "replace")
            except (ValueError, UnicodeDecodeError):
                detail = body.decode("utf-8", "replace")
            raise RuntimeError(f"{method} {path} -> HTTP {status}: {detail}")
        try:
            return json.loads(body.decode("utf-8"))
        except (ValueError, UnicodeDecodeError) as exc:
            raise RuntimeError(f"{method} {path} -> 响应不是合法 JSON: {exc}") from exc

    def login(self, password):
        result = self.json_request("POST", "/auth/login", data={"password": password})
        if not result.get("success"):
            raise RuntimeError("登录失败")
        log("已登录后端 API")

    def list_games(self):
        games = []
        page = 1
        while True:
            result = self.json_request("GET", f"/games?page={page}&limit={PAGE_LIMIT}")
            pagination = result.get("pagination") or {}
            games.extend(result.get("data") or [])
            total_pages = pagination.get("totalPages", 1)
            if page >= total_pages:
                break
            page += 1
        return games

    def get_game_detail(self, public_id):
        result = self.json_request("GET", f"/games/{public_id}")
        return result.get("data") or {}

    def download(self, path):
        # 素材文件走站点根路径（/assets/...），不带 /api 前缀。
        status, body = self.url_request(self.origin() + path, "GET")
        if not (200 <= status < 300):
            raise RuntimeError(f"GET {path} -> HTTP {status}")
        return body

    def upload_poster(self, game_id, jpeg_bytes):
        boundary = "----GameManagerBackfill" + uuid.uuid4().hex
        parts = []
        parts.append(f"--{boundary}\r\nContent-Disposition: form-data; name=\"game_id\"\r\n\r\n{game_id}\r\n".encode("utf-8"))
        parts.append(
            f"--{boundary}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"poster.jpg\"\r\n"
            f"Content-Type: {POSTER_CONTENT_TYPE}\r\n\r\n".encode("utf-8")
        )
        parts.append(jpeg_bytes)
        parts.append(f"\r\n--{boundary}--\r\n".encode("utf-8"))
        body = b"".join(parts)
        headers = {"Content-Type": f"multipart/form-data; boundary={boundary}"}
        result = self.json_request("POST", "/assets/poster", data=body, headers=headers)
        return (result.get("data") or {}).get("path"), (result.get("data") or {}).get("asset_uid")

    def update_aggregate(self, public_id, payload):
        result = self.json_request(
            "PUT", f"/games/{public_id}/aggregate",
            data=payload, headers={"Content-Type": "application/json"},
        )
        return result


def run_ffprobe_duration(ffprobe, video_path):
    try:
        proc = subprocess.run(
            [ffprobe, "-v", "error", "-show_entries", "format=duration",
             "-of", "default=noprint_wrappers=1:nokey=1", video_path],
            capture_output=True, text=True, timeout=120,
        )
        if proc.returncode != 0:
            return None
        return float(proc.stdout.strip())
    except (OSError, ValueError, subprocess.TimeoutExpired):
        return None


def extract_poster_frame(ffmpeg, video_path, seek_time, output_path):
    """与前端 frontend/src/utils/video-poster.ts 的策略一致：
    时长 >1s 取 min(1, duration*0.1)，否则取 0.01s；输出 640 宽 JPEG。"""
    subprocess.run(
        [ffmpeg, "-y", "-v", "error", "-ss", f"{seek_time:.3f}", "-i", video_path,
         "-frames:v", "1", "-vf", "scale=640:-2", output_path],
        check=True, capture_output=True, timeout=300,
    )


def pick_seek_time(duration):
    if duration is not None and duration > 1:
        return min(1.0, duration * 0.1)
    return 0.01


def build_aggregate_payload(detail, poster_by_uid):
    """聚合接口是全量替换语义：必须把详情里的所有资产/文件完整回显，
    否则不在提交集合内的资产会被删除。视频只补 poster_path，不重传视频。"""
    game = {
        "title": detail.get("title", ""),
        "title_alt": detail.get("title_alt"),
        "visibility": detail.get("visibility", "public"),
        "summary": detail.get("summary"),
        "release_date": detail.get("release_date"),
        "logo_visible": detail.get("logo_visible", False),
        "series_id": (detail.get("series") or {}).get("id") if detail.get("series") else None,
        "developer_ids": [item.get("id") for item in (detail.get("developers") or [])],
        "publisher_ids": [item.get("id") for item in (detail.get("publishers") or [])],
    }

    assets = {
        "files": [
            {
                "id": item.get("id"),
                "file_path": item.get("file_path", ""),
                "label": item.get("label"),
                "notes": item.get("notes"),
            }
            for item in (detail.get("files") or [])
        ],
        "new_assets": [],
        "screenshot_order_asset_uids": [a.get("asset_uid") for a in (detail.get("screenshots") or [])],
        "video_order_asset_uids": [a.get("asset_uid") for a in (detail.get("preview_videos") or [])],
        "cover_order_asset_uids": [a.get("asset_uid") for a in (detail.get("covers") or [])],
        "logo_order_asset_uids": [a.get("asset_uid") for a in (detail.get("logos") or [])],
        "banner_order_asset_uids": [a.get("asset_uid") for a in (detail.get("banners") or [])],
        "logo_positions": [
            {
                "asset_uid": a.get("asset_uid"),
                "position_x": a.get("position_x"),
                "position_y": a.get("position_y"),
                "width_pct": a.get("width_pct"),
            }
            for a in (detail.get("logos") or [])
        ],
    }

    for group, asset_type in (
        ("screenshots", "screenshot"),
        ("preview_videos", "video"),
        ("covers", "cover"),
        ("logos", "logo"),
        ("banners", "banner"),
    ):
        for asset in detail.get(group) or []:
            entry = {
                "asset_uid": asset.get("asset_uid"),
                "asset_type": asset_type,
                "path": asset.get("path", ""),
            }
            if asset_type == "video" and asset.get("asset_uid") in poster_by_uid:
                entry["poster_path"] = poster_by_uid[asset.get("asset_uid")]
            assets["new_assets"].append(entry)

    return {"game": game, "assets": assets}


def video_without_poster(video):
    poster = video.get("poster_path")
    return poster is None or not str(poster).strip()


def process_game(client, ffmpeg, ffprobe, public_id, dry_run):
    detail = client.get_game_detail(public_id)
    videos = detail.get("preview_videos") or []
    missing = [v for v in videos if video_without_poster(v)]

    if not videos:
        log(f"[{public_id}] 无预告片，跳过")
        return 0, 0, []
    if not missing:
        log(f"[{public_id}] 所有预告片已有封面，跳过")
        return 0, 0, []
    if dry_run:
        for v in missing:
            log(f"[{public_id}] 预检：将补生成封面 uid={v.get('asset_uid')} path={v.get('path')}")
        return len(missing), 0, []

    posters = {}
    failures = []
    with tempfile.TemporaryDirectory(prefix="gameatlas-backfill-") as tmp_dir:
        for video in missing:
            uid = video.get("asset_uid")
            video_path = video.get("path", "")
            filename = os.path.basename(video_path.rstrip("/"))
            log(f"[{public_id}] 处理预告片 uid={uid} path={video_path}")
            try:
                if not filename:
                    raise RuntimeError("详情中视频 path 为空")
                local_video = os.path.join(tmp_dir, filename)
                with open(local_video, "wb") as out:
                    out.write(client.download(f"/assets/{public_id}/{filename}"))

                duration = run_ffprobe_duration(ffprobe, local_video)
                seek_time = pick_seek_time(duration)
                local_poster = os.path.join(tmp_dir, f"poster-{uid}.jpg")
                extract_poster_frame(ffmpeg, local_video, seek_time, local_poster)
                with open(local_poster, "rb") as poster_file:
                    poster_jpeg = poster_file.read()

                staging_path, _ = client.upload_poster(detail.get("id"), poster_jpeg)
                if not staging_path:
                    raise RuntimeError("上传 poster 未返回 path")
                posters[uid] = staging_path
                log(f"[{public_id}] 已上传封面帧 {staging_path}")
            except Exception as exc:  # noqa: BLE001 - 单条视频失败不中断整个任务
                log(f"[{public_id}] 视频 uid={uid} 处理失败: {exc}")
                failures.append((uid, str(exc)))

    if not posters:
        log(f"[{public_id}] 没有成功生成封面，跳过聚合更新")
        return len(missing), 0, failures

    payload = build_aggregate_payload(detail, posters)
    client.update_aggregate(public_id, payload)
    log(f"[{public_id}] 聚合更新完成，补写 {len(posters)} 个封面")

    # 回读校验：确认 poster_path 已写入
    refreshed = client.get_game_detail(public_id)
    remaining = [v for v in (refreshed.get("preview_videos") or []) if video_without_poster(v)]
    if remaining:
        log(f"[{public_id}] 警告：仍有 {len(remaining)} 个预告片缺封面")
        failures.extend((v.get("asset_uid"), "回读校验仍缺封面") for v in remaining)
    return len(missing), len(posters), failures


def resolve_targets(client, game_ids):
    if game_ids:
        return [{"public_id": pid} for pid in game_ids]
    return client.list_games()


def main(argv=None):
    parser = argparse.ArgumentParser(description="为已有预告片批量补生成封面帧（临时 helper，v1.1.2 后移除）")
    parser.add_argument("game_ids", nargs="*", help="可选：指定游戏 public_id；不传则处理全部游戏")
    parser.add_argument("--dry-run", action="store_true", help="只预检缺封面的预告片，不做上传和更新")
    parser.add_argument("--password", default=None, help="管理员密码（默认取 ADMIN_PASSWORD 环境变量，缺省 1234）")
    args = parser.parse_args(argv)

    api_base = os.environ.get("API_BASE", DEFAULT_API_BASE).rstrip("/")
    password = args.password or os.environ.get("ADMIN_PASSWORD") or DEFAULT_PASSWORD
    ffmpeg = os.environ.get("FFMPEG", "ffmpeg")
    ffprobe = os.environ.get("FFPROBE", "ffprobe")

    client = ApiClient(api_base)
    client.login(password)

    targets = resolve_targets(client, args.game_ids)
    log(f"共 {len(targets)} 个游戏待检查" + ("（dry-run，不会写任何数据）" if args.dry_run else ""))

    total_videos = 0
    total_backfilled = 0
    total_failures = []
    for target in targets:
        public_id = target.get("public_id")
        if not public_id:
            continue
        try:
            missing, backfilled, failures = process_game(client, ffmpeg, ffprobe, public_id, args.dry_run)
        except Exception as exc:  # noqa: BLE001 - 单个游戏失败不中断整个任务
            log(f"[{public_id}] 处理失败: {exc}")
            total_failures.append((public_id, str(exc)))
            continue
        total_videos += missing
        total_backfilled += backfilled
        total_failures.extend((public_id, f"{uid}: {reason}") for uid, reason in failures)

    log("")
    log(f"完成：缺封面预告片 {total_videos}，成功补写 {total_backfilled}，失败 {len(total_failures)}")
    for public_id, reason in total_failures:
        log(f"  失败 [{public_id}] {reason}")
    if total_failures:
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
