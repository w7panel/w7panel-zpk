# 文档系统

本目录作为文档系统的项目根目录，后端和 UI 不引用宿主 ZPK 的业务代码。当前使用根 Go module 管理依赖，通过本目录的后端入口独立启动。ZPK 主服务不注册文档 Provider，不连接文档数据库或初始化文档正文存储。

## 业务与页面归属

| 业务 | 后端入口 | UI 页面 | UI 路由 |
| --- | --- | --- | --- |
| 分类管理 | `app/document/provider.go`，controller 和 logic 各自的 `category.go` | `ui/src/views/categories.vue` | `/categories` |
| 文档库、文件及下载 | `app/document/provider.go` | `ui/src/views/libraries.vue` | `/documents` |
| 面板登录及当前用户 | `app/document/provider.go`，controller 和 logic 各自的 `login.go` | `ui/src/App.vue` | 登录入口 |

分类业务放在 `app/document/http/controller/category.go`、`app/document/logic/category.go`，由 document 的 provider 统一注册路由和初始化数据。分类和文档库各自维护独立页面和页面状态。文档库页面调用分类列表用于选择、筛选分类；分类管理页面负责分类 CRUD 和授权。分类管理仅管理员可操作，文档库列表按当前用户过滤，管理员可查看全局。

分类管理、文档库管理及其菜单、路由统一放在本目录 UI 内，ZPK 主 UI 不注册这些管理页面。

`ui/src/components/workbench.vue` 是本系统文档库页面使用的文档操作台。ZPK 保留原有 `description.vue` 和 `/respo/share-file/*` 请求协议，由制品后端通过调用方 SDK 转接读取、保存、删除及排序；ZPK UI 不引用本目录代码。`docs/.order` 由后端转换成文档系统的排序请求，空内容沿用旧接口的删除语义。默认 `index.md` 保留在文档库根目录，在文档系统 UI 查看，不映射成可编辑的制品文档。

## 调用边界

- 数据模型和生成 DAO 放在本目录的 `common/entity`、`common/dao`，业务直接使用 DAO。模型定义由 `database/schema.sql`、`database/table.yaml` 驱动；在本目录运行 `go run ./tools/generate`，生成过程使用临时数据库。
- 权限 SDK 在 `common/service/permission`，经通用 `/zpk/v2/api/permission` HTTP 接口访问当前权限系统。身份来自 Auth 中间件，资源构造和继承顺序由 SDK 维护。
- ZPK 请求文档的 SDK 属于调用方，位于宿主 `common/service/document`。制品默认文档经 SDK 初始化，宿主保存 `applications/制品标识` 引用。
- 分类、文档库、文件分别对应权限类型 `document_category`、`document_library`、`document_file`。文件显式策略优先于库和分类；默认首页的限制使用权限策略表达。
- 元数据保存在数据库，正文保存在 `common/service/storage` 抽象后的本机文件存储中，采用不可变内容键。数据库和正文目录均需持久化、备份。
- 整体下载支持 ZIP 和合并 Markdown，使用一致内容快照，只包含当前可读文件。外部链接限定创建时可读的路径范围，并检查当前权限、有效期和撤销状态。

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

独立后端入口为 `main.go`，只负责加载配置、创建 HTTP 服务、注册文档 Provider 和启动应用。`common/service/sdk.go` 的 `InitSDK` 负责数据库目录、数据库连接、正文存储和权限客户端初始化；`app/document/provider.go` 负责组装依赖、初始化模型和注册路由。ZPK 仅通过 `common/service/document` SDK 请求文档服务，不在主服务内初始化或注册文档系统。

配置见本目录 `config.yaml`；前端开发代理仅在明确设置 `VUE_PROXY_TARGET` 后启用。服务启动会初始化数据库，验证代码时不要使用真实数据库或直接启动服务。

## 独立服务配置

- ZPK 配置 `DOCUMENT_BASE_URL`（完整 API 前缀，默认 `http://127.0.0.1:8001/zpk/document`）。
- 文档服务配置 `DOCUMENT_PERMISSION_URL`、`DOCUMENT_DB_PATH` 和 `DOCUMENT_STORAGE_PATH`。
- 文档服务独立初始化默认 `applications` 分类；创始人在分类管理中给需要创建制品文档的用户授予该分类的 `read`、`create` 权限。ZPK 不直接写入文档分类或默认分类权限。
- 文档服务不可用或拒绝操作时，原制品接口返回错误；制品创建已成功但默认文档未初始化时保留错误提示，后续访问原文档接口可重试初始化。
- 旧文档仅通过一次性命令 `respo:migrate-documents` 迁移，日常接口不再自动复制旧文档。源文件保留，文档列表以文档服务结果为准。
- 原编辑页不发送版本号；后端使用本次查询到的版本提交，文档服务仍检查修改期间的版本冲突，但不能识别浏览器长期打开后提交的过期内容。

项目间调用鉴权暂不实现；当前请求仍传递用户登录 token，保留用户身份及文档业务权限校验。

## 一次性旧文档迁移

迁移入口和全部迁移逻辑位于 ZPK 的 `app/respo/command/migrate_documents.go`。先启动配置好的 ZPK 和独立文档服务，通过 `ZPK_TOKEN` 环境变量提供创始人现有的登录 token，再手动执行已构建的服务端程序：

```sh
./server respo:migrate-documents
./server respo:migrate-documents --identifie demo-app
```

省略 `--identifie` 时分批迁移全部制品。新文档库和默认首页通过文档 API 初始化；仅复制 `docs/*.md`，按 `docs/.order` 恢复顺序，不迁移 readme、manifest 或图标。复用已有文档标识，跳过目标中已有的正文，保留旧文件。单个制品的复制及排序成功后才保存其新文档标识；失败会停止并指出制品，可以重试。重试时跳过已复制正文，排序会按旧顺序重新同步。

该命令在旧文档切换使用前手动执行，数据库切换禁写期间拒绝执行。代码验证不运行真实迁移。
