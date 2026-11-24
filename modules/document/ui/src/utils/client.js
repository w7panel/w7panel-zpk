import axios from 'axios';

function documentBaseURL() {
    const props = window.$wujie?.props || {};
    const frontend = props.frontend_props || props.frontendProps || {};
    const roles = Object.values(props.roleConfig || {});
    const base = String(
        frontend.document_url ||
            frontend.url ||
            roles.find((role) => role?.frontend_props?.url)?.frontend_props.url ||
            props.url ||
            ''
    ).replace(/\/+$/, '');
    if (!base) return '/zpk/document';
    if (/\/document$/.test(base)) return base;
    return /\/zpk$/.test(base) ? base + '/document' : base + '/zpk/document';
}
function createDocumentClient(request, baseURL) {
    async function post(path, params = {}) {
        const response = await request({ method: 'POST', url: path, data: params });
        if (!response || response.status !== 200 || response.data?.error)
            throw new Error(response?.data?.error || '文档请求未成功');
        return response.data?.data;
    }
    return {
        post,
        downloadURL(reference) {
            return new URL(baseURL.replace(/\/$/, '') + '/download?ref=' + encodeURIComponent(reference), location.origin)
                .href;
        },
    };
}

// An independent login session for the document frontend.
export function createStandaloneClient(baseURL = documentBaseURL()) {
    const transport = axios.create({ baseURL, timeout: 30000 });
    const storageKey = 'document-token:' + new URL(baseURL, location.origin).href;
    let loginPromise = null;
    const accessToken = () => window.$wujie?.props?.access_token || window.wujie?.props?.access_token || '';
    async function login(token = accessToken()) {
        if (!token) throw new Error('请提供面板登录凭据');
        if (!loginPromise)
            loginPromise = transport
                .post('/login', { access_token: token })
                .then((response) => {
                    const value = response.data?.data?.token;
                    if (!value) throw new Error('登录未返回凭据');
                    localStorage.setItem(storageKey, value);
                })
                .finally(() => {
                    loginPromise = null;
                });
        await loginPromise;
    }
    async function request(options) {
        try {
            return await transport({
                ...options,
                headers: { ...options.headers, 'X-Document-Token': localStorage.getItem(storageKey) || '' },
            });
        } catch (error) {
            if (error.response?.status === 401 && !options.retried && accessToken()) {
                await login();
                return request({ ...options, retried: true });
            }
            throw new Error(error.response?.data?.error || error.message || '文档请求失败');
        }
    }
    return {
        ...createDocumentClient(request, baseURL),
        async start() {
            if (accessToken()) await login();
            return this.post('/user');
        },
    };
}
