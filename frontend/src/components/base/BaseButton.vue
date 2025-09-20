<template>
    <v-btn
        :color="color"
        :variant="variant"
        :size="size"
        :disabled="props.disabled"
        class="d-flex align-center text-none"
    >
        <v-icon v-if="props.icon" :start="!!props.text">{{ props.icon }}</v-icon>
        <span>{{ props.text }}</span>
    </v-btn>
</template>

<script setup>
// Props
const props = defineProps({
    text: {
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
        default: "default"
    },
    disabled: {
        type: Boolean,
        default: false
    }
});
</script>

<style scoped>
.v-btn.v-btn--disabled {
  background-color: #ff9248 !important;
  opacity: 0.5 !important;
}
</style>