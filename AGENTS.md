# w7panel-zpk 开发指引

## 范围与协作

- 本文件适用于本仓库及子模块；子目录存在更具体的 `AGENTS.md` 时，在对应范围遵循其指引。
- 默认中文沟通，先区分解释 / 审查与实施要求；仅检查问题时不自动修改代码。
- 开始先查看 `git status --short`、相关差异和文档，保留用户的修改、未跟踪文件与暂存状态；不擅自提交、暂存、回退或清理。
- 保持任务范围和现有架构；跨市场、面板、控制台的协议变更需要说明兼容性，不自动修改其他仓库。

## 项目结构

- 主 Go 模块为 `github.com/w7panel/w7panel-zpk`，Go 版本见 `go.mod`（当前 1.25.0），使用 Gin、Rangine、GORM、OCI / ORAS、Helm 和 Kubernetes SDK。
- `main.go`：配置、数据库切换保护、SDK 和 provider 初始化；`app/respo/`：制品管理；`app/registry/`：仓库接口与权限；`app/system/`：用户、OIDC、存储和数据库管理。
- 各模块的 `provider.go` 注册路由 / 命令，`http/controller/` 处理输入和响应，`logic/` 实现业务。
- `app/respo/logic/formula/`：制品 manifest、版本、附件、依赖和安装 ticket；`logic/helm/`：Chart 生成和动态打包；`logic/goods/`：商品发布；`logic/zpkmarket/`：市场订单及运行配置。
- `common/service/`：W7 SDK、OCI、registry 和本地 / S3 附件存储；`common/entity/`、`common/dao/` 为模型及生成查询，`common/accessor/` 为字段类型；`database/table.yaml` 为生成配置。
- `cli/` 与 `tradition-plugin/` 各有独立 `go.mod`，需分别测试；CLI 通过本地 replace 引用主模块。根目录的测试不会自动覆盖两个子模块。
- `ui/` 是 Vue 3、Vue Router、Vuex、Arco Design、JavaScript / Vue CLI 前端；根 `package.json` 不是业务前端。
- `charts/` 是 ZPK 服务自身的部署 Chart；制品运行 Chart 的生成逻辑在 `app/respo/logic/helm/`，不要混淆两者。

## 开发规范

- 请求绑定和响应沿用控制器及 Rangine 的约定，业务放在 logic，外部 API 放在 service；新路由保留对应身份、权限和签名中间件。
- Go 改动使用 `gofmt`、处理错误、及时释放文件 / 响应体，复用已有函数和常量；避免复制逻辑和无关重构。
- 在边界校验输入与远程响应，注意上下文、网络超时和失败恢复；不能把外部调用失败当作成功，也不能随意扩大兼容性降级范围。
- 修改 `*.gen.go` 前先核对数据库结构、`database/table.yaml` 与 accessor，优先使用项目既有生成流程，不进行无关的手工格式化或重构。
- 数据库逻辑兼顾 MySQL 和 SQLite 以及现有数据库切换 / 禁写保护，不未经授权运行迁移命令或操作工作目录中的真实数据库。
- 前端复用既有组件、Vuex / utils 和请求方式；不为局部功能改用 Vite、TypeScript 或另一套 UI 库。

## 制品与打包契约

