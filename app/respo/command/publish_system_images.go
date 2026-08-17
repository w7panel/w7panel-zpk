package command

// Import the distributions published by oowy/systemd as first-class system
// image artifacts.  The command is intentionally self-contained so it can be
// run on a fresh depot without preparing manifests by hand.

import (
	"embed"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/console"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

type PublishSystemImages struct{ console.Abstract }

func (PublishSystemImages) GetName() string { return "respo:publish-system-images" }
func (PublishSystemImages) GetDescription() string {
	return "publish each oowy/systemd Docker tag as an independent system-image artifact"
}
func (PublishSystemImages) Configure(cmd *cobra.Command) {
	cmd.Flags().Int("user-id", 0, "console UID used when publishing artifacts")
	cmd.Flags().String("version", "1.0.0", "artifact version")
	cmd.Flags().StringSlice("tags", nil, "only publish these tags (default: all built-in tags)")
	cmd.Flags().Bool("dry-run", false, "list generated artifacts without writing to the depot")
}

func (c PublishSystemImages) Handle(cmd *cobra.Command, _ []string) {
	version, _ := cmd.Flags().GetString("version")
	consoleUID, _ := cmd.Flags().GetInt("user-id")
	requested, _ := cmd.Flags().GetStringSlice("tags")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	systems := append([]systemImageSpec(nil), supportedSystems...)
	if len(requested) > 0 {
		allowed := map[string]bool{}
		for _, tag := range requested {
			allowed[tag] = true
		}
		filtered := systems[:0]
		for _, system := range systems {
			keep := false
			for _, tag := range system.Tags {
				if allowed[tag] {
					keep = true
				}
			}
			if keep {
				filtered = append(filtered, system)
			}
		}
		systems = filtered
	}
	if len(systems) == 0 {
		panic("no publishable systems found")
	}
	sort.Slice(systems, func(i, j int) bool { return systems[i].Key < systems[j].Key })
	if dryRun {
		for _, system := range systems {
			fmt.Fprintf(cmd.OutOrStdout(), "%s -> %s image_version=%s\n", system.Key, artifactName(system.Key), strings.Join(system.Versions, ","))
		}
		return
	}

	username := facade.GetConfig().GetString("registry_cli.default.username")
	user, err := dao.Q.RegistryUser.Where(dao.Q.RegistryUser.Username.Eq(username)).First()
	if err != nil || user == nil {
		panic(fmt.Errorf("default registry user %q not found", username))
	}
	depot, err := logic.NewDepot()
	if err != nil {
		panic(err)
	}
	for _, system := range systems {
		name := artifactName(system.Key)
		if err := depot.AddFormula(name, version, user); err != nil {
			panic(fmt.Errorf("create %s: %w", name, err))
		}
		row, err := dao.Q.Formula.Where(dao.Q.Formula.Name.Eq(name)).First()
		if err != nil {
			panic(err)
		}
		ver, err := dao.Q.Version.Where(dao.Q.Version.FormulaID.Eq(row.ID)).Where(dao.Q.Version.Name.Eq(version)).First()
		if err != nil {
			panic(err)
		}
		manifest := systemImageManifest(system, name, version)
		formula := &logic.Formula{Name: name, VersionId: ver.ID}
		content, _ := yaml.Marshal(manifest)
		if err := depot.SaveManifestFile(formula, "manifest.yaml", string(content)); err != nil {
			panic(err)
		}
		icon, iconSource, err := normalizedSystemIcon(system.Key)
		if err != nil {
			panic(fmt.Errorf("icon for %s: %w", system.Key, err))
		}
		if err := logic.GetLocalClient().UploadByContent(formula.GetIconRelativePath(), string(icon)); err != nil {
			panic(err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "icon %s <- %s\n", name, iconSource)
		row.Title = manifest.Application.Name
		if _, err := dao.Q.Formula.Where(dao.Q.Formula.ID.Eq(row.ID)).Updates(entity.Formula{Title: row.Title}); err != nil {
			panic(err)
		}
		published, err := depot.GetFormula(name, version, nil)
		if err != nil {
			panic(err)
		}
		if err := (logic.Version{}).PublishFormula(int32(consoleUID), published); err != nil {
			panic(fmt.Errorf("publish %s: %w", name, err))
		}
		fmt.Fprintf(cmd.OutOrStdout(), "published %s (%s)\n", name, strings.Join(system.Versions, ","))
	}
}

// Keep this list local and deterministic. It mirrors the maintained tags in
// oowy/systemd; update it deliberately when a new base distribution is added.
type systemImageSpec struct {
	Key, ImagePrefix string
	Versions, Tags   []string
}

var supportedSystems = []systemImageSpec{
	{Key: "almalinux", ImagePrefix: "almalinux", Versions: []string{"10", "10-minimal", "9", "9-minimal", "8", "8-minimal"}},
	{Key: "amazonlinux", ImagePrefix: "amazonlinux", Versions: []string{"2023", "2"}},
	{Key: "centos-stream", ImagePrefix: "centos-stream", Versions: []string{"10", "10-minimal", "9", "9-minimal", "8", "8-minimal"}},
	{Key: "debian", ImagePrefix: "debian", Versions: []string{"trixie", "trixie-slim", "bookworm", "bookworm-slim", "bullseye", "bullseye-slim", "buster", "buster-slim"}},
	{Key: "oraclelinux", ImagePrefix: "oraclelinux", Versions: []string{"10", "10-slim", "9", "9-slim", "8", "8-slim"}},
	{Key: "redhat", ImagePrefix: "redhat-ubi", Versions: []string{"10", "10-minimal", "9", "9-minimal", "8", "8-minimal"}},
	{Key: "rockylinux", ImagePrefix: "rockylinux", Versions: []string{"10", "10-minimal", "9", "9-minimal", "8", "8-minimal"}},
	{Key: "ubuntu", ImagePrefix: "ubuntu", Versions: []string{"26.04", "resolute", "24.04", "noble", "22.04", "jammy", "20.04", "focal"}},
}

func init() {
	for i := range supportedSystems {
		for _, version := range supportedSystems[i].Versions {
			supportedSystems[i].Tags = append(supportedSystems[i].Tags, supportedSystems[i].ImagePrefix+"-"+version)
		}
	}
}

var unsafeTag = regexp.MustCompile(`[^a-z0-9.-]+`)

func artifactName(tag string) string {
	// Formula identifiers are constrained to a single separator: prefix-suffix
	// where suffix contains only lowercase letters and digits. Keep the full
	// Docker tag semantics while removing its punctuation for uniqueness.
	tag = strings.ToLower(tag)
	tag = unsafeTag.ReplaceAllString(tag, "")
	tag = strings.NewReplacer("-", "", ".", "").Replace(tag)
	return "oowy-" + tag
}

func systemImageManifest(system systemImageSpec, name, version string) commonlogic.Manifest {
	hostUsers := false
	rootfs := fmt.Sprintf(`[{"name":"%s","volumeName":"system-rootfs","path":"%s/system","persistentSpecialMounts":true}]`, name, name)
	return commonlogic.Manifest{Application: commonlogic.Application{Name: "Oowy systemd " + system.Key, Identifie: name, Description: "基于 oowy/systemd 的 " + system.Key + " 系统镜像", Type: commonlogic.SystemImageApp, Version: version, ClusterPrivileged: true, Annotation: map[string]interface{}{"w7.cc/system_image_category": "operating-system", "w7.cc/image_version": strings.Join(system.Versions, ","), "sysbox/rootfs-rw-layer": rootfs}}, Platform: commonlogic.Platform{BaseInfo: commonlogic.BaseInfo{Name: "Oowy systemd " + system.Key, Identifie: name}, Workload: commonlogic.Workload{Type: commonlogic.K8sWorkloadTypeDeployment}, RuntimeClassName: "sysbox-runc", HostUsers: &hostUsers, ContainerV2s: []commonlogic.ContainerV2{{Name: name, Image: "oowy/systemd:" + system.ImagePrefix + "-{version}", ImagePullPolicy: v1.PullIfNotPresent, Command: []string{"/sbin/init"}, VolumeMounts: []v1.VolumeMount{{Name: "system-rootfs", MountPath: "/system-rootfs"}}}}, Volumes: []v1.Volume{{Name: "system-rootfs", VolumeSource: v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: ""}}}}, Depends: []commonlogic.Depend{{Identifie: "w7panel-sysbox", Name: "微擎sysbox", Required: true, Type: "out", From: "https://zpk.w7.cc"}}, StartParams: []commonlogic.StartParams{{Mark: "system-image", Name: "IMAGE_VERSION", Title: "镜像版本", Description: "选择要安装的镜像版本", Type: "select", Required: true, ValuesText: strings.Join(system.Versions, "|")}, {Mark: "storage", Name: "global.cluster.storageRWmode", Title: "读写模式", Required: true, Type: "text", ValuesText: "%STORAGE_RW_MODE%"}, {Mark: "storage", Name: "global.cluster.storageSize", Title: "存储大小", Required: true, Type: "text", ValuesText: "%STORAGE_SIZE%"}, {Mark: "storage", Name: "global.cluster.storageClassName", Title: "存储类", Required: true, Type: "text", ValuesText: "%STORAGE_CLASS_NAME%"}}}}
}

