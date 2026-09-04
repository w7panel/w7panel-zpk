const environmentApp = Object.freeze({
    nginxDependencyIdentifie: 'w7-sitemanager-nginx',
    nginxDependencyName: 'w7-sitemanager-nginx',
    nginxDependencySource: 'https://zpk.fan.b2.sz.w7.com',
    storageName: 'site-storage',
});

export const environmentStorageVolumeName = environmentApp.storageName;
export const environmentNginxIdentifie = environmentApp.nginxDependencyIdentifie;
export const environmentSystemRebootRestoreAnnotation = 'w7.cc/system-reboot-restore';
export const environmentSysboxRootfsAnnotation = 'sysbox/rootfs-rw-layer';

export const environmentNginxDependency = Object.freeze({
    identifie: environmentApp.nginxDependencyIdentifie,
    name: environmentApp.nginxDependencyName,
    source: environmentApp.nginxDependencySource,
});

export function isEnvironmentAppDependency(record) {
    return record?.identifie == environmentApp.nginxDependencyIdentifie;
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
 * Bind w7-sitemanager-nginx's PVC startup parameter to the environment application's
 * PVC parameter. The imported nginx manifest is returned as a copy so the
 * caller can safely update its child file and editor state together.
 */
export function withEnvironmentNginxPvcDependencySource(
    manifest = {},
    mainApplicationIdentifie = '',
) {
    const nextManifest = cloneManifest(manifest);
    const identifie = String(mainApplicationIdentifie || '').trim();
    if (!identifie) {
        return nextManifest;
    }

    const platform = nextManifest.platform || {};
    const parameterLists = [platform.startParams, platform.container?.startParams]
        .filter(Array.isArray);
    for (const parameters of parameterLists) {
        const pvcParameter = parameters.find(item =>
            String(item?.name || '').trim().toLowerCase() === 'pvc_name');
        if (!pvcParameter) {
            continue;
        }
        const name = String(pvcParameter.name || '').trim();
        pvcParameter.dependencySource = {
            ...(pvcParameter.dependencySource || {}),
            identifie,
            name: name || 'PVC_NAME',
        };
        break;
    }
    return nextManifest;
}

export function removeEnvironmentAppCodeStorage(json) {
    const platform = json?.platform;
    if (!platform) return;
    platform.volumes = (platform.volumes || [])
        .filter(item => item?.name != environmentApp.storageName);
    (platform['container-v2'] || []).forEach(container => {
        container.volumeMounts = (container.volumeMounts || [])
            .filter(item => item?.name != environmentApp.storageName);
    });
}

export function environmentAppRootfsAnnotation(container = {}) {
    const name = String(container?.name || '').trim();
    if (!name) return '';
    return JSON.stringify([{
        name,
        volumeName: environmentStorageVolumeName,
        path: `www/server/${name}/system`,
        persistentSpecialMounts: true,
    }]);
}

export function withEnvironmentAppSysbox(
    platform = {},
    sourceAnnotations = {},
    restoreOverride,
) {
    const nextPlatform = { ...(platform || {}) };
    const annotations = { ...(sourceAnnotations || {}) };
    let restore = restoreOverride;
    if (restore === undefined) {
        restore = true;
        const value = annotations[environmentSystemRebootRestoreAnnotation];
        if (value !== undefined) {
            restore = typeof value === 'boolean'
                ? value
                : String(value).trim().toLowerCase() !== 'false';
        }
    }

    delete annotations[environmentSysboxRootfsAnnotation];
    if (restore) {
        delete nextPlatform.hostUsers;
        if (nextPlatform.runtimeClassName === 'sysbox-runc') {
            delete nextPlatform.runtimeClassName;
        }
        return { platform: nextPlatform, annotations };
    }

    nextPlatform.runtimeClassName = 'sysbox-runc';
    nextPlatform.hostUsers = false;
    const containers = nextPlatform['container-v2'] || [];
    const container = containers.find(item => !item?.isInitContainer);
    const rootfs = environmentAppRootfsAnnotation(container);
    if (rootfs) {
        annotations[environmentSysboxRootfsAnnotation] = rootfs;
    }
    return { platform: nextPlatform, annotations };
}

export function withEnvironmentAppStorage(platform = {}) {
    const nextPlatform = { ...(platform || {}) };
    const containers = Array.isArray(nextPlatform['container-v2'])
        ? nextPlatform['container-v2'].map(container => ({
            ...container,
            volumeMounts: [...(container?.volumeMounts || [])],
        }))
        : [];
    const volumes = (nextPlatform.volumes || [])
        .filter(item => item?.name !== environmentStorageVolumeName);
    volumes.push({
        name: environmentStorageVolumeName,
        persistentVolumeClaim: { claimName: '' },
    });
    nextPlatform.volumes = volumes;

    const container = containers.find(item => !item?.isInitContainer);
    if (container) {
        container.volumeMounts = container.volumeMounts
            .filter(item => item?.name !== environmentStorageVolumeName);
        container.volumeMounts.push({
            name: environmentStorageVolumeName,
            mountPath: '/www/wwwroot',
            subPath: 'nginx-web-dir',
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
 * environment form does not need to know every historical port shape.
 */
export function environmentAppDefaultContainerPort(source = {}, fallback = 80) {
    const value = source?.platform || source || {};
    for (const container of value['container-v2'] || []) {
        if (container?.isInitContainer) continue;
        const port = firstPort(container);
        if (port) return port;
    }
    const legacyPort = firstPort(value.container) || firstPort(value.port);
    return legacyPort || fallback;
}

function matchesApplication(source, identifie) {
    if (!source || !identifie) return false;
    const names = [source.identifie, source.identifier, source.id, source.name]
        .filter(Boolean)
        .map(value => String(value));
    return names.some(value => value === identifie || value.endsWith(`-${identifie}`));
}

/**
 * Resolve the Nginx dependency's service port from imported child manifests or
 * app_ports. A fallback is only used while the dependency is not imported yet
 * (for example during the first render of the gateway switch).
 */
export function environmentNginxPort(sources = [], fallback = 80) {
    const values = Array.isArray(sources) ? sources : [sources];
    for (const source of values) {
        const data = source?.data || source;
        const manifest = data?.platform ? data : source?.manifest;
        if (manifest?.platform && matchesApplication(manifest?.application || {}, environmentNginxIdentifie)) {
            const ingressPort = firstPort((manifest.platform.ingress || [])
                .flatMap(item => (item?.routes || []).map(route => route?.backend)));
            if (ingressPort) return ingressPort;
            const port = environmentAppDefaultContainerPort(manifest, 0);
            if (port) return port;
        }
        if (matchesApplication(source, environmentNginxIdentifie)) {
            const port = firstPort(source.port)
                || firstPort(source.ports)
                || environmentAppDefaultContainerPort(manifest || data, 0)
                || environmentAppDefaultContainerPort(source, 0);
            if (port) return port;
        }
    }
    return Number(fallback) > 0 ? Number(fallback) : 80;
}

export function withEnvironmentAppIngress(
    platform = {},
    applicationIdentifie = '',
    nginxGateway = false,
    nginxPortSource = [],
) {
    const nextPlatform = { ...(platform || {}) };
    const containerPort = environmentAppDefaultContainerPort(nextPlatform);
    const nginxPort = environmentNginxPort(nginxPortSource, containerPort);
    const defaultBackend = nginxGateway
        ? { name: environmentNginxIdentifie, port: nginxPort, match: 'Prefix' }
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
                backend.name = environmentNginxIdentifie;
                backend.port = nginxPort;
                backend.match = backend.match || 'Prefix';
                return;
            }
            if (backend.name === environmentNginxIdentifie || !backend.name) {
                backend.name = applicationIdentifie;
                backend.port = containerPort;
                backend.match = backend.match || 'Prefix';
            }
        });
    });
    return nextPlatform;
}
