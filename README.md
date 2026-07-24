# 测试证书
openssl genpkey -algorithm RSA -out registry-key.pem -pkeyopt rsa_keygen_bits:4096
openssl req -new -x509 -key registry-key.pem -out registry-cert.pem -days 365 -subj "/CN=registry.w7.com"

## 制品市场服务入口

制品存在商品订单时，仓库信息接口会返回 `external_services`，供面板安装后写入 AppGroup 并展示“授权与续费”入口。

市场前端地址通过环境变量 `DEPOT_MARKET_FRONTEND_URL` 配置，默认值为 `https://zm.w7.com`。
