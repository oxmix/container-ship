<script setup>
import {inject, onMounted, onUnmounted, ref} from "vue";

const fetch = inject('fetch')
const nodeTimeout = 20
let updater = null
const states = ref({})
const indicators = ref({
  nodes: {
    good: 0,
    total: 0
  },
  containers: {
    good: 0,
    total: 0
  }
})

onMounted(() => {
  refresh()
  updater = setInterval(() => refresh(), 3000)
})
onUnmounted(() => clearInterval(updater))

function refresh() {
  fetch('/internal/states').then((r) => {
    if (r.ok) {
      const nodesTotal = [],
        nodesTimeout = [],
        containersTotal = [],
        containersFailed = []
      Object.keys(r.data).forEach(space => {
        Object.keys(r.data[space]).forEach(manifest => {
          Object.keys(r.data[space][manifest]).forEach(nodeName => {
            r.data[space][manifest][nodeName].forEach(c => {
              if (!nodesTotal.includes(c.node)) {
                nodesTotal.push(c.node)
              }
              if (!nodesTimeout.includes(c.node) && (c.refresh > nodeTimeout || c.refresh === -999)) {
                nodesTimeout.push(c.node)
              }
              if (!containersTotal.includes(c.node + '-' + c.name)) {
                containersTotal.push(c.node + '-' + c.name)
              }
              if (!containersFailed.includes(c.node + '-' + c.name) && c.state !== 'running') {
                containersFailed.push(c.node + '-' + c.name)
              }
            })
          })
        })
      })

      indicators.value = {
        nodes: {
          good: nodesTotal.length - nodesTimeout.length,
          total: nodesTotal.length
        },
        containers: {
          good: containersTotal.length - containersFailed.length,
          total: containersTotal.length
        }
      }

      states.value = r.data
    }
  })
}
</script>
<template>
  <div style="display: flex; margin: 0 0 12px">
    <h2>States overview</h2>
    <div class="labels" style="margin: 16px 12px; font-size: 1.3rem">
      <div
        class="label"
        :class="{[indicators.nodes.good < indicators.nodes.total ? 'red' : 'green']: indicators.nodes.total > 0}"
      >
        Nodes {{ indicators.nodes.good }}/{{ indicators.nodes.total }}
      </div>
      <div
        class="label"
        :class="{[indicators.containers.good < indicators.containers.total ? 'red' : 'green']: indicators.containers.total > 0}"
      >
        Containers {{ indicators.containers.good }}/{{ indicators.containers.total }}
      </div>
    </div>
  </div>

  <fieldset v-for="namespace in Object.keys(states)" :key="namespace" :class="$style.deployments">
    <legend>{{ namespace }}<span>namespace</span></legend>

    <fieldset v-for="manifest in Object.keys(states[namespace])" :key="namespace+manifest">
      <legend>{{ manifest }}<span>manifest</span></legend>
      <table>
        <thead>
          <tr>
            <td>container</td>
            <td>status</td>
            <td>node</td>
          </tr>
        </thead>
        <tbody>
          <slot v-for="(nodes, host) in states[namespace][manifest]" :key="namespace+manifest+host">
            <tr
              v-for="n in nodes"
              :key="namespace+manifest+host+n.name"
              @click="$router.push('/logs/'+host+'/'+namespace+'.'+n.name)"
            >
              <td>
                <router-link :to="'/logs/'+host+'/'+namespace+'.'+n.name">
                  {{ n.name }}
                </router-link>
              </td>
              <td>
                <span class="label" :class="n.state || 'no-data'">{{ n.status.toLowerCase() }}</span>
              </td>
              <td>
                <span v-show="n.refresh !== -999" class="label" :class="[n.refresh > nodeTimeout ? 'red' : 'green']">
                  {{ host }}: {{ n.refresh }} sec.
                </span>
              </td>
            </tr>
          </slot>
        </tbody>
      </table>
    </fieldset>
  </fieldset>
</template>
<style module>
.deployments table td {
  width: 33.3%;
}

.deployments table tbody tr {
  cursor: pointer;
}

.deployments table :global(.label.paused),
.deployments table :global(.label.no-data) {
  border-color: rgba(254, 171, 58, .5);
}

.deployments table :global(.label.restarting) {
  border-color: rgba(255, 69, 0, .5);
}

.deployments table :global(.label.running) {
  border-color: rgba(154, 205, 50, .3);
}
</style>