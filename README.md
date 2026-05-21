# 测试证书
openssl genpkey -algorithm RSA -out registry-key.pem -pkeyopt rsa_keygen_bits:4096
openssl req -new -x509 -key registry-key.pem -out registry-cert.pem -days 365 -subj "/CN=registry.w7.com"
