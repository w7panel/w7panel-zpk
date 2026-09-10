# Helm Sidecar 集成约定

> 本文按当前工作树实现整理，描述 ZPK 在公式打包过程中如何下载、登记和渲染 Helm sidecar。最后核对日期：2026-09-09。

## 1. 核心结论

Helm sidecar 不是普通 `platform.depends`，而是一套独立的组合协议：

```text
远程 ZPK Helm 制品
  -> 最终 Chart 的 charts/<sidecar>/ 本地 dependency
  -> values.yaml 中的 w7panelSidecars 引用
  -> sidecar Chart.yaml annotations 指向 named templates
  -> 宿主 helper 把模板输出合并进 Workload、Job 或独立资源
```

当前只有一个自动触发来源：

```text
application.registerSite=true
  -> https://zpk.w7.cc/zpk/respo/info/w7panel-cloudnoauth
  -> w7panel-cloudnoauth
```

manifest 当前没有允许用户任意填写 sidecar 列表的字段。`registerSite=false` 时不会自动下载 sidecar，生成 Chart 中的 `w7panelSidecars` 默认为空。

## 2. 与普通应用依赖的区别

| 对比项 | Helm sidecar | 普通内嵌应用 |
| --- | --- | --- |
| 来源 | 当前由 `registerSite` 能力触发并使用代码内固定 info URL | 同一公式版本下已经保存的子 manifest |
| 是否读取 `platform.depends` 下载 | 否 | 当前打包器也不会仅凭 `depends.from` 即时下载 |
| Chart 形态 | 建议为 library Chart，导出模板片段 | 完整的 application Chart |
| 接入方式 | 宿主通过 `w7panel.*` helper 合并 | 作为独立子 Chart 渲染资源 |
| 最终安装 | 随主 tgz 本地安装，不再远程下载 | 随主 tgz 本地安装 |

普通子应用的实际打包输入是 `AllManifest/SubManifest`。远程导入流程应先下载附件并把子 manifest 保存到公式版本目录，再由 `generateSubCharts` 生成 `charts/<child>/`。

因此，只有下面这条记录并不能保证打包阶段自动下载 NGINX：

```yaml
depends:
  - identifie: w7-sitemanagernginx
    type: in
    from: https://zpk.w7.cc
```

环境应用需要确保 `w7-sitemanagernginx` 已经作为子 manifest 导入；打包器随后才会对这个子应用补充环境专属配置并生成子 Chart。

## 3. Sidecar 下载和打包流程

Sidecar 准备发生在原生、传统、环境、Helm 和系统镜像等应用类型分流之前，流程如下：

1. `requiredSidecarInfoURLs` 根据 `application.registerSite` 计算所需 sidecar，并按 `Chart` 名称去重。
2. 请求 sidecar 的 ZPK info URL，从响应中读取 `data.helm_url`；请求上下文超时为 2 分钟。
3. 下载 Helm tgz 到临时目录并解包。
4. 将解包后的 Chart 复制到主 Chart 的 `charts/<Chart>`；同名目录已存在时会被本次远程内容替换。
5. 将 `Chart` 记录到打包器的 `Sidecars` 列表。
6. 生成或改写 `values.yaml`，写入 `w7panelSidecars`。
7. 打包末尾扫描 `charts/*/Chart.yaml`，把 sidecar 登记成 `file://./charts/<目录名>` 本地 dependency。
8. 向主 Chart 注入 `_w7panel-sidecars.tpl` 和 `w7panel-sidecar-resources.yaml`。

最终目录示例：

```text
example-app/
├── Chart.yaml
├── values.yaml
├── charts/
│   └── w7panel-cloudnoauth/
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
└── templates/
    ├── _w7panel-sidecars.tpl
    ├── w7panel-sidecar-resources.yaml
    └── ...
```

用户提供 Helm tgz 时，解包采用目录合并方式，不会清空提前准备好的 sidecar。用户 Chart 原有 dependencies 会保留，再按 dependency name 或 repository 与扫描到的本地 dependency 合并。

完成打包后，sidecar 已包含在最终应用 tgz 内。安装阶段不再访问 sidecar info URL 或 `helm_url`。

## 4. 名称一致性要求

`HelmSidecar` 当前只有一个字段：

```yaml
chart: example-sidecar
```

这个名称同时参与：

- `charts/<chart>/` 复制目录；
- `values.yaml` 中的 `w7panelSidecars[].chart`；
- `.Subcharts[chart]` 查找；
- 主 Chart dependency 的匹配。

