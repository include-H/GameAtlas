#!/usr/bin/env python3
"""
GameAtlas Wiki 批量同步脚本
将本地 Game_Wiki 仓库重构后的 Wiki 内容推送到生产环境 GameAtlas。

用法:
    python3 sync_wiki_to_prod.py                     # 全部同步（含无变化跳过）
    python3 sync_wiki_to_prod.py --force             # 强制全部覆盖
    python3 sync_wiki_to_prod.py --dry-run           # 只打印计划，不写入
    python3 sync_wiki_to_prod.py --game "半条命"      # 只同步标题匹配的游戏
    python3 sync_wiki_to_prod.py --list              # 列出映射关系

环境变量:
    GA_URL       目标环境地址（必填）
    GA_PASSWORD  管理员密码（写入模式必填）

匹配策略:
    1. 先尝试按标题自动匹配（精确/子串，归一化全半角）
    2. 未命中则查手工映射表 MANUAL_MAP
    3. 标题重复时用 AUTO_MAP_OVERRIDE 按 public_id 消歧
"""
import argparse
import http.cookiejar
import json
import os
import re
import sys
import time
import urllib.error
import urllib.request

DEFAULT_WIKI_ROOT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "Game_Wiki")

GA_URL = os.environ.get("GA_URL", "")
GA_PASSWORD = os.environ.get("GA_PASSWORD", "")

_jar = http.cookiejar.CookieJar()
_opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(_jar))

# 自动匹配无法消除歧义时，按 public_id 强制指定本地文件
AUTO_MAP_OVERRIDE = {
    # (public_id 前缀, 本地相对路径)
    ("febf55a6", "使命召唤/使命召唤_现代战争_2019_WIKI.md"),
    ("38ac95a3", "使命召唤/使命召唤4_现代战争_WIKI.md"),
}

