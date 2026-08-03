package logic

import (
	"regexp"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	Tradition_App    = "tradition"
	Docker_App       = "docker"
	Help_App         = "helm"
	GatewayPluginApp = "gateway-plugin"
	EnvironmentApp   = "environment"

	GatewayPluginDriverHigressWasmV1 = "higress-wasm/v1"
)

const (
	K8sWorkloadTypeDeployment  = "Deployment"
	K8sWorkloadTypeStatefulSet = "StatefulSet"
	K8sWorkloadTypeDaemonSet   = "DaemonSet"
)

type Manifest struct {
	Application Application `yaml:"application" json:"application"`
	Platform    Platform    `yaml:"platform" json:"platform"`
	Bindings    []Bindings  `yaml:"bindings" json:"bindings"`
	Source      Source      `yaml:"source" json:"source"`
	Web         Source      `yaml:"web" json:"web"`
	Version     int         `yaml:"v" json:"v"`
	VersionV2   int         `yaml:"version" json:"version"`
}

type GatewayPlugin struct {
	Category       string                 `yaml:"category,omitempty" json:"category,omitempty"`
	Supports       GatewayPluginSupports  `yaml:"supports" json:"supports"`
	DefaultEnabled *bool                  `yaml:"defaultEnabled,omitempty" json:"defaultEnabled,omitempty"`
	DefaultConfig  map[string]interface{} `yaml:"defaultConfig" json:"defaultConfig"`
	ConfigSchema   map[string]interface{} `yaml:"configSchema,omitempty" json:"configSchema,omitempty"`
	Runtime        GatewayPluginRuntime   `yaml:"runtime" json:"runtime"`
}

type GatewayPluginSupports struct {
	Global bool `yaml:"global" json:"global"`
	Rule   bool `yaml:"rule" json:"rule"`
}

type GatewayPluginRuntime struct {
	Driver string                     `yaml:"driver" json:"driver"`
	Config GatewayPluginRuntimeConfig `yaml:"config" json:"config"`
}

type GatewayPluginRuntimeConfig struct {
	URL      string `yaml:"url" json:"url"`
	Phase    string `yaml:"phase" json:"phase"`
	Priority int    `yaml:"priority" json:"priority"`
}

func (plugin GatewayPlugin) IsSupportGlobal() bool {
	return plugin.Supports.Global
}

func (plugin GatewayPlugin) IsEnabledByDefault() bool {
	return plugin.DefaultEnabled == nil || *plugin.DefaultEnabled
}

func (plugin GatewayPlugin) Normalize() GatewayPlugin {
	if plugin.DefaultConfig == nil {
		plugin.DefaultConfig = make(map[string]interface{})
	}
	return plugin
}

type Application struct {
	Name              string                 `yaml:"name" json:"name"`
	Identifie         string                 `yaml:"identifie" json:"identifie"`
	Description       string                 `yaml:"description" json:"description"`
	Author            string                 `yaml:"author" json:"author"`
	InstallOnlyOnce   bool                   `yaml:"once" json:"once"`
	ClusterPrivileged bool                   `yaml:"clusterPrivileges" json:"clusterPrivileges"`
	RegisterSite      bool                   `yaml:"registerSite" json:"registerSite"`
	Type              string                 `yaml:"type" json:"type"`
	Annotation        map[string]interface{} `yaml:"annotation" json:"annotation"`
	Version           string                 `yaml:"version" json:"version"`
}

type Platform struct {
	BaseInfo             BaseInfo                   `yaml:"baseInfo" json:"baseInfo"`
	GatewayPlugin        GatewayPlugin              `yaml:"gatewayPlugin" json:"gatewayPlugin"`
	Container            Container                  `yaml:"container" json:"container"`
	ContainerV2s         []ContainerV2              `yaml:"container-v2" json:"container-v2"`
	Volumes              []v1.Volume                `yaml:"volumes" json:"volumes"`
	VolumeClaimTemplates []v1.PersistentVolumeClaim `yaml:"volumeClaimTemplates" json:"volumeClaimTemplates"`
	Workload             Workload                   `yaml:"workload" json:"workload"`
	Helm                 Helm                       `yaml:"helm" json:"helm"`
	Tradition            Tradition                  `yaml:"tradition" json:"tradition"`
	Ingress              []Ingress                  `yaml:"ingress" json:"ingress"`
	Depends              []Depend                   `yaml:"depends" json:"depends"`
	StartParams          []StartParams              `yaml:"startParams" json:"startParams"`
	Gpu                  string                     `yaml:"runtimeClassName" json:"runtimeClassName"`
	Shells               []Shell                    `yaml:"shells" json:"shells"`
}

