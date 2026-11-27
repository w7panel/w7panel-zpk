# 文档系统

本目录作为文档系统的项目根目录，后端和 UI 不引用宿主 ZPK 的业务代码。当前使用根 Go module 管理依赖，通过本目录的后端入口独立启动。ZPK 主服务不注册文档 Provider，不连接文档数据库或初始化文档正文存储。

## 构建与部署包

在本目录运行以下命令，默认产物写入 `builder/`。后端编译、前端和 Chart 打包可通过 `BUILD_DIR` 指定其他绝对目录；镜像构建使用文档根目录作为上下文，读取默认的 `builder/server`：

```sh
make build                 # Linux amd64 musl，依赖 x86_64-linux-musl-gcc/g++
make dockerbuild           # 编译后构建 document 镜像
make ui-deps               # 按 package-lock.json 安装前端依赖
make webzip                # builder/document-ui.zip，根目录直接包含 index.html
make chart                 # builder/w7-zpkdocument-0.1.0.tgz
make lint CHART_PACKAGE=/绝对路径/平台最终包.tgz
make test
```

镜像构建只复制文档二进制。文档前端源码仍保留在本目录，主项目通过 `@document-ui` 构建别名直接引用，与主 UI 共用 Vue、Arco 等依赖，最终只发布主前端包。`make publish` 构建后端镜像和 Chart 并推送镜像；文档 UI 由主项目 `ui/` 统一构建；`make beta BETA_SUFFIX=beta1` 在版本后添加后缀。镜像仓库和版本可通过 `IMAGE_REPOSITORY`、`IMAGE_TAG` 覆盖，Chart 版本可通过 `HELM_CHART_VERSION` 覆盖；这些目标不自动向 ZPK 上传制品附件。首次使用前需要发布实际镜像，默认的 `zpk.w7.cc/public/zpk-document:v0.1.0` 仅为构建发布目标。

`charts/` 是文档后端自己的独立 application Chart，默认单副本、Recreate 升级，SQLite 与正文共同保存在 `/home/document/data` 的 PVC 中。`persistence.existingClaim` 可使用已有 PVC；`persistence.storageClass` 留空使用集群默认 StorageClass，`-` 表示显式不使用 StorageClass。不要让多个文档实例共用同一个 SQLite 文件。 Deployment 的 `affinity` 使用 `tpl` 渲染，支持 ZPK 打包器为共享存储注入的 `{{ .Release.Name }}` 等表达式，安装时转换为实际 Release 名称。

ZPK 制品标识为 `w7-zpkv2`，按单次安装规则使用相同的 Release 名，当前 Chart 的 Service 名也是 `w7-zpkv2`，端口为 8000。按部署在 default 命名空间的约定，`zpk.baseURL` 和 `permission.serverURL` 均默认使用完整地址 `http://w7-zpkv2.default.svc.cluster.local:8000`，不需要安装时传入 `--set`。地址仅包含 origin，接口路径由 SDK 拼接；若单独部署或修改了 ZPK 的命名空间、Service 名或端口，可在 values 中调整。权限服务当前由 ZPK 提供，`permission.serverURL` 显式留空时跟随 `zpk.baseURL`。`service.port` 为文档 Service 端口，后端容器固定监听 8001。Chart 不直接提供前端静态服务或手写 MicroApp，文档菜单配置在主应用的 bindings 中，使用主 MicroApp 打开 `#/documents`。

文档制品使用 `application.identifie=w7-zpkdocument`、`application.once=true`，Chart 名也使用 `w7-zpkdocument`，使面板传入的子应用 values 与 dependency 名一致。安装时面板将其 `fullnameOverride` 设为 `w7-zpkdocument`，文档 Service 使用这个名称。ZPK Chart 通过环境变量 `DOCUMENT_BASE_URL` 默认注入 `http://w7-zpkdocument.default.svc.cluster.local:8001`，用于 ZPK 调用文档 SDK；两端均无需额外传入地址参数。仅原 Chart 的命名无法代替制品 manifest 的单次安装配置，发布文档制品时需按此配置。