- 变更 manifest、应用类型、依赖、PVC、生命周期或 Chart 生成前，阅读 `APPLICATION_HELM_PACKAGING.md` 中相关章节；涉及 Sidecar 时同时阅读 `HELM_SIDECAR.md`。实现与文档矛盾时先说明，不擅自新增协议字段。
- 应用类型包括 `docker`、`app-plugin`、`tradition`、`helm`、`system-image`；按现有分流与存储边界处理，不能套用单一打包路径。
- `application.order` 原样写入 MicroApp 的 `w7.cc/order` 标签，主应用为 0，子应用按列表排序为 1...n，市场 MicroApp 为 9999；不根据 `platform.depends` 重写顺序。
- 市场绑定遵循 README：无 Binding 时返回原 Helm 包；有 Binding 时按原包状态及 Bindings 内容缓存动态包。市场域名进入 `backend_config[role=zpk-market].backend_url`，菜单 `do` 保持路由语义。
- 市场菜单使用已有 MicroApp 字段及 founder 可见性；网关 WasmPlugin 与配置 MicroApp 使用相同 `w7.cc/group-name` 标签归组，不恢复已废弃的关联注解。
- 子应用导入读取所选版本的完整 manifest，附件存储键包含来源；全部子文件保存成功后再更新主 manifest，失败保持 / 恢复原状态。
- 安装 ticket 中的 `domain` 和规范化 `app_identify` 需保持校验及传递；保留绑定冲突的 HTTP 409 与结构化信息。`reinstall=true` 只用于允许的非升级覆盖，不放宽升级身份校验。
- 依赖、Sidecar 和 PVC 变更须检查生成 values、模板、挂载路径、生命周期与传统应用恢复行为；优先在临时目录生成 / 渲染验证，不安装到真实集群。
- 路径、压缩包解压、远程下载和附件操作要验证目标范围，防止路径穿越与同名来源覆盖；保留已有权限和签名校验。

## 验证命令

先跑受影响包的测试，再根据修改范围扩大；以下 Go 命令从对应模块根目录运行。

- 主模块：`go test . ./app/... ./common/...`；静态检查 `go vet ./...`；编译检查 `go build ./...`。
- `cli/`：`go test ./...`、`go vet ./...`、`go build ./...`。
- `tradition-plugin/`：`go test ./...`、`go vet ./...`、`go build ./...`。根 `make test` 只覆盖主模块和 CLI，涉及 tradition-plugin 时需额外执行其测试。
- 并发改动对相关包运行 `go test -race`；单测使用隔离文件 / 数据库，外部请求优先使用 mock 或 `httptest`。
- 前端在 `ui/`：`npm run lint -- --no-fix src/相关文件.vue`、`npm run build`；保留现有 dist 时使用 `npm run build -- --dest <临时目录>/dist`。
- 前端 CI 当前使用 npm；安装前检查现有锁文件，不混用 npm / yarn 或无故更新依赖锁文件。
- 修改服务部署 Chart 时可运行 `helm lint ./charts` 与 `helm template zpk ./charts`；修改制品 Chart 生成器时运行相关 Go 模板测试，并检查实际生成的 Chart，不能只验证服务 Chart。
- 全量检查失败时区分已有问题与本次引入的问题，报告命令与失败点，不随手修改无关代码，也不声称未执行的测试通过。

## 构建、发布与安全

- 根 `make build` 包含 clean / tidy，使用 Linux amd64 musl 交叉编译工具链生成 `builder/server`，不是普通本机编译检查；`make dev` 也会先执行 clean。
- `make publish` / `make beta` 会推送镜像并重打 Helm 包；GitHub `v*` 标签会触发 release 工作流。无明确授权不推送、打发布标签、部署或安装应用。
- CLI 批量构建会 tidy 模块；`build-compress` 依赖 UPX。只需编译验证时优先用 Go 原生命令，不触发整套发布流程。
- 不清理或覆盖用户的 `zpk.db`、`runtime/`、附件、证书、`builder/`、`cli/bin/`、`*.tgz` 或 `ui/dist/`。构建产物变更仅在任务要求时进行。
- 不读取 / 输出私钥内容，不在日志、指引或提交中写入 Cookie、令牌、密码及签名密钥；`registry-key.pem` 等本地证书不应因任务而提交。
- `ui/vue.config.js` 支持 `VUE_PROXY_TARGET`，启动前确认代理目标，不能把现有代理默认地址视为隔离环境。
- 交付说明改动、验证和兼容性影响；打包规则变化同步维护相关文档，但不能声称已部署或已发布，除非实际获得授权并完成。
