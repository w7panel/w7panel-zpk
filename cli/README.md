# ZPK 发布工具

`app` 是原有应用制品（respo）的 CLI 命令名称，复用原 OCI artifact、`files.json`、图标及前端 / 后端 / Helm 附件格式，不增加应用包嵌套层或独立索引协议。文档发布暂不支持。

## 登录与仓库

```sh
zpk login --host registry.example.com --username developer --password '<password>'
```

沿用原登录凭据的本地加密方式，各仓库凭据按 host 分开保存；最近一次登录的 host 是默认仓库。完整远程引用不会使用其他 host 的凭据。旧会话仍兼容。

新 app 命令默认使用 HTTPS。开发环境的 HTTP Registry 需要显式加 `--plain-http`，也可在完整引用中使用 `http://`。匿名拉取无需登录，可通过 `--host` 指定默认仓库。

## 多子应用目录

```text
my-project/
├── frontend/                       # 开发工程，自行组织
├── backend/
└── zpk/                            # 应用交付目录
    ├── manifest.yaml               # 主应用
    ├── README.md                   # 可选
    ├── icon.png                    # 可选
    └── apps/
        ├── demo/                   # 主应用附件，标识与根 manifest 一致
        │   └── assets/
        │       ├── backend/        # 后端 ZIP 内容，按需存在
        │       │   └── config/config.php
        │       └── helm/           # Helm TGZ 内容，保留包内顶层目录
        │           └── demo/
        │               ├── Chart.yaml
        │               ├── values.yaml
        │               └── templates/
        ├── api/
        │   ├── manifest.yaml
        │   └── assets/
        │       └── frontend/
        │           └── index.html
        └── admin/
            ├── manifest.yaml
            └── assets/
                └── frontend/
                    └── index.html
```

根 manifest 使用现有字段声明内置子应用：

```yaml
application:
  identifie: demo
  name: 示例应用
  type: docker
  order: 0
platform:
  depends:
    - identifie: api
      type: in
    - identifie: admin
      type: in
source:
  type: zip
  url: ./apps/demo/assets/backend
```

每个子应用有独立 manifest 与 assets，子目录名须与 `application.identifie` 相同，包内标识唯一。内置依赖和子应用目录必须匹配；`type: out` 保持外部依赖语义，不要求本地目录。`from`、依赖版本、required、order 等现有字段保留，不按文件夹扫描顺序重写 order。

本地附件以解压目录作为编辑来源：主应用 `source.url` 指向后端目录，`web.url` 指向前端目录，`platform.helm.chartName` 指向 Helm 解压目录。`source.type` / `web.type` 仍为 `zip`；push 自动将后端、前端目录打成 ZIP，将 Helm 目录打成 TGZ。OCI 仍沿用一个本地后端 ZIP、多个前端包的原有约定，不扩展多后端包协议。

前端、后端和 Helm 附件统一放在 `apps/<identifie>/assets/`。根 `manifest.yaml` 仍保存主应用配置，主应用附件引用为 `./apps/<主应用标识>/assets/...`；子应用 manifest 内的引用仍为 `./assets/...`。例如 admin 的 `web.url: ./assets/frontend` 指向 `apps/admin/assets/frontend/`。主应用目录只存附件，不再放一份重复的 manifest。远程 HTTP(S) 引用和带 repository 的远程 Helm Chart 保留。普通说明和资源可放在交付目录的任意相对路径，不做目录、扩展名或隐藏文件白名单限制。禁止符号链接、硬链接压缩条目、特殊文件和逃逸路径。开发工程、`.git`、构建缓存等请放在交付目录之外。

已有本地目录若使用根 `assets/`，需将其移到 `apps/<主应用标识>/assets/`，并同步根 manifest 的本地附件引用。原服务端 OCI 包不受目录调整影响，pull 时按新目录恢复。

push 读取主 / 子 manifest 及其本地附件目录，确认打包所需文件可读；业务规范由 ZPK 校验，转换附件引用时保留未知 manifest 字段。CLI 不执行应用构建脚本，开发者先把构建产物写入相应 assets 目录。本地 manifest 的附件引用必须指向解压目录，不支持直接引用 ZIP/TGZ。只有 pull 解包，update 只操作目录和文件，push 最后统一打包。

## 应用推送

```sh
# 从根 manifest 读取标识，使用默认仓库。
zpk app push ./zpk

# 显式指定应用及版本。
zpk app push demo:1.2.3 --path ./zpk

# 上传后请求服务端发布该版本。
zpk app push demo --path ./zpk --publish
```

