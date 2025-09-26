<template>
    <v-dialog v-model="isOpen" :width="width" :persistent="persistent">
        <v-card>
            <v-card-title class="d-flex justify-space-between align-center">
                <span>{{ title }}</span>
                <v-btn 
                    v-if="closable" 
                    variant="text" 
                    icon="mdi-close" 
                    @click="closeDialog">
                </v-btn>
            </v-card-title>
            <v-card-text>
                <slot></slot>
            </v-card-text>
            <v-card-actions>
                <v-spacer></v-spacer>
                <slot name="actions"></slot>
            </v-card-actions>
        </v-card>   
    </v-dialog>
</template>

<script setup>
import { ref } from 'vue';
import { watch } from 'vue';

const props = defineProps({
    title: {
        type: String,
        required: true
    },
    width: {
        type: [String, Number],
        default: 600
    },
    persistent: {
        type: Boolean,
        default: false
    },
    closable: {
        type: Boolean,
        default: true
    },
    modelValue: {
        type: Boolean,
        required: true
    }
});

// Data
const isOpen = ref(props.modelValue);

// Emits
const emit = defineEmits(['update:modelValue', 'close']);

// Watchers
watch(
    () => props.modelValue,
    (val) => (isOpen.value = val)
);

watch(isOpen, (val) => {
    if (!val) {
        emit('update:modelValue', false);
    }   
});

// Methods
function closeDialog() {
    isOpen.value = false;
    emit('close');
}
</script>
