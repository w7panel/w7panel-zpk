const traditionApp = Object.freeze({
    toolDependencyIdentifie: 'w7-traditiontool',
    toolDependencyName: 'w7-traditiontool',
    toolDependencySource: 'https://zpk.fan.b2.sz.w7.com',
    storageName: 'site-storage',
});

export const traditionStorageVolumeName = traditionApp.storageName;
export const traditionToolIdentifie = traditionApp.toolDependencyIdentifie;
export const traditionToolGatewayStartParam = 'gatewayEnabled';
export const traditionSystemRebootRestoreAnnotation = 'w7.cc/system-reboot-restore';
export const traditionSysboxRootfsAnnotation = 'sysbox/rootfs-rw-layer';
export const traditionSysboxRuntimeClassName = 'sysbox-runc-lite';

export const traditionToolDependency = Object.freeze({
    identifie: traditionApp.toolDependencyIdentifie,
    name: traditionApp.toolDependencyName,
    source: traditionApp.toolDependencySource,
});

export function isTraditionAppDependency(record) {
    return record?.identifie == traditionApp.toolDependencyIdentifie;
}

function cloneManifest(manifest = {}) {
    if (!manifest || typeof manifest !== 'object' || Array.isArray(manifest)) {
        return {};
    }
    try {
        return JSON.parse(JSON.stringify(manifest));
    } catch {
        return { ...manifest };
    }
}

/**
 * Bind w7-traditiontool's PVC startup parameter to the traditional application
 * and enable its gateway support when requested.
 */
export function withTraditionToolConfig(
    manifest = {},
    mainApplicationIdentifie = '',
    gatewayEnabled = false,
) {
    const nextManifest = cloneManifest(manifest);
    const identifie = String(mainApplicationIdentifie || '').trim();
    nextManifest.platform = nextManifest.platform || {};
    const platform = nextManifest.platform;
    let startParams = Array.isArray(platform.startParams)
        ? platform.startParams
        : [];
    startParams = startParams
        .filter(item => String(item?.name || '').trim() !== traditionToolGatewayStartParam)
        .map(item => ({ ...item }));
    if (identifie) {
        const pvcParameter = startParams.find(item =>
            String(item?.name || '').trim().toLowerCase() === 'pvc_name');
        if (pvcParameter) {
            pvcParameter.module_name = identifie;
            pvcParameter.hidden = true;
            delete pvcParameter.dependencySource;
        }
    }
    startParams.push({
        name: traditionToolGatewayStartParam,
        title: '开启网关工具',
        description: '启用传统应用网关工具',
        mark: 'tradition-tool-gateway',
        required: false,
        type: 'text',
        values_text: String(Boolean(gatewayEnabled)),
        module_name: '',
        hidden: true,
    });
    platform.startParams = startParams;
    return nextManifest;
}

export function removeTraditionAppCodeStorage(json) {
    const platform = json?.platform;
    if (!platform) return;
    platform.volumes = (platform.volumes || [])
        .filter(item => item?.name != traditionApp.storageName);
    (platform['container-v2'] || []).forEach(container => {
        container.volumeMounts = (container.volumeMounts || [])
            .filter(item => item?.name != traditionApp.storageName);
    });
}

export function traditionAppRootfsAnnotation(
    applicationIdentifie = '',
    containerName = '',
) {
    const identifie = String(applicationIdentifie || '').trim()
        .replace(/[^A-Za-z0-9._-]+/g, '-')
        .replace(/^-+|-+$/g, '');
    const name = String(containerName || '').trim().replaceAll('_', '-');
    if (!identifie || !name) return '';
    // Keep the manifest independent of Helm syntax. The packer turns this
    // platform placeholder into .Values.IMAGE_VERSION when generating a chart.
    const version = '${IMAGE_VERSION}';
    return JSON.stringify([{
        name,
        volumeName: traditionStorageVolumeName,
        path: `www/server/${identifie}-${version}/system`,
        persistentSpecialMounts: true,
    }]);
}

