import axios from "axios";
import { messageError } from './ui-feedback'
import { getZpkToken } from './panel-token';
import { isOfficialZpkRequest, removeZpkAuthHeaders } from './request-auth';

const myAxios = axios.create({
    baseURL: '/zpk',
    timeout: 9000
});


myAxios.interceptors.request.use(async config => {
    if (isOfficialZpkRequest(config)) {
        removeZpkAuthHeaders(config.headers);
        return config;
    }

    config.headers['X-Zpk-Token'] = getZpkToken();
    return config
}, err => {
    Promise.reject(err)
})

myAxios.interceptors.response.use(res => {
    if (res.status >= 200 && res.status < 300 && res) {
        return Promise.resolve(res)
    }
}, error => {
    if (error?.response?.status == 422) {
        let errorinfo = error.response.data.errors;
        if (!errorinfo) { return }
        return Promise.reject(error);
    }

    if (error?.response?.status == 429) { return }
    if (error?.response?.status == 408) { return }
    if (!error?.config?.dontalert) {
        if (error.response && !error.config.headers.cancelerror && error?.response?.data?.error) {
            messageError(error.response.data.error)
        }
    }

    return Promise.reject(error)
});

export default myAxios;
