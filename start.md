# Nova 项目快速入门

## 1. 项目是什么

**Nova** — 一个面向长篇网文创作者的 AI-native 创作工作台。它像 IDE 管理代码一样管理小说创作：提供文件树、Markdown 编辑器、多 Tab、全局搜索、章节统计、结构化资料库（角色/世界观/地点等）、互动故事试跑（剧情分支）、Agent 辅助写作、Skills 技能系统、版本管理（基于 git）和自动化工作流。支持导入 AI 酒馆角色卡和现有小说进行同人/改编。

> 当前版本 **v0.1.10** (Beta)，作者 alfredxw/nova

---

## 2. 技术栈

| 层 | 技术 |
|---|---|
| **后端语言** | Go 1.26+ |
| **HTTP 框架** | Hertz (cloudwego) |
| **AI Agent 框架** | Eino (cloudwego LLM编排库) |
| **前端** | React 19 + TypeScript 6 |
| **构建** | Vite 8 |
| **样式** | Tailwind CSS 4 |
| **富文本编辑器** | TipTap 3 |
| **状态管理** | TanStack Query + Zustand |
| **包管理** | pnpm |
| **编辑器** | Monaco Editor |
| **其他** | SSE (流式), go-git (版本管理), i18next (国际化) |

---

## 3. 入口文件

**后端入口**：`cmd/nova/main.go`

- 加载配置 → 初始化应用运行时 (`internal/app`) → 启动 HTTP 服务器 (`internal/api`)

**前端入口**：`web/index.html` → `web/src/main.tsx`

**前端 App 根组件**：`web/src/App.tsx`

---

## 4. 启动、构建和测试命令

| 操作 | 命令 |
|---|---|
| **启动全部（前后端）** | `./bootstrap.sh` |
| **仅前端** | `./bootstrap.sh fe` |
| **仅后端** | `./bootstrap.sh be` |
| **生产构建** | `./build.sh` |
| **前端测试** | `cd web && pnpm test` (vitest) / `pnpm test:watch` |
| **Go 测试** | `go test ./...` |
| **运行产物** | `cd output && ./nova --workspace /path/to/my-novel` |

默认地址：前端 `http://localhost:5173`，后端 `http://localhost:8080`

---

## 5. 如果要新增功能，从哪里开始

### 前端功能（UI/交互）

从 `web/src/` 开始。目录结构：

| 文件/目录 | 说明 |
|---|---|
| `main.tsx` | 应用挂载 |
| `App.tsx` | 根组件 |
| `lib/api.ts` | 后端 API 调用封装 |
| `stores/` | Zustand 状态管理 |
| `hooks/` | 自定义 hooks（如 `useChat.ts`, `useWorkspace.ts`） |
| `components/` | UI 组件 |

前端通过 Vite proxy 把 `/api/*` 请求转发到后端。

### 后端功能（API/业务逻辑）

从 `internal/` 开始。主要模块：

| 目录 | 说明 |
|---|---|
| `internal/api/handlers/` | HTTP 处理器（加路由 handler） |
| `internal/api/sse/` | SSE 流式推送 |
| `internal/app/` | 应用运行时（初始化、workspace 管理） |
| `internal/agent/` | AI Agent 编排 |
| `internal/book/` | 书籍文件操作和版本管理 |
| `internal/interactive/` | 互动故事模式 |
| `internal/skills/` | Skills 系统 |
| `internal/session/` | 会话管理 |
| `internal/automation/` | 自动化（定时任务等） |
| `internal/prompts/` | Prompt 模板 |

### 典型场景增改路线

1. **新增 API 路由** → 在 `internal/api/` 加 handler，在 `api.NewServer()` 中注册路由
2. **新增前端页面** → 在 `web/src/` 新建页面组件，在 `App.tsx` 加路由
3. **新增 Agent 技能** → 在 `skills/` 目录加提示词文件，或通过 UI 写入
4. **新增模型/配置项** → 在 `config/` 加配置结构体，在 `config.toml` 加字段

建议先阅读 `docs/project-structure.md` 和 `CONTEXT.md` 了解更详细的约定和术语。
