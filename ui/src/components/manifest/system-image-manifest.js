export const systemImageRootfsVolumeName = 'system-rootfs';

export const systemImageAnnotationKeys = Object.freeze({
    category: 'w7.cc/system_image_category',
    versions: 'w7.cc/image_version',
    rootfs: 'sysbox/rootfs-rw-layer',
});

const systemImageBuiltInStartParamNames = Object.freeze([
    'IMAGE_VERSION',
    'PVC_NAME',
    'global.cluster.storageRWmode',
    'global.cluster.storageSize',
    'global.cluster.storageClassName',
]);

export function isSystemImageBuiltInStartParamName(name) {
    const normalizedName = String(name || '').trim().toUpperCase();
    return systemImageBuiltInStartParamNames
        .some(item => item.toUpperCase() === normalizedName);
}

export function withSystemImageStartParams(startParams = [], versions = []) {
    const customParams = (startParams || [])
        .filter(item => !isSystemImageBuiltInStartParamName(item?.name));
    return [
        { mark: 'system-image', name: 'IMAGE_VERSION', title: '镜像版本', required: true, values_text: versions.join('|'), module_name: '', description: '选择要安装的镜像版本', type: 'select' },
        { name: 'PVC_NAME', title: '存储', required: true, values_text: '%PVC_NAME%', module_name: '', description: '安装时选择的 PVC 名称', type: 'text', hidden: false },
        { mark: 'storage', name: 'global.cluster.storageRWmode', title: '读写模式', required: true, values_text: '%STORAGE_RW_MODE%', module_name: '', description: '', type: 'text' },
        { mark: 'storage', name: 'global.cluster.storageSize', title: '存储大小', required: true, values_text: '%STORAGE_SIZE%', module_name: '', description: '', type: 'text' },
        { mark: 'storage', name: 'global.cluster.storageClassName', title: '存储类', required: true, values_text: '%STORAGE_CLASS_NAME%', module_name: '', description: '', type: 'text' },
        ...customParams,
    ];
}

export function systemImageRootfsAnnotation(
    applicationIdentifie = '',
    containerName = '',
) {
    const name = String(containerName || '').trim().replaceAll('_', '-');
    const identifie = String(applicationIdentifie || '').trim()
        .replace(/[^A-Za-z0-9._-]+/g, '-')
        .replace(/^-+|-+$/g, '');
    if (!name || !identifie) return '';
    const imageVersion = '${IMAGE_VERSION}';
    const rootfsPathName = `${identifie}-${imageVersion}`;
    return JSON.stringify([{
        name,
        volumeName: systemImageRootfsVolumeName,
        path: `${systemImageRootfsVolumeName}/${rootfsPathName}/system`,
        persistentSpecialMounts: true,
    }]);
}

export function withSystemImageRuntime(
    platform = {},
    sourceAnnotations = {},
    applicationIdentifie = '',
    imageTemplate = '',
    command = [],
) {
    const nextPlatform = { ...(platform || {}) };
    const annotations = { ...(sourceAnnotations || {}) };
    delete nextPlatform.container;

    const containers = Array.isArray(nextPlatform['container-v2'])
        ? nextPlatform['container-v2'].map(container => ({ ...container }))
        : [];
    if (!containers.length) {
        containers.push({
            name: applicationIdentifie || 'system-image',
            image: '',
            imagePullPolicy: 'IfNotPresent',
        });
    }

    const mainContainer = containers[0];
    mainContainer.image = String(imageTemplate || '').trim();
    const normalizedCommand = (command || [])
        .map(item => String(item || '').trim())
        .filter(Boolean);
    if (normalizedCommand.length) {
        mainContainer.command = normalizedCommand;
    } else {
        delete mainContainer.command;
    }
    mainContainer.volumeMounts = [{
        name: systemImageRootfsVolumeName,
        mountPath: '/system-rootfs',
    }];

    nextPlatform['container-v2'] = containers;
    nextPlatform.workload = { ...(nextPlatform.workload || {}), type: 'Deployment' };
    nextPlatform.runtimeClassName = 'sysbox-runc';
    nextPlatform.hostUsers = false;
    nextPlatform.volumes = [{
        name: systemImageRootfsVolumeName,
        persistentVolumeClaim: { claimName: '' },
    }];

    delete annotations[systemImageAnnotationKeys.rootfs];
    const rootfs = systemImageRootfsAnnotation(applicationIdentifie, mainContainer?.name);
    if (rootfs) {
        annotations[systemImageAnnotationKeys.rootfs] = rootfs;
    }
    return { platform: nextPlatform, annotations };
}

export function systemImageFormDefaults() {
    return {
        systemImageCategory: 'operating-system',
        systemImageTemplate: '',
        systemImageVersions: [],
    };
}

export function systemImageValidationRules(component) {
    return {
        systemImageCategory: [
            { required: true, message: '请选择系统镜像分类', trigger: 'change' },
        ],
        systemImageTemplate: [{
            required: true,
            trigger: 'blur',
            validator: (value, callback) => component.validateSystemImageTemplate(value, callback),
        }],
        systemImageVersions: [{
            required: true,
            trigger: 'change',
            validator: (value, callback) => component.validateSystemImageVersions(value, callback),
        }],
    };
}

