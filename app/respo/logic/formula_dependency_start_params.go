package logic

import (
	"strings"

	"github.com/w7panel/w7panel-zpk/common/logic"
)

const dependencyReleaseNameSuffix = "_RELEASE_NAME"

func DependencyReleaseStartParamName(dependency logic.Depend) string {
	identify := strings.TrimSpace(dependency.SubIdentifie)
	if identify == "" {
		identify = strings.TrimSpace(dependency.Identifie)
	}
	identify = strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(identify))
	if identify == "" {
		return ""
	}
	return identify + dependencyReleaseNameSuffix
}

// ApplyDependencyReleaseStartParams derives internal release-name parameters
// from external dependencies. These parameters are runtime installation data,
// not author-maintained manifest configuration.
func ApplyDependencyReleaseStartParams(manifest *logic.Manifest) {
	if manifest == nil {
		return
	}

	generatedNames := make(map[string]struct{})
	generated := make([]logic.StartParams, 0, len(manifest.Platform.Depends))
	for _, dependency := range manifest.Platform.Depends {
		if !isExternalDependency(dependency) {
			continue
		}
		name := DependencyReleaseStartParamName(dependency)
		if name == "" {
			continue
		}
		if _, exists := generatedNames[name]; exists {
			continue
		}
		generatedNames[name] = struct{}{}
		moduleName := strings.TrimSpace(dependency.SubIdentifie)
		if moduleName == "" {
			moduleName = strings.TrimSpace(dependency.Identifie)
		}
		generated = append(generated, logic.StartParams{
			Name:       name,
			Title:      dependency.Name + " ReleaseName",
			Required:   dependency.Required,
			Type:       "text",
			ValuesText: strings.TrimSpace(dependency.ReleaseName),
			ModuleName: moduleName,
			Hidden:     true,
		})
	}

	startParams := make([]logic.StartParams, 0, len(manifest.Platform.StartParams)+len(generated))
	for _, param := range manifest.Platform.StartParams {
		if _, generated := generatedNames[param.Name]; generated {
			continue
		}
		startParams = append(startParams, param)
	}
	manifest.Platform.StartParams = append(startParams, generated...)
}
