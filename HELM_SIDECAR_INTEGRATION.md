# Helm Chart 接入 Sidecar 示例

本文面向上传或导入 Helm 包的 Chart 作者，以 `w7panel-cloudnoauth` 为例，说明如何调整自己的 Chart。示例按当前 ZPK 实现编写，最后核对日期：2026-11-06。

## 1. 平台和 Chart 各负责什么

在 ZPK 制品中开启“创建站点”（`application.registerSite=true`）后，平台会下载 cloudnoauth，将它作为 library dependency 放入最终包，写入 `w7panelSidecars`，并补齐 sidecar helper 和资源入口。Site 由根制品生成。

平台不会解析或改写用户 Chart 的 Deployment、StatefulSet、DaemonSet、Job、ServiceAccount 和业务 RBAC。Chart 作者需要完成下面三件事：

1. 在需要代理的 Pod 模板中调用 sidecar helper，接入容器、hostAliases、volumes 和注解。
2. 通过 `w7panelArtifact.serviceAccountName` 声明 Pod 实际使用的账号，账号仍由原 Chart 或安装方创建。
3. 包含主子应用时，让需要代理的 Pod 使用根 sidecar Binding 授权的账号，或自行补齐其他账号的权限。

每个 Binding 只绑定一个账号；共享子应用显式声明账号时会生成该账号的额外 Binding。平台不会自动发现任意 Pod 使用的账号。`w7panelArtifact.serviceAccountName` 是授权声明，不会改变 Pod 的 `serviceAccountName`，也不会创建账号。


### 主子应用接入与账号特别说明

> **凡实际使用 Sidecar 的用户 Helm Chart，无论主应用还是子应用，都建议显式配置 `w7panelArtifact.serviceAccountName`，并与 Pod / Job 实际使用的账号保持一致，包括使用 `default` 的情况。** 此字段仅向 Sidecar 资源模板声明授权对象，不会创建 ServiceAccount，也不会修改 Pod 的账号。账号仍由 Chart 或安装方创建。

```yaml
w7panelArtifact:
  # 使用当前宿主 Chart 的实际账号 helper；主、子 Chart 各自填写。
  serviceAccountName: '{{ include "your-chart.serviceAccountName" . }}'
```

模板字符串在当前宿主 Chart 上下文执行；Helm 子应用应引用自己的 helper，也可以填写实际账号的固定名称。不要把 Helm 表达式直接填入不支持 `tpl` 的 `serviceAccount.name`。

| 主应用 / 子应用 | Sidecar 接入及账号要求 |
| --- | --- |
| 生成型主应用 / 生成型子应用 | 跟随根制品创建站点开关，自动配置代理插槽及账号声明；默认共用根账号，未开启集群权限时用 `default`。局部覆盖账号后仍需核对账号存在及权限。 |
| 生成型主应用 / Helm 子应用 | 子 Chart 自行接入 Pod / Job 插槽，并显式声明实际账号；使用根账号时关闭重复创建，使用其他账号时保证该账号存在。 |
| Helm 主应用 / 生成型子应用 | 主 Chart 自行接入插槽、创建或引用账号并声明账号；根账号不会自动传给生成型子应用，必要时在最终安装 values 中显式设置子应用的 `serviceAccount.name`。 |
| Helm 主应用 / Helm 子应用 | 各自接入插槽并声明各自实际账号；共用账号时明确由谁创建，使用不同账号时分别保证账号存在及授权。 |

“Helm 应用”指实际进入用户 Chart / YAML 打包分支的应用；生成型 Chart 指由 ZPK 生成工作负载模板的应用。所有 ZPK 子应用跟随根制品的创建站点开关；仅把 library 打入包不会自动改写用户 Helm Chart 的 Pod。

资源输出规则：

- 根 Chart 输出完整 Sidecar resources；未声明账号时仍会求值，实际账号回退行为由 Sidecar 自己的模板决定，不能假定一定匹配业务 Pod。
- 共享子应用有有效账号时，仅追加 RoleBinding / ClusterRoleBinding，并改写名称和 subjects；Service、Role、ClusterRole 等共享资源由根 Chart 输出。
- 共享子应用账号缺失或求值为空时，跳过 resources 模板求值及输出，不报错，也不关闭容器插槽。若仍使用代理，须自行保证权限，否则可能运行时报 403。
- 不使用 Sidecar 的 Helm 工具子应用（例如未调用代理插槽的 `w7-traditiontool`）无需为了打包而补账号声明。
- 主子应用必须使用一致的共享资源名称和连接配置，避免子代理引用不存在的 Service 或角色。