type Workload struct {
	Type           string      `yaml:"type" json:"type"`
	UpdateStrategy interface{} `yaml:"updateStrategy" json:"updateStrategy"`
}

type BaseInfo struct {
	Name        string `yaml:"name" json:"name"`
	Identifie   string `yaml:"identifie" json:"identifie"`
	Description string `yaml:"description" json:"description"`
}

type ContainerV2 struct {
	Name            string                  `yaml:"name" json:"name" protobuf:"bytes,1,opt,name=name"`
	Image           string                  `yaml:"image" json:"image,omitempty" protobuf:"bytes,2,opt,name=image"`
	CodeAttachUrl   string                  `yaml:"-" json:"-"`
	Command         []string                `yaml:"command" json:"command,omitempty" protobuf:"bytes,3,rep,name=command"`
	Args            []string                `yaml:"args" json:"args,omitempty" protobuf:"bytes,4,rep,name=args"`
	Ports           []v1.ContainerPort      `yaml:"ports" json:"ports,omitempty" patchStrategy:"merge" patchMergeKey:"containerPort" protobuf:"bytes,6,rep,name=ports"`
	Env             []v1.EnvVar             `yaml:"env" json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,7,rep,name=env"`
	Resources       v1.ResourceRequirements `yaml:"resources" json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`
	VolumeMounts    []v1.VolumeMount        `yaml:"volumeMounts" json:"volumeMounts,omitempty" patchStrategy:"merge" patchMergeKey:"mountPath" protobuf:"bytes,9,rep,name=volumeMounts"`
	LivenessProbe   *v1.Probe               `yaml:"livenessProbe" json:"livenessProbe,omitempty" protobuf:"bytes,10,opt,name=livenessProbe"`
	StartupProbe    *v1.Probe               `yaml:"startupProbe" json:"startupProbe,omitempty" protobuf:"bytes,22,opt,name=startupProbe"`
	ReadinessProb   *v1.Probe               `yaml:"readinessProbe" json:"readinessProbe,omitempty" protobuf:"bytes,10,opt,name=readinessProbe"`
	Lifecycle       *v1.Lifecycle           `yaml:"lifecycle" json:"lifecycle,omitempty" protobuf:"bytes,12,opt,name=lifecycle"`
	ImagePullPolicy v1.PullPolicy           `yaml:"imagePullPolicy" json:"imagePullPolicy,omitempty" protobuf:"bytes,14,opt,name=imagePullPolicy,casttype=PullPolicy"`
	SecurityContext *v1.SecurityContext     `yaml:"securityContext" json:"securityContext,omitempty" protobuf:"bytes,15,opt,name=securityContext"`
	IsInitContainer bool                    `yaml:"isInitContainer" json:"isInitContainer"`

	Shells []Shell `yaml:"shells" json:"shells"`
	Build  Build   `yaml:"build" json:"build"`
}

type Tradition struct {
	EnvironmentName     string   `yaml:"environmentName" json:"environmentName"`
	EnvironmentVersion  string   `yaml:"environmentVersion" json:"environmentVersion"`
	EnvironmentLanguage string   `yaml:"environmentLanguage" json:"environmentLanguage"`
	CodeAttachUrl       string   `yaml:"-" json:"-"`
	Cmd                 []string `yaml:"cmd" json:"cmd"`
}

type HelmDependYaml struct {
	Name string `yaml:"name" json:"name"`
	Yaml string `yaml:"yaml" json:"yaml"`
}

