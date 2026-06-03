import axios from "axios";
import { alert, message } from './ui-feedback';
import { getPanelToken } from './panel-token';

const myAxios = axios.create({
    baseURL: process.env.NODE_ENV === 'production' ? (window?.$wujie?.props?.url ? window?.$wujie?.props?.url + '/zpk' : '/zpk') : '/zpk',
    timeout: 90000
});

myAxios.interceptors.request.use(config => {
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

    if (error?.response?.status == 422) {
        let errorinfo = error.response.data.errors;
        if (!errorinfo) { return }
        let keys = Object.keys(errorinfo);
        let messages = keys.map(key => {
            return errorinfo[key].join('\n');
        });

        alert({ title: "提示", content: messages.join('<br/>'), confirmButtonText: "确定", dangerouslyUseHTMLString: true });

        return Promise.reject(error);
    }

    if (error?.response?.status == 429) { return }
    if (error?.response?.status == 408) { return }
    if (!error?.config?.dontalert) {
        if (error.response && !error.config.headers.cancelerror && error.response?.data?.error) {
            message({
                message: error.response.data.error,
                duration: 3000,
                type: 'error',
            });
        }
    }

    return Promise.reject(error)
});

export default myAxios;
