<template>
    <v-chip
        :color="color"
        v-bind="$attrs"
        class="mr-1 my-1"
        label
        size="small"
    >
        <template #prepend>
            <span class="mr-1">
                <slot name="prepend" class="mr-1"></slot>
            </span>
        </template>
        <slot></slot>
    </v-chip>
</template>

<script setup>
defineProps({
    color: {
        type: String,
        default: 'primary'
    }
});

</script>