import axios from "axios";
import { ElMessage } from 'element-plus';

const myAxios = axios.create({
    baseURL: '',
    timeout: 90000
});

myAxios.interceptors.request.use(config => {
    config.headers['X-W7Panel-Token'] = window?.$wujie?.props?.paneltoken || localStorage.getItem('X-W7Panel-Token') || "";
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
        ElMessage({
            message: error.response.data.error,
            duration: 3000,
            type: 'error',
        });
    }

    return Promise.reject(error)
});

export default myAxios;
