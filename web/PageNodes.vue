<script setup>
import {inject, onMounted, onUnmounted, ref} from "vue";
import PopupModal from "~/components/PopupModal.vue";
import CopyObj from "~/components/CopyObj.vue";
import {Alert, Confirm, Delete} from "~/components/alert/alert";

const fetch = inject('fetch')
const showNewNode = ref(false)
let keyConnect = ''
const nodes = ref(null)
let updater = null

const showAddDeployment = ref(false)
const nameNodeToDeploy = ref('')
const nameDeployment = ref('')
const manifestsList = ref([])
const suggestOpen = ref(false)
const suggestIndex = ref(-1)

onMounted(() => {
  refresh()
  updater = setInterval(() => refresh(), 3000);
})

onUnmounted(() => clearInterval(updater))

function refresh() {
  fetch('/internal/nodes').then(r => {
    if (!r.ok) {
      return
    }
    let list = []
    r.data?.forEach(e => {
      let deps = {}
      e.deployments.forEach(d => {
        let s = d.split('.')
        if (s.length > 1) {
          if (!deps[s[0]]) {
            deps[s[0]] = []
          }
          deps[s[0]].push(s.slice(1).join('.'))
        } else {
          if (!deps['!without']) {
            deps['!without'] = []
          }
          deps['!without'].push(s[0])
        }
      })
      e.deployments = deps
      list.push(e)
    })
    nodes.value = list
  })
}

function openNewNode() {
  fetch('/internal/nodes/connect').then(r => {
    if (!r.ok) {
      return Alert(r.message)
    }
    keyConnect = r.data.key
    showNewNode.value = !showNewNode.value
  })
}

function connectCommand() {
  return 'curl -s' + (document.location.port === '8443' ? 'k' : '')
    + ' ' + document.URL.split('/', 3).join('/') + '/connection/'
    + keyConnect + ' | sh -'
}

function openAddDeployment(name) {
  nameNodeToDeploy.value = name
  nameDeployment.value = ''
  manifestsList.value = []
  suggestOpen.value = false
  suggestIndex.value = -1
  fetch('/internal/manifests').then(r => {
    if (!r.ok) {
      return
    }
    const existing = nodes.value?.find(n => n.name === name)?.deployments || {}
    const taken = new Set()
    Object.entries(existing).forEach(([space, deps]) => {
      deps.forEach(d => taken.add(space === '!without' ? d : `${space}.${d}`))
    })
    manifestsList.value = (r.data || [])
      .map(m => m.name)
      .filter(n => !taken.has(n))
  })
  showAddDeployment.value = !showAddDeployment.value
}

const filteredManifests = () => {
  const q = nameDeployment.value.toLowerCase().trim()
  const list = manifestsList.value
  if (!q) return list
  return list.filter(n => n.toLowerCase().includes(q))
}

function highlight(name) {
  const q = nameDeployment.value.trim()
  if (!q) return [{t: 'text', v: name}]
  const i = name.toLowerCase().indexOf(q.toLowerCase())
  if (i < 0) return [{t: 'text', v: name}]
  return [
    {t: 'text', v: name.slice(0, i)},
    {t: 'mark', v: name.slice(i, i + q.length)},
    {t: 'text', v: name.slice(i + q.length)}
  ]
}

function pickManifest(name) {
  nameDeployment.value = name
  suggestOpen.value = false
  suggestIndex.value = -1
}

function onSuggestKey(e) {
  const list = filteredManifests()
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    suggestOpen.value = true
    suggestIndex.value = Math.min(suggestIndex.value + 1, list.length - 1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    suggestIndex.value = Math.max(suggestIndex.value - 1, 0)
  } else if (e.key === 'Enter') {
    if (suggestOpen.value && suggestIndex.value >= 0 && list[suggestIndex.value]) {
      e.preventDefault()
      pickManifest(list[suggestIndex.value])
    }
  } else if (e.key === 'Escape') {
    suggestOpen.value = false
  }
}

function onSuggestBlur() {
  setTimeout(() => suggestOpen.value = false, 120)
}

