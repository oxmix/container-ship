<template>
  <section class="hub">
    <fieldset v-for="repo in Object.keys(list)" :key="repo">
      <legend @click="remove(repo, 'latest')">{{ repo }}<span>latest</span></legend>
      <ul v-for="osPlatform in Object.keys(list[repo])" :key="repo+osPlatform">
        <li>{{ osPlatform }} <span>{{ list[repo][osPlatform] }}</span></li>
      </ul>
    </fieldset>
  </section>
  <div class="info-line">{{ totalSizeView }}</div>
  <div class="info-line">{{ err }}</div>
</template>

<script>
export default {
  name: 'PageHub',
  data() {
    return {
      list: {},
      totalSize: 0,
      totalSizeView: '0 Byte',
      err: ''
    }
  },
  created() {
    this.get()
  },
  methods: {
    get() {
      this.$fetch('/v2/_catalog').then((catalog) => {
        catalog.repositories.forEach((repo) => {
          this.$fetch('/v2/' + repo + '/manifests/latest', {
            headers: {'Accept': 'application/vnd.docker.distribution.manifest.list.v2+json'}
          }).then((m) => {
            m.manifests.forEach((e) => {
              if (!this.list[repo]) {
                this.list[repo] = {};
              }
              this.list[repo][e.platform.os + '/' + e.platform.architecture] = '';
              this.getSize(repo, e.platform.os + '/' + e.platform.architecture, e.digest);
            });
          });
        });
      }).catch((e) => {
        this.totalSizeView = document.URL.split('/', 3).join('/')
          + '/v2/ – entrypoint is not api registry hub';
        this.err = e;
      });
    },

    getSize(repo, osPlatform, digest) {
      return this.$fetch('/v2/' + repo + '/manifests/' + digest, {
        headers: {'Accept': 'application/vnd.docker.distribution.manifest.list.v2+json'}
      }).then((d) => {
        if (d.errors) {
          console.error(d.errors);
        }
        let size = 0;
        d.layers.map((e) => size += e.size)
        this.list[repo][osPlatform] = (size / 1024 / 1024).toFixed(2) + ' MB';
        this.totalSize += size;
        this.totalSizeView =
          'Total compressed size: ' + (this.totalSize / 1024 / 1024).toFixed(2) + ' MB'
      });
    },

    remove(repo, tag) {
      if (!confirm('Remove latest tag?'))
        return false;

      this.$fetch('/v2/' + repo + '/manifests/' + tag, {
        method: 'GET',
        headers: {
          'Accept': 'application/vnd.docker.distribution.manifest.v2+json'
            + ', application/vnd.oci.image.manifest.v1+json'
            + ', application/vnd.docker.distribution.manifest.list.v2+json'
            + ', application/vnd.oci.image.index.v1+json'
        }
      }).then((m) => {
        if (m.errors) {
          console.error(m.errors);
          return;
        }
        const queue = [];
        m.manifests.forEach((mm) => {
          queue.push(this.$fetch('/v2/' + repo + '/manifests/' + mm.digest, {
            method: 'DELETE',
            headers: {
              'Accept': 'application/vnd.docker.distribution.manifest.v2+json'
                + ', application/vnd.oci.image.manifest.v1+json'
            }
          }, (d) => {
            console.info("delete manifest: ", d);
          }));
        });

        Promise.all(queue).then(() => {
          delete this.list[repo];
        });
      })
    }
  },
}
</script>

<style>
.hub {
  display: flex;
  justify-content: space-around;
  flex-wrap: wrap;
}

.info-line {
  margin: 20px 0 0;
  text-align: center;
  color: grey;
}

.hub fieldset {
  min-width: 250px;
}

.hub fieldset legend {
  cursor: pointer;
}

.hub fieldset legend:hover {
  color: orangered;
}

.hub ul {
  flex-flow: column;
}

.hub ul li {
  margin: 0 5px 5px 0;
  border: 1px solid #ddd;
  border-radius: 10px;
  padding: 4px 8px;
  font-size: .8rem;
}

.hub li > span {
  color: gray;
}
</style>