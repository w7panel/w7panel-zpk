# 测试证书
openssl genpkey -algorithm RSA -out registry-key.pem -pkeyopt rsa_keygen_bits:4096
openssl req -new -x509 -key registry-key.pem -out registry-cert.pem -days 365 -subj "/CN=registry.w7.com"

## Helm Sidecar 接入

上传或导入 Helm 包时，Chart 作者需主动接入 sidecar 模板插槽并声明实际运行账号，参考 [Helm Chart 接入 Sidecar 示例](HELM_SIDECAR_INTEGRATION.md)。完整打包协议见 [Helm Sidecar 集成约定](HELM_SIDECAR.md)。

## 修改制品为环境应用分类和原生应用类型

在 ZPK 服务所在环境中执行（`server` 为服务端可执行文件，不是独立上传 CLI）：

```sh
./server respo:set-environment-category --identifie=w7-phptool
```

命令使用服务当前配置的数据库，按标识查找制品，将原分类替换为“环境应用”，同时将最新版本的根 `manifest.yaml` 中 `application.type` 改为 `docker`（原生应用）。历史版本、子应用和其他配置保持原样，重复执行不会重写已经是原生应用的 manifest。

分类须存在于当前配置的制品市场中；分类保存失败时会恢复原 manifest，恢复失败会明确报错，数据库迁移期间禁止修改。需要指定服务配置时可附加 `--config-file /path/to/config.yaml`。类型修改沿用配置编辑器的 manifest 保存方式，不会自动发布 OCI、重新打包 Helm 或同步修改市场商品；要让安装包使用新类型，需再按原流程打包发布。

## 制品市场菜单

制品存在商品订单菜单时，仓库 `info` 接口会基于当前 Helm 包状态和 Bindings 内容生成缓存键，覆盖包内 MicroApp 的 `name: market` 菜单及独立的外部 iframe `backend_config`。没有市场 Binding 时直接返回原 Helm 包地址，不解包或重打；相同基础包和 Bindings 复用已有动态包，内容变化时才重新生成。动态包与 `PackFormulaToHelmAndPack` 产物位于同一目录，文件名格式为 `{原文件名去扩展名}-{hash}.tgz`，空闲 24 小时后清理。

manifest 使用 `application.order` 声明 MicroApp 顺序。生成的 MicroApp 会将该值原样写入 `metadata.labels["w7.cc/order"]`：主应用为 `0`，子应用在新建、导入或删除后按当前列表顺序重新编号为 `1...n`。ZPK Market 动态 MicroApp 固定使用 `9999`。打包过程不再根据 `platform.depends` 推导或改写顺序。

`market` 菜单默认只对 `founder` 显示，使用现有 MicroApp 字段，不新增 `external_services`、`roles`、`icon` 或 `key` 协议字段。

非网关插件制品的默认前端注入表包含 `reverse_dependent_apps: ${system.reverse_dependent_apps}`，值由面板根据 MicroApp 的反向依赖标签注入；每项包含 `appgroup`、`identifie`、`type`、`title`、`version`。

市场域名只写入 `backend_config[role=zpk-market].backend_url`，菜单 `do` 仅写入 `#/user-orders?...` 路由；应用详情页通过 Binding 名称选择同名运行配置。

市场前端地址通过环境变量 `DEPOT_MARKET_FRONTEND_URL` 配置，默认值为 `https://zm.w7.com`。

### 安装授权 HTML 提示

制品 `respo/info` 通过查询参数 `notice_format=html` 协商 HTML 提示。只有主动声明支持的请求，其试用到期、未找到有效已购订单、VIP 未开通/等级不足/过期及退款处理中限制才返回 HTTP 403、`code: ZPK_HTML_NOTICE`。`error` 保留纯文本提示，`data` 提供 `title`、`html`、`reason`、`goods_id`、`order_sn`、`trial_expire_at`、`restriction` 和 `market_url`。

