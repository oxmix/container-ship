<script setup>
import {computed, inject, onMounted, onUnmounted, ref} from "vue";

const fetch = inject('fetch')
const nodeTimeout = 20
let updater = null
const states = ref({})
const query = ref('')
const issuesOnly = ref(false)

onMounted(() => {
  refresh()
  updater = setInterval(() => refresh(), 3000)
})
onUnmounted(() => clearInterval(updater))

function refresh() {
  fetch('/internal/states').then((r) => {
    if (r.ok) {
      states.value = r.data || {}
    }
  })
}

function offline(c) {
  return c.refresh === -999 || c.refresh > nodeTimeout
}

function versionTag(imageVer) {
  if (!imageVer) {
    return ''
  }
  const pos = imageVer.lastIndexOf(':')
  if (pos < 0 || imageVer.slice(pos).includes('/')) {
    return imageVer.split('/').pop()
  }
  return imageVer.slice(pos + 1)
}

function shortName(space, manifest) {
  let name = manifest.startsWith(space + '.') ? manifest.slice(space.length + 1) : manifest
  if (name.endsWith('-deployment') && name !== '-deployment') {
    name = name.slice(0, -'-deployment'.length)
  }
  return name
}

function groupVersions(nodes) {
  const groups = []
  nodes.forEach(n => {
    if (!n.imageVer) {
      return
    }
    const found = groups.find(g => g.full === n.imageVer)
    if (found) {
      found.count++
    } else {
      groups.push({full: n.imageVer, tag: versionTag(n.imageVer), count: 1})
    }
  })
  return groups
}

const overview = computed(() => {
  // freshness of a node is the best refresh among its containers,
  // -999 means only that the container is absent on the node
  const nodes = {}
  const byState = {}
  let containers = 0, running = 0, manifests = 0
  Object.keys(states.value).forEach(space => {
    Object.keys(states.value[space]).forEach(manifest => {
      manifests++
      Object.keys(states.value[space][manifest]).forEach(host => {
        states.value[space][manifest][host].forEach(c => {
          const node = c.node || host
          const prev = nodes[node]
          if (prev === undefined || prev === -999
            || (c.refresh !== -999 && c.refresh < prev)) {
            nodes[node] = c.refresh
          }
          containers++
          if (c.state === 'running' && !offline(c)) {
            running++
          }
          const state = c.refresh > nodeTimeout ? 'offline' : (c.state || 'no-data')
          byState[state] = (byState[state] || 0) + 1
        })
      })
    })
  })
  const online = Object.values(nodes).filter(r => r !== -999 && r <= nodeTimeout).length
  return {
    nodes: {good: online, total: Object.keys(nodes).length},
    containers: {good: running, total: containers},
    namespaces: Object.keys(states.value).length,
    manifests,
    byState: Object.keys(byState)
      .filter(s => s !== 'running')
      .sort()
      .map(s => ({state: s, count: byState[s]}))
  }
})

const tree = computed(() => {
  const q = query.value.toLowerCase().trim()
  const spaces = []

  Object.keys(states.value).forEach(space => {
    const manifests = []

    Object.keys(states.value[space]).forEach(manifest => {
      const byName = new Map()
      const hosts = new Set()

      Object.keys(states.value[space][manifest]).forEach(host => {
        states.value[space][manifest][host].forEach(c => {
          hosts.add(c.node || host)
          if (!byName.has(c.name)) {
            byName.set(c.name, [])
          }
          byName.get(c.name).push({
            node: c.node || host,
            refresh: c.refresh,
            state: c.state,
            status: c.status,
            imageVer: c.imageVer,
            offline: offline(c)
          })
        })
      })

      const containers = []
      byName.forEach((nodes, name) => {
        nodes.sort((a, b) => a.node.localeCompare(b.node))
        const versions = groupVersions(nodes)
        const up = nodes.filter(n => n.state === 'running' && !n.offline).length
        const hay = [space, manifest, name, nodes.map(n => n.node).join(' '),
          versions.map(v => v.full).join(' ')].join(' ').toLowerCase()

        if (q && !hay.includes(q)) {
          return
        }
        if (issuesOnly.value && up === nodes.length) {
          return
        }

        containers.push({
          name,
          nodes,
          versions,
          drift: versions.length > 1,
          up,
          total: nodes.length
        })
      })

      if (containers.length === 0) {
        return
      }

      manifests.push({
        name: manifest,
        // namespace shown by the section and suffix is always the same
        short: shortName(space, manifest),
        containers,
        hosts: hosts.size,
        up: containers.reduce((a, c) => a + c.up, 0),
        total: containers.reduce((a, c) => a + c.total, 0)
      })
    })

    if (manifests.length === 0) {
      return
    }

    spaces.push({
      name: space,
      manifests,
      up: manifests.reduce((a, m) => a + m.up, 0),
      total: manifests.reduce((a, m) => a + m.total, 0)
    })
  })

  return spaces
})

