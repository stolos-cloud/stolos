<template>
    <div class="baseLabelBar">
        <div class="d-flex align-center justify-space-between py-4">
            <BaseTitle  
                :level="2"
                :title="title"
                :subheading="subheading"
            />
            <div v-if="actions && actions.length" class="d-flex align-center">
                <BaseButton 
                    v-for="(action, index) in actions" 
                    class="ml-2"
                    :key="index"
                    :text="action.text"
                    :color="action.color || 'primary'"
                    :variant="action.variant || 'elevated'"
                    :icon="action.icon"
                    :size="action.size || 'small'"
                    :disabled="action.disabled"
                    @click="action.onClick"
                />
            </div>
        </div>
        <v-divider></v-divider>
    </div>
</template>

<script setup>
import BaseTitle from '@/components/base/BaseTitle.vue';
import BaseButton from '@/components/base/BaseButton.vue';

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