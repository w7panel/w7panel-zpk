package logic

const (
	siteManagerPersistentVolumeClaim = "w7-sitemanager-site-manager"
	siteManagerStorageVolumeName     = "site-storage"
)

func siteManagerPodAffinityValues() map[string]interface{} {
	return map[string]interface{}{"podAffinity": map[string]interface{}{
		"requiredDuringSchedulingIgnoredDuringExecution": []interface{}{map[string]interface{}{
			"labelSelector": map[string]interface{}{"matchLabels": map[string]string{
				"app.kubernetes.io/instance": "w7-sitemanager",
				"app.kubernetes.io/name":     "site-manager",
				"w7.cc/identifie":            "w7-sitemanager",
			}},
			"topologyKey": "kubernetes.io/hostname",
		}},
	}}
}
