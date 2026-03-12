# 定时任务管理模块 V1 方案

## 1. 背景

当前 fork 已具备客户端级、代理级治理能力，但代理的启用/禁用链路仍不完整：

- `frps` 侧可以关闭代理并阻止重新注册。
- `frpc` 侧还缺少稳定的“单代理停用/恢复”闭环。
- 现有代理禁用状态是单值模型，不适合后续叠加“手动禁用 + 定时禁用”。

因此，定时任务模块不能直接建立在当前代理启/禁用实现之上。建议先补齐单代理治理闭环，再实现定时调度。

## 2. 目标

V1 目标是提供一个中心化的“定时任务管理模块”，支持：

1. 管理指定隧道（代理）的定时启动与定时禁用。
2. 从当前代理列表中选择代理，加入到任务中。
3. 在 `frps` 侧统一配置、执行、审计和展示执行结果。
4. 保证“手动禁用”和“计划禁用”互不覆盖。

## 3. V1 范围

### 3.1 包含内容

- 任务 CRUD（创建、查询、修改、删除）
- 任务启用/停用
- 任务立即执行
- 任务执行日志
- 代理选择接口
- 支持多代理绑定到一个任务
- 支持以下时间规则：
  - 单次
  - 每日
  - 每周

### 3.2 不包含内容

- 不直接修改 `frpc.toml` 或其他静态配置文件
- 不做复杂 cron 表达式编辑器
- 不做按标签、按分组、按正则动态选择代理
- 不做数据库持久化（V1 采用文件持久化）
- 不做跨任务优先级系统（V1 用多来源禁用聚合规则解决冲突）

## 4. 总体设计

### 4.1 设计原则

- 调度中心放在 `frps` 端
- 任务对象是“代理”，不是“客户端”
- 任务保存的是“精确代理名列表”
- 运行时控制优先，不改源配置文件
- 所有任务执行结果可审计、可追踪

### 4.2 为什么放在 `frps`

- `frps` 已经掌握当前代理列表、在线状态、所属客户端信息。
- 可以统一治理，不依赖每个 `frpc` 暴露本地管理接口。
- 便于做任务列表、执行日志和审计页面。
- 与现有 dashboard 和治理 API 体系一致。

## 5. 前置改造

在实现调度模块前，建议先完成以下基础改造：

### 5.1 修复单代理启用/禁用闭环

现状问题：

- 仅关闭 `frps` 侧代理，不等于 `frpc` 侧代理真正停用。
- 启用后缺少明确的 client-side 恢复机制。

建议：

- 新增 server -> client 的单代理控制消息，例如 `ProxyControl`。
- 支持以下动作：
  - `disable`
  - `enable`
- `frpc` 收到 `disable` 后，将对应 proxy wrapper 置为暂停状态，并停止重试注册。
- `frpc` 收到 `enable` 后，恢复对应 proxy wrapper，并立即尝试重新注册。

### 5.2 将代理禁用模型改为“多来源禁用”

当前模型类似：

- `proxyName -> disabled(bool)`

建议改为：

- `proxyName -> set(sources)`

来源示例：

- `manual:ops`
- `schedule:task_01`

最终规则：

- 只要某代理仍存在至少一个禁用来源，该代理就保持禁用。

这样可以避免：

- 手动禁用被定时任务错误解除
- 多个任务相互覆盖禁用状态

## 6. 任务模型

### 6.1 任务结构

```json
{
  "id": "task_01",
  "name": "办公时段代理控制",
  "enabled": true,
  "timezone": "Asia/Shanghai",
  "targets": [
    {
      "proxyName": "alice.rdp",
      "user": "alice",
      "clientID": "office-pc-01",
      "type": "tcp"
    },
    {
      "proxyName": "alice.ssh",
      "user": "alice",
      "clientID": "office-pc-01",
      "type": "tcp"
    }
  ],
  "startRule": {
    "mode": "weekly",
    "daysOfWeek": [1, 2, 3, 4, 5],
    "time": "09:00"
  },
  "stopRule": {
    "mode": "weekly",
    "daysOfWeek": [1, 2, 3, 4, 5],
    "time": "18:00"
  },
  "remark": "工作日开放",
  "createdAt": "2026-03-12T10:00:00+08:00",
  "updatedAt": "2026-03-12T10:00:00+08:00"
}
```

### 6.2 字段说明

| 字段 | 说明 |
| --- | --- |
| `id` | 任务唯一标识 |
| `name` | 任务名称 |
| `enabled` | 任务是否启用 |
| `timezone` | 任务计算时间所用时区 |
| `targets` | 目标代理列表 |
| `startRule` | 启动规则 |
| `stopRule` | 禁用规则 |
| `remark` | 备注 |
| `createdAt` | 创建时间 |
| `updatedAt` | 修改时间 |