### Sidecar 接入

Chart 只保留 Sidecar 调用位置和 values 占位，`_w7panel-sidecars.tpl` 及资源入口由制品打包器自动注入，源码中不重复维护。Pod 已接入 annotations、hostAliases、initContainers、containers、volumes 插槽，配置 Sidecar 后自动添加根 CA 注入注解。Deployment 和 Pod 均使用当前 Release 的 `w7.cc/group-name`，`w7panelArtifact.serviceAccountName` 与 Pod 使用相同账号 helper。原 Chart 可以直接打包；lint 和渲染需使用平台注入模板后的最终包，原包直接渲染会缺少 `w7panel.*` helper。

原包保留 `w7panelSidecars: []` 占位；将 Chart 和前端 ZIP 发布为 Helm 类型制品，作为 ZPK 的子应用导入后，根制品开启 `application.registerSite`，由现有打包流程注入 cloudnoauth library、Sidecar 列表及共享资源配置。原包仅接好插槽，没有内置 cloudnoauth 或创建 Site；直接安装原包不具备签名凭据来源。共享子应用的 `sharedSidecarResources=true` 由打包器设置，根 Chart 统一输出共享资源和账号 Binding。

文档初始化立即经代理请求 `http://api.w7.cc/api/app/info` 获取凭据，目前请求失败会退出，需由 Kubernetes 重启策略在代理及 Site 就绪后重启；探针不会阻止主进程启动。这里不改变现有 SDK 初始化逻辑。主子应用需属于同一 Release/AppGroup，使用同一 Site 的 AppID/AppSecret。

### GitHub 发布流程

推送 `v*` 标签触发主 `Release` workflow，由其调用 `.github/workflows/release-document.yml`。文档子 workflow 使用相同标签构建 Linux 镜像，沿用主流程的 `IMAGE_PUSH_REGISTRY`（默认 `ghcr.io`），推送到 `<推送仓库>/<小写 GitHub owner/repo>-document:<标签>`，通过 `GITHUB_TOKEN` 登录，不再向 ZPK 镜像仓库推送。Helm 包沿用 `HELM_PACKAGE_IMAGE_REGISTRY`（默认 `ghcr.registry.cdn.w7.cc`）作为镜像拉取地址，Makefile 通过 `HELM_PACKAGE_IMAGE_REPOSITORY` 接收该地址。生成对应版本的 `w7-zpkdocument-<版本>.tgz`，通过 GitHub Actions artifact 返回主流程。

主流程等待文档构建成功后，拉取现有 `w7-zpkv2` 制品，通过 CLI 的 `app update --app=w7-zpkdocument` 更新已添加子应用的后端 Helm 附件，文档 UI 通过主项目构建引用编译进主前端附件，与主应用附件一起推送并发起发布。根 manifest 的依赖、子应用后端部署配置、单次安装配置及 manifest 版本保持已有值；缺少该子应用时终止，不推送这次主制品。版本处理与 ZPK 主项目一致：镜像 tag 和 Helm appVersion 使用 Git 标签，Chart version 去掉标签前缀 `v`，制品版本由现有 `app push --publish` 流程分配，不额外修改子应用 manifest。文档构建不会单独发布远端 `w7-zpkdocument` 制品，最终后台 Helm 打包状态仍需在制品管理端确认。

## 业务与页面归属

| 业务 | 后端入口 | UI 页面 | UI 路由 |
| --- | --- | --- | --- |
| 文档库、文件及下载 | `app/document/provider.go` | `ui/src/views/libraries.vue` | `/documents` |
| 面板自动登录及当前用户 | `app/document/provider.go`，controller 和 logic 各自的 `login.go` | `ui/src/App.vue` | 页面启动时自动认证 |

主 UI 通过嵌套路由加载本目录原有 `App.vue` 和文档库页面，沿用文档自动登录及 JWT。文档请求仍拼接 MicroApp 代理地址与 `/doc`，由 ZPK 根路由 `/doc/*path` 转发到配置的 `setting.document.base_url`，保留路径、查询参数、认证头、响应状态及下载响应；不套用 ZPK 用户认证中间件，资源身份与权限继续由文档后端验证。`/doc/openapi/*` 不通过浏览器代理开放，ZPK 的 `DocumentSdk` 仍直接调用文档服务。整体下载 Ingress 保持原样。

