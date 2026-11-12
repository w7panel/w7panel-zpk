# 镜像自动任务

镜像仓库是工作流的首个调用方。模块集中管理规则、执行日志、Helm 包和执行流程；entity、DAO 仍统一放在 `common/entity`、`common/dao`，复用原规则表和日志表，没有新增表。镜像版本表只增加 `is_rebuilt` 保存当前重构状态。

logic 沿用 respo 的结构体方法风格：`Workflow` 管理配置、触发及运行记录，`Helm` 管理模板打包及 release 安装、卸载和超时清理，`Executor` 管理排队调度、运行参数及执行状态，`Deploy` 承接规则增删改查、集群配置与旧部署兼容；配置校验和任务匹配分别放在 `Config`、`Task` 的方法中，日志和结果仍放在 workflow；控制器只处理参数、权限和响应，Kubernetes 查询、补丁及代理由 `common/service.Kubernetes` 调用。

registry 在推送入口直接组成源镜像引用，生成一个仓库令牌并传入工作流，再通过 `ImagePushed` 传给 `Workflow.OnImagePushed`。workflow 只维护规则、执行记录和构建结果日志，返回 `ImagePushResult.Rebuilt`，由 registry 单独更新镜像版本的重构标记。事件入口不包裹整体数据库事务；工作流入队保留短事务，Helm 安装在后台执行。源镜像和令牌随配置快照加密保存，执行时直接使用，不重新获取镜像地址或签发令牌。没有 `Source` 适配器或仓库锁注入。保存和入队不加仓库锁；调度器在短事务内锁定本仓库最早的工作流协调记录，再检查运行状态并领取任务，保证同仓库流程串行，Helm 执行期间不持锁。镜像表及 `IsRebuilt` 的迁移归 registry Provider，workflow Provider 只处理规则和执行日志。registry 根据返回结果调用旧部署兼容入口，内部构建回推不再次触发。

Helm 包根文件由 `helm_templates/Chart.yaml.tpl` 和 `values.yaml.tpl` 在保存配置时生成，默认镜像及运行参数结构统一在 values 模板维护；`serviceaccount.yaml.tpl`、`configmap.yaml.tpl`、`role.yaml.tpl`、`rolebinding.yaml.tpl`、`secret.yaml.tpl`、`job.yaml.tpl` 原样打包到 Chart 的 `templates/`，在触发安装时由 Helm 使用本轮 values 渲染。`helm.go` 中每种资源对应一个 `generate*Yaml` 方法，`GenerateChart` 只负责组织各生成步骤及打包。

## 配置与执行

仓库详情的“镜像自动任务”页面先填写任务名称，再配置触发方式和版本名称，再选择“部署应用”或“镜像重构”。执行时机合并为一个下拉框，直接选择“最先执行”“在某任务之后执行”或“和某任务一起执行”；列表显示对应关系，点击修改即可调整，新增、修改或删除任务时直接保存完整配置并更新 Helm 包，无需再次保存列表。保存成功后更新列表；失败保留原列表和编辑内容。

只有触发方式、匹配方式和版本条件相同的任务才能设置先后、并行或引用构建结果；更新版本不比较无效的匹配方式字段，版本条件只忽略首尾空白。列表按这些条件归组，顺序调整只影响本组；不同前缀或正则即使可能命中同一版本，也不允许互相引用。旧配置中跨条件的镜像来源在保存和触发时会被拒绝，需要重新选择。

“之后执行”等待所选任务所在组全部成功，“一起执行”加入所选任务的执行组。界面将选择转换为原有 stage 字段，已有数字顺序配置自动回显，不新增接口或数据库字段；这些关系用于配置分组，不是独立的任务依赖图。镜像来源可选择原始镜像或构建结果，不再单独选择来源任务：选“之后执行”时使用所选构建任务的输出，所选任务为部署时沿用其输入；选“一起执行”时沿用所选任务的输入。已有配置未调整时保留原来源，避免改变历史镜像语义。调整时同时校验已有任务的来源引用，禁止引用同组或后续任务的构建结果。

