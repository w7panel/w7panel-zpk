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

export function normalizeChildImportList(list = [], repositoryBaseURL = '') {
    const sourceRepositoryURL = String(repositoryBaseURL || '').trim().replace(/\/+$/, '');
    return (Array.isArray(list) ? list : [])
        .filter(item => item?.identifie && item?.install_only_once)
        .map(item => ({
            ...item,
            name: item.name || item.identifie,
            sourceRepositoryURL,
        }));
}

export function fetchChildImportList(client, tab, params = {}) {
    return client.get(childImportRepositoryURL(tab), {
        params,
        dontalert: true,
    }).then(response => {
        const data = response?.data?.data || {};
        // The request base may point at the panel's micro-app proxy. Persist
        // the repository's public URL returned by the repository itself.
        const repositoryBaseURL = data.webUrl || childImportRepositoryBaseURL(tab);
        return normalizeChildImportList(data.list || [], repositoryBaseURL);
    });
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

function clonePersistedRootManifest(rootRef) {
    try {
        if (rootRef?.getSavedManifest) return rootRef.getSavedManifest();
        if (typeof rootRef?.data == 'string') return jsyaml.load(rootRef.data) || {};
        if (rootRef?.data) return JSON.parse(JSON.stringify(rootRef.data));
    } catch {
        // Fall back to the editor state when the original manifest cannot be parsed.
    }
    return JSON.parse(JSON.stringify(rootRef?.json || {}));
}

export function getImportedChildIdentifies(rootIdentifie, list = {}, dependencies = []) {
    const identifies = new Set();
    const dependenciesByIdentifie = new Map((dependencies || [])
        .filter(item => item?.identifie)
        .map(item => [item.identifie, item]));
    const visit = (identifie) => {
        if (!identifie || identifies.has(identifie)) { return; }
        identifies.add(identifie);
        const file = importedChildFilePath(identifie);
        const raw = list?.[file] || dependenciesByIdentifie.get(identifie)?.manifest || '';
        try {
            const manifest = typeof raw == 'string' ? (jsyaml.load(raw) || {}) : raw;
            (manifest?.platform?.depends || [])
                .filter(item => item?.type !== 'out')
                .forEach(item => visit(item?.identifie));
        } catch {
            // Ignore malformed child manifests while resolving the imported tree.
        }
    };
    visit(rootIdentifie);
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

    const previousJSON = clonePersistedRootManifest(rootRef);
    if (replaceExisting) {
        rootRef.replaceImportedDependencies(dependencies);
    } else {
        rootRef.addImportedDependencies(dependencies);
    }
    const rootJSON = JSON.parse(JSON.stringify(previousJSON));
    rootJSON.platform = rootJSON.platform || {};
    rootJSON.platform.depends = JSON.parse(JSON.stringify(rootRef.json?.platform?.depends || []));
    const childOrders = new Map((rootJSON.platform.depends || [])
        .filter(item => item?.identifie && item?.type !== 'out')
        .map((item, index) => [item.identifie, index + 1]));
    imported.forEach((entry, index) => {
        entry.data = entry.data || {};
        entry.data.application = entry.data.application || {};
        entry.data.application.order = childOrders.get(entry.identifie) || (index + 1);
        entry.manifest = jsyaml.dump(entry.data);
    });
    const rootManifest = jsyaml.dump(rootJSON);
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
    orderedIdentifies = [],
}) {
    if (!rootRef?.json) {
        throw new Error('主应用 manifest 尚未加载完成');
    }
    const importedIdentifies = new Set((identifies || []).filter(Boolean));
    if (!importedIdentifies.size) {
        return { existingFiles: [], orderedFiles: {}, rootManifest: jsyaml.dump(clonePersistedRootManifest(rootRef)) };
    }
    const existingFiles = [...importedIdentifies]
        .map(identifie => importedChildFilePath(identifie))
        .filter(file => Object.prototype.hasOwnProperty.call(list || {}, file));
    const previousJSON = clonePersistedRootManifest(rootRef);

    if (rootRef.removeChildDependencies) {
        rootRef.removeChildDependencies([...importedIdentifies]);
    } else {
        rootRef.removeImportedDependencies([...importedIdentifies]);
    }
    const childOrder = new Map((orderedIdentifies || [])
        .filter(identifie => identifie && !importedIdentifies.has(identifie))
        .map((identifie, index) => [identifie, index]));
    const dependencies = rootRef.json?.platform?.depends || [];
    const childDependencies = dependencies
        .map((dependency, index) => ({ dependency, index }))
        .filter(item => item.dependency?.type !== 'out')
        .sort((first, second) => {
            const firstOrder = childOrder.get(first.dependency?.identifie);
            const secondOrder = childOrder.get(second.dependency?.identifie);
            if (firstOrder !== undefined && secondOrder !== undefined) return firstOrder - secondOrder;
            if (firstOrder !== undefined || secondOrder !== undefined) return firstOrder !== undefined ? -1 : 1;
            return first.index - second.index;
        })
        .map(item => item.dependency);
    const orderedDependencies = childDependencies.concat(
        dependencies.filter(item => item?.type === 'out'),
    );
    rootRef.json.platform.depends = orderedDependencies;
    const rootJSON = JSON.parse(JSON.stringify(previousJSON));
    rootJSON.platform = rootJSON.platform || {};
    rootJSON.platform.depends = JSON.parse(JSON.stringify(orderedDependencies));

    const orderedFiles = {};
    (rootJSON.platform.depends || [])
        .filter(item => item?.identifie && item?.type !== 'out')
        .forEach((item, index) => {
            const filename = importedChildFilePath(item.identifie);
            const raw = list?.[filename];
            if (raw === undefined) return;
            try {
                const manifest = typeof raw == 'string'
                    ? (jsyaml.load(raw) || {})
                    : JSON.parse(JSON.stringify(raw));
                if (!manifest.application) return;
                manifest.application.order = index + 1;
                orderedFiles[filename] = jsyaml.dump(manifest);
            } catch {
                // Keep deletion available even when an unrelated child manifest is malformed.
            }
        });

    const rootManifest = jsyaml.dump(rootJSON);
    const previousFiles = Object.fromEntries(
        [...new Set(existingFiles.concat(Object.keys(orderedFiles)))]
            .map(filename => [filename, list?.[filename] ?? '']),
    );
    try {
        await client.post('/respo/manifest/file', {
            identifie: rootIdentifie,
            filename: 'manifest.yaml',
            content: rootManifest,
            version: versionId,
        });
        const fileWrites = await Promise.allSettled([
            ...Object.entries(orderedFiles).map(([filename, content]) => client.post('/respo/manifest/file', {
                identifie: rootIdentifie,
                filename,
                content,
                version: versionId,
            })),
            ...existingFiles.map(filename => client.post('/respo/manifest/file', {
                identifie: rootIdentifie,
                filename,
                content: '',
                version: versionId,
            })),
        ]);
        const failedWrite = fileWrites.find(result => result.status === 'rejected');
        if (failedWrite) throw failedWrite.reason;
    } catch (error) {
        await Promise.allSettled([
            client.post('/respo/manifest/file', {
                identifie: rootIdentifie,
                filename: 'manifest.yaml',
                content: jsyaml.dump(previousJSON),
                version: versionId,
            }),
            ...Object.entries(previousFiles).map(([filename, content]) => client.post('/respo/manifest/file', {
                identifie: rootIdentifie,
                filename,
                content,
                version: versionId,
            })),
        ]);
        rootRef.replaceZpk(previousJSON);
        throw error;
    }
    return {
        existingFiles,
        orderedFiles,
        rootManifest,
        identifies: [...importedIdentifies],
    };
}
