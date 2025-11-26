<template>
    <v-chip
        :color="color"
        v-bind="$attrs"
        class="mr-1 my-1"
        label
        size="small"
    >
        <template #prepend>
            <span :class="hasPrepend ? 'mr-1' : ''">
                <slot name="prepend"></slot>
            </span>
        </template>
        <slot></slot>
    </v-chip>
</template>

<script setup>
import { useSlots } from 'vue';

defineProps({
    color: {
        type: String,
        default: 'primary'
    }
});

const slots = useSlots();
const hasPrepend = !!slots.prepend;
</script>