package logic

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/errdef"
)

var priorityPolicy = Policy{Plugins: map[string]int{
	"plugin-low":  100,
	"plugin-high": 200,
}}

func TestPluginPriorityAndUninstallFallback(t *testing.T) {
	service, root, base := newTestInstaller(t)
	writeTestFile(t, base, "system.php", "base")
	writeTestFile(t, base, "base-only.php", "base-only")
	writeTestFile(t, root, "system.php", "base")
	writeTestFile(t, root, "base-only.php", "base-only")

	assertTestFile(t, root, "system.php", "base")

	low := t.TempDir()
	writeTestFile(t, low, "system.php", "low")
	writeTestFile(t, low, "low-only.php", "low-only")
	if _, err := service.InstallPlugin(root, low, "plugin-low", priorityPolicy); err != nil {
		t.Fatal(err)
	}
	assertTestFile(t, root, "system.php", "low")
	assertBaseFiles(t, service, []string{"system.php"})

	high := t.TempDir()
	writeTestFile(t, high, "system.php", "high")
	if _, err := service.InstallPlugin(root, high, "plugin-high", priorityPolicy); err != nil {
		t.Fatal(err)
	}
	assertTestFile(t, root, "system.php", "high")
	assertCurrentOCI(t, service, 3, 2)

	if _, err := service.UninstallPlugin(root, "plugin-high", priorityPolicy); err != nil {
		t.Fatal(err)
	}
	assertTestFile(t, root, "system.php", "low")
	assertCurrentOCI(t, service, 2, 1)

	if _, err := service.UninstallPlugin(root, "plugin-low", priorityPolicy); err != nil {
		t.Fatal(err)
	}
	assertTestFile(t, root, "system.php", "base")
	assertTestFile(t, root, "base-only.php", "base-only")
	assertTestMissing(t, root, "low-only.php")
	assertCurrentOCI(t, service, 1, 0)
}

func TestApplicationUpdateReappliesManagedPlugin(t *testing.T) {
	policy := Policy{Plugins: map[string]int{"plugin-a": 100}}
	service, root, base := newTestInstaller(t)
	writeTestFile(t, base, "system.php", "base-v1")
	writeTestFile(t, base, "unrelated.php", "unrelated-v1")
	writeTestFile(t, root, "system.php", "base-v1")
	writeTestFile(t, root, "unrelated.php", "unrelated-v1")
	if _, err := service.UpdateApplication(root, base, policy); err != nil {
		t.Fatal(err)
	}

	plugin := t.TempDir()
	writeTestFile(t, plugin, "system.php", "plugin")
	if _, err := service.InstallPlugin(root, plugin, "plugin-a", policy); err != nil {
		t.Fatal(err)
	}

	baseV2 := t.TempDir()
	writeTestFile(t, baseV2, "system.php", "base-v2")
	writeTestFile(t, baseV2, "unrelated.php", "unrelated-v2")
	writeTestFile(t, root, "system.php", "base-v2")
	writeTestFile(t, root, "unrelated.php", "unrelated-v2")
	if _, err := service.UpdateApplication(root, baseV2, policy); err != nil {
		t.Fatal(err)
	}
	assertTestFile(t, root, "system.php", "plugin")
	assertTestFile(t, root, "unrelated.php", "unrelated-v2")
	assertBaseFiles(t, service, []string{"system.php"})

	if _, err := service.UninstallPlugin(root, "plugin-a", policy); err != nil {
		t.Fatal(err)
	}
	assertTestFile(t, root, "system.php", "base-v2")
}

func TestUnmanagedPluginIsIgnored(t *testing.T) {
	policy := Policy{Plugins: map[string]int{}}
	service, root, base := newTestInstaller(t)
	writeTestFile(t, base, "system.php", "base")
	writeTestFile(t, root, "system.php", "base")
	plugin := t.TempDir()
	writeTestFile(t, plugin, "system.php", "plugin")
	result, err := service.InstallPlugin(root, plugin, "unmanaged", policy)
	if err != nil {
		t.Fatal(err)
	}
	if result.Managed {
		t.Fatal("unconfigured plugin must not be managed")
	}
	assertTestFile(t, root, "system.php", "base")
	store, err := service.store()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Resolve(context.Background(), currentTag); !errors.Is(err, errdef.ErrNotFound) {
		t.Fatalf("unmanaged plugin must not update OCI, got %v", err)
	}
}