function logsLink(space, container, node) {
  return '/logs/' + node + '/' + space + '.' + container
}
</script>
<template>
  <div :class="$style.head">
    <h2>States overview</h2>
    <div class="labels" :class="$style.summary">
      <div
        class="label"
        :class="{[overview.nodes.good < overview.nodes.total ? 'red' : 'green']: overview.nodes.total > 0}"
      >
        Nodes {{ overview.nodes.good }}/{{ overview.nodes.total }}
      </div>
      <div
        class="label"
        :class="{[overview.containers.good < overview.containers.total ? 'red' : 'green']: overview.containers.total > 0}"
      >
        Containers {{ overview.containers.good }}/{{ overview.containers.total }}
      </div>
      <div class="label">{{ overview.namespaces }} namespaces</div>
      <div class="label">{{ overview.manifests }} manifests</div>
      <div v-for="s in overview.byState" :key="s.state" class="label" :class="s.state">
        {{ s.state }} {{ s.count }}
      </div>
    </div>
    <div :class="$style.tools">
      <input
        v-model="query"
        type="text"
        placeholder="filter namespace, manifest, container, node, image"
        autocomplete="off"
        spellcheck="false"
      >
      <label><input v-model="issuesOnly" type="checkbox"> issues only</label>
    </div>
  </div>

  <fieldset v-for="space in tree" :key="space.name" :class="$style.states">
    <legend>
      {{ space.name }}<span>namespace</span><span
        :class="space.up < space.total ? $style.red : $style.green"
        title="running containers of the namespace"
      >{{ space.up }}/{{ space.total }} running</span>
    </legend>

    <div :class="$style.grid">
      <fieldset
        v-for="manifest in space.manifests"
        :key="space.name+manifest.name"
        :class="[$style.card, {[$style.alarm]: manifest.up < manifest.total}]"
      >
        <legend>
          <span :class="$style.card_name" :title="manifest.name">{{ manifest.short }}</span>
          <span>manifest</span>
          <span
            :class="manifest.up < manifest.total ? $style.red : $style.green"
            title="running containers of the manifest"
          >{{ manifest.up }}/{{ manifest.total }}</span>
        </legend>

        <div v-for="c in manifest.containers" :key="c.name" :class="$style.cont">
          <div :class="$style.cont_head">
            <span :class="[$style.dot, c.up < c.total ? $style.bad : $style.ok]" />
            <span :class="$style.cont_name">{{ c.name }}</span>
            <span :class="$style.versions">
              <span
                v-for="v in c.versions"
                :key="v.full"
                :class="[$style.version, {[$style.drift]: c.drift}]"
                :title="v.full"
              >
                {{ v.tag }}<i v-if="v.count > 1">&times;{{ v.count }}</i>
              </span>
            </span>
          </div>
          <div :class="$style.cont_nodes">
            <router-link
              v-for="n in c.nodes"
              :key="n.node"
              class="label"
              :class="n.state || 'no-data'"
              :to="logsLink(space.name, c.name, n.node)"
              :title="n.refresh === -999 ? 'container absent on the node'
                : n.status + ', node update ' + n.refresh + ' sec. ago'"
            >
              <span :class="$style.node">{{ n.node }}</span>
              <i>{{ (n.status || 'no data').toLowerCase() }}</i>
              <em v-if="n.refresh !== -999" :class="{[$style.late]: n.offline}">{{ n.refresh }}s</em>
            </router-link>
          </div>
        </div>

        <div :class="$style.card_foot">
          {{ manifest.containers.length }} {{ manifest.containers.length > 1 ? 'containers' : 'container' }}
          &middot; {{ manifest.hosts }} {{ manifest.hosts > 1 ? 'nodes' : 'node' }}
        </div>
      </fieldset>
    </div>
  </fieldset>

  <div v-if="!tree.length" :class="$style.empty">
    {{ Object.keys(states).length ? 'nothing matched the filter' : 'no states yet' }}
  </div>
