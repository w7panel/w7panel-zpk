package logic

import (
	"strings"

	"github.com/w7panel/w7panel-zpk/common/logic"
)

const dependencyReleaseNameSuffix = "_RELEASE_NAME"

func DependencyReleaseStartParamName(dependency logic.Depend) string {
	identify := dependency.SubIdentifie
	if identify == "" {
		identify = dependency.Identifie
	}
	identify = strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(identify))
	if identify == "" {
		return ""
	}
	return identify + dependencyReleaseNameSuffix
}

// PopulateManifestStartParamsWithDependencyReleaseNames adds hidden
// dependency release-name parameters to the current manifest's startParams.
// These parameters are runtime installation data, not author-maintained
// dependency configuration.
func PopulateManifestStartParamsWithDependencyReleaseNames(manifest *logic.Manifest) {
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
		moduleName := dependency.SubIdentifie
		if moduleName == "" {
			moduleName = dependency.Identifie
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