打包器不会重写 sidecar 自己的 `Chart.yaml.name`。dependency name 和 version 是从该文件读取的，所以以下内容必须保持一致：

```text
sidecar 制品标识
= 下载目标目录名
= w7panelSidecars[].chart
= sidecar Chart.yaml.name
= Helm dependency key
```

名称不一致时，打包可能仍能完成，但 Helm 渲染会因 `.Subcharts[chart]` 不存在或上下文错误而失败。

## 5. Sidecar Chart 契约

### 5.1 Chart.yaml

Sidecar 按协议应使用 Helm v2 library Chart，并声明 sidecar manifest 类型：

```yaml
apiVersion: v2
name: example-sidecar
type: library
version: 0.1.0
annotations:
  w7.cc/manifest-type: sidecar
  w7.cc/sidecar-container-template: example-sidecar.containers
```

选择 library Chart 是因为它只向宿主导出 named templates，不应自行渲染普通应用 Workload。

当前打包器不会强制验证以下内容：

- `type` 是否为 `library`；
- `w7.cc/manifest-type` 是否为 `sidecar`；
- annotation 指向的 named template 是否存在；
- named template 输出是否为要求的 YAML 结构。

这些是协议要求，但错误通常要到 `helm lint` 或 `helm template` 阶段才会暴露。

### 5.2 支持的 annotations

| Annotation | named template 输出 | 宿主使用位置 |
| --- | --- | --- |
| `w7.cc/sidecar-pod-annotations-template` | Pod annotations YAML map | Workload 和 Shell Job 的 Pod metadata |
| `w7.cc/sidecar-host-aliases-template` | `hostAliases` YAML 数组 | Workload；开启 Job 契约时也用于 Job |
| `w7.cc/sidecar-init-template` | init container YAML 数组 | Workload initContainers；开启 Job 契约时也用于 Job |
| `w7.cc/sidecar-container-template` | container YAML 数组 | Workload containers |
| `w7.cc/sidecar-job-container-template` | container YAML 数组 | Shell Job 的 initContainers，通常作为原生 sidecar init container |
| `w7.cc/sidecar-volumes-template` | volume YAML 数组 | Workload volumes；开启 Job 契约时也用于 Job |
| `w7.cc/sidecar-resources-template` | 一份或多份 Kubernetes YAML | 主 Chart 顶层独立资源 |

缺少某个可选 annotation 时，对应 helper 输出为空。`w7.cc/sidecar-container-template` 缺失时不会产生 Workload sidecar 容器。

### 5.3 named template 上下文

宿主通过 `.Subcharts[chart]` 获取 sidecar 上下文。因此 sidecar 模板内的 `.Values`、`.Chart` 及其他内置对象属于 sidecar 子 Chart，而不是主 Chart：

```yaml
# sidecar values.yaml
image:
  repository: example/sidecar
  tag: "1.0.0"
```

```gotemplate
{{- define "example-sidecar.containers" -}}
- name: example-sidecar
  image: {{ .Values.image.repository }}:{{ .Values.image.tag }}
{{- end -}}
```

## 6. 主 Chart values

主 Chart 只保存 sidecar 名称和顺序：

```yaml
w7panelSidecars:
  - chart: example-sidecar
  - chart: another-sidecar
```

顺序也是 annotations、containers、volumes 和 resources 的聚合顺序。不要再写旧版字段：

```yaml
# 当前版本不兼容这种旧写法
w7panelSidecars:
  - chart: example-sidecar
    containerTemplate: example-sidecar.containers
    initTemplate: example-sidecar.init
```

对于用户提供的 Helm Chart，只要存在自动 sidecar，`configureHelmSidecarHost` 会覆盖整个 `w7panelSidecars` key，而不是与用户原有列表合并。宿主若已有手工 sidecar 配置，需要在打包结果中重点检查这一项。

## 7. 宿主 Helper

| Helper | 作用 |
| --- | --- |
| `w7panel.sidecars.podAnnotations` | 只聚合 sidecar annotations |
| `w7panel.podAnnotations` | 合并宿主与 sidecar annotations |
| `w7panel.sidecars.hostAliases` | 聚合 Workload hostAliases |
| `w7panel.sidecars.initContainers` | 聚合 Workload init containers |
| `w7panel.sidecars.containers` | 聚合 Workload sidecar containers |
| `w7panel.sidecars.volumes` | 聚合 Workload volumes |
| `w7panel.sidecars.jobHostAliases` | 聚合启用 Job 契约的 hostAliases |
| `w7panel.sidecars.jobInitContainers` | 聚合 Job init template 和 job container template |
| `w7panel.sidecars.jobVolumes` | 聚合启用 Job 契约的 volumes |
| `w7panel.sidecars.resources` | 聚合独立 Kubernetes 资源 |

