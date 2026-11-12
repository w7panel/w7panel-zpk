# 保存配置时写入任务定义，触发时通过本轮 values 覆盖运行参数。
tasks:
{{ toYaml .Tasks | indent 2 }}
stages:
{{ toYaml .Stages | indent 2 }}

run:
  original: ""
  repository: ""
  host: ""
  tag: ""
  plainHTTP: false

selected: {}
remote: {}
kubeconfigs: {}
registryToken: ""

buildImage: ccr.ccs.tencentyun.com/afan-public/kaniko:w7console-new5-26
deployImage: registry.cdn.w7.cc/alpine/k8s:1.33.0
deadlineSeconds: 1800
