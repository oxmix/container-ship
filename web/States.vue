<template>
  <fieldset v-for="namespace in Object.keys(states)" :key="namespace" class="deployments">
    <legend>{{ namespace }}<span>namespace</span></legend>
    <fieldset v-for="d in states[namespace]" :key="d.space+d.name">
      <legend>{{ d.name }}<span>manifest</span></legend>
      <fieldset v-for="host in Object.keys(d.nodes)" :key="d.space+d.name+host">
        <legend>{{ host }}<span>node</span></legend>
        <ul>
          <router-link
            v-for="n in d.nodes[host]"
            :key="d.space+d.name+host+n.name"
            :class="n.state"
            :to="'/logs/'+host+'/'+d.space+'.'+n.name"
          >
            {{ n.name }}: {{ n.status.toLowerCase() }}
          </router-link>
        </ul>
      </fieldset>
    </fieldset>
  </fieldset>
</template>

<script>
export default {
  name: 'PageStates',
  data() {
    return {
      states: {},
      updater: null
    }
  },
  created() {
    this.refresh()
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
.deployments ul,
.deployments fieldset {
  display: flex;
}

.deployments fieldset {
  flex-wrap: wrap;
}

.deployments > fieldset > fieldset {
  min-width: 300px;
}

.deployments ul {
  flex-flow: column;
}

.deployments a {
  margin: 0 5px 5px 0;
  border: 1px solid #ddd;
  border-radius: 10px;
  padding: 4px 8px;
  font-size: .8rem;
  cursor: pointer;
  text-decoration: none;
}

.deployments a.wrong {
  border-color: orangered;
}

.deployments a.running {
  border-color: yellowgreen;
}

.deployments a:hover {
  border-color: orange;
}
</style>