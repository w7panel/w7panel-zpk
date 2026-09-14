export const systemImageRootfsVolumeName = 'system-rootfs';

export const systemImageAnnotationKeys = Object.freeze({
    category: 'w7.cc/system_image_category',
    versions: 'w7.cc/image_version',
    rootfs: 'sysbox/rootfs-rw-layer',
});

export const systemImageFixedStartParamNames = Object.freeze([
    'IMAGE_VERSION',
    'global.cluster.storageRWmode',
    'global.cluster.storageSize',
    'global.cluster.storageClassName',
]);

export function isSystemImageFixedStartParamName(name) {
    return systemImageFixedStartParamNames.includes(name);
}

export function withSystemImageStartParams(startParams = [], versions = []) {
    const customParams = (startParams || [])
        .filter(item => !isSystemImageFixedStartParamName(item?.name));
    return [
        { mark: 'system-image', name: 'IMAGE_VERSION', title: '镜像版本', required: true, values_text: versions.join('|'), module_name: '', description: '选择要安装的镜像版本', type: 'select' },
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
