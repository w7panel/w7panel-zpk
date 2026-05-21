#!/bin/sh
set -e

tmp_config=/tmp/registry_config.yml

# 先移除原配置中的整个 notifications 顶层段，避免依赖其必须位于文件尾部。
awk '
BEGIN { skip = 0 }
{
  if (skip == 0 && $0 ~ /^notifications:$/) {
    skip = 1
    next
  }
  if (skip == 1) {
    if ($0 ~ /^[^[:space:]]/ && $0 !~ /^notifications:$/) {
      skip = 0
    } else {
      next
    }
  }
  print
}
' /etc/docker/registry/config.yml > "$tmp_config"

# 重新构建 notifications 配置，notify_config.yaml 仅保存 endpoints 列表内容。
cat >> "$tmp_config" <<'EOF'
notifications:
  events:
    includereferences: true
  endpoints:
EOF
sed 's/^/    /' /etc/docker/registry/notify_config.yaml >> "$tmp_config"

# 启动registry
exec registry serve "$tmp_config"
