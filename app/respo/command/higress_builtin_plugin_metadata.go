package command

const (
	higressAIPluginIcon        = "https://img.alicdn.com/imgextra/i1/O1CN018iKKih1iVx287RltL_!!6000000004419-2-tps-42-42.png"
	higressAuthPluginIcon      = "https://img.alicdn.com/imgextra/i4/O1CN01BPFGlT1pGZ2VDLgaH_!!6000000005333-2-tps-42-42.png"
	higressSecurityPluginIcon  = "https://img.alicdn.com/imgextra/i1/O1CN01jKT9vC1O059vNaq5u_!!6000000001642-2-tps-42-42.png"
	higressTransformPluginIcon = "https://img.alicdn.com/imgextra/i3/O1CN01bAFa9k1t1gdQcVTH0_!!6000000005842-2-tps-42-42.png"
)

type higressBuiltinPluginMetadata struct {
	Title       string
	Description string
	IconURL     string
}

func higressPluginMetadata(name string) higressBuiltinPluginMetadata {
	metadata, exists := higressBuiltinPluginMetadataList[name]
	if exists {
		return metadata
	}
	return higressBuiltinPluginMetadata{Title: higressPluginTitle(name)}
}

// Metadata is copied from Higress Console's built-in plugin spec.yaml files.
var higressBuiltinPluginMetadataList = map[string]higressBuiltinPluginMetadata{
	"ai-agent": {
		Title: "AI智能体", Description: "通过零代码实现智能体应用的快速构建，支持大模型与外部服务 API 的交互和调用。", IconURL: higressAIPluginIcon,
	},
	"ai-cache": {
		Title: "AI缓存", Description: "缓存大语言模型的响应结果，显著降低相似问题的响应时延并节省成本。", IconURL: higressAIPluginIcon,
	},
	"ai-data-masking": {
		Title: "AI数据脱敏", Description: "对请求/响应中的敏感信息进行拦截、替换、还原", IconURL: higressAIPluginIcon,
	},
	"ai-history": {
		Title: "AI历史对话", Description: "自动缓存对应用户的历史对话，在后续对话中自动填充到上下文。", IconURL: higressAIPluginIcon,
	},
	"ai-intent": {
		Title: "AI意图识别", Description: "智能判断用户请求与某个领域或 agent 的功能契合度，从而提升不同模型的应用效果和用户体验。", IconURL: higressAIPluginIcon,
	},
	"ai-json-resp": {
		Title: "AI格式化", Description: "LLM响应结构化插件，用于根据默认或用户配置的Json Schema对AI的响应进行结构化，以便后续插件处理。注意目前只支持非流式响应。", IconURL: higressAIPluginIcon,
	},
	"ai-load-balancer": {
		Title: "AI负载均衡", Description: "对LLM服务提供热插拔的负载均衡策略，如果关闭插件，负载均衡策略会退化为服务本身的负载均衡策略（轮训、本地最小请求数、随机、一致性hash等）", IconURL: higressAIPluginIcon,
	},
	"ai-prompt-decorator": {
		Title: "AI提示词", Description: "在用户输入的提示词前后添加额外的修饰，简化用户与大语言模型的交互。", IconURL: higressAIPluginIcon,
	},
	"ai-prompt-template": {
		Title: "AI提示词模板", Description: "基于模板快速构建固定格式的提示词。", IconURL: higressAIPluginIcon,
	},
	"ai-proxy": {
		Title: "AI代理", Description: "实现了基于 OpenAI API 规范的代理功能，通过统一的接口调用不同的 AI 服务提供商。", IconURL: higressAIPluginIcon,
	},
	"ai-quota": {
		Title: "AI配额管理", Description: "根据分配固定的 quota 进行 quota 策略限流，同时支持 quota 管理能力，包括查询 quota、刷新 quota、增减 quota。", IconURL: higressAIPluginIcon,
	},
	"ai-rag": {
		Title: "AI检索增强生成", Description: "通过对接阿里云向量检索服务（DashVector）简化 RAG 应用的开发，优化大模型的生成内容。", IconURL: higressAIPluginIcon,
	},
	"ai-search": {
		Title: "AI搜索增强", Description: "higress 支持通过集成搜索引擎（夸克/Google/Bing/Arxiv/Elasticsearch等）的实时结果，增强DeepSeek-R1等模型等回答准确性和时效性", IconURL: higressAIPluginIcon,
	},
	"ai-security-guard": {
		Title: "AI内容安全", Description: "基于阿里云内容安全服务对大模型的输入输出进行安全检测。", IconURL: higressAIPluginIcon,
	},
	"ai-statistics": {
		Title: "AI统计", Description: "提供了对 token 用量的统计信息，包括日志、监控以及告警。", IconURL: higressAIPluginIcon,
	},
	"ai-token-ratelimit": {
		Title: "AIToken限流", Description: "基于特定键值实现 token 限流，键值来源可以是 URL 参数、HTTP 请求头、客户端 IP 地址等。", IconURL: higressAIPluginIcon,
	},
	"ai-transformer": {
		Title: "AI请求响应转换", Description: "无须编写代码，使用自然语言的方式对网关的请求/响应进行修改。", IconURL: higressAIPluginIcon,
	},
	"basic-auth": {
		Title: "Basic认证", Description: "实现基于 HTTP Basic Auth 标准进行认证鉴权的功能。", IconURL: higressAuthPluginIcon,
	},
	"bot-detect": {
		Title: "机器人拦截", Description: "用于识别并阻止互联网爬虫对站点资源的爬取。", IconURL: higressSecurityPluginIcon,
	},
	"cache-control": {
		Title: "浏览器缓存控制", Description: "为响应头部添加 Expires 和 Cache-Control 头部，从而方便浏览器对特定后缀的文件进行缓存，例如 jpg、png 等图片文件。", IconURL: higressTransformPluginIcon,
	},
	"cors": {
		Title: "跨域资源共享", Description: "为服务端启用 CORS（Cross-Origin Resource Sharing，跨域资源共享）的返回 HTTP 响应头。", IconURL: higressSecurityPluginIcon,
	},
	"custom-response": {
		Title: "自定义应答", Description: "支持配置自定义的响应，包括自定义 HTTP 应答状态码、HTTP 应答头，以及 HTTP 应答 Body。", IconURL: higressTransformPluginIcon,
	},
	"de-graphql": {
		Title: "DeGraphQL", Description: "将 Restful API 转换为 GraphQL 请求。", IconURL: higressTransformPluginIcon,
	},
	"ext-auth": {
		Title: "外部认证", Description: "实现了向外部授权服务发送鉴权请求，以检查客户端请求是否得到授权。", IconURL: higressAuthPluginIcon,
	},
	"frontend-gray": {
		Title: "前端灰度", Description: "实现了前端用户灰度的的功能，通过此插件，不但可以用于业务 A/B 实验，同时通过可灰度配合可监控，可回滚策略保证系统发布运维的稳定性。", IconURL: higressTransformPluginIcon,
	},
	"geo-ip": {
		Title: "IP地理位置", Description: "根据用户 IP 查询出地理位置信息，然后通过请求属性和新添加的请求头把地理位置信息传递给后续插件。", IconURL: higressTransformPluginIcon,
	},
	"hmac-auth": {
		Title: "HMAC认证", Description: "基于 HMAC 算法为 HTTP 请求生成不可伪造的签名，并基于签名实现身份认证和鉴权。", IconURL: higressAuthPluginIcon,
	},
	"ip-restriction": {
		Title: "IP限制", Description: "通过将 IP 地址列入白名单或黑名单来限制对服务或路由的访问。", IconURL: higressTransformPluginIcon,
	},
	"jwt-auth": {
		Title: "JWT认证", Description: "实现了基于 JSON Web Token 进行认证鉴权的功能，支持从 HTTP 请求的 URL 参数、请求头、Cookie 字段解析 JWT，同时验证该 Token 是否有权限访问。", IconURL: higressAuthPluginIcon,
	},
	"key-auth": {
		Title: "Key认证", Description: "基于 API Key 实现身份认证和鉴权。", IconURL: higressAuthPluginIcon,
	},
	"key-rate-limit": {
		Title: "基于Key限流", Description: "根据特定键值实现限流，键值来源可以是 URL 参数、HTTP 请求头。", IconURL: higressTransformPluginIcon,
	},
	"mcp-server": {
		Title: "MCP服务器", Description: "托管MCP服务器，实现AI工具集成。", IconURL: "https://img.alicdn.com/imgextra/i1/O1CN01wv8H4g1mS4MUzC1QC_!!6000000004952-2-tps-1764-597.png",
	},
	"model-mapper": {
		Title: "AI模型映射", Description: "实现了将 LLM 协议中的模型参数取值按照规则进行映射的功能。", IconURL: higressAIPluginIcon,
	},
	"model-router": {
		Title: "AI模型路由", Description: "实现了基于 LLM 协议中的 model 参数路由的功能", IconURL: higressAIPluginIcon,
	},
	"oauth": {
		Title: "OAuth2认证", Description: "基于 OAuth2 实现身份认证和鉴权。", IconURL: higressAuthPluginIcon,
	},
	"oidc": {
		Title: "OIDC认证", Description: "实现基于 OpenID Connect 标准的用户身份验证。", IconURL: higressAuthPluginIcon,
	},
	"opa": {
		Title: "OPA", Description: "实现了 OPA 策略控制", IconURL: higressTransformPluginIcon,
	},
	"request-block": {
		Title: "请求屏蔽", Description: "基于 URI、请求头等特征屏蔽 HTTP 请求，可以用于防护部分站点资源不对外部暴露。", IconURL: higressTransformPluginIcon,
	},
	"request-validation": {
		Title: "请求校验", Description: "提前验证向上游服务转发的请求，可以验证请求的 Body 以及 Header 的数据。", IconURL: higressTransformPluginIcon,
	},
	"traffic-tag": {
		Title: "流量染色", Description: "根据权重或特定请求内容通过添加特定请求头的方式对请求流量进行标记。", IconURL: higressTransformPluginIcon,
	},
	"transformer": {
		Title: "请求响应转换", Description: "对请求/响应头、请求查询参数、请求/响应体参数进行转换。", IconURL: higressTransformPluginIcon,
	},
	"waf": {
		Title: "WAF防护", Description: "支持基于 OWASP ModSecurity Core Rule Set (CRS) 的 WAF 规则配置。", IconURL: higressSecurityPluginIcon,
	},
}
