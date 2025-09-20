<template>
  <v-radio-group
    v-model="model"
    :rules="RadioGroup.rules"
    :disabled="RadioGroup.disabled"
    hide-details="auto"
    class="mt-4"
  >
    <div class="flex flex-col">
        <div class="flex items-center ga-3">
          <span class="text-body-1">{{ RadioGroup.label }}</span>
          <span v-if="RadioGroup.required" class="text-red">*</span>
        </div>
        <span class="text-caption text-grey">{{ RadioGroup.precision }}</span>
    </div>
    <v-radio
      v-for="option in RadioGroup.options"
      :key="option.value"
      :label="option.label"
      :value="option.value"
    />
  </v-radio-group>
</template>

<script setup>
import { computed } from 'vue';
const props = defineProps({
    RadioGroup: {
        type: Object,
        required: true
    }
});

const emit = defineEmits(['changed']);

const model = computed({
  get() {
    return props.RadioGroup.value;
  },
  set(value) {
    props.RadioGroup.change(value);
    emit('changed', value);
  }
});

</script>
