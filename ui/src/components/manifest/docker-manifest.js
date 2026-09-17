const dockerPVCStartParamName = 'PVC_NAME';

function hasDockerPersistentVolumeClaim(volumes = []) {
    return (volumes || []).some(volume => Boolean(volume?.persistentVolumeClaim));
}

export function isDockerBuiltInStartParamName(name, volumes = []) {
    return hasDockerPersistentVolumeClaim(volumes)
        && String(name || '').trim().toUpperCase() === dockerPVCStartParamName;
}

export function withDockerBuiltInStartParams(startParams = [], volumes = []) {
    if (!hasDockerPersistentVolumeClaim(volumes)) {
        return startParams || [];
    }
    const params = (startParams || [])
        .filter(item => !isDockerBuiltInStartParamName(item?.name, volumes));
    return [...params, {
        name: dockerPVCStartParamName,
        title: '存储',
        required: true,
        values_text: '%PVC_NAME%',
        module_name: '',
        description: '安装时选择的 PVC 名称',
        type: 'text',
        hidden: false,
    }];
}

export function normalizeDockerApplicationType(type) {
    return type === 'front' ? 'docker' : (type || 'docker');
}

export const dockerManifestMethods = {
    syncDockerBuiltInStartParams() {
        this.form.startParams = withDockerBuiltInStartParams(
            this.form.startParams,
            this.json?.platform?.volumes,
        );
    },
    isDockerBuiltInStartParam(name) {
        return isDockerBuiltInStartParamName(
            name,
            this.json?.platform?.volumes,
        );
    },
    applyDockerPlatform() {
        if (this.form.type != 'docker') return;
        this.applyPlatformShells();
        this.json.platform = this.json.platform || {};
        this.json.platform.ingress = (this.form.ingress || []).map(item => ({
            name: item.name,
            routes: this.formatIngressRoutes(item.routes),
        }));
    },
};