function delNode(name) {
  Delete([name, 'destroy all deployments of ' + name, 'destroy cargo-deployer of ' + name]).then(() => {
    Confirm('Are you sure?', [name]).then(() => {
      fetch('/internal/nodes', {
        method: 'DELETE',
        data: {name}
      }).then(r => {
        if (!r.ok) {
          return Alert(r.message)
        }
        refresh()
      })
    }, () => {
    })
  }, () => {
  })
}

function addDeployment() {
  fetch('/internal/nodes/deployments', {
    method: 'POST',
    data: {
      name: nameDeployment.value,
      node: nameNodeToDeploy.value
    }
  }).then(r => {
    if (!r.ok) {
      return Alert(r.message)
    }
    nameDeployment.value = ''
    showAddDeployment.value = null
    refresh()
  })
}

function delDeployment(space, name, node) {
  name = `${space}.${name}`
  Delete([`${name} of ${node} node`]).then(() => {
    fetch('/internal/nodes/deployments', {
      method: 'DELETE',
      data: {
        name: name.replace('!without.', ''),
        node
      }
    }).then(r => {
      if (!r.ok) {
        return Alert(r.message)
      }
      refresh()
    })
  }, () => {
  })
}
</script>
<template>
  <div style="display: flex; margin: 0 0 12px">
    <h2>Connected nodes</h2>
    <button style="margin: 16px" @click="openNewNode">Connect new node</button>
  </div>

  <popup-modal :open="showNewNode" @close="refresh">
    <h2>New connect</h2>
    <div>
      Command with unique key, lifetime 5 minute
    </div>
    <div :class="$style.connect">
      <copy-obj :payload="connectCommand()" :slot-show="true">Copy</copy-obj>
      <div>{{ connectCommand() }}</div>
    </div>
  </popup-modal>

  <div :class="$style.nodes">
    <fieldset v-for="(e, key) in nodes" :key="key">
      <legend>{{ e.name }}<span>node</span></legend>
      <div style="margin: 10px">
        <h3>Node info</h3>
        <ul>
          <li :class="$style['mod-btn']">
            <span :class="$style.name"><span :class="$style.grey">Hostname: </span>{{ e.name }}</span>
            <span :class="[$style.del, $style['del-blue']]" @click="delNode(e.name)" />
          </li>
          <li><span :class="$style.grey">IP Address: </span>{{ e.ip }}</li>
          <li :class="{[$style.wrong]: ((new Date()).getTime() / 1000 - e.update) > 20}">
            <span :class="$style.grey">Update: </span>
            <span v-if="e.update">{{ ((new Date()).getTime() / 1000 - e.update).toFixed(0) + ' sec. ago' }}</span>
            <span v-else>offline</span>
          </li>
        </ul>
        <ul v-if="e.uptime">
          <li><span :class="$style.grey">Load Average: </span>{{ e.uptime.split('load average')[1].substring(2) }}</li>
          <li><span :class="$style.grey">Uptime: </span>{{ e.uptime.split('load average')[0].slice(0, -3) }}</li>
          <li><span :class="$style.grey">Containers Workers: </span>{{ e.workersContainers }}</li>
          <li><span :class="$style.grey">Deployments in Queue: </span>{{ e.inQueue }}</li>
        </ul>
        <h3>
          <span>Deployments manifests</span>
          <button style="margin-left: 16px" @click="openAddDeployment(e.name)">Add</button>
        </h3>
        <div>
          <div :class="$style['space-deployments']">
            <div v-for="(deps, space) in e.deployments" :key="space">
              <fieldset>
                <legend>{{ space }}<span>space</span></legend>
                <ul :class="$style['list-deployments']">
                  <li v-for="name in deps" :key="name" :class="$style['mod-btn']">
                    <span :class="$style.name">{{ name }}</span>
                    <span :class="$style.del" @click="delDeployment(space, name, e.name)" />
                  </li>
                </ul>
              </fieldset>
            </div>
            <span v-for="i in 10" :key="i" />
          </div>
        </div>
      </div>
    </fieldset>

    <popup-modal :open="showAddDeployment" @close="refresh">
      <h2>Add deployment to {{ nameNodeToDeploy }}</h2>
      <div :class="$style['edit-form']">
        <div :class="$style.label">Name manifest</div>
        <div :class="$style.suggest">
          <input
            v-model="nameDeployment"
            type="text"
            placeholder="example: project.nginx-deployment"
            autocomplete="off"
            spellcheck="false"
            @focus="suggestOpen = true"
            @input="suggestOpen = true; suggestIndex = -1"
            @blur="onSuggestBlur"
            @keydown="onSuggestKey"
          >
          <div
            v-if="suggestOpen && filteredManifests().length"
            :class="$style['suggest-list']"
          >
            <div
              v-for="(m, i) in filteredManifests()"
              :key="m"
              :class="[$style['suggest-item'], {[$style.active]: i === suggestIndex}]"
              @mouseenter="suggestIndex = i"
              @mousedown.prevent="pickManifest(m)"
            >
              <template v-for="(part, k) in highlight(m)" :key="k">
                <b v-if="part.t === 'mark'">{{ part.v }}</b>
                <span v-else>{{ part.v }}</span>
              </template>
            </div>
          </div>
        </div>
      </div>
      <button @click="addDeployment">Add</button>
    </popup-modal>
  </div>
