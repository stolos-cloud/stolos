<template>
    <v-expansion-panels class="my-4" v-bind="panelProps">
        <v-expansion-panel :disabled="disabled" class="border">
            <v-expansion-panel-title class="px-4 py-2">
                <BaseTitle :level="6" :title="title" />
                <slot name="actions"></slot>
            </v-expansion-panel-title>
            <v-expansion-panel-text>
                <slot></slot>
            </v-expansion-panel-text>
        </v-expansion-panel>
    </v-expansion-panels>
</template>

<script setup>
defineProps({
    title: { type: String, required: true },
    disabled: { type: Boolean, default: false },
    panelProps: { type: Object, default: () => ({}) }
});
</script>