export function withTraditionAppSysbox(
    platform = {},
    sourceAnnotations = {},
    restoreOverride,
    applicationIdentifie = '',
) {
    const nextPlatform = { ...(platform || {}) };
    const annotations = { ...(sourceAnnotations || {}) };
    let restore = restoreOverride;
    if (restore === undefined) {
        restore = true;
        const value = annotations[traditionSystemRebootRestoreAnnotation];
        if (value !== undefined) {
            restore = typeof value === 'boolean'
                ? value
                : String(value).trim().toLowerCase() !== 'false';
        }
    }

    delete annotations[traditionSysboxRootfsAnnotation];
    delete nextPlatform.hostUsers;
    if (restore) {
        if (nextPlatform.runtimeClassName === traditionSysboxRuntimeClassName) {
            delete nextPlatform.runtimeClassName;
        }
        return { platform: nextPlatform, annotations };
    }

    nextPlatform.runtimeClassName = traditionSysboxRuntimeClassName;
    const containers = nextPlatform['container-v2'] || [];
    const container = containers.find(item => !item?.isInitContainer);
    const rootfs = traditionAppRootfsAnnotation(applicationIdentifie, container?.name);
    if (rootfs) {
        annotations[traditionSysboxRootfsAnnotation] = rootfs;
    }
    return { platform: nextPlatform, annotations };
}

export function withTraditionAppStorage(platform = {}) {
    const nextPlatform = { ...(platform || {}) };
    const containers = Array.isArray(nextPlatform['container-v2'])
        ? nextPlatform['container-v2'].map(container => ({
            ...container,
            volumeMounts: [...(container?.volumeMounts || [])],
        }))
        : [];
    const volumes = (nextPlatform.volumes || [])
        .filter(item => item?.name !== traditionStorageVolumeName);
    volumes.push({
        name: traditionStorageVolumeName,
        persistentVolumeClaim: { claimName: '' },
    });
    nextPlatform.volumes = volumes;

    const container = containers.find(item => !item?.isInitContainer);
    if (container) {
        container.volumeMounts = container.volumeMounts
            .filter(item => item?.name !== traditionStorageVolumeName);
        container.volumeMounts.push({
            name: traditionStorageVolumeName,
            mountPath: '/www/wwwroot',
            subPath: 'nginx-web-dir',
        }, {
            name: traditionStorageVolumeName,
            mountPath: '/www/server',
            subPath: 'server-dir',
        });
    }
    nextPlatform['container-v2'] = containers;
    return nextPlatform;
}

function firstPort(value) {
    if (value === undefined || value === null || value === '') return 0;
    if (typeof value === 'number' || typeof value === 'string') {
        const port = Number(value);
        return Number.isFinite(port) && port > 0 ? port : 0;
    }
    if (Array.isArray(value)) {
        for (const item of value) {
            const port = firstPort(item);
            if (port) return port;
        }
        return 0;
    }
    if (typeof value !== 'object') return 0;
    for (const candidate of [
        value.containerPort,
        value.port,
        value.targetPort,
        value.servicePort,
    ]) {
        const port = firstPort(candidate);
        if (port) return port;
    }
    return firstPort(value.ports);
}

/**
 * Resolve the first workload port from a manifest/platform or the compact
 * app_ports entries used by the editor. Keeping this lookup here means the
 * tradition form does not need to know every historical port shape.
 */
export function traditionAppDefaultContainerPort(source = {}, fallback = 80) {
    const value = source?.platform || source || {};
    for (const container of value['container-v2'] || []) {
        if (container?.isInitContainer) continue;
        const port = firstPort(container);
        if (port) return port;
    }
    const legacyPort = firstPort(value.container) || firstPort(value.port);
    return legacyPort || fallback;
}

export function withTraditionAppIngress(
    platform = {},
    applicationIdentifie = '',
    nginxGateway = false,
) {
    const nextPlatform = { ...(platform || {}) };
    const containerPort = traditionAppDefaultContainerPort(nextPlatform);
    const defaultBackend = nginxGateway
        ? { name: traditionToolIdentifie, port: 80, match: 'Prefix' }
        : { name: applicationIdentifie, port: containerPort, match: 'Prefix' };
    nextPlatform.ingress = (platform?.ingress || []).map(item => ({
        ...item,
        routes: (item?.routes || []).map(route => ({
            ...route,
            backend: { ...(route?.backend || {}) },
        })),
    }));
    if (!nextPlatform.ingress.length) {
        nextPlatform.ingress = [{ name: '/', routes: [{ path: '/', backend: { ...defaultBackend } }] }];
        return nextPlatform;
    }

    nextPlatform.ingress.forEach(item => {
        if (!item.routes.length) {
            item.routes.push({ path: '/', backend: { ...defaultBackend } });
        }
        item.routes.forEach(route => {
            const backend = route.backend;
            if (nginxGateway) {
                backend.name = traditionToolIdentifie;
                backend.port = 80;
                backend.match = backend.match || 'Prefix';
                return;
            }
            // Without the NGINX gateway every traditional-app route must
            // target the application service itself.
            backend.name = applicationIdentifie;
            backend.port = containerPort;
            backend.match = backend.match || 'Prefix';
        });
    });
    return nextPlatform;
}
