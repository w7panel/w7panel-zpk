# w7panel-zpk 开发指引

## 范围与协作

- 本文件适用于本仓库及子模块；子目录存在更具体的 `AGENTS.md` 时，在对应范围遵循其指引。
- 默认中文沟通，先区分解释 / 审查与实施要求；仅检查问题时不自动修改代码。
- 开始先查看 `git status --short`、相关差异和文档，保留用户的修改、未跟踪文件与暂存状态；不擅自提交、暂存、回退或清理。
- 保持任务范围和现有架构；跨市场、面板、控制台的协议变更需要说明兼容性，不自动修改其他仓库。

## AI 编码阅读入口

- 本文件提供默认加载的项目规则与核心逻辑摘要。分析或修改代码时，先阅读 [AI_CODING_GUIDE.md](AI_CODING_GUIDE.md) 的相关模块导航，再按任务阅读 [AI_CODE_LOGIC.md](AI_CODE_LOGIC.md) 的具体调用链、UI 状态转换与失败分支；纯文档或简单局部任务按需读取。
- 以 `provider.go → controller → logic → service / DAO` 追踪后端，以 `router → views → components → utils → API` 追踪 UI；不能只根据文件名或依赖列表推断架构。
- 两份参考文档是代码扫描快照，不是测试通过证明。协议修改同时核对专项文档、当前实现和相关测试；发现不一致先说明，不自行新增字段或扩大兼容降级。
- 保留历史目录和协议拼写 `respo`、`identifie`。区分数据库版本 ID、版本字符串、登录 token、下载 token 和安装 ticket，不能因命名相似而互换。

## 项目结构

- 主 Go 模块为 `github.com/w7panel/w7panel-zpk`，Go 版本见 `go.mod`（当前 1.25.0），使用 Gin、Rangine、GORM、OCI / ORAS、Helm 和 Kubernetes SDK。
- `main.go`：配置、数据库切换保护、SDK 和 provider 初始化；`app/respo/`：制品管理；`app/registry/`：仓库接口与权限；`app/system/`：用户、OIDC、存储和数据库管理。
- 各模块的 `provider.go` 注册路由 / 命令，`http/controller/` 处理输入和响应，`logic/` 实现业务。
- `app/respo/logic/formula/`：制品 manifest、版本、附件、依赖和安装 ticket；`logic/helm/`：Chart 生成和动态打包；`logic/goods/`：商品发布；`logic/zpkmarket/`：市场订单及运行配置。
- `common/service/`：W7 SDK、OCI、registry 和本地 / S3 附件存储；`common/entity/`、`common/dao/` 为模型及生成查询，`common/accessor/` 为字段类型；`database/table.yaml` 为生成配置。
- `cli/` 与 `tradition-plugin/` 各有独立 `go.mod`，需分别测试；CLI 通过本地 replace 引用主模块。根目录的测试不会自动覆盖两个子模块。
- `ui/` 是 Vue 3、Vue Router、Arco Design、JavaScript / Vue CLI 前端；实际状态主要通过组件 data/computed/watch、props、事件与 utils 管理。虽声明 Vuex 依赖，目前 `ui/src` 未见 store 使用，不假定已有统一状态仓库；根 `package.json` 不是业务前端。
- `charts/` 是 ZPK 服务自身的部署 Chart；制品运行 Chart 的生成逻辑在 `app/respo/logic/helm/`，不要混淆两者。

## 后端核心逻辑摘要

