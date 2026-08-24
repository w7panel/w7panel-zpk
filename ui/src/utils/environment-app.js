const environmentApp = Object.freeze({
    dependencyIdentifie: 'w7-sitemanager',
    dependencyName: '站点管理',
    dependencySource: 'https://zpk.w7.cc',
    storageName: 'site-storage',
});

export function isEnvironmentAppDependency(record) {
    return record?.identifie == environmentApp.dependencyIdentifie;
}

export function createEnvironmentAppDependency() {
    return {
        identifie: environmentApp.dependencyIdentifie,
        name: environmentApp.dependencyName,
        subidentifie: '',
        subname: '',
        required: true,
        type: 'out',
        from: environmentApp.dependencySource,
    };
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

export function getEnvironmentAppRootfsAnnotation(json) {
    const container = json?.platform?.['container-v2']?.find(item => !item?.isInitContainer);
    const name = String(container?.name || '').trim();
    if (!name) return '';
    return JSON.stringify([{
        name,
        volumeName: environmentApp.storageName,
        path: `www/server/${name}/system`,
        persistentSpecialMounts: true,
    }]);
}
