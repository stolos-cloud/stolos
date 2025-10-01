<template>
    <v-checkbox
        v-model="model"
        :label="Checkbox.label"
        :rules="Checkbox.rules"
        :readonly="Checkbox.readonly" 
        :disabled="Checkbox.readonly"
        hide-details="auto"
        color="primary"
        density="compact"
    ></v-checkbox>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
    Checkbox: {
        type: Object,
        required: true
    }
});

const emit = defineEmits(['changed']);

const model = computed({
  get() {
    return props.Checkbox.value;
  },
  set(value) {
    props.Checkbox.change(value);
    emit('changed', value);
  }
});

</script>