</template>
<style module>
.connect {
  background-color: rgba(0, 0, 0, .8);
  padding: 12px;
  color: white;
  border-radius: 8px;
  margin: 12px 0 0;
}

.connect div {
  font-family: monospace;
  margin: 8px 0 0;
  line-break: anywhere;
}

.nodes {
  display: flex;
  flex-wrap: wrap;
}

.nodes > fieldset {
  width: 100%;
}

.nodes ul {
  display: flex;
  flex-flow: row;
  flex-wrap: wrap;
}

.nodes li {
  margin: 0 8px 8px 0;
  border: 1px solid var(--bg-1);
  border-radius: 10px;
  padding: 4px 8px;
  font-size: .9rem;
}

.nodes .grey {
  color: var(--text-light);
}

.nodes li.wrong {
  border-color: rgba(255, 69, 0, .5);
}

.nodes .space-deployments {
  display: flex;
  flex-wrap: wrap;
}

.nodes .space-deployments > * {
  width: 300px;
  flex-grow: 1;
  display: flex;
}

.nodes .space-deployments > * > fieldset {
  flex-grow: 1;
}

.nodes .list-deployments {
  display: flex;
  flex-direction: column;
}

.nodes .list-deployments li {
  width: fit-content;
}

.nodes li.mod-btn {
  padding: 0;
}

.nodes li.mod-btn:hover {
  border-color: var(--bg-2);
}

.nodes li.mod-btn .name {
  padding: 4px 8px;
  display: inline-block;
  border-right: 1px solid var(--bg-1);
  border-radius: 8px;
}

.nodes li.mod-btn .del {
  padding: 4px 8px 4px 6px;
  border-radius: 8px;
  cursor: pointer;
}

.nodes li.mod-btn .del:after {
  content: '';
  display: inline-block;
  -webkit-mask: var(--icon-cross) no-repeat 2px/13px;
  mask: var(--icon-cross) no-repeat 2px/13px;
  background-color: var(--text-light);
  width: 15px;
  height: 15px;
  vertical-align: middle;
}

.nodes li.mod-btn .del.del-blue:after {
  background-color: #feab3a;
}

.nodes li.mod-btn .del:hover:after {
  background-color: rgba(255, 69, 0, .8);
}

.edit-form .label {
  font-size: .7rem;
  margin: 8px 6px -2px;
  text-transform: uppercase;
}

.edit-form input {
  min-width: auto;
  width: 66%;
}

.suggest {
  position: relative;
  width: 66%;
}

.suggest input {
  width: 100%;
  box-sizing: border-box;
}

.suggest-list {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  background: #fff;
  border: 1px solid rgba(0, 0, 0, .12);
  border-radius: 8px;
  box-shadow: 0 6px 18px rgba(0, 0, 0, .18);
  max-height: 220px;
  overflow-y: auto;
  z-index: 50;
  padding: 4px;
}

.suggest-item {
  padding: 6px 10px;
  cursor: pointer;
  font-size: .9rem;
  border-radius: 6px;
  line-height: 1.2;
  color: #333;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.suggest-item b {
  font-weight: 600;
  color: #1763d6;
}

.suggest-item.active {
  background: rgba(23, 99, 214, .12);
}
</style>