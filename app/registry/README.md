# Registry 公共资源与权限

本模块提供公共命名空间、资源公开状态、用户授权和认证。镜像是资源类型之一，文档库通过相同协议使用权限服务；镜像的 OCI、tag、webhook 和远程删除仍由镜像业务处理。

## 1. 项目结构与职责

```text
app/registry/
├── provider.go                         # 注册路由、事件、命令；初始化表结构和签名证书
├── http/controller/
│   ├── common/                         # 登录用户使用的通用管理接口及认证入口
│   │   ├── auth.go                     # 解析 scope、校验账号、申请 Registry token
│   │   ├── info.go                     # 查询 Registry 服务地址
│   │   ├── namespace.go                # 命名空间管理、子路径查询
│   │   ├── namespace_permission.go     # namespace 授权管理、CheckNamespacePermission
│   │   └── resource.go                 # 按 resource_type 创建、查询、列出资源
│   ├── openapi/                        # 服务间调用，统一使用 App 签名
│   │   ├── namespace_service.go        # 公共命名空间 get/create/rename/delete
│   │   ├── resource.go                 # 资源状态 get/set/check/delete
│   │   └── resource_permission.go      # 用户授权 add/delete/clear，以及 check 实现
│   └── repository/                     # 镜像专属管理接口
│       ├── image_repository.go         # 镜像编辑、OCI 复制、tag、远程删除
│       └── webhook.go                  # 接收镜像推送、拉取事件
├── logic/
│   ├── common/
│   │   ├── auth.go                     # scope 鉴权、签发和验证 token
│   │   ├── namespace.go                # 公共 namespace、重命名及关联检查
│   │   ├── resource.go                 # 资源状态、公开权限准备、管理操作
│   │   ├── resource_permission.go      # 精确授权、namespace 兜底、动作集合校验
│   │   ├── visible_type.go             # visible_type 常量及公开动作转换
│   │   └── resource_migrate.go         # 旧文档授权向现有资源表补齐状态
│   └── repository/
│       ├── image_repository.go         # 镜像事件、tag、OCI 操作和删除实现
│       └── deploy.go                   # 镜像推送事件与后续业务的衔接
└── command/
    ├── clear_images.go                 # 镜像清理命令入口
    ├── clear_images_logic.go           # 清理命令专属实现
    ├── migrate.go                      # 镜像迁移命令入口
    └── migrate_logic.go                # 迁移命令专属实现
```

相关目录：

| 位置 | 用途 |
| --- | --- |
| `common/entity/`、`common/dao/` | 数据模型和生成查询 |
| `common/service/w7/permission/client.go` | 资源状态、namespace、用户授权及权限校验 SDK |
| `common/service/w7/base/base.go` | App 请求签名与验签 |
| `common/middleware/app.go` | 服务间签名中间件 |
| `common/service/registry/`、`common/service/oci/` | Registry 协议客户端、OCI 存储操作 |
| `modules/document/common/service/permission/` | 独立文档模块的权限 HTTP 客户端 |
| `ui/src/views/registry.vue`、`registry-detail.vue` | 镜像列表、详情界面 |

通用逻辑当前仍随 ZPK 部署，使用同一数据库。这里的独立指职责与 API 边界独立；文档业务和文件存储在文档服务中。业务接入通过 SDK 调用，迁移命令可以直接操作数据库。

## 2. 数据与资源标识

沿用现有三张表，不新增资源表：

| 模型 / 默认表名 | 保存内容 |
| --- | --- |
| `RegistryNamespace` / `ims_registry_namespace` | 公共 namespace、公开状态、所属用户、列表可见性 |
| `RegistryRepository` / `ims_registry_repository` | 各类型资源的 namespace、name、user_id、visible_type 等 |
| `RegistryUserPermission` / `ims_registry_user_permission` | user_id 对 resource_type + resource_value 的 actions 授权 |