# 自动匹配无法命中的标题 -> 本地相对路径（相对于 WIKI_ROOT）
MANUAL_MAP = {
    "人中之龙 极2": "如龙/如龙_极2_WIKI.md",
    "人中之龙 极": "如龙/如龙_极_WIKI.md",
    "人中之龙０ 誓约的场所": "如龙/如龙0_誓约的场所_WIKI.md",
    "人中之龙3": "如龙/如龙3_WIKI.md",
    "人中之龙4 继承传说者": "如龙/如龙4_传说的继承者_WIKI.md",
    "人中之龙5 圆梦者": "如龙/如龙5_实现梦想者_WIKI.md",
    "人中之龙6 生命诗歌": "如龙/如龙6_生命诗篇_WIKI.md",
    "人中之龙7 光与暗的去向": "如龙/如龙7_光与暗的去向_WIKI.md",
    "细胞分裂2：明日潘多拉": "汤姆克兰西/细胞分裂/细胞分裂_潘多拉明日_WIKI.md",
    "细胞分裂3：混沌理论": "汤姆克兰西/细胞分裂/细胞分裂_混沌理论_WIKI.md",
    "细胞分裂4：双重间谍": "汤姆克兰西/细胞分裂/细胞分裂_双重间谍_WIKI.md",
    "细胞分裂5：断罪": "汤姆克兰西/细胞分裂/细胞分裂_断罪_WIKI.md",
    "细胞分裂6：黑名单": "汤姆克兰西/细胞分裂/细胞分裂_黑名单_WIKI.md",
    "空之轨迹 THE 1ST": "英雄传说/轨迹系列/英雄传说_空之轨迹FC_WIKI.md",
    "上古卷轴4：湮灭": "上古卷轴/上古卷轴_IV_湮灭_WIKI.md",
    "上古卷轴5：天际": "上古卷轴/上古卷轴_V_天际_WIKI.md",
    "死或生：维纳斯璀璨假期": "死或生沙滩排球/死或生_沙滩排球_维纳斯假期_WIKI.md",
    "古墓丽影4-6复刻版": "古墓丽影/古墓丽影_IV-VI_重制版_WIKI.md",
    "古墓丽影1-3复刻版": "古墓丽影/古墓丽影_I-III_重制版_WIKI.md",
    "潜龙谍影 Δ：食蛇者": "合金装备/合金装备_3_食蛇者_生存_WIKI.md",
    "生化危机4：重置版": "生化危机/生化危机_4_2023_WIKI.md",
    "神秘海域：盗贼遗产合集": "神秘海域/神秘海域_2_纵横四海_WIKI.md",
    "帝国时代4": "帝国时代/帝国时代_IV_WIKI.md",
    "帝国时代3：决定版": "帝国时代/帝国时代_III_WIKI.md",
    "帝国时代2：决定版": "帝国时代/帝国时代_II_WIKI.md",
    "帝国时代：决定版": "帝国时代/帝国时代_初代_与_罗马复兴_WIKI.md",
    "狙击手：幽灵战士契约 2": "狙击手幽灵战士/狙击手_幽灵战士_Contracts_2_WIKI.md",
    "狙击手：幽灵战士契约": "狙击手幽灵战士/狙击手_幽灵战士_Contracts_WIKI.md",
    "往日不在": "往日不再/往日不再_与_Days_Gone_Remastered_WIKI.md",
    "极品飞车：热力追踪": "极品飞车/极品飞车_14_热力追踪_WIKI.md",
    "极品飞车：热力追踪 2": "极品飞车/极品飞车_6_热力追踪_2_WIKI.md",
    "极品飞车：宿敌": "极品飞车/极品飞车_18_宿敌_WIKI.md",
    "极品飞车：亡命天涯": "极品飞车/极品飞车_16_亡命天涯_WIKI.md",
    "黑手党": "黑手党/黑手党_2002_与_黑手党_最终版_WIKI.md",
    "黑手党2": "黑手党/黑手党_II_与_黑手党_II_最终版_WIKI.md",
    "黑手党3": "黑手党/黑手党_III_与_黑手党_III_最终版_及_DLC_WIKI.md",
    "黑山：起源": "半条命/黑山_WIKI.md",
    "地铁：逃离": "地铁/地铁_离去_WIKI.md",
    "飙酷车神2": "飙酷车神/飙酷车神_1_与_2_代_WIKI.md",
    "尼尔：机械纪元": "尼尔/尼尔_自动人形_WIKI.md",
    "使命召唤：无限战争": "使命召唤/使命召唤_无尽战争_WIKI.md",
    "文明5": "文明/文明_V_WIKI.md",
    "文明6": "文明/文明_VI_WIKI.md",
    "蝙蝠侠：阿卡姆骑士": "蝙蝠侠/蝙蝠侠_阿甘骑士_WIKI.md",
    "蝙蝠侠：阿卡姆之城": "蝙蝠侠/蝙蝠侠_阿甘之城_WIKI.md",
    "德军总部：旧血液": "德军总部/德军总部_旧血脉_WIKI.md",
    "合金装备5：幻痛/原爆点": "合金装备/合金装备_V_原爆点_幻痛_与整合版_WIKI.md",
    "看门狗": "看门狗/看门狗_2014_WIKI.md",
    "细胞分裂": "汤姆克兰西/细胞分裂/细胞分裂_2002_WIKI.md",
    "刺客信条：黑旗": "刺客信条/刺客信条_IV_黑旗_WIKI.md",
    "杀手5：赦免": "杀手/杀手_赦免_WIKI.md",
    "荣誉勋章v": "荣誉勋章/荣誉勋章_2010_WIKI.md",
    "荣誉勋章：三部曲合集": "荣誉勋章/荣誉勋章_联合袭击_与资料片_先锋_突破_WIKI.md",
    "侠盗猎车手 3": "侠盗猎车手/侠盗猎车手_III_WIKI.md",
    "半条命": "半条命/半条命与资料片_针锋相对_蓝色沸点_衰变_WIKI.md",
    "半条命2": "半条命/半条命_2_与_第一章_第二章_失落的海岸线_WIKI.md",
    "使命召唤": "使命召唤/使命召唤_2003_与资料片_联合进攻_WIKI.md",
    "使命召唤：现代战争": "使命召唤/使命召唤_现代战争_2019_WIKI.md",
    "使命召唤：现代战争 2": "使命召唤/使命召唤_现代战争2_2009_WIKI.md",
    "使命召唤：现代战争 3": "使命召唤/使命召唤_现代战争3_2011_WIKI.md",
    "丧尸围城": "丧尸围城/丧尸围城_初代_WIKI.md",
    "古墓丽影": "古墓丽影/古墓丽影_2013_与_DLC_WIKI.md",
    "古墓丽影：崛起": "古墓丽影/古墓丽影_崛起_与_DLC_WIKI.md",
    "古墓丽影：暗影": "古墓丽影/古墓丽影_暗影_与_DLC_WIKI.md",
    "古墓丽影：地下世界": "古墓丽影/古墓丽影_地下世界_与_DLC_WIKI.md",
    "古墓丽影：十周年纪念版": "古墓丽影/古墓丽影_十周年纪念_WIKI.md",
    "生化危机1": "生化危机/生化危机_1996_WIKI.md",
    "生化危机7": "生化危机/生化危机_7_生化危机_WIKI.md",
    "孤岛惊魂5": "孤岛惊魂/孤岛惊魂_5_DLC_与_新曙光_WIKI.md",
    "天国：拯救": "天国拯救/天国_拯救_与_Royal_Edition_WIKI.md",
    "重返德军总部：新巨像": "德军总部/德军总部_II_新巨像_WIKI.md",
    "奇异人生：风暴前夕": "奇异人生/奇异人生_暴风前夕_WIKI.md",
    "地平线：西之绝境": "地平线/地平线_西之绝境与_炽焰海岸_WIKI.md",
    "地平线：零之曙光": "地平线/地平线_零之曙光与_冰尘雪野_WIKI.md",
    "命令与征服：红色警戒": "红色警戒/命令与征服_红色警戒_资料片与复刻版_WIKI.md",
    "命令与征服：红色警戒3": "红色警戒/命令与征服_红色警戒_3_与_起义时刻_WIKI.md",
    "漫威蜘蛛侠：莫拉莱斯": "漫威蜘蛛侠/漫威蜘蛛侠_迈尔斯_莫拉莱斯_WIKI.md",
    "漫威蜘蛛侠：重制版": "漫威蜘蛛侠/漫威蜘蛛侠_WIKI.md",
    "孤岛危机": "孤岛危机/孤岛危机_初代_WIKI.md",
    "敌后敢死队2：血战太平洋": "敌后敢死队/敌后敢死队_太平洋战场_WIKI.md",
    "模拟人生": "模拟人生/模拟人生_1_与_2_代_WIKI.md",
    "反恐精英": "反恐精英/反恐精英初代_1_6_零点行动_与_起源_WIKI.md",
    "侠盗猎车手4": "侠盗猎车手/侠盗猎车手_IV_WIKI.md",
    "侠盗猎车手5": "侠盗猎车手/侠盗猎车手_V_WIKI.md",
    "消逝的光芒2": "消逝的光芒/消逝的光芒_2_人与仁之战_WIKI.md",
    "消逝的光芒": "消逝的光芒/消逝的光芒_与_The_Following_WIKI.md",
    "消逝的光芒：困兽": "消逝的光芒/消逝的光芒_困兽_WIKI.md",
    "仙剑客栈": "仙剑奇侠传/仙剑客栈_WIKI.md",
    "英雄连3": "英雄连/英雄连_3_WIKI.md",
    "仙剑奇侠传2": "仙剑奇侠传/仙剑奇侠传二_WIKI.md",
    "仙剑奇侠传3": "仙剑奇侠传/仙剑奇侠传三_WIKI.md",
    "仙剑奇侠传4": "仙剑奇侠传/仙剑奇侠传四_WIKI.md",
    "仙剑奇侠传5": "仙剑奇侠传/仙剑奇侠传五_WIKI.md",
    "仙剑奇侠传6": "仙剑奇侠传/仙剑奇侠传六_WIKI.md",
    "仙剑奇侠传7": "仙剑奇侠传/仙剑奇侠传七_WIKI.md",
    "最终幻想1-6复刻版": "最终幻想/最终幻想_I_VI_WIKI.md",
    "最终幻想7 重置版": "最终幻想/最终幻想_VII_Remake_WIKI.md",
    "最终幻想7 重生": "最终幻想/最终幻想_VII_Rebirth_WIKI.md",
    "最终幻想：核心危机 重聚": "最终幻想/最终幻想_VII_原版与_Compilation_WIKI.md",
    "最终幻想16": "最终幻想/最终幻想_XVI_WIKI.md",
    "最终幻想13 合集": "最终幻想/新水晶神话/最终幻想_XIII_及其衍生作品_WIKI.md",
    "最终幻想15": "最终幻想/新水晶神话/Final_Fantasy_Versus_XIII_与_最终幻想_XV_WIKI.md",
    "真三国无双4": "真三国无双/真・三國無双_4_WIKI.md",
    "真三国无双5": "真三国无双/真・三國無双_5_WIKI.md",
    "真三国无双6": "真三国无双/真・三國無双_6_WIKI.md",
    "真三国无双7": "真三国无双/真・三國無双_7_WIKI.md",
    "真三国无双8": "真三国无双/真・三國無双_8_WIKI.md",
    "真三国无双：起源": "真三国无双/真・三國無双_ORIGINS_WIKI.md",
    "战神4": "战神/战神_2018_WIKI.md",
    "战神：诸神黄昏": "战神/God_of_War_Ragnar_k_与_Valhalla_WIKI.md",
    "光环2：周年版": "光环/光环_2_周年版_WIKI.md",
    "战地1942": "战地/战地_系列其余作品综述_WIKI.md",
    "战地6": "战地/战地_6_WIKI.md",
    "镜之边缘": "镜之边缘/镜之边缘_2008_WIKI.md",
    "大富翁": "大富翁/大富翁_系列合并_WIKI.md",
    "最后的生还者：第一章": "最后生还者/最后的生还者_2013_遗落_与主要版本关系_WIKI.md",
    "最后的生还者：第二章": "最后生还者/最后的生还者_第二部与_Remastered_WIKI.md",
    "樱花大战": "樱花大战/樱花大战_1996_WIKI.md",
    "纪念碑谷": "纪念碑谷/纪念碑谷与_Forgotten_Shores_WIKI.md",
}


