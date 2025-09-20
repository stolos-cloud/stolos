<template>
    <v-text-field 
        v-model="model"
        :label="Textfield.label"
        :type="Textfield.type"
        :rules="Textfield.rules"
        :readonly="Textfield.readonly" 
        :disabled="Textfield.readonly"
        :min="Textfield.minDate"
        :max="Textfield.maxDate"
        class="mt-4"
    ></v-text-field>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
    Textfield: {
        type: Object,
        required: true
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

// model: {
//             get() {
//                 return this.localTextBox._unformatedText || ''
//             },
//             set(value) {
//                 if(this.localTextBox){
//                     this.localTextBox._unformatedText = value;
//                     this.TextBox.unformatedText = value;
//                     this.localTextBox._text = value;
//                     this.TextBox.text = value;
//                     this.$emit('changed', value);
//                 }
//             }
//         },
</script>