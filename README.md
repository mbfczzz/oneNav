# 自由墨客 / ZMark — 复刻版(Vue 3 + Go + MySQL)

对 [`nav.rss.ink`](https://nav.rss.ink/)(品牌「自由墨客」,产品名 **ZMark**)的忠实复刻:相同的前端框架与界面结构,配套一个 **Go + MySQL** 的真实后端与**后台管理界面**。

- **前端**:Vue 3 + Vite + Tailwind + vue-router + Pinia + vue-i18n + vue-draggable-plus + Iconify(离线打包)。
- **后端**:Go(stdlib `net/http` + `go-sql-driver/mysql`),实现 `/api` 契约,连接 MySQL。
- **可独立运行**:不配后端时,前端用浏览器 localStorage 的 Mock 适配器,开箱即用。

---

## 两种运行模式

前端通过环境变量 `VITE_API_BASE` 在两种模式间切换(见 `.env.example`):

### 模式 A — 独立 Mock(零依赖,默认)
不设置 `VITE_API_BASE`(删除 `.env.local` 即可),数据存于浏览器 localStorage。

```bash
npm install
npm run dev          # http://localhost:5173
```

### 模式 B — 真实后端(Go + MySQL)
1) 启动后端(会自动在你的 MySQL 里创建 `onenav_*` 表并灌入种子数据):

```bash
npm run server       # 等价于 cd backend-go && go run .
# 连接信息从 backend-go/.env 读取(已为你预填你给的 MySQL),也可用环境变量覆盖(见下表)
```

2) 让前端走后端(`.env.local` 写入 `VITE_API_BASE=/api`,Vite 会把 `/api` 代理到 `:8787`):

```bash
echo "VITE_API_BASE=/api" > .env.local
npm run dev          # http://localhost:5173,API 走真实后端
```

3) 进入后台:点右上角 → 登录,默认 **admin / admin123**。

> 生产构建见下方「生产部署」一节(含 SPA history 回退与反代配置)。

---

## 后端配置

配置从环境变量读取;本地开发可放在 **`backend-go/.env`**(已 gitignore,启动时加载,真实环境变量优先)。仓库提供 `backend-go/.env.example`;**`backend-go/.env` 已为你预填你给的 MySQL 连接**。`DB_PASSWORD` **必填** —— 缺失时后端拒绝启动(源码不再内置任何密码/主机)。

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `DB_HOST` | `127.0.0.1` | MySQL 主机(你的 `.env` 已设为 `106.14.165.141`) |
| `DB_PORT` | `3306` | 端口 |
| `DB_NAME` | `zmark` | 库名(不存在会自动 `CREATE DATABASE`;你的 `.env` 设为 `beadforge`) |
| `DB_USERNAME` | `root` | 用户名 |
| `DB_PASSWORD` | (必填,无默认) | 密码,放 `backend-go/.env` 或用环境变量 |
| `TABLE_PREFIX` | `onenav_` | 表前缀,避免与库中已有表冲突 |
| `PORT` | `8787` | 后端监听端口 |
| `ADMIN_USER` / `ADMIN_PASS` | `admin` / `admin123` | 初始管理员(仅在 users 表为空时创建;用默认密码会打印启动告警) |
| `AUTH_TTL_HOURS` | `168` | 登录态有效期(小时,默认 7 天) |

PowerShell 覆盖示例:
```powershell
$env:DB_PASSWORD="真实密码"; npm run server
```

### 数据库表(均为 `CREATE TABLE IF NOT EXISTS`,只新增、不动你已有数据)
- `onenav_categories(id, name, icon, weight, create_time)`
- `onenav_links(id, category_id, title, url, description, icon, clicks, weight, create_time)`
- `onenav_users(id, username, password_hash, create_time)` — 密码 bcrypt 存储

---

## API 契约(前后端共用)