type Helm struct {
	ChartName   string           `yaml:"chartName" json:"chartName"`
	Repository  string           `yaml:"repository" json:"repository"`
	Version     string           `yaml:"version" json:"version"`
	Configs     []interface{}    `yaml:"kv" json:"kv"`
	DependYamls []HelmDependYaml `yaml:"depend_yamls" json:"depend_yamls"`
}

type Build struct {
	Context      string `yaml:"context" json:"context"`
	BuildContext string `yaml:"build_context" json:"build_context"`
}

type Shell struct {
	Shell     string `yaml:"shell" json:"shell"`
	Title     string `yaml:"title" json:"title"`
	Type      string `yaml:"type" json:"type"`
	Image     string `yaml:"image" json:"image"`
	Container string `yaml:"container" json:"container"`
}

type StartParams struct {
	Name        string `yaml:"name" json:"name"`
	Title       string `yaml:"title" json:"title"`
	Description string `yaml:"description" json:"description"`
	Mark        string `yaml:"mark" json:"mark"`
	Required    bool   `yaml:"required" json:"required"`
	Type        string `yaml:"type" json:"type"`
	ValuesText  string `yaml:"values_text" json:"values_text"`
	ModuleName  string `yaml:"module_name" json:"module_name"`
}

type Container struct {
	BaseInfo      BaseInfo      `yaml:"baseInfo" json:"baseInfo"`
	ContainerPort int           `yaml:"containerPort" json:"containerPort"`
	MinNum        int           `yaml:"minNum" json:"minNum"`
	MaxNum        int           `yaml:"maxNum" json:"maxNum"`
	Cpu           int           `yaml:"cpu" json:"cpu"`
	Mem           int           `yaml:"mem" json:"mem"`
	Gpu           string        `yaml:"runtimeClassName" json:"runtimeClassName"`
	Image         string        `yaml:"image" json:"image"`
	Volumes       []Volumes     `yaml:"volumes" json:"volumes"`
	StartParams   []StartParams `yaml:"startParams" json:"startParams"`
	Shells        []Shell       `yaml:"shells" json:"shells"`
	Build         Build         `yaml:"build" json:"build"`
	Env           []Env         `yaml:"env" json:"env"`
	Ports         []Port        `yaml:"ports" json:"ports"`
	Privileged    bool          `yaml:"privileged" json:"privileged"`
	Cmd           []string      `yaml:"cmd" json:"cmd"`
	Hook          struct {
		RequireInstall string `yaml:"requireInstall" json:"requireInstall"`
	} `yaml:"hook" json:"hook"`
	SecurityContext struct {
		FsGroup      int  `yaml:"fsGroup" json:"fsGroup"`
		RunAsGroup   int  `yaml:"runAsGroup" json:"runAsGroup"`
		RunAsNonRoot bool `yaml:"runAsNonRoot" json:"runAsNonRoot"`
		RunAsUser    int  `yaml:"runAsUser" json:"runAsUser"`
	} `yaml:"securityContext" json:"securityContext"`
}

type Volumes struct {
	Type      string `yaml:"type" json:"type"`
	MountPath string `yaml:"mountPath" json:"mountPath"`
	SubPath   string `yaml:"subPath" json:"subPath"`
	HostPath  struct {
		Path string `yaml:"path" json:"path"`
		Type string `yaml:"type" json:"type"`
	} `yaml:"hostPath" json:"hostPath"`
}

type EnvValueFromFieldRef struct {
	FieldPath  string `yaml:"fieldPath" json:"fieldPath"`
	ApiVersion string `yaml:"apiVersion" json:"apiVersion"`
}

type EnvValueFromResourceFieldRef struct {
	Divisor  int    `yaml:"divisor" json:"divisor"`
	Resource string `yaml:"resource" json:"resource"`
}

type EnvValueFrom struct {
	FieldRef         *EnvValueFromFieldRef         `yaml:"fieldRef" json:"fieldRef"`
	ResourceFieldRef *EnvValueFromResourceFieldRef `yaml:"resourceFieldRef" json:"resourceFieldRef"`
}