**共享 Binding 去重：** ZPK 注入的资源入口由根 Chart 统一收集共享子应用的 Binding，按 kind、namespace、roleRef 和 subjects 去重，优先保留根 Binding。同账号的相同授权只输出一次，不同账号保留各自绑定。子应用入口不再独立输出资源；容器与卷插槽不受影响。发布前仍须检查自定义资源的标识唯一性。

## 2. 最小 Chart 示例

示例 Chart 名为 `sidecar-demo`。实际使用时保留自己的应用镜像、账号 helper、存储和业务资源，将示例插槽合并进对应模板。

```text
sidecar-demo/
├── Chart.yaml
├── values.yaml
└── templates/
    ├── _helpers.tpl
    ├── _w7panel-sidecars.tpl
    ├── deployment.yaml
    ├── lifecycle-job.yaml
    ├── serviceaccount.yaml
    └── w7panel-sidecar-resources.yaml
```

### Chart.yaml

```yaml
apiVersion: v2
name: sidecar-demo
description: Helm sidecar integration example
type: application
version: 0.1.0
appVersion: "1.0.0"
```

通过 ZPK 自动打包时，原 Chart 可以不包含 cloudnoauth dependency，平台会添加。原 Chart 自己已有的依赖仍需完整打入包中。

### values.yaml

```yaml
replicaCount: 1

image:
  repository: nginx
  tag: "1.27-alpine"
  pullPolicy: IfNotPresent

serviceAccount:
  create: true
  name: ""

podAnnotations: {}

# 原包默认不加载 sidecar；ZPK 开启创建站点后会写入此列表。
w7panelSidecars: []

# 必须与 Deployment、Job 实际使用的账号一致。
# 模板字符串由资源 helper 在当前宿主 Chart 上下文中执行。
w7panelArtifact:
  serviceAccountName: '{{ include "demo.serviceAccountName" . }}'
  sharedSidecarResources: false

lifecycleJob:
  enabled: false
  image: busybox:1.36
```

`serviceAccount.name` 在本示例中是普通字符串，不执行 Helm 表达式。`w7panelArtifact.serviceAccountName` 则支持模板字符串；不要把两者混用。

### templates/_helpers.tpl

```gotemplate
{{- define "demo.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "demo.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "demo.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/* 当前 cloudnoauth 接入时显式请求根 CA 注入。 */}}
{{- define "demo.podAnnotations" -}}
{{- $annotations := include "w7panel.podAnnotations" . | fromYaml | default dict -}}
{{- if .Values.w7panelSidecars -}}
{{- $_ := set $annotations "w7.cc/inject-root-ca" "true" -}}
{{- end -}}
{{- if $annotations -}}
{{ toYaml $annotations }}
{{- end -}}
{{- end -}}
```

实际 Chart 可以继续使用已有的账号命名 helper，只需让 Pod 和 `w7panelArtifact.serviceAccountName` 调用同一个 helper。

根 CA 注解要落在 `spec.template.metadata.annotations`，不能只写在 Deployment 顶层。当前 Helm 打包分支会在创建站点时向用户 Chart values 写入根 CA 注解，但 Chart 模板仍须把这些 values 渲染到 Pod，不能仅依赖制品中的开关。集群还需具备根 CA 注入、cert-manager CSI 和对应 ClusterIssuer，业务进程需信任注入后的 CA。

### templates/_w7panel-sidecars.tpl

使用平台提供的完整 [sidecar helper](app/respo/logic/helm/helm_templates/_w7panel-sidecars.tpl)，复制到自己的 `templates/_w7panel-sidecars.tpl`。例如，在本仓库根目录创建示例 Chart 时：

```bash
cp app/respo/logic/helm/helm_templates/_w7panel-sidecars.tpl \
  sidecar-demo/templates/_w7panel-sidecars.tpl
```

ZPK 打包时会写入平台版本的该文件。原包也携带 helper，可以在 `w7panelSidecars: []` 时直接进行本地 lint 和渲染，避免上传前出现 `no template "w7panel.podAnnotations"`。

### templates/serviceaccount.yaml

```gotemplate
{{- if .Values.serviceAccount.create }}
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ include "demo.serviceAccountName" . }}
{{- end }}
```

