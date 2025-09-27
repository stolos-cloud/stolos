<template>
    <v-alert
        :type="type"
        :title="title"
        :text="text"
        :density="density"
        variant="tonal"
        class="mt-2"
    />
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