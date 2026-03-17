# GameManager

一个面向 NAS 场景的游戏库管理系统。

当前主线架构：

- 后端：Go + Gin + SQLite
- 前端：Vue 3 + TypeScript + Vite + Pinia + Arco Design Vue

项目目标是替换旧的 Node/Vue 版本，提供更稳定、易部署的单机 / 局域网游戏库管理体验。

## 主要功能

- 游戏列表、详情、编辑
- 文件条目管理与下载
- Wiki 编辑与历史记录
- 封面、横幅、截图上传与删除
- Steam 搜索、素材抓取、简介导入
- 元数据管理
  - 系列
  - 平台
  - 开发商
  - 发行商
- 目录浏览与根目录限制

## 项目结构

```text
.
├── backend/        # Go 后端
├── frontend/       # Vue 3 前端
├── docs/           # 重建过程文档
├── vnite/          # 参考项目
├── build-release.sh
└── start-dev.sh
```

## 开发启动

根目录直接执行：

```bash
bash start-dev.sh
```

脚本会：

- 检查 `go`、`npm`、`curl`
- 预热 Go 依赖
- 启动后端
- 等待健康检查
- 再启动前端开发服务器

默认地址：

- 前端开发：`http://127.0.0.1:5173`
- 后端接口：`http://127.0.0.1:3000`

## 后端配置

后端读取 `backend/.env`。

可参考：

- [backend/.env.example](/home/Hao/Game/backend/.env.example)

当前默认路径规范：

- 数据库：`data/db.db`
- 游戏素材：`data/gamelist`
- ROM 根目录：`ROM`

素材示例：

```text
data/gamelist/1/cover.jpg
data/gamelist/1/banner.jpg
data/gamelist/1/1.jpg
data/gamelist/1/2.jpg
```

## 生产打包

执行：

```bash
bash build-release.sh
```

或指定版本名：

```bash
bash build-release.sh v1.0.0
```

脚本会：

- 构建前端
- 将前端构建结果内嵌进 `game-server`
- 编译 Go 后端
- 生成一个可直接部署的发布目录

发布目录中只保留：

- `game-server`
- `.env`
- `data/gamelist`
- `ROM`

不再需要额外携带 `frontend/dist`。

## 部署建议

生产环境可直接运行发布目录中的：

```bash
./start.sh
```

或将 `game-server` 配成 `systemd` 服务。

## 说明

- 旧版重建参考代码不再作为正式主线
- 当前仓库以 Go 后端和 Vue 3 前端为准
- 如需发布，优先使用 `build-release.sh` 生成的目录