- 相同顺序构成一个阶段，在同一个 Job 的不同容器中并行运行；不同阶段按 Helm hook 权重串行执行。
- 当前阶段所有任务成功才进入下一阶段；失败后停止后续阶段，同阶段已启动任务继续运行。
- 镜像来源为“原始镜像”或指定前面阶段的构建任务。来源通过 digest 固定，不使用含义不明确的“最近一次执行”或可变 tag。
- 来源任务必须匹配本次事件；来源缺失会记录失败，不自动改用原镜像。
- 构建目标为原始 tag 加自定义后缀；留空覆盖原 tag。后缀始终相对触发 tag 计算。
- 同阶段不能写同一个 tag 或同一个部署容器；不同阶段可以连续修改同一 tag。
- 每个工作流最多 32 个任务，单阶段最长 30 分钟，一轮执行最长 2 小时；每个服务进程最多同时运行两轮。同一仓库的执行记录通过数据库抢占串行推进。

例如：阶段 1 构建基础版；阶段 2 并行构建虚拟机版和开发版；阶段 3 的部署任务明确选择阶段 2 中某条任务的结果。

保存配置时从 `logic/helm_templates/*.tpl` 生成真实 Helm `.tgz`，沿用 ZPK 本地 Helm 根目录，每个仓库只有一个当前包：`{setting.depot.storage.local.path}/Helm/workflow/{仓库ID}/workflow.tgz`。编辑时通过同目录临时文件写完后原子替换，不再按配置摘要累积多个当前版本。

触发时在数据库事务内读取规则、匹配任务，并把当前包复制为 `Helm/workflow/{仓库ID}/runs/{运行ID}.tgz` 后入队；不会重新生成包。临时执行包禁止覆盖，后续编辑不会影响排队或执行中的流程。执行只加载本轮包，丢失或损坏直接失败，不回退到当前包。规则、日志均不新增 Helm 路径字段，路径分别按仓库 ID、运行 ID 计算。包的 Chart 注解保存不含凭证的配置摘要，只用于校验包与触发配置一致，不用于目录命名。

执行结束（成功或失败）删除本轮临时包；调度器同时重试终态清理，保留 pending/running 对应的包。事务失败或进程中断留下的无执行记录文件，超过三小时后清理。文件与数据库仍不构成跨存储原子事务：如果包已替换但数据库提交失败，触发时会明确报告配置与包不一致，需重新保存任务配置；不会用错包。多副本需共享本地 Helm 目录。

旧摘要目录不会被本次逻辑主动删除。升级此未发布结构后需重新保存任务配置；旧排队记录缺少本轮临时包时明确失败，不使用当前包替代。历史执行日志仍保留，已完成的临时包不用于重放。

构建参考面板现有 Kaniko 执行镜像，直接挂载 Dockerfile 片段，不要求源码压缩包。第一版不提供 `FROM`、`ONBUILD`、`ADD`、`COPY`，不提供自定义脚本动作。按当前约定不再执行镜像来源预检：不检查多平台索引，也不检查排队期间原 tag 是否发生变化。构建始终使用触发时的 digest；构建过程与外部上传之间没有跨系统事务，覆盖原 tag 时仍需避免并发外部上传。

## 数据职责

规则表保存动作、执行分组顺序和来源引用，不保存 Helm 包路径或二进制。旧规则在首次保存为工作流时保留数据库 ID，转换为部署动作；未转换规则继续使用旧部署入口。转换后旧修改接口拒绝改写，避免使 Helm 缓存失效。

日志表同时保存每轮执行记录（数据库 `workflow_task_key` 为空）和对应规则的任务记录。执行快照和 KUBECONFIG 使用项目已有 AES 机制加密；运行快照不随规则编辑变化，历史日志不因任务删除而删除。旧日志的 `workflow_run_id` 保持 SQL NULL，与新日志的唯一键兼容。

