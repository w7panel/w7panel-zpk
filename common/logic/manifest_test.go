package logic

import "testing"

func TestProcessManifestIdentifyKeepsTemplatedBackendURL(t *testing.T) {
	manifest := Manifest{
		Bindings: []Bindings{
			{
				BackendConfig: BackendConfig{
					BackendIdentifie: "https://{{.Values.DOMAIN_URL}}/tesfrt",
				},
			},
			{
				BackendConfig: BackendConfig{
					BackendIdentifie: "internal_service",
				},
			},
		},
	}

	got := ProcessManifestIdentify(manifest)

	if got.Bindings[0].BackendConfig.BackendIdentifie != "https://{{.Values.DOMAIN_URL}}/tesfrt" {
		t.Fatalf("templated backend url was changed: %q", got.Bindings[0].BackendConfig.BackendIdentifie)
	}
	if got.Bindings[1].BackendConfig.BackendIdentifie != "internal-service" {
		t.Fatalf("internal backend identifier was not normalized: %q", got.Bindings[1].BackendConfig.BackendIdentifie)
	}
}