文档库管理入口为 `#/documents`，不提供分类管理、分类筛选或分类选择。创始人查看全部文档库，普通用户只查看自己的文档库。页面通过面板注入的 `access_token` 自动登录，不提供手动登录表单。

文档库名称下方提供整体下载地址及复制按钮。文档子应用通过启动参数引用 ZPK 主项目的 `DOMAIN_URL`，文档 Ingress 将该域名下的 `/doc` 直接转发到文档 Service，地址由文档自身生成，公私权限及打包也由文档服务处理；私有库需携带 `X-Document-Token` 或 `X-W7Panel-Access-Token`，地址不含凭据。未配置主域名时不创建文档 Ingress，获取下载地址报错，不影响列表加载。ZPK 不提供文档下载转发或查询地址接口。

文档子应用启动参数配置如下，`module_name` 指向主应用，`values_text` 按主应用现有的 `%DOMAIN_SSL_URL%` 域名占位符匹配，参数名无需与主应用相同：

```json
{ "name": "domainURL", "title": "主应用域名", "values_text": "%DOMAIN_SSL_URL%", "module_name": "w7-zpkv2", "hidden": true }
```

`domainURL` 接收面板传入的纯域名（例如 `zpk.example.com`），写入文档 Chart values，并由 Deployment 注入 `DOMAIN_URL`。文档 Ingress 和整体下载地址固定使用 HTTPS；独立运行可直接设置环境变量 `DOMAIN_URL=zpk.example.com`。只接收纯域名，不接收协议、端口、凭据、路径、查询参数或片段。HTTPS 使用与主 Chart 一致的 `<Release.Name>-tls-secret`；Ingress class 为 `higress`，路径使用 Prefix 匹配，不重写路径。

`ui/src/components/workbench.vue` 是本系统文档库页面使用的文档操作台，新建文件直接使用库内根路径，不固定添加 `docs/`。ZPK 的 `description.vue` 通过 `/respo/document/path-tree` 和 `/respo/document/file` 请求制品后端，由调用方 SDK 转接读取、保存、删除及排序；ZPK UI 不引用本目录代码。文档路径直接使用 `guide.md`、`manual/guide.md` 等库内相对路径，不添加 `docs/` 前缀。`.order` 仅为制品编辑器的排序字段，由适配层转换成文档系统的排序请求；空内容沿用制品编辑器的删除语义。制品简介通过 `/respo/setting/get` 和 `/respo/setting/set` 读写制品表的 `readme` 字段，旧 `/respo/share-file/*` 接口已移除；文档库中的 `readme.md` 不会覆盖制品简介。默认 `index.md` 保留在文档库根目录，在文档系统 UI 查看，不映射成可编辑的制品文档。

操作台维护目录、草稿、文件切换、保存和下载状态，Markdown 编辑及预览由 `ui/src/components/markdown-editor.vue` 负责。后端文档库、文件和下载业务分别放在 `app/document/logic/library.go`、`file.go`、`download.go`，`document.go` 仅保留公共错误、内容大小常量和引用解析。请求参数的格式、长度、取值与路径校验统一由 controller 的 `Validate` 处理，自定义验证规则在 Provider 注册阶段显式注册；logic 保留权限、关联存在性与业务保护检查，不重复校验参数。

制品默认首页由 ZPK 根据当前有效 manifest 生成：按应用类型说明用途，列出已配置能力、可见启动参数、容器端口、访问入口、面板菜单、数据挂载和依赖，并分别描述内嵌子应用。首页不输出参数值、环境变量值、脚本、源码下载地址或网关配置内容。首页前面的“应用介绍”保留制品简介的 Markdown 排版。初始化以及版本发布时，通过 SDK 更新受保护的 `index.md`；发布使用本次发布版本的 manifest，页面发布和下架后重新发布前一版本均执行首页更新，CLI 发布流程保持原样，已有普通文档不变。首页同步失败会返回错误并中止后续发布，可重试。配置、基础信息和简介保存不触发首页同步，统一在版本发布时更新；发布与文档同步不是事务，同步失败会中止后续发布并返回错误，可再次发布重试。