type Env struct {
	Name      string        `yaml:"name" json:"name"`
	Value     string        `yaml:"value" json:"value"`
	ValueFrom *EnvValueFrom `yaml:"valueFrom" json:"valueFrom"`
}

type Port struct {
	Name     string `yaml:"name" json:"name"`
	Port     int    `yaml:"port" json:"port"`
	Protocol string `yaml:"protocol" json:"protocol"`
	LbPort   string `yaml:"lbPort" json:"lbPort"`
}

type RequestProxy struct {
	Headers map[string]string `yaml:"headers" json:"headers"`
	Query   map[string]string `yaml:"query" json:"query"`
}

type BackendConfig struct {
	Type             string            `yaml:"type" json:"type"`
	BackendIdentifie string            `yaml:"backend_identifie" json:"backend_identifie"`
	BackendUrl       string            `yaml:"backend_url" json:"backend_url"`
	BackendPort      int               `yaml:"backend_port" json:"backend_port"`
	RequestProxy     RequestProxy      `yaml:"proxy_request" json:"proxy_request"`
	FrontendProps    map[string]string `yaml:"frontend_props" json:"frontend_props"`
}

type Bindings struct {
	Title         string        `yaml:"title" json:"title"`
	Name          string        `yaml:"name" json:"name"`
	Status        int           `yaml:"status" json:"status"`
	Support       string        `yaml:"support" json:"support"`
	Menu          []Menu        `yaml:"menu" json:"menu"`
	LoadMode      string        `yaml:"load_mode" json:"load_mode"`
	BackendConfig BackendConfig `yaml:"backend_config" json:"backend_config"`
}

type Menu struct {
	DisplayOrder int         `yaml:"displayorder" json:"displayorder"`
	Do           string      `yaml:"do" json:"do"`
	Title        string      `yaml:"title" json:"title"`
	Icon         string      `yaml:"icon" json:"icon"`
	IconSvg      interface{} `yaml:"icon_svg" json:"icon_svg"`
	Location     string      `yaml:"location" json:"location"`
	IsDefault    int         `yaml:"is_default" json:"is_default"`
	Parent       string      `yaml:"parent" json:"parent"`
}

type IngressBackend struct {
	Name      string `yaml:"name" json:"name"`
	Port      int    `yaml:"port" json:"port"`
	Match     string `yaml:"match" json:"match"`
	MoreMatch *struct {
		Header []struct {
			Key   string `yaml:"key" json:"key"`
			Value string `yaml:"value" json:"value"`
			Type  string `yaml:"type" json:"type"`
		} `yaml:"header" json:"header"`
		Method []string `yaml:"method" json:"method"`
		Query  []struct {
			Key   string `yaml:"key" json:"key"`
			Value string `yaml:"value" json:"value"`
			Type  string `yaml:"type" json:"type"`
		} `yaml:"query" json:"query"`
	} `yaml:"moreMatch" json:"moreMatch"`
	Rewrite *struct {
		Host string `yaml:"host" json:"host"`
		Path string `yaml:"path" json:"path"`
	} `yaml:"rewrite" json:"rewrite"`
	Strategy map[string]interface{} `yaml:"strategy" json:"strategy"`
}

type IngressRoute struct {
	Backend IngressBackend `yaml:"backend" json:"backend"`
	Path    string         `yaml:"path" json:"path"`
}

type Ingress struct {
	Name   string         `yaml:"name" json:"name"`
	Routes []IngressRoute `yaml:"routes" json:"routes"`
}

type Depend struct {
	Identifie    string `yaml:"identifie" json:"identifie"`
	Name         string `yaml:"name" json:"name"`
	From         string `yaml:"from" json:"from"`
	Required     bool   `yaml:"required" json:"required"`
	Type         string `yaml:"type" json:"type"`
	SubIdentifie string `yaml:"subidentifie" json:"subidentifie"`
	SubName      string `yaml:"subname" json:"subname"`
}

type Source struct {
	Url  string `yaml:"url" json:"url"`
	Type string `yaml:"type" json:"type"`
}