未携带该参数或参数值不是 `html` 时，试用到期仍返回原 HTTP 403 / `ZPK_TRIAL_EXPIRED` 及原消息、元数据，其他授权限制返回原 HTTP 500 / 数字错误码的纯文本错误（VIP 保留具体原因），不返回 HTML。绑定冲突始终保持原 HTTP 409。新面板后端在请求 ZPK 时添加能力参数；旧 ZPK 忽略未知参数，新面板继续处理原有响应。因此可先升级 ZPK 或先升级面板，不要求同步部署；前后端分开部署面板时应先更新支持 HTML 的前端，再更新声明能力的后端。

HTML 由 ZPK 生成并转义动态文案，按市场的 Hash 路由生成购买/续费入口 `/#/site-detail/{goods_id}`、VIP 入口 `/#/vip` 和退款订单入口 `/#/user-center/orders`，保留市场前端部署路径前缀；仅接受 HTTP(S) 且不含用户名密码的市场地址，无有效地址时不生成链接。面板使用已实现的白名单过滤及 Shadow DOM 内容弹框展示，无需为每种业务新增弹框。

市场返回独立授权错误码，SDK 仅在安装及升级校验的市场业务响应中按码识别，不匹配文案；网络、签名、解析及数据库异常不会被转成购买提示。正常返回的 `service_expire` 及可用历史版本规则保持不变，不因展示调整额外禁止安装。域名和应用标识绑定冲突继续返回 HTTP 409 及原结构化字段。

示例响应：

```json
{
  "code": "ZPK_HTML_NOTICE",
  "error": "该应用需要 VIP2，请先开通 VIP",
  "data": {
    "title": "需要开通 VIP",
    "reason": "vip_required",
    "html": "<p>该应用需要 VIP2，请先开通 VIP</p><a href=\"https://zm.w7.com/#/vip\">去购买 VIP</a>",
    "market_url": "https://zm.w7.com/#/vip"
  }
}
```

面板在安装阶段调用制品 `info` 时传入应用域名和规范化应用标识，ZPK 将二者写入加密 ticket。安装完成通知不再单独传递应用标识，ZPK 解票后再把 ticket 中的 `domain` 和 `app_identify` 传给制品市场订单核销接口。

订单安装或升级预检会返回已绑定的 `panel_url`、`panel_device_sn` 和 `app_identify`；校验失败时通过 `conflict_reason` 区分 `domain_mismatch`（域名不一致）和 `app_identify_exists`（已有应用标识绑定）。

绑定冲突使用 HTTP 409 返回结构化 `data`，其中包含 `conflict_reason`、原绑定 `domain`、`panel_url`、`panel_device_sn` 和 `app_identify`，供面板安装接口生成可操作的错误提示，并支持跳转原面板定位原应用。用户确认强制覆盖后可通过 `reinstall=true` 重新安装；该标记只允许非升级安装跳过旧绑定，升级仍严格校验应用标识。

网关插件 WasmPlugin 与配置 MicroApp 统一写入相同的 `metadata.labels["w7.cc/group-name"]` 归组关联，不再生成 `w7.cc/plugin-microapp` 注解。

子应用导入会按选择的制品版本读取完整 manifest，并将附件来源纳入本地存储键，避免不同仓库中的同名同版本附件相互覆盖。子文件全部保存成功后才更新主 manifest；导入失败会恢复导入前的主 manifest。带 `from` 的子应用会通过原制品列表接口检查新版本，并支持在编辑页重新导入更新。

制品 `info` 接口会按市场分类设置汇总并返回 `support_cluster=all|main|sub|none`。分类查询的签名与市场请求共用 10 秒超时；超时时返回空值，由旧版面板兼容为不限制安装，其他接口或数据错误仍正常返回。


### 自定义弹框与安装冲突操作


按钮格式为 {"label":"按钮文案","type":"link","url":"完整 HTTP(S) 地址"} 或 {"label":"强制清除并安装","type":"reinstall"}。域名冲突仅展示绑定域名；应用绑定冲突提供原面板卸载链接，非升级时才下发重装动作。面板只执行预先支持的动作，重装必须有本地安装重试上下文、非采集模式且经过固定二次确认；不能用远程 HTML 执行脚本或任意请求。原面板链接复用 /app/apps?uninstallApp=应用标识，应用标识经 URL 编码，所有动态 HTML 内容转义。


