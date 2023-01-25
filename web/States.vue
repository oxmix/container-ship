<template>
  <fieldset v-for="namespace in Object.keys(states)" :key="namespace" class="deployments">
    <legend>{{ namespace }}<span>namespace</span></legend>

    <fieldset v-for="d in states[namespace]" :key="d.space+d.name">
      <legend>{{ d.name }}<span>manifest</span></legend>
      <table>
        <thead>
          <tr>
            <td>container</td>
            <td>status</td>
            <td>node</td>
          </tr>
        </thead>
        <tbody>
          <slot v-for="host in Object.keys(d.nodes)" :key="d.space+d.name+host">
            <tr
              v-for="n in d.nodes[host]"
              :key="d.space+d.name+host+n.name"
              @click="$router.push('/logs/'+host+'/'+d.space+'.'+n.name);"
            >
              <td>
                <router-link :to="'/logs/'+host+'/'+d.space+'.'+n.name">
                  {{ n.name }}
                </router-link>
              </td>
              <td>
                <span :class="'label '+n.state">{{ n.status.toLowerCase() }}</span>
              </td>
              <td>
                <span :class="'label ' + (n.nodeLive > 20 ? 'wrong' : 'running')">
                  {{ host }}: {{ n.nodeLive }} sec.
                </span>
              </td>
            </tr>
          </slot>
        </tbody>
      </table>
    </fieldset>
  </fieldset>
</template>

<script>
export default {
  name: 'PageStates',
  data() {
    return {
      states: {},
      updater: null,
      memSpace: ''
    }
  },
  created() {
    this.refresh();
    this.updater = setInterval(() => this.refresh(), 3000);
  },
  unmounted() {
    clearInterval(this.updater);
  },
  methods: {
    refresh() {
      this.$fetch('/states').then((r) => {
        if (r.ok) {
          this.states = r.data;
        }
      });
    }
  }
}
</script>

<style>
.deployments fieldset {
  display: flex;
  flex-wrap: wrap;
}

.deployments table {
  width: 100%;
}

.deployments table td {
  width: 33.3%;
  padding: 8px 10px;
  vertical-align: top;
  white-space: nowrap;
}

.deployments table thead td {
  color: grey;
  text-transform: uppercase;
  font-size: 10px;
}

.deployments table tbody tr {
  cursor: pointer;
}

.deployments table tbody tr:nth-child(odd) {
  background-color: var(--bg-03);
  border-radius: 2px;
}

.deployments table tbody tr:hover {
  background-color: var(--bg-dark-05);
}

.deployments .label {
  margin: 0 5px 5px 0;
  border: 1px solid #ddd;
  border-radius: 10px;
  padding: 4px 8px;
  font-size: .8rem;
}

.deployments .label.wrong {
  border-color: rgba(255, 69, 0, .5);
}

.deployments .label.running {
  border-color: rgba(154, 205, 50, .5);
}
</style>