func ProcessManifestIdentify(manifestRow Manifest) Manifest {
	if manifestRow.Application.Type == GatewayPluginApp {
		manifestRow.Platform.GatewayPlugin = manifestRow.Platform.GatewayPlugin.Normalize()
	}
	if manifestRow.Platform.BaseInfo.Identifie == "" {
		manifestRow.Platform.BaseInfo = manifestRow.Platform.Container.BaseInfo
	}
	manifestRow.Platform.BaseInfo.Identifie = strings.ReplaceAll(manifestRow.Platform.BaseInfo.Identifie, "_", "-")
	manifestRow.Application.Identifie = strings.ReplaceAll(manifestRow.Application.Identifie, "_", "-")
	for index, val := range manifestRow.Platform.Depends {
		val.Identifie = strings.ReplaceAll(val.Identifie, "_", "-")
		val.SubIdentifie = strings.ReplaceAll(val.SubIdentifie, "_", "-")
		manifestRow.Platform.Depends[index] = val
	}
	for index, item := range manifestRow.Platform.Ingress {
		for iindex, iitem := range item.Routes {
			iitem.Backend.Name = strings.ReplaceAll(iitem.Backend.Name, "_", "-")
			item.Routes[iindex] = iitem
		}
		manifestRow.Platform.Ingress[index] = item
	}
	for index, item := range manifestRow.Bindings {
		existsDefaultMenu := false
		for mi, mitem := range item.Menu {
			if mitem.IsDefault > 0 {
				mitem.IsDefault -= 1
			}
			if mitem.IsDefault == 1 {
				existsDefaultMenu = true
			}
			item.Menu[mi] = mitem
		}
		if !existsDefaultMenu && len(item.Menu) > 0 {
			item.Menu[0].IsDefault = 1
		}
		if item.BackendConfig.BackendUrl == "" {
			item.BackendConfig.BackendUrl = item.BackendConfig.BackendIdentifie
		}
		item.BackendConfig.BackendUrl = normalizeBackendIdentifie(item.BackendConfig.BackendUrl)

		manifestRow.Bindings[index] = item
	}
	manifestRow.Platform.Tradition.EnvironmentName = strings.ReplaceAll(manifestRow.Platform.Tradition.EnvironmentName, "_", "-")

	return manifestRow
}

func normalizeBackendIdentifie(identifie string) string {
	if strings.Contains(identifie, "{{") || strings.Contains(identifie, "${") || strings.Contains(identifie, "://") {
		return identifie
	}
	return strings.ReplaceAll(identifie, "_", "-")
}

func GetManifestV2(manifest Manifest) Manifest {
	manifest = normalizeLegacyManifestPlaceholders(manifest)

	if len(manifest.Platform.StartParams) == 0 {
		manifest.Platform.StartParams = manifest.Platform.Container.StartParams
	}
	//兼容面板
	manifest.Platform.Container.StartParams = manifest.Platform.StartParams

	if manifest.Platform.Gpu == "" {
		manifest.Platform.Gpu = manifest.Platform.Container.Gpu
	}

	if manifest.Platform.Workload.Type == "" {
		manifest.Platform.Workload.Type = K8sWorkloadTypeDeployment
	}

	if manifest.Platform.Container.Image == "" && manifest.Source.Url == "" {
		return manifest
	}
	if manifest.Application.Type == "tradition" || manifest.Application.Type == "helm" {
		return manifest
	}

	if manifest.Platform.ContainerV2s == nil || len(manifest.Platform.ContainerV2s) == 0 {
		manifest.Platform.ContainerV2s = []ContainerV2{
			convertContainerToContainerV2(manifest.Platform.Container),
		}
		manifest.Platform.Volumes = convertToV1Volume(manifest.Platform.Container.Volumes)
		manifest.Platform.ContainerV2s[0].Shells = manifest.Platform.Container.Shells
		manifest.Platform.ContainerV2s[0].Build = manifest.Platform.Container.Build
	}

	for index, item := range manifest.Platform.ContainerV2s {
		if item.Name == "" {
			manifest.Platform.ContainerV2s[index].Name = manifest.Application.Identifie + strconv.Itoa(index)
		}
	}
	if len(manifest.Platform.ContainerV2s) > 0 {
		manifest.Platform.ContainerV2s[0].CodeAttachUrl = manifest.Source.Url
		if len(manifest.Platform.Shells) == 0 {
			manifest.Platform.Shells = manifest.Platform.ContainerV2s[0].Shells
		}
	}

	return manifest
}

