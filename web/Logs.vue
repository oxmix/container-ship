<template>
  <div class="logs">
    <fieldset>
      <legend>{{ $route.params.container }}<span>{{ $route.params.node }}</span></legend>
      <!-- eslint-disable vue/no-v-html, vue/html-closing-bracket-newline, vue/html-indent -->
      <TransitionGroup tag="div" name="list" @vnode-updated="nodeUpdated">
      <pre v-for="e in logs" :key="e.time"><span
        class="logs-time">{{ e.time }}: </span><span
        v-if="e.json" class="logs-message" v-html="e.json" /><span
        v-else>{{ e.text }}</span></pre>
      </TransitionGroup>
      <!--eslint-enable-->
      {{ err }}
    </fieldset>
  </div>
</template>

<script>
export default {
  name: 'PageLogs',
  data() {
    return {
      updater: null,
      logs: [],
      lastLine: {time: ''},
      autoScroll: true,
      err: ''
    };
  },
  created() {
    this.refresh()
    this.updater = setInterval(() => this.refresh(this.lastLine.time), 3000);
  },
  unmounted() {
    clearTimeout(this.updater);
  },
  methods: {
    nodeUpdated() {
      if (this.autoScroll) {
        const clf = document.querySelector('.logs > fieldset');
        clf.scrollTo(0, clf.scrollHeight);
      }
    },
    refresh(time) {
      this.$fetch('/logs?node=' + this.$route.params.node
        + '&container=' + this.$route.params.container
        + (time ? '&since=' + time : ''))
        .then((r) => {
          if (r.ok) {
            const clf = document.querySelector('.logs > fieldset');
            this.autoScroll = clf.scrollHeight < clf.scrollTop + clf.offsetHeight;

            r.data.forEach((e) => {
              if (e.mess.substring(0, 1) === '{') {
                this.logs.push({
                  time: this.time(e.time),
                  json: this.jsonPretty(JSON.parse(e.mess))
                });
              } else {
                this.logs.push({
                  time: this.time(e.time),
                  text: e.mess
                });
              }
            });
            this.lastLine = r.data[r.data.length - 1] || {time: this.lastLine.time}
            this.err = '';
          } else {
            this.err = r.message + ', auto update via 3 sec...';
          }
        })
    },
    time(date) {
      const time = new Date(date);
      time.setMinutes(time.getMinutes() - time.getTimezoneOffset());
      return time.toISOString().slice(0, 23).replace('T', ' ');
    },
    jsonPretty(json, tab) {
      const jn = JSON.stringify(json, undefined, tab || 4)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;');

      return jn.replace(
        /("(\\u[a-zA-Z\d]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+-]?\d+)?)/g,
        function (match) {
          let cls = 'json-number';
          if (/^"/.test(match)) {
            if (/:$/.test(match)) {
              cls = 'json-key';
            } else {
              cls = 'json-string';
            }
          } else if (/true|false/.test(match)) {
            cls = 'json-number';
          } else if (/null/.test(match)) {
            cls = 'json-null';
          }
          return '<span class="logs-' + cls + '">' + match + '</span>';
        });
    }
  }
}
</script>

<style>
.list-enter-active,
.list-leave-active {
  transition: background-color 3s linear;
}

.list-enter-from,
.list-leave-to {
  background-color: #eee;
}

.logs fieldset {
  height: calc(100vh - 180px);
  overflow-y: scroll;
  overflow-x: hidden;
  padding-bottom: 0;
}

.logs-time {
  color: grey;
}

.logs-message {
  white-space: break-spaces;
}

pre {
  border-radius: 2px;
  padding: 10px;
  margin: 10px 0;
}

.logs-json-key {
  color: #75bfff;
}

.logs-json-number {
  color: #86de74;
}

.logs-json-string {
  color: #ff7de9;
}

.logs-json-null {
  color: #939395;
}
</style>