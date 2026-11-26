package helm

import (
	"fmt"
	"strings"

	formulalogic "github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
)

// detectAffinityDependencies resolves existing PVC imports to actual applications.
// The input order is Formula.AllManifest order; app[n] must use that exact list.
func (hc *HelmPack) detectAffinityDependencies() (map[string][]string, error) {
	dependencies := make(map[string][]string)
	normalize := func(name string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "_", "-")
	}
	byName := make(map[string]string)
	claims := make(map[string]map[string]bool)
	for _, manifest := range hc.allManifest {
		if manifest != nil {
			byName[normalize(manifest.Application.Identifie)] = manifest.Application.Identifie
		}
	}
	add := func(app, target string) {
		for _, existing := range dependencies[app] {
			if existing == target {
				return
			}
		}
		dependencies[app] = append(dependencies[app], target)
	}
	for _, manifest := range hc.allManifest {
		if manifest == nil {
			continue
		}
		app := manifest.Application.Identifie
		fixedClaims, usedParams := mountedPVCReferences(*manifest)
		claims[app] = fixedClaims
		isHelm := (&HelmPack{Manifest: *manifest}).isHelmPackage()
		params := formulalogic.ResolveStartParamAppReferencesCopy(manifest.Platform.StartParams, hc.allManifest, app)
		for _, param := range params {
			target := byName[normalize(param.ModuleName)]
			if target != "" && target != app && strings.TrimSpace(param.ValuesText) == "%PVC_NAME%" && (isHelm || usedParams[param.Name]) {
				add(app, target)
			}
		}
	}
	// Reject explicit cycles before writing any Chart files.
	for _, manifest := range hc.allManifest {
		if manifest == nil {
			continue
		}
		app := manifest.Application.Identifie
		for _, target := range dependencies[app] {
			if hasAffinityPath(dependencies, target, app, make(map[string]bool)) {
				return nil, fmt.Errorf("应用 %s 与 %s 存在 PVC 亲和性循环依赖", app, target)
			}
		}
	}
	// Fixed shared claims are symmetric. Keep existing list order unless an
	// explicit path already requires the reverse direction.
	for index, manifest := range hc.allManifest {
		if manifest == nil {
			continue
		}
		app := manifest.Application.Identifie
		for _, previous := range hc.allManifest[:index] {
			if previous == nil || previous.Application.Identifie == app {
				continue
			}
			target := previous.Application.Identifie
			for claim := range claims[app] {
				if claims[target][claim] && !hasAffinityPath(dependencies, target, app, make(map[string]bool)) {
					add(app, target)
					break
				}
			}
		}
	}
	return dependencies, nil
}

func hasAffinityPath(dependencies map[string][]string, from, target string, visited map[string]bool) bool {
	if from == target {
		return true
	}
	if visited[from] {
		return false
	}
	visited[from] = true
	for _, next := range dependencies[from] {
		if hasAffinityPath(dependencies, next, target, visited) {
			return true
		}
	}
	return false
}

// Return fixed PVC names and values parameters used by mounted PVC volumes.
// Helm templates are opaque; declared volumes may be consumed inside the Chart.
func mountedPVCReferences(manifest logic2.Manifest) (map[string]bool, map[string]bool) {
	claims, params := make(map[string]bool), make(map[string]bool)
	mounted := make(map[string]bool)
	for _, container := range manifest.Platform.ContainerV2s {
		for _, mount := range container.VolumeMounts {
			mounted[mount.Name] = true
		}
	}
	isHelm := (&HelmPack{Manifest: manifest}).isHelmPackage()
	for _, volume := range manifest.Platform.Volumes {
		if volume.PersistentVolumeClaim == nil || (!isHelm && !mounted[volume.Name]) {
			continue
		}
		claim := strings.TrimSpace(volume.PersistentVolumeClaim.ClaimName)
		switch claim {
		case "", "%PVC_NAME%", "${PVC_NAME}":
			params["PVC_NAME"] = true
		default:
			if strings.HasPrefix(claim, "{{") && strings.HasSuffix(claim, "}}") {
				expression := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(claim, "{{"), "}}"))
				if strings.HasPrefix(expression, ".Values.") {
					name := strings.TrimPrefix(expression, ".Values.")
					if name != "" && !strings.ContainsAny(name, " |(){}\t\n") {
						params[name] = true
					}
				}
			} else if !strings.ContainsAny(claim, "{}%$") {
				claims[claim] = true
			}
		}
	}
	return claims, params
}