var legacyHelmValuesPlaceholderRegexp = regexp.MustCompile(`"?\{\{\s*\.Values\.([A-Za-z_][A-Za-z0-9_.-]*)\s*\}\}"?`)

func normalizeLegacyManifestPlaceholders(manifest Manifest) Manifest {
	for index, item := range manifest.Bindings {
		item.BackendConfig.BackendUrl = normalizeLegacyHelmValuesPlaceholders(item.BackendConfig.BackendUrl)
		item.BackendConfig.RequestProxy.Headers = normalizeLegacyHelmValuesPlaceholdersMap(item.BackendConfig.RequestProxy.Headers)
		item.BackendConfig.RequestProxy.Query = normalizeLegacyHelmValuesPlaceholdersMap(item.BackendConfig.RequestProxy.Query)
		item.BackendConfig.FrontendProps = normalizeLegacyHelmValuesPlaceholdersMap(item.BackendConfig.FrontendProps)
		manifest.Bindings[index] = item
	}
	return manifest
}

func normalizeLegacyHelmValuesPlaceholders(value string) string {
	return legacyHelmValuesPlaceholderRegexp.ReplaceAllString(value, `${$1}`)
}

func normalizeLegacyHelmValuesPlaceholdersMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return values
	}
	normalized := make(map[string]string, len(values))
	for key, value := range values {
		normalized[key] = normalizeLegacyHelmValuesPlaceholders(value)
	}
	return normalized
}

// convertContainerToContainerV2 将旧版 Container 转换为新版 ContainerV2
func convertContainerToContainerV2(c Container) ContainerV2 {
	containerV2 := ContainerV2{
		Name:            strings.ReplaceAll(c.BaseInfo.Identifie, "_", "-"),
		Image:           c.Image,
		Command:         c.Cmd, // 假设 Cmd 对应 Command
		Ports:           convertPorts(c.Ports),
		Env:             convertEnvs(c.Env),
		Resources:       buildResourceRequirements(c.Cpu, c.Mem),
		VolumeMounts:    convertVolumeMounts(c.Volumes),
		ImagePullPolicy: v1.PullIfNotPresent, // 可根据需要调整默认策略
	}

	// SecurityContext 转换
	if c.Privileged || c.SecurityContext.RunAsUser != 0 || c.SecurityContext.FsGroup != 0 {
		containerV2.SecurityContext = &v1.SecurityContext{
			Privileged:   &c.Privileged,
			RunAsUser:    int64Ptr(int64(c.SecurityContext.RunAsUser)),
			RunAsGroup:   int64Ptr(int64(c.SecurityContext.RunAsGroup)),
			RunAsNonRoot: &c.SecurityContext.RunAsNonRoot,
		}
	}

	return containerV2
}

// convertPorts 转换 Port 列表
func convertPorts(ports []Port) []v1.ContainerPort {
	result := make([]v1.ContainerPort, 0)
	for _, p := range ports {
		protocol := v1.ProtocolTCP
		if p.Protocol != "" {
			// 支持 TCP/UDP/SCTP
			switch p.Protocol {
			case "UDP":
				protocol = v1.ProtocolUDP
			case "SCTP":
				protocol = v1.ProtocolSCTP
			default:
				protocol = v1.ProtocolTCP
			}
		}
		lbPort := 0
		if p.LbPort != "" {
			lbPort, _ = strconv.Atoi(p.LbPort)
		}
		containerPort := v1.ContainerPort{
			ContainerPort: int32(p.Port),
			Protocol:      protocol,
			HostPort:      int32(lbPort),
		}
		containerPort.Name = GetPortName(containerPort)
		result = append(result, containerPort)
	}
	return result
}