`create=false` 表示账号由外部或主 Chart 提供。业务 Role、ClusterRole 和 Binding 应按自身需要保留，不能因为不再创建账号就连业务权限一起关闭。

### templates/w7panel-sidecar-resources.yaml

```gotemplate
{{- include "w7panel.sidecars.resources" . }}
```

ZPK 会补齐同名入口。自己的 Chart 携带时只保留这一处调用，避免另建文件重复输出共享 Service/RBAC。内嵌 ZPK 子应用由打包器设置 `sharedSidecarResources=true`，其入口不重复输出 Service、Role 等共享资源；有账号时仅输出额外 Binding，无账号时不输出 resources。

### templates/deployment.yaml

```gotemplate
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "demo.fullname" . }}
  labels:
    w7.cc/group-name: {{ .Release.Name | quote }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ .Chart.Name | quote }}
      app.kubernetes.io/instance: {{ .Release.Name | quote }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ .Chart.Name | quote }}
        app.kubernetes.io/instance: {{ .Release.Name | quote }}
        w7.cc/group-name: {{ .Release.Name | quote }}
      {{- $annotations := include "demo.podAnnotations" . }}
      {{- if $annotations }}
      annotations:
        {{- $annotations | nindent 8 }}
      {{- end }}
    spec:
      serviceAccountName: {{ include "demo.serviceAccountName" . }}
      {{- $hostAliases := include "w7panel.sidecars.hostAliases" . }}
      {{- if $hostAliases }}
      hostAliases:
        {{- $hostAliases | nindent 8 }}
      {{- end }}
      {{- $initContainers := include "w7panel.sidecars.initContainers" . }}
      {{- if $initContainers }}
      initContainers:
        {{- $initContainers | nindent 8 }}
      {{- end }}
      containers:
        - name: main
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          volumeMounts:
            - name: app-data
              mountPath: /tmp/app-data
        {{- include "w7panel.sidecars.containers" . | nindent 8 }}
      volumes:
        - name: app-data
          emptyDir: {}
        {{- include "w7panel.sidecars.volumes" . | nindent 8 }}
```

保留 Pod 上的 `w7.cc/group-name: <Release.Name>`，使代理可以直接找到当前制品的 Site。对于 StatefulSet、DaemonSet，也在自己的 Pod 模板中接入相同插槽。

已有 `hostAliases`、`initContainers`、`containers`、`volumes` 时，将 sidecar 片段追加进同一个数组，不能重复声明同名 YAML key。示例的 `app-data` 展示了保留业务 volume 的方式；容器名、volume 名还需避免与 sidecar 重名。

Workload 中主容器和普通 sidecar 并行启动。业务启动时立即调用授权接口的应用，需要自行等待或重试；readinessProbe 不会阻止主容器进程启动。

### templates/lifecycle-job.yaml（可选）

```gotemplate
{{- if .Values.lifecycleJob.enabled }}
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ printf "%s-lifecycle" (include "demo.fullname" .) | trunc 63 | trimSuffix "-" }}
  annotations:
    helm.sh/hook: post-install,post-upgrade
    helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded
spec:
  backoffLimit: 2
  template:
    metadata:
      labels:
        w7.cc/group-name: {{ .Release.Name | quote }}
      {{- $annotations := include "demo.podAnnotations" . | fromYaml | default dict }}
      {{- $annotations = omit $annotations "sysbox/rootfs-rw-layer" }}
      {{- if $annotations }}
      annotations:
        {{- toYaml $annotations | nindent 8 }}
      {{- end }}
    spec:
      serviceAccountName: {{ include "demo.serviceAccountName" . }}
      restartPolicy: Never
      {{- $hostAliases := include "w7panel.sidecars.jobHostAliases" . }}
      {{- if $hostAliases }}
      hostAliases:
        {{- $hostAliases | nindent 8 }}
      {{- end }}
      {{- $initContainers := include "w7panel.sidecars.jobInitContainers" . }}
      {{- if $initContainers }}
      initContainers:
        {{- $initContainers | nindent 8 }}
      {{- end }}
      containers:
        - name: task
          image: {{ .Values.lifecycleJob.image | quote }}
          command: ["/bin/sh", "-c"]
          args: ["echo lifecycle-complete"]
      {{- $volumes := include "w7panel.sidecars.jobVolumes" . }}
      {{- if $volumes }}
      volumes:
        {{- $volumes | nindent 8 }}
      {{- end }}
{{- end }}
```