push 仅使用 `zpk login` 保存的默认仓库，不接受完整仓库引用或 `--host` 覆盖。应用标识映射到 `<登录 host>/<DEPOT_OCI_NAMESPACE，默认 zpk_oci>/<应用标识>`，沿用物理 OCI repository 中连字符转下划线的历史规则。

应用发布版本采用三个非负数字段，可带小写 `v` 前缀，例如 `1.2.3`、`v1.2.3`；不接受缺段版本、文字 tag 或预发布后缀。省略版本或 push 指定 `latest` 时，从远端符合规则的版本中选择最新版并递增 patch，例如 `1.2.3 → 1.2.4`、`v1.2.3 → v1.2.4`；没有符合规则的版本时初始使用 `1.0.0`。同名版本沿用原有更新行为，允许推送新内容。

自动版本与 latest 选择先过滤 `release`、`shared`、`2026.11` 等不符合版本规则的 tag，再复用项目现有的 `SortTagsDesc` 数字排序规则。push 只推送具体版本 tag，不创建或更新远端 `latest` 标签。pull 中的 `latest` 表示按版本号排序最新的已上传 OCI 包，不是数据库业务版本，也不保证 Helm 或市场发布已完成。本地 manifest 中的旧版本字段不会决定自动版本。

版本分配在 CLI 内完成，同一应用的自动版本推送应串行；显式指定同名 tag 时，以仓库最后一次成功写入的内容为准。

上传是完整快照：不继承旧版本的文件、附件或已删除子应用。通过 OCI mount 按 digest 复用同一仓库已有 blob，挂载成功不传输内容；blob 不存在或仓库不支持挂载并返回 202 时，由 ORAS 继续上传本地内容，不维护额外的本地差异记录；附件级增量中，目录内文件变化仍会重新上传该 ZIP/TGZ 附件。打包按固定顺序、固定时间戳生成，保留文件权限和空目录，不记录本机属主与修改时间；未修改的目录可复用已有 blob。失败重试可复用已经上传但尚未打 tag 的内容。

本地 `apps/<id>/manifest.yaml` 转为现有 `files.json` 中的 `<id>/manifest.yaml`，附件沿用现有 media type 和 `file:///Storage/...` 引用。可编辑文件路径用标准 OCI descriptor title annotation 保存；旧工具忽略该 annotation 仍可读取已有应用层。附件目录内部文件只进入所属 ZIP/TGZ，不逐文件增加 OCI 层。普通说明及未引用资源使用标准 MIME 内容层，不新增 ZPK 专有文件协议。

默认只上传 OCI 并由回调同步版本。加 `--publish` 时，CLI 在标准 OCI manifest annotations 中写入 `w7.cc/auto_publish: "true"`；服务端同步版本后校验推送者为应用拥有者或管理员，再按现有设置同步市场商品并将 Helm 构建加入发布队列，不再重新打包或推送 OCI。启用自动同步市场时，使用服务端已保存的制品云端账号绑定，缺少绑定会记录发布失败。`latest` 和 `shared` 回调不创建或发布业务版本。

命令成功表示 OCI 推送完成，携带发布标记不代表版本已发布；最终状态和失败原因在管理端查看。任务在队列中时沿用现有去重，重启恢复 pending 任务。不新增 prepare / publish / status 接口。`--publish` 需要配套服务端支持，旧服务端忽略该 annotation，仍只同步版本。

## 拉取与修改

```sh
zpk app pull demo --path ./demo
zpk app pull demo:1.2.3 --path ./demo-1.2.3
zpk app pull registry.example.com/team/demo:latest --path ./demo-remote
zpk app pull demo --host registry.example.com --path ./demo-anonymous
```

pull 支持短标识或完整远程地址；完整地址保留 namespace，仅转换最后的应用名。pull 未指定版本或指定 `latest` 时，始终查询符合版本规则的 tag 并按现有规则选择最新版，即使远端 latest 别名缺失或没有更新，也能取到最新版本。显式版本沿用 push 的三个数字段规则（可带小写 `v` 前缀），不存在时直接报错；历史非数字 tag 不参与拉取及自动版本选择。pull 下载并校验附件后安全解压为目录；OCI 路径 annotation 表示本地附件目录，没有 annotation 时按附件类型使用 frontend、backend、helm 目录。Helm 保留包内 Chart 顶层目录。远程 HTTP(S) 引用和远程 Chart 仓库配置继续保留，不额外下载。

