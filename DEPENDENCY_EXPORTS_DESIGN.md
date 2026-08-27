# ZPK 依赖应用参数导出与共享 PVC 方案

## 1. 背景

当前 ZPK 已支持外部依赖应用，并具备以下能力：

- 依赖声明可以区分单实例和多实例。
- 安装端能够确定依赖应用实际绑定的 Helm `releaseName`。
- 主应用会生成隐藏的依赖 ReleaseName 启动参数。
- 安装界面可以通过 `/zpk/out-depends/env` 读取已安装依赖的环境变量。
- 主应用启动参数可以通过 `module_name` 和 `%PARAM%` 占位符引用依赖环境变量。

现有能力主要解决了“依赖哪个实例”和“读取依赖环境变量”的问题，但没有明确描述：

- 依赖应用允许向外暴露哪些参数。
- 主应用的参数是用户填写的，还是由依赖应用注入的。
- 参数名不一致时如何映射。
- 依赖实例变化或应用升级时，哪些值需要重新计算。
- 共享 PVC 时如何处理访问模式、调度亲和性和多实例隔离。

## 2. 设计目标

1. 依赖应用显式声明可导出的参数，避免向消费方暴露全部环境变量。
2. 主应用显式声明参数来源，区分用户输入和依赖注入。
3. 支持主应用参数名和依赖导出名不同。
4. 支持单实例和多实例依赖，并始终绑定到确定的 `releaseName`。
5. 支持导出 Service 地址、端口、普通启动参数和 PVC。
6. 更新应用或切换依赖实例时，可以正确刷新依赖注入值。
7. 共享 RWO PVC 时，将主应用和相关 Job 调度到依赖实例所在节点。
8. 保持现有未声明导出能力的 ZPK 可继续安装。

## 3. 依赖应用导出定义

在被依赖应用的 `platform` 下增加 `exportsStartParams`：

```yaml
platform:
  exportsStartParams:
    - name: MYSQL_HOST
      title: MySQL 服务地址
      type: value
      valuesText: "%HOST%"

    - name: MYSQL_PORT
      title: MySQL 服务端口
      type: value
      valuesText: "%PORT%"

    - name: MYSQL_PASSWORD
      title: MySQL 密码
      type: value
      valuesText: "%MYSQL_ROOT_PASSWORD%"
      sensitive: true

    - name: MYSQL_PVC_NAME
      title: MySQL 数据存储
      type: pvc
      valuesText: "%PVC_NAME%"
```

字段含义：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `name` | 是 | 对外稳定的导出名称 |
| `title` | 否 | UI 和错误信息中的显示名称 |
| `type` | 是 | 导出类型，第一阶段支持 `value`、`pvc` |
| `valuesText` | 是 | 从已安装依赖实例中解析值的表达式 |
| `sensitive` | 否 | 是否为敏感值，默认 `false` |

第一阶段支持以下系统内置值：

| 占位符 | 含义 |
| --- | --- |
| `%HOST%` | 依赖应用的集群内 Service 地址 |
| `%PORT%` | 依赖应用的默认服务端口 |
| `%RELEASE_NAME%` | 实际绑定的 Helm ReleaseName |
| `%NAMESPACE%` | 依赖应用所在命名空间 |
| `%PVC_NAME%` | 默认 PVC 名称，第一阶段仅支持单 PVC |
| `%XXX%` | 依赖应用已安装配置中的启动参数 `XXX` |

`exportsStartParams` 是白名单。`/zpk/out-depends/env` 不应再默认将所有 Helm values 或容器环境变量作为公共协议暴露给新制品。

## 4. 主应用依赖注入定义

主应用继续使用 `startParams`，并增加可选的 `dependencySource`：

```yaml
platform:
  startParams:
    - name: DATABASE_HOST
      title: 数据库地址
      required: true
      dependencySource:
        moduleName: w7-mysql
        exportName: MYSQL_HOST
        allowOverride: false

    - name: DATABASE_PASSWORD
      title: 数据库密码
      required: true
      dependencySource:
        moduleName: w7-mysql
        exportName: MYSQL_PASSWORD
        allowOverride: false

    - name: DATABASE_PVC_NAME
      title: 数据库存储
      required: true
      dependencySource:
        moduleName: w7-mysql
        exportName: MYSQL_PVC_NAME
        allowOverride: false
```

字段含义：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `moduleName` | 是 | 依赖模块标识，应和 `depends` 中的依赖对应 |
| `exportName` | 是 | 依赖应用 `exportsStartParams.name` |
| `allowOverride` | 否 | 是否允许用户覆盖自动注入值，默认 `false` |

规则：

- 存在 `dependencySource`：该启动参数来源于依赖应用。
- 不存在 `dependencySource`：该启动参数是普通用户输入或系统参数。
- 参数匹配必须使用 `moduleName + exportName`，不能仅按参数名匹配。
- 主应用参数 `name` 无需和依赖导出 `name` 相同。
- 多实例依赖必须使用安装流程已经绑定的 `releaseName` 查询导出值。