### 6.3 目标代理结构

| 字段 | 说明 |
| --- | --- |
| `proxyName` | 精确代理名，作为执行主键 |
| `user` | 用户名，便于展示 |
| `clientID` | 所属客户端，便于展示和过滤 |
| `type` | 代理类型，便于展示和筛选 |

## 7. 时间规则设计

V1 支持三种模式：

### 7.1 单次

```json
{
  "mode": "once",
  "date": "2026-03-20",
  "time": "09:00"
}
```

### 7.2 每日

```json
{
  "mode": "daily",
  "time": "09:00"
}
```

### 7.3 每周

```json
{
  "mode": "weekly",
  "daysOfWeek": [1, 2, 3, 4, 5],
  "time": "09:00"
}
```

### 7.4 设计说明

- V1 不直接开放 cron 表达式。
- 前端先做结构化配置，降低误配置概率。
- 后端结构可预留 `cron` 模式，供 V2 扩展。

## 8. 状态与冲突规则

### 8.1 代理最终状态计算

一个代理的禁用状态由多个来源共同决定：

- 手动禁用
- 计划任务禁用
- 其他未来扩展来源

最终规则：

- 只要来源集合非空，代理即为 `disabled`
- 仅当来源集合为空时，代理才视为 `enabled`

### 8.2 典型场景

#### 场景 A：任务禁用，随后手动禁用

- 任务写入来源：`schedule:task_01`
- 手动写入来源：`manual:ops`
- 当任务到启动时间，只移除 `schedule:task_01`
- 由于 `manual:ops` 仍存在，代理仍保持禁用

#### 场景 B：两个任务同时控制同一代理

- `task_01` 写入 `schedule:task_01`
- `task_02` 写入 `schedule:task_02`
- 若 `task_01` 到启动时间，仅移除自己的来源
- 因 `task_02` 仍存在，该代理仍保持禁用

## 9. 执行语义

### 9.1 到达禁用时间

对每个目标代理执行：

1. 写入禁用来源 `schedule:{taskID}`
2. 若代理在线，则下发 client-side `disable`
3. 同步关闭 `frps` 当前在线代理实例
4. 写入执行日志

### 9.2 到达启动时间

对每个目标代理执行：

1. 移除禁用来源 `schedule:{taskID}`
2. 若该代理已无其他禁用来源，则下发 client-side `enable`
3. 允许重新注册并恢复在线
4. 写入执行日志

### 9.3 客户端离线时的处理

- 若客户端离线，仍然要先更新 `frps` 的禁用来源状态
- 等客户端后续上线时，再根据当前聚合状态决定是否允许注册
- 若当前仍处于禁用窗口，则继续拦截注册

## 10. 调度器设计

### 10.1 运行方式

- `frps` 内部常驻一个 scheduler engine goroutine
- 每 15~30 秒扫描一次任务
- 计算是否命中启动或禁用时间点

### 10.2 重启行为

- `frps` 启动时从持久化文件加载任务
- 重新计算每个任务的下一次执行时间

### 10.3 漏执行策略

V1 建议采用：`skip`

即：

- 若服务停机期间错过某次执行时间，不补跑
- 重启后只计算下一次执行点

原因：

- 行为更可预测
- 风险更低
- 便于先稳定上线

## 11. API 设计

### 11.1 任务管理

#### `GET /api/admin/schedules`

查询任务列表。

#### `POST /api/admin/schedules`

创建任务。

#### `GET /api/admin/schedules/{id}`

查询任务详情。

#### `PUT /api/admin/schedules/{id}`

更新任务。

#### `DELETE /api/admin/schedules/{id}`

删除任务。

#### `POST /api/admin/schedules/{id}/enable`

启用任务。

#### `POST /api/admin/schedules/{id}/disable`

停用任务。

#### `POST /api/admin/schedules/{id}/run?action=start|stop`

立即执行一次任务动作。

### 11.2 执行日志

#### `GET /api/admin/schedules/{id}/logs`

查询任务执行日志。

日志字段建议包含：

- `executionID`
- `taskID`
- `action`
- `scheduledAt`
- `executedAt`
- `targetProxyName`
- `result`
- `error`

### 11.3 代理选项接口

#### `GET /api/admin/proxy-options`

用于任务创建/编辑时选择目标代理。

返回建议：

```json
[
  {
    "proxyName": "alice.rdp",
    "displayName": "alice.rdp",
    "type": "tcp",
    "user": "alice",
    "clientID": "office-pc-01",
    "status": "online",
    "disabled": false
  }
]
```