| 方法 & 路径 | 鉴权 | 说明 |
| --- | --- | --- |
| `GET /api/all?scope=global\|mine` | global 公开 / mine 需登录 | 返回该范围的 `{categories, links}` |
| `POST /api/register` | 公开 | `{username,password,code}` → `{token,user}`;**需有效邀请码** |
| `GET/POST /api/invites` · `PUT/DELETE /api/invites/:id` · `GET /api/invites/:id/uses` | 管理员 | 邀请码 列出/创建/停用启用/删除 + 注册记录 |
| `POST /api/login` | 公开 | `{username,password}` → `{token, user{role}}` |
| `POST /api/logout` / `GET /api/me` | Bearer | 登出 / 当前用户(含 role) |
| `POST /api/categories\|links?scope=global\|mine` | Bearer | 新建到指定范围;**全局仅管理员** |
| `POST/PUT/DELETE /api/categories[/:id]` | Bearer | 分类增改删 |
| `PUT /api/categories/order` | Bearer | `{orderedIds}` 重排序 |
| `POST/PUT/DELETE /api/links[/:id]` | Bearer | 链接增改删 |
| `PUT /api/links/order` | Bearer | 链接重排序 |
| `POST /api/links/:id/click` | 公开 | 点击 +1,返回 `{clicks}` |
| `POST /api/import` | Bearer | 批量导入(容错解析,单次上限 2000 条) |
| `POST /api/reset` | Bearer | 清空并重灌种子(演示用) |

鉴权:登录返回 bearer token(后端内存 + 7 天 TTL),前端存在 `localStorage(zmark.token)`,变更类接口需带 `Authorization: Bearer <token>`。

### 多用户 · 全局与个人导航
- **全局(通用)导航**:`owner_id='global'`,所有人(含未登录)默认看到;**仅管理员**可在后台配置。
- **个人导航**:每个注册用户有自己的分类/链接(`owner_id=<用户id>`),仅本人可见可改。
- **角色**:`admin`(初始管理员,配置全局)/ `user`(注册用户,管理自己的)。
- **默认视图**:未登录 → 通用;登录的普通用户 → 有自己的就看「我的」,否则看通用;顶栏可在「通用 / 我的」间切换。
- **自助注册(邀请制)**:`/user` 页在登录 / 注册间切换;**注册需填写有效邀请码**,注册即登录。
- 数据隔离与权限由**后端强制**:用户无法读取或修改他人 / 全局数据,越权返回 403。

### 邀请码
- **注册需邀请码**:管理员在后台「邀请码」生成并分发;系统**不预置任何邀请码**,无有效码无法注册。
- 每个码可设:**备注、可用次数(0 = 不限)、有效期(永久 / 7 / 30 / 90 天)、注册后角色(普通用户 / 管理员)**;注册时**原子消费**(行锁,单次码不会被并发超用),用户记录其所用的码并按该码授予角色。
- 状态自动判定:**有效 / 已用尽 / 已过期 / 已停用**;管理员可**停用 / 启用、删除**。表:`onenav_invites`。
- **注册记录**:后台每个邀请码可展开查看使用者(用户名 + 注册时间)。
- **用户管理**:可删除任意账号(含被「管理员邀请码」授予管理员的账号),仅保护**主管理员(`ADMIN_USER`)**与当前登录者。

---

## 已实现功能

**前台(/)**:三段式布局(顶栏搜索 + 左侧分类栏 + 1→5 列响应式卡片网格)、加载壳、分类/链接拖拽排序、点击计数、复制链接、二维码、搜索、移动端抽屉、中英双语、返回顶部;**登录后顶栏可切换「通用 / 我的」**,未登录只看通用,「我的」为空时引导去后台添加。

**后台(/dashboard,需登录)**:
- 分类:新建 / 编辑 / 删除(级联删链接)/ 拖拽排序
- 链接:新建 / 编辑 / 删除 / 拖拽排序(按分类管理)
- **批量导入**:粘贴或上传 ZMark/OneNav 风格 JSON,容错解析,同名分类合并,单次上限 2000
- 统计卡(分类 / 链接 / 总点击)、登录态、API 模式标识(MySQL / Mock)
- 路由守卫:未登录访问 `/dashboard` 自动跳登录页
- **多用户**:普通用户的后台只管理「我的」;管理员可在后台用「通用 / 我的」开关切换,配置全局导航

**导入 JSON 示例**:
```json
{
  "categories": [{ "name": "我的收藏", "icon": "ri:star-line" }],
  "links": [{ "category": "我的收藏", "title": "GitHub", "url": "https://github.com" }]
}
```

