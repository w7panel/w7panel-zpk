const trimTrailingSlash = value => String(value || '').replace(/\/+$/, '');

export function joinUrl(base, path = '') {
    const normalizedBase = trimTrailingSlash(base);
    const normalizedPath = String(path || '');

    if (!normalizedBase) {
        return normalizedPath || '';
    }

    if (!normalizedPath) {
        return normalizedBase;
    }

    return `${normalizedBase}/${normalizedPath.replace(/^\/+/, '')}`;
}

export function getWujieProps() {
    return window?.$wujie?.props || {};
}

export function getMicroAppProxyBase() {
    const props = getWujieProps();
    const frontendProps = props.frontend_props || props.frontendProps || {};
    const roleConfig = props.roleConfig || {};
    const roleFrontendProps = Object.values(roleConfig).find(item => item?.frontend_props?.url)?.frontend_props || {};

    return trimTrailingSlash(frontendProps.url || roleFrontendProps.url || props.url || '');
}

export function getZpkBaseURL() {
    const proxyBase = getMicroAppProxyBase();
    if (!proxyBase) {
        return '/zpk';
    }

    return /\/zpk(?:\/)?$/.test(proxyBase) ? proxyBase : joinUrl(proxyBase, '/zpk');
}
