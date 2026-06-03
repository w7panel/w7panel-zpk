import { createApp } from 'vue'
import App from './App.vue'
import ArcoVue from '@arco-design/web-vue'
import '@arco-design/web-vue/dist/arco.css'

import './assets/css/style.css'
import routes from './router'

import 'highlight.js/styles/atom-one-dark.css'
import hljs from 'highlight.js/lib/common'

import mavonEditor from 'mavon-editor'
import 'mavon-editor/dist/css/index.css'

if (window.__POWERED_BY_WUJIE__) {

	window.__webpack_public_path__ = window.__WUJIE_PUBLIC_PATH__;
}

window.hljs = hljs;

if (process.env.NODE_ENV === 'development') {
	const devPanelToken = process.env.VUE_APP_PANEL_TOKEN || process.env.VUE_APP_TOKEN;
	if (devPanelToken) {
		localStorage.setItem('X-W7Panel-Token', devPanelToken);
	}
}

const app = createApp(App);

const installPlugins = () => {
	app.use(ArcoVue);
	app.use(routes);
	app.use(mavonEditor);
};

if (window.__POWERED_BY_WUJIE__) {
	document.body.style.minHeight = 'calc(-146px + 100vh)';
	window.__WUJIE_MOUNT = () => {
		installPlugins();
		app.mount('#zpk');
	};
	window.__WUJIE_UNMOUNT = () => {
		app.unmount();
	};
} else {
	installPlugins();
	app.mount('#zpk');
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