---

## 目录结构

```
index.html                 首屏加载壳 + meta(lang=zh、keywords)
backend-go/                Go + MySQL 后端
  main.go                  net/http 服务、迁移、种子、/api 实现
  seed.json                种子数据(由 scripts/gen-seed.mjs 生成)
  go.mod
scripts/
  gen-icons.mjs            扫描 src 生成离线图标子集 ri-icons.json
  gen-seed.mjs             由 src/data/seed.js 生成 backend-go/seed.json
src/
  api/                     index(适配器切换) mock(localStorage) http(fetch) token
  stores/nav.js            Pinia:分类/链接/筛选/排序/CRUD/导入/鉴权
  router/index.js          路由 + 登录守卫
  i18n/                    vue-i18n + zh/en
  components/              TopBar Sidebar LinkGrid LinkCard LinkFavicon QrDialog ToastHost
    ui/Modal.vue           通用弹窗(Teleport)
    dashboard/             CategoryFormModal LinkFormModal ImportModal
  views/                   HomeView DashboardView UserView NotFoundView
  ri-icons.json            离线图标子集(npm run gen:icons 重新生成)
```

## 常用命令

```bash
npm run dev          # 前端开发服务器(:5173)
npm run build        # 生产构建 -> dist/(/static/assets 哈希分包)
npm run preview      # 预览生产构建(:4173)
npm run server       # 启动 Go + MySQL 后端(:8787)
npm run gen:icons    # 重新生成离线图标子集(新增 ri:* 图标后)
npm run gen:seed     # 重新生成后端种子数据
```

## 与原站 / 你的技术栈的关系
- 原站 `nav.rss.ink` 真实后端不公开;本仓库后端按你提供的 MySQL 自建,数据契约与界面对齐。
- 你给的是 Spring datasource 配置,但按你的选择后端用 **Go** 实现;表名加 `onenav_` 前缀放在 `beadforge` 库中,纯增量,不影响该库已有业务表。
- 与开源 PHP 项目 `helloxz/onenav` 无代码关系(原站也仅是导入格式兼容 + SEO 关键词层面的关系)。

## 生产部署
前端是 **HTML5 history 模式 SPA**,生产托管需满足两点:
1. **SPA 回退**:所有未匹配路径回退到 `index.html`,否则刷新 `/dashboard` 会 404。
2. **反代 `/api`** 到 Go 后端(Vite 的 `/api` 代理只在开发时生效);并建议在最外层做 **TLS 终止**(后端是明文 HTTP,token/密码不应裸跑公网)。

Nginx 示例:
```nginx
server {
  listen 443 ssl;
  root /var/www/zmark/dist;            # npm run build 产物
  location /api/ { proxy_pass http://127.0.0.1:8787; }
  location / { try_files $uri $uri/ /index.html; }   # SPA 回退
}
```
注:`VITE_API_BASE` 在 **构建时** 注入;改后端地址需重新 `npm run build`。Netlify/Vercel 等静态平台用各自的 rewrite(`/* -> /index.html`)实现回退。

## 与原站 / 你的技术栈的关系
- 你给的是 Spring datasource,但按你的选择后端用 **Go** 实现;表名加 `onenav_` 前缀放在你的库中,纯增量。

## 已知限制 / 后续可加固
- **后台界面为中文(zh)**:公开前台支持中英双语,后台管理区当前为中文硬编码;如需双语后台可后续接入 vue-i18n。
- **隐私**:二维码用 qrserver、favicon 用 Google favicon 服务(会把域名发给第三方);更注重隐私可改为后端代理 favicon。图标已离线打包,不依赖网络。
- **鉴权**:token 存后端内存 + 7 天 TTL;多实例/重启不掉线需换 JWT/Redis。暂无改密接口与登录限流(bcrypt 已减缓暴力破解);默认 `admin123` 会打印告警,生产请改 `ADMIN_PASS`。
- `/api/links/:id/click` 为公开接口(贴合"公开访问计点击"的产品语义),无限流。
- 像素级配色按原站可观察到的设计 token 还原;如有真实截图可进一步 1:1 微调。
