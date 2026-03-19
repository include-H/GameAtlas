# 游戏标签系统设计

## 目标

为游戏库增加一套可扩展的标签系统，用于：

- 支持多维度分类，而不是单一树状分类
- 支持列表页按标签筛选
- 支持编辑页给游戏打标签
- 与现有 `系列 / 平台 / 开发商 / 发行商 / 公开私有` 并存

本设计的核心原则：

- `公开 / 私有` 不是标签，继续保留为独立字段
- `系列 / 平台 / 开发商 / 发行商` 继续保留现有 metadata 体系
- 新增标签系统只负责“题材、子类型、视角、内容属性、玩法特征”等多维分类
- 不做一棵覆盖全部标签的大树，改做“标签组 + 标签”的分面筛选

## 为什么不能只做一棵分类树

例如：

- `GAL`
- `第一人称射击`

这两个概念并不一定处于同一层级，也不应该强行挂在同一棵树上。

更准确的表达方式是：

- `视觉小说` 属于题材
- `GAL` 属于子类型或内容导向
- `射击` 属于题材
- `第一人称` 属于视角

因此系统应采用“多维度标签”而非“全局树”。

## 设计总览

建议将分类分为两类：

### 1. 结构化元数据

继续沿用现有模型：

- 系列
- 平台
- 开发商
- 发行商
- 可见性

这些字段要么已经有独立表，要么本身就是游戏主表字段。

### 2. 标签系统

新增一套通用标签结构，用来表达：

- 题材：视觉小说、射击、RPG、策略
- 子类型：GAL、类银河恶魔城、战棋、生存恐怖
- 视角：第一人称、第三人称、俯视角、横版
- 玩法特征：单人、多人、合作、开放世界、潜行
- 内容属性：全年龄、R18、百合、科幻、悬疑
- 展示用途：精品、入门推荐、短流程、剧情向

## 数据模型

新增三张表：

### `tag_groups`

用于定义标签维度。

建议字段：

- `id INTEGER PRIMARY KEY AUTOINCREMENT`
- `key TEXT NOT NULL UNIQUE`
- `name TEXT NOT NULL`
- `description TEXT`
- `sort_order INTEGER NOT NULL DEFAULT 0`
- `allow_multiple INTEGER NOT NULL DEFAULT 1`
- `is_filterable INTEGER NOT NULL DEFAULT 1`
- `created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP`
- `updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP`

说明：

- `key` 用于接口和前端识别，例如 `genre`、`subgenre`、`perspective`
- `allow_multiple` 表示同一游戏在该组下能否选择多个标签
- `is_filterable` 表示该组是否在列表页筛选面板展示

### `tags`

用于存储具体标签。

建议字段：

- `id INTEGER PRIMARY KEY AUTOINCREMENT`
- `group_id INTEGER NOT NULL`
- `name TEXT NOT NULL`
- `slug TEXT NOT NULL`
- `parent_id INTEGER`
- `sort_order INTEGER NOT NULL DEFAULT 0`
- `is_active INTEGER NOT NULL DEFAULT 1`
- `created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP`
- `updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP`

约束建议：

- `UNIQUE(group_id, name)`
- `UNIQUE(group_id, slug)`
- `FOREIGN KEY (group_id) REFERENCES tag_groups(id) ON DELETE CASCADE`
- `FOREIGN KEY (parent_id) REFERENCES tags(id) ON DELETE SET NULL`

说明：

- `parent_id` 只作为同组内的轻量层级辅助，不作为全系统主分类逻辑
- 例如可以在 `subgenre` 组中表达一部分父子关系，但不要依赖它完成筛选逻辑

### `game_tags`

用于保存游戏和标签的多对多关系。

建议字段：

- `game_id INTEGER NOT NULL`
- `tag_id INTEGER NOT NULL`
- `sort_order INTEGER NOT NULL DEFAULT 0`
- `created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP`

约束建议：

- `PRIMARY KEY (game_id, tag_id)`
- `FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE`
- `FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE`

## 建议的首批标签组

第一期不要铺太大，建议先上 4 组：