文档系统只提供通用的 `POST /doc/library/create`，接收库属性及可选的初始文件列表，不内置制品初始化流程或默认文件名。普通页面只提交标识、名称、描述和公有/私有设置，创建空库。制品端负责生成默认文档数据，通过 SDK 的服务入口 `POST /doc/openapi/library/create` 提交相同数据：

```json
{
  "ref": "app/demo",
  "name": "示例应用",
  "owner_id": 42,
  "public": 0,
  "protected": 1,
  "files": [
    {"path": "index.md", "title": "首页", "content": "# 示例应用\n\n应用标识：demo\n", "protected": 1}
  ]
}
```

库级 `protected=1` 表示库不可删除，文件级 `protected=0` 可正常编辑删除，`protected=1` 表示在普通用户接口不可修改、改名或删除，`protected=2` 表示可编辑和修改标题但不可删除，两者独立，不绑定文件名。`owner_id` 省略时使用当前用户，仅管理员可为其他用户建库。新建时保存所有者用户名快照 `username`，当前用户建库取认证信息中的用户名；管理员代建时由调用方通过 `owner_name` 提供所有者用户名，制品推送从 ZPK 用户记录获取。用户名用于列表展示，权限仍按 `user_id` 判断。库与初始文件元数据在同一事务内创建，重复标识返回 409。制品端遇到库已存在时，直接通过 `POST /doc/openapi/file/save` 更新本次推送的首页，不再查询库信息和文件列表作复用确认，不修改库的所有者、公私状态或其他文档。签名服务保存接口可显式传入 `protected` 维护受保护文件，仍校验签名请求中 `user_id` 对文档库的维护权限；普通保存接口不能通过传入该字段绕过保护。正文存储与数据库不共享事务，数据库失败时正文可能保留。

## 调用边界

- 数据模型和生成 DAO 放在本目录的 `common/entity`、`common/dao`，业务直接使用 DAO。模型定义由 `database/schema.sql`、`database/table.yaml` 驱动；在本目录运行 `go run ./tools/generate`，生成过程使用临时数据库。
- ZPK 用户 SDK 在 `common/service/zpk`，携带前端提供的面板 `access_token` 请求 `POST /zpk/v2/openapi/user/current`，仅通过 OIDC 中间件验证面板凭据，不挂 App、CORS 或域名限制中间件；复用 `CurUserInfo`，在 `data.user` 中返回用户 ID、用户名、角色和管理员标记。文档直接使用返回结果建立独立会话，不再重复请求面板或计算角色。登录和 JWT 签发、验证逻辑统一在 `app/document/logic/login.go` 的 `Login` 中。ZPK SDK 由 `main.go` 调用 `service.InitSDK` 初始化，登录逻辑和认证中间件直接使用 `service.ZPKSdk`，不通过 Provider 传递，也不增加用户转发层。`common/service/permission` 是权限服务 SDK，公有/私有和所有者的操作权限均通过远程接口判断；文档只维护默认首页不可修改等业务保护规则。
- ZPK 请求文档的 SDK 位于宿主 `common/service/w7/document`，复用 `w7/base` 签名。`FormulaDocument.Initialize` 使用 `app/制品标识` 创建文档库并推送名称、所有者、首页及保护标记，不再创建分类。
- `protected` 是库和文件的通用保护属性，不记录来源，普通用户接口中创始人也遵循保护限制。制品端提交库及 `index.md` 的保护属性，通过签名服务保存接口维护首页；普通制品文档推送及旧文档迁移均显式提交 `protected=2`，只允许编辑，不允许删除。文档管理中新建的文件默认 `protected=0`，仍可编辑删除。文档后端不按来源或 `index.md` 文件名作特殊处理。已有文件只有在再次推送时才更新保护属性，本次不自动改写历史数据。
- 元数据保存在数据库，正文通过 `storage.Store` 接口访问，本版启动时选择本机文件驱动并采用不可变内容键。接口统一提供写入、读取和关闭，业务不依赖具体驱动。数据库和正文目录均需持久化、备份。
- 整体下载统一提供固定 ZIP 地址：`GET /doc/download?ref=命名空间%2F文档标识`，基于一致内容快照导出整个文档库。公有库直接访问；私有库请求需携带 `X-Document-Token` 文档会话凭据，服务端代调可使用 `X-W7Panel-Access-Token` 面板 OIDC 凭据。地址不包含凭据，改为私有后原地址立即按私有规则校验；不生成临时分享链接，不需要 `download_link` 表。
- 公有库可通过 UI 的 `#/documents?ref=命名空间%2F文档标识` 地址匿名阅读；管理列表仍保持普通用户仅看自己的、创始人查看全局。

