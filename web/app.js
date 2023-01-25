import {createApp} from 'vue';
import {createRouter, createWebHistory} from 'vue-router';
import App from '@/App.vue'

const pluginFetch = {
	position: null,
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
				.finally(() => {
					if (pluginFetch.position?.top > 0) {
						window.setTimeout(() => {
							window.scrollTo(pluginFetch.position.left, pluginFetch.position.top);
							pluginFetch.position = null;
						}, 1);
					}
				})
		}
	}
};

const router = createRouter({
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
	],
	scrollBehavior(to, from, savedPosition) {
		return pluginFetch.position = savedPosition || {top: 0, left: 0};
	},
});

createApp(App)
	.use(router)
	.use(pluginFetch)
	.mount('#app');