- `genre`
  - 题材
  - 例：视觉小说、射击、RPG、策略、动作、模拟经营
- `subgenre`
  - 子类型
  - 例：GAL、战棋、类银河恶魔城、生存恐怖、Roguelite
- `perspective`
  - 视角
  - 例：第一人称、第三人称、俯视角、横版
- `theme`
  - 内容属性
  - 例：全年龄、R18、百合、悬疑、科幻、剧情向

后续可再补：

- `feature`
- `audience`
- `curation`

## 筛选语义

筛选规则必须统一，否则用户很难理解。

建议规则：

- 同一标签组内，多选按“或”处理
- 不同标签组之间，按“且”处理

例子：

- `genre=射击` 且 `perspective=第一人称`
  - 得到第一人称射击游戏
- `genre=视觉小说` 且 `subgenre=GAL`
  - 得到 GAL
- `genre=视觉小说` 且 `theme=全年龄`
  - 得到全年龄视觉小说

这个规则足够直观，也方便 SQL 实现。

## 后端接口设计

当前列表接口在 [backend/internal/http/handlers/games.go](/home/Hao/Game/backend/internal/http/handlers/games.go) 中主要支持：

- `search`
- `series`
- `platform`
- `needs_review`

建议在此基础上增加标签筛选参数。

### 游戏列表接口

`GET /api/games`

新增查询参数：

- `tag`
  - 可重复出现
  - 例：`/api/games?tag=12&tag=27&tag=31`
- `tag_mode`
  - 可选
  - 默认 `grouped`

建议后端内部语义不要直接依赖传入顺序，而是：

1. 根据 `tag_id` 查到所属 `group_id`
2. 按组聚合
3. 同组内做 OR，不同组做 AND

### 标签组列表接口

新增：

- `GET /api/tag-groups`
- `POST /api/tag-groups`

返回：

- 标签组基础信息
- 是否可筛选
- 是否允许多选

### 标签列表接口

新增：

- `GET /api/tags`
- `POST /api/tags`

查询参数支持：

- `group_id`
- `group_key`
- `active`

### 游戏详情接口

`GET /api/games/:id`

返回中新增：

- `tags`
  - 扁平列表
- `tag_groups`
  - 按组聚合后的结构，便于前端编辑页直接渲染

建议实际返回更偏向按组聚合，例如：

```json
{
  "tag_groups": [
    {
      "key": "genre",
      "name": "题材",
      "tags": [
        { "id": 1, "name": "视觉小说", "slug": "visual-novel" }
      ]
    },
    {
      "key": "perspective",
      "name": "视角",
      "tags": [
        { "id": 8, "name": "第一人称", "slug": "first-person" }
      ]
    }
  ]
}
```

### 游戏写入接口

`POST /api/games`
`PUT /api/games/:id`

在现有 `GameWriteInput` 上增加：

- `tag_ids: number[]`

服务端校验：

- 标签必须存在
- 标签必须是激活状态
- 对 `allow_multiple = 0` 的标签组，同一游戏不能提交多个标签

## 后端实现建议

### 领域层

扩展 [backend/internal/domain/game.go](/home/Hao/Game/backend/internal/domain/game.go)：

- `GamesListParams` 增加 `TagIDs []int64`
- `GameWriteInput` 增加 `TagIDs []int64`
- 新增 `TagGroup`、`Tag` 领域模型

### Repository 层

扩展 [backend/internal/repositories/games.go](/home/Hao/Game/backend/internal/repositories/games.go)：

- `List` 支持标签筛选
- `GetByID` 附加标签查询
- `Create / Update` 时维护 `game_tags`

标签筛选的 SQL 思路：

1. 先查出传入标签所属分组
2. 将标签按 `group_id` 分桶
3. 每个分组生成一个 `EXISTS` 子查询
4. 主查询中把这些子查询用 `AND` 连接

示意：

```sql
EXISTS (
  SELECT 1
  FROM game_tags gt
  WHERE gt.game_id = g.id
    AND gt.tag_id IN (1, 2, 3)
)
AND EXISTS (
  SELECT 1
  FROM game_tags gt
  WHERE gt.game_id = g.id
    AND gt.tag_id IN (8, 9)
)
```

