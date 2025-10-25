<template>
    <v-overlay
        v-model="internalVisible"
        class="d-flex align-center justify-center"
        persistent
    >
        <v-progress-circular indeterminate />
    </v-overlay>
</template>

<script setup>
import { ref, watch } from 'vue';

const props = defineProps({
    modelValue: {
        type: Boolean,
        required: true
    }
});

// State
const internalVisible = ref(props.modelValue);

// Emits
const emit = defineEmits(['update:modelValue']);

// Watchers
watch(() => props.modelValue, (val) => {
    internalVisible.value = val;
});
watch(internalVisible, (val) => {
    emit('update:modelValue', val);
});
</script>