export const systemImageManifestMethods = {
    initSystemImageManifest(platform = {}) {
        this.form.systemImageTemplate = platform?.['container-v2']?.[0]?.image || '';
    },
    applySystemImageManifest() {
        if (this.form.type != 'system-image') return;
        this.form.systemImageVersions = this.normalizeApplicationVersions(this.form.systemImageVersions);
        this.form.startParams = this.systemImageStartParams();
        this.ensureSystemImageContainer();
    },
    filterSystemImageAnnotations(annotation = {}, type = this.form.type) {
        const filtered = { ...(annotation || {}) };
        if (type != 'system-image') {
            Object.values(systemImageAnnotationKeys)
                .filter(key => type != 'tradition'
                    || ![systemImageAnnotationKeys.versions, systemImageAnnotationKeys.rootfs].includes(key))
                .forEach(key => delete filtered[key]);
            return filtered;
        }
        const versions = this.normalizeApplicationVersions(this.form.systemImageVersions);
        Object.assign(filtered, {
            [systemImageAnnotationKeys.category]: this.form.systemImageCategory || 'operating-system',
            [systemImageAnnotationKeys.versions]: versions.join(','),
        });
        return filtered;
    },
    cleanupSystemImageType(previousType, nextType) {
        if (previousType == 'system-image' && nextType != 'system-image') {
            this.form.startParams = (this.form.startParams || [])
                .filter(item => !isSystemImageBuiltInStartParamName(item?.name));
            delete this.json.platform.hostUsers;
            if (this.json.platform.runtimeClassName == 'sysbox-runc') {
                delete this.json.platform.runtimeClassName;
            }
            this.json.platform.volumes = (this.json.platform.volumes || [])
                .filter(item => item?.name != systemImageRootfsVolumeName);
            if (!this.json.platform.volumes.length) delete this.json.platform.volumes;
            (this.json.platform['container-v2'] || []).forEach(container => {
                container.volumeMounts = (container.volumeMounts || [])
                    .filter(item => item?.name != systemImageRootfsVolumeName);
                if (!container.volumeMounts.length) delete container.volumeMounts;
            });
            delete this.json.platform['container-v2'];
            this.form.containers = [];
            this.form.cmd = [''];
        }
        if (previousType != 'system-image' && nextType == 'system-image') {
            delete this.json.source;
            delete this.json.web;
            delete this.json.platform.ingress;
            delete this.json.platform.volumeClaimTemplates;
            delete this.json.platform.container;
            this.json.platform['container-v2'] = [];
            this.form.containers = [];
            this.form.cmd = [''];
            this.form.ingress = [];
            this.form.domain = false;
        }
    },
    validateSystemImageTemplate(value, callback) {
        const template = String(value || '').trim();
        if (!template) { callback('请输入镜像地址'); return; }
        if (/\s/.test(template)) { callback('镜像地址不能包含空格或换行'); return; }
        if (!template.includes('{version}')) { callback('镜像地址必须包含 {version} 占位符'); return; }
        const unsupported = [...new Set(template.match(/\{[^{}]+\}/g) || [])]
            .filter(item => item != '{version}');
        callback(unsupported.length
            ? '镜像地址包含不支持的占位符：' + unsupported.join('、')
            : undefined);
    },
    validateSystemImageVersions(value, callback) {
        const versions = this.normalizeApplicationVersions(value);
        if (!versions.length) { callback('请输入至少一个系统版本'); return; }
        const invalid = versions
            .filter(version => !/^[A-Za-z0-9_][A-Za-z0-9._-]{0,127}$/.test(version));
        callback(invalid.length
            ? '系统版本格式不正确：' + invalid.slice(0, 3).join('、')
            : undefined);
    },
    systemImageStartParams() {
        return withSystemImageStartParams(
            this.form.startParams,
            this.normalizeApplicationVersions(this.form.systemImageVersions),
        );
    },
    syncSystemImageConfig() {
        if (this.form.type != 'system-image') return;
        this.form.systemImageVersions = this.normalizeApplicationVersions(this.form.systemImageVersions);
        this.form.startParams = this.systemImageStartParams();
        this.form.storage = true;
        this.json.platform = this.json.platform || {};
        this.ensureSystemImageContainer();
        this.changeForm();
    },
    ensureSystemImageContainer() {
        if (this.form.type != 'system-image') return;
        delete this.json.source;
        delete this.json.web;
        const applicationIdentifie = (this.form.author && this.form.identifie)
            ? `${this.form.author}-${this.form.identifie}`
            : 'system-image';
        const result = withSystemImageRuntime(
            this.json.platform,
            this.json.application?.annotation || {},
            applicationIdentifie,
            this.form.systemImageTemplate,
            this.form.cmd,
        );
        this.json.platform = result.platform;
        this.form.containers = this.json.platform['container-v2'];
        this.containerPluginData.kind = 'Deployment';
        this.json.application = this.json.application || {};
        this.json.application.annotation = result.annotations;
    },
    applySystemImageAnnotationForm(annotation = {}) {
        this.form.systemImageCategory = String(
            annotation[systemImageAnnotationKeys.category] || 'operating-system',
        );
        this.form.systemImageVersions = this.normalizeApplicationVersions(
            annotation[systemImageAnnotationKeys.versions],
        );
    },
};
