# 测试证书
openssl genpkey -algorithm RSA -out registry-key.pem -pkeyopt rsa_keygen_bits:4096
openssl req -new -x509 -key registry-key.pem -out registry-cert.pem -days 365 -subj "/CN=registry.w7.com"

## 制品市场服务入口

制品存在商品订单时，仓库信息接口会返回 `external_services`，供面板安装后写入 AppGroup 并展示“授权与续费”入口。

市场前端地址通过环境变量 `DEPOT_MARKET_FRONTEND_URL` 配置，默认值为 `https://zm.w7.com`。

面板在安装阶段调用制品 `info` 时传入应用域名和规范化应用标识，ZPK 将二者写入加密 ticket。安装完成通知不再单独传递应用标识，ZPK 解票后再把 ticket 中的 `domain` 和 `app_identify` 传给制品市场订单核销接口。

订单安装或升级预检会返回已绑定的 `panel_url`、`panel_device_sn`；校验失败时通过 `conflict_reason` 区分 `domain_mismatch`（域名不一致）和 `app_identify_exists`（已有应用标识绑定）。

绑定冲突使用 HTTP 409 返回结构化 `data`，其中包含 `conflict_reason`、原绑定 `domain`、`panel_url` 和 `panel_device_sn`，供面板安装接口生成可操作的错误提示。
