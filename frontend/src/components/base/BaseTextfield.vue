<template>
    <v-text-field 
        v-model="model"
        :label="Textfield.label"
        :type="Textfield.type"
        :rules="Textfield.rules"
        :readonly="Textfield.readonly" 
        :disabled="Textfield.readonly"
        :min="Textfield.min"
        :max="Textfield.max"
        :append-inner-icon="iconAction"
        @click:append-inner="emit('clickIcon')"
        hide-details="auto"
        variant="outlined"
        class="my-4"
    >
        <template #label>
            {{ Textfield.label }}
            <span v-if="Textfield.required" class="text-red">*</span>
        </template>
    </v-text-field>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
    Textfield: {
        type: Object,
        required: true
    },
    iconAction: {
        type: String,
        default: ''
    }
});
const emit = defineEmits(['changed']);

const model = computed({
  get() {
    return props.Textfield.value;
  },
  set(value) {
    props.Textfield.change(value);
    emit('changed', value);
  }
});
</script>