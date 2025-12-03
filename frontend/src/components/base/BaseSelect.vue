<template>
    <v-select
        v-model="internalValue"
        :items="Select.options"
        :label="Select.label"
        :readonly="Select.readonly"
        :disabled="Select.readonly"
        :rules="Select.rules"
        item-title="label"
        item-value="value"
        variant="outlined"
        density="comfortable"
        hide-details="auto"
        class="my-4"
    >
        <template #label>
            {{ Select.label }}
            <span v-if="Select.required" class="text-red">*</span>
        </template>
    </v-select>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
    modelValue: [String, Number, Object],
    Select: {
        type: Object,
        required: true
    },
})

const emit = defineEmits(['update:modelValue'])

const internalValue = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})
</script>