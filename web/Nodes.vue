<template>
  <fieldset class="connection">
    <legend>connection new node</legend>
    <ul>
      <li>{{ curl }}</li>
    </ul>
  </fieldset>
  <div class="nodes">
    <fieldset v-for="(e, key) in nodes" :key="key">
      <legend>{{ e.name }}</legend>
      <ul>
        <li><span>ip:</span> {{ e.ip }}</li>
        <li :class="((new Date()).getTime() / 1000 - e.update) > 20 ? 'wrong' : ''">
          <span>update:</span> {{ ((new Date()).getTime() / 1000 - e.update).toFixed(0) + ' sec. ago' }}
        </li>
        <li><span>loadAverage:</span> {{ e.uptime.split(',  load average')[1].substring(2) }}</li>
        <li><span>uptime:</span> {{ e.uptime.split(',  load average')[0] }}</li>
        <li><span>workersContainers:</span> {{ e.workersContainers }}</li>
        <li><span>inQueue:</span> {{ e.inQueue }}</li>
      </ul>
    </fieldset>
  </div>
</template>

<script>
export default {
  name: 'PageNodes',
  data() {
    return {
      curl: 'curl -s' + (document.location.port === '8443' ? 'k' : '')
        + ' ' + document.URL.split('/', 3).join('/') + '/connection | sudo bash -',
      nodes: {},
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
      this.$fetch('/nodes/stats').then((r) => {
        if (r.ok) {
          this.nodes = r.data;
        }
      });
    }
  }
}
</script>

<style>
.connection {
  display: inline-block;
}

.connection li {
  list-style-type: '$ ';
  margin: 8px 25px;
}

.nodes {
  display: flex;
  flex-wrap: wrap;
}

.nodes > fieldset {
  width: 300px;
}

.nodes ul {
  display: flex;
  flex-flow: row;
  flex-wrap: wrap;
}

.nodes li {
  margin: 0 5px 5px 0;
  border: 1px solid var(--bg-1);
  border-radius: 10px;
  padding: 4px 8px;
  font-size: .8rem;
}

.nodes li > span {
  color: gray;
}

.nodes li.wrong {
  border-color: rgba(255, 69, 0, .5);
}
</style>