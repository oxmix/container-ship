import app from '../../app';
import component from "./AlertConfirm";
import {createVNode, render} from 'vue'

export function Alert(title, objects) {
	if (!title) {
		return
	}
	const container = document.createElement('div')
	document.body.append(container)

	const cvm = createVNode(component, {
		container, title, objects, escapeOk: true
	})
	cvm.appContext = app.$.appContext
	render(cvm, container)

	return new Promise(resolve => {
		cvm.component.proxy.actOk = resolve
	})
}

export function Confirm(title, objects) {
	const container = document.createElement('div')
	document.body.append(container)

	const cvm = createVNode(component, {
		container, title, objects
	})
	cvm.appContext = app.$.appContext
	render(cvm, container)

	return new Promise((resolve, reject) => {
		cvm.component.proxy.actOk = resolve
		cvm.component.proxy.actCancel = reject
	})
}

export function Delete(objects) {
	const container = document.createElement('div')
	document.body.append(container)

	const cvm = createVNode(component, {
		container, objects
	})
	cvm.appContext = app.$.appContext
	render(cvm, container)

	return new Promise((resolve, reject) => {
		cvm.component.proxy.actDel = resolve
		cvm.component.proxy.actCancel = reject
	})
}
