<script setup>
import {getCurrentInstance, onMounted, ref, render} from 'vue';

const instance = getCurrentInstance()
const props = defineProps({
  container: {
    type: Element,
    required: true
  },
  title: {
    type: String,
    required: false,
    default: ''
  },
  objects: {
    type: Array,
    required: false,
    default: null
  },
  escapeOk: {
    type: Boolean,
    required: false,
    default: false
  }
})

const showAlert = ref(false)

onMounted(() => {
  setTimeout(() => showAlert.value = true, 10)
  document.onkeydown = e => {
    // fixed double action when tap of enter
    // change focus from the element that triggered alert
    if (e.key === 'Escape' || e.keyCode === 27 || e.key === 'Enter' || e.keyCode === 13) {
      e.preventDefault()
    }

    if (e.key === 'Enter' || e.keyCode === 13) {
      action()
    }
    if (e.key === 'Escape' || e.keyCode === 27) {
      if (props.escapeOk) {
        return action()
      }
      cancel()
    }
  }
})

function close() {
  document.onkeydown = null
  showAlert.value = false
  setTimeout(() => {
    render(null, props.container)
    props.container.remove()
  }, 200)
}

function cancel() {
  if ((instance.proxy.actOk || instance.proxy.actDel) && instance.proxy.actCancel) {
    instance.proxy.actCancel()
  }
  close()
}

function action() {
  if (instance.proxy.actOk) {
    instance.proxy.actOk()
  }
  if (instance.proxy.actDel) {
    instance.proxy.actDel()
  }
  close()
}
</script>
<template>
  <div :class="[$style['modal-popup'], {[$style.show]: showAlert}]">
    <div :class="$style['modal-content']">
      <div>
        <h2 v-if="instance.proxy.actDel">Do you want to delete?</h2>
        <h2 v-else>{{ props.title }}</h2>
        <div v-for="(e, k) in props.objects" :key="k" :class="$style.objects">
          <slot v-if="e?.type === 'link'">
            <a :href="e.href" :target="e.target" v-text="e.text" />
          </slot>
          <slot v-else>
            <slot v-if="props.objects.length > 1">•</slot>
            {{ e }}
          </slot>
        </div>
        <div :class="$style.buttons">
          <button v-if="instance.proxy.actCancel" :class="[$style.btn]" @click="cancel()">Cancel</button>
          <button v-if="instance.proxy.actDel" :class="[$style.btn, $style.red]" @click="action()">Delete</button>
          <button v-else :class="[$style.btn]" @click="action()">Ok</button>
        </div>
      </div>
    </div>
  </div>
</template>
<style module>
.modal-popup {
  display: flex;
  justify-content: center;
  position: fixed;
  left: 0;
  bottom: 0;
  right: 0;
  top: 0;
  overflow: auto;
  z-index: 300;
  background-color: rgba(0, 0, 0, .6);
  -webkit-backdrop-filter: blur(1px);
  backdrop-filter: blur(1px);
  padding: 35px;
  transition: visibility 0s .20s, opacity .20s linear;
  visibility: hidden;
  opacity: 0;
  will-change: opacity, transform;
}

.modal-popup.show {
  transition: opacity .20s linear;
  visibility: visible;
  opacity: 1;
}

.modal-content {
  display: flex;
  flex-grow: 0;
  margin: auto 0;
  position: relative;
  width: 100%;
  min-width: 150px;
  max-width: 380px;
  background-color: rgba(255, 255, 255, .8);
  padding: 20px 25px;
  color: #333;
  box-sizing: border-box;
  font-size: 0.9rem;
  border-radius: 8px;
  transform: scale(50%);
  transition: transform .20s cubic-bezier(.6, -0.28, .74, .05);
}

.modal-popup.show .modal-content {
  transition: transform .20s cubic-bezier(.18, .89, .32, 1.28);
  transform: scale(100%);
}

.modal-popup .modal-content > div {
  width: 100%;
}

.modal-content a {
  color: #333;
}

.modal-content h2 {
  font-weight: normal;
  margin: 0 0 16px;
}

.modal-content .btn {
  margin: 0 0 0 16px;
  color: #333;
  background-color: rgba(255, 255, 255, .8);
}

.modal-content .btn:hover {
  background-color: rgba(255, 255, 255, 1);
}

.modal-content .objects {
  font-weight: normal;
}

.modal-content h2:first-letter {
  text-transform: uppercase;
}

.modal-content .buttons {
  display: flex;
  justify-content: end;
  margin: 20px 0 0;
}

.modal-content .btn.red {
  background-color: rgba(255, 0, 0, .3);
}

.modal-content .btn.red:hover {
  background-color: rgba(255, 0, 0, .5);
}
</style>