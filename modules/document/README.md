# 文档系统

本目录作为文档系统的项目根目录，后端和 UI 不引用宿主 ZPK 的业务代码。当前使用根 Go module 管理依赖，通过本目录的后端入口独立启动。ZPK 主服务不注册文档 Provider，不连接文档数据库或初始化文档正文存储。

## 业务与页面归属

| 业务 | 后端入口 | UI 页面 | UI 路由 |
| --- | --- | --- | --- |
| 分类管理 | `app/document/provider.go`，controller 和 logic 各自的 `category.go` | `ui/src/views/categories.vue` | `/categories` |
| 文档库、文件及下载 | `app/document/provider.go` | `ui/src/views/libraries.vue` | `/documents` |
| 面板登录及当前用户 | `app/document/provider.go`，controller 和 logic 各自的 `login.go` | `ui/src/App.vue` | 登录入口 |

分类业务放在 `app/document/http/controller/category.go`、`app/document/logic/category.go`，由 document 的 provider 统一注册路由和初始化数据。分类和文档库各自维护独立页面和页面状态。文档库页面调用分类列表用于选择、筛选分类；分类管理页面负责分类 CRUD。分类的创建、编辑和删除由 `Admin` 中间件统一限制为管理员；分类列表向登录用户开放，文档库列表按当前用户过滤，管理员可查看全局。

分类管理、文档库管理及其菜单、路由统一放在本目录 UI 内，ZPK 主 UI 不注册这些管理页面。

`ui/src/components/workbench.vue` 是本系统文档库页面使用的文档操作台，新建文件直接使用库内根路径，不固定添加 `docs/`。ZPK 的 `description.vue` 通过 `/respo/document/path-tree` 和 `/respo/document/file` 请求制品后端，由调用方 SDK 转接读取、保存、删除及排序；ZPK UI 不引用本目录代码。文档路径直接使用 `guide.md`、`manual/guide.md` 等库内相对路径，不添加 `docs/` 前缀。`.order` 仅为制品编辑器的排序字段，由适配层转换成文档系统的排序请求；空内容沿用制品编辑器的删除语义。`/respo/share-file/*` 单独处理制品简介和其他共享文件，文档库中的 `readme.md` 不会覆盖制品简介。默认 `index.md` 保留在文档库根目录，在文档系统 UI 查看，不映射成可编辑的制品文档。

操作台维护目录、草稿、文件切换、保存和下载状态，Markdown 编辑及预览由 `ui/src/components/markdown-editor.vue` 负责。后端文档库、文件和下载业务分别放在 `app/document/logic/library.go`、`file.go`、`download.go`，`document.go` 仅保留公共错误、内容大小常量和引用解析。请求参数的格式、长度、取值与路径校验统一由 controller 的 `Validate` 处理，自定义验证规则在 Provider 注册阶段显式注册；logic 保留权限、关联存在性与业务保护检查，不重复校验参数。

文档系统只提供通用的 `POST /zpk/document/library/create`，接收库属性及可选的初始文件列表，不内置制品初始化流程或默认文件名。普通页面只提交标识、名称、描述和公有/私有设置，创建空库。制品端负责生成默认文档数据，通过 SDK 的服务入口 `POST /zpk/document/openapi/library/create` 提交相同数据：

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

库级 `protected=1` 表示库不可删除，文件级 `protected=1` 表示该文件不可修改、改名或删除，两者独立，不绑定文件名。`owner_id` 省略时使用当前用户，仅管理员可为其他用户建库。库与初始文件元数据在同一事务内创建，重复标识返回 409，不覆盖或升级已有数据。制品端遇到重复时，通过通用 `/library/info` 和 `/file/list` 确认所有者、库及首页保护属性，符合自己的默认文档规则才复用，后续保存文档引用失败可重试。正文存储与数据库不共享事务，数据库失败时正文可能保留。

## 调用边界

