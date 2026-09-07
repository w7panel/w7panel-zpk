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

export function childImportRepositoryURLFromSource(source) {
    const repository = String(source || '').trim().replace(/\/+$/, '');
    const localRepository = childImportRepositoryBaseURL('local').replace(/\/+$/, '');
    if (!repository || repository === localRepository) {
        return childImportRepositoryURL('local');
    }
    try {
        const url = new URL(repository);
        url.hash = '';
        url.search = '';
        let path = url.pathname.replace(/\/+$/, '');
        if (/\/respo\/(?:v2\/)?info(?:\/.*)?$/i.test(path)) {
            path = path.replace(/\/respo\/(?:v2\/)?info(?:\/.*)?$/i, '/respo/list');
        } else if (/\/zpk$/i.test(path)) {
            path += '/respo/list';
        } else {
            path += '/zpk/respo/list';
        }
        url.pathname = path;
        url.searchParams.append('status', '2');
        url.searchParams.append('status', '99');
        return url.toString();
    } catch {
        return '';
    }
}

export function fetchChildImportListFromSource(client, source, params = {}) {
    const url = childImportRepositoryURLFromSource(source);
    if (!url) {
        return Promise.reject(new Error('子应用来源地址无效'));
    }
    return client.get(url, {
        params,
        dontalert: true,
        _skipZpkAuth: true,
    }).then(response => (Array.isArray(response?.data?.data?.list)
        ? response.data.data.list
        : []).filter(item => item?.identifie));
}

function parseChildImportVersion(value) {
    const match = String(value || '').trim().match(
        /^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?(?:\+[0-9A-Za-z.-]+)?$/,
    );
    if (!match) return null;
    return {
        core: match.slice(1, 4).map(Number),
        prerelease: match[4] ? match[4].split('.') : [],
    };
}

export function isChildImportVersionNewer(latest, current) {
    const next = parseChildImportVersion(latest);
    const now = parseChildImportVersion(current);
    if (!next || !now) return false;
    for (let index = 0; index < next.core.length; index++) {
        if (next.core[index] !== now.core[index]) {
            return next.core[index] > now.core[index];
        }
    }
    if (!next.prerelease.length || !now.prerelease.length) {
        return now.prerelease.length > next.prerelease.length;
    }
    const length = Math.max(next.prerelease.length, now.prerelease.length);
    for (let index = 0; index < length; index++) {
        const nextPart = next.prerelease[index];
        const nowPart = now.prerelease[index];
        if (nextPart === undefined || nowPart === undefined) {
            return nowPart === undefined;
        }
        if (nextPart === nowPart) continue;
        const nextNumber = /^\d+$/.test(nextPart) ? Number(nextPart) : null;
        const nowNumber = /^\d+$/.test(nowPart) ? Number(nowPart) : null;
        if (nextNumber !== null && nowNumber !== null) return nextNumber > nowNumber;
        if (nextNumber !== null || nowNumber !== null) return nextNumber === null;
        return nextPart > nowPart;
    }
    return false;
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
    existingManifests = {},
    replaceExisting = false,
}) {
    if (!rootRef?.json) {
        throw new Error('主应用 manifest 尚未加载完成');
    }
    const existing = new Set((existingDependencies || [])
        .map(item => item?.identifie).filter(Boolean));
    const imported = (entries || []).filter(entry => entry?.identifie
        && entry.identifie !== rootIdentifie
        && (replaceExisting || !existing.has(entry.identifie)));
    if (!imported.length) {
        throw new Error('没有可导入的子应用');
    }

    const rootVersion = entries.find(entry => entry?.identifie === sourceDependency.identifie)
        ?.data?.application?.version || sourceDependency.version || '';
    const existingByIdentifie = new Map((existingDependencies || [])
        .filter(item => item?.identifie)
        .map(item => [item.identifie, item]));
    const dependencies = imported.map(entry => {
        const isImportedRoot = entry.identifie === sourceDependency.identifie;
        const existingDependency = existingByIdentifie.get(entry.identifie);
        const required = existingDependency?.required ?? sourceDependency.required ?? true;
        return importedChildDependency(entry, required, isImportedRoot ? {
            from: sourceDependency.from,
            version: rootVersion,
        } : {});
    });

    const previousJSON = JSON.parse(JSON.stringify(rootRef.json));
    if (replaceExisting) {
        rootRef.replaceImportedDependencies(dependencies);
    } else {
        rootRef.addImportedDependencies(dependencies);
    }
    const rootManifest = jsyaml.dump(rootRef.json);
    try {
        const childWrites = await Promise.allSettled(imported.map(entry =>
            client.post('/respo/manifest/file', {
                identifie: rootIdentifie,
                filename: importedChildFilePath(entry.identifie),
                content: entry.manifest,
                version: versionId,
            })));
        const failedWrite = childWrites.find(result => result.status === 'rejected');
        if (failedWrite) {
            throw failedWrite.reason;
        }
        await client.post('/respo/manifest/file', {
            identifie: rootIdentifie,
            filename: 'manifest.yaml',
            content: rootManifest,
            version: versionId,
        });
    } catch (error) {
        await Promise.allSettled([
            ...imported.map(entry => client.post('/respo/manifest/file', {
                identifie: rootIdentifie,
                filename: importedChildFilePath(entry.identifie),
                content: replaceExisting
                    ? (existingManifests[importedChildFilePath(entry.identifie)] || '')
                    : '',
                version: versionId,
            })),
            client.post('/respo/manifest/file', {
                identifie: rootIdentifie,
                filename: 'manifest.yaml',
                content: jsyaml.dump(previousJSON),
                version: versionId,
            }),
        ]);
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
