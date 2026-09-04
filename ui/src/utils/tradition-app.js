export const traditionInstallTypes = Object.freeze({
    site: 'site',
    extension: 'extension',
});

export const traditionStorageVolumeName = 'site-storage';

/**
 * Keep the traditional application's shared environment volume in the
 * manifest. The editor deliberately stores an empty claimName; the Helm
 * packer resolves it from the selected environment release at render time.
 */
export function withTraditionAppStorage(platform = {}) {
    const nextPlatform = { ...(platform || {}) };
    const volumes = [];
    let storageVolume = null;
    for (const volume of nextPlatform.volumes || []) {
        if (volume?.name !== traditionStorageVolumeName) {
            volumes.push(volume);
            continue;
        }
        if (!storageVolume) {
            storageVolume = {
                ...volume,
                persistentVolumeClaim: {
                    ...(volume.persistentVolumeClaim || {}),
                    claimName: '',
                },
            };
        }
    }
    volumes.unshift(storageVolume || {
        name: traditionStorageVolumeName,
        persistentVolumeClaim: { claimName: '' },
    });
    nextPlatform.volumes = volumes;
    return nextPlatform;
}

export function removeTraditionAppStorage(platform = {}) {
    const nextPlatform = { ...(platform || {}) };
    nextPlatform.volumes = (nextPlatform.volumes || [])
        .filter(volume => volume?.name !== traditionStorageVolumeName);
    return nextPlatform;
}

export function createTraditionEnvironmentDependency(environment) {
    const formulaURL = String(environment?.formula_url || '').trim();
    let from = '';
    if (formulaURL) {
        try {
            from = new URL(formulaURL, window.location.origin).origin;
        } catch {
            from = '';
        }
    }
    return {
        identifie: environment?.identifie || '',
        goodsId: Number(environment?.goods_id || environment?.goodsId || environment?.id || 0),
        name: environment?.name || '',
        subidentifie: '',
        subname: '',
        required: true,
        type: 'out',
        multipleInstances: true,
        temporary: true,
        from,
    };
}

export function applyTraditionEnvironmentDependencyStartParams(
    dependencies,
    environmentName,
    environmentVersion,
) {
    return (dependencies || []).map(dependency => {
        if (dependency?.type !== 'out' || dependency?.identifie !== environmentName) {
            return dependency;
        }
        const startParams = { ...(dependency.startParams || {}) };
        if (environmentVersion) {
            startParams.IMAGE_VERSION = String(environmentVersion);
        } else {
            delete startParams.IMAGE_VERSION;
        }
        return { ...dependency, startParams };
    });
}

export function normalizeTraditionInstall(form) {
    const installType = form.installType == traditionInstallTypes.extension
        ? traditionInstallTypes.extension
        : traditionInstallTypes.site;
    return { installType };
}
