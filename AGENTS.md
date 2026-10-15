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
