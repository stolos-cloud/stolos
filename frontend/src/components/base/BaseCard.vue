<template>
    <v-card class="border my-4" elevation="0" :color="colorDialog">
        <template v-if="$slots.title">
            <v-card-title class="d-flex align-center py-2">
                <slot name="title"></slot>
            </v-card-title>
        </template>
        <v-divider v-if="$slots.title && $slots.default" />
        <v-card-text v-if="$slots.default">
            <slot></slot>
        </v-card-text>
    </v-card>
</template>

<script setup>
defineProps({
    colorDialog: {
        type: String,
        default: 'dialog'
    }
})
</script>