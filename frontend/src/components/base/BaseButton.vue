<template>
    <v-btn
        :color="color"
        :variant="variant"
        :size="size"
        :disabled="props.disabled"
        :elevation="elevation"
        class="d-flex align-center justify-center text-none"
    >
        <v-tooltip v-if="showTooltip" activator="parent">{{ props.tooltip }}</v-tooltip>
        <v-icon v-if="props.icon" :start="showText">{{ props.icon }}</v-icon>
        <span v-if="showText">{{ props.text }}</span>
    </v-btn>
</template>

<script setup>
import { computed } from 'vue';
import { useDisplay } from 'vuetify/lib/composables/display';

// Props
const props = defineProps({
    text: {
        type: String,
        default: ""
    },
    tooltip: {
        type: String,
        default: ""
    },
    icon: {
        type: String,
        default: ""
    },
    color: {
        type: String,
        default: "primary"
    },
    variant: {
        type: String,
        default: "flat"
    },
    size: {
        type: String,
        default: "small"
    },
    elevation: {
        type: [String, Number],
        default: 0
    },
    disabled: {
        type: Boolean,
        default: false
    }
});

const display = useDisplay();
const showText = computed(() => !props.icon || display.mdAndUp.value);
const showTooltip = computed(() => !!props.icon && !display.mdAndUp.value && !!props.tooltip);
</script>

<style scoped>
.v-btn:hover {
    background-color: #bf5b26 !important;
}

.v-btn.v-btn--disabled {
  background-color: #ff9248 !important;
  opacity: 0.5 !important;
}
</style>