<div align="center">

# ⚡ MiMo Free API

**将小米 MiMo 网页端反向代理为 OpenAI / Anthropic 兼容 API**

[中文](#中文) · [English](#english) · [特性](#特性) · [Docker 部署](#-docker-部署推荐) · [截图](#截图)

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)
![React](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=black)
![Docker](https://img.shields.io/badge/Docker-✓-2496ED?logo=docker&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green)

</div>

---

## 中文

### 这是什么？

MiMo Free API 是一个高性能反向代理网关，将小米 MiMo AI 网页端（aistudio.xiaomimimo.com）转换为标准 OpenAI 和 Anthropic 兼容 API。你可以用它在任何支持 OpenAI API 的客户端、Agent 框架中免费使用 MiMo 大模型。

> ⚠️ **自用声明**：本项目逆向 MiMo 网页端，仅建议个人自用。商用或对外提供服务可能违反小米服务条款并面临法律风险，请自行评估。账号也有被风控的可能。

### 特性

- 🔧 **工具调用（Tool Calling）** — OpenAI / Anthropic 双格式，自动注入工具提示、解析输出、长上下文裁剪
- 🔥 **OpenAI + Anthropic 双格式兼容** — `/v1/chat/completions` 和 `/v1/messages`（支持 `x-api-key` 头）
- 🧠 **深度思考支持** — 自动处理 MiMo 的 thinking 内容
- 🖼️ **多模态智能路由** — 检测图片/音频/文件自动选择模型
- 👥 **多账号轮转** — 配置多个账号自动轮转分担负载
- 📊 **实时统计仪表盘** — Token 用量、请求量、模型分布可视化
- 🔒 **安全加固** — 管理接口鉴权、Cookie 脱敏、并发上限、密钥常量时间比较、非 root 容器、数据文件 0600 权限
- 🌐 **中英双语 + 浅色/暗色主题**
- ⚡ **单二进制 / 单容器部署** — Go 编译，前端内嵌

### 截图

**仪表盘** — 实时统计 Token 用量、请求量、模型分布

![Dashboard](assets/dashboard.png)

**配置管理** — API 设置、模型切换、账号池管理（已加鉴权）

![Config](assets/config.png)

---

### 🐳 Docker 部署（推荐）

最简单的运行方式，无需本地安装 Go / Node 环境。

#### 前置条件

- 已安装 [Docker](https://docs.docker.com/get-docker/)（20.10+）
- 已安装 Docker Compose（v2 已内置于 Docker Desktop / 新版 Docker CLI）
- 已获取至少一个 MiMo 账号的 Cookie（[获取方法见下](#-获取-mimo-账号-cookie-详细攻略)）

#### 第一步：克隆仓库

```bash
git clone https://github.com/wtz44/mimo-free-api.git
cd mimo-free-api
git checkout docker-deploy   # Docker 相关改动在此分支
```

#### 第二步：设置 API Key（强烈建议修改默认值）

编辑 `docker-compose.yml`，把 `MIMO_API_KEY` 改成一个强随机值：

```yaml
environment:
  MIMO_API_KEY: "sk-改成你自己的强随机值"   # 例如 openssl rand -hex 24
```

> 也可以不改文件，改用环境变量启动：
> ```bash
> export MIMO_API_KEY="sk-你的强随机值"
> docker compose up -d
> ```

> 这个 Key 既是**客户端调用 API 的凭证**，也是**管理面板的解锁密钥**。务必保管好。

#### 第三步：一键启动

```bash
docker compose up -d --build
```

首次构建需要拉取镜像并编译前端/后端，约 3–6 分钟。启动后访问：

```
http://localhost:8080
```

#### 第四步：解锁管理面板并添加 MiMo 账号

1. 浏览器打开 `http://localhost:8080`，进入「配置」页
2. 输入上一步设置的 `MIMO_API_KEY`，点击「解锁」
3. 在「MiMo 账号池」点击「添加」，填入 Cookie 字段（`service_token` / `user_id` / `ph`），保存
4. 账号会自动持久化到 Docker 卷，**容器重启/重建都不会丢失**

> 也可以不用面板，直接编辑配置文件启动（见[备选方式](#备选直接挂载-configjson)）。

#### 第五步：验证调用

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-你的Key" \
  -d '{"model":"mimo-v2.5","messages":[{"role":"user","content":"你好"}],"stream":true}'
```

返回流式文本即部署成功。

---

#### 常用运维命令

```bash
# 查看实时日志
docker compose logs -f

# 查看容器状态（含 healthcheck）
docker compose ps

# 停止
docker compose down

# 重启
docker compose restart

# 更新代码后重新构建并启动
git pull
docker compose up -d --build

# 彻底删除（含数据卷，谨慎！会清空账号与统计）
docker compose down -v
```

#### 环境变量参考

在 `docker-compose.yml` 的 `environment` 中配置。环境变量优先级**高于**配置文件。

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `MIMO_API_KEY` | `sk-mimo` | 网关 API Key（客户端调用 + 管理面板解锁） |
| `MIMO_PORT` | `8080` | 容器内监听端口（一般无需改） |
| `MIMO_DEFAULT_MODEL` | `mimo-v2.5` | 默认模型：`mimo-v2.5` / `mimo-v2.5-pro` |
| `MAX_CONCURRENCY` | `8` | 全局并发上限，超限返回 503 |
| `CONFIG_PATH` | `/app/data/config.json` | 配置文件路径（建议保持在卷内） |
| `TZ` | `Asia/Shanghai` | 时区 |

#### 自定义宿主机端口

想用别的端口（比如 9000）暴露，改 `docker-compose.yml` 的 `ports`：

```yaml
ports:
  - "9000:8080"   # 宿主机 9000 -> 容器 8080
```

然后访问 `http://localhost:9000`。

#### 数据持久化说明

- 数据统一存放在命名卷 `mimo-data`，挂载到容器 `/app/data`
- 包含：`config.json`（账号/Key/模型）、`stats.json`（用量统计）、`conversations.json`（对话记录）
- 容器删除重建（`docker compose up -d --build`）数据不丢；只有 `docker compose down -v` 才会清空
- 查看卷内容：`docker volume inspect mimo-free-api_mimo-data`

#### 备选：直接挂载 config.json

若你更习惯用配置文件管理账号（而非面板），可跳过环境变量，直接挂载文件：

```bash
# 1. 准备配置文件
cp config.example.json config.json
# 编辑 config.json：填 api_key 和 accounts

# 2. 在 docker-compose.yml 的 volumes 加一行（注意先创建文件，否则会被当成目录）
volumes:
  - ./config.json:/app/config.json
  - mimo-data:/app/data
# 并把 environment 里的 CONFIG_PATH 改回 /app/config.json
```

#### 不用 Compose，直接 docker run

```bash
# 构建镜像
docker build -t mimo-gateway .

# 运行
docker run -d --name mimo-gateway \
  -p 8080:8080 \
  -e MIMO_API_KEY="sk-你的强随机值" \
  -v mimo-data:/app/data \
  --restart unless-stopped \
  mimo-gateway
```

---

### 📦 从 Release 下载预构建镜像（无需本地构建）

每次推送 `v*` 标签（如 `v1.0.0`）时，GitHub Actions 会自动构建 **amd64** 与 **arm64** 两种架构的 Docker 镜像，并打包成 `.tar.gz` 上传到 [Releases 页面](https://github.com/toong1995/mimo-free-api/releases)。这样无需本地安装 Go/Node，也无需 `docker build`，直接导入即可运行。

#### 使用方法

```bash
# 1. 从 Release 下载对应架构的压缩包（以 v1.0.0 amd64 为例）
#    也可在 Releases 页面手动点击下载
wget https://github.com/toong1995/mimo-free-api/releases/download/v1.0.0/mimo-gateway-v1.0.0-amd64.tar.gz

# 2. 解压 + 导入镜像
gunzip mimo-gateway-v1.0.0-amd64.tar.gz
docker load -i mimo-gateway-v1.0.0-amd64.tar
# 导入成功后会提示：Loaded image: mimo-gateway:amd64

# 3. 运行
docker run -d --name mimo-gateway \
  -p 8080:8080 \
  -e MIMO_API_KEY="sk-改成强随机值" \
  -v mimo-data:/app/data \
  --restart unless-stopped \
  mimo-gateway:amd64
```

> arm64 机器（Apple Silicon / 树莓派 / ARM 服务器）把上面的 `amd64` 换成 `arm64` 即可。

#### 配合 docker-compose.yml 使用预构建镜像

把 `docker-compose.yml` 里的 `build: .` 删掉，改用导入的镜像：

```yaml
services:
  mimo-gateway:
    image: mimo-gateway:amd64   # 或 arm64
    container_name: mimo-gateway
    # ... 其余 environment / volumes / ports 保持不变
```

然后 `docker compose up -d` 即可。

#### CI 触发方式（仓库维护者）

| 方式 | 操作 | 结果 |
|------|------|------|
| 自动 | `git tag v1.0.0 && git push origin v1.0.0` | 构建 amd64+arm64，发布正式 Release |
| 手动 | 在 GitHub 仓库 → Actions → 「Build and Release Docker Image」→ Run workflow | 发布 prerelease（便于测试） |

> 手动触发不依赖 tag，会用短 SHA 作为版本号，并标记为 prerelease，方便验证流程。

---

### 🍪 获取 MiMo 账号 Cookie 详细攻略

#### 第一步：登录 MiMo AI Studio

1. 打开浏览器，访问 **https://aistudio.xiaomimimo.com**
2. 使用小米账号登录
3. 登录成功后在对话框随便发一条消息，确认能正常对话

#### 第二步：打开开发者工具

1. 按 **F12**（或 `Ctrl+Shift+I` / `Cmd+Option+I`）打开开发者工具
2. 切换到 **Network（网络）** 面板
3. 勾选左上角的 **Preserve log（保留日志）**

#### 第三步：触发一个请求

1. 在 MiMo 对话框中发送任意消息（如 "你好"）
2. 在 Network 面板找到 `chat` 开头的请求（通常是 `chat?appId=...`）
3. 点击该请求

#### 第四步：复制 Cookie 信息

在 **Request Headers** 中找到 `cookie:` 字段，从中提取三个值：

| 你需要的字段 | Cookie 中的 key | 示例格式 |
|---|---|---|
| `service_token` | `serviceToken` | `/Qzv9hyEQZi...`（很长的 base64） |
| `user_id` | `userId` | `3215624450`（纯数字） |
| `ph` | `xiaomichatbot_ph` | `0Fjou2NP2l54M8SRzvNO/g==`（base64） |

> 推荐：右键 `chat` 请求 → **Copy** → **Copy as cURL**，从 cURL 的 `--cookie` 中提取更省事。

#### 第五步：填入

通过管理面板「添加账号」表单填入，或写入 `config.json` 的 `accounts` 数组：

```json
{
  "accounts": [
    {
      "id": "account-1",
      "service_token": "serviceToken 完整值",
      "user_id": "userId",
      "ph": "xiaomichatbot_ph",
      "active": true
    }
  ]
}
```

> ⚠️ Cookie 有有效期，过期需重新获取；**不要**分享给他人；`ph` 含 `=` 等特殊字符原样保留。

---

### 使用方法

#### OpenAI 格式

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-你的Key" \
  -d '{
    "model": "mimo-v2.5",
    "messages": [{"role": "user", "content": "你好"}],
    "stream": true
  }'
```

#### Anthropic 格式

```bash
curl http://localhost:8080/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: sk-你的Key" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "mimo-v2.5",
    "messages": [{"role": "user", "content": "你好"}],
    "max_tokens": 4096
  }'
```

#### 多轮对话隔离（可选）

默认按「首条消息」复用 MiMo 会话。若不同对话首条消息相同（如都是「你好」），建议通过请求头 `X-Conv-Key` 显式区分，避免上下文串扰：

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-你的Key" \
  -H "X-Conv-Key: my-session-001" \
  -H "Content-Type: application/json" \
  -d '{"model":"mimo-v2.5","messages":[{"role":"user","content":"记住数字 777"}]}'
```

#### 第三方客户端接入

| 客户端 | Base URL | API Key | Model |
|--------|----------|---------|-------|
| ChatGPT-Next-Web / LobeChat / Open WebUI | `http://localhost:8080/v1` | 你的 Key | `mimo-v2.5` |
| ChatBox / TypingMind | `http://localhost:8080` | 你的 Key | `mimo-v2.5` |
| Claude Desktop (Anthropic) | `http://localhost:8080` | 你的 Key | `mimo-v2.5` |

#### 模型说明

| 模型 | 能力 | 适用场景 |
|------|------|---------|
| `mimo-v2.5` | 图片、音频、文件、文本 | 全模态通用 |
| `mimo-v2.5-pro` | 图片、文本（推理增强） | 深度推理任务 |

---

### 源码编译运行（备选）

无需 Docker，本地直接编译：

```bash
# 前端
cd web && npm install && npm run build && cd ..

# 后端（需 Go 1.22+）
go build -o mimo-free-api .

# 运行（首次自动生成 config.json）
./mimo-free-api
```

---

### 安全说明

本分支已做如下加固（仍建议自用、不暴露公网）：

- `/admin/api/*` 与 `/v1/*` 均需 API Key 鉴权
- 管理接口返回的 `service_token` / `ph` 已脱敏
- API Key 比较使用常量时间算法，防时序攻击
- 全局并发上限，防资源耗尽
- 配置/统计数据文件权限 `0600`
- Docker 容器以非 root 用户运行
- 日志不再打印用户 prompt 与原始输出明文

如需暴露公网，请额外配置反向代理 + HTTPS + 强随机 API Key。

---

### 常见问题

**Q: 启动后管理面板要输入什么？**
A: 输入你设置的 `MIMO_API_KEY`（默认 `sk-mimo`，强烈建议改）。

**Q: 添加的账号重启后丢失？**
A: 不会。账号保存在 `mimo-data` 卷的 `config.json` 里。只有 `docker compose down -v` 才会清空。

**Q: Cookie 多久过期？**
A: 数天到数周不等。请求返回认证错误时需重新获取。

**Q: healthcheck 一直 unhealthy？**
A: 首次构建启动较慢，等待 `start_period`（10s）后观察。`docker compose logs` 看是否启动成功。

**Q: 想换端口怎么办？**
A: 改 `docker-compose.yml` 的 `ports`（如 `"9000:8080"`），无需改容器内端口。

### License

MIT

---

## English

### What is this?

A reverse-proxy gateway that exposes Xiaomi MiMo's web UI as OpenAI / Anthropic compatible APIs. Self-use only — reverse-engineering the web UI may violate Xiaomi's ToS.

### Features

OpenAI + Anthropic compatible, tool calling, multi-modal routing, multi-account rotation, real-time dashboard, security hardening (admin auth, secret masking, concurrency limit, constant-time key compare, non-root container, 0600 data files), i18n + themes, single binary/container.

### Docker Quick Start (recommended)

```bash
git clone https://github.com/wtz44/mimo-free-api.git
cd mimo-free-api
git checkout docker-deploy

# (Optional) set a strong API key instead of the default sk-mimo
export MIMO_API_KEY="sk-$(openssl rand -hex 24)"

# Build & run
docker compose up -d --build
```

Open `http://localhost:8080`, enter your `MIMO_API_KEY` to unlock the admin panel, then add your MiMo account cookies.

Verify:

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $MIMO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"mimo-v2.5","messages":[{"role":"user","content":"Hello"}],"stream":true}'
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MIMO_API_KEY` | `sk-mimo` | Gateway API key (clients + admin unlock) |
| `MIMO_PORT` | `8080` | In-container port |
| `MIMO_DEFAULT_MODEL` | `mimo-v2.5` | Default model |
| `MAX_CONCURRENCY` | `8` | Global concurrency limit |
| `CONFIG_PATH` | `/app/data/config.json` | Config file path |
| `TZ` | `Asia/Shanghai` | Timezone |

### Operations

```bash
docker compose logs -f          # logs
docker compose restart          # restart
docker compose down             # stop (keeps data)
docker compose down -v          # stop + wipe data
docker compose up -d --build    # rebuild after update
```

Data persists in the named volume `mimo-data` (config / stats / conversations).

### Without Compose

```bash
docker build -t mimo-gateway .
docker run -d --name mimo-gateway -p 8080:8080 \
  -e MIMO_API_KEY="sk-your-key" \
  -v mimo-data:/app/data --restart unless-stopped mimo-gateway
```

### Build from source (alternative)

```bash
cd web && npm install && npm run build && cd ..
go build -o mimo-free-api .
./mimo-free-api
```

### License

MIT
