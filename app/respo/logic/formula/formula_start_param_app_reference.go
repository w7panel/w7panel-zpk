package formula

import (
	"strconv"
	"strings"

	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
)

// ResolveStartParamAppReferencesCopy returns resolved start parameters without
// changing the persisted manifest representation used by editing APIs.
func ResolveStartParamAppReferencesCopy(
	params []commonlogic.StartParams,
	manifests []*commonlogic.Manifest,
) []commonlogic.StartParams {
	resolved := append([]commonlogic.StartParams(nil), params...)
	resolveStartParamModuleNames(resolved, startParamAppIdentifies(manifests))
	return resolved
}

func startParamAppIdentifies(manifests []*commonlogic.Manifest) []string {
	identifies := make([]string, len(manifests))
	for index, manifest := range manifests {
		if manifest != nil {
			identifies[index] = manifest.Application.Identifie
		}
	}
	return identifies
}

func resolveStartParamModuleNames(params []commonlogic.StartParams, identifies []string) {
	for index := range params {
		appIndex, suffix, ok := parseStartParamAppReference(params[index].ModuleName)
		if !ok || appIndex >= len(identifies) || identifies[appIndex] == "" {
			continue
		}
		params[index].ModuleName = identifies[appIndex] + suffix
	}
}

func parseStartParamAppReference(moduleName string) (int, string, bool) {
	if !strings.HasPrefix(moduleName, "app[") {
		return 0, "", false
	}
	closingBracket := strings.IndexByte(moduleName, ']')
	if closingBracket < len("app[0") {
		return 0, "", false
	}
	suffix := moduleName[closingBracket+1:]
	if suffix != "" && !strings.HasPrefix(suffix, ".") {
		return 0, "", false
	}
	index, err := strconv.Atoi(moduleName[len("app["):closingBracket])
	if err != nil || index < 0 {
		return 0, "", false
	}
	return index, suffix, true
}
