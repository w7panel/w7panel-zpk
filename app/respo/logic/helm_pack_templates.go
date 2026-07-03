package logic

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
)

//go:embed helm_templates/*.tpl
var helmTemplateFS embed.FS

func loadHelmTemplate(name string) (string, error) {
	content, err := helmTemplateFS.ReadFile(filepath.Join("helm_templates", name))
	if err != nil {
		return "", fmt.Errorf("读取 Helm 模板失败 %s: %w", name, err)
	}
	return string(content), nil
}

func writeHelmTemplateFile(rootDir, outputName, templateName string) error {
	content, err := loadHelmTemplate(templateName)
	if err != nil {
		return err
	}
	return writeFile(filepath.Join(rootDir, outputName), content)
}

func renderHelmTemplatePlaceholders(content string, values map[string]string) string {
	replacements := make([]string, 0, len(values)*2)
	for placeholder, value := range values {
		replacements = append(replacements, placeholder, value)
	}
	return strings.NewReplacer(replacements...).Replace(content)
}
