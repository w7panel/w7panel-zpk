package formula

import (
	"testing"

	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
)

func TestHasExternalDependencies(t *testing.T) {
	tests := []struct {
		name     string
		depends  []commonlogic.Depend
		expected bool
	}{
		{name: "no dependencies", expected: false},
		{name: "internal dependency", depends: []commonlogic.Depend{{Type: "in"}}, expected: false},
		{name: "external dependency", depends: []commonlogic.Depend{{Type: "out"}}, expected: true},
		{name: "mixed dependencies", depends: []commonlogic.Depend{{Type: "in"}, {Type: "out"}}, expected: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := commonlogic.Manifest{Platform: commonlogic.Platform{Depends: test.depends}}
			if actual := HasExternalDependencies(manifest); actual != test.expected {
				t.Fatalf("HasExternalDependencies() = %v, want %v", actual, test.expected)
			}
		})
	}
}