### 7.1 合并规则

- Pod annotations 按 `podAnnotations`、`annotations`、sidecar annotations 的顺序合并，同名 key 后写覆盖前写。
- 多个 sidecar 按 `w7panelSidecars` 顺序处理，后面的同名 annotation 覆盖前面的。
- hostAliases 按 IP 合并，hostname 去重，最终按 IP 字母序输出。
- hostAliases 条目缺少 IP 时直接 `fail`。
- 同一 hostname 对应不同 IP 时直接 `fail`，使 `helm template` 失败。
- containers、init containers 和 volumes 直接拼接数组，不检查重名，也不会执行 Kubernetes strategic merge。
- resources 忽略空输出，不同 sidecar 的非空输出用 `---` 分隔。

## 8. Workload 宿主接入

ZPK 生成型 Workload 已经接入这些 helper。用户提供的 Helm/K8sYaml 应用只会被注入 helper 和资源模板，ZPK 不会重写用户的 Deployment、StatefulSet、DaemonSet 或 Job。

用户 Workload 必须主动调用插槽，下面是结构正确的 Deployment 片段：

```gotemplate
spec:
  template:
    metadata:
      {{- $podAnnotations := include "w7panel.podAnnotations" . }}
      {{- if $podAnnotations }}
      annotations:
        {{- $podAnnotations | nindent 8 }}
      {{- end }}
    spec:
      {{- $sidecarHostAliases := include "w7panel.sidecars.hostAliases" . }}
      {{- if $sidecarHostAliases }}
      hostAliases:
        {{- $sidecarHostAliases | nindent 8 }}
      {{- end }}

      {{- $sidecarVolumes := include "w7panel.sidecars.volumes" . }}
      {{- if $sidecarVolumes }}
      volumes:
        {{- $sidecarVolumes | nindent 8 }}
      {{- end }}

      containers:
        - name: main
          image: example/main:1.0.0
        {{- include "w7panel.sidecars.containers" . | nindent 8 }}

      {{- $sidecarInitContainers := include "w7panel.sidecars.initContainers" . }}
      {{- if $sidecarInitContainers }}
      initContainers:
        {{- $sidecarInitContainers | nindent 8 }}
      {{- end }}
```

如果用户 Chart 没有调用这些 helper，`w7panelSidecars` 即使已经登记，sidecar 也不会自动进入用户 Workload。纯 YAML 输入同样不会被重写。

`templates/w7panel-sidecar-resources.yaml` 会独立调用 `w7panel.sidecars.resources`，所以不依赖 Workload 插槽的额外资源仍可以输出。

## 9. Shell Job 接入

Shell Job 使用独立的 sidecar helper：

- 只有声明 `w7.cc/sidecar-job-container-template` 的 sidecar 才进入 Job hostAliases、initContainers 和 volumes 聚合。
- `sidecar-init-template` 的输出先加入 Job `initContainers`。
- `sidecar-job-container-template` 的输出随后也加入 Job `initContainers`，不会加入普通 `containers`。
- Job 主任务仍是 `containers` 中的标准容器。
- Job Pod annotations 会使用 `w7panel.podAnnotations` 的合并结果，但移除 `sysbox/rootfs-rw-layer`。

Job sidecar 通常应使用 Kubernetes 原生 sidecar init container 语义：

```yaml
annotations:
  w7.cc/sidecar-job-container-template: example-sidecar.jobContainers
```

```gotemplate
{{- define "example-sidecar.jobContainers" -}}
- name: example-sidecar
  image: example/sidecar:1.0.0
  restartPolicy: Always
{{- end -}}
```

需要确认目标 Kubernetes 版本支持带 `restartPolicy: Always` 的原生 sidecar init container。否则该 Job 可能无法通过 API 校验或无法按预期结束。

## 10. 五类应用的接入差异

