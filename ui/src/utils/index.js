import axios from "axios";
import { alert, message } from './ui-feedback';
import { getPanelToken, getWujieAccessToken, getZpkToken, setZpkToken } from './panel-token';
import { getZpkBaseURL } from './request-base';
import { isOfficialZpkRequest, removeZpkAuthHeaders } from './request-auth';

const myAxios = axios.create({
    baseURL: getZpkBaseURL(),
    timeout: 90000
});

// Keep one login request in flight so a burst of expired-token responses does
// not result in a burst of login requests. The promise is cleared when the
// request settles, allowing a later expiry to refresh the token again.
let zpkLoginPromise = null;

export function loginWithWujieAccessToken() {
    const accessToken = getWujieAccessToken();
    if (!accessToken) {
        return Promise.resolve('');
    }

    if (!zpkLoginPromise) {
        zpkLoginPromise = myAxios.post('/oidc/w7panel/login', {
            access_token: accessToken
        }, {
            // A failed login must not recursively trigger another login.
            _skipZpkTokenRefresh: true,
            dontalert: true
        }).then(res => {
            const token = res.data?.data?.token || res.headers?.['x-zpk-token'] || '';
            if (!token) {
                throw new Error('登录成功但未返回 zpk-token');
            }

            setZpkToken(token);
            return token;
        }).finally(() => {
            zpkLoginPromise = null;
        });
    }

    return zpkLoginPromise;
}

function isZpkTokenError(error) {
    if (error?.response?.status !== 401) {
        return false;
    }

    const errorMessage = String(error.response?.data?.error || '').trim().toLowerCase();
    return errorMessage === 'token 错误'
        || errorMessage.includes('token 错误')
        || errorMessage.includes('token error')
        || errorMessage === '令牌错误'
        || errorMessage.includes('令牌错误')
        || /(?:token|令牌).*(?:expired|invalid|过期|失效)/i.test(errorMessage);
}

function isZpkLoginRequest(config) {
    return /(?:^|\/)oidc\/w7panel\/login(?:$|\?)/.test(String(config?.url || ''));
}

function rejectResponseError(error) {
    if (error?.response?.status == 422) {
        let errorinfo = error.response?.data?.errors;
        if (!errorinfo) { return; }
        let keys = Object.keys(errorinfo);
        let messages = keys.map(key => {
            return errorinfo[key].join('\n');
        });

        alert({ title: "提示", content: messages.join('<br/>'), confirmButtonText: "确定", dangerouslyUseHTMLString: true });

        return Promise.reject(error);
    }

    if (error?.response?.status == 429) { return; }
    if (error?.response?.status == 408) { return; }
    if (!error?.config?.dontalert) {
        if (error.response && !error.config?.headers?.cancelerror && error.response?.data?.error) {
            message({
                message: error.response.data.error,
                duration: 3000,
                type: 'error',
            });
        }
    }

    return Promise.reject(error);
}

function setRequestHeader(headers, name, value) {
    if (typeof headers?.set === 'function') {
        headers.set(name, value);
        return;
    }

    headers[name] = value;
}

myAxios.interceptors.request.use(config => {
    if (isOfficialZpkRequest(config) || config._skipZpkTokenRefresh || isZpkLoginRequest(config)) {
        removeZpkAuthHeaders(config.headers);
        return config;
    }

    config.headers = config.headers || {};
    setRequestHeader(config.headers, 'X-Zpk-Token', getZpkToken());
    setRequestHeader(config.headers, 'X-W7Panel-Token', getPanelToken());
    return config
}, err => Promise.reject(err))

myAxios.interceptors.response.use(res => {
    setZpkToken(res.headers?.['x-zpk-token']);
    if (res.status >= 200 && res.status < 300 && res) {
        return Promise.resolve(res)
    }
}, error => {
    const requestConfig = error?.config;
    if (isZpkTokenError(error)
        && requestConfig
        && !requestConfig._zpkTokenRetried
        && !requestConfig._skipZpkTokenRefresh
        && !isZpkLoginRequest(requestConfig)
        && getWujieAccessToken()) {
        requestConfig._zpkTokenRetried = true;

        return loginWithWujieAccessToken()
            // Handle only a refresh-login failure here. Errors from the
            // replayed request are handled by the interceptor on that request
            // and must not be replaced with the original 401 error.
            .then(() => myAxios.request(requestConfig), () => rejectResponseError(error));
    }

    return rejectResponseError(error)
});

export default myAxios;
