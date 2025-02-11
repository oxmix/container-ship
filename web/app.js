import {createApp} from 'vue';
import {createRouter, createWebHistory} from 'vue-router';
import App from '@/App.vue'

const pluginFetch = {
	position: null,
	install: app => {
		const func = (url, params) => {
			params = params || {}

			if (!params.headers) {
				params.headers = {}
			}

			if ('method' in params && params.method !== 'GET') {
				params.headers['Content-Type'] = 'application/json'
				params.body = JSON.stringify(params.data)
				delete params.data
			}

			return fetch(url, params)
				.then(r => r.json())
				.finally(() => {
					if (pluginFetch.position?.top > 0) {
						window.setTimeout(() => {
							window.scrollTo(pluginFetch.position.left, pluginFetch.position.top)
							pluginFetch.position = null
						}, 1)
					}
				})
		}

		// for options api
		app.config.globalProperties.$fetch = func

		// for composition api
		app.provide('fetch', func)
	}
}

const router = createRouter({
	history: createWebHistory(),
	routes: [
		{
			path: '',
			component: () => import(/* webpackChunkName: "states" */ '@/PageStates'),
			alias: ['/', '/states']
		}, {
			path: '/nodes',
			component: () => import(/* webpackChunkName: "nodes" */ '@/PageNodes'),
		}, {
			path: '/manifests',
			component: () => import(/* webpackChunkName: "manifests" */ '@/PageManifests'),
		}, {
			path: '/variables',
			component: () => import(/* webpackChunkName: "variables" */ '@/PageVariables'),
		}, {
			path: '/logs/:node/:container',
			component: () => import(/* webpackChunkName: "logs" */ '@/PageLogs'),
		}
	],
	scrollBehavior(to, from, savedPosition) {
		return pluginFetch.position = savedPosition || {top: 0, left: 0}
	},
})

const app = createApp(App)
	.use(router)
	.use(pluginFetch)
	.mount('#app')

export default app