## 验证

从仓库根目录执行后端检查：

```sh
go test ./modules/document/app/... ./modules/document/common/...
go build ./modules/document/...
```

从 `modules/document/ui` 执行独立 UI 检查：

```sh
npm run lint -- --no-fix
npm run build -- --dest /private/tmp/zpk-document-ui-check/dist
```

独立后端入口为 `main.go`，负责加载配置、创建 HTTP 服务、初始化 ZPK SDK、注册文档 Provider 和启动应用。`common/service/sdk.go` 的 `InitSDK` 在入口调用，负责数据库目录、默认 DAO、正文存储、ZPK 和权限客户端初始化；`app/document/provider.go` 负责初始化模型和注册路由。Provider、控制器和 logic 使用无状态结构体，业务直接使用模块内的 `dao.Q` 及已初始化的服务，不通过构造函数传递数据库、存储或会话。认证中间件位于 `app/document/http/middleware`，登录 token 逻辑放在 `app/document/logic/login.go`，公共目录不反向依赖业务层。`Auth` 缺少或收到无效凭据时立即返回 401；身份服务异常返回 503。仅公开阅读和下载路由使用 `OptionalAuth`，无凭据可作为匿名用户进入权限检查，有凭据则必须通过认证。认证中间件将 `zpk.User` 直接放入用户上下文，logic 接收该用户。所有控制器沿用框架 `Validate`；业务错误统一处理，标识重复返回 409，数据库异常返回 503。ZPK 仅通过 `common/service/w7/document` SDK 请求文档服务，不在主服务内初始化或注册文档系统。

配置见本目录 `config.yaml`；前端开发代理仅在明确设置 `VUE_PROXY_TARGET` 后启用。服务启动会初始化数据库，验证代码时不要使用真实数据库或直接启动服务。

## 公共命名空间

公共命名空间继续存放在 ZPK 的 namespace 表中，新增 `visible` 字段，默认 `true`。它只控制现有镜像命名空间列表的展示，与表示镜像权限的 `visible_type`、文档库的 `public` 相互独立。namespace 列表固定只返回 `visible=true`，不接受显隐筛选参数，管理员也遵循该规则。已有命名空间加列后默认保持显示。

文档不维护分类表和分类接口。手动新建文档默认使用公共命名空间 `app_doc`，制品推送使用 `app`。文档库保存 `namespace` 和 `identifie`，资源值为 `namespace/identifie`。创建文档库时通过权限 SDK 确认公共命名空间存在，不存在则创建隐藏的私有空间，已存在时保留其设置。命名空间只参与资源定位与权限，不提供文档侧管理页面。

## 独立服务配置

