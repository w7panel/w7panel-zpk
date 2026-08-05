# 测试证书
openssl genpkey -algorithm RSA -out registry-key.pem -pkeyopt rsa_keygen_bits:4096
openssl req -new -x509 -key registry-key.pem -out registry-cert.pem -days 365 -subj "/CN=registry.w7.com"

## 制品市场菜单

制品存在商品订单菜单时，仓库 `info` 接口会基于当前 Helm 包状态和 Bindings 内容生成缓存键，覆盖包内 MicroApp 的 `name: market` 菜单及独立的外部 iframe `backend_config`。没有市场 Binding 时直接返回原 Helm 包地址，不解包或重打；相同基础包和 Bindings 复用已有动态包，内容变化时才重新生成。动态包与 `PackFormulaToHelmAndPack` 产物位于同一目录，文件名格式为 `{原文件名去扩展名}-{hash}.tgz`，空闲 24 小时后清理。

`market` 菜单默认只对 `founder` 显示，使用现有 MicroApp 字段，不新增 `external_services`、`roles`、`icon` 或 `key` 协议字段。

市场前端地址通过环境变量 `DEPOT_MARKET_FRONTEND_URL` 配置，默认值为 `https://zm.w7.com`。

面板在安装阶段调用制品 `info` 时传入应用域名和规范化应用标识，ZPK 将二者写入加密 ticket。安装完成通知不再单独传递应用标识，ZPK 解票后再把 ticket 中的 `domain` 和 `app_identify` 传给制品市场订单核销接口。

订单安装或升级预检会返回已绑定的 `panel_url`、`panel_device_sn` 和 `app_identify`；校验失败时通过 `conflict_reason` 区分 `domain_mismatch`（域名不一致）和 `app_identify_exists`（已有应用标识绑定）。

绑定冲突使用 HTTP 409 返回结构化 `data`，其中包含 `conflict_reason`、原绑定 `domain`、`panel_url`、`panel_device_sn` 和 `app_identify`，供面板安装接口生成可操作的错误提示，并支持跳转原面板定位原应用。用户确认强制覆盖后可通过 `reinstall=true` 重新安装；该标记只允许非升级安装跳过旧绑定，升级仍严格校验应用标识。

网关插件 WasmPlugin 与配置 MicroApp 统一写入相同的 `metadata.labels["w7.cc/group-name"]` 归组关联，不再生成 `w7.cc/plugin-microapp` 注解。
