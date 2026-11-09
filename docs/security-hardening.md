# frps 公网加固与升级说明

本文描述本 fork 为关闭已复现的 frps 风险所做的默认行为变化。已经配置了 `auth.token` 的 frpc（包括 arvinws/frpc-launchpad）不需要改登录协议。

## 升级时会拒绝启动的配置

frps 在校验配置时直接退出，并打印原因。

- `bindAddr` 不是回环地址（包括 `0.0.0.0` 和 `::`），且 `auth.method = "token"` 时 `auth.token` 与 `auth.tokenSource` 都为空。只写了 `bindPort = 7000` 的旧文件属于这种情况，因为未写的 `bindAddr` 仍默认为 `0.0.0.0`。
- `webServer` 或 `metricsServer` 监听在非回环地址，且密码为空，或用户名和密码仍是 `admin` / `admin`。
- `auth.maxTimeDiffSeconds` 被设成小于 30 或大于 86400。未设置时默认为 600 秒。

本机调试可以保持：

```toml
bindAddr = "127.0.0.1"
bindPort = 7000
```

公网监听使用长随机 token，不要启用 `HeartBeats` / `NewWorkConns`，除非 frpc 也写了同样的 `auth.additionalScopes`。只在服务端打开这两项会让现有 frpc 的工作连接登录成功但转发失败。

## 登录

客户端仍发送当前时间戳和原来的 `MD5(token + 时间戳)`。服务端默认只接受与本机时钟相差 10 分钟以内的登录。完全相同的登录标识（token 摘要、主机名、用户、clientID、runID、系统、架构）在窗口内最多接受 4 次，用来挡住抓包重放，同时允许 frpc 在同一秒内的快速重试。

认证失败时，返回给客户端的错误固定为 `authorization failed`，并且不再带版本号。成功登录的响应仍包含版本和 `runID`。详细原因只写在 frps 日志里。

同一来源 IP 连续 10 次认证失败后，60 秒内继续拒绝。成功登录会清掉该计数。

## Dashboard、重启与 CSRF

命令行 `--dashboard_addr` 的默认值改为 `127.0.0.1`，不再默认用户名和密码 `admin`。

`webServer.pprofEnable` 的 `/debug/pprof/`，包括 `cmdline`，走与 Dashboard 相同的 Basic Auth。密码为空时认证中间件仍会放行，因此 pprof 只能和本机空密码一起使用；公网地址会在启动时被拒绝。

管理写接口（治理、计划任务、重启、删除代理、清理）会检查浏览器附带的 `Origin`、`Referer` 和 `Sec-Fetch-Site`。与当前 Host 不一致则返回 403。不带这些头的 curl 或脚本仍可调用。审计日志里的换行会被替换成空格。

`POST /api/admin/server/restart` 先关闭监听，再 `exec` 当前可执行文件和原来的参数。成功时 PID 不变，systemd 看不到进程退出。`exec` 失败时进程以状态码 1 退出：

| systemd `Restart=` | 成功 re-exec | re-exec 失败后的退出码 1 |
| --- | --- | --- |
| `no`（默认） | 服务继续运行 | 保持停止 |
| `on-failure` | 服务继续运行 | systemd 重新启动 |
| `always` | 服务继续运行 | systemd 重新启动 |

建议使用 `Restart=on-failure`。不要依赖退出码 0；以前的 `os.Exit(0)` 在 `on-failure` 下不会被拉起。Windows 没有进程内 `exec`，该路径直接以状态码 1 退出，交给服务管理器重启。

## 指纹

默认 vhost / tcpmux 404 页面不再包含 frp 或 GitHub 链接，响应也不再发送 `Server: frp/<version>`。未配置证书时自动生成的证书使用随机的 `localhost-` 名称和 90 天有效期，不再使用空主题加 10 年有效期。公网仍应配置自己的证书，并设置 `transport.tls.force = true`。

控制端口上的 `GET /~!frp` 是 frpc WebSocket 传输使用的路径，这次没有修改。

## 公网示例

```toml
bindAddr = "0.0.0.0"
bindPort = 7000
proxyBindAddr = "0.0.0.0"

auth.method = "token"
auth.token = "<长随机串>"
auth.maxTimeDiffSeconds = 600

transport.tls.force = true
transport.tls.certFile = "/etc/frp/server.crt"
transport.tls.keyFile = "/etc/frp/server.key"

allowPorts = [
  { start = 20000, end = 30000 },
]
maxPortsPerClient = 10
detailedErrorsToClient = false

webServer.addr = "127.0.0.1"
webServer.port = 7500
webServer.user = "<管理员>"
webServer.password = "<长随机串>"
webServer.pprofEnable = false

metricsServer.addr = "127.0.0.1"
metricsServer.port = 7501
metricsServer.user = "<监控账号>"
metricsServer.password = "<另一组长随机串>"
metricsServer.pprofEnable = false

custom404Page = "/etc/frp/404.html"
log.to = "/var/log/frps.log"
log.level = "info"
```

自定义 404 不要写产品名或版本。防火墙只放行实际使用的端口，frps 不要以 root 运行。
