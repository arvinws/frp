# Fork 自定义功能文档

本文档记录 [arvinws/frp](https://github.com/arvinws/frp) fork 仓库相对于上游 [fatedier/frp](https://github.com/fatedier/frp) 的所有自定义修改，供后续维护、升级和重构参考。

---

## 一、治理功能概述

本 fork 提供**两层治理粒度**，可独立或组合使用：

| 粒度 | 操作 | 效果 | 适用场景 |
|------|------|------|---------|
| **代理级别** | 禁用/启用单条代理 | 关闭指定隧道，其他代理不受影响 | 只想关闭某一条隧道 |
| **客户端级别** | 禁用/启用整个客户端 | 断开连接、阻止登录、所有代理失效 | 整台机器下线 |

附加能力：
- **IP 封禁**（可选）：对未配置 `clientID` 的客户端，可手动勾选同时封禁 IP
- **审计日志**：所有治理操作自动记录

---

## 二、代理级别治理

### 2.1 工作流程

```
管理员在 Dashboard 代理卡片上点击「禁用代理」
  └─ POST /api/admin/proxies/{name}/disable
       ├─ banStore.DisableProxy(name)      # 加入代理禁用名单
       ├─ ctlManager.CloseProxyByName(name) # 关闭当前在线代理
       └─ 记录审计日志

客户端重连/重载后尝试注册该代理
  └─ control.go handleNewProxy
       ├─ banStore.IsProxyDisabled(proxyName)
       │    ├─ 已禁用 → 拒绝注册，返回错误
       │    └─ 未禁用 → 正常注册
       └─ 其他代理不受影响

管理员启用该代理
  └─ POST /api/admin/proxies/{name}/enable
       └─ banStore.EnableProxy(name)       # 从禁用名单移除
       # 客户端下次重连/重载后可恢复注册
```

### 2.2 管理 API

#### POST `/api/admin/proxies/{name}/disable`

关闭指定在线代理并加入禁用名单，阻止重新注册。

- 请求体：`{ "reason": "string", "operator": "string" }`（均可选）
- 响应：`{ "proxyName", "result" }`（result: `disabled_and_closed` 或 `disabled`）

#### POST `/api/admin/proxies/{name}/enable`

从禁用名单移除，客户端重连/重载后恢复注册。

- 响应：`{ "proxyName", "result": "enabled" }`

### 2.3 前端交互

- 代理列表页和客户端详情页的 ProxyCard 上，在线代理显示「禁用代理」按钮
- 点击后弹出确认框，确认即关闭 + 禁用
- 操作完成后自动刷新代理列表

---

## 三、客户端级别治理

### 3.1 设计方案

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

### 3.2 effectiveClientID 机制

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

### 3.3 IP 封禁（可选）

当管理员禁用客户端时，可手动勾选「同时封禁该 IP」开关（默认关闭）：

- 开启后，该客户端的 IP 地址也被加入封禁名单
- 登录时先检查 clientID 禁用，未命中再检查 IP 禁用
- 启用客户端时自动清除关联的 IP 封禁
- **注意**：同一 IP 下的其他客户端也会受影响，前端有明确警告

适用场景：未配置 `clientID` 的客户端，`runID` 每次重连都变，IP 封禁是唯一的持久封禁手段。

### 3.4 版本兼容性

| 客户端类型 | 禁用+断开 | 持久禁止重连 |
|-----------|----------|-------------|
| 配置了 `clientID` 的新版客户端 | 有效 | **永久有效**（clientID 跨重连不变） |
| 未配置 `clientID` 的新版客户端 | 有效 | 需配合 IP 封禁 |
| 旧版 / 官方 frpc | 有效 | 同上 |

> **建议**：所有客户端都配置 `clientID`，这是最干净的治理方式。配置方法：在 frpc.toml 顶部加 `clientID = "your-unique-name"`。

### 3.5 状态流

```
客户端发起连接
  └─ frps 登录处理 (service.go RegisterControl)
       ├─ 计算 effectiveClientID = clientID || runID
       ├─ 查询 banStore.IsDisabled(effectiveClientID)
       │    ├─ 未命中 → 查询 banStore.IsIPDisabled(remoteIP)
       │    └─ 任一命中 → 拒绝登录
       ├─ 创建 Control，注册到 ctlManager
       ├─ 注册到 clientRegistry
       ├─ 注册到 sessionManager（使用 effectiveClientID）
       └─ 启动 Control

客户端请求新建代理
  └─ control.go handleNewProxy
       ├─ 检查客户端级别禁用（clientID/IP）
       ├─ 检查代理级别禁用（proxyName）
       │    ├─ 任一命中 → 拒绝创建
       │    └─ 均未命中 → 正常注册
       └─ ...

管理员执行「禁用客户端」（在线时默认同时断开）
  └─ adminapi DisableAndDisconnect
       ├─ banStore.Disable(clientID, reason, operator)
       ├─ 可选：banStore.DisableIP(remoteAddr, ...)
       ├─ sessionManager.GetByClientID → DisconnectByRunID
       └─ 记录审计日志

客户端被断开后
  └─ frpc 内置重连机制（指数退避，最大 20s 间隔）
       ├─ 已禁用 → 登录被拒，持续重试失败
       └─ 未禁用 → 登录成功，恢复在线
```

### 3.6 管理 API

#### POST `/api/admin/clients/{clientID}/disable`

禁用指定客户端，后续登录与新代理注册一律拒绝。

- 请求体：`{ "reason": "string", "operator": "string", "banIP": false }`
- `banIP: true` 时同步封禁客户端 IP
- 幂等：重复禁用返回 `already_disabled`
- 响应：`{ "clientID", "status", "changed", "reason", "operator", "updatedAt", "result" }`

#### POST `/api/admin/clients/{clientID}/enable`

解除禁用，同时清除关联的 IP 封禁。

- 幂等：对未禁用客户端返回 `already_enabled`
- 响应结构同上

#### POST `/api/admin/sessions/{runID}/disconnect`

立即断开当前在线会话，不改变长期禁用状态。

- 目标不在线时返回 `already_offline`
- 响应：`{ "runID", "clientID", "result" }`

#### POST `/api/admin/clients/{clientID}/disable-and-disconnect`

组合动作：先落库禁用，再查找并断开当前在线会话。顺序固定不可颠倒。

- 同样支持 `banIP` 参数
- 响应：`{ "clientID", "status", "ruleChanged", "reason", "operator", "updatedAt", "runID", "disconnectResult" }`

### 3.7 前端交互

**客户端详情页操作按钮：**

| 客户端状态 | 显示的按钮 |
|-----------|-----------|
| 正常 + 在线 | 禁用、断开连接 |
| 正常 + 离线 | 禁用 |
| 已禁用 + 在线 | 启用、断开连接 |
| 已禁用 + 离线 | 启用 |

**禁用对话框（客户端在线时）：**

- 「同时断开当前连接」开关 — 默认 **开启**（推荐）
- 「同时封禁该 IP」开关 — 默认 **关闭**，附带警告
- 支持填写原因和操作人
- 操作完成后自动刷新；如遇 404（客户端条目变更）自动跳转回列表

**客户端列表页：**

- 状态筛选 tab：全部 / 在线 / 离线 / **已禁用**
- `ClientCard` 卡片显示禁用状态红色标签和红色指示灯

---

## 四、数据模型

```
ClientBanRecord:
  clientID    string     # effectiveClientID
  status      string     # "enabled" | "disabled"
  reason      string     # 禁用原因
  operator    string     # 操作人
  updatedAt   time.Time

OnlineSession:
  runID       string
  clientID    string     # effectiveClientID
  user        string
  connectedAt time.Time
  remoteAddr  string
  controller  SessionController  # 用于断开

AuditEvent:
  action          string  # disable / enable / disconnect / disable_and_disconnect /
                          #   disable_proxy / enable_proxy
  targetClientID  string
  targetRunID     string
  operator        string
  result          string
  detail          string
  timestamp       time.Time
```

---

## 五、后端实现

### 5.1 新增包

| 包路径 | 职责 | 关键类型/函数 |
|--------|------|-------------|
| `pkg/ext/banlist/` | 禁用规则存储（客户端 + IP + 代理） | `Store` 接口、`MemoryStore` |
| `pkg/ext/clientmgr/` | 在线会话索引与断开 | `Manager`、`OnlineSession`、`SessionController` |
| `pkg/ext/audit/` | 治理操作审计记录 | `Recorder` 接口、`MemoryRecorder`、`Event` |
| `pkg/ext/adminapi/` | 管理 API handler | `Handler`、`ProxyCloser` 接口 |

### 5.2 banlist Store 接口

```go
type Store interface {
    // 客户端级别
    Disable(clientID, reason, operator string) (ClientBanRecord, bool)
    Enable(clientID, operator string) (ClientBanRecord, bool)
    IsDisabled(clientID string) (bool, ClientBanRecord)

    // IP 级别
    DisableIP(ip, clientID, reason, operator string)
    EnableByClientID(clientID string)  // 清除关联的 IP 封禁
    IsIPDisabled(ip string) (bool, ClientBanRecord)

    // 代理级别
    DisableProxy(proxyName, reason, operator string)
    EnableProxy(proxyName string)
    IsProxyDisabled(proxyName string) bool
}
```

### 5.3 核心 Hook 点

| 文件 | Hook 位置 | 用途 |
|------|----------|------|
| `server/service.go` | `RegisterControl` | 登录禁用检查（clientID + IP）+ 会话注册 |
| `server/control.go` | `handleNewProxy` | 客户端禁用检查 + 代理名称禁用检查 |
| `server/control.go` | `ControlManager.CloseProxyByName` | 按名称查找并关闭在线代理 |
| `server/control.go` | `worker` 退出 | 会话注销 |
| `server/api_router.go` | `registerRouteHandlers` | 全部治理 API 路由注册 |
| `server/http/controller.go` | `buildClientInfoResp` | 客户端禁用状态查询 |
| `server/registry/registry.go` | `MarkOfflineByRunID` | 离线时保留条目（不删除） |

### 5.4 ClientInfoResp 扩展

```go
type ClientInfoResp struct {
    // ... 原有字段 ...
    Online   bool `json:"online"`
    Disabled bool `json:"disabled"`
}
```

---

## 六、前端实现

### 6.1 技术栈

Vue 3 + TypeScript + Element Plus + Vite，与原 Dashboard 一致。

### 6.2 新增/修改文件

| 文件 | 变更类型 | 内容 |
|------|---------|------|
| `src/types/client.ts` | 修改 | `ClientInfoData` 新增 `disabled`、`banIP` + 治理 API 类型 |
| `src/utils/client.ts` | 修改 | `Client` 类新增 `disabled` 属性 |
| `src/api/admin.ts` | **新增** | 封装 6 个治理 API（客户端 4 个 + 代理 2 个） |
| `src/i18n/index.ts` | 修改 | `governance` 命名空间中英文翻译 |
| `src/components/ClientCard.vue` | 修改 | 禁用状态标签 + 指示灯适配 |
| `src/components/ProxyCard.vue` | 修改 | 在线代理「禁用代理」按钮 |
| `src/views/Clients.vue` | 修改 | 「已禁用」筛选 tab |
| `src/views/ClientDetail.vue` | 修改 | 客户端治理按钮 + 对话框 + 自动刷新 |
| `src/views/Proxies.vue` | 修改 | ProxyCard 禁用后自动刷新 |

---

## 七、CI/CD 工作流

### 7.1 自动打包

`fork-auto-package.yml`：每次 push 到任何分支时运行，编译产物上传为 Actions artifacts。

### 7.2 自动发布

`fork-auto-release.yml`：推送 `v*` 标签时触发，自动创建 GitHub Release。

```bash
git tag -a v0.68.0-arvin.7 -m "release message"
git push origin --tags
```

---

## 八、版本历史

| 标签 | 内容 |
|------|------|
| `v0.67.0-arvin.1` | 初始 fork 发布 |
| `v0.68.0-arvin.1` | 后端治理功能 + CI 工作流 + i18n |
| `v0.68.0-arvin.2` | 前端治理 UI（客户端禁用/启用/断开）+ Dashboard 状态展示 |
| `v0.68.0-arvin.3` | 修复 clientID 为空时治理失效 + 禁用对话框 UX 优化 |
| `v0.68.0-arvin.4` | 修复断开后详情页 404 报错 |
| `v0.68.0-arvin.5` | IP 封禁机制 + 离线客户端保留 |
| `v0.68.0-arvin.6` | IP 封禁改为可选，避免误封 |
| `v0.68.0-arvin.7` | **代理级别禁用/启用功能** |

---

## 九、升级与维护策略

### 9.1 仓库 Remote 配置

```
origin    → https://github.com/arvinws/frp.git    (你的 fork)
upstream  → https://github.com/fatedier/frp.git    (上游原仓库)
```

### 9.2 同步上游流程

```bash
git fetch upstream
# 查看上游 release note，重点关注锚点文件的变化
git checkout dev
git rebase upstream/dev
# 解决冲突（如有），重点关注下方锚点
git push origin dev --force-with-lease
```

### 9.3 升级时必须检查的锚点

| 锚点文件 | 关注内容 |
|---------|---------|
| `server/service.go` | `RegisterControl` 登录处理路径 |
| `server/control.go` | `handleNewProxy` / `ControlManager` / `worker` 生命周期 |
| `server/api_router.go` | Dashboard 路由注册方式 |
| `server/http/controller.go` | `buildClientInfoResp` / `NewController` 签名 |
| `server/http/model/types.go` | `ClientInfoResp` 结构体字段 |
| `server/registry/` | `ClientInfo.ClientID()` 回退逻辑、`MarkOfflineByRunID` |
| `server/proxy/` | `Proxy` 接口、`Manager` |
| `web/frps/` | 前端路由、API 层、组件结构 |
| `pkg/msg/msg.go` | `Login` 消息的 `ClientID` / `RunID` 字段 |

### 9.4 升级后回归测试清单

| 场景 | 预期结果 | 备注 |
|------|---------|------|
| 正常客户端登录 | 成功 | 未禁用时不受影响 |
| 已禁用客户端登录 | 被拒绝 | 检查审计日志 |
| 已在线客户端新建代理 | 被拒绝 | 验证 `control.go` hook |
| 禁用单条代理 | 该代理关闭，其他不受影响 | 代理级别治理 |
| 启用代理后客户端重连 | 代理恢复注册 | |
| 断开当前 runID | 连接断开 | 不改变禁用状态 |
| 禁用并断开客户端 | 先禁用再断开 | 客户端无法立即重连 |
| 启用客户端后重新登录 | 成功 | 规则恢复 |
| 未配置 clientID 的客户端 | 治理功能正常 | 使用 runID 回退 |
| IP 封禁 | 该 IP 所有连接被拒 | 需手动勾选 |
| 旧版/官方 frpc | 断开和代理禁用有效 | 客户端级别持久禁用需配合 IP 封禁 |

---

## 十、frpc 客户端配置建议

建议所有客户端在 `frpc.toml` 中配置 `clientID`：

```toml
serverAddr = "x.x.x.x"
serverPort = 7000
clientID = "office-server-01"    # 唯一标识，推荐用机器名

auth.method = "token"
auth.token = "your-token"

[[proxies]]
name = "rdp50050"
type = "tcp"
localIP = "127.0.0.1"
localPort = 3389
remotePort = 50050
```

命名建议：
- 用有意义的名称，如 `office-server-01`、`dev-machine-wang`
- 一台机器一个 frpc 进程用机器名即可
- 一台机器多个 frpc 进程，给每个取不同名称

---

## 十一、风险与规避

| 风险 | 规避措施 |
|------|---------|
| 上游重构 dashboard / route 注册位置 | handler 保持独立，只在一个注册点接入 |
| runID 生命周期或 control 管理方式变化 | ext 层不长期持有内部对象，只做查询与调用 |
| clientID 使用不规范 | 制定命名规范，在接入文档中固化 |
| 未配置 clientID 的客户端无法持久封禁 | IP 封禁（可选）或要求配置 clientID |
| IP 封禁误伤同 IP 其他客户端 | 默认关闭，前端有明确警告 |
| banlist 内存存储重启丢失 | 当前 `MemoryStore`，可扩展为文件/数据库持久化 |

---

## 十二、目录结构速查

```
pkg/ext/
├── adminapi/
│   ├── handler.go          # 6 个治理 API handler + ProxyCloser 接口
│   └── handler_test.go
├── banlist/
│   ├── store.go            # Store 接口 + MemoryStore（客户端/IP/代理三层）
│   └── store_test.go
├── clientmgr/
│   ├── manager.go          # 在线会话管理器
│   └── manager_test.go
└── audit/
    └── recorder.go         # 审计记录器

server/
├── service.go              # Hook: 登录禁用检查（clientID + IP）+ 会话注册
├── control.go              # Hook: 新代理禁用检查（客户端 + 代理名称）+ CloseProxyByName
├── api_router.go           # Hook: 全部治理路由注册
├── registry/registry.go    # 离线客户端保留（不删除）
└── http/
    ├── controller.go       # buildClientInfoResp 增加 disabled 查询
    └── model/types.go      # ClientInfoResp 增加 Disabled 字段

web/frps/src/
├── api/admin.ts            # 6 个治理 API 前端封装
├── types/client.ts         # 治理相关类型定义
├── utils/client.ts         # Client 类增加 disabled 属性
├── i18n/index.ts           # governance 命名空间翻译
├── components/
│   ├── ClientCard.vue      # 禁用状态标签
│   └── ProxyCard.vue       # 代理禁用按钮
└── views/
    ├── Clients.vue          # 「已禁用」筛选 tab
    ├── ClientDetail.vue     # 客户端治理操作 + 对话框
    └── Proxies.vue          # 代理禁用后自动刷新
```