func TestPluginIdentifierUsesExactValue(t *testing.T) {
	policy := Policy{Plugins: map[string]int{"Plugin_Name": 100}}
	service, root, _ := newTestInstaller(t)
	writeTestFile(t, root, "system.php", "base")
	plugin := t.TempDir()
	writeTestFile(t, plugin, "system.php", "plugin")

	result, err := service.InstallPlugin(root, plugin, "plugin-name", policy)
	if err != nil {
		t.Fatal(err)
	}
	if result.Managed {
		t.Fatal("plugin identifier must not be normalized")
	}
	assertTestFile(t, root, "system.php", "base")

	result, err = service.InstallPlugin(root, plugin, "Plugin_Name", policy)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Managed {
		t.Fatal("exact plugin identifier should be managed")
	}
	assertTestFile(t, root, "system.php", "plugin")
}

func TestRestoreReappliesCurrentOCI(t *testing.T) {
	policy := Policy{Plugins: map[string]int{"plugin-a": 100}}
	service, root, base := newTestInstaller(t)
	writeTestFile(t, base, "system.php", "base")
	writeTestFile(t, root, "system.php", "base")
	if _, err := service.UpdateApplication(root, base, policy); err != nil {
		t.Fatal(err)
	}
	plugin := t.TempDir()
	writeTestFile(t, plugin, "system.php", "plugin")
	if _, err := service.InstallPlugin(root, plugin, "plugin-a", policy); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "system.php", "broken")

	result, err := service.Restore(root)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Recovered {
		t.Fatal("expected restore to run")
	}
	assertTestFile(t, root, "system.php", "plugin")
}

func TestPriorityOnlyChangesManifestOrder(t *testing.T) {
	service, root, base := newTestInstaller(t)
	writeTestFile(t, base, "system.php", "base")
	writeTestFile(t, root, "system.php", "base")

	low := t.TempDir()
	writeTestFile(t, low, "system.php", "low")
	if _, err := service.InstallPlugin(root, low, "plugin-low", priorityPolicy); err != nil {
		t.Fatal(err)
	}
	high := t.TempDir()
	writeTestFile(t, high, "system.php", "high")
	if _, err := service.InstallPlugin(root, high, "plugin-high", priorityPolicy); err != nil {
		t.Fatal(err)
	}

	before := currentLayers(t, service)
	assertLayerOrder(t, before, "base", "plugin-low", "plugin-high")
	reversed := Policy{Plugins: map[string]int{
		"plugin-low":  200,
		"plugin-high": 100,
	}}
	writeTestFile(t, root, "system.php", "base")
	if _, err := service.UpdateApplication(root, base, reversed); err != nil {
		t.Fatal(err)
	}
	after := currentLayers(t, service)
	assertLayerOrder(t, after, "base", "plugin-high", "plugin-low")
	assertTestFile(t, root, "system.php", "low")

	beforeDigest := layerDigests(before)
	afterDigest := layerDigests(after)
	for _, plugin := range []string{"plugin-low", "plugin-high"} {
		if beforeDigest[plugin] != afterDigest[plugin] {
			t.Fatalf("plugin layer %s was rebuilt", plugin)
		}
	}

	writeTestFile(t, root, "system.php", "broken")
	if _, err := service.Restore(root); err != nil {
		t.Fatal(err)
	}
	assertTestFile(t, root, "system.php", "low")
}