</template>
<style module>
.head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  margin: 0 0 12px;
}

.head h2 {
  margin: 0 12px 0 0;
}

.summary {
  display: flex;
  flex-wrap: wrap;
}

.summary :global(.label) {
  font-size: .8rem;
}

.summary :global(.label.paused),
.summary :global(.label.exited),
.summary :global(.label.no-data) {
  border-color: rgba(254, 171, 58, .5);
}

.summary :global(.label.offline),
.summary :global(.label.restarting),
.summary :global(.label.dead) {
  border-color: rgba(255, 69, 0, .5);
}

.tools {
  display: flex;
  align-items: center;
  margin-left: auto;
  gap: 12px;
}

.tools input[type="text"] {
  min-width: auto;
  width: 300px;
  padding: 6px 10px;
  font-size: .8rem;
}

.tools label {
  color: var(--text-light);
  font-size: .8rem;
  white-space: nowrap;
  cursor: pointer;
}

.states {
  padding: 6px 14px 10px;
}

.states legend .red {
  border-color: rgba(255, 69, 0, .5);
}

.states legend .green {
  border-color: rgba(154, 205, 50, .5);
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(330px, 1fr));
  gap: 10px;
  margin: 4px 0 2px;
}

.card {
  background-color: var(--bg-02);
  margin: 0;
  padding: 2px 12px 8px;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.card.alarm,
.card.alarm:hover {
  border-color: rgba(255, 69, 0, .35);
}

.card_name {
  color: var(--text);
  border: 0;
  border-radius: 0;
  margin: 0;
  padding: 0;
  top: 0;
  font-size: 1rem;
  display: inline-block;
  max-width: 210px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: bottom;
}

.card_foot {
  color: var(--text-light);
  font-size: .7rem;
  margin-top: auto;
  padding: 4px 0 0;
}

.cont {
  padding: 6px 0 2px;
}

.cont_head {
  display: flex;
  align-items: baseline;
  gap: 6px;
  font-size: .85rem;
}

.cont_name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.versions {
  margin-left: auto;
  padding-left: 6px;
  white-space: nowrap;
}

.cont_nodes {
  display: flex;
  flex-wrap: wrap;
  margin: 3px 0 0 13px;
}

.cont_nodes a:global(.label) {
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
  margin: 3px 4px 0 0;
  padding: 1px 7px;
  font-size: .72rem;
  text-decoration: none;
  background-color: var(--bg-02);
}

.cont_nodes a:global(.label):hover {
  border-color: var(--text);
  background-color: var(--bg-05);
}

.cont_nodes i {
  color: var(--text-light);
  font-style: normal;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 200px;
}

.cont_nodes em {
  color: var(--text-light);
  font-style: normal;
  font-family: monospace;
  font-size: .68rem;
}

.cont_nodes em.late {
  color: orangered;
}

.node {
  white-space: nowrap;
}

.dot {
  display: inline-block;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  margin-right: 6px;
  vertical-align: middle;
}

.dot.ok {
  background-color: yellowgreen;
}

.dot.bad {
  background-color: orangered;
}

.version {
  font-family: monospace;
  font-size: .8rem;
  opacity: .8;
  margin-right: 6px;
}

.version i {
  font-style: normal;
  margin-left: 3px;
}

.version.drift {
  color: #feab3a;
  opacity: 1;
}

.empty {
  color: var(--text-light);
  margin: 24px 8px;
}

.cont_nodes :global(.label.paused),
.cont_nodes :global(.label.no-data),
.cont_nodes :global(.label.exited) {
  border-color: rgba(254, 171, 58, .5);
}

.cont_nodes :global(.label.restarting),
.cont_nodes :global(.label.dead) {
  border-color: rgba(255, 69, 0, .5);
}

.cont_nodes :global(.label.running) {
  border-color: rgba(154, 205, 50, .3);
}
</style>