`RegistryRepository.resource_type` 默认是 `repository`，旧镜像记录继续使用此类型。文档库使用 `document_library`。同名路径的镜像与文档库分别查询、分别授权。

资源调用参数：

```json
{"type":"repository","value":"team/demo"}
```

- `type`：资源类型字符串，不限制为固定枚举。
- `value`：不含域名和 tag 的资源路径。`team/group/demo` 解析为 namespace `team`、name `group/demo`；只解析第一层，不逐级继承 `group`。
- 无 `/` 的路径使用 `default` namespace。生成路径沿用 `BuildRepositoryName`，默认 namespace 可省略前缀；授权存取应统一使用相同路径形式。
- 文档库通常为 `{category}/{identifie}`，如 `team/handbook`。

`user_id` 记录所有者，但镜像 Registry scope 鉴权不会仅因所有者身份自动赋予 pull/push。业务创建资源时，需同时维护所有者授权。

### visible_type

常量统一定义在 `logic/common/visible_type.go`：

| 值 | 常量 | 公开动作 | 含义 |
| --- | --- | --- | --- |
| 1 | `VisibleTypePrivate` | 空 | 私有读写，操作需用户授权 |
| 2 | `VisibleTypePublicWrite` | `pull,push` | 公有读写 |
| 3 | `VisibleTypeFollowNamespace` | 取 namespace 的公开动作 | 跟随 namespace；namespace 自身不用此值 |
| 4 | `VisibleTypePublicRead` | `pull` | 公有读、私有写 |

公开动作由 `PublicActionsForVisibleType` 临时转换，不作为额外数据库字段保存。`PublicActions` 是鉴权时准备的数据，不能由外部请求设置。

切换公私只修改 `visible_type`，不替换用户授权。公有状态不通过新增 `user_id=0` 的授权实现。

## 3. 鉴权原理

### 3.1 两种 namespace 规则

**公开状态继承**由 `Resource.Prepare` 处理：

1. 查询指定类型、指定路径的资源。
2. 已有资源且 `visible_type != 3`，直接使用资源自身的公开状态。
3. 资源不存在，或资源为跟随状态，查询 namespace 的公开状态。
4. 需要继承但 namespace 不存在时返回错误；数据库故障也返回错误，不按查无记录放行。

**用户授权兜底**由 `ResourcePermission.CheckResource` 处理，独立于公开状态继承。因此，私有镜像仍可能通过 namespace 的用户授权访问。

### 3.2 镜像 Registry 鉴权调用链

```text
/registry/auth
  → controller/common.Auth：解析 scope、校验账号密码
  → logic/common.Auth.CheckScopes
      → Registry 专用超级管理员直接通过
      → Resource.Prepare：准备资源 / namespace 的公开动作
      → prepareScopeResource：保留旧版 push 优先、pull 其次的公开判断
      → ResourcePermission.CheckResource：公开权限、精确授权、namespace 兜底
          → CheckPermission：对准备好的动作集合做最终判断
  → 全部 scope 通过后签发 Registry token
```

公开权限的判断保留旧逻辑：scope 含 `push` 时先检查公开写；仅在不含 `push` 时检查 `pull` 的公开读。命中即放行整个 scope。这保留了旧版混合动作行为；用户授权则要求覆盖全部请求动作。

### 3.3 公开权限未通过后的用户授权顺序

1. 未登录用户直接拒绝。
2. 查询**当前用户 + 当前资源类型 + 当前资源路径**的第一条精确授权。
3. 有精确授权：按该条 actions 校验全部请求动作。不够就拒绝，不合并 namespace 授权，也不合并重复记录。
4. 没有精确授权且请求包含 `push`：查询该资源是否给任何用户配置过精确授权。只查询存在性，不加载全部授权。
5. 若已配置任何精确授权，拒绝 push 的 namespace 兜底。
6. 否则查询当前用户的 namespace 授权，校验全部请求动作。
7. 请求不含 `push` 时，没有精确授权就直接尝试 namespace 授权。
8. 资源本身类型为 `namespace` 时，不再尝试上级 namespace。

