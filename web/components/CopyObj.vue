<script setup>
import {useCssModule} from "vue";

const style = useCssModule()

const props = defineProps({
  payload: {
    type: [String, Function],
    required: true
  },
  secret: {
    type: Boolean,
    required: false,
    default: false
  },
  length: {
    type: Number,
    required: false,
    default: 13
  },
  slotShow: {
    type: Boolean,
    required: false,
    default: false
  }
})

const click = elm => {
  const act = content => {
    copy(content).then(() => {
      elm.target.classList.add(style.blink)
      setTimeout(() => elm.target.classList.remove(style.blink), 500)
    })
  }

  if (typeof props.payload === 'function') {
    let res = props.payload()
    if (res instanceof Promise) {
      return res.then(p => act(p))
    } else {
      return act(res)
    }
  }
  act(props.payload)
}

function copy(payload) {
  return new Promise((resolve, reject) => {
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(payload).then(resolve, (e) => {
        console.error('navigator.clipboard:', e)
        reject()
      })
      return
    }

    if (window.clipboardData && window.clipboardData.setData) {
      try {
        window.clipboardData.setData('Text', payload)
        resolve()
      } catch (e) {
        console.error('window.clipboardData:', e)
        reject()
      }
      return
    }

    if (document.queryCommandSupported && document.queryCommandSupported('copy')) {
      let textarea = document.createElement('textarea')
      textarea.textContent = payload
      textarea.style.opacity = '0'
      textarea.style.position = 'fixed'
      document.body.appendChild(textarea)
      textarea.select()
      try {
        document.execCommand('copy')
        resolve()
      } catch (e) {
        console.error('document.execCommand copy:', e)
        reject()
      } finally {
        document.body.removeChild(textarea)
      }
    }
  })
}
</script>
<template>
  <span
    :class="[style['copy-obj'], {[style['copy-obj-secret']]: secret}]"
    title="Copy to clipboard" data-placement="left"
    @click="click"
  >
    <span
      v-if="typeof payload === 'string' && !slotShow"
      v-text="secret ? payload.substring(0, props.length)+'...' : payload"
    />
    <span v-else-if="$slots.default"><slot /></span>
  </span>
</template>
<style module>
.copy-obj {
  display: inline-flex;
  cursor: copy;
  padding: 2px 5px;
  background-color: var(--bg-1);
  border-radius: 3px;
  text-decoration: none;
  white-space: nowrap;
  align-items: center;
  font-family: Consolas, monospace;
  font-size: .8rem;
}

.copy-obj:before {
  content: '';
  display: inline-block;
  -webkit-mask: var(--icon-copy) no-repeat 2px/12px;
  mask: var(--icon-copy) no-repeat 2px/12px;
  background-color: var(--text);
  width: 14px;
  height: 12px;
  min-width: 14px;
  vertical-align: middle;
}

@keyframes blink {
  0% {
    background-color: inherit;
    box-shadow: 0 0 3px transparent;
  }
  50% {
    background-color: var(--bg-2);
    box-shadow: 0 0 3px var(--bg-2);
  }
  100% {
    background-color: inherit;
    box-shadow: 0 0 3px transparent;
  }
}

.blink {
  animation: .25s infinite blink;
}

.copy-obj.blink:before {
  -webkit-mask-image: var(--icon-ok);
  mask-image: var(--icon-ok);
}

html:not([screen="touch"]) .copy-obj:hover {
  background-color: var(--bg-2);
}

.copy-obj > span {
  pointer-events: none;
  user-select: none;
  margin-top: 1px;
  margin-left: 5px;
}

.copy-obj.copy-obj-secret > span {
  background: -webkit-linear-gradient(0deg, var(--text) 30%, transparent 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}
</style>