## 5. 安装界面行为

安装界面加载配置后执行以下流程：

1. 根据主应用 `depends` 确定依赖实例和 `releaseName`。
2. 调用 `/zpk/out-depends/env` 查询指定 namespace、依赖标识和 `releaseName`。
3. 获取依赖应用的 `exportsStartParams` 及解析后的导出值。
4. 按 `dependencySource.moduleName + dependencySource.exportName` 匹配主应用参数。
5. 自动填充主应用参数。
6. 根据 `allowOverride` 决定是否允许编辑。
7. 安装时仍将最终值写入主应用的 `envKv`。

建议 UI 状态：

- 必选依赖未安装：阻止安装，并提示先安装依赖。
- 可选依赖未安装且允许覆盖：允许用户手工填写。
- 依赖已安装且不允许覆盖：自动填充并锁定输入框。
- 依赖实例切换：重新解析该依赖对应的所有注入参数。
- 解析失败：显示具体的依赖名称和导出参数名称，不静默使用空值。

## 6. 参数值优先级

普通参数和依赖注入参数需要使用不同的更新策略。

普通启动参数：

```text
显式安装参数 > 已安装版本保存值 > 制品默认值
```

依赖注入参数：

```text
允许覆盖时的显式安装参数 > 当前绑定依赖的导出值 > 制品默认值
```

当 `allowOverride: false` 时，更新应用应重新读取当前绑定依赖，而不是直接沿用主应用上一次保存的解析结果。

## 7. 来源关系持久化

安装完成后不能只保存解析后的字符串，还需要保存来源关系：

```yaml
dependencyBindings:
  w7-mysql:
    releaseName: w7-mysql-abc123
    namespace: default

dependencyInjectedParams:
  DATABASE_HOST:
    moduleName: w7-mysql
    exportName: MYSQL_HOST
    releaseName: w7-mysql-abc123
  DATABASE_PVC_NAME:
    moduleName: w7-mysql
    exportName: MYSQL_PVC_NAME
    releaseName: w7-mysql-abc123
```

这些信息可以保存在 Helm release values、AppGroup spec 或稳定的资源 annotation 中，具体位置由面板安装模型统一决定。

持久化来源关系用于：

- 应用升级时重新解析依赖值。
- 判断当前值是用户输入还是依赖注入。
- 切换依赖实例时刷新相关参数。
- 共享 PVC 时生成正确的实例级 affinity。
- 展示应用当前绑定的依赖实例。

## 8. 共享 PVC

### 8.1 基本约束

- PVC 是 namespace 级资源，主应用和依赖应用必须位于同一 namespace。
- PVC 的生命周期由导出它的依赖应用持有。
- 卸载主应用不能删除依赖应用的 PVC。
- 依赖应用必须显式通过 `type: pvc` 允许其他应用引用该 PVC。
- 第一阶段仅支持依赖应用导出一个默认 PVC。
- 后续多 PVC 应改为具名导出，不应继续使用单一 `PVC_NAME`。

面板后端拿到导出的 PVC 名称后，应查询 Kubernetes PVC 的真实 `spec.accessModes`，不能相信制品中手工填写的访问模式。

### 8.2 访问模式处理

| 访问模式 | 处理方式 |
| --- | --- |
| `ReadWriteMany` | 可以跨节点挂载，不生成共享存储 affinity |
| `ReadOnlyMany` | 通常不需要 affinity |
| `ReadWriteOnce` | 主应用和相关 Job 必须调度到依赖 Pod 所在节点 |
| `ReadWriteOncePod` | 不允许被另一个应用 Pod 共享，安装前报错 |

如果 PVC 同时声明多种 access mode，应以实际存储驱动和最严格的共享约束为准，不能只取数组第一项。

## 9. RWO PVC 的实例级 affinity

RWO PVC 共享时，主应用必须匹配具体依赖实例，而不是依赖的应用标识：

```yaml
affinity:
  podAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: w7.cc/group-name
              operator: In
              values:
                - w7-mysql-abc123
        topologyKey: kubernetes.io/hostname
```

其中 `w7-mysql-abc123` 来自依赖绑定的实际 `releaseName`。

不能使用：

```yaml
key: w7.cc/identifie
values:
  - w7-mysql
```

原因是同一依赖应用可能安装多个实例。按应用标识匹配时，调度器可能选择另一个实例所在节点，导致 RWO PVC 挂载失败。

为了支持实例级 affinity，依赖 Workload 的 Pod template 必须带有：

```yaml
metadata:
  labels:
    w7.cc/group-name: {{ .Release.Name }}
```

当前仅在 Workload 对象 metadata 上设置该 label 不够，PodAffinity 匹配的是 Pod label。

## 10. Job 的 affinity

任何挂载共享 RWO PVC 的 Job 都必须应用同一份依赖实例 affinity，包括：

- `pre-install`
- `post-install`
- `pre-upgrade`
- `post-upgrade`
- 自定义 shell Job