`pull` 和 `push` 不互相隐含。只有 `push` 的用户授权，不能满足 `pull,push` 的请求。

| 资源状态 / 授权情况 | pull | push |
| --- | --- | --- |
| 公有读写（2） | 匿名通过 | 匿名通过 |
| 公有读、私有写（4） | 匿名通过 | 按用户授权顺序校验 |
| 私有读写（1） | 按用户授权顺序校验 | 按用户授权顺序校验 |
| 当前用户有精确授权 | 以该条授权为准 | 以该条授权为准 |
| 当前用户无精确授权，资源也未给任何人授权 | 可用 namespace 用户授权 | 可用 namespace 用户授权 |
| 当前用户无精确授权，但资源给其他人授权 | 可用 namespace 用户授权 | 拒绝 namespace 兜底 |

公开权限在用户精确授权前判断。因此，公开读资源不会因为当前用户的精确授权只有 push 而失去公开 pull；公开读写资源也不会被用户授权记录收窄。

例如：`team/demo` 为私有，张三在 `team` 上有 pull/push，李四在该镜像上有精确授权，而张三没有镜像精确授权，则张三 pull 通过、push 拒绝。如果镜像完全没有精确授权，张三可通过 namespace 授权 pull/push。

### 3.4 通用资源 check 与 token

服务间统一通过 `/resource/check` 校验：

- 提供 `token` 时，验证 token 签名、有效期、issuer、audience，以及匹配的资源类型、路径和动作；不会再用 user_id 重新计算授权。
- 未提供 token 时，按 user_id 获取用户，再调用 `Resource.Check`。`user_id=0` 表示匿名。
- `Resource.Check` 不根据创始人或面板管理员角色直接放行，只校验资源公私状态与用户授权。制品 `formula/info` 和文档业务层各自处理业务管理员豁免；权限 SDK 仅负责请求。镜像 `/registry/auth` 仍仅对 `UserTypeSuperAdminForRegistry` 直接放行，保留原有镜像鉴权行为。
- Registry token 使用 RS256，audience 为 `docker-registry`，当前有效期为 7 天。已签发 token 不会因数据库授权变化而立即失效。
- 当前 Auth 会验证各 scope 声明的资源类型；非 repository 类型不再被直接忽略。镜像准备事件的处理器只处理 repository 类型。

通用 action 是字符串，可增加细粒度动作。但当前公开状态只生成 pull/push，阻止 namespace 兜底的动作也是 push；新增如 download、edit 时，需要业务明确映射及继承语义，不能假定它们自动等同于 pull/push。

### 3.5 Formula.Info 调用市场的使用场景

`app/respo/http/controller/formula.go` 的 `Info` 按请求凭据区分普通仓库与市场访问。应用已上架本身不会跳过仓库权限；没有 `order_sn` 和市场 `token` 时，仍走 Permission SDK 的本地 pull 校验。

| 使用场景 | 请求凭据 / 模式 | 调用及作用 |
| --- | --- | --- |
| 市场管理后台审核 / 测试安装 | 商品后台列表生成的 URL 带 `token` | `ZpkMarketSdk.CheckToken` 检查应用标识和有效期；通过后进入安装信息生成，不要求购买订单或本地仓库 pull |
| 普通购买用户安装 / 升级 | `order_sn`，并有已认证的市场用户身份 | `CheckFormulaCanInstallOrUpgrade` 验证订单安装 / 升级资格及域名、应用标识绑定 |
| 订单允许版本解析 | 订单安装资格已通过 | `GetFormulaCanUpgradeVersion` 解析允许版本、到期状态及目标制品绑定 |
| 已安装应用查询运行上下文 | `order_sn` + `runtime_context=true` | `GetFormulaLicenseInfo` 读取授权状态等运行信息并返回，不生成完整安装响应，也不作为安装授权结果 |