- 数据模型和生成 DAO 放在本目录的 `common/entity`、`common/dao`，业务直接使用 DAO。模型定义由 `database/schema.sql`、`database/table.yaml` 驱动；在本目录运行 `go run ./tools/generate`，生成过程使用临时数据库。
- ZPK 用户 SDK 在 `common/service/zpk`，携带前端提供的面板 `access_token` 请求 `POST /zpk/v2/api/user/current`，由 ZPK 验证面板凭据并返回用户 ID、用户名、角色和管理员标记。文档直接使用返回结果建立独立会话，不再重复请求面板或计算角色。登录及会话逻辑分别在 `app/document/logic/login.go`、`session.go`。ZPK SDK 由 `main.go` 调用 `service.InitSDK` 初始化，登录逻辑和认证中间件直接使用 `service.ZPKSdk`，不通过 Provider 传递，也不增加用户转发层。`common/service/permission` 是权限服务 SDK，公有/私有和所有者的操作权限均通过远程接口判断；文档只维护默认首页不可修改等业务保护规则。
- ZPK 请求文档的 SDK 属于调用方，位于宿主 `common/service/w7/document`。宿主入口统一调用 `w7.InitW7Sdk`，其内部使用已有 AppID/AppSecret 初始化 `w7.DocumentSdk`，签名复用 `w7/base`。`app/respo/logic/document.go` 的 `FormulaDocument.Initialize` 生成 `app/制品标识` 引用、文档名称、首页正文与保护标记，通过 SDK 的通用 `Create` 接口推送，普通制品流程和一次性制品文档迁移命令共用这套规则。SDK 不生成制品业务内容，文档系统没有 `Initialize` 接口或对应逻辑。
- 文档库通过 `public` 字段设置公有（1）或私有（0，默认），库内所有文件继承。创建及修改可见性时，文档 SDK 将所有者的维护权限及公有库的匿名读取权限提交给 permission 服务；删除库时清除对应资源的权限。文件读取、编辑、删除、排序和整体下载均请求 permission 校验，不依据本地 `public` 或管理员标记绕过远程验证。公有库允许匿名查看及下载，私有库按用户授权及原有 namespace 兜底规则校验；默认仅向所有者授予维护动作，管理员沿用现有放行规则。分类关联公共命名空间；登录用户可选择文档本地已有分类新建自己的文档库。文档当前使用 `push/pull`：所有者拥有两者，公有库额外授予匿名用户 `pull`；文档内部将 `pull` 对应查看和下载，将 `push` 对应编辑、新建、删除等维护操作。权限接口按格式校验动作名称，允许业务后续增加细粒度动作，文档当前操作保持不变。授权验证沿用镜像的完整流程，namespace 兜底为内置规则；调用前准备公共动作：镜像根据现有 `visible_type` 及是否跟随 namespace，先算出最终公共 `actions`；文档直接使用保存的匿名授权动作。权限分两层：上层按资源完整路径解析 namespace，优先选择当前用户的资源授权；未命中时，push 在资源配置过任何精确授权后停止兜底，pull 仍可使用 namespace 用户授权。底层只校验准备好的公共动作和用户授权，不查询 namespace，也不读取可见性枚举。文档当前仅传文档库资源校验，文件继承库权限，文档库公私状态独立设置，用户授权按内置 namespace 规则兜底。
- `protected` 是库和文件的通用保护属性，不记录来源，创始人也遵循保护限制。制品端提交库及 `index.md` 的保护属性，其他文件默认可正常维护；文档后端不对 `index.md` 作特殊处理。
- 元数据保存在数据库，正文通过 `storage.Store` 接口访问，本版启动时选择本机文件驱动并采用不可变内容键。接口统一提供写入、读取和关闭，业务不依赖具体驱动。数据库和正文目录均需持久化、备份。
- 整体下载统一提供固定 ZIP 地址：`GET /zpk/document/download?ref=分类%2F文档标识`，基于一致内容快照导出整个文档库。公有库直接访问；私有库请求需携带 `X-Document-Token` 文档会话凭据，服务端代调可使用 `X-W7Panel-Access-Token` 面板 OIDC 凭据。地址不包含凭据，改为私有后原地址立即按私有规则校验；不生成临时分享链接，不需要 `download_link` 表。
- 公有库可通过 UI 的 `#/documents?ref=分类%2F文档标识` 地址匿名阅读；管理列表仍保持普通用户仅看自己的、创始人查看全局。

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