var iconSlugs = map[string]string{"ubuntu": "ubuntu", "debian": "debian", "centos": "centos", "rocky": "rockylinux", "almalinux": "almalinux", "amazonlinux": "amazonwebservices", "redhat": "redhat"}

// systemIconFiles are bundled with the command so publishing does not depend
// on a CDN being reachable at runtime. Keep these files under source control
// and update them deliberately when the icon set changes.
//
//go:embed system_icons/*.png
var systemIconFiles embed.FS

func normalizedSystemIcon(tag string) ([]byte, string, error) {
	base := strings.ToLower(strings.Split(tag, ".")[0])
	keys := []string{"linux"}
	for key, value := range iconSlugs {
		if strings.Contains(base, key) {
			keys = []string{value, "linux"}
			break
		}
	}
	if strings.Contains(base, "oraclelinux") {
		keys = []string{"oraclelinux", "oracle", "linux"}
	}
	for _, key := range keys {
		path := "system_icons/" + key + ".png"
		data, err := systemIconFiles.ReadFile(path)
		if err != nil {
			continue
		}
		return data, "bundled:" + path, nil
	}
	// Keep the fallback in the same format as bundled assets: no conversion or
	// external image tooling is needed during publishing.
	data, _ := systemIconFiles.ReadFile("system_icons/linux.png")
	return data, "built-in-placeholder", nil
}