后台安装 token 的签发位于市场项目的 `Goods.List → GetFormulaFounderToken`，目前有效期 30 分钟，由市场验证；普通购买用户无需使用它。请求同时包含 token 与订单时，当前优先进入 token 分支。

源站所有者和管理员保留 `Info` 的管理豁免；订单辅助逻辑对市场发布者也保留原有豁免。市场验证通过后的安装包仍由源 ZPK 提供，订单安装完成通过安装 ticket 回调核销。市场 token 请求参数、Registry Bearer token、订单号和安装 ticket 是不同凭据，不能互换。

## 4. 接口使用

### 4.1 服务间开放接口

以下均为 POST，前缀为 `/zpk/v2/openapi`，挂载 `middleware.App`。App 签名确认调用服务身份；调用方负责验证用户身份、决定操作是否允许，再传入可信 user_id。

| 路径 | 主要请求参数 | 作用 |
| --- | --- | --- |
| `/resource/get` | resource | 查询资源元数据；不存在返回 404 |
| `/resource/set` | resource、user_id、visible_type | 不存在时创建；已有时仅修改公开状态，保留所有者和用户授权 |
| `/resource/check` | resource、action、user_id 或 token | 返回 `data.allowed`、`data.explicit` |
| `/resource/delete` | resource | 删除资源元数据及该资源用户授权；不删除 OCI 内容或文档文件 |
| `/permission/add` | user_id、resource、actions | 不存在时添加该用户精确授权；已有时不覆盖 actions |
| `/permission/delete` | user_id、resource | 删除指定用户、指定资源的授权 |
| `/permission/clear` | user_id（大于 0） | 删除该用户全部资源及 namespace 授权 |
| `/namespace/get` | name | 查询公共 namespace |
| `/namespace/create` | name、visible_type（1/2/4）、可选 visible | 新建 namespace；visible 控制列表显示，独立于公私 |
| `/namespace/rename` | id、name | 重命名并同步关联资源 namespace 与授权路径 |
| `/namespace/delete` | id | 删除未被资源 / 关联授权占用的 namespace；default 受保护 |

旧的 `/permission/get`、`/permission/set`、`/permission/check` 已移除；资源状态和检查统一使用 `/resource/*`，用户授权使用 `/permission/add,delete,clear`。

授权 add 不是修改接口。需要替换已有用户的 actions 时，当前可删除后重新添加；这是两次请求，中间可能失败，不是原子操作。

`explicit=true` 表示最终使用了用户授权记录，也可能来自 namespace 兜底；不能单凭此字段判断命中了资源精确授权。成功放行应检查 `allowed`。

请求示例为**签名前的业务 JSON**：

```json
{
  "resource": {"type": "repository", "value": "team/demo"},
  "user_id": 42,
  "visible_type": 1
}
```

签名由 SDK 的 `Base.Sign` 处理：把完整业务 JSON 放入 body 字符串，与 appid、timestamp、nonce、请求方法、URI、相关 token 摘要一起签名。`middleware.App` 验签后还原业务 body，再交给 controller 绑定参数。直接发送以上裸 JSON 不会通过签名验证。

### 4.2 SDK 接入示例

以下示例中的客户端已由主项目初始化；新建 namespace、资源状态和所有者授权是分开的请求，调用方逐步处理错误。

```go
import (
    "context"
    "errors"

    "github.com/w7panel/w7panel-zpk/common/service/w7"
    permissionsdk "github.com/w7panel/w7panel-zpk/common/service/w7/permission"
)

func createPrivateResource(ctx context.Context, userID int32) error {
    resource := permissionsdk.Resource{Type: "repository", Value: "team/demo"}
    if err := w7.PermissionSdk.EnsureNamespace(ctx, "team"); err != nil {
        return err
    }
    if err := w7.PermissionSdk.SetResource(ctx, resource, userID, 1); err != nil {
        return err
    }
    return w7.PermissionSdk.Add(ctx, permissionsdk.Grant{
        UserID: userID,
        Resource: permissionsdk.Resource{Type: resource.Type, Value: resource.Value},
        Actions: []string{"pull", "push"},
    })
}

func requirePull(ctx context.Context, userID int32) error {
    allowed, err := w7.PermissionSdk.Check(ctx, userID,
        permissionsdk.Resource{Type: "repository", Value: "team/demo"}, "pull", "")
    if err != nil {
        return err
    }
    if !allowed {
        return errors.New("无读取权限")
    }
    return nil
}
```

