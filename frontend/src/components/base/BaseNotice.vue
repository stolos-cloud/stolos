<template>
    <v-alert :type="type" :title="title" :text="text" :density="density" variant="tonal"
        :class="`alert-border-${type} mt-2`" />
</template>
<script setup>
defineProps({
    type: {
        type: String,
        default: 'info'
    },
    title: {
        type: String,
        default: ''
    },
    text: {
        type: String,
        default: ''
    },
    density: {
        type: String,
        default: 'default'
    }
});
</script>

<style>

.v-alert__content {
    color: rgba(var(--v-theme-text), 1);
}

.v-alert-title {
    color: rgba(var(--v-theme-title), 1);
}

.alert-border-info {
    border: 1px solid rgba(var(--v-theme-info), 0.5);
}

.alert-border-success {
    border: 1px solid rgba(var(--v-theme-success), 0.5);
}

.alert-border-warning {
    border: 1px solid rgba(var(--v-theme-warning), 0.5);
}

.alert-border-error {
    border: 1px solid rgba(var(--v-theme-error), 0.5);
}
</style>