- **启动有写入副作用**：`main.go` 加载配置及数据库切换状态、初始化 DAO/SDK，再注册三个 Provider。Provider 可能迁移数据库、规范化制品名、初始化默认用户/命名空间、恢复后台任务。不能为查看帮助或验证启动直接使用真实配置运行服务。
- **制品加载不是纯查询**：`Depot.GetFormula` 聚合数据库元数据、版本文件、制品级 BaseInfo 与子 manifest，可能从 OCI 恢复附件并异步恢复图标。传入非管理员 user 才在此处限制所属用户；传 nil 的调用者须有自己的准入规则。轻量选择器优先沿用 `SelectableList` / `FormulaInstallOnlyOnce`。
- **数据分层**：版本目录保存根/子 manifest；制品级设置、简介和图标独立维护；完整 manifest 不只存在数据库中。根 manifest 保存会应用 BaseInfo。`FormulaSetting.Get` 在旧记录缺 BaseInfo 时可能写回；设置中的 `*bool` 区分省略与显式 false。
- **保存与发布分阶段**：版本发布先同步打包 OCI，按设置同步市场商品，再进入 `publish_loop.go` 后台构建 Helm。接口“发起打包成功”不等于全部发布完成；保留 pending=1、success=2、fail=3、未发布/下架=-1 与历史已发布状态 0 的语义。同制品按队列串行、任务去重，并在启动恢复 pending。
- **跨存储无统一事务**：文件、数据库、OCI、市场及 Kubernetes 操作可能部分成功。修改前明确阶段、失败返回和补偿方式；不能把尽力恢复说成原子回滚。版本下架还会选择前一已发布版本并重新发布。
- **安装 info 多模式**：`Formula.Info` 区分身份/token、普通安装/升级、`runtime_context` 和 `full_manifest`。订单可能切换目标制品或限定版本；运行上下文不是安装授权结果，完整导入响应不能按普通响应裁剪。预检、ticket 与订单核销中的域名、应用标识、升级/重装信息必须一致。
- **文件与附件协议**：manifest/file 与 share-file 分开，后者禁止 manifest；Depot 空内容表示删除。复用路径规范化和范围校验。上传返回的 `file://` 是逻辑引用；一次性下载映射命中即删除，重试可能需要重新获取 info。
- **集群配置有真实副作用**：Registry 存储设置会更新 ConfigMap 并触发 Deployment rollout，可能更新 CronJob；部署规则在 push 事件中按 tag 匹配并 Patch 容器镜像。不能把这些接口当作普通配置保存或隔离验证入口。

## UI 核心逻辑摘要

- **启动与身份**：`main.js` 支持 Wujie/独立挂载；`App.vue` 通过宿主 access_token 登录 OIDC，再获取当前用户并经 router-view props 传递。`user-mixin.js` 只控制展示，不能替代后端权限。
- **请求统一入口**：复用 `utils/index.js`、`request-base.js`、`request-auth.js`、`panel-token.js`。区分宿主 access_token、本地 `X-Zpk-Token` 和代理用 `X-W7Panel-Token`；跨来源请求保留 token 发送限制。过期登录共用进行中的 Promise，原请求仅重试一次，登录请求禁止递归刷新。
- **错误传播需核对**：当前 Axios 对部分 408/429/422 分支返回 undefined，页面的部分 publish catch 仅提示不重新抛错；不能假设所有异常都会 reject，或 Promise resolve 就代表业务成功。改动时追踪调用方 then/catch 与跳转、成功提示。
- **编辑器三层状态**：`respo-edit.vue` 管文件树、主子应用和持久化；`files-manifest.vue` 组合 `components/manifest/*.js` 管表单。流程为 YAML → json → form → apply/serialize → json → YAML → complete 事件。新增字段同时核对初始化、watch、类型切换、提交及重新加载；保留 `_initializing` 与保存快照，避免初始化误改和未保存内容丢失。
- **提交顺序**：表单校验 → 同步类型依赖 → 保存类型设置/分类 → 清理并序列化 manifest → 父页面写文件 → 按路径继续编辑或发布。传统应用还要确保工具子应用存在。基础信息弹框依次保存 BaseInfo、readme、分类、可选图标，均不是单次事务。
- **类型与局部编辑**：切换类型经确认后清理旧类型专属配置；网关插件 once=true，传统/系统镜像 once=false。`pureManifest` 避免写制品级设置；`startParamsOnly` 克隆原 manifest 后仅更新启动参数，不能用默认表单覆盖导入字段。
- **子应用导入**：后端按选定版本下载完整 manifest 与来源隔离附件；前端 `utils/child-app-import.js` 先保存全部子文件并检查 allSettled 结果，再写根依赖，失败时尽力恢复旧文件和组件状态。保留 required/from/version、order、更新检查 token 和主应用未保存保护。
- **菜单配置**：`respo-editfront.vue` / `files-manifestfront.vue` 把角色、菜单、后端地址/端口、代理参数与 frontend props 转为 bindings；`getMenu` 按 iframe/internal/external/网关插件分流。端口候选来自主子 manifest；`${system.*}` 等占位符留给运行时注入，不替换为当前浏览器值。
- **上传与异步清理**：`files-upload.vue` 计算全文件 MD5，按 1 MiB 顺序上传，多分片后另发 finish；当前 abort 为空实现。保留响应嵌套和后端分片参数。新增异步逻辑检查过期响应、timer、事件监听和 object URL 的清理。

