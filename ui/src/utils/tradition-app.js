export const traditionInstallTypes = Object.freeze({
    site: 'site',
    extension: 'extension',
});

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
