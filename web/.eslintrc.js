module.exports = {
	"env": {
		"browser": true,
		"es2021": true
	},
	"extends": [
		"eslint:recommended",
		"plugin:vue/vue3-recommended"
	],
	"parserOptions": {
		"ecmaVersion": "latest",
		"sourceType": "module"
	},
	"rules": {
        "vue/max-attributes-per-line": 0,
		"vue/singleline-html-element-content-newline": 0
	}
}
