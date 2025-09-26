<template>
    <div class="d-flex flex-column">
        <component 
            :is="`h${level}`"
            :class="['font-weight-bold']"
        >
            {{ title }}
        </component>
        <component 
            v-if="subheading"
            :is="`p`"
            :class="[`text-${subheadingSize}`, `text-medium-emphasis`]"
        >
            {{ subheading }}
        </component>
    </div>
</template>

<script setup>
// Props
defineProps({
    title: {
        type: String,
        required: true
    },
    level: {
        type: Number,
        default: 6,
        validator: (value) => [1, 2, 3, 4, 5, 6].includes(value)
    },
    subheading: {
        type: String,
        default: ""
    },
    subheadingSize: {
        type: String,
        default: 'body-2'
    }
});
</script>