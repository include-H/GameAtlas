#!/usr/bin/env python3
"""
GameAtlas 游戏库扫描器
自动扫描目录中的 VHD 文件并添加到 GameAtlas
"""

import os
import re
import json
import requests
from pathlib import Path
from typing import Optional

# 配置
API_BASE = "http://127.0.0.1:3000/api"
ADMIN_PASSWORD = "1234"  # 默认密码，首次运行后请修改
GAME_ROOT = "/mnt/Game"  # 游戏库根目录

class GameScanner:
    def __init__(self, api_base: str, password: str):
        self.api_base = api_base
        self.session = requests.Session()
        self.login(password)
    
    def login(self, password: str):
        """登录获取 token"""
        resp = self.session.post(f"{self.api_base}/auth/login", json={"password": password})
        if resp.status_code != 200:
            raise Exception(f"登录失败: {resp.text}")
        print("✓ 登录成功")
    
    def get_existing_games(self) -> dict:
        """获取已存在的游戏列表"""
        resp = self.session.get(f"{self.api_base}/games", params={"page_size": 1000})
        if resp.status_code != 200:
            return {}
        data = resp.json().get("data", {})
        games = data.get("games", [])
        return {g["title"]: g for g in games}
    
    def get_existing_series(self) -> dict:
        """获取已存在的系列列表"""
        resp = self.session.get(f"{self.api_base}/series")
        if resp.status_code != 200:
            return {}
        series = resp.json().get("data", [])
        return {s["name"]: s for s in series}
    
    def create_game(self, title: str, visibility: str = "public") -> Optional[dict]:
        """创建游戏"""
        resp = self.session.post(f"{self.api_base}/games", json={
            "title": title,
            "visibility": visibility
        })
        if resp.status_code != 200:
            print(f"  ✗ 创建失败: {resp.text}")
            return None
        return resp.json().get("data")
    
    def update_game_aggregate(self, public_id: str, title: str, file_path: str, 
                              series_id: Optional[int] = None) -> bool:
        """更新游戏聚合信息（包含文件路径）"""
        payload = {
            "game": {
                "title": title,
                "visibility": "public"
            },
            "assets": {
                "files": [{"file_path": file_path}],
                "new_assets": [],
                "screenshot_order_asset_uids": [],
                "video_order_asset_uids": [],
                "cover_order_asset_uids": [],
                "logo_order_asset_uids": [],
                "banner_order_asset_uids": [],
                "logo_positions": []
            }
        }
        if series_id:
            payload["game"]["series_id"] = series_id
        
        resp = self.session.put(
            f"{self.api_base}/games/{public_id}/aggregate",
            json=payload
        )
        return resp.status_code == 200
    
    def create_series(self, name: str) -> Optional[dict]:
        """创建系列"""
        resp = self.session.post(f"{self.api_base}/series", json={"name": name})
        if resp.status_code != 200:
            return None
        return resp.json().get("data")
    
    def extract_game_title(self, filename: str) -> str:
        """从文件名提取游戏标题"""
        # 移除扩展名
        title = Path(filename).stem
        # 移除版本信息（如 v1.0, Build.xxx）
        title = re.sub(r'\s*v?\d+[\.\d]*', '', title)
        title = re.sub(r'\s*Build\.\d+', '', title)
        return title.strip()
    
    def extract_series_name(self, dir_name: str) -> str:
        """从目录名提取系列名称（移除字母前缀）"""
        # 移除 "X " 格式的前缀（如 "B 半条命" -> "半条命"）
        match = re.match(r'^[A-Z]\s+(.+)$', dir_name)
        if match:
            return match.group(1)
        return dir_name
    
    def scan_directory(self, root_dir: str):
        """扫描目录并添加游戏"""
        print(f"\n扫描目录: {root_dir}")
        
        # 获取已存在的游戏和系列
        existing_games = self.get_existing_games()
        existing_series = self.get_existing_series()
        print(f"已存在游戏: {len(existing_games)} 个")
        print(f"已存在系列: {len(existing_series)} 个")
        
        # 统计
        stats = {
            "total_vhd": 0,
            "new_games": 0,
            "skipped": 0,
            "errors": 0
        }
        
        # 扫描所有 VHD 文件
        for dirpath, dirnames, filenames in os.walk(root_dir):
            for filename in filenames:
                if not filename.lower().endswith(('.vhd', '.vhdx')):
                    continue
                
                stats["total_vhd"] += 1
                filepath = os.path.join(dirpath, filename)
                rel_path = os.path.relpath(filepath, root_dir)
                
                # 提取游戏标题
                title = self.extract_game_title(filename)
                
                # 检查是否已存在
                if title in existing_games:
                    print(f"  ⊘ 跳过已存在: {title}")
                    stats["skipped"] += 1
                    continue
                
                # 确定系列
                series_id = None
                rel_dir = os.path.relpath(dirpath, root_dir)
                if rel_dir.startswith("有系列的游戏/") and rel_dir != "有系列的游戏":
                    series_dir = rel_dir.split("/")[1]
                    series_name = self.extract_series_name(series_dir)
                    
                    # 获取或创建系列
                    if series_name not in existing_series:
                        series = self.create_series(series_name)
                        if series:
                            existing_series[series_name] = series
                            print(f"  + 创建系列: {series_name}")
                    
                    if series_name in existing_series:
                        series_id = existing_series[series_name]["id"]
                
                # 创建游戏
                print(f"  + 添加游戏: {title}")
                game = self.create_game(title)
                if not game:
                    stats["errors"] += 1
                    continue
                
                # 更新游戏文件路径
                # 注意：这里使用相对路径，需要根据实际配置调整
                file_path = rel_path
                if self.update_game_aggregate(game["public_id"], title, file_path, series_id):
                    existing_games[title] = game
                    stats["new_games"] += 1
                else:
                    print(f"    ✗ 更新文件路径失败")
                    stats["errors"] += 1
        
        # 打印统计
        print(f"\n扫描完成:")
        print(f"  VHD 文件总数: {stats['total_vhd']}")
        print(f"  新增游戏: {stats['new_games']}")
        print(f"  跳过已存在: {stats['skipped']}")
        print(f"  错误: {stats['errors']}")


def main():
    import argparse
    parser = argparse.ArgumentParser(description="GameAtlas 游戏库扫描器")
    parser.add_argument("--api", default=API_BASE, help="API 地址")
    parser.add_argument("--password", default=ADMIN_PASSWORD, help="管理员密码")
    parser.add_argument("--dir", default=GAME_ROOT, help="游戏库目录")
    args = parser.parse_args()
    
    scanner = GameScanner(args.api, args.password)
    scanner.scan_directory(args.dir)


if __name__ == "__main__":
    main()