| 应用类型 | Workload sidecar | Shell Job sidecar | 说明 |
| --- | --- | --- | --- |
| 原生应用 | 已接入通用 Workload | 已接入公共 Shell Job | Deployment、StatefulSet、DaemonSet 共用模板 |
| 传统应用 | 没有常驻 Workload | 已接入公共 Shell Job | Sidecar 只能影响安装、升级、卸载等 Job |
| 运行环境 | 已接入环境 Workload | 公共 Shell Job 可接入 | 具体代码/生命周期 Job 取决于所用模板 |
| Helm/K8sYaml 应用 | 不自动接入用户 Workload | 不自动接入用户 Job | 用户模板必须显式调用 helper |
| 系统镜像 | 已接入通用 Workload | 已接入公共 Shell Job | Job 会移除 Sysbox rootfs 持久层 annotation |

仓库自身的 ZPK 部署 Chart（`charts/`）也包含相同 helper，且它的 Deployment 已调用 annotations、hostAliases、initContainers、containers 和 volumes 插槽。其 `charts/values.yaml` 默认使用空列表：

```yaml
w7panelSidecars: []
```

## 11. 存储边界

Sidecar 框架没有固定存储，也不自动创建、选择或回收 PVC。实际存储完全由 sidecar 模板决定：

- `sidecar-volumes-template` 可以输出 `emptyDir`、Secret、ConfigMap、PVC 等合法 Pod volume。
- container/init/job container template 需要自行输出与 volume 对应的 `volumeMounts`。
- 引用 PVC 时，PVC 必须由安装方、宿主 Chart 或 `sidecar-resources-template` 创建。
- Sidecar 框架不会自动解释或传递环境应用使用的 `PVC_NAME` 协议。
- Job 只有在 sidecar 声明 Job container template 时才会带入该 sidecar 的 volumes。
- `sidecar-resources-template` 可以输出 PVC、ConfigMap、Service、RBAC 等对象，但命名、升级、删除和 hook 语义由 sidecar 自己负责。

## 12. 当前限制和风险

1. 当前 sidecar 来源写死在代码中，manifest 不能任意声明。
2. 打包依赖远程 info 接口和 `helm_url`，网络或制品错误会阻断主应用打包。
3. library 类型、manifest 类型、named template 和输出结构不在打包阶段完整校验。
4. 用户 Helm Chart 的 workload/job 不会被自动改写，缺少 helper 插槽时只会登记依赖而不会注入容器。
5. Chart 名称、目录名和 values 引用必须一致，但当前没有前置一致性校验。
6. containers、volumes 和额外资源不检查重名，可能在 Kubernetes 校验或安装时失败。
7. 用户 Chart 原有 `w7panelSidecars` 会被自动列表整体覆盖。
8. Job sidecar 依赖目标 Kubernetes 对原生 sidecar init container 的支持。

## 13. 验收清单

至少执行：

```bash
helm dependency list <host-chart>
helm lint <host-chart>
helm template test-release <host-chart> --debug
```

重点检查：

- `charts/<sidecar>/Chart.yaml` 存在，名称和版本正确；
- 主 `Chart.yaml.dependencies` 已登记本地 sidecar；
- `values.yaml.w7panelSidecars[].chart` 与 dependency key 一致；
- Workload 中 annotations、hostAliases、initContainers、containers 和 volumes 的位置正确；
- volume 和 volumeMount 一一对应，引用的 PVC/Secret/ConfigMap 已存在或随 Chart 创建；
- 多个 sidecar 没有容器名、volume 名和资源名冲突；
- Shell Job 的 sidecar 位于 `initContainers`，并符合目标 Kubernetes 版本能力；
- 多个资源之间有正确的 `---` YAML 文档分隔符。

## 14. 主要代码位置

| 内容 | 文件 |
| --- | --- |
| Sidecar 来源、下载和 values 注入 | `app/respo/logic/helm/helm_sidecar.go` |
| 打包顺序与 dependency 生成 | `app/respo/logic/helm/helm_pack.go` |
| Sidecar 聚合 helper | `app/respo/logic/helm/helm_templates/_w7panel-sidecars.tpl` |
| Shell Job 接入 | `app/respo/logic/helm/helm_templates/shell-job.yaml.tpl` |
| Job annotations 处理 | `app/respo/logic/helm/helm_templates/_helpers.tpl` |
| 生成型 Workload 插槽 | `app/respo/logic/helm/helm_templates/workload.yaml.tpl` |
| ZPK 自身部署 Chart | `charts/templates/_w7panel-sidecars.tpl`、`charts/templates/deployment.yaml` |
| 五类应用总览 | `APPLICATION_HELM_PACKAGING.md` |