权限遵循 Registry 现有授权：应用对应的 OCI 仓库创建时沿用“跟随 namespace”，制品 namespace 的默认配置为公开读，因此未发布的应用也可匿名拉取。发布商品成功后，按本次商品授权费用 `install_service_fee` 设置单个仓库权限，价格为 0 时为公有读、私有写，可以匿名拉取，否则为私有。修改价格后再次发布商品时同步，单独保存价格不改变仓库权限。namespace 权限和创建用户的原有授权保持不变。公开读覆盖该仓库全部 tag，包括历史版本和已上传未发布版本。

非公开应用需要登录目标仓库且具有该应用的 pull 权限，登录任意账号不会自动获得下载权限。完整远程地址由远端 Registry 执行相同鉴权，CLI 不根据售价字段绕过服务端校验。免费改收费并成功发布商品后，新申请的匿名拉取凭证会被拒绝；已签发的 Registry token 仍沿用现有有效期。

pull 先下载并校验 blob，再由内部 unpack 从本地 blob 恢复 manifest、普通资源和解压附件，完成后将临时目录移到目标位置，目标目录必须不存在，不覆盖本地修改。push / pull 不写入同步状态文件，远端出现新版本后仍可直接推送本地目录；增量上传通过 OCI mount 按 digest 复用仓库已有 blob。

日常直接编辑 manifest、附件目录中的文件或重新构建 assets 后即可 push，不需要每次先 pull。pull 得到的是设计配置与发布产物，不保证包含前端、后端原始开发工程。

## 本地更新

统一使用 `app update <backend|frontend|helm|manifest>`，仅修改本地交付目录，不登录或上传。`--path` 默认当前目录；不传 `--app` 时选择主应用，也可显式填写主应用标识。子应用使用 `--app <identifie>`，本地后端仍只支持主应用。

```sh
# 拉取一个完整版本。
zpk app pull demo:latest --path ./demo

# 完整目录更新：复制本地目录，完整替换对应附件目录，清除旧文件。
zpk app update backend --path ./demo --dir ./backend
zpk app update frontend --path ./demo --app admin --dir ./dist
zpk app update helm --path ./demo --dir ./chart

# 单文件更新：--target 相对于附件目录，Helm 需带包内 Chart 顶层目录。
zpk app update backend --path ./demo --file ./config.php --target config/config.php
zpk app update frontend --path ./demo --app admin --file ./index.html --target index.html
zpk app update helm --path ./demo --file ./values.yaml --target demo/values.yaml

# 新增文件必须显式加 --create，默认目标必须已经存在。
zpk app update backend --path ./demo --file ./extra.php --target lib/extra.php --create

# 整份覆盖 manifest。
zpk app update manifest --path ./demo --app admin --file ./manifest.yaml

# 修改 manifest 节点，默认 JSON Merge Patch，也支持 YAML patch 文件。
zpk app update manifest --path ./demo --patch '{"application":{"name":"新版应用"}}'
zpk app update manifest --path ./demo --app admin --patch-file ./patch.yaml

# JSON Patch 可精确操作数组及含特殊字符的键，路径遵循 JSON Pointer。
zpk app update manifest --path ./demo --type json \
  --patch '[{"op":"replace","path":"/application/name","value":"新版应用"}]'

# 所有修改完成后统一上传。
zpk app push --path ./demo
```

`--dir`、`--file`、`--patch`、`--patch-file` 四选一。附件完整更新使用 `--dir`；附件单文件更新使用 `--file --target`；manifest 覆盖使用 `--file`，不支持 `--dir`。`--target` / `--create` 仅适用于附件；`--type merge|json` 仅适用于 manifest 局部修改。Merge Patch 的对象递归合并、数组整体替换、`null` 删除字段；JSON Patch 支持 `add/remove/replace/test/move/copy`。更新保留未知配置字段，但 YAML 重写可能改变排版及注释，不支持通过 update 重命名应用标识。

命令按 manifest 引用定位附件，不要求目录使用固定名称。目录更新也可为原本没有附件、或使用远程地址的应用导入本地附件，并同步 manifest 引用；远程 Helm 切换为本地时清除 `repository`，保留其他 Helm 配置。单文件更新要求已有本地目录，update 不接受压缩包、不执行解包或打包。Helm 的 `--dir` 目录内需保留 Chart 顶层目录，例如 `chart/demo/Chart.yaml`。附件文件保留原有目录层级和普通权限，单文件替换保留目标权限。