- ZPK Chart 默认注入 `DOCUMENT_BASE_URL=http://w7-zpkdocument.default.svc.cluster.local:8001`；直接运行后端时可通过此环境变量配置文档服务地址，未设置则使用开发默认值 `http://127.0.0.1:8001`。固定接口前缀 `/doc/openapi` 由文档 SDK 拼接。
- 文档服务配置 `DOCUMENT_ZPK_URL`（ZPK 服务地址，默认 `http://127.0.0.1:8000`，固定 `/zpk` 前缀由 ZPK SDK 拼接）、`DOCUMENT_DB_PATH` 和 `DOCUMENT_STORAGE_PATH`。面板认证地址由 ZPK 配置。
- 权限服务配置 `permission.server_url`，环境变量为 `DOCUMENT_PERMISSION_SERVER_URL`，默认 `http://127.0.0.1:8000`，固定接口前缀 `/zpk/v2/openapi` 由权限 SDK 拼接。权限系统目前仍部署在 ZPK，文档只通过 HTTP SDK 访问，可单独配置服务地址。
- 文档前端独立维护请求工具：从宿主 frontend props / roleConfig 取得主 ZPK 微应用的代理前缀，追加 `/doc`。登录与业务请求均只在匹配该代理 origin/path 时携带 `X-W7Panel-Token`（宿主 paneltoken，缺省读取 localStorage）；宿主 access_token 通过登录请求体换取文档 JWT，业务请求只向文档服务路径发送 `X-Document-Token`。文档 token 失效时合并并发登录，原请求只重试一次；面板代理拒绝和登录失败不会递归触发登录。不引用主项目 UI 代码。前端请求回归测试：在 `ui/` 中执行 `npm test`。
- 文档登录返回 JWT 格式的 `data.token`，后续请求通过 `X-Document-Token` 携带；不创建或查询会话表，不使用 Cookie。`auth.token_ttl_seconds` 配置 token 的有效期，环境变量为 `DOCUMENT_TOKEN_TTL_SECONDS`（默认 86400 秒），到期后由前端使用宿主面板凭据重新登录。JWT 只保存 ZPK 返回的用户身份和签发、到期等字段，不保存面板访问凭据。使用 HS256，由 cloudnoauth 提供的 AppSecret 按文档登录用途派生签名密钥，同时验证 issuer、AppID audience 和时间字段；ZPK 与文档使用不同用途密钥及 issuer，登录 token 不能互换。
- 两个系统各自部署、各自持久化数据库，不共享 SQLite 文件或目录。文档库的 `user_id` 仍为 ZPK 的整数用户 ID，不复制用户表、不新增文档用户表。ZPK 用户接口按面板稳定 `sub` 解析或创建现有用户，不接受客户端指定用户 ID 或角色。
- 文档服务启动仅初始化文档库和文件数据表；创建文档库时才确认或创建对应公共命名空间。ZPK 通过文档 SDK 操作，不直接写文档数据库。
- 文档服务不可用或拒绝操作时，原制品接口返回错误；制品创建已成功但默认文档未初始化时保留错误提示，后续访问原文档接口可重试初始化。
- 旧文档仅通过一次性命令 `respo:migrate-documents` 迁移，日常接口不再自动复制旧文档。源文件保留，文档列表以文档服务结果为准。
- 文档按单人维护的场景处理，不维护 `revision` 或应用层互斥锁，保存直接覆盖当前内容。文档库和文件接口及调用方 SDK 均不传版本号；多处同时编辑时，以后成功保存的内容为准。相关写操作沿用数据库事务和唯一索引，在事务内检查库状态与操作权限；删除和更新检查执行结果，避免记录不存在时误报成功。整体下载在同一数据库读取事务内获取库与文件元数据，再按不可变内容键打包。

ZPK 的 OIDC 登录同样返回 JWT，不在服务端保存登录态。制品前端仅对配置的 ZPK origin/path 发送 `X-W7Panel-Access-Token`；ZPK 请求文档时不转发该凭据，而是在 App 签名请求中指定 ZPK 用户 ID。文档登录时才携带面板 `access_token` 请求 ZPK 用户接口验证；JWT 不包含面板访问凭据。远程官方仓库、第三方地址、跳过认证的请求不发送本地 JWT 和面板访问凭据。初始化直接沿用制品所有者 ID，创始人为他人初始化时不改成当前登录人的 ID。

