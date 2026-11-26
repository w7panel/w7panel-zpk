const trimTrailingSlash = (value) => String(value || '').replace(/\/+$/, '');

export function getMicroAppProxyBase() {
    const props = window.$wujie?.props || {};
    const frontend = props.frontend_props || props.frontendProps || {};
    const roles = Object.values(props.roleConfig || {});
    return trimTrailingSlash(frontend.url || roles.find((role) => role?.frontend_props?.url)?.frontend_props.url || props.url);
}

export function getDocumentBaseURL() {
    const props = window.$wujie?.props || {};
    const frontend = props.frontend_props || props.frontendProps || {};
    const base = trimTrailingSlash(frontend.document_url || getMicroAppProxyBase());
    if (!base) return '/zpk/document';
    if (/\/document$/.test(base)) return base;
    return /\/zpk$/.test(base) ? base + '/document' : base + '/zpk/document';
}

export function isBaseRequest(url, baseURL) {
    if (!baseURL) return false;
    try {
        const target = new URL(url, window.location.origin);
        const base = new URL(baseURL, window.location.origin);
        const basePath = base.pathname.replace(/\/+$/, '');
        return target.origin === base.origin && (target.pathname === basePath || target.pathname.startsWith(basePath + '/'));
    } catch {
        return false;
    }
}
