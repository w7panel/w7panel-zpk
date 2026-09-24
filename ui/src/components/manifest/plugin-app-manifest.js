import myAxios from '@/utils/index';

const pluginStorageVolumeName = 'site-storage';

const pluginTraditionStartParamDefinitions = Object.freeze([
    {
        name: 'DOMAIN_URL',
        title: '站点域名',
        values_text: '%DOMAIN_URL%',
        description: '传统应用的站点域名，用于找到对应的站点目录',
    },
    {
        name: 'PVC_NAME',
        title: '存储',
        values_text: '%PVC_NAME%',
        description: '与所选传统应用共用的站点存储空间',
    },
    {
        name: 'TRADITION_PLUGIN_POLICY',
        title: '插件文件优先级配置',
        values_text: '%TRADITION_PLUGIN_POLICY%',
        description: '从所选传统应用读取插件文件优先级配置',
    },
]);

export function isPluginTraditionStartParamName(name) {
    const normalizedName = String(name || '').trim().toUpperCase();
    return pluginTraditionStartParamDefinitions
        .some(item => item.name === normalizedName);
}

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
    const moduleName = String(traditionName || '').trim();
    const params = (startParams || []).filter(param => (
        !isPluginTraditionStartParamName(param?.name)
    ));
    if (!moduleName) {
        return params;
    }
    const managedParams = pluginTraditionStartParamDefinitions
        .map(item => ({
            ...item,
            required: true,
            type: 'text',
            module_name: moduleName,
            hidden: true,
        }));
    return [...managedParams, ...params];
}

export function appPluginFormDefaults() {
    return {
        traditionName: '',
        traditionVersion: '',
        traditionImageTemplate: '',
    };
}

export function appPluginManifestState() {
    return {
        traditionList: [],
    };
}

export const appPluginManifestMethods = {
    initAppPluginManifest(platform = {}) {
        this.form.traditionName = platform?.plugin?.traditionName || '';
        this.form.traditionVersion = platform?.plugin?.traditionVersion || '';
        this.form.traditionImageTemplate = platform?.plugin?.traditionImageTemplate || '';
    },
    applyAppPluginManifest(platform = {}) {
        if (this.form.type != 'app-plugin') return platform;
        this.syncApplicationBuiltInStartParams();
        const nextPlatform = withPluginAppStorage(platform);
        const selectedTradition = this.traditionList
            ?.find?.(item => item.identifie == this.form.traditionName);
        nextPlatform.plugin = {
            traditionName: this.form.traditionName,
            traditionVersion: this.form.traditionVersion,
            traditionLanguage: selectedTradition?.environment_language
                || nextPlatform?.plugin?.traditionLanguage
                || '',
            traditionImageTemplate: this.form.traditionImageTemplate,
        };
        return nextPlatform;
    },
    cleanupAppPluginType(previousType, nextType) {
        if (previousType != 'app-plugin' || nextType == 'app-plugin') return;
        this.form.startParams = (this.form.startParams || [])
            .filter(item => !isPluginTraditionStartParamName(item?.name));
        delete this.json.platform.plugin;
        this.json.platform = removePluginAppStorage(this.json.platform);
    },
    async changeEnv(item) {
        if (!item.identifie || item.identifie == this.form.traditionName) return;
        this.form.depends = this.form.depends?.filter?.(dependency => !dependency.temporary) || [];
        const index = this.form.depends
            ?.findIndex(dependency => dependency.identifie == item.identifie && dependency.name == item.name);
        if (index != -1) {
            this.form.depends[index] = {
                ...this.form.depends[index],
                ...createPluginTraditionDependency(item),
            };
        } else {
            this.form.depends.push(createPluginTraditionDependency(item));
        }
        this.form.traditionName = item.identifie;
        this.form.traditionImageTemplate = String(
            item.image_template || item.image || item.extra?.image || '',
        ).trim();
        this.form.traditionVersion = item.versions?.[0] || '';
        this.syncApplicationBuiltInStartParams();
        this.getStart();
        this.changeForm();
    },
    getTraditionList() {
        const listUrl = 'https://api.zm.w7.com/zpk-market/formula/list';
        return myAxios.post(listUrl, {
            status: [2, 99],
            page: 1,
            limit: 99,
            tag: '传统应用',
        }, { dontalert: true }).then(res => {
            const keepSavedState = Boolean(this.savedEditorState)
                && !this.hasUnsavedChanges?.();
            const list = res.data?.data?.list || [];
            this.traditionList = list.map(item => {
                const identifie = item.identifie || item.identify || item.identifier
                    || item.formula_identifie || item.formula_identify || '';
                return {
                    ...item,
                    identifie,
                    name: item.name || item.formula_name || item.title || identifie,
                    environment_language: String(item.environment_language || '').trim(),
                    image_template: String(
                        item.image_template || item.image || item.extra?.image || '',
                    ).trim(),
                    versions: String(item.support_version || '')
                        .split(',')
                        .map(version => version.trim())
                        .filter(Boolean),
                };
            }).filter(item => item.identifie);
            this.form.depends?.forEach?.(dependency => {
                if (this.form.type != 'app-plugin') return;
                const selected = this.traditionList
                    .find(item => item.identifie == this.form.traditionName);
                if (selected && dependency.name == selected.name
                    && dependency.identifie == selected.identifie) {
                    dependency.goodsId = Number(
                        selected.goods_id || selected.goodsId || selected.id || 0,
                    );
                    dependency.temporary = true;
                }
            });
            if (keepSavedState) this.markManifestSaved?.();
        }).catch(() => {
            this.traditionList = [];
        });
    },
    isPluginTraditionDependency(record) {
        if (this.form.type != 'app-plugin') return false;
        const dependencyIdentify = String(record?.identifie || '').replaceAll('_', '-');
        const traditionIdentify = String(this.form.traditionName || '').replaceAll('_', '-');
        return Boolean(traditionIdentify) && dependencyIdentify == traditionIdentify;
    },
};
