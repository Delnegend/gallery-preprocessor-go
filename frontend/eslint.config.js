import hagemanto from "eslint-plugin-hagemanto";
import pluginVue from "eslint-plugin-vue";

/** @type {import('eslint').Linter.Config[]} */
export default [
	{ files: ["**/*.{ts,vue}"] },
	{ ignores: ["**/*.config.js", "wailsjs/**/*", "dist/**/*"] },
	...hagemanto({ vueConfig: pluginVue.configs["flat/essential"], enableJsx: false }),
];