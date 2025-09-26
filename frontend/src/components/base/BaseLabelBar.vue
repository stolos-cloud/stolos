<template>
    <div class="d-flex align-center justify-space-between px-4 py-2">
        <CCTitle 
            :level="2"
            :title="title"
            :subheading="subheading"
        />
        <div v-if="actions && actions.length" class="d-flex align-center">
            <CCButton 
                v-for="(action, index) in actions" 
                class="ml-2"
                :key="index"
                :text="action.text"
                :color="action.color || 'primary'"
                :variant="action.variant || 'elevated'"
                :icon="action.icon"
                :size="action.size || 'small'"
                @click="action.onClick"
            />
        </div>
    </div>
</template>

<script setup>
import CCTitle from '@/components/base/BaseTitle.vue';
import CCButton from '@/components/base/BaseButton.vue';

// Props
defineProps({
    title: {
        type: String,
        required: true
    },
    subheading: {
        type: String,
        default: ""
    },
    actions: {
        type: Array,
        default: () => [],
        validator: (arr) => arr.length <= 2
    }
});
</script>