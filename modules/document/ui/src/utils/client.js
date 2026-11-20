import axios from 'axios';

export function documentBaseURL() {
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
export function createDocumentClient(request, baseURL = documentBaseURL()) {
    async function post(path, params = {}, options = {}) {
        const response = await request({ method: 'POST', url: path, data: params, ...options });
        if (!response || response.status !== 200 || response.data?.error)
            throw new Error(response?.data?.error || '文档请求未成功');
        return response.data?.data;
    }
    return {
        post,
        linkURL(token) {
            return new URL(baseURL.replace(/\/$/, '') + '/download?token=' + encodeURIComponent(token), location.origin)
                .href;
        },
        async download(ref, format) {
            const response = await request({
                method: 'POST',
                url: '/download',
                data: { ref, format },
                responseType: 'blob',
                timeout: 300000,
            });
            if (!response || response.status !== 200) throw new Error('下载失败');
            const url = URL.createObjectURL(response.data);
            const link = document.createElement('a');
            link.href = url;
            link.download = ref.replace(/\//g, '-') + (format === 'markdown' ? '.md' : '.zip');
            document.body.appendChild(link);
            link.click();
            link.remove();
            setTimeout(() => URL.revokeObjectURL(url), 1000);
        },
    };
}

// An independent login session for the document frontend. A host can instead inject its authenticated transport.
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
        login,
        async start() {
            if (accessToken()) await login();
            return this.post('/user');
        },
    };
}
