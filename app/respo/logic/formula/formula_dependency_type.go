package formula

import (
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
)

// isExternalDependency identifies a dependency installed as a separate
// application release. Its existing type-based semantics are intentionally
// preserved for type "out" dependencies.
func isExternalDependency(dependency commonlogic.Depend) bool {
	return dependency.Type == "out"
}

// HasExternalDependencies reports whether a manifest contains dependencies
// that are installed as separate application releases.
func HasExternalDependencies(manifest commonlogic.Manifest) bool {
	for _, dependency := range manifest.Platform.Depends {
		if isExternalDependency(dependency) {
			return true
		}
	}
	return false
}