Job 使用 `jobInitContainers`，cloudnoauth 会作为 `restartPolicy: Always` 的原生 sidecar 放入 `initContainers`，避免代理阻止主任务结束。不要在 Job 中调用普通 `sidecars.containers`。目标 Kubernetes 集群必须支持原生 sidecar。

本示例使用 post hook。共享 Service/RBAC 遵循普通资源生命周期，`pre-install` 创建前、`post-delete` 删除后获取不到这些资源，保持真实安装和卸载时序。代理进程启动成功也不代表 Site 已经产生授权凭据，业务请求仍需处理尚未就绪的情况。

## 3. 包含主子应用时如何配置账号

以下 `worker` 表示最终 `Chart.yaml.dependencies` 中的子 Chart key，实际使用时替换为自己的子应用名称。配置的是最终聚合包，不能只修改被导入子应用的原始 values。

### Helm 主应用＋生成型子应用

本示例的根 Chart 会创建账号，所有需要代理的子 Pod 和 Job 使用这个账号。假设安装 Release 为 `demo`，创建一份安装时使用的 `values-shared-account.yaml`：

```yaml
serviceAccount:
  name: demo-sidecar-demo

worker:
  serviceAccount:
    name: demo-sidecar-demo
```

根 `w7panelArtifact.serviceAccountName` 调用账号 helper，自动得到相同名称。生成型子应用无论其 `serviceAccount.create` 默认值是什么，都会使用显式 `name`；账号由根 Helm Chart 创建。

当前实现不会自动把 Helm 根 Chart 的账号传给生成型子应用。仅设置 `global.name` 或根 `w7panelArtifact.serviceAccountName` 不足以完成对齐。尤其根制品开启集群权限时，未指定子账号会让生成型子应用引用一个没有创建的账号。

此 values 文件是针对本次安装的配置。更换 Release 或账号时，主子账号一起修改；原 Chart 的默认账号名仍可保持为空，避免不同安装固定共用同一个账号。

### 生成型主应用＋Helm 子应用

根制品未开启集群权限时，生成型主应用默认用 `default`。Helm 子 Chart 如果支持上述 `serviceAccount` 配置，可以这样调整最终包：

```yaml
worker:
  serviceAccount:
    create: false
    name: default
```

根制品开启集群权限时，先确认根账号实际名称，再将子 Chart 的 `name` 设置为同一名称，`create=false`，由根 Chart 创建。保留子 Chart 所需的业务 RBAC，并使其 Binding 指向这个账号。

Helm 主应用＋Helm 子应用也遵循相同原则：根 Chart 创建账号，子 Chart 选择同一个账号并关闭重复创建。某些 Chart 使用其他配置名或把账号写死，需要作者调整自己的 helper、Pod 和权限模板；ZPK 不推断这些语义。

### 原 Chart 自带依赖或多个工作负载

原 Helm Chart 自带的数据库、registry 等 dependency 不等同于 ZPK 导入的子应用，平台不会遍历并自动修改它们的 Pod 模板。只有需要调用授权 API 的组件才需要接入 sidecar。

如果在这些依赖中接入，需为该宿主准备自己的 library dependency、`w7panelSidecars` 和 helper 插槽，并设置 `w7panelArtifact.sharedSidecarResources=true`，复用根共享资源。账号和 sidecar 的共享资源名称配置必须与根一致。

一个 Chart 的多个代理 Pod 使用不同账号时，当前单账号声明无法覆盖全部账号。可以将这些 Pod 调整为共享账号，或者由 Chart 自行追加所需的 RoleBinding/ClusterRoleBinding；平台不会根据任意 Chart 内容自动发现账号。

## 4. 本地验证和打包

### 原始 Chart

创建第 2 节文件，并复制完整 helper。默认 `w7panelSidecars: []` 时不需要 sidecar dependency：

```bash
helm lint ./sidecar-demo
helm template demo ./sidecar-demo --namespace demo-ns
helm package ./sidecar-demo --destination /tmp
```

将生成的原包上传到 ZPK，开启“创建站点”，再取得平台生成的最终聚合包。原始 Helm 包和平台最终包是两个不同产物；仅上传原包不会自动改写其中的业务模板。

### 用真实 sidecar 做本地渲染

