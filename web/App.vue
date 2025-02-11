<template>
  <h1 class="colorfulness">Container Ship</h1>
  <nav>
    <div>
      <router-link to="/states">States</router-link>
      <span class="dot" />
      <router-link to="/nodes">Nodes</router-link>
      <router-link to="/manifests">Manifests</router-link>
      <router-link to="/variables">Variables</router-link>
      <span class="dot" />
      <a href="https://github.com/oxmix/container-ship/blob/master/README.md" target="_blank">Readme</a>
    </div>
  </nav>
  <div class="content">
    <router-view />
  </div>
</template>
<style>
:root {
  --font: Arial, -apple-system, sans-serif;
  --text: rgba(255, 255, 255, .88);
  --text-light: rgba(255, 255, 255, .44);
  --bg-02: rgba(255, 255, 255, .02);
  --bg-03: rgba(255, 255, 255, .03);
  --bg-05: rgba(255, 255, 255, .05);
  --bg-1: rgba(255, 255, 255, .1);
  --bg-2: rgba(255, 255, 255, .2);
  --bg-3: rgba(255, 255, 255, .3);
  --icon-ok: url('data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path d="M13.743 3.756a.872.872 0 0 1 0 1.237l-7.317 7.25a.88.88 0 0 1-1.242 0L2.257 9.366a.872.872 0 0 1 0-1.237.88.88 0 0 1 1.242 0l2.306 2.261L12.5 3.756a.88.88 0 0 1 1.242 0Z"/></svg>');
  --icon-cross: url('data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 30 30"><path d="M8 6A2 2 0 0 0 6.6 9.4L12.2 15 6.6 20.6A2 2 0 1 0 9.4 23.4L15 17.8 20.6 23.4A2 2 0 1 0 23.4 20.6L17.8 15 23.4 9.4A2 2 0 0 0 22 6 2 2 0 0 0 20.6 6.6L15 12.2 9.4 6.6A2 2 0 0 0 8 6z"/></svg>');
  --icon-copy: url('data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="currentColor"><path d="M0 6.75C0 5.784.784 5 1.75 5h1.5a.75.75 0 0 1 0 1.5h-1.5a.25.25 0 0 0-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 0 0 .25-.25v-1.5a.75.75 0 0 1 1.5 0v1.5A1.75 1.75 0 0 1 9.25 16h-7.5A1.75 1.75 0 0 1 0 14.25Z"/><path d="M5 1.75C5 .784 5.784 0 6.75 0h7.5C15.216 0 16 .784 16 1.75v7.5A1.75 1.75 0 0 1 14.25 11h-7.5A1.75 1.75 0 0 1 5 9.25Zm1.75-.25a.25.25 0 0 0-.25.25v7.5c0 .138.112.25.25.25h7.5a.25.25 0 0 0 .25-.25v-7.5a.25.25 0 0 0-.25-.25Z"/></svg>');
  --icon-edit: url('data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 28 28" fill="currentColor"><path fill-rule="evenodd" d="M21.782 3.335a3 3 0 0 0-2.724 0c-.428.218-.8.591-1.23 1.023L4.677 17.508c-.354.355-.609.609-.818.897a4.5 4.5 0 0 0-.804 1.94C3 20.698 3 21.058 3 21.558V22.681c-.001.282-.002.628.098.937a2 2 0 0 0 1.284 1.284c.309.1.655.1.937.098H6.443c.5 0 .86 0 1.211-.055a4.5 4.5 0 0 0 1.941-.804c.288-.21.542-.464.896-.818l13.151-13.15c.432-.431.805-.803 1.023-1.231a3 3 0 0 0 0-2.724c-.218-.428-.591-.8-1.023-1.23l-.63-.63c-.43-.432-.802-.805-1.23-1.023Zm-1.816 1.782a1 1 0 0 1 .908 0c.106.054.237.168.819.75l.44.44c.582.582.696.713.75.819a1 1 0 0 1 0 .908c-.054.106-.168.237-.75.819l-.858.858-2.986-2.986.858-.858c.582-.582.713-.696.82-.75ZM16.875 8.14 6.142 18.872c-.425.425-.56.564-.665.708a2.5 2.5 0 0 0-.446 1.079c-.028.176-.03.369-.03.97v.971c-.001.197 0 .305.004.383v.012h.012c.078.005.186.005.383.005h.97c.602 0 .795-.003.971-.03a2.5 2.5 0 0 0 1.079-.447c.144-.105.283-.24.708-.665l10.733-10.733-2.986-2.986Z"/></svg>');
}

body {
  font-family: var(--font);
  min-width: 600px;
  margin: 26px;
  color: var(--text);
  background-color: #121212;
  letter-spacing: .2px;
  font-synthesis: none;
  text-rendering: optimizeLegibility;
  -moz-osx-font-smoothing: grayscale;
}

a {
  color: var(--text);
}

a:hover {
  text-decoration: none;
}

ul {
  list-style: none;
  margin: 0;
  padding: 0;
}