def http_request(method, path, payload=None, timeout=60):
    url = GA_URL + path
    data = None
    headers = {}
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with _opener.open(req, timeout=timeout) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8", errors="replace")
        print(f"  [HTTP {e.code}] {method} {path}: {body[:200]}")
        return None
    except urllib.error.URLError as e:
        print(f"  [网络错误] {method} {path}: {e}")
        return None


def login():
    resp = http_request("POST", "/api/auth/login", {"password": GA_PASSWORD})
    if not resp or not resp.get("success"):
        print(f"登录失败: {resp}")
        sys.exit(1)
    print(f"已登录管理员: {resp.get('data', {}).get('admin_display_name', 'ok')}")


def fetch_all_games():
    games = []
    page = 1
    while True:
        url = f"{GA_URL}/api/games?page={page}&limit=100"
        try:
            with _opener.open(url, timeout=15) as r:
                d = json.load(r)
        except Exception as e:
            print(f"拉取游戏列表失败: {e}")
            return games
        data = d.get("data", [])
        games.extend(data)
        pag = d.get("pagination", {})
        if page >= (pag.get("totalPages") or 1):
            break
        page += 1
        time.sleep(0.15)
    return games


def norm_title(s):
    s = (s or "").replace("：", ":").replace("，", ",").replace("．", ".")
    s = re.sub(r"[\s_\-—、。，,.:：()（）\[\]【】★™®～〜]", "", s)
    for a, b in zip("０１２３４５６７８９", "0123456789"):
        s = s.replace(a, b)
    return s.lower()


