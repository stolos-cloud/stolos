<template>
  <v-snackbar
    v-model="model"
    :timeout="computedTimeout"
    :color="snackbarColor"
    location="bottom right"
    class="rounded px-4 py-2"
    elevation="8"
  >
    <span>{{ text }}</span>
    <template v-slot:actions v-if="closable">
        <v-btn
            icon="mdi-close"
            size="small"
            variant="text"
            @click="model = false"
        />
    </template>
  </v-snackbar>
</template>

<script setup>
import { computed } from "vue";

// Props
const props = defineProps({
  modelValue: {
    type: Boolean,
    required: true,
  },
  text: {
    type: String,
    required: true,
  },
  type: {
    type: String,
    default: "info",
  },
  timeout: {
    type: Number,
    default: 4000,
  },
  closable: {
    type: Boolean,
    default: false,
  }
});

// Emits
const emit = defineEmits(["update:modelValue"]);

// Computed
const model = computed({
  get: () => props.modelValue,
  set: (val) => emit("update:modelValue", val),
});

const snackbarColor = computed(() => {
  switch (props.type) {
    case "success":
      return "success";
    case "error":
      return "error";
    case "warning":
      return "warning";
    default:
      return "info";
  }
});

const computedTimeout = computed(() =>
  props.closable ? -1 : props.timeout
);
</script>
