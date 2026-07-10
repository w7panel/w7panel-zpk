export function getWujieAccessToken() {
    return window?.$wujie?.props?.access_token || window?.wujie?.props?.access_token || '';
}

export function getZpkToken() {
    return localStorage.getItem('X-Zpk-Token') || '';
}

export function setZpkToken(token) {
    if (!token) {
        return;
    }
    localStorage.setItem('X-Zpk-Token', token);
}