这样天然满足：

- 组内 OR
- 组间 AND

### Metadata 体系如何共存

不建议第一期把 `series / platforms / developers / publishers` 全部重构成统一标签。

原因：

- 现有功能已经稳定
- 这些字段语义清晰，且在详情和编辑里是高频结构化信息
- 一次性通用化会扩大改动面

更稳的方案是：

- 保留现有 metadata 表和接口
- 只为标签新建一套 `tag_groups / tags / game_tags`

## 前端设计

### 列表页筛选面板

当前 [frontend/src/views/GamesView.vue](/home/Hao/Game/frontend/src/views/GamesView.vue) 已有：

- 搜索
- 系列
- 平台
- 排序
- 待处理筛选

建议新增“标签筛选”区域，表现为：

- 题材：多选
- 子类型：多选
- 视角：多选
- 内容属性：多选

交互建议：

- 默认折叠为“更多筛选”
- 展开后按标签组展示
- 每组使用多选下拉或 tag 选择器
- 当前筛选条件继续展示在顶部 active tags 区

### 编辑页

在 [frontend/src/components/EditGameModal.vue](/home/Hao/Game/frontend/src/components/EditGameModal.vue) 中新增“标签”区域。

布局建议：

- 单独一个分区，不与平台、系列混在一起
- 每个标签组一行
- 支持搜索和多选
- 若某组 `allow_multiple = 0`，前端自动切换单选模式

例：

- 题材
- 子类型
- 视角
- 内容属性

### 详情页

在详情页将标签按组展示，而不是全部平铺。

展示形式建议：

- 题材：视觉小说
- 子类型：GAL
- 视角：第一人称
- 内容属性：全年龄、剧情向

这样用户能快速理解标签的含义，不会看到一排混杂 tag。

## Steam 导入的关系

当前项目已经有 Steam 信息抓取和素材导入能力。

建议处理方式：

- Steam 的 `genres` 与 `tags` 不直接原样落库
- 先映射到你自己的标签体系

例如：

- Steam `Visual Novel` -> 本地 `genre: 视觉小说`
- Steam `FPS` -> 本地拆成 `genre: 射击` + `perspective: 第一人称`

原因：

- Steam 标签很杂
- 同义词很多
- 会把你本地筛选体系污染掉

因此建议新增一层“标签映射表”或在服务层写静态映射规则，先人工维护少量常用映射。

## 迁移策略

建议分三期上线。

### 第一期：数据层与基础读写

- 建表：`tag_groups`、`tags`、`game_tags`
- 新增标签组和标签管理接口
- 游戏读写支持 `tag_ids`
- 游戏详情返回标签

### 第二期：列表筛选

- 列表接口支持 `tag` 查询参数
- 前端列表页增加标签筛选
- 顶部 active filters 支持移除标签条件

### 第三期：增强体验

- 编辑页支持按组打标签
- Steam 标签映射
- 标签管理页
- 标签热度统计与常用筛选预设

## 非目标

以下内容不建议放进第一期：

- 全局无限层级标签树
- 标签别名自动合并
- 标签权重排序
- 标签互斥规则引擎
- 基于标签的推荐系统

这些都可以后续演进，但第一期先保证：

- 结构清晰
- 录入简单
- 筛选可理解
- SQL 可维护

## 最终建议

本项目的标签系统应采用：

- 结构化 metadata 保持现状
- 新增“标签组 + 标签 + 游戏标签关系”
- 以“分面筛选”代替“单树分类”
- 同组 OR、组间 AND

这样既能表达：

- GAL
- 第一人称射击
- 全年龄视觉小说
- 第三人称开放世界动作游戏

也不会破坏你当前已经成型的 `series / platform / visibility` 体系。

## 建议的下一步实现顺序

1. 先补 migration 和 domain model
2. 再补 repository / service / handler
3. 然后补前端 types 和 service
4. 最后接列表页筛选和编辑页打标

如果继续往下做，建议直接按这个顺序实施。