func currentLayers(t *testing.T, service *Installer) []v1.Descriptor {
	t.Helper()
	store, err := service.store()
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := store.Resolve(context.Background(), currentTag)
	if err != nil {
		t.Fatal(err)
	}
	manifestContent, err := content.FetchAll(context.Background(), store, descriptor)
	if err != nil {
		t.Fatal(err)
	}
	var manifest v1.Manifest
	if err := json.Unmarshal(manifestContent, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest.Layers
}

func assertLayerOrder(t *testing.T, layers []v1.Descriptor, expected ...string) {
	t.Helper()
	if len(layers) != len(expected) {
		t.Fatalf("got %d layers, want %d", len(layers), len(expected))
	}
	for index, name := range expected {
		if layers[index].Annotations[v1.AnnotationTitle] != name {
			t.Fatalf("layer %d is %q, want %q", index, layers[index].Annotations[v1.AnnotationTitle], name)
		}
		if _, exists := layers[index].Annotations["io.w7.tradition-plugin.priority"]; exists {
			t.Fatalf("layer %q must not store runtime priority", name)
		}
	}
}

func layerDigests(layers []v1.Descriptor) map[string]string {
	result := make(map[string]string, len(layers))
	for _, layer := range layers {
		result[layer.Annotations[v1.AnnotationTitle]] = layer.Digest.String()
	}
	return result
}

func assertCurrentOCI(t *testing.T, service *Installer, layerCount, pluginCount int) {
	t.Helper()
	store, err := service.store()
	if err != nil {
		t.Fatal(err)
	}
	var tags []string
	if err := store.Tags(context.Background(), "", func(page []string) error {
		tags = append(tags, page...)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || tags[0] != currentTag {
		t.Fatalf("got OCI tags %v, want [current]", tags)
	}
	indexContent, err := os.ReadFile(filepath.Join(service.dataDir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index v1.Index
	if err := json.Unmarshal(indexContent, &index); err != nil {
		t.Fatal(err)
	}
	if len(index.Manifests) != 1 {
		t.Fatalf("got %d OCI manifests, want 1", len(index.Manifests))
	}
	descriptor, err := store.Resolve(context.Background(), currentTag)
	if err != nil {
		t.Fatal(err)
	}
	manifestContent, err := content.FetchAll(context.Background(), store, descriptor)
	if err != nil {
		t.Fatal(err)
	}
	var manifest v1.Manifest
	if err := json.Unmarshal(manifestContent, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Layers) != layerCount {
		t.Fatalf("got %d OCI layers, want %d", len(manifest.Layers), layerCount)
	}
	configContent, err := content.FetchAll(context.Background(), store, manifest.Config)
	if err != nil {
		t.Fatal(err)
	}
	var state State
	if err := json.Unmarshal(configContent, &state); err != nil {
		t.Fatal(err)
	}
	if len(state.PluginFiles) != pluginCount {
		t.Fatalf("got %d plugins, want %d", len(state.PluginFiles), pluginCount)
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(configContent, &config); err != nil {
		t.Fatal(err)
	}
	if len(config) != 1 || config["pluginFiles"] == nil {
		t.Fatalf("config contains redundant fields: %v", config)
	}
	blobs, err := filepath.Glob(filepath.Join(service.dataDir, "blobs", "sha256", "*"))
	if err != nil {
		t.Fatal(err)
	}
	if expected := layerCount + 2; len(blobs) != expected {
		t.Fatalf("got %d OCI blobs, want %d current blobs", len(blobs), expected)
	}
	for _, name := range []string{"policy.json", "active.json", "materialized.json"} {
		if _, err := os.Stat(filepath.Join(service.dataDir, name)); !os.IsNotExist(err) {
			t.Fatalf("redundant state file %s exists", name)
		}
	}
}

func assertBaseFiles(t *testing.T, service *Installer, expected []string) {
	t.Helper()
	state, err := service.loadState()
	if err != nil {
		t.Fatal(err)
	}
	if state.Base == nil {
		t.Fatal("expected base layer")
	}
	baseDir, err := service.baseTree(state.Base)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(baseDir)
	files, err := treeFiles(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	if stringList(files) != stringList(expected) {
		t.Fatalf("got base files %v, want %v", files, expected)
	}
}

func stringList(value []string) string {
	content, _ := json.Marshal(value)
	return string(content)
}

func newTestInstaller(t *testing.T) (*Installer, string, string) {
	t.Helper()
	data := filepath.Join(t.TempDir(), "data")
	root := filepath.Join(t.TempDir(), "root")
	base := filepath.Join(t.TempDir(), "base")
	for _, name := range []string{root, base} {
		if err := os.MkdirAll(name, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return NewInstaller(data), root, base
}

func writeTestFile(t *testing.T, root, name, value string) {
	t.Helper()
	file := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertTestFile(t *testing.T, root, name, expected string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != expected {
		t.Fatalf("%s: got %q, want %q", name, content, expected)
	}
}

func assertTestMissing(t *testing.T, root, name string) {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(name)))
	if !os.IsNotExist(err) {
		t.Fatalf("%s should not exist, got %v", name, err)
	}
}
