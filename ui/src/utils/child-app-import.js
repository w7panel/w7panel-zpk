import jsyaml from 'js-yaml';
import { getZpkBaseURL } from '@/utils/request-base';

const officialRepositoryURL = 'https://zpk.w7.cc';

export function childImportRepositoryBaseURL(tab) {
    if (tab === 'official') {
        return officialRepositoryURL;
    }
    const baseURL = getZpkBaseURL();
    if (/^https?:\/\//i.test(baseURL)) {
        return baseURL;
    }
    return `${window.location.origin}/${String(baseURL || 'zpk').replace(/^\/+/, '')}`;
}

export function childImportRepositoryURL(tab) {
    return tab === 'official'
        ? `${childImportRepositoryBaseURL(tab)}/zpk/respo/list?status=2&status=99`
        : '/respo/list';
}

export function normalizeChildImportList(list = []) {
    return (Array.isArray(list) ? list : [])
        .filter(item => item?.identifie && item?.install_only_once)
        .map(item => ({
            ...item,
            name: item.name || item.identifie,
        }));
}

export function fetchChildImportList(client, tab, params = {}) {
    return client.get(childImportRepositoryURL(tab), {
        params,
        dontalert: true,
    }).then(response => normalizeChildImportList(response?.data?.data?.list || []));
}

export function importChildApplication(client, params) {
    return client.post('/respo/manifest/import', params)
        .then(response => {
            const manifests = response?.data?.data?.manifests || {};
            return normalizeImportedManifests(manifests);
        });
}

export function normalizeImportedManifests(manifests) {
    const entries = [];
    const source = Array.isArray(manifests)
        ? manifests.map(item => [item?.identifie, item?.manifest])
        : Object.entries(manifests || {});
    const seen = new Set();
    source.forEach(([fallbackIdentifie, content]) => {
        if (!content) return;
        let manifest = content;
        if (typeof content === 'string') {
            try {
                manifest = jsyaml.load(content) || {};
            } catch {
                return;
            }
        }
        const identifie = manifest?.application?.identifie || fallbackIdentifie;
        if (!identifie || seen.has(identifie)) return;
        seen.add(identifie);
        entries.push({
            identifie,
            name: manifest?.application?.name || identifie,
            manifest: typeof content === 'string' ? content : jsyaml.dump(manifest),
            data: manifest,
        });
    });
    return entries;
}

export function importedChildDependency(entry, required = true, source = {}) {
    const dependency = {
        identifie: entry.identifie,
        name: entry.name || entry.identifie,
        subidentifie: '',
        subname: '',
        required,
        type: 'in',
        from: source.from || '',
    };
    if (source.version) {
        dependency.version = source.version;
    }
    return dependency;
}

export function importedChildFilePath(identifie) {
    return `${identifie}/manifest.yaml`;
}

export function getImportedChildIdentifies(rootIdentifie, list = {}, dependencies = []) {
    const identifies = new Set([rootIdentifie].filter(Boolean));
    const rootFile = importedChildFilePath(rootIdentifie);
    const rootDependency = (dependencies || []).find(item => item?.identifie == rootIdentifie);
    const rootRaw = list?.[rootFile] || rootDependency?.manifest || '';
    try {
        const root = typeof rootRaw == 'string' ? (jsyaml.load(rootRaw) || {}) : rootRaw;
        (root?.platform?.depends || []).forEach(item => {
            if (item?.identifie) { identifies.add(item.identifie); }
        });
    } catch { }
    return [...identifies];
}

export async function saveImportedChildren(client, {
    rootRef,
    rootIdentifie,
    versionId,
    entries = [],
    sourceDependency = {},
    existingDependencies = [],
}) {
    if (!rootRef?.json) {
        throw new Error('主应用 manifest 尚未加载完成');
    }
    const existing = new Set((existingDependencies || [])
        .map(item => item?.identifie).filter(Boolean));
    const imported = (entries || []).filter(entry => entry?.identifie
        && entry.identifie !== rootIdentifie
        && !existing.has(entry.identifie));
    if (!imported.length) {
        throw new Error('没有可导入的子应用');
    }

    const rootVersion = entries.find(entry => entry?.identifie === sourceDependency.identifie)
        ?.data?.application?.version || sourceDependency.version || '';
    const dependencies = imported.map(entry => {
        const isImportedRoot = entry.identifie === sourceDependency.identifie;
        return importedChildDependency(entry, true, isImportedRoot ? {
            from: sourceDependency.from,
            version: rootVersion,
        } : {});
    });

    const previousJSON = JSON.parse(JSON.stringify(rootRef.json));
    rootRef.addImportedDependencies(dependencies);
    const rootManifest = jsyaml.dump(rootRef.json);
    try {
        await Promise.all([
            ...imported.map(entry => client.post('/respo/manifest/file', {
                identifie: rootIdentifie,
                filename: importedChildFilePath(entry.identifie),
                content: entry.manifest,
                version: versionId,
            })),
            client.post('/respo/manifest/file', {
                identifie: rootIdentifie,
                filename: 'manifest.yaml',
                content: rootManifest,
                version: versionId,
            }),
        ]);
    } catch (error) {
        await Promise.all(imported.map(entry => client.post('/respo/manifest/file', {
            identifie: rootIdentifie,
            filename: importedChildFilePath(entry.identifie),
            content: '',
            version: versionId,
        }))).catch(() => { });
        rootRef.replaceZpk(previousJSON);
        throw error;
    }
    return { imported, dependencies, rootManifest };
}

export async function removeImportedChildren(client, {
    rootRef,
    rootIdentifie,
    versionId,
    list = {},
    identifies = [],
}) {
    if (!rootRef?.json) {
        throw new Error('主应用 manifest 尚未加载完成');
    }
    const importedIdentifies = new Set((identifies || []).filter(Boolean));
    if (!importedIdentifies.size) { return { existingFiles: [], rootManifest: jsyaml.dump(rootRef.json) }; }
    const existingFiles = [...importedIdentifies]
        .map(identifie => importedChildFilePath(identifie))
        .filter(file => Object.prototype.hasOwnProperty.call(list || {}, file));
    const previousJSON = JSON.parse(JSON.stringify(rootRef.json));

    rootRef.removeImportedDependencies([...importedIdentifies]);
    const rootManifest = jsyaml.dump(rootRef.json);
    try {
        await Promise.all([
            ...existingFiles.map(filename => client.post('/respo/manifest/file', {
                identifie: rootIdentifie,
                filename,
                content: '',
                version: versionId,
            })),
            client.post('/respo/manifest/file', {
                identifie: rootIdentifie,
                filename: 'manifest.yaml',
                content: rootManifest,
                version: versionId,
            }),
        ]);
    } catch (error) {
        rootRef.replaceZpk(previousJSON);
        throw error;
    }
    return { existingFiles, rootManifest, identifies: [...importedIdentifies] };
}