## 开发规范

- 请求绑定和响应沿用控制器及 Rangine 的约定，业务放在 logic，外部 API 放在 service；新路由保留对应身份、权限和签名中间件。
- Go 改动使用 `gofmt`、处理错误、及时释放文件 / 响应体，复用已有函数和常量；避免复制逻辑和无关重构。
- 在边界校验输入与远程响应，注意上下文、网络超时和失败恢复；不能把外部调用失败当作成功，也不能随意扩大兼容性降级范围。
- 修改 `*.gen.go` 前先核对数据库结构、`database/table.yaml` 与 accessor，优先使用项目既有生成流程，不进行无关的手工格式化或重构。
- 数据库逻辑兼顾 MySQL 和 SQLite 以及现有数据库切换 / 禁写保护，不未经授权运行迁移命令或操作工作目录中的真实数据库。
- 前端复用既有组件、状态传递、utils 和请求方式；不为局部功能改用 Vite、TypeScript 或另一套 UI 库。

## respo/info HTML 提示职责与契约

先按以下职责划分确定修改位置，再处理协议细节：

| 层级 | 负责内容 | 开发约束 |
| --- | --- | --- |
| ZPK | 判断安装 / 升级授权与绑定冲突；生成业务标题、文案、风险说明、导航链接和允许的按钮 HTML；执行后端授权与身份校验 | 业务提示内容统一在 `formula_notice.go` 维护；新增授权原因或调整提示内容优先修改 ZPK，不要求 server / UI 按原因补写文案或专用弹框 |
| w7panel-server | 向 ZPK 声明 HTML 能力；校验远程响应状态、错误码、结构与长度；转发约定的提示和冲突字段；保留本地请求及身份校验 | 沿用 `repo.go` 与 `app/zpk/http/zpk.go` 的协商 / 转发入口，不拼接提示 HTML、购买链接，不根据文案猜测业务类型，不因收到按钮自动重装 |
| 面板 UI 的 `repo-notice.ts` | 识别 ZPK 提示协议，适配通用弹框；持有本地安装重试上下文，执行重装二次确认 | 只承载协议适配和重装交互，不为购买、VIP、试用、退款等原因重新拼接 ZPK 的提示内容 |
| 面板 UI 的 `html-notice.ts` | 通用 HTML 净化、Shadow DOM 展示、安全链接、底部操作区、关闭按钮、去重和动作执行锁 | 不引入 ZPK 业务错误码、授权原因或重装规则；只执行调用方明确允许的本地动作回调 |