退款处理中提示的“查看订单”链接携带 Hash 路由参数 `order_sn`，例如 `https://zm.w7.com/#/user-center/orders?order_sn=ORDER-123`，市场订单页据此筛选相关订单。订单号为空时保留订单列表入口。

提示操作全部包含在 data.html 中，不再使用 actions 字段。绑定冲突显示应用、订单及原绑定信息；域名冲突额外显示当前安装域名，空字段显示“未提供”。退款提示正文显示订单号，并通过“查看此订单”携带订单号跳转。面板过滤后为空的 HTML 统一使用纯文本提示兜底，不再保留旧试用到期或绑定冲突弹框。

市场安装/升级授权拒绝按数字业务码识别：40301 未找到有效订单，40302 未开通 VIP，40303 VIP 等级不足，40304 VIP 过期。不再匹配错误文案；未知码和旧市场纯文字错误保留原错误处理。应先升级市场再升级 ZPK，以维持完整 HTML 提示体验。

提示 HTML 可通过 `<div data-notice-footer>…</div>` 标记底部操作区；面板净化后将该区内容与统一“关闭”按钮并排展示，操作仍由 ZPK HTML 定义。不支持该标记的旧面板继续在正文显示，不改变链接和动作含义。

购买、开通/升级/续费 VIP、查看订单及前往原面板卸载的跳转入口统一使用蓝底白字按钮外观、文字居中且无下划线；链接保留原有新窗口跳转行为。

安装冲突底栏：“前往原面板卸载”使用蓝底白字按钮，“强制清除并安装”使用红底白字危险按钮，按钮之间保持 16px 间距，空间不足时换行。

## 清空镜像

服务端命令 `registry:clear-images` 直接访问指定 Registry，删除指定 namespace
路径下所有可见镜像的全部 tag，包含子路径；`team` 不会匹配 `team-other`。
账号需具有 catalog、读取和删除权限，Registry 需开启删除支持。

```sh
# 密码也可以通过 --password 指定；环境变量可避免进入命令行参数。
export REGISTRY_USERNAME='admin'
read -rs 'REGISTRY_PASSWORD?Registry password: '
export REGISTRY_PASSWORD
./server registry:clear-images --registry 192.168.1.10:5000 --http --namespace team --dry-run
# 确认预览范围后，去掉 --dry-run 执行删除。
./server registry:clear-images --registry 192.168.1.10:5000 --http --namespace team
unset REGISTRY_PASSWORD
```

HTTPS 默认验证证书，自签名证书可用 `--ca /path/to/ca.pem`；
账号可用 `--username` 指定。`--timeout` 默认为 `1m`，用于单次请求（含分页枚举）。
沿用镜像版本列表的删除语义：tag 先解析为 digest，再删除 manifest，
同一 manifest 的多个 tag 会一起删除。远程删除全部成功后，在数据库事务中删除指定 namespace（含子路径）的 namespace、镜像记录及关联 tag、部署规则、部署日志和权限；namespace 和镜像沿用现有软删除约定，默认 namespace 禁止删除。不主动清理 blob，
磁盘空间回收仍由 Registry GC 负责。列出 tag、解析 tag 或删除 manifest 时明确返回不存在（404），会记录并跳过，不影响其他删除或后续数据库清理；权限、网络及其他错误仍按失败处理。其他枚举或解析失败时不开始删除，
删除阶段失败会继续处理其他项并返回错误，已成功删除的内容不会回滚。
`--dry-run` 同时列出数据库清理范围，不写入数据库。远程删除失败时不清理数据库；数据库事务失败时回滚数据库修改，但无法恢复已删除的远程 tag，可重试命令。
数据库为当前启动配置连接的 ZPK 数据库，必须与指定内网 Registry 对应。仅能处理账号可见的 tag，执行期间应停止对目标 namespace 的推送。
该命令仍经过服务端正常启动初始化流程，请使用合适的本地配置，避免误操作本地数据库。

共享 sidecar 资源时，w7panelArtifact.serviceAccountName 指定当前 Pod 账号；子应用只生成独立 RoleBinding/ClusterRoleBinding，引用父应用共享角色，不重复创建 Service 或角色。父子应用须使用相同的共享角色名称。
