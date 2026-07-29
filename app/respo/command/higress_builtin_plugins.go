package command

import (
	"fmt"
	"strings"

	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
)

const (
	higressBuiltinPluginVersion      = "1.0.0"
	higressBuiltinPluginImageVersion = "2.0.0"
	higressBuiltinPluginRegistry     = "higress-registry.cn-hangzhou.cr.aliyuncs.com/plugins"
)

type higressBuiltinPlugin struct {
	Name        string
	Identifie   string
	Title       string
	Description string
	Image       string
	IconURL     string
	Readme      string
}

// higressBuiltinPlugins returns the plugins listed by Higress Console's
// plugins.properties. Cluster key rate limit is intentionally excluded.
func higressBuiltinPlugins() []higressBuiltinPlugin {
	names := []string{
		"ai-prompt-decorator",
		"ai-prompt-template",
		"ai-rag",
		"ai-search",
		"ai-security-guard",
		"ai-statistics",
		"ai-token-ratelimit",
		"ai-transformer",
		"ai-cache",
		"ai-proxy",
		"ai-history",
		"ai-intent",
		"ai-quota",
		"ai-agent",
		"ai-data-masking",
		"ai-json-resp",
		"ai-load-balancer",
		"model-router",
		"model-mapper",
		"mcp-server",
		"basic-auth",
		"key-auth",
		"oidc",
		"jwt-auth",
		"hmac-auth",
		"ext-auth",
		"oauth",
		"opa",
		"custom-response",
		"transformer",
		"cache-control",
		"de-graphql",
		"geo-ip",
		"frontend-gray",
		"request-block",
		"key-rate-limit",
		"cluster-key-rate-limit",
		"ip-restriction",
		"request-validation",
		"traffic-tag",
		"bot-detect",
		"waf",
		"cors",
	}

	plugins := make([]higressBuiltinPlugin, 0, len(names))
	for _, name := range names {
		if isExcludedHigressBuiltinPlugin(name) {
			continue
		}
		metadata := higressPluginMetadata(name)
		plugins = append(plugins, higressBuiltinPlugin{
			Name:        name,
			Identifie:   higressPluginIdentifie(name),
			Title:       metadata.Title,
			Description: metadata.Description,
			Image:       fmt.Sprintf("oci://%s/%s:%s", higressBuiltinPluginRegistry, name, higressBuiltinPluginImageVersion),
			IconURL:     metadata.IconURL,
			Readme:      higressPluginReadme(name, metadata.Title, metadata.Description),
		})
	}
	return plugins
}

func higressPluginIdentifie(name string) string {
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	compactName := strings.NewReplacer("-", "", "_", "").Replace(normalizedName)
	if normalizedName == "key-rate-limit" {
		compactName = "ratelimit"
	}
	return "w7panel-plugin" + compactName
}

func isExcludedHigressBuiltinPlugin(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return name == "cluster-key-rate-limit"
}

func higressPluginTitle(name string) string {
	parts := strings.Split(name, "-")
	for index, part := range parts {
		switch strings.ToLower(part) {
		case "ai":
			parts[index] = "AI"
		case "api":
			parts[index] = "API"
		case "cors":
			parts[index] = "CORS"
		case "graphql":
			parts[index] = "GraphQL"
		case "hmac":
			parts[index] = "HMAC"
		case "ip":
			parts[index] = "IP"
		case "jwt":
			parts[index] = "JWT"
		case "mcp":
			parts[index] = "MCP"
		case "oauth":
			parts[index] = "OAuth"
		case "oidc":
			parts[index] = "OIDC"
		case "opa":
			parts[index] = "OPA"
		case "rag":
			parts[index] = "RAG"
		case "waf":
			parts[index] = "WAF"
		default:
			parts[index] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

func newHigressBuiltinPluginManifest(plugin higressBuiltinPlugin) commonlogic.Manifest {
	return commonlogic.Manifest{
		Application: commonlogic.Application{
			Name:            plugin.Title,
			Identifie:       plugin.Identifie,
			Description:     plugin.Description,
			Author:          "Higress",
			InstallOnlyOnce: true,
			Type:            commonlogic.GatewayPluginApp,
			Version:         higressBuiltinPluginVersion,
			Annotation: map[string]interface{}{
				"w7.cc/official-app": "true",
			},
		},
		Platform: commonlogic.Platform{
			GatewayPlugin: commonlogic.GatewayPlugin{
				Supports: commonlogic.GatewayPluginSupports{
					Global: true,
					Rule:   true,
				},
				DefaultConfig: higressPluginDefaultConfig(plugin.Name),
				Runtime: commonlogic.GatewayPluginRuntime{
					Driver: commonlogic.GatewayPluginDriverHigressWasmV1,
					Config: map[string]interface{}{
						"url":      plugin.Image,
						"phase":    "UNSPECIFIED_PHASE",
						"priority": 0,
					},
				},
			},
		},
		Version:   3,
		VersionV2: 3,
	}
}
