package logic

import (
	"fmt"
	"html"
	"strings"

	"github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
)

// buildDocumentHomepage describes declared configuration, without parameter values or executable content.
func buildDocumentHomepage(item *formula.Formula) string {
	var output strings.Builder
	title := item.Title
	if title == "" {
		title = item.Manifest.Application.Name
	}
	if title == "" {
		title = item.Name
	}
	fmt.Fprintf(&output, "# %s\n\n", documentHomepageText(title))
	fmt.Fprintf(&output, "制品标识：%s\n\n", documentHomepageText(item.Name))
	if item.Version != "" {
		fmt.Fprintf(&output, "配置版本：%s\n\n", documentHomepageText(item.Version))
	}
	output.WriteString("本首页由制品配置自动生成，随版本发布更新。\n\n")
	if item.Readme != nil && strings.TrimSpace(*item.Readme) != "" {
		output.WriteString("## 应用介绍\n\n")
		output.WriteString(html.EscapeString(strings.TrimSpace(*item.Readme)))
		output.WriteString("\n\n")
	}
	writeDocumentApplication(&output, item.Manifest, "应用说明")
	for _, child := range item.AllManifest {
		if child.Application.Identifie == item.Manifest.Application.Identifie {
			continue
		}
		name := child.Application.Name
		if name == "" {
			name = child.Application.Identifie
		}
		writeDocumentApplication(&output, child, "子应用："+name)
	}
	return output.String()
}

func writeDocumentApplication(output *strings.Builder, manifest *commonlogic.Manifest, title string) {
	fmt.Fprintf(output, "## %s\n\n", documentHomepageText(title))
	application, platform := manifest.Application, manifest.Platform
	kind, usage := documentApplicationType(manifest)
	fmt.Fprintf(output, "类型：%s。%s\n\n", kind, usage)
	if application.Description != "" {
		fmt.Fprintf(output, "%s\n\n", documentHomepageText(application.Description))
	}
	if application.Identifie != "" {
		fmt.Fprintf(output, "应用标识：%s\n\n", documentHomepageText(application.Identifie))
	}
	var features []string
	if application.InstallOnlyOnce {
		features = append(features, "单次安装")
	}
	if application.RegisterSite {
		features = append(features, "注册面板站点")
	}
	if application.ClusterPrivileged {
		features = append(features, "申请集群级权限")
	}
	if manifest.Web.Url != "" {
		features = append(features, "包含前端包")
	}
	if manifest.Source.Url != "" {
		features = append(features, "包含后端源码包")
	}
	if platform.RuntimeClassName != "" {
		features = append(features, "运行时："+platform.RuntimeClassName)
	}
	if platform.Workload.Type != "" {
		features = append(features, "工作负载："+platform.Workload.Type)
	}
	if application.Type == commonlogic.GatewayPluginApp {
		if platform.GatewayPlugin.Supports.Global {
			features = append(features, "支持网关全局配置")
		}
		if platform.GatewayPlugin.Supports.Rule {
			features = append(features, "支持网关规则配置")
		}
	}
	if len(features) > 0 {
		output.WriteString("### 已配置功能\n\n")
		for _, feature := range features {
			fmt.Fprintf(output, "- %s\n", documentHomepageText(feature))
		}
		output.WriteString("\n")
	}
	var rows [][]string
	for _, param := range platform.StartParams {
		if param.Hidden {
			continue
		}
		required := "否"
		if param.Required {
			required = "是"
		}
		rows = append(rows, []string{param.Name, param.Title, param.Description, required})
	}
	writeDocumentTable(output, "启动参数", []string{"参数", "名称", "说明", "必填"}, rows)
	rows = nil
	for _, container := range platform.ContainerV2s {
		var ports []string
		for _, port := range container.Ports {
			protocol := string(port.Protocol)
			if protocol == "" {
				protocol = "TCP"
			}
			ports = append(ports, fmt.Sprintf("%d/%s", port.ContainerPort, protocol))
		}
		role := "运行容器"
		if container.IsInitContainer {
			role = "初始化容器"
		}
		rows = append(rows, []string{container.Name, role, container.Image, strings.Join(ports, "、")})
	}
	writeDocumentTable(output, "容器与端口", []string{"容器", "用途", "镜像", "容器端口"}, rows)
	rows = nil
	for _, ingress := range platform.Ingress {
		for _, route := range ingress.Routes {
			rows = append(rows, []string{ingress.Name, ingress.DomainParam, route.Path, route.Backend.Name, fmt.Sprint(route.Backend.Port)})
		}
	}
	writeDocumentTable(output, "访问入口", []string{"入口", "域名参数", "路径", "后端", "端口"}, rows)
	rows = nil
	for _, binding := range manifest.Bindings {
		for _, menu := range binding.Menu {
			rows = append(rows, []string{binding.Title, menu.Title, menu.Do})
		}
	}
	writeDocumentTable(output, "面板菜单", []string{"角色", "菜单", "路由"}, rows)
	rows = nil
	for _, container := range platform.ContainerV2s {
		for _, mount := range container.VolumeMounts {
			mode := "读写"
			if mount.ReadOnly {
				mode = "只读"
			}
			rows = append(rows, []string{container.Name, mount.Name, mount.MountPath, mode})
		}
	}
	writeDocumentTable(output, "数据挂载", []string{"容器", "卷", "挂载目录", "模式"}, rows)
	rows = nil
	for _, dependency := range platform.Depends {
		required := "否"
		if dependency.Required {
			required = "是"
		}
		relation := "内嵌子应用"
		if dependency.Type == "out" {
			relation = "外部依赖"
		}
		rows = append(rows, []string{dependency.Name, dependency.Identifie, dependency.Version, relation, required})
	}
	writeDocumentTable(output, "应用依赖", []string{"名称", "标识", "版本", "关系", "必需"}, rows)
}