文档登录时只请求 ZPK 用户接口，ZPK 再向面板验证凭据；已登录的文档请求在有效期内直接验证 JWT，不查询会话表，也不重复请求面板。文档 JWT 中的角色、管理员标记在重新登录后更新；无状态 token 不提供单个 token 的服务端撤销，轮换 AppSecret 会使对应 token 失效。切换前的会话 token 不再接受，用户需重新登录。

文档库作为 `document_library` 类型的资源，以 `命名空间/文档库标识` 为资源值，与镜像共用 ZPK 的资源表和验证规则。App 签名保护的 `/zpk/v2/openapi/resource/get,set,check,delete` 负责资源状态；公私沿用 `visible_type`（1 私有、4 公开只读），不再写入匿名用户授权。新建时通过 `/openapi/permission/add` 给所有者授予 `push/pull`，修改公私只更新资源状态，不替换已有用户授权；删除库时删除对应资源及其授权。旧文档匿名授权会在 ZPK 启动迁移中转换为资源状态，已有新资源设置不覆盖。文档业务和正文存储仍独立，不导入主项目代码。


权限服务失败会返回错误，不回退到本地权限判断；设置权限失败时文档数据库事务回滚。权限服务与文档数据库不共享事务，远程设置成功后本地提交失败仍可能留下需要重新设置的权限记录。

本版数据表只有文档库、文件两张，均通过模型生成配置维护，不创建或使用 `document_session`、框架 `sessions` 表。按新项目直接初始化模型，不维护旧文档系统的数据兼容或升级命令。验证过程不操作真实数据库。

文档库和文件均使用 GORM 的 `deleted_at` 软删除，默认查询、排序和导出自动排除已删除记录。删除文档库时，在同一事务内软删除库与库内文件；正文不因删除操作立即清理。文档库引用保留原唯一约束，删除后仍占用标识；文件删除后向原路径保存会复用原文件记录、写入新内容并清除删除时间。

数据库模型初始化默认执行，删除状态统一使用 `deleted_at`。

## 一次性旧文档迁移

迁移入口和全部迁移逻辑位于 ZPK 的 `app/respo/command/migrate_documents.go`。配置好 ZPK 和独立文档服务后，手动执行已构建的服务端程序；服务调用使用 App 签名及各制品所有者 ID，无需提供面板访问凭据：

```sh
./server respo:migrate-documents
./server respo:migrate-documents --identifie demo-app
```

省略 `--identifie` 时分批迁移全部制品。新文档库和默认首页通过文档 API 初始化；仅复制旧 `docs/` 下的 Markdown 文件，移除最外层 `docs/` 后保存到文档库，嵌套子路径保留；`docs/.order` 中的路径同步转换后恢复顺序。目标文档库已有 `index.md` 时保留原内容；没有时按当前制品配置和应用说明生成受保护首页并提交。应用说明优先使用数据库简介，尚未保存时使用本次迁移的 `readme.md`。旧 `docs/index.md` 不覆盖该首页。源数据先读取旧 `shared` OCI，再用本地 `<depot>/Formula/<name>/files` 覆盖同路径内容。根目录 `readme.md` 写入制品表，manifest 和图标不迁移；已保存的简介（包括主动清空的简介）不会被覆盖。复用已有文档标识，跳过目标中已有的正文，保留旧文件。单个制品的复制及排序成功后才保存其新文档标识和简介；单个制品读取或迁移失败时记录错误并跳过，继续迁移后续制品，最后汇总成功和失败数量。可以重试失败制品；重试时跳过已复制正文，排序会按旧顺序重新同步。

该命令在旧文档切换使用前手动执行，数据库切换禁写期间拒绝执行。代码验证不运行真实迁移。

迁移确认完成后，可单独清理旧 `shared` tag：

```sh
./server respo:clean-shared-tags --dry-run
./server respo:clean-shared-tags --identifie demo-app
./server respo:clean-shared-tags
```

