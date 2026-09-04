package logic

import (
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
)

// isExternalDependency identifies a dependency installed as a separate
// application release. Its existing type-based semantics are intentionally
// preserved for type "out" dependencies.
func isExternalDependency(dependency commonlogic.Depend) bool {
	return dependency.Type == "out"
}
