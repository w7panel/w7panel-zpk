export const pluginStorageVolumeName = 'site-storage';

/**
 * Keep the traditional application's shared storage volume in the
 * manifest. The editor deliberately stores an empty claimName; the Helm
 * packer resolves it from the selected traditional application release at render time.
 */
export function withPluginAppStorage(platform = {}) {
    const nextPlatform = { ...(platform || {}) };
    const volumes = [];
    let storageVolume = null;
    for (const volume of nextPlatform.volumes || []) {
        if (volume?.name !== pluginStorageVolumeName) {
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
        name: pluginStorageVolumeName,
        persistentVolumeClaim: { claimName: '' },
    });
    nextPlatform.volumes = volumes;
    return nextPlatform;
}

export function removePluginAppStorage(platform = {}) {
    const nextPlatform = { ...(platform || {}) };
    nextPlatform.volumes = (nextPlatform.volumes || [])
        .filter(volume => volume?.name !== pluginStorageVolumeName);
    return nextPlatform;
}

export function createPluginTraditionDependency(traditionalApplication) {
    const formulaURL = String(traditionalApplication?.formula_url || '').trim();
    let from = '';
    if (formulaURL) {
        try {
            from = new URL(formulaURL, window.location.origin).origin;
        } catch {
            from = '';
        }
    }
    return {
        identifie: traditionalApplication?.identifie || '',
        goodsId: Number(traditionalApplication?.goods_id || traditionalApplication?.goodsId || traditionalApplication?.id || 0),
        name: traditionalApplication?.name || '',
        subidentifie: '',
        subname: '',
        required: true,
        type: 'out',
        multipleInstances: true,
        temporary: true,
        from,
    };
}

export function applyPluginTraditionDependencyStartParams(
    dependencies,
    traditionName,
    traditionVersion,
) {
    return (dependencies || []).map(dependency => {
        if (dependency?.type !== 'out' || dependency?.identifie !== traditionName) {
            return dependency;
        }
        const startParams = { ...(dependency.startParams || {}) };
        if (traditionVersion) {
            startParams.IMAGE_VERSION = String(traditionVersion);
        } else {
            delete startParams.IMAGE_VERSION;
        }
        return { ...dependency, startParams };
    });
}

export function withPluginTraditionStartParams(startParams, traditionName) {
    const managedNames = new Set(['DOMAIN_URL', 'PVC_NAME']);
    const params = (startParams || []).filter(param => (
        !managedNames.has(String(param?.name || '').trim().toUpperCase())
    ));
    const moduleName = String(traditionName || '').trim();
    if (!moduleName) {
        return params;
    }
    return params.concat([
        {
            name: 'DOMAIN_URL',
            values_text: '%DOMAIN_URL%',
            module_name: moduleName,
            hidden: true,
        },
        {
            name: 'PVC_NAME',
            values_text: '%PVC_NAME%',
            module_name: moduleName,
            hidden: true,
        },
    ]);
}
