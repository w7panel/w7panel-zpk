export function getPanelToken() {
    const wujieToken = window?.$wujie?.props?.paneltoken || '';
    if (wujieToken) {
        return wujieToken;
    }

    return localStorage.getItem('X-W7Panel-Token') || '';
}