支持过滤参数：

- `keyword`
- `clientID`
- `user`
- `type`
- `status`

## 12. 持久化方案

V1 建议沿用当前项目已存在的 JSON 文件持久化思路。

### 12.1 推荐文件

- `./data/frps-schedules.json`
- `./data/frps-schedule-logs.json`

### 12.2 设计原因

- 实现简单
- 和现有 `StoreSource` 的文件持久化风格一致
- 便于先验证业务闭环
- 后续可替换为数据库，不影响接口层

## 13. 前端页面方案

### 13.1 页面入口

新增一级菜单：`计划任务`

### 13.2 列表页字段

- 任务名
- 启用状态
- 目标数量
- 下次启动时间
- 下次禁用时间
- 最近执行结果
- 操作：编辑 / 启停 / 立即执行 / 删除 / 查看日志

### 13.3 新建/编辑任务弹窗

建议包含三个区域：

1. 基本信息
   - 任务名
   - 时区
   - 备注
2. 时间规则
   - 启动规则
   - 禁用规则
3. 目标代理
   - 从代理列表筛选
   - 多选后加入任务

### 13.4 代理选择器

建议交互：

- 左侧：筛选条件
  - 客户端
  - 类型
  - 搜索关键字
- 中间：可选代理列表
- 右侧：已选代理列表

### 13.5 现有页面联动

后续可在以下页面加入快捷入口：

- `web/frps/src/views/Proxies.vue`
  - 批量选择代理后加入任务
- `web/frps/src/views/ProxyDetail.vue`
  - 基于当前代理快速创建任务

## 14. 后端代码落点建议

### 14.1 新增目录

- `pkg/ext/scheduler/`
  - `types.go`
  - `store.go`
  - `engine.go`
  - `service.go`

- `pkg/ext/governance/`
  - 统一封装手动治理与计划治理的代理状态操作

### 14.2 需要修改的现有文件

- `pkg/ext/banlist/store.go`
- `pkg/ext/adminapi/handler.go`
- `server/api_router.go`
- `server/control.go`
- `server/http/controller.go`
- `client/control.go`
- `client/proxy/proxy_manager.go`
- `client/proxy/proxy_wrapper.go`
- `web/frps/src/api/`
- `web/frps/src/views/`

## 15. 建议实施顺序

### 阶段 1：修复当前代理治理闭环

- 修复单代理 disable/enable 的 client-side 真正停用/恢复
- 补充服务端到客户端的单代理控制能力

### 阶段 2：升级代理禁用模型

- 将 `proxy disabled` 从单值改为多来源聚合
- 保证手动禁用与计划禁用不会互相覆盖

### 阶段 3：实现调度核心

- 任务模型
- 文件持久化
- scheduler engine
- 执行日志

### 阶段 4：实现管理 API

- 任务 CRUD
- 启停任务
- 立即执行
- 代理选择接口

### 阶段 5：实现前端页面

- 任务列表页
- 新建/编辑弹窗
- 日志查看
- 与代理列表页联动

## 16. 验收标准

以下场景通过，可视为 V1 基本可用：

1. 能从代理列表选择一个或多个代理创建任务。
2. 任务到禁用时间后，目标代理立即下线并阻止重新注册。
3. 任务到启动时间后，目标代理恢复可注册并重新上线。
4. 手动禁用的代理不会被定时任务误放开。
5. 两个任务同时控制同一代理时，状态计算正确。
6. `frps` 重启后任务仍存在，并能继续计算下一次执行时间。
7. 页面能看到最近执行结果和失败原因。

## 17. 风险与注意事项

### 17.1 当前已知风险

- 现有单代理治理闭环未补齐前，定时任务不能稳定上线。
- 仅依赖 server-side close 可能导致 client-side 状态不一致。
- 代理列表来源若仅依赖当前 dashboard 统计，历史从未出现过的代理无法直接选取。

### 17.2 V1 权衡

- 为降低复杂度，V1 先用“精确代理名列表”而非动态匹配规则。
- 为降低实现风险，V1 先用文件存储，不引入数据库。
- 为降低误配置概率，V1 先用结构化时间规则，不开放 cron。

## 18. 结论

该方案适合作为当前 fork 的第一版定时调度能力：

- 与现有治理体系兼容
- 支持从代理列表选择目标
- 能满足“定时启动 / 定时禁用指定隧道”的核心诉求
- 具备后续扩展到 cron、数据库、动态分组任务的演进空间

但建议严格按顺序推进：

1. 先修复当前单代理启/禁用闭环
2. 再实现多来源禁用模型
3. 最后再做定时任务管理模块
