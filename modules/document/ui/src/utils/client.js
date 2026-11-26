import axios from 'axios';

import { getDocumentBaseURL, getMicroAppProxyBase, isBaseRequest } from './request-base';
import { getPanelToken, getWujieAccessToken } from './panel-token';

function createDocumentClient(request) {
    async function post(path, params = {}) {
        const response = await request({ method: 'POST', url: path, data: params });
        if (!response || response.status !== 200 || response.data?.error)
            throw new Error(response?.data?.error || '文档请求未成功');
        return response.data?.data;
    }
    return {
        post,
    };
}

// An independent login session for the document frontend.
export function createStandaloneClient(baseURL = getDocumentBaseURL()) {
    const transport = axios.create({ baseURL, timeout: 30000 });
    const proxyBase = getMicroAppProxyBase();
    transport.interceptors.request.use((config) => {
        if (isBaseRequest(transport.getUri(config), proxyBase)) {
            config.headers = config.headers || {};
            config.headers['X-W7Panel-Token'] = getPanelToken();
        }
        return config;
    });
    const storageKey = 'document-token:' + new URL(baseURL, location.origin).href;
    let loginPromise = null;
    async function login(token = getWujieAccessToken()) {
        if (!token) throw new Error('请提供面板登录凭据');
        if (!loginPromise)
            loginPromise = transport
                .post('/login', { access_token: token })
                .then((response) => {
                    if (response.status !== 200 || response.data?.error)
                        throw new Error(response.data?.error || '文档登录未成功');
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
                headers: {
                    ...options.headers,
                    ...(isBaseRequest(transport.getUri(options), baseURL)
                        ? { 'X-Document-Token': localStorage.getItem(storageKey) || '' }
                        : {}),
                },
            });
        } catch (error) {
            if (error.response?.status === 401 && error.response?.data?.error === 'token 错误'
                && isBaseRequest(transport.getUri(options), baseURL) && !options.retried && getWujieAccessToken()) {
                await login();
                return request({ ...options, retried: true });
            }
            throw new Error(error.response?.data?.error || error.message || '文档请求失败');
        }
    }
    return {
        ...createDocumentClient(request),
        async start() {
            if (getWujieAccessToken()) await login();
            return this.post('/user');
        },
    };
}
