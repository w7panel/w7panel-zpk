import { createApp } from 'vue';
import ArcoVue from '@arco-design/web-vue';
import '@arco-design/web-vue/dist/arco.css';
import mavonEditor from 'mavon-editor';
import 'mavon-editor/dist/css/index.css';
import App from './App.vue';
import router from './router';
const app = createApp(App);
app.use(ArcoVue);
app.use(router);
app.use(mavonEditor);
if (window.__POWERED_BY_WUJIE__) {
    window.__WUJIE_MOUNT = () => app.mount('#document');
    window.__WUJIE_UNMOUNT = () => app.unmount();
} else {
    app.mount('#document');
}
