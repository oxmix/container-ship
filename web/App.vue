<template>
  <h1 class="colorfulness">Container ship</h1>
  <nav>
    <div>
      <router-link to="/o/states">States</router-link>
      <router-link to="/o/nodes">Nodes</router-link>
      <router-link to="/o/hub">Hub</router-link>
      <a href="https://github.com/oxmix/container-ship/blob/master/README.md" target="_blank">Readme</a>
    </div>
  </nav>
  <div class="content">
    <router-view />
  </div>
</template>

<script>
export default {}
</script>

<style>
body {
  font-family: sans-serif;
  font-weight: 100;
  margin: 30px;
  color: #333;
}

a {
  color: #333;
}

a:hover {
  text-decoration: none;
}

ul {
  list-style: none;
  margin: 0;
  padding: 0;
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

body > h1 {
  text-align: center;
  font-weight: inherit;
}

nav,
.tabs-line {
  height: 32px;
  margin: 0 0 20px;
  position: relative;
  overflow: hidden;
  text-align: center;
}

nav:after {
  display: block;
  content: '';
  border-bottom: 1px solid #eee;
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
}

nav div {
  display: block;
  list-style: none;
  padding: 0;
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
  border: 1px solid #ddd;
  border-radius: 2px;
  margin: 10px;
}

legend {
  white-space: nowrap;
  padding: 5px 10px;
  color: #333;
  position: relative;
}

legend:before {
  position: absolute;
  content: '';
  height: 6px;
  border-left: 1px solid #ddd;
  left: 0;
  top: 10px;
}

legend:after {
  position: absolute;
  content: '';
  height: 6px;
  border-right: 1px solid #ddd;
  right: 0;
  top: 10px;
}

legend > span {
  color: grey;
  margin-left: 10px;
  font-size: .8rem;
  border: 1px solid #ddd;
  border-radius: 10px;
  padding: 2px 5px;
  position: relative;
  top: -1px;
}

fieldset:hover,
fieldset:hover > legend:before,
fieldset:hover > legend:after {
  border-color: #888;
}
</style>