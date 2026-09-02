package logic

import (
	"strings"

	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
)

// isExternalDependency identifies a dependency installed as a separate
// application release. Its existing type-based semantics are intentionally
// preserved for type "out" dependencies.
func isExternalDependency(dependency commonlogic.Depend) bool {
	return dependency.Type == "out"
}

// isEmbeddedHelmDependency identifies a non-external dependency whose Helm
// chart must be downloaded and placed in the parent chart's charts directory.
func isEmbeddedHelmDependency(dependency commonlogic.Depend) bool {
	return dependency.Type != "out" && strings.TrimSpace(dependency.From) != ""
}