规则和日志均删除 `workflow_chart_path`，也不保存包二进制和重复的 `selected` 列。选中任务从同仓库、同 `workflow_run_id` 的任务记录推导。初始化 SQL 和 DAO 已同步；未对真实数据库执行迁移或删除列。若此前试用了未发布的存包结构，需按旧配置重新保存包后再清理旧列；启动不会自动丢弃旧列数据。

日志只保存构建输出 tag 和 digest；镜像版本表 `ims_registry_repository_tag.is_rebuilt` 保存当前 tag 是否为重构产物。每个构建任务使用只允许当前仓库 pull/push 的独立签名身份；Webhook 必须能匹配执行快照中的任务和目标 tag 才认定为内部重构。内部推送标记重构状态，不另起自动任务；外部上传清除状态，即使 digest 相同也会清除，不修改历史构建产物记录。版本列表通过仓库模块 `Repository.RebuiltImages` 查询镜像版本表的状态，展示“重构”标签。工作流只负责在推送确认后更新状态。

新增字段使用业务语义命名：规则中的 `workflow_task_key` 为稳定任务标识、`execution_order` 为并行分组顺序、`source_build_task_key` 为指定来源构建任务；日志中的 `trigger_*` 保存触发镜像，`build_output_*` 保存构建产物，`encrypted_config_snapshot` 保存加密配置。Go 字段与数据库列名同步优化，原 JSON 字段名保留，兼容现有页面和接口。规则和日志均不保存执行命名空间，任务执行、日志读取和资源清理统一使用 `default`。部署目标仍使用 `k8s_namespace`。这些新增字段尚未发布，若本地已试用旧命名，需要先转换数据；初始化只添加缺失列，不自动重命名或删除旧列。

## 部署与权限

实际执行使用当前集群 ServiceAccount，不读取开发机器的 kubeconfig。执行命名空间固定为 `default`，不提供环境变量或页面配置；服务 Helm Chart 将工作流 Role 和 RoleBinding 创建在 `default`，绑定 ZPK 所在命名空间的 ServiceAccount。

服务 Chart 的 `workflow.enabled` 控制执行器及相关权限，`workflow.buildImage` / `workflow.deployImage` 可指定执行镜像。部署镜像默认使用 `registry.cdn.w7.cc/alpine/k8s:1.33.0`，包含 `/bin/sh` 和 kubectl；自定义部署镜像也需同时提供这两者。旧服务的 `WORKFLOW_DEPLOY_IMAGE` 或 Helm `workflow.deployImage` 若仍指向无 shell 的镜像，需更新后重新触发。构建 Job 不使用宿主 Docker socket。工作流的 Job、ConfigMap、ServiceAccount、Role、RoleBinding 以及认证 Secret 均由模板定义，执行端只准备本轮 values 并调用 Helm。镜像令牌通过 `values.registryToken` 传入，模板生成仓库认证 Secret，挂载至构建容器 `/kaniko/.docker`；目标集群 KUBECONFIG 通过 `values.kubeconfigs[任务ID]` 传入，模板生成集群 Secret，挂载至部署容器 `/target/config`。两类 Secret 随 release 一起清理，后端不再单独创建或删除；可复用的模板包不包含凭证。本轮凭证随 values 和 Secret manifest 保存在执行期间的 Helm release 中。配置摘要包含模板契约版本，之前试用的旧格式配置需重新保存任务配置。

当前集群部署仅管理员可以配置；其他用户必须提供目标集群 KUBECONFIG。目标集群配置只接受内嵌证书、Token 等，不执行外部认证程序，也不引用本地凭证文件。任务 ServiceAccount 仅获结果 ConfigMap 读取/更新权限，以及管理员明确配置的目标工作负载读取/更新权限。

ZPK 的 ServiceAccount 需要在执行命名空间管理 Job、Secret、ServiceAccount 和读取 Pod 日志；还需创建、删除任务所需的 Role / RoleBinding。使用自定义 ServiceAccount 时需自行配置这些权限。Kubernetes 对 Role 创建/绑定的权限提升校验继续生效。

