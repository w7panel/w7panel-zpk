export function getWujieAccessToken() {
    return window?.$wujie?.props?.accesstoken || window?.wujie?.props?.accesstoken || '';
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
