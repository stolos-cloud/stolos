<template>
    <v-dialog v-model="isOpen" :width="width" :persistent="persistent" content-class="elevation-8 rounded-lg"
        style="backdrop-filter: blur(4px);">
        <v-card class="border rounded-lg">
            <template v-slot:title>
                <div class="d-flex align-center my-2">
                    <BaseTitle :level="5" :title="title" />
                    <v-spacer></v-spacer>
                    <v-btn v-if="closable" variant="text" icon="mdi-close" size="small" @click="closeDialog" rounded
                        class="close-btn" />
                </div>
            </template>
            <v-divider></v-divider>
            <v-card-text class="my-2">
                <slot></slot>
            </v-card-text>
            <template v-if="hasActions">
                <v-card-actions class="mx-4">
                    <slot name="actions"></slot>
                </v-card-actions>
            </template>
        </v-card>
    </v-dialog>
</template>

<script setup>
import { ref, watch, computed, useSlots } from 'vue';

const slots = useSlots();

const props = defineProps({
    title: {
        type: String,
        required: true
    },
    width: {
        type: [String, Number],
        default: 700
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

// Computed
const hasActions = computed(() => !!slots.actions);

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

<style scoped>
.close-btn:hover,
.close-btn:active {
    background-color: rgba(var(--v-theme-primary));
    color: white;
}
</style>