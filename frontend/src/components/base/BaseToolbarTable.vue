<template>
    <div>
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
        <v-text-field 
            v-if="modelValue !== undefined"
            v-model="localSearch" 
            :label="$t('actionButtons.search')" 
            prepend-inner-icon="mdi-magnify" 
            variant="outlined" 
            hide-details 
            single-line 
            dense 
            class="pa-3"
        />
    </div>
</template>

<script setup>
import { ref, watch } from 'vue';

const props = defineProps({
    title: {
        type: String,
        required: true
    },
    buttons: {
        type: Array,
        default: () => []
    },
    modelValue: {
        type: String,
        default: undefined
    }
});

// state
const localSearch = ref(props.modelValue);

// Emits
const emit = defineEmits(['update:modelValue']);

// Watchers
watch(localSearch, (val) => emit('update:modelValue', val));
watch(() => props.modelValue, (val) => {
    if (val !== localSearch.value) localSearch.value = val;
});
</script>