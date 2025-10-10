module.exports = {
    root: true,
    env: {
        node: true,
    },
    extends: ['plugin:vue/vue3-essential', 'eslint:recommended'],
    rules: {
        'vue/no-multiple-template-root': 'off',
        'vue/require-name': 'off',
        'vue/multi-word-component-names': 'off',
        'vue/valid-v-slot': 'off',
    },
};
