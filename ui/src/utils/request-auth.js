const OFFICIAL_ZPK_ORIGINS = new Set([
    'https://zpk.w7.cc',
    'https://api.zm.w7.com',
]);
const ZPK_AUTH_HEADERS = ['X-Zpk-Token', 'X-W7Panel-Token'];

function getRequestUrl(config = {}) {
    const url = String(config.url || '');

    if (/^https?:\/\//i.test(url)) {
        return url;
    }

    const baseURL = String(config.baseURL || '').replace(/\/+$/, '');
    const path = url.replace(/^\/+/, '');

    return baseURL ? `${baseURL}/${path}` : url;
}

export function isOfficialZpkRequest(config = {}) {
    try {
        return OFFICIAL_ZPK_ORIGINS.has(new URL(getRequestUrl(config), window.location.origin).origin);
    } catch {
        return false;
    }
}

export function removeZpkAuthHeaders(headers) {
    if (!headers) {
        return;
    }

    if (typeof headers.delete === 'function') {
        ZPK_AUTH_HEADERS.forEach(header => headers.delete(header));
        return;
    }

    Object.keys(headers).forEach(header => {
        if (ZPK_AUTH_HEADERS.some(authHeader => authHeader.toLowerCase() === header.toLowerCase())) {
            delete headers[header];
        }
    });
}
