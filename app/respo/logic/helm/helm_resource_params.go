package helm

import (
	"strings"

	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	corev1 "k8s.io/api/core/v1"
)

// Resource declarations stay in startParams. This is only generated Chart data.
func installResourceConfig(platform commonlogic.Platform) map[string]interface{} {
	parameters := map[string]string{}
	weights := map[string]map[string]int64{}
	for _, param := range platform.StartParams {
		kind := strings.TrimPrefix(param.Type, "resource-")
		if kind == param.Type {
			continue
		}
		switch kind {
		case "cpu", "memory", "gpu", "bandwidth":
		default:
			continue
		}
		parameters[kind] = param.Name
	}
	for _, kind := range []string{"cpu", "memory", "gpu"} {
		weights[kind] = map[string]int64{}
		key := corev1.ResourceName(kind)
		if kind == "gpu" {
			key = "nvidia.com/gpu"
		}
		for _, container := range platform.ContainerV2s {
			if container.IsInitContainer {
				continue
			}
			q := container.Resources.Limits[key]
			value := q.Value()
			if kind == "cpu" {
				value = q.MilliValue()
			}
			if value > 0 {
				weights[kind][strings.ReplaceAll(container.Name, "_", "-")] = value
			}
		}
	}
	return map[string]interface{}{"parameters": parameters, "weights": weights}
}