独立后端入口为 `main.go`，负责加载配置、创建 HTTP 服务、初始化 ZPK SDK、注册文档 Provider 和启动应用。`common/service/sdk.go` 的 `InitSDK` 在入口调用，负责数据库目录、默认 DAO、正文存储、ZPK 和权限客户端初始化；`app/document/provider.go` 负责初始化模型和注册路由。Provider、控制器和 logic 使用无状态结构体，业务直接使用模块内的 `dao.Q` 及已初始化的服务，不通过构造函数传递数据库、存储或会话。认证及管理员中间件位于 `app/document/http/middleware`，session 继续放在 `app/document/logic/session.go`，公共目录不反向依赖业务层。`Auth` 缺少或收到无效凭据时立即返回 401；身份服务异常返回 503。仅公开阅读和下载路由使用 `OptionalAuth`，无凭据可作为匿名用户进入权限检查，有凭据则必须通过认证。认证中间件将 `zpk.User` 直接放入用户上下文，logic 接收该用户；分类列表从文档本地关联记录读取，不按公共命名空间的隐藏状态推断业务归属。所有控制器沿用框架 `Validate`；业务错误统一处理，标识重复返回 409，数据库异常返回 503。ZPK 仅通过 `common/service/w7/document` SDK 请求文档服务，不在主服务内初始化或注册文档系统。

配置见本目录 `config.yaml`；前端开发代理仅在明确设置 `VUE_PROXY_TARGET` 后启用。服务启动会初始化数据库，验证代码时不要使用真实数据库或直接启动服务。

## 公共命名空间与本地分类

公共命名空间继续存放在 ZPK 的 namespace 表中，新增 `visible` 字段，默认 `true`。它只控制现有镜像命名空间列表的展示，与表示镜像权限的 `visible_type`、文档库的 `public` 相互独立。namespace 列表固定只返回 `visible=true`，不接受显隐筛选参数，管理员也遵循该规则。已有命名空间加列后默认保持显示。

文档分类保留本地 `DocumentCategory` 记录，`identifie` 对应公共 namespace 的 `name`，本地名称、描述用于文档页面展示。新建分类先通过 ZPK SDK 按名称查询公共空间；存在则关联并保留其全部设置，不存在则创建 `visible=false`、镜像权限为私有的系统空间，成功后保存本地记录。服务接口统一使用 `/zpk/v2/openapi/namespace/get`、`/create`、`/rename`、`/delete`，通过 AppID/Sign 验证同一应用组的服务调用；原镜像管理接口仍保留用户认证。文档只读取本地已关联的分类，其他业务创建隐藏空间不会被自动导入。

编辑分类标识时，先同步重命名公共 namespace 及其通用权限资源标识，再在文档本地事务中更新分类与所属文档库的分类标识；显示名称、描述只保存在本地。默认 `app` 分类标识不可修改，分类不可删除。删除空分类先删除远程 namespace，成功后软删除本地分类；存在文档库时由文档阻止删除，公共 namespace 有镜像或其他资源授权时由公共服务拒绝删除，不包含文档专项判断。修改 `visible` 不影响文档的读取、下载或维护权限。

公共 namespace、permission 与文档数据库不共享事务。公共空间创建成功但本地保存失败时会保留公共空间，再次操作可以关联；改名后本地更新失败会尝试恢复远程旧名，恢复失败同时返回错误。远程删除成功但本地提交失败仍可能造成不一致。权限写入与本地提交的失败处理仍按下文约定执行。本版不增加文档旧数据迁移逻辑。

## 独立服务配置

