<template>
    <v-file-input
        v-model="model"
        :label="FileInput.label"
        :rules="FileInput.rules"
        :readonly="FileInput.readonly" 
        :disabled="FileInput.readonly"
        :accept="FileInput.accept"
        show-size
        prepend-icon=""
        prepend-inner-icon="mdi-paperclip"
        hide-details="auto"
        variant="outlined"
        class="my-4"
    >
        <template #label>
            {{ FileInput.label }}
            <span v-if="FileInput.required" class="text-red">*</span>
        </template>
    </v-file-input>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
    FileInput: {
        type: Object,
        required: true
    }
});

const emit = defineEmits(['changed']);

const model = computed({
  get() {
    return props.FileInput.value;
  },
  set(value) {
    props.FileInput.change(value);
    emit('changed', value);
  }
});
</script>
