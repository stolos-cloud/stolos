<template>
    <v-toolbar flat>
        <v-toolbar-title>{{ title }}</v-toolbar-title>
        <template v-if="buttons.length > 0">
            <BaseButton 
                class="ml-2"
                v-for="(btn, i) in buttons"
                :key="i"
                :icon="btn.icon"
                :text="btn.text" 
                :tooltip="btn.tooltip"
                :disabled="btn.disabled" 
                elevation="2"
                @click="btn.click" 
            />
        </template>
    </v-toolbar>
</template>

<script setup>
defineProps({
    title: {
        type: String,
        required: true
    },
    buttons: {
        type: Array,
        default: () => []
    }
});
</script>