<template>
    <div class="view-details-template-dialog">
        <BaseDialog v-model="isOpen" title="$t('templateDefinitions.dialogs.viewDetailsTemplate.title')" closable>
            <p>TBD</p>
        </BaseDialog>
    </div>
</template>

<script setup>
import { ref, watch } from "vue";

const props = defineProps({
    modelValue: {
        type: Boolean,
        required: true
        
    },
    template: {
        type: Object
    }
});

// State
const isOpen = ref(props.modelValue);

// Emits
const emit = defineEmits(['update:modelValue']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => emit('update:modelValue', val));

// Methods

</script>