清理前检查简介已写入数据库；旧包含文档时，还通过文档 API 确认对应路径存在，使用 App 签名和制品所有者 ID，无需面板访问凭据。发现未知旧资源、未迁移文档或与其他 tag 共用 manifest 时拒绝删除。`--dry-run` 只检查；实际清理删除 shared manifest 及对应本地 tag 记录，保留其他 tag 和本地源文件。命令不执行 Registry blob 垃圾回收。上述命令需手动执行，代码验证不运行真实删除。


## 服务间签名

文档作为 ZPK 制品的子应用，与主应用打入同一个 Helm 包，主、子应用的 cloudnoauth Sidecar 从同一 AppGroup/Site 取得相同 AppID/AppSecret。ZPK 沿用 `InitW7Sdk` 获取的凭据；文档使用相同方式，由 `common/service/sdk.go` 的 `fetchAppCredential` 在初始化时固定请求 `GET http://api.w7.cc/api/app/info`，经注入的 cloudnoauth Sidecar 获取 AppID/AppSecret，再由自己的 `common/service/app` 使用凭据签名和验签，不引用宿主代码，也不单独配置 cloudnoauth 服务地址。凭据缺失或获取失败时不启用无签名降级。

签名覆盖三个方向：

| 调用方向 | 接口与校验 |
| --- | --- |
| ZPK → 文档 | 调用方 SDK 在 `common/service/w7/document`，复用 `w7/base.Base` 的签名；`/doc/openapi` 的路由验证 App 签名后，从签名请求体读取 `user_id`，再执行文档业务授权，不要求用户 token |
| 文档 → ZPK | `/zpk/v2/openapi/user/current` 直接验证面板 access_token，不要求应用签名；公共 namespace 的 get/create 接口均验签 |
| 文档 → 权限服务 | `/zpk/v2/openapi/resource/set,check,delete` 与 `/zpk/v2/openapi/permission/add` 均验签，业务数据中的 user_id 由已认证的文档服务提供 |

请求沿用市场 SDK 的 body 封装方式：将完整业务 JSON 序列化为 `body` 字符串，外层仅携带字符串参数 `body`、`appid`、`timestamp`、`nonce`、`sign` 等。签名统一使用 `getSign`：排除 sign 后将外层参数按键排序并进行 URL 编码，拼接 AppSecret 后计算 MD5。接收端验签成功后，将 `body` 还原为请求体，控制器继续按原业务结构绑定参数，无需展开 files、grants 等嵌套字段。服务请求还签入 `request_method`、`request_uri` 和 `access_token_sha256`，分别绑定方法、完整请求路径/查询参数，以及面板访问凭据与文档会话凭据的摘要。时间戳允许前后 300 秒；nonce 使用随机值，本版不持久化 nonce 或维护重放去重缓存。同一签名只能在该时间窗口、原路径和原用户凭据下验证，不用于生成长期访问授权。

签名错误、缺少签名、其他 AppID、过期签名或篡改请求返回 401；验签凭据未初始化返回 503。服务签名不替代资源权限：ZPK SDK 在签名请求体中指定操作用户 `user_id`，文档按该用户校验维护和读取权限，不请求 ZPK 重复认证，也不接收未签名的用户 ID。浏览器继续使用原登录与页面接口，公有文档阅读/下载仍按原匿名权限处理，浏览器不接触 AppSecret。

这次是服务接口协议变更，ZPK 与文档服务需要配套更新；需验签的服务接口不提供无签名降级，文档登录请求 ZPK 当前用户接口时仅要求面板 access_token。主、子应用必须实际注入 cloudnoauth 并属于同一 AppGroup，单独部署时也需提供对应的 Sidecar 凭据来源。未修改 Helm 打包规则或其他仓库。

ZPK 编辑文件和发布默认首页统一使用 `POST /doc/openapi/file/save`，请求体携带 ZPK 用户 ID，由 App 签名保护。App 中间件只负责验签，随后由 AppUser 中间件读取用户 ID 并设置用户身份，文件控制器使用既有保存逻辑和资源权限校验；文档系统不区分同步和编辑。浏览器登录与编辑继续使用文档 JWT。发布首页更新位于 ZPK 服务端的统一发布流程，CLI 本身不增加文档请求。
