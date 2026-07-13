import axios from "axios";
import { message } from './ui-feedback';
import { getPanelToken } from './panel-token';
import { isOfficialZpkRequest, removeZpkAuthHeaders } from './request-auth';

const myAxios = axios.create({
    baseURL: '',
    timeout: 90000
});

myAxios.interceptors.request.use(config => {
    if (isOfficialZpkRequest(config)) {
        removeZpkAuthHeaders(config.headers);
        return config;
    }

    config.headers['X-W7Panel-Token'] = getPanelToken();
    return config
}, err => {
    Promise.reject(err)
})

myAxios.interceptors.response.use(res => {
    if (res.status >= 200 && res.status < 300 && res) {
        return Promise.resolve(res)
    }
}, error => {
    if (error.response && !error.config.headers.cancelerror && error.response?.data?.error) {
        message({
            message: error.response.data.error,
            duration: 3000,
            type: 'error',
        });
    }

    return Promise.reject(error)
});

export default myAxios;
