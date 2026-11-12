import prettierConfig from "eslint-config-prettier"
import pluginVue from "eslint-plugin-vue"
import tseslint from "typescript-eslint"
import globals from "globals"
import js from "@eslint/js"

export default tseslint.config(
    { ignores: ["dist", "bindings", "node_modules"] },
    js.configs.recommended,
    ...tseslint.configs.recommended,
    ...pluginVue.configs["flat/recommended"],
    {
        files: ["**/*.{ts,vue}"],
        languageOptions: {
            ecmaVersion: "latest",
            sourceType: "module",
            globals: globals.browser,
            parserOptions: {
                parser: tseslint.parser,
                extraFileExtensions: [".vue"],
            },
        },
        rules: {
            "@typescript-eslint/no-unused-vars": [
                "error",
                { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
            ],
            "vue/multi-word-component-names": "off",
        },
    },
    {
        files: ["src/components/ui/**/*.vue"],
        rules: { "vue/require-default-prop": "off" },
    },
    prettierConfig,
)
