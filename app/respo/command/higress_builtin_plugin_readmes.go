package command

import "fmt"

func higressPluginReadme(name, title, fallback string) string {
	summary := higressBuiltinPluginReadmeSummaries[name]
	if summary == "" {
		summary = fallback
	}
	return fmt.Sprintf("# %s\n\n%s\n", title, summary)
}

// Summaries are distilled from the Chinese README files shipped by Higress
// Console. They are embedded so importing artifacts does not depend on GitHub.
var higressBuiltinPluginReadmeSummaries = map[string]string{
	"ai-agent":            "提供可定制的 API AI Agent，可把 GET、POST 接口作为工具交给大模型调用，并支持多轮对话、流式和非流式响应。使用前需要配置大模型服务、外部 API 及相应认证信息。",
	"ai-cache":            "缓存大模型的流式或非流式响应，减少相似问题的响应时间和模型调用成本。请求携带 `x-higress-skip-ai-cache: on` 时可跳过缓存读取和写入。",
	"ai-data-masking":     "对 OpenAI 对话字段、指定 JSONPath 或完整请求和响应进行敏感信息检测、拦截、替换与还原，支持系统词库、自定义敏感词和正则脱敏规则。",
	"ai-history":          "根据请求头识别用户，将历史对话保存到 Redis，并在后续请求中自动补充上下文；也支持通过查询接口读取用户历史记录。",
	"ai-intent":           "调用大模型识别用户请求所属的业务意图或领域，可用于把请求分配给更合适的模型、服务或 Agent。",
	"ai-json-resp":        "按照内置或自定义 JSON Schema 将大模型回答转换为结构化 JSON，便于后续插件和业务处理；当前仅支持非流式响应。",
	"ai-load-balancer":    "为大模型服务提供可插拔负载均衡，支持基于 Redis 的全局最少请求、Prompt 前缀缓存亲和以及 least-busy 策略；关闭插件后回退到服务自身的负载均衡。",
	"ai-prompt-decorator": "在大模型请求的用户提示词前后自动插入预设 Prompt，适合统一角色设定、回复语言和回答约束。",
	"ai-prompt-template":  "使用模板和变量快速构造固定格式的大模型请求，适合复用相同结构的 Prompt。",
	"ai-proxy":            "提供统一的 OpenAI 兼容接口，并将请求转换到不同大模型厂商协议；支持文生文和 Embedding 等场景，具体能力取决于所选提供商。",
	"ai-quota":            "按认证后的 consumer 分配和消耗固定 Token 配额，并提供配额查询、刷新和增减能力。需要配合认证插件、AI 统计插件以及 Redis 使用。",
	"ai-rag":              "对接通义千问和阿里云 DashVector，在调用大模型前检索相关文档并注入上下文，实现检索增强生成；支持设置召回数量和距离阈值。",
	"ai-search":           "聚合 Google、Bing、Arxiv、Elasticsearch 等搜索引擎的实时结果并注入提示词，提高模型回答的准确性和时效性，也可在回答中附带引用来源。",
	"ai-security-guard":   "对接阿里云内容安全服务，检测大模型的输入和输出内容，可分别控制请求检测和响应检测。",
	"ai-statistics":       "为大模型请求提供指标、日志和链路追踪能力，默认统计输入/输出 Token、首 Token 延迟和总耗时，并支持从请求或响应中提取自定义属性。通常接在 AI 代理插件之后。",
	"ai-token-ratelimit":  "按 URL 参数、请求头、客户端 IP、consumer 或 Cookie 等键值统计 Token 用量并限流，状态存储在 Redis 中。",
	"ai-transformer":      "通过大模型理解自然语言指令并修改 HTTP 请求或响应的 Header 和 Body，无需编写转换代码。",
	"basic-auth":          "基于 HTTP Basic Auth 对请求进行认证鉴权，可配置多个调用方凭证以及允许访问的 consumer。",
	"bot-detect":          "识别搜索引擎和自动化爬虫等机器人流量，并按配置返回拦截状态码和提示信息。",
	"cache-control":       "根据 URL 文件后缀为响应添加 `Expires` 和 `Cache-Control`，用于控制浏览器对图片、静态资源等内容的缓存时间。",
	"cors":                "为服务端统一生成跨域资源共享响应头，可配置允许的来源、方法、请求头、暴露头、凭证和预检缓存时间。",
	"custom-response":     "生成自定义 HTTP 状态码、响应头和响应体，可用于 Mock 接口，也可针对限流等特定状态返回统一内容。",
	"de-graphql":          "把 REST 风格的 URI 和参数映射为 GraphQL 查询，使客户端能够以传统 HTTP 接口方式访问 GraphQL 上游。",
	"ext-auth":            "在转发业务请求前调用外部 HTTP 授权服务判断请求是否允许通过，能力参考 Envoy ext_authz 过滤器。",
	"frontend-gray":       "按照用户标识和自定义标签把前端请求分配到基线或灰度版本，适用于 A/B 实验、灰度发布和快速回滚。",
	"geo-ip":              "根据来源 IP 查询地理位置信息，并通过请求属性和新增请求头传递给后续插件或上游服务。",
	"hmac-auth":           "使用 HMAC 算法为 HTTP 请求生成和校验不可伪造的签名，实现调用方身份认证和访问控制。",
	"ip-restriction":      "通过白名单或黑名单限制服务和路由访问，支持单个 IP、多个 IP 以及 CIDR 网段。",
	"jwt-auth":            "从 URL 参数、请求头或 Cookie 中解析并验证 JWT，同时识别不同调用方并为其配置独立的 JWT 凭证。",
	"key-auth":            "从 URL 参数或请求头提取 API Key，对调用方进行身份认证并校验其访问权限。",
	"key-rate-limit":      "按 URL 参数或 HTTP 请求头中的指定键值执行请求限流，可分别配置每秒、每分钟等限额。",
	"mcp-server":          "基于 Model Context Protocol 将已有 REST API 转换为 AI 助手可调用的工具，并复用 Higress 的认证、鉴权、限流和可观测能力。",
	"model-mapper":        "按精确匹配、前缀通配或兜底规则改写 LLM 请求中的 model 参数，将客户端模型名映射为服务提供商支持的模型名。",
	"model-router":        "提取 LLM 请求中的 model 或 provider 并写入指定请求头，供网关路由规则选择对应模型服务；也可移除 model 中的 provider 前缀。",
	"oauth":               "按照 RFC 9068 签发 OAuth2 JWT Access Token，为配置的调用方提供客户端凭证认证和令牌获取能力。",
	"oidc":                "实现 OpenID Connect 登录认证，支持 CSRF 防护、注销端点和刷新令牌；认证通过后会向上游请求写入包含 Access Token 的 Authorization 头。",
	"opa":                 "调用 Open Policy Agent 执行策略判断，把统一的访问控制规则应用到网关请求。",
	"request-block":       "根据 URL、请求头或请求体中的特征拦截 HTTP 请求，可用于隐藏管理入口或阻止包含特定内容的访问。",
	"request-validation":  "在请求转发到上游前，使用 JSON Schema 校验请求 Body 和 Header；校验失败时返回自定义状态码和提示。",
	"traffic-tag":         "根据权重、请求头、参数等条件组合匹配流量，并添加指定请求头完成流量染色，供后续路由和灰度策略使用。",
	"transformer":         "转换请求和响应的 Header、查询参数及 Body，支持删除、重命名、更新、添加、追加、映射和去重等操作。",
	"waf":                 "使用 ModSecurity 规则引擎检测并拦截可疑请求，支持自定义安全规则和 OWASP Core Rule Set，为站点提供基础 Web 防护。",
}