修改只暂存受影响的文件或目录，完成基础输入和路径检查后再替换原路径，不复制、打包或复检整个项目。manifest 更新检查 YAML 可解析和应用标识不变，允许在本地分步编辑配置；完整业务规则由 ZPK 处理，push 只检查打包所需的主子 manifest 和附件目录是否可读取。落盘失败时尽力恢复原文件并保留临时备份，报错中给出备份位置；多文件落盘不宣称跨文件原子事务。

## 镜像推送

```sh
zpk image push demo:local
```

目标固定为 `<登录 host>/<登录用户名>/<本地镜像名>:<tag>`，未指定 tag 时采用 latest，不提供 `--name` 或 `--host` 覆盖。命令通过 Docker 执行登录、tag、push；凭据经标准输入传递，并使用临时 Docker 配置，不修改现有 Docker 登录文件。镜像分层去重复用 Docker 行为。需要已安装并运行 Docker；不负责构建镜像或安装应用。

## 代码组织

`app` 是命令名，应用业务统一放在 `logic/respo`。包内按职责命名，不再混用 `app_*`、`formula_*` 前缀：

```text
app/application/logic/
├── common/                 # 会话、仓库和版本
├── respo/
│   ├── archive.go          # ZIP/TGZ 安全解压和稳定打包
│   ├── update.go           # 本地附件、文件及 manifest 更新
│   └── oci.go              # 应用 push / pull、目录打包与解包
└── image/                  # 镜像目标引用、Docker 登录与推送
```

应用流程对外使用 `respo.Push`、`respo.Pull`、`respo.Update`，`pack` / `unpack` 为内部方法，分别只由 push / pull 调用。临时打包目录由 push 管理，下载和临时 blob 由 pull 管理，unpack 只处理本地内容；appPackage 仅保存打包结果。推送始终以本地完整目录为准，不维护旧的附件合并流程。

OCI 传输直接使用主项目 `common/logic.PushOciToRemote`、`GetOciManifest` 和 `UnPackOciToLocal`，推送方法统一处理 blob 复用、manifest 生成和版本 tag；CLI 负责自动版本选择和拉取最新版的解析。命令入口直接设置请求超时，负责参数组合、版本格式和应用标识校验，并在调用处完成应用仓库映射及历史连字符 / 下划线转换。

各领域依赖 CLI 的 `common`，不相互依赖。CLI 共享层只提供会话、仓库和版本能力。manifest 解析、字段读写和标识校验直接使用主项目 `common/logic/manifest.go`；路径解析、文件读写和复制使用主项目 `common/function`，临时文件采用通用的 `.<目标文件名>.tmp-` 前缀；后续文档可复用仓库、版本和传输能力。

## 命令与验证

旧 `use`、`attach add`、顶层 `push`、`list` 命令已移除。统一改用 `app push --path`、`app pull`、`app update`；推送使用登录仓库，拉取支持完整远程地址，版本可在标识后指定，不再保存当前应用和待上传附件状态。已有登录凭据仍可读取；应用通过 `app pull` 转为解压目录，再修改和推送；不提供本地压缩包目录的兼容流程。

`app list` 不再提供；自动版本与 latest 所需的内部版本查询继续保留。文档命令待实现时再增加对应代码包。

服务端推送回调只识别应用包、同步版本及失效缓存，跳过 latest 别名；不参与 CLI 打包或增量合并。服务端打包、附件恢复均保持原有逻辑，一个后端 ZIP、多个前端包沿用原媒体类型约定。管理端重新打包可能丢失 CLI 附加的普通资源及原始路径 annotation。

`make build` / `make build-compress` 的产物为 `bin/zpk_darwin_amd64_v2`、`bin/zpk_linux_amd64_v2` 和 `bin/zpk_windows_amd64_v2.exe`。Release 工作流上传这些文件，并使用 Linux CLI 拉取已有 `w7-zpkv2` 应用，更新解压后的 Helm 目录和前端构建目录，再通过 `app push --publish` 上传新版本并请求服务端自动发布；最终发布状态以服务端为准。

```sh
# 在 cli/ 下执行；测试只访问本地模拟 Registry 和模拟 Docker。
go test ./... -count=1
go test -race ./app/application/logic/... -count=1
go vet ./...
go build ./...
```
