<script setup>
import {ref, watch} from "vue";

const show = ref(false)
const display = ref('none')

const props = defineProps({
  open: {
    type: Boolean,
    required: true
  }
})

const emit = defineEmits(['close'])

watch(() => props.open, v => {
  if (v === null) {
    return close()
  }

  document.onkeydown = e => {
    if (e.target.type === 'textarea') {
      return
    }

    // change focus from the element that triggered popup
    if (e.key === 'Escape' || e.keyCode === 27 || e.key === 'Enter' || e.keyCode === 13) {
      e.preventDefault()
    }

    if (e.key === 'Escape' || e.keyCode === 27) {
      close()
    }
  }
  display.value = 'flex'
  setTimeout(() => show.value = true, 10)
})

function close() {
  document.onkeydown = null
  show.value = false
  setTimeout(() => display.value = 'none', 300)
  emit('close')
}
</script>
<template>
  <div :class="[$style['modal-popup'], {[$style.show]: show}]" :style="{display: display}">
    <div :class="$style['modal-content']">
      <div>
        <div :class="$style.close" @click="close" />
        <slot />
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
  transition: visibility 0s .3s, opacity .3s linear;
  visibility: hidden;
  opacity: 0;
}

.modal-popup.show {
  transition: opacity .3s linear;
  visibility: visible;
  opacity: 1;
}

.modal-content {
  display: flex;
  flex-grow: 0;
  margin: auto 0;
  position: relative;
  width: 100%;
  min-width: 300px;
  max-width: 600px;
  background-color: rgba(255, 255, 255, .8);
  padding: 20px 25px;
  color: #333;
  box-sizing: border-box;
  font-size: 0.9rem;
  border-radius: 8px;
  transform: scale(50%);
  transition: transform .3s cubic-bezier(.6, -0.28, .74, .05);
}

.modal-popup.show .modal-content {
  transition: transform .3s cubic-bezier(.18, .89, .32, 1.28);
  transform: scale(100%);
}

.modal-popup .modal-content > div {
  width: 100%;
}

.modal-content .close {
  position: absolute;
  right: -30px;
  top: -30px;
  font-size: 0;
  cursor: pointer;
  border-radius: 8px;
  background-color: transparent;
  transition: background-color .15s ease;
}

html:not([screen="touch"]) .modal-content .close:hover {
  background-color: var(--bg-2);
}

.modal-content .close:after {
  content: '';
  display: inline-block;
  width: 25px;
  height: 25px;
  -webkit-mask: var(--icon-cross) no-repeat;
  mask: var(--icon-cross) no-repeat;
  background-color: white;
}

.modal-content a {
  color: #333;
}

.modal-content h2 {
  margin-top: 0;
}
</style>
