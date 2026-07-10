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

export function getPanelToken() {
    const wujieToken = window?.$wujie?.props?.paneltoken || '';
    if (wujieToken) {
        return wujieToken;
    }

    return localStorage.getItem('X-W7Panel-Token') || '';
}