上述调用统一使用 `w7.PermissionSdk`，初始化地址为 `setting.permission.server_url`（`PERMISSION_SERVER_URL`），不再单独初始化 Registry SDK。

主项目内部若引用 logic 常量，可用 `VisibleTypePrivate` 替代示例中的 1；独立服务按协议值调用，不依赖主项目 logic 包。SDK serverURL 配置为服务根地址，不包含 `/zpk/v2/openapi`，SDK 自行附加路径。

文档模块独立维护 HTTP 客户端：`SetLibraryVisibility` 设置 document_library 的状态，`AddOwner` 添加 pull/push，`CanRead` 对应 pull，`CanManage` 对应 push，`ClearLibrary` 删除资源与授权。保护文件、禁止删除等文档业务约束仍由文档服务处理。

### 4.3 面向登录用户的管理接口

前缀为 `/zpk/v2/api`，使用登录中间件；这些接口的列表 / 所属用户规则与 Registry pull/push scope 鉴权职责不同。

| 路径 | 用途 |
| --- | --- |
| POST `/resource/create` | 创建资源；传 resource_type、name、namespace、visible_type、可选 desc |
| POST `/resource/list` | 按 resource_type、可选 namespace / sub_namespace 查询 |
| POST `/resource/info` | 按 id + resource_type 查询 |
| POST `/namespace/add,info,list,edit,del` | namespace 管理（逗号表示分别注册的路径） |
| POST `/namespace/sub_namespace/list` | 按 namespace 和 resource_type 查询子路径 |
| POST `/permission/namespace/get,set` | 管理员查看 / 整体替换 namespace 用户授权 |
| GET `/registry/info` | Registry 内外部地址 |
| POST `/repository/edit,del` | 镜像专属编辑和删除 |
| POST `/repository/tags/list,del,batch-del` | 镜像 tag 操作 |

镜像前端通用资源请求固定传 `resource_type: "repository"`。列表沿用 `setting.registry.list_ignore_namespaces` 过滤。`CheckNamespacePermission` 位于 controller/common/namespace_permission.go，服务于列表 / 创建的管理准入，不替代 scope 鉴权。

## 5. 迁移与验证

历史镜像继续使用现有 visible_type，resource_type 默认 repository。启动表结构更新后，`Resource.MigrateLegacy` 根据旧 document_library 用户授权补齐文档资源状态与所有者；已有资源不覆盖，旧文档匿名授权转为公开状态后移除。此处不迁移文档正文。

旧制品 OCI namespace 迁移、文档正文迁移、镜像清理分别由对应命令处理，不在运行时权限检查中执行。资源服务与业务服务的文件 / 数据库操作没有统一事务，不能将多次 HTTP 成功或失败理解为整体原子提交 / 回滚。

定向回归命令：

```sh
go test ./app/registry/logic/common ./app/registry/logic/repository \
  ./app/registry/http/controller/common ./app/registry/http/controller/openapi \
  -run 'Test(Resource|RegistryPermission|Namespace)' -count=1
```

本次审查中以上测试通过，并使用隔离临时 SQLite 对照旧镜像判断分支验证了 28,000 组组合：资源缺失或各公开状态、namespace 缺失或各公开状态、本人 / 他人精确授权、namespace 授权、匿名 / 普通用户 / 创始人 / Registry 超级管理员、多动作请求，以及同路径文档资源干扰，放行 / 拒绝结果一致。组合审查使用临时测试，不作为仓库常驻测试命令；此结论不代表已连接真实 Registry 完成推拉验收。
