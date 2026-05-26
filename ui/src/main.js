import { createApp } from 'vue'
import App from './App.vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import locale from 'element-plus/dist/locale/zh-cn.mjs'

import './assets/css/style.css'
import routes from './router'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

import 'highlight.js/styles/atom-one-dark.css'
import hljs from 'highlight.js/lib/common'

import mavonEditor from 'mavon-editor'
import 'mavon-editor/dist/css/index.css'

if (window.__POWERED_BY_WUJIE__) {

	window.__webpack_public_path__ = window.__WUJIE_PUBLIC_PATH__;
}

window.hljs = hljs;

const app = createApp(App);

if (window.__POWERED_BY_WUJIE__) {
	document.body.style.minHeight = 'calc(-146px + 100vh)';
	window.__WUJIE_MOUNT = () => {
		app.use(ElementPlus, { locale });
		app.use(routes);
		app.use(mavonEditor);
		app.mount('#zpk');
	};
	window.__WUJIE_UNMOUNT = () => {
		app.unmount();
	};
} else {
	app.use(ElementPlus, { locale });
	app.use(routes);
	app.use(mavonEditor);
	app.mount('#zpk');
}




for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
	app.component(key, component)
}

const debounce = (fn, delay) => {
	let timer = null;
	return function () {
		let context = this;
		let args = arguments;
		clearTimeout(timer);
		timer = setTimeout(function () {
			fn.apply(context, args);
		}, delay);
	};
};
const _ResizeObserver = window.ResizeObserver;
window.ResizeObserver = class ResizeObserver extends _ResizeObserver {
	constructor(callback) {
		callback = debounce(callback, 16);
		super(callback);
	}
};