Job 不能只依赖主应用 Pod 的 affinity，原因是 `pre-install` Job 执行时主应用 Pod 可能还不存在。

因此，只要 Job 挂载了来自依赖的 RWO PVC，其 `jobAffinity` 就应直接匹配依赖的 `w7.cc/group-name=<releaseName>`。

没有挂载共享 RWO PVC 的 Job 继续使用现有调度策略。

## 11. 接口建议

`GET /zpk/out-depends/env` 保留现有安装状态字段，并增加解析后的 exports：

```json
{
  "installed": true,
  "name": "w7-mysql-abc123",
  "releaseName": "w7-mysql-abc123",
  "namespace": "default",
  "exports": {
    "MYSQL_HOST": {
      "type": "value",
      "value": "w7-mysql-abc123.default.svc.cluster.local",
      "sensitive": false
    },
    "MYSQL_PORT": {
      "type": "value",
      "value": "3306",
      "sensitive": false
    },
    "MYSQL_PVC_NAME": {
      "type": "pvc",
      "value": "w7-mysql-abc123-data",
      "accessModes": ["ReadWriteOnce"],
      "sensitive": false
    }
  }
}
```

兼容期内可以继续返回现有 `envs`、`pvcName` 等字段，旧版 UI 仍使用旧字段，新版 UI 优先使用 `exports`。

## 12. 敏感参数

第一阶段可以沿用当前启动参数传值方式，但需要做到：

- `sensitive: true` 的值在 UI 中使用密码输入框。
- 不在日志和错误信息中打印实际值。
- 接口调试输出中进行脱敏。
- 不在非必要的 annotation 中保存明文。

长期建议支持导出 Secret 引用，而不是导出密码明文：

```yaml
type: secretKeyRef
secretName: mysql-auth
key: root-password
```

## 13. 兼容策略

- 没有 `exportsStartParams` 的旧依赖应用继续使用当前 `envs` 替换逻辑。
- 没有 `dependencySource` 的旧主应用继续使用 `module_name + %PARAM%` 逻辑。
- 新制品存在 `dependencySource` 时优先使用新 exports 协议。
- 新协议解析失败时不能悄悄退回同名环境变量，避免引用错误依赖实例或错误参数。
- 更新旧应用时保留其历史值，不强制转换为依赖注入来源。

## 14. 不采用的方案

### Helm `lookup` 自动发现依赖资源

不建议使用 Helm `lookup` 搜索依赖 Service 或 PVC：

- 多实例时可能匹配到错误资源。
- 离线 `helm template` 无法获得结果。
- Helm 执行账号可能没有查询权限。
- 资源名称或 label 变化会引入隐式兼容问题。

依赖订单和安装流程已经掌握准确的 `releaseName`，应由面板后端解析并传入。

### 根据 releaseName 拼接资源名

不建议使用 `releaseName + 固定后缀` 推算 PVC 或 Service 名称。资源名是依赖制品的实现细节，可能在新版本中发生变化，应使用依赖显式导出的实际名称。

## 15. 分阶段实施

### 第一阶段：参数导出和自动注入

- ZPK manifest 增加 `exportsStartParams`。
- `StartParams` 增加 `dependencySource`。
- 打包结果保留导出定义。
- 面板后端解析 exports 并扩展 `/zpk/out-depends/env`。
- UI 根据依赖来源自动填充并处理是否允许覆盖。
- 保存依赖绑定和参数来源关系。

### 第二阶段：单 PVC 共享

- 支持 `type: pvc` 和 `%PVC_NAME%`。
- 后端查询 PVC access mode。
- RWX/ROX 直接使用。
- RWO 为 Workload 和相关 Job 注入实例级 affinity。
- RWOP 在安装前拒绝共享。
- Pod template 增加 `w7.cc/group-name` label。

### 第三阶段：多 PVC 和 Secret 引用

- PVC 改为具名导出，例如 `data`、`backup`。
- 支持 Service 多端口导出。
- 支持 `secretKeyRef`、`configMapKeyRef` 类型。
- 增加依赖实例切换和重新绑定界面。

## 16. 验收场景

至少覆盖以下测试：

1. 单实例依赖自动注入 Service 地址和端口。
2. 多实例依赖按绑定的 `releaseName` 获取正确 exports。
3. 主应用参数名与依赖 export 名不同也能正确映射。
4. 不允许覆盖的参数在 UI 中锁定。
5. 可选依赖未安装时允许填写 fallback。
6. 应用升级时普通参数保留，依赖参数重新解析。
7. RWX PVC 不生成 affinity。
8. RWO PVC 生成匹配具体依赖 release 的硬 affinity。
9. 挂载 RWO PVC 的 pre-install Job 能直接匹配依赖实例。
10. RWOP PVC 在安装前被拒绝。
11. 两个相同标识的依赖实例不会发生 affinity 串实例。
12. 旧 ZPK 在没有新字段时保持原安装行为。