input[type="text"], textarea, select, button {
  font-family: var(--font);
  padding: 8px 12px;
  border: 0;
  border-radius: 8px;
  margin: 8px 0;
  background-color: rgba(255, 255, 255, .8);
  min-width: 360px;
  box-sizing: border-box;
}

input[type="text"]:focus, textarea:focus, button:focus {
  outline: 0;
  box-shadow: 0 0 6px #feab3a;
}

input[type="text"], textarea {
  font-family: monospace;
}

button {
  min-width: auto;
  cursor: pointer;
}

textarea {
  min-height: 150px;
}

button:hover {
  background-color: rgba(255, 255, 255, 1);
}

table {
  width: 100%;
}

table td {
  padding: 8px 10px;
  vertical-align: middle;
}

table thead td {
  color: var(--text-light);
  text-transform: uppercase;
  font-size: 10px;
}

table tbody tr:nth-child(odd) {
  background-color: var(--bg-03);
  border-radius: 2px;
}

table tbody tr:hover {
  background-color: var(--bg-1);
}

table tbody td {
  height: 35px;
}

.labels .label,
table .label {
  margin: 5px;
  border: 1px solid rgba(221, 221, 221, .5);
  border-radius: 10px;
  padding: 4px 8px;
  font-size: .8em;
  display: inline-block;
  white-space: nowrap;
}

.labels .label.red,
table .label.red {
  border-color: rgba(255, 69, 0, .5);
}

.labels .label.green,
table .label.green {
  border-color: rgba(154, 205, 50, .5);
}

table .remove,
table .edit {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 25px;
  height: 25px;
  cursor: pointer;
}

table .edit:after {
  background-color: var(--text);
  content: "";
  display: inline-block;
  width: 17px;
  height: 17px;
  margin-top: 1px;
  -webkit-mask: var(--icon-edit) no-repeat;
  mask: var(--icon-edit) no-repeat;
  transition: background-color .25s ease-in-out,
  transform .25s ease-in-out;
}

table .edit:hover:after {
  transform: scale(1.1);
}

table .remove:after {
  background-color: orange;
  content: "";
  display: inline-block;
  width: 18px;
  height: 18px;
  -webkit-mask: var(--icon-cross) no-repeat;
  mask: var(--icon-cross) no-repeat;
  transition: background-color .25s ease-in-out,
  transform .25s ease-in-out;
}

table .remove:hover:after {
  background-color: rgba(255, 69, 0, .8);
  transform: scale(1.1);
}

.colorfulness {
  color: #bf87ff;
  background-image: -webkit-linear-gradient(90deg, #f35626, #feab3a);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  -webkit-animation: hue 30s infinite linear;
}

@-webkit-keyframes hue {
  from {
    -webkit-filter: hue-rotate(0deg);
  }

  to {
    -webkit-filter: hue-rotate(-360deg);
  }
}

h1, h2, h3 {
  font-weight: inherit;
}

body > h1 {
  text-align: center;
}

nav {
  height: 32px;
  margin: 0 0 20px;
  position: relative;
  overflow: hidden;
  text-align: center;
}

nav:after {
  display: block;
  content: '';
  border-bottom: 1px solid var(--bg-1);
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
}

nav div {
  display: flex;
  justify-content: center;
  align-items: center;
  list-style: none;
  padding: 0 0 30px;
  margin: 0;
  white-space: nowrap;
  position: relative;
  overflow-x: auto;
  overflow-y: hidden;
  -webkit-overflow-scrolling: touch;
  -webkit-user-select: none;
  user-select: none;
  z-index: 1;
  font-size: 0;
  height: 32px;
}

nav .dot {
  border: 2px solid var(--text);
  border-radius: 50%;
  opacity: .3;
  margin-right: 10px;
}

nav a {
  height: 32px;
  margin-right: 10px;
  box-sizing: border-box;
  cursor: pointer;
  position: relative;
  display: inline-block;
  font-size: 1rem;
  text-decoration: none;
  padding: 6px 10px;
}

nav a:last-child {
  margin-right: 0;
}

nav a.router-link-active:after,
nav a:hover:after {
  display: block;
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  border-bottom: 1px solid #feab3a;
}

nav a:hover:after {
  border-color: #fec475;
}

fieldset {
  border: 1px solid var(--bg-1);
  border-radius: 2px;
  margin: 0 8px 16px;
  padding: 14px 24px 18px;
}

legend {
  white-space: nowrap;
  padding: 4px 14px;
  color: var(--text);
  position: relative;
}

legend:before {
  position: absolute;
  content: '';
  height: 6px;
  border-left: 1px solid var(--bg-1);
  left: 0;
  top: 10px;
}

legend:after {
  position: absolute;
  content: '';
  height: 6px;
  border-right: 1px solid var(--bg-1);
  right: 0;
  top: 10px;
}

legend > span {
  color: var(--text-light);
  margin-left: 10px;
  font-size: .8rem;
  border: 1px solid var(--bg-1);
  border-radius: 10px;
  padding: 2px 5px;
  position: relative;
  top: -1px;
}

fieldset:hover,
fieldset:hover > legend:before,
fieldset:hover > legend:after {
  border-color: var(--bg-3);
}
</style>