def strip_version_mods(s):
    s = s or ""
    for m in ["决定版", "复刻版", "重制版", "最终版", "周年版", "周年纪念版",
              "合集", "终极版", "高清版", "豪华版", "导演剪辑版", "特别版", "典藏版", "dx"]:
        s = s.replace(m, "")
    return s


def build_local_index(wiki_root):
    idx = {}
    for dp, dn, fn in os.walk(wiki_root):
        dn[:] = [d for d in dn if d != ".git" and d != ".omo"]
        for f in fn:
            if not f.endswith("WIKI.md"):
                continue
            rel = os.path.relpath(os.path.join(dp, f), wiki_root).replace(os.sep, "/")
            base = os.path.basename(f)[:-8]
            key = norm_title(base).replace("_", "")
            idx.setdefault(key, []).append(rel)
    return idx


def auto_match(title, title_alt, local_idx):
    for cand in (title, title_alt):
        tn = norm_title(strip_version_mods(cand))
        if not tn:
            continue
        if tn in local_idx and len(local_idx[tn]) == 1:
            return local_idx[tn][0]
    for cand in (title, title_alt):
        tn = norm_title(strip_version_mods(cand))
        if not tn:
            continue
        for key, rels in local_idx.items():
            if (len(key) >= 4 and key in tn) or (len(tn) >= 4 and tn in key):
                if len(rels) == 1:
                    return rels[0]
    return None


def resolve_local_file(game, local_idx, wiki_root):
    title = game["title"]
    public_id = game["public_id"]
    for prefix, rel in AUTO_MAP_OVERRIDE:
        if public_id.startswith(prefix):
            return rel
    rel = MANUAL_MAP.get(title)
    if rel and os.path.exists(os.path.join(wiki_root, rel)):
        return rel
    hit = auto_match(title, game.get("title_alt"), local_idx)
    if hit:
        return hit
    return None


def fetch_wiki(public_id):
    resp = http_request("GET", f"/api/games/{public_id}/wiki")
    if resp and resp.get("success"):
        return resp["data"].get("content")
    return None


def update_wiki(public_id, content, summary):
    resp = http_request(
        "PUT", f"/api/games/{public_id}/wiki",
        {"content": content, "change_summary": summary},
    )
    return bool(resp and resp.get("success"))


def main():
    global GA_URL, GA_PASSWORD
    ap = argparse.ArgumentParser(description="同步本地 Game_Wiki 到生产 GameAtlas")
    ap.add_argument("--force", action="store_true", help="强制覆盖，跳过内容比对")
    ap.add_argument("--dry-run", action="store_true", help="仅打印计划不写入")
    ap.add_argument("--game", help="只同步标题包含该关键词的游戏")
    ap.add_argument("--list", action="store_true", help="列出映射关系")
    ap.add_argument("--unmatched", action="store_true", help="只显示无法匹配的游戏")
    ap.add_argument("--url", default=GA_URL, help="目标环境地址")
    ap.add_argument("--password", default=GA_PASSWORD, help="管理员密码")
    args = ap.parse_args()
    GA_URL = args.url.rstrip("/")
    GA_PASSWORD = args.password
    if not GA_URL:
        ap.error("缺少目标地址：--url 或环境变量 GA_URL")

    wiki_root = DEFAULT_WIKI_ROOT
    if not os.path.isdir(wiki_root):
        print(f"Wiki 仓库目录不存在: {wiki_root}")
        sys.exit(1)

    local_idx = build_local_index(wiki_root)

    if not args.dry_run and not args.list and not args.unmatched:
        if not GA_PASSWORD:
            ap.error("缺少管理员密码：--password 或环境变量 GA_PASSWORD")
        login()

    games = fetch_all_games()
    if not games:
        print("无法获取生产游戏列表，退出")
        sys.exit(1)

    print(f"生产库共 {len(games)} 个游戏")
    if args.list or args.unmatched:
        print("=" * 70)

    if args.unmatched:
        for g in games:
            rel = resolve_local_file(g, local_idx, wiki_root)
            if not rel:
                print(f"  ??? {g['title']}")
        return

    if args.list:
        for g in games:
            rel = resolve_local_file(g, local_idx, wiki_root)
            mark = "OK  " if rel else "MISS"
            print(f"{mark} | {g['title']:35s} -> {rel or '<<< 无匹配'}")
        return

    print("=" * 70)
    updated = skipped = failed = missing = 0
    for g in games:
        if args.game and args.game not in g["title"]:
            continue
        rel = resolve_local_file(g, local_idx, wiki_root)
        if not rel:
            print(f"[无映射] {g['title']}")
            missing += 1
            continue

        local_path = os.path.join(wiki_root, rel)
        if not os.path.exists(local_path):
            print(f"[缺失] {rel}")
            missing += 1
            continue

        with open(local_path, encoding="utf-8") as f:
            local_content = f.read()

        remote_content = g.get("wiki_content") if not args.dry_run else None
        if not args.force and not args.dry_run and (remote_content or "").strip() == local_content.strip():
            print(f"[无变化] {g['title']}")
            skipped += 1
            continue

        summary = f"同步本地重构后的 Wiki（{os.path.basename(rel)}）"
        if args.dry_run:
            print(f"[计划] {g['title']} -> {rel} ({len(local_content)} bytes)")
            updated += 1
            continue

        if update_wiki(g["public_id"], local_content, summary):
            print(f"[成功] {g['title']} ({len(local_content)} bytes)")
            updated += 1
        else:
            print(f"[失败] {g['title']}")
            failed += 1
        time.sleep(0.3)

    print("=" * 70)
    print(f"完成: 更新 {updated}，无变化跳过 {skipped}，失败 {failed}，无映射/缺失 {missing}")


if __name__ == "__main__":
    main()
