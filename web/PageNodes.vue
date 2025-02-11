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
  showAddDeployment.value = !showAddDeployment.value
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
        <input v-model="nameDeployment" type="text" placeholder="example: project.nginx-deployment">
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
</style>