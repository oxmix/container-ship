<script setup>
import {inject, onMounted, ref} from "vue";
import CopyObj from "~/components/CopyObj.vue";
import {Alert, Delete} from "~/components/alert/alert";
import PopupModal from "~/components/PopupModal.vue";

const fetch = inject('fetch')
const list = ref({})
onMounted(refresh)

function refresh() {
  fetch('/internal/variables').then(e => {
    list.value = e.data
  })
}

const showNewVar = ref(false)
const name = ref('')
const data = ref('')
const node = ref('')
const newName = ref('')
const newNode = ref('')
const editSignal = ref(null)

function open() {
  name.value = ''
  node.value = ''
  data.value = ''
  showNewVar.value = !showNewVar.value
}

function add() {
  fetch('/internal/variables', {
    method: 'POST',
    data: {
      name: name.value,
      node: node.value,
      data: data.value
    }
  }).then(r => {
    if (!r.ok) {
      return Alert(r.message)
    }
    showNewVar.value = null
    refresh()
  })
}

function edit(currName, currNode) {
  name.value = currName
  node.value = currNode
  newName.value = currName
  newNode.value = currNode
  editSignal.value = !editSignal.value
}

function editSave() {
  fetch('/internal/variables', {
    method: 'PATCH',
    data: {
      name: name.value,
      node: node.value,
      newName: newName.value,
      newNode: newNode.value
    },
  }).then(r => {
    if (!r.ok) {
      return Alert(r.message)
    }
    editSignal.value = null
    refresh()
  })
}

function remove(name, node) {
  Delete([`${name}${node ? ` of ${node}` : ''}`]).then(() => {
    fetch('/internal/variables', {
      method: 'DELETE',
      data: {name, node},
    }).then(() => {
      refresh()
    })
  }, () => {
  })
}
</script>
<template>
  <div style="display: flex; margin: 0 0 12px">
    <h2>Environment variables</h2>
    <button style="margin: 16px" @click="open">Add new variable</button>
  </div>

  <popup-modal :open="showNewVar">
    <h2>New variable</h2>
    <div :class="$style['edit-form']">
      <div :class="$style.label">Name</div>
      <input v-model="name" type="text" placeholder="PROJECT_DB_PASSWORD">
    </div>
    <div :class="$style['edit-form']">
      <div :class="$style.label">Exclusive for node (optional)</div>
      <input v-model="node" type="text" placeholder="hostname or empty field">
    </div>
    <div :class="$style['edit-form']">
      <div :class="$style.label">Data</div>
      <textarea v-model="data" spellcheck="false" placeholder="dBpAsSwOrD" />
    </div>
    <button @click="add">Add</button>
  </popup-modal>

  <fieldset>
    <legend>Values visible in terminal only</legend>
    <div>
      <copy-obj payload="less assets/manifests/_variables.yaml">less assets/manifests/_variables.yaml</copy-obj>
    </div>
  </fieldset>

  <fieldset>
    <legend>List of variables</legend>

    <table>
      <thead>
        <tr>
          <td style="width: 400px">Name</td>
          <td style="width: 100px">Size</td>
          <td style="width: 180px">Exclusive for Node</td>
          <td>Uses in Deployments</td>
          <td style="width: 25px" />
          <td style="width: 25px" />
        </tr>
      </thead>
      <tbody>
        <tr v-for="e in list" :key="e.name">
          <td>
            <span>{{ e.name }}</span>
            <copy-obj :payload="`{{${e.name}}}`" :slot-show="true" :class="$style['copy-hidden']">Copy</copy-obj>
          </td>
          <td>{{ e.size }}</td>
          <td>
            <span v-if="e.node">{{ e.node }}</span>
            <span v-else style="color: var(--text-light)">–</span>
          </td>
          <td>
            <span v-if="!e.uses" style="color: var(--text-light)">–</span>
            <span v-for="u in e.uses" v-else :key="u" class="label">{{ u.replace('-deployment', '') }}</span>
          </td>
          <td>
            <span class="edit" @click="edit(e.name, e.node)" />
          </td>
          <td>
            <span class="remove" @click="remove(e.name, e.node)" />
          </td>
        </tr>
      </tbody>
    </table>
  </fieldset>

  <popup-modal :open="editSignal">
    <h2>Edit variable: {{ name }}<slot> of {{ node }}</slot></h2>
    <div :class="$style['edit-form']">
      <div :class="$style.label">Name</div>
      <input v-model="newName" type="text" placeholder="PROJECT_NEW_NAME_VAR">
    </div>
    <div :class="$style['edit-form']">
      <div :class="$style.label">Exclusive for node (optional)</div>
      <input v-model="newNode" type="text" placeholder="hostname or empty field">
    </div>
    <button @click="editSave">Save</button>
  </popup-modal>
</template>
<style module>
.copy-hidden {
  display: none;
}

table td:hover .copy-hidden {
  display: inline;
  margin: -1px 8px;
}

.edit-form .label {
  font-size: .7rem;
  margin: 8px 6px -2px;
  text-transform: uppercase;
}

.edit-form input,
.edit-form textarea {
  min-width: auto;
  width: 66%;
}

.edit-form textarea {
  white-space: nowrap;
}
</style>