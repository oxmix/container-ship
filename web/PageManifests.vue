<script setup>
import {inject, onMounted, ref} from "vue";
import CopyObj from "~/components/CopyObj.vue";
import {Alert, Delete} from "~/components/alert/alert";
import PopupModal from "~/components/PopupModal.vue";

const fetch = inject('fetch')
const manifests = ref({})
onMounted(refresh)

function refresh() {
  fetch('/internal/manifests').then(e => {
    let list = {}
    e.data.forEach(m => {
      let spl = m.name.split('.')
      let space = spl[0] ? spl[0] : '!without'
      if (!list[space]) {
        list[space] = []
      }
      list[space].push(m)
    })
    manifests.value = list
  })
}

const showNewMan = ref(false)
const data = ref('')
const editName = ref('')
const editSignal = ref(null)

function save() {
  fetch('/internal/manifests', {
    method: 'POST',
    data: {
      data: data.value
    }
  }).then(r => {
    if (!r.ok) {
      return Alert(r.message)
    }
    if (r.data['without-deploy']) {
      Alert("Only saved", ['no sets nodes for deployment'])
    } else {
      Alert("Saved and sent signal for deployment to nodes:", r.data)
    }
    showNewMan.value = null
    editSignal.value = null
    data.value = ''
    refresh()
  })
}

function edit(name, conf) {
  editName.value = name
  data.value = conf
  editSignal.value = !editSignal.value
}

function remove(name) {
  Delete([name, 'as well all nodes destroy this deployment']).then(() => {
    fetch('/internal/manifests?name=' + name, {
      method: 'DELETE'
    }).then(() => {
      refresh()
    })
  }, () => {
  })
}

function templateCanary() {
  data.value = `space: project
name: nginx-canary-deployment
canary:
  delay: 5 # sec. wait each container
containers:
  - name: nginx-ingress-1
    from: nginx:latest
  - name: nginx-ingress-2
    from: nginx:latest
  - name: nginx-ingress-3
    from: nginx:latest
`
}

function templateConfig() {
  data.value = `space: project
name: nginx-ingress-deployment
webhook: https://.../event-finish # optional
containers:
  - name: nginx-ingress
    from: nginx:latest

    # next, optionally
    stop-timeout: 10 # sec. wait terminate
    runtime: nvidia
    pid: host
    privileged: true
    restart: always | unless-stopped
    caps:
      - sys_nice
      - others...
    sysctls:
      - net.ipv6.conf.all.disable_ipv6=1
      - others...
    user: www-data
    hostname: custom-sample-name
    network-mode: host
    hosts:
      - host.docker.internal:host-gateway
      - example.ltd:10.11.12.13
    ports:
      - 80:80/tcp
      - 443:443/tcp
      - 443:443/udp
    mounts:
      - type=bind,source=/path/from,target=/path/to
    volumes:
      - /path/from1:/path/to1
      - /path/from2:/path/to2:ro
    environment:
       - EXAMPLE_ENV=regular_variable
       # uses container ship variables
       - EXAMPLE_SECRET={{PROJECT_EXAMPLE_SECRET}}
    entrypoint: nginx -g 'daemon off;'
    command: nginx -g 'daemon off;'
    executions:
      - htpasswd -cb /accounts user1 pass1 2>&1
      - htpasswd -b /accounts user2 pass2 2>&1
`
}

function ago(seconds, locale) {
  locale = locale || 'en'
  seconds = new Date().getTime() / 1000 - seconds
  let interval = seconds / 31536000
  const rtf = new Intl.RelativeTimeFormat(locale || navigator.language, {
    numeric: 'auto'
  })
  if (interval > 1) {
    return rtf.format(-Math.floor(interval), 'year')
  }
  interval = seconds / 2592000
  if (interval > 1) {
    return rtf.format(-Math.floor(interval), 'month')
  }
  interval = seconds / 86400
  if (interval > 1) {
    return rtf.format(-Math.floor(interval), 'day')
  }
  interval = seconds / 3600
  if (interval > 1) {
    return rtf.format(-Math.floor(interval), 'hour')
  }
  interval = seconds / 60
  if (interval > 1) {
    return rtf.format(-Math.floor(interval), 'minute')
  }
  return rtf.format(-Math.floor(interval), 'second')
}
</script>
<template>
  <div style="display: flex; margin: 0 0 12px">
    <h2>Deployments manifests</h2>
    <button style="margin: 16px" @click="data = ''; showNewMan = !showNewMan">Add new manifest</button>
  </div>

  <popup-modal :open="showNewMan">
    <h2>New deployment manifest</h2>

    <div :class="$style['edit-form']">
      <div :class="$style.label">Sample templates</div>
      <button @click="templateConfig">Paste with all params</button>
      <span style="margin: 0 5px" />
      <button @click="templateCanary">Pate with canary deploy</button>
    </div>

    <div :class="$style['edit-form']">
      <div :class="$style.label">Manifest yaml config</div>
      <textarea v-model="data" spellcheck="false" />
    </div>
    <button @click="save">Add and Deploy</button>
  </popup-modal>

  <div v-for="(mans, space) in manifests" :key="space">
    <fieldset>
      <legend>{{ space }}<span>space</span></legend>
      <table>
        <thead>
          <tr>
            <td style="width: 360px">Name</td>
            <td style="width: 130px">Modify</td>
            <td>Uses in Nodes</td>
            <td style="width: 25px" />
            <td style="width: 25px" />
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in mans" :key="e.key">
            <td>
              <span>{{ e.name }}</span>
              <copy-obj :payload="e.name" :slot-show="true" :class="$style['copy-hidden']">Copy</copy-obj>
            </td>
            <td>{{ ago(e.modify) }}</td>
            <td>
              <span v-if="!e.nodes">–</span>
              <span v-for="n in e.nodes" v-else :key="n" class="label">{{ n }}</span>
            </td>
            <td>
              <span class="edit" @click="edit(e.name, e.config)" />
            </td>
            <td>
              <span class="remove" @click="remove(e.name)" />
            </td>
          </tr>
        </tbody>
      </table>
    </fieldset>
  </div>

  <popup-modal :open="editSignal">
    <h2>Edit variable: {{ editName }}</h2>
    <div :class="$style['edit-form']">
      <div :class="$style.label">Manifest yaml config</div>
      <textarea v-model="data" spellcheck="false" />
    </div>
    <button @click="save">Save and Deploy</button>
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
  min-width: 100%;
  height: 600px;
  line-height: 20px;
  white-space: nowrap;
}
</style>