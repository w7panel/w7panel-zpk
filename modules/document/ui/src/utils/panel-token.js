export function getWujieAccessToken() {
    return window.$wujie?.props?.access_token || window.wujie?.props?.access_token || '';
}

export function getPanelToken() {
    return window.$wujie?.props?.paneltoken || localStorage.getItem('X-W7Panel-Token') || '';
}
