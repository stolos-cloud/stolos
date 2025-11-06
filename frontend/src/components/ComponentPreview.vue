<template>
    <v-sheet class="mt-4 border rounded">
        <v-toolbar flat>
            <v-toolbar-title>{{ title }}</v-toolbar-title>
            <template v-if="buttons.length > 0">
                <BaseButton class="ml-2" v-for="(btn, i) in buttons" :key="i" :icon="btn.icon" :text="btn.text"
                    :tooltip="btn.tooltip" :disabled="btn.disabled" elevation="2" @click="btn.click" />
            </template>
        </v-toolbar>
        <div v-if="$slots.controls" class="pa-4">
            <slot name="controls"></slot>
        </div>
        <v-divider></v-divider>
        <div :class="contentClass">
            <slot></slot>
        </div>
        <v-expand-transition>
            <div v-if="showCode">
                <v-sheet class="pa-3 border bg-grey-darken-4">
                    <v-code class="bg-transparent">
                        {{ code }}
                    </v-code>
                </v-sheet>
            </div>
        </v-expand-transition>
    </v-sheet>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n();

const props = defineProps({
    title: String,
    code: String,
    fullWidth: { type: Boolean, default: false }
})

// State
const showCode = ref(false)

// Computed
const buttons = computed(() => [
    {
        icon: "mdi-content-copy",
        tooltip: t('actionButtons.copyCode'),
        text: t('actionButtons.copyCode'),
        click: copyCode
    },
    {
        icon: showCode.value ? "mdi-eye-off" : "mdi-code-tags",
        tooltip: showCode.value ? t('actionButtons.hideCode') : t('actionButtons.showCode'),
        text: showCode.value ? t('actionButtons.hideCode') : t('actionButtons.showCode'),
        click: () => { showCode.value = !showCode.value }
    }
]);
const contentClass = computed(() =>
    props.fullWidth ? 'my-5 pa-2' : 'd-flex justify-center my-5 mx-2'
)

// Methods
async function copyCode() {
    await navigator.clipboard.writeText(props.code)
}
</script>
