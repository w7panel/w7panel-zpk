package command

type higressBuiltinPluginProperties struct {
	Category     string
	Phase        string
	Priority     int
	SupportsRule bool
}

func higressPluginProperties(name string) higressBuiltinPluginProperties {
	properties, exists := higressBuiltinPluginPropertiesList[name]
	if exists {
		return properties
	}
	return higressBuiltinPluginProperties{
		Category:     "custom",
		Phase:        "UNSPECIFIED_PHASE",
		SupportsRule: true,
	}
}

func higressPluginCategory(name string) string {
	return higressPluginProperties(name).Category
}

// Values are copied from Higress Console's built-in plugin spec.yaml files.
var higressBuiltinPluginPropertiesList = map[string]higressBuiltinPluginProperties{
	"ai-agent":               {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 200, SupportsRule: true},
	"ai-cache":               {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 800},
	"ai-data-masking":        {Category: "ai", Phase: "AUTHN", Priority: 991, SupportsRule: true},
	"ai-history":             {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 650, SupportsRule: true},
	"ai-intent":              {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 700, SupportsRule: true},
	"ai-json-resp":           {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 150, SupportsRule: true},
	"ai-load-balancer":       {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 50, SupportsRule: true},
	"ai-prompt-decorator":    {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 450, SupportsRule: true},
	"ai-prompt-template":     {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 500, SupportsRule: true},
	"ai-proxy":               {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 100, SupportsRule: true},
	"ai-quota":               {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 750, SupportsRule: true},
	"ai-rag":                 {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 400, SupportsRule: true},
	"ai-search":              {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 440, SupportsRule: true},
	"ai-security-guard":      {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 300, SupportsRule: true},
	"ai-statistics":          {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 900, SupportsRule: true},
	"ai-token-ratelimit":     {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 600, SupportsRule: true},
	"ai-transformer":         {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 550, SupportsRule: true},
	"basic-auth":             {Category: "auth", Phase: "AUTHN", Priority: 320, SupportsRule: true},
	"bot-detect":             {Category: "security", Phase: "AUTHZ", Priority: 310, SupportsRule: true},
	"cache-control":          {Category: "transform", Phase: "AUTHN", Priority: 420, SupportsRule: true},
	"cluster-key-rate-limit": {Category: "traffic", Phase: "UNSPECIFIED_PHASE", Priority: 20, SupportsRule: true},
	"cors":                   {Category: "security", Phase: "AUTHZ", Priority: 340, SupportsRule: true},
	"custom-response":        {Category: "transform", Phase: "AUTHN", Priority: 910, SupportsRule: true},
	"de-graphql":             {Category: "transform", Phase: "AUTHN", Priority: 430, SupportsRule: true},
	"ext-auth":               {Category: "auth", Phase: "AUTHN", Priority: 360, SupportsRule: true},
	"frontend-gray":          {Category: "transform", Phase: "UNSPECIFIED_PHASE", Priority: 450, SupportsRule: true},
	"geo-ip":                 {Category: "o11y", Phase: "AUTHN", Priority: 440, SupportsRule: true},
	"hmac-auth":              {Category: "auth", Phase: "AUTHN", Priority: 330, SupportsRule: true},
	"ip-restriction":         {Category: "security", Phase: "AUTHN", Priority: 210, SupportsRule: true},
	"jwt-auth":               {Category: "auth", Phase: "AUTHN", Priority: 340, SupportsRule: true},
	"key-auth":               {Category: "auth", Phase: "AUTHN", Priority: 310, SupportsRule: true},
	"key-rate-limit":         {Category: "traffic", Phase: "UNSPECIFIED_PHASE", Priority: 10, SupportsRule: true},
	"mcp-server":             {Category: "ai", Phase: "UNSPECIFIED_PHASE", Priority: 999, SupportsRule: true},
	"model-mapper":           {Category: "ai", Phase: "AUTHN", Priority: 800},
	"model-router":           {Category: "ai", Phase: "AUTHN", Priority: 900},
	"oauth":                  {Category: "auth", Phase: "AUTHN", Priority: 350, SupportsRule: true},
	"oidc":                   {Category: "auth", Phase: "AUTHN", Priority: 350, SupportsRule: true},
	"opa":                    {Category: "auth", Phase: "AUTHN", Priority: 225, SupportsRule: true},
	"request-block":          {Category: "security", Phase: "AUTHZ", Priority: 320, SupportsRule: true},
	"request-validation":     {Category: "traffic", Phase: "AUTHN", Priority: 220, SupportsRule: true},
	"traffic-tag":            {Category: "traffic", Phase: "UNSPECIFIED_PHASE", Priority: 400, SupportsRule: true},
	"transformer":            {Category: "transform", Phase: "AUTHN", Priority: 410, SupportsRule: true},
	"waf":                    {Category: "security", Phase: "AUTHZ", Priority: 330, SupportsRule: true},
}