- ZPK 配置 `DOCUMENT_BASE_URL`（文档服务地址，默认 `http://127.0.0.1:8001`），固定接口前缀 `/zpk/document/openapi` 由文档 SDK 拼接。
- 文档服务配置 `DOCUMENT_ZPK_URL`（ZPK 服务地址，默认 `http://127.0.0.1:8000`，固定 `/zpk` 前缀由 ZPK SDK 拼接）、`DOCUMENT_DB_PATH` 和 `DOCUMENT_STORAGE_PATH`。面板认证地址由 ZPK 配置。
- 权限服务配置 `permission.server_url`，环境变量为 `DOCUMENT_PERMISSION_SERVER_URL`，默认 `http://127.0.0.1:8000`，固定接口前缀 `/zpk/v2/openapi/permission` 由权限 SDK 拼接。权限系统目前仍部署在 ZPK，文档只通过 HTTP SDK 访问，可单独配置服务地址。
- 文档会话通过生成的 `DocumentSession` 模型和 DAO 存取。登录返回随机生成的 `data.token`，数据库仅保存 token 摘要、用户信息和过期时间，不保存面板访问凭据。后续请求通过 `X-Document-Token` 携带凭据；没有 Cookie 读写、SessionStore、Session 中间件或旧凭据兼容分支。`auth.token_ttl_seconds` 配置文档登录 token 的有效期，环境变量为 `DOCUMENT_TOKEN_TTL_SECONDS`（默认 86400 秒），不需要 Cookie 或会话签名密钥配置。
- 两个系统各自部署、各自持久化数据库，不共享 SQLite 文件或目录。分类、文档库的 `user_id` 仍为 ZPK 的整数用户 ID，不复制用户表、不新增文档用户表。ZPK 用户接口按面板稳定 `sub` 解析或创建现有用户，不接受客户端指定用户 ID 或角色。
- 文档服务首次初始化默认 `app` 分类时，先通过 SDK 确认或创建公共空间，成功后保存本地分类，首次初始化需要 ZPK namespace 服务可用。后续建库仅检查本地分类关联，不反复请求远程 namespace；不调整制品 OCI 存储或历史包的定位。ZPK 不直接写入文档分类或权限表。
- 文档服务不可用或拒绝操作时，原制品接口返回错误；制品创建已成功但默认文档未初始化时保留错误提示，后续访问原文档接口可重试初始化。
- 旧文档仅通过一次性命令 `respo:migrate-documents` 迁移，日常接口不再自动复制旧文档。源文件保留，文档列表以文档服务结果为准。
- 文档按单人维护的场景处理，不维护 `revision` 或应用层互斥锁，保存直接覆盖当前内容。分类、文档库和文件接口及调用方 SDK 均不传版本号；多处同时编辑时，以后成功保存的内容为准。相关写操作沿用数据库事务和唯一索引，在事务内检查库状态与操作权限；删除和更新检查执行结果，避免记录不存在时误报成功。整体下载在同一数据库读取事务内获取库与文件元数据，再按不可变内容键打包。

项目间服务密钥鉴权暂不实现；用户身份验证仍保留。ZPK 的 OIDC 登录将面板访问凭据保存在服务端会话，制品后端通过调用方 SDK 以 `X-W7Panel-Access-Token` 传给文档服务，不改变原制品 UI 请求。旧 ZPK 会话缺少该凭据时返回 401，由原 UI 重新登录。初始化直接沿用制品所有者 ID，创始人为他人初始化时不改成当前登录人的 ID。

文档登录及以面板凭据代调时只请求 ZPK 用户接口，ZPK 再向面板验证凭据；已建立的文档会话在有效期内从文档数据库读取。

权限服务只保留三个 POST 接口：`/check` 接收服务端认证后确定的 `user_id`、单个 `resource` 和 `action`，支持 `user_id=0` 校验匿名读取；`/set` 接收 `grants` 并整体替换一个资源的权限；`/get` 接收 `resource` 并返回原始权限记录（`id/user_id/resource_type/resource_value/action` 等），不转换为所有者、公有或私有等业务字段。三个接口暂不验证服务调用凭据，`set/get` 不校验调用人的资源管理权限；`check` 仍执行实际资源授权判断。文档入口的用户认证、公私权限检查及默认文档保护继续保留，公有/私有界面不提供任意用户授权。

权限服务失败会返回错误，不回退到本地权限判断；设置权限失败时文档数据库事务回滚。权限服务与文档数据库不共享事务，远程设置成功后本地提交失败仍可能留下需要重新设置的权限记录。

本版数据表为分类、文档库、文件和 `document_session` 四张表，均通过模型生成配置维护，不使用框架的 `sessions` 表。按新项目直接初始化模型，不维护旧文档系统的数据兼容或升级命令。验证过程不操作真实数据库。

分类、文档库和文件均使用 GORM 的 `deleted_at` 软删除，默认查询、排序和导出自动排除已删除记录。删除文档库时，在同一事务内软删除库与库内文件；正文不因删除操作立即清理。分类下仍有未删除的文档库时不能删除分类。分类标识、文档库标识保留原唯一约束，删除后仍占用标识；文件删除后向原路径保存会复用原文件记录、写入新内容并清除删除时间。

数据库模型初始化默认执行，删除状态统一使用 `deleted_at`。

## 一次性旧文档迁移