func documentApplicationType(manifest *commonlogic.Manifest) (string, string) {
	if manifest.Application.Type == commonlogic.GatewayPluginApp {
		return "网关插件", "用于扩展网关能力，按声明的全局或规则作用范围配置。"
	}
	if manifest.Platform.Helm.ChartName != "" || manifest.Platform.Helm.Repository != "" || len(manifest.Platform.Helm.DependYamls) > 0 {
		return "Helm 应用", "使用提供的 Helm Chart 或 Kubernetes YAML 部署，运行资源由包内模板决定。"
	}
	switch manifest.Application.Type {
	case commonlogic.Docker_App:
		return "容器应用", "按声明的容器配置部署，安装时填写启动参数并配置所需资源。"
	case commonlogic.HelmApp:
		return "Helm 应用", "使用 Helm Chart 或 Kubernetes YAML 部署；当前尚未配置部署包。"
	case commonlogic.PluginApp:
		return "应用插件", "安装到所选传统应用环境，代码与安装任务作用于目标应用。"
	case commonlogic.TraditionApp:
		return "传统应用", "安装到传统应用运行环境，按配置部署代码并提供站点访问。"
	case commonlogic.SystemImageApp:
		return "系统镜像", "提供系统容器运行环境，运行与持久化方式以声明的配置为准。"
	case commonlogic.MetricsPluginApp:
		return "监控插件", "部署监控相关资源，具体能力以当前配置和部署包为准。"
	case "":
		return "尚未设置", "请先配置应用类型与运行信息。"
	default:
		return documentHomepageText(manifest.Application.Type), "具体行为以当前配置为准。"
	}
}

func writeDocumentTable(output *strings.Builder, title string, headers []string, rows [][]string) {
	if len(rows) == 0 {
		return
	}
	fmt.Fprintf(output, "### %s\n\n", title)
	writeRow := func(cells []string) {
		output.WriteString("| ")
		for _, cell := range cells {
			output.WriteString(documentHomepageText(cell))
			output.WriteString(" | ")
		}
		output.WriteString("\n")
	}
	writeRow(headers)
	separators := make([]string, len(headers))
	for index := range separators {
		separators[index] = "---"
	}
	writeRow(separators)
	for _, row := range rows {
		writeRow(row)
	}
	output.WriteString("\n")
}

func documentHomepageText(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	return strings.NewReplacer("\\", "\\\\", "&", "&amp;", "<", "&lt;", ">", "&gt;", "|", "\\|", "[", "\\[", "]", "\\]", "*", "\\*", "_", "\\_", "`", "\\`", "#", "\\#").Replace(value)
}