每轮完成后清理临时凭证、普通 Chart 资源和 Helm release 元数据。hook Job 保留一天以查看 Pod 日志，然后由 Kubernetes TTL 清理。异常退出不会自动重放已有副作用；超过恢复窗口的运行记录标为失败，需要检查集群后重新触发。

SQLite、MySQL 非 debug 启动沿用项目初始化规则，以增量方式添加字段。业务查询和更新统一使用生成 DAO 的字段表达式，保留 GORM 事务和行锁，不拼接数据库列名。数据库初始化统一由 provider 处理，只创建缺失表、添加缺失字段及索引，不改写旧列；debug 环境仍跳过自动补齐，数据库切换禁写期间也跳过结构变更，不提供额外迁移命令。本次开发没有对真实数据库执行迁移。

## 接口与兼容

新接口保留仓库调用路径，路由、控制器及业务实现由本模块管理：

- `POST /zpk/v2/api/repository/workflow/get`：读取配置。
- `POST /zpk/v2/api/repository/workflow/save`：保存整组任务并更新 Helm 包。
- `POST /zpk/v2/api/repository/workflow/runs`：可传 `task_id`，查询该任务最近 50 次执行；省略时保留整轮执行查询。
- `POST /zpk/v2/api/repository/workflow/logs`：读取指定执行的 Pod 日志，`tail_lines` 默认 100，范围 1～2000，返回 `log/archived/truncated/status`。

均传入 `repository_id`；保存额外传 `config`，日志额外传 `run_id`。仓库模块注入所属用户校验及签名令牌生成方法。旧 `deploy_rule` 路由保留，控制器已迁入本模块；旧模型与表名保持兼容。

任务列表每行提供“执行记录”入口，弹窗显示该任务历史，不再在页面底部展示仓库执行记录。“任务日志”使用记录的 `run_id` 和 `task_id`，只显示当前任务在该轮的日志。列表显示任务 ID，执行记录显示流程 ID；日志窗口采用黑底终端和顶部控制栏，支持全屏。参照面板安装日志提供 50、100、200、500、1000、2000 行选择，以及“跟踪”开关。本模块沿用共享 JSON 请求入口，跟踪模式每次请求完成 3 秒后更新一次快照，并自动滚动；关闭跟踪后只读取一次。切换行数、关闭弹窗或组件卸载会取消旧请求并清理定时器，错误时暂停跟踪。

Pod 尚在时按选择的行数读取，每个容器最多显示 1 MiB，达到上限时明确提示。Pod 已清理后的历史归档仍是每个容器末尾 100 行、最多 16 KiB；归档响应明确标记并停止跟踪，选择更多行不会恢复已丢弃日志。此功能不是全量日志存储。

## 验证

```sh
go test ./app/workflow/... ./app/registry/http/controller
go test -race ./app/workflow/...
go vet ./app/workflow/... ./app/registry/http/controller ./common/dao ./common/entity
go build ./...
```

模块测试使用临时 SQLite 和实际 `helm template`，覆盖阶段并行、指定来源、输出冲突、旧规则/日志兼容、快照、事件去重及内部/外部推送识别。不连接真实集群；尚未进行真实 Kaniko 构建及集群安装验证。

任务结果以容器退出状态为准，并用 Helm 已完成的 hook 结果保留前面执行组的成功状态。仅确认未启动的任务标记 skipped；无法采集或确认的结果标记 unknown，不把采集失败当作跳过。流程失败不覆盖已确认的任务成功结果。

部署脚本读取构建结果和更新当前集群应用时，显式指定当前集群 API 地址、ServiceAccount CA 和 token；更新第三方集群应用时显式指定 `/target/config`。两条连接路径独立，不依赖工具镜像内的默认 kubeconfig，也不回退到 localhost:8080。该脚本模板更新后需重新保存任务配置，再触发新流程；已固定的运行包不自动替换。