// convertEnvs 转换 Env 列表
func convertEnvs(envs []Env) []v1.EnvVar {
	result := make([]v1.EnvVar, 0)
	for _, e := range envs {
		envVar := v1.EnvVar{
			Name:  e.Name,
			Value: e.Value,
		}
		if e.ValueFrom != nil {
			if e.ValueFrom.FieldRef != nil {
				envVar.ValueFrom = &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath:  e.ValueFrom.FieldRef.FieldPath,
						APIVersion: e.ValueFrom.FieldRef.ApiVersion,
					},
				}
			} else if e.ValueFrom.ResourceFieldRef != nil {
				envVar.ValueFrom = &v1.EnvVarSource{
					ResourceFieldRef: &v1.ResourceFieldSelector{
						Resource: e.ValueFrom.ResourceFieldRef.Resource,
						Divisor:  resource.MustParse(strconv.Itoa(e.ValueFrom.ResourceFieldRef.Divisor)),
					},
				}
			}
		}
		result = append(result, envVar)
	}
	return result
}

// convertVolumeMounts 转换 Volumes 到 VolumeMounts
func convertVolumeMounts(vols []Volumes) []v1.VolumeMount {
	mounts := make([]v1.VolumeMount, 0)
	for _, vol := range vols {
		mount := v1.VolumeMount{
			Name:      generateVolumeName(vol),
			MountPath: vol.MountPath,
			SubPath:   vol.SubPath,
			ReadOnly:  false,
		}
		mounts = append(mounts, mount)
	}
	return mounts
}

func convertToV1Volume(vols []Volumes) []v1.Volume {
	existsVolumeName := make(map[string]bool)

	v1Vols := make([]v1.Volume, 0)
	for _, vol := range vols {
		volumeName := generateVolumeName(vol)
		if _, ok := existsVolumeName[volumeName]; ok {
			continue
		}

		v1Vol := v1.Volume{
			Name: volumeName,
		}

		switch vol.Type {
		case "hostStorage":
			hostPathType := v1.HostPathType(vol.HostPath.Type)
			v1Vol.VolumeSource = v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: vol.HostPath.Path,
					Type: &hostPathType,
				},
			}
		case "emptyStorage":
			v1Vol.VolumeSource = v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			}
		case "diskStorage":
			v1Vol.VolumeSource = v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: "",
				},
			}

		default:
			// 未知类型，降级为 emptyDir
			v1Vol.VolumeSource = v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			}
		}

		v1Vols = append(v1Vols, v1Vol)
		existsVolumeName[volumeName] = true
	}

	return v1Vols
}

// buildResourceRequirements 构建资源请求/限制
// 假设 Cpu 单位为毫核（m），Mem 单位为 MiB
func buildResourceRequirements(cpu int, mem int) v1.ResourceRequirements {
	res := v1.ResourceRequirements{}
	if cpu > 0 || mem > 0 {
		//res.Requests = v1.ResourceList{
		//	v1.ResourceCPU:    resource.MustParse(strconv.Itoa(cpu) + "m"),
		//	v1.ResourceMemory: resource.MustParse(strconv.Itoa(mem) + "Mi"),
		//}
		res.Limits = v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse(strconv.Itoa(cpu) + "m"),
			v1.ResourceMemory: resource.MustParse(strconv.Itoa(mem) + "Mi"),
		}
	}
	return res
}

func generateVolumeName(vol Volumes) string {
	// 从 MountPath 生成名称
	name := strings.TrimPrefix(vol.MountPath, "/")
	name = strings.ReplaceAll(name, "/", "-")
	if vol.Type == "diskStorage" {
		name = strings.ToLower(vol.Type)
	}
	name = "volume-" + name

	if len(name) > 63 {
		name = name[:63]
	}

	return name
}

// 辅助函数：int64 指针
func int64Ptr(i int64) *int64 {
	return &i
}

func GetPortName(port v1.ContainerPort) string {
	return "port-" + strconv.Itoa(int(port.ContainerPort))
}
