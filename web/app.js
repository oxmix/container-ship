import {createApp} from 'vue';
import {createRouter, createWebHistory} from 'vue-router';
import App from '@/App.vue'

const pluginFetch = {
	install: (app) => {
		app.config.globalProperties.$fetch = (url, params) => {
			params = params || {};

			if (sessionStorage.getItem('token')) {
				params.headers.Authorization = 'Bearer ' + sessionStorage.getItem('token');
			}

			if ('method' in params && params.method !== 'GET') {
				params.headers['Content-Type'] = 'application/json';
				params.body = JSON.stringify(params.data);
				delete params.data;
			}

			return fetch(url, params)
				.then(r => r.json())
		}
	}
};

createApp(App)
	.use(createRouter({
		history: createWebHistory(),
		routes: [
			{
				path: '',
				name: 'States',
				component: () => import(/* webpackChunkName: "states" */ '@/States'),
				alias: ['/', '/o/states']
			}, {
				path: '/logs/:node/:container',
				name: 'Logs',
				component: () => import(/* webpackChunkName: "states" */ '@/Logs'),
			}, {
				path: '/o/nodes',
				name: 'Nodes',
				component: () => import(/* webpackChunkName: "nodes" */ '@/Nodes'),
			}, {
				path: '/o/hub',
				name: 'Hub',
				component: () => import(/* webpackChunkName: "hub" */ '@/Hub'),
			}
		]
	}))
	.use(pluginFetch)
	.mount('#app');