- HTML 描述业务提示与可选操作，不承载授权结果或执行权限。ZPK 决定是否允许安装 / 升级及是否提供重装入口，server / UI 保留各自校验；按钮展示、点击或二次确认均不能替代后端校验。
- **面板不过多干预 ZPK 的业务错误提示逻辑。** 对协议合法、可安全展示的提示，server 转发 ZPK 返回的内容，UI 经通用净化后展示；不根据 `reason`、错误文案、商品或订单状态重新判断提示含义，不覆盖标题、正文、风险说明、按钮文案和导航地址，不增删业务提示或另行拼接专用弹框。新增购买、VIP、试用、退款等提示分支应由 ZPK 维护，面板不为每个分支增加业务判断。
- 面板的必要干预限于协议 / 长度校验、HTML 和 URL 安全过滤、通用展示，以及已支持动作的本地上下文与权限约束（如重装二次确认）。只有响应无法按协议处理或净化后无有效内容时才使用纯文本兜底；不能以统一提示为由替换有效的 ZPK 业务内容，也不能为了展示完整而放宽安全规则。
- 文案、购买地址、风险说明或已有动作的展示变化由 ZPK 维护；只有响应协议、HTML 白名单或本地动作能力变化时才需要核对 server / UI 的配套实现。跨仓库修改先说明职责和兼容性影响，按已授权范围实施。

- 修改安装 / 升级授权错误或绑定冲突提示前，核对 `app/respo/http/controller/formula.go`、`formula_notice.go`、`formula_notice_test.go` 和 README 的“安装授权 HTML 提示”；同时只读核对面板仓库 `~/Develop/go/w7panel/w7panel-ui/src/utils/html-notice.ts` 与 `repo-notice.ts`（实际文件名不是 `respo-notice`）。本仓库 `ui/` 不是该面板前端，不自动修改其他仓库。
- HTML 是 JSON 错误响应中的 `data.html` 片段，不是直接返回 `text/html` 页面。只有查询参数严格为 `notice_format=html` 才启用：授权限制返回 HTTP 403、字符串 `code: ZPK_HTML_NOTICE`，保留纯文本 `error`，以及 `data.title/html/reason/goods_id/order_sn/trial_expire_at/restriction/market_url`。不要把网络、未知市场错误或所有 500 错误统一转成 HTML 提示。
- 未声明 HTML 能力时保持旧协议：试用到期为 HTTP 403 / `ZPK_TRIAL_EXPIRED`，其他授权限制沿用原 HTTP 500 / 数字错误码及具体纯文本原因；绑定冲突无论是否协商均保持 HTTP 409、数字 `code: 409` 和原结构化字段，协商后仅补充 `data.title/html`。面板 `repo-notice.ts` 仅识别约定的 403 提示，以及 `domain_mismatch` / `app_identify_exists` 的 409 提示。
- 展示文案、导航和操作放在 `data.html`；不恢复独立 `actions` 字段或为每种业务增加专用弹框。操作区使用 `<div data-notice-footer>…</div>`，由面板净化后移入弹框底部，与面板原生“关闭”按钮并排展示；HTML 不自行实现关闭按钮。面板通用 `html-notice.ts` 负责白名单净化、Shadow DOM 隔离、重复错误去重及纯文本兜底，输入长度上限为 64 × 1024 个 JavaScript 字符。
- HTML 只使用前端白名单支持的标签和内联排版样式，不依赖脚本、事件属性、iframe、表单、外部样式或远程 CSS 变量。动态文案、字段和链接属性须在 ZPK 端转义；URL 必须校验为不含用户名密码的完整 HTTP(S) 地址，不能仅靠 HTML 转义或前端过滤。
- 市场链接复用 `formulaAccessMarketURL`：保留部署路径前缀，按 Hash 路由生成购买 / 续费 `/#/site-detail/{goods_id}`、VIP `/#/vip`、退款订单 `/#/user-center/orders?order_sn=…`；订单号正确编码，无有效市场地址时只显示文字。原面板卸载链接复用 `formulaOriginalPanelURL`，沿用 `/app/apps?uninstallApp=…`。导航使用安全的新窗口链接。
- 强制重装只使用 `<button data-panel-action="reinstall">`，仅应用标识绑定冲突且非升级时由 ZPK 输出；域名冲突和升级不能提供该动作。面板还须具备本地 `onReinstall` 重试上下文，并通过固定二次确认；HTML 不能授予权限、执行任意请求或绕过后端 `reinstall` / 升级身份校验。空绑定字段显示“未提供”，订单信息优先使用市场返回值。
- 跨版本兼容保持能力协商：面板后端声明参数，旧 ZPK 可忽略未知参数；前后端分开部署时先更新支持 HTML 的面板前端，再更新声明能力的后端。市场授权拒绝按既有数字业务码识别，不匹配错误文案；未知码和旧市场纯文字错误保持原错误处理，不扩大降级范围。