在临时副本的 `charts/w7panel-cloudnoauth/` 放入真实 library Chart，再将以下 dependency 合并到该副本的 `Chart.yaml`。这里以本地 cloudnoauth Chart `0.3.4` 为例，其他版本以实际 `Chart.yaml.version` 为准：

```yaml
dependencies:
  - name: w7panel-cloudnoauth
    version: 0.3.4
    repository: file://./charts/w7panel-cloudnoauth
```

创建 `sidecar-enabled.yaml`，用于临时副本或已包含该 dependency 的最终包：

```yaml
w7panelSidecars:
  - chart: w7panel-cloudnoauth
lifecycleJob:
  enabled: true
```

```bash
helm dependency list ./sidecar-demo
helm lint ./sidecar-demo -f sidecar-enabled.yaml
helm template demo ./sidecar-demo --namespace demo-ns \
  -f sidecar-enabled.yaml > /tmp/sidecar-demo-rendered.yaml
```

本地添加 library dependency 只验证模板接入，不会自动生成 Site 或替代 ZPK 的创建站点设置。实际部署使用平台最终包。

### 平台最终包

平台最终包会包含 Site。离线渲染时用 `final-render-values.yaml` 提供安装端通常传入的站点参数，下面是本地验证用的示例值：

```yaml
DOMAIN_URL: demo.example.test
global:
  panel:
    userName: demo-user
```

```bash
helm show values ./final-app.tgz
helm lint ./final-app.tgz -f final-render-values.yaml \
  -f values-shared-account.yaml
helm template demo ./final-app.tgz --namespace demo-ns \
  -f final-render-values.yaml \
  -f values-shared-account.yaml > /tmp/final-app-rendered.yaml
```

没有子应用时省略 `-f values-shared-account.yaml`。实际安装使用安装端的真实站点参数。原 Chart 使用严格 `values.schema.json` 时，也要允许新增的 `w7panelSidecars`、`w7panelArtifact` 及实际注入的 dependency values，否则 schema 校验可能阻止渲染。

检查渲染结果：

- Workload 有 cloudnoauth 容器、`api.w7.cc` hostAliases、CSI volume 和对应 volumeMount。
- Pod 有 `w7.cc/inject-root-ca: "true"` 和正确的 `w7.cc/group-name`。
- Job 的代理在 `initContainers`，带 `restartPolicy: Always`。
- 所有需要代理的 Pod/Job 的账号与共享两个 Binding 的 subject 一致；非 `default` 账号已由根 Chart 创建或由安装方提供。
- 整个制品只有一份共享 sidecar Service、Role、ClusterRole；子应用可以追加账号 Binding。检查所有资源的 kind、namespace、name 组合唯一，尤其多个子应用共用账号时。
- 关闭创建站点时，最终包不注入 cloudnoauth，也不因本示例条件额外开启根 CA 注解。

## 5. 常见问题

| 现象 | 检查位置 |
| --- | --- |
| 有 sidecar dependency/RBAC，但 Pod 没有代理 | 原 Workload 是否调用了容器和其他插槽 |
| `no template "w7panel.*"` | 原包是否携带 helper；最终包是否经过平台打包 |
| `.Subcharts` 中找不到 sidecar | library dependency、Chart name、引用 key 是否一致且已启用 |
| 子 Pod 报 ServiceAccount 不存在 | 子账号是否误用自己的默认名称；是否由根创建对应账号 |
| 代理读取 Pod/Site 返回 403 | Pod 账号与共享 Binding subject 是否一致，业务 RBAC 是否遗漏 |
| HTTPS 提示证书不受信任 | Pod 根 CA 注解、集群注入及业务进程信任库 |
| 主程序第一次获取凭据失败后退出 | 是否处理代理启动或 Site 凭据尚未就绪的重试 |
| Job 主任务结束后仍不完成 | 是否误把普通 sidecar 放在 Job 的 `containers` 中 |

完整协议和实现边界见 [Helm Sidecar 集成约定](HELM_SIDECAR.md)。

- 导入现成 Helm Chart 时，ZPK 在根制品开启注册站点时，为每个主/子 Chart 统一设置 podAnnotations["w7.cc/inject-root-ca"]="true"，保留其他 Pod 注解；Chart 需通过 w7panel.podAnnotations helper 渲染 Pod 注解。子应用继承根制品注册站点开关；CA 注入不依赖 sidecar 名称或资源共享方式。