迁移入口和全部迁移逻辑位于 ZPK 的 `app/respo/command/migrate_documents.go`。先启动配置好的 ZPK 和独立文档服务，通过 `W7PANEL_ACCESS_TOKEN` 环境变量提供创始人的面板 OIDC 访问凭据，再手动执行已构建的服务端程序：

```sh
./server respo:migrate-documents
./server respo:migrate-documents --identifie demo-app
```

省略 `--identifie` 时分批迁移全部制品。新文档库和默认首页通过文档 API 初始化；仅复制旧 `docs/` 下的 Markdown 文件，移除最外层 `docs/` 后保存到文档库，嵌套子路径保留；`docs/.order` 中的路径同步转换后恢复顺序。旧 `docs/index.md` 会与默认首页冲突，命令会报错，需先重命名该旧文档。制品根目录的 readme、manifest 和图标不迁移。复用已有文档标识，跳过目标中已有的正文，保留旧文件。单个制品的复制及排序成功后才保存其新文档标识；失败会停止并指出制品，可以重试。重试时跳过已复制正文，排序会按旧顺序重新同步。

该命令在旧文档切换使用前手动执行，数据库切换禁写期间拒绝执行。代码验证不运行真实迁移。

## 服务间签名

文档作为 ZPK 制品的子应用，与主应用打入同一个 Helm 包，主、子应用的 cloudnoauth Sidecar 从同一 AppGroup/Site 取得相同 AppID/AppSecret。ZPK 沿用 `InitW7Sdk` 获取的凭据；文档使用相同方式，由 `common/service/sdk.go` 的 `fetchAppCredential` 在初始化时固定请求 `GET http://api.w7.cc/api/app/info`，经注入的 cloudnoauth Sidecar 获取 AppID/AppSecret，再由自己的 `common/service/app` 使用凭据签名和验签，不引用宿主代码，也不单独配置 cloudnoauth 服务地址。凭据缺失或获取失败时不启用无签名降级。

签名覆盖三个方向：

| 调用方向 | 接口与校验 |
| --- | --- |
| ZPK → 文档 | 调用方 SDK 在 `common/service/w7/document`，复用 `w7/base.Base` 的签名；`/zpk/document/openapi` 的路由先验签，再执行原用户认证及文档业务授权 |
| 文档 → ZPK | `/zpk/v2/api/user/current` 直接验证面板 access_token，不要求应用签名；公共 namespace 的 get/create/rename/delete 接口均验签 |
| 文档 → 权限服务 | `/zpk/v2/openapi/permission/check`、`/set`、`/get` 均验签，业务数据中的 user_id 由已认证的文档服务提供 |

请求沿用市场 SDK 的 body 封装方式：将完整业务 JSON 序列化为 `body` 字符串，外层仅携带字符串参数 `body`、`appid`、`timestamp`、`nonce`、`sign` 等。签名统一使用 `getSign`：排除 sign 后将外层参数按键排序并进行 URL 编码，拼接 AppSecret 后计算 MD5。接收端验签成功后，将 `body` 还原为请求体，控制器继续按原业务结构绑定参数，无需展开 files、grants 等嵌套字段。服务请求还签入 `request_method`、`request_uri` 和 `access_token_sha256`，分别绑定方法、完整请求路径/查询参数，以及面板访问凭据与文档会话凭据的摘要。时间戳允许前后 300 秒；nonce 使用随机值，本版不持久化 nonce 或维护重放去重缓存。同一签名只能在该时间窗口、原路径和原用户凭据下验证，不用于生成长期访问授权。

签名错误、缺少签名、其他 AppID、过期签名或篡改请求返回 401；验签凭据未初始化返回 503。服务签名不替代用户权限：调用文档的 SDK 仍携带面板用户凭据，非管理员不能通过服务入口为其他用户建库，也不能操作无权限的私有文档。浏览器继续使用原登录与页面接口，公有文档阅读/下载仍按原匿名权限处理，浏览器不接触 AppSecret。

这次是服务接口协议变更，ZPK 与文档服务需要配套更新；需验签的服务接口不提供无签名降级，当前用户查询仅要求面板 token。主、子应用必须实际注入 cloudnoauth 并属于同一 AppGroup，单独部署时也需提供对应的 Sidecar 凭据来源。未修改 Helm 打包规则或其他仓库。