## 制品与打包契约

- 变更 manifest、应用类型、依赖、PVC、生命周期或 Chart 生成前，阅读 `APPLICATION_HELM_PACKAGING.md` 中相关章节；涉及 Sidecar 时同时阅读 `HELM_SIDECAR.md`。实现与文档矛盾时先说明，不擅自新增协议字段。
- 应用类型包括 `docker`、`app-plugin`、`tradition`、`helm`、`system-image` 和独立的 `gateway-plugin`。打包优先级为网关插件 → 存在 Helm 输入配置 → 应用插件 → 传统应用 → 通用 Workload，不能只凭 type 推断。`SidecarApp` 常量不代表支持任意 sidecar manifest；按实际分流与存储边界处理。
- `application.order` 原样写入 MicroApp 的 `w7.cc/order` 标签，主应用为 0，子应用按列表排序为 1...n，市场 MicroApp 为 9999；不根据 `platform.depends` 重写顺序。
- 市场绑定遵循 README：无 Binding 时返回原 Helm 包；有 Binding 时按原包状态及 Bindings 内容缓存动态包。市场域名进入 `backend_config[role=zpk-market].backend_url`，菜单 `do` 保持路由语义。
- 市场菜单使用已有 MicroApp 字段及 founder 可见性；网关 WasmPlugin 与配置 MicroApp 使用相同 `w7.cc/group-name` 标签归组，不恢复已废弃的关联注解。
- 子应用导入读取所选版本的完整 manifest，附件存储键包含来源；全部子文件保存成功后再更新主 manifest，失败保持 / 恢复原状态。
- 安装 ticket 中的 `domain` 和规范化 `app_identify` 需保持校验及传递；保留绑定冲突的 HTTP 409 与结构化信息。`reinstall=true` 只用于允许的非升级覆盖，不放宽升级身份校验。
- 依赖、Sidecar 和 PVC 变更须检查生成 values、模板、挂载路径、生命周期与传统应用恢复行为；优先在临时目录生成 / 渲染验证，不安装到真实集群。
- 路径、压缩包解压、远程下载和附件操作要验证目标范围，防止路径穿越与同名来源覆盖；保留已有权限和签名校验。

## 验证命令

- 完整测试矩阵、Helm 渲染验证要求和已知静态检查问题见 `TESTING.md`；新增测试工作流覆盖三个 Go 模块与 UI 请求单测。

先跑受影响包的测试，再根据修改范围扩大；以下 Go 命令从对应模块根目录运行。

- 主模块：`go test . ./app/... ./common/...`；静态检查 `go vet ./...`；编译检查 `go build ./...`。
- `cli/`：`go test ./...`、`go vet ./...`、`go build ./...`。
- `tradition-plugin/`：`go test ./...`、`go vet ./...`、`go build ./...`。根 `make test` 只覆盖主模块和 CLI，涉及 tradition-plugin 时需额外执行其测试。
- 并发改动对相关包运行 `go test -race`；单测使用隔离文件 / 数据库，外部请求优先使用 mock 或 `httptest`。
- 修改 HTML 提示协议时运行 `go test ./app/respo/http/controller -run 'TestFormula(AccessNotice|RefundNotice|InstallConflictNotice|ConflictNotice)' -count=1`，覆盖能力协商、旧协议、绑定冲突、升级禁重装、字段转义和 URL 安全；涉及面板端时核对其 `scripts/repo-html-notice.test.cjs` 的过滤、底部操作区和动作确认测试，遵循该仓库指引，不因本仓库文档调整自动修改面板。
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
