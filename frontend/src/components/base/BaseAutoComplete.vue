<template>
    <v-autocomplete
        v-model="model"
        :items="AutoComplete.items"
        :label="AutoComplete.label"
        :disabled="AutoComplete.disabled"
        :multiple="AutoComplete.multiple"
        :rules="AutoComplete.rules"
        item-title="label"
        item-value="value"
        :no-data-text="AutoComplete.noDataText || $t('errors.noData')"
        hide-details="auto"
        variant="outlined"
        class="my-4"
        @update:search-input="emit('search', $event)"
        @click:clear="emit('cleared')"
    >
        <template #label>
            {{ AutoComplete.label }}
            <span v-if="AutoComplete.required" class="text-red">*</span>
        </template>
    </v-autocomplete>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
    AutoComplete: {
        type: Object,
        required: true
    }
});
const emit = defineEmits(['changed']);

const model = computed({
  get() {
    return props.AutoComplete.value;
  },
  set(value) {
    props.AutoComplete.change(value);
    emit('changed', value);
  }
});
</script>