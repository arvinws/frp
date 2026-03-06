# Fork 自定义功能文档

本文档记录 [arvinws/frp](https://github.com/arvinws/frp) fork 仓库相对于上游 [fatedier/frp](https://github.com/fatedier/frp) 的所有自定义修改，供后续维护、升级和重构参考。

---

## 一、客户端治理功能

### 1.1 设计方案

采用「**双层治理**」架构：

| 层级 | 名称 | 标识依据 | 用途 |
|------|------|---------|------|
| C 方案（规则层） | 禁用 / 启用 | `clientID`（回退到 `runID`） | 长期治理：控制客户端是否允许登录、是否允许新建代理 |
| A 方案（会话层） | 断开连接 | `runID` | 即时运维：立即关闭当前在线会话 |
| 组合动作 | 禁用并断开 | `clientID` + `runID` | 先落禁用规则，再断开当前会话，防止客户端被踢后立即重连 |

**设计原则：**

- 所有业务规则收敛在新增 `pkg/ext/` 包中，核心代码仅保留少量 hook 点
- 不重写整套 Dashboard、不改造 frp 核心协议、不深耦合到所有连接流程
- 上游在 0.67.0 新增了 `clientID` 与 connected clients dashboard，治理方案以此为基础扩展

### 1.2 数据模型

```
ClientBanRecord:
  clientID    string     # 客户端标识（effectiveClientID：优先 clientID，为空回退 runID）
  status      string     # "enabled" | "disabled"
  reason      string     # 禁用原因
  operator    string     # 操作人
  updatedAt   time.Time  # 更新时间

OnlineSession:
  runID       string     # 当前运行 ID
  clientID    string     # 客户端标识（effectiveClientID）
  user        string     # 用户名
  connectedAt time.Time  # 连接时间
  remoteAddr  string     # 远程地址
  controller  SessionController  # 控制连接引用（用于断开）

AuditEvent:
  action          string  # disable / enable / disconnect / disable_and_disconnect
  targetClientID  string
  targetRunID     string
  operator        string
  result          string
  detail          string
  timestamp       time.Time
```

### 1.3 状态流

```
客户端发起连接
  └─ frps 登录处理 (service.go RegisterControl)
       ├─ 计算 effectiveClientID = clientID || runID
       ├─ 查询 banStore.IsDisabled(effectiveClientID)
       │    ├─ 已禁用 → 拒绝登录，返回错误
       │    └─ 未禁用 → 继续
       ├─ 创建 Control，注册到 ctlManager
       ├─ 注册到 clientRegistry
       ├─ 注册到 sessionManager（使用 effectiveClientID）
       └─ 启动 Control

客户端请求新建代理
  └─ control.go handleNewProxy
       ├─ 计算 effectiveID = clientID || runID
       ├─ 查询 banStore.IsDisabled(effectiveID)
       │    ├─ 已禁用 → 拒绝创建
       │    └─ 未禁用 → 继续正常流程
       └─ ...

管理员执行「禁用并断开」
  └─ adminapi DisableAndDisconnect
       ├─ banStore.Disable(clientID, reason, operator)    # 规则生效
       ├─ sessionManager.GetByClientID(clientID)          # 查找在线会话
       │    ├─ 找到 → DisconnectByRunID → controller.Close() → 断开连接
       │    └─ 未找到 → 返回 already_offline
       └─ 记录审计日志

客户端被断开后
  └─ frpc 内置重连机制（指数退避，最大 20s 间隔）
       ├─ 已禁用 → 登录被拒，持续重试失败
       └─ 未禁用 → 登录成功，恢复在线
```

### 1.4 effectiveClientID 机制

当 frpc 客户端未配置 `clientID`（包括旧版客户端和官方版本），系统使用 `runID` 作为回退标识：

```go
effectiveClientID := loginMsg.ClientID
if effectiveClientID == "" {
    effectiveClientID = loginMsg.RunID
}
```

此机制应用于：
- **登录禁用检查**（`server/service.go`）
- **新代理注册检查**（`server/control.go`）
- **会话注册**（`server/service.go` → `sessionManager.Register`）
- **Dashboard 显示**（`server/http/controller.go` → `ClientInfo.ClientID()` 回退逻辑）

**版本兼容性：**

| 客户端类型 | 禁用+断开 | 持久禁止重连 |
|-----------|----------|-------------|
| 配置了 `clientID` 的新版客户端 | 有效 | **永久有效**（clientID 跨重连不变） |
| 未配置 `clientID` 的新版客户端 | 有效 | 当次有效，重连后获得新 runID 绕过 |
| 旧版 / 官方 frpc | 有效 | 同上 |

> 注：即使客户端不发送 RunID，frps 也会自动生成一个（`service.go` 第 594 行）。

---

## 二、后端实现

### 2.1 新增包

| 包路径 | 职责 | 关键类型/函数 |
|--------|------|-------------|
| `pkg/ext/banlist/` | 客户端禁用/启用规则存储 | `Store` 接口、`MemoryStore`、`ClientBanRecord` |
| `pkg/ext/clientmgr/` | 在线会话索引与断开 | `Manager`、`OnlineSession`、`SessionController` |
| `pkg/ext/audit/` | 治理操作审计记录 | `Recorder` 接口、`MemoryRecorder`、`Event` |
| `pkg/ext/adminapi/` | 管理 API handler | `Handler`、四个接口处理函数 |

### 2.2 核心 Hook 点

改动控制在 4 个文件、少量代码行：

| 文件 | Hook 位置 | 用途 | 改动量 |
|------|----------|------|-------|
| `server/service.go` | `RegisterControl` 函数 | 登录禁用检查 + 会话注册（使用 effectiveClientID） | ~15 行 |
| `server/control.go` | `handleNewProxy` 函数 | 新代理注册禁用检查（使用 effectiveClientID） | ~12 行 |
| `server/control.go` | `worker` 退出清理 | 会话注销 `sessionManager.Unregister` | 3 行 |
| `server/api_router.go` | `registerRouteHandlers` | 治理 API 路由注册 | 7 行 |
| `server/service.go` | `Service` 结构体 | 新增 `clientBanStore`、`sessionManager`、`auditRecorder` 字段 | 5 行 |

### 2.3 管理 API

所有接口位于 Dashboard 认证保护下，复用 `webServer.authMiddleware`。

#### POST `/api/admin/clients/{clientID}/disable`

禁用指定客户端，后续登录与新代理注册一律拒绝。

- 请求体：`{ "reason": "string", "operator": "string" }`（均可选）
- 幂等：重复禁用返回 `already_disabled`
- 响应：`{ "clientID", "status", "changed", "reason", "operator", "updatedAt", "result" }`

#### POST `/api/admin/clients/{clientID}/enable`

解除禁用。

- 幂等：对未禁用客户端返回 `already_enabled`
- 响应结构同上

#### POST `/api/admin/sessions/{runID}/disconnect`

立即断开当前在线会话，不改变长期禁用状态。

- 目标不在线时返回 `already_offline`
- 响应：`{ "runID", "clientID", "result" }`

#### POST `/api/admin/clients/{clientID}/disable-and-disconnect`

组合动作：先落库禁用，再查找并断开当前在线会话。顺序固定不可颠倒。

- 响应：`{ "clientID", "status", "ruleChanged", "reason", "operator", "updatedAt", "runID", "disconnectResult" }`

### 2.4 ClientInfoResp 扩展

`server/http/model/types.go` 中 `ClientInfoResp` 新增 `disabled` 字段：

```go
type ClientInfoResp struct {
    // ... 原有字段 ...
    Online   bool `json:"online"`
    Disabled bool `json:"disabled"`  // 新增：客户端是否被禁用
}
```

在 `server/http/controller.go` 的 `buildClientInfoResp` 中查询 `banStore` 填充，Controller 新增 `banStore` 依赖。

---

## 三、前端实现

### 3.1 技术栈

Vue 3 + TypeScript + Element Plus + Vite，与原 Dashboard 一致。

### 3.2 新增/修改文件

| 文件 | 变更类型 | 内容 |
|------|---------|------|
| `web/frps/src/types/client.ts` | 修改 | `ClientInfoData` 新增 `disabled` 字段 + 治理 API 响应类型 |
| `web/frps/src/utils/client.ts` | 修改 | `Client` 类新增 `disabled` 属性 |
| `web/frps/src/api/admin.ts` | **新增** | 封装 4 个治理 API 调用函数 |
| `web/frps/src/i18n/index.ts` | 修改 | `governance` 命名空间中英文翻译 |
| `web/frps/src/components/ClientCard.vue` | 修改 | 禁用状态标签（红色 tag）+ 状态指示灯/徽章适配 |
| `web/frps/src/views/Clients.vue` | 修改 | 状态筛选新增「已禁用」tab |
| `web/frps/src/views/ClientDetail.vue` | 修改 | 治理操作按钮 + 操作对话框 + 禁用状态展示 |

### 3.3 交互设计

**客户端详情页操作按钮：**

| 客户端状态 | 显示的按钮 |
|-----------|-----------|
| 正常 + 在线 | 禁用、断开连接 |
| 正常 + 离线 | 禁用 |
| 已禁用 + 在线 | 启用、断开连接 |
| 已禁用 + 离线 | 启用 |

**禁用对话框智能提示：**

- 当客户端在线时，对话框自动显示「同时断开当前连接」开关（默认开启，带「推荐」标签）
- 开关开启 → 调用 `disableAndDisconnect` 接口
- 开关关闭 → 仅调用 `disable` 接口
- 附带提示："如果不断开，已有代理将继续工作，直到客户端自行断开"

**所有操作均通过对话框二次确认**，支持填写原因和操作人，操作完成后自动刷新状态。

### 3.4 客户端列表页

- 状态筛选 tab：全部 / 在线 / 离线 / **已禁用**
- `ClientCard` 卡片上显示禁用状态红色标签和红色指示灯

---

## 四、CI/CD 工作流

### 4.1 自动打包

`fork-auto-package.yml`：每次 push 到任何分支时运行，编译产物上传为 Actions artifacts。

### 4.2 自动发布

`fork-auto-release.yml`：推送 `v*` 标签时触发，自动创建 GitHub Release 并上传 `release/packages/*` 中的所有文件。

```bash
git tag -a v0.68.0-arvin.3 -m "release message"
git push origin --tags
```

---

## 五、版本历史

| 标签 | 内容 |
|------|------|
| `v0.67.0-arvin.1` | 初始 fork 发布 |
| `v0.68.0-arvin.1` | 后端治理功能 + CI 工作流 + i18n |
| `v0.68.0-arvin.2` | 前端治理 UI（禁用/启用/断开/禁用并断开）+ Dashboard 状态展示 |
| `v0.68.0-arvin.3` | 修复 clientID 为空时治理失效 + 禁用对话框 UX 优化 |

---

## 六、升级与维护策略

### 6.1 仓库 Remote 配置

```
origin    → https://github.com/arvinws/frp.git    (你的 fork)
upstream  → https://github.com/fatedier/frp.git    (上游原仓库)
```

### 6.2 同步上游流程

```bash
git fetch upstream
# 查看上游 release note，重点关注锚点文件的变化
git checkout dev
git rebase upstream/dev
# 解决冲突（如有），重点关注下方锚点
git push origin dev --force-with-lease
```

### 6.3 升级时必须检查的锚点

| 锚点文件 | 关注内容 |
|---------|---------|
| `server/service.go` | `RegisterControl` 函数的登录处理路径是否变化 |
| `server/control.go` | `handleNewProxy` / `worker` / `Close` 生命周期是否重构 |
| `server/api_router.go` | Dashboard 路由注册方式是否变化 |
| `server/http/controller.go` | `buildClientInfoResp` / `NewController` 签名是否变化 |
| `server/http/model/types.go` | `ClientInfoResp` 结构体字段是否变化 |
| `server/registry/` | `ClientInfo.ClientID()` 回退逻辑是否变化 |
| `web/frps/` | 前端路由、API 层、组件结构是否重构 |
| `pkg/msg/msg.go` | `Login` 消息结构体 `ClientID` / `RunID` 字段是否变化 |

### 6.4 升级后回归测试清单

| 场景 | 预期结果 | 备注 |
|------|---------|------|
| 正常客户端登录 | 成功 | 未禁用时不受影响 |
| 已禁用客户端登录 | 被拒绝 | 检查日志记录审计 |
| 已在线客户端新建代理 | 被拒绝 | 验证 `control.go` hook |
| 断开当前 runID | 连接断开 | 不改变禁用状态 |
| 禁用并断开 | 先禁用再断开 | 客户端无法立即重连 |
| 启用后重新登录 | 成功 | 规则恢复 |
| 未配置 clientID 的客户端 | 治理功能正常 | 使用 runID 回退 |
| 旧版/官方 frpc | 断开有效 | 持久禁用受限（runID 变化） |

---

## 七、风险与规避

| 风险 | 规避措施 |
|------|---------|
| 上游重构 dashboard / route 注册位置 | handler 保持独立，只在一个注册点接入 |
| runID 生命周期或 control 管理方式变化 | ext 层不长期持有内部对象，只做查询与调用 |
| clientID 使用不规范 | 建议制定 clientID 命名规范，在接入文档中固化 |
| 未配置 clientID 的客户端无法持久封禁 | 当前使用 runID 回退，可在后续版本增加 IP 封禁维度 |
| banlist 内存存储重启丢失 | 当前使用 `MemoryStore`，可扩展为文件/数据库持久化 |

---

## 八、目录结构速查

```
pkg/ext/
├── adminapi/
│   ├── handler.go          # 4 个治理 API handler
│   └── handler_test.go
├── banlist/
│   ├── store.go            # Store 接口 + MemoryStore
│   └── store_test.go
├── clientmgr/
│   ├── manager.go          # 在线会话管理器
│   └── manager_test.go
└── audit/
    └── recorder.go         # 审计记录器

server/
├── service.go              # Hook: 登录禁用检查 + 会话注册
├── control.go              # Hook: 新代理禁用检查 + 会话清理
├── api_router.go           # Hook: 治理 API 路由注册
└── http/
    ├── controller.go       # buildClientInfoResp 增加 disabled 查询
    └── model/types.go      # ClientInfoResp 增加 Disabled 字段

web/frps/src/
├── api/admin.ts            # 治理 API 前端封装
├── types/client.ts         # 治理相关类型定义
├── utils/client.ts         # Client 类增加 disabled 属性
├── i18n/index.ts           # governance 命名空间翻译
├── components/
│   └── ClientCard.vue      # 禁用状态标签
└── views/
    ├── Clients.vue          # 「已禁用」筛选 tab
    └── ClientDetail.vue     # 治理操作按钮 + 对话框
```
