<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('cloudProvider.title')"
            :subheading="$t('cloudProvider.subheading')"
            :actions="actions"
        />
    </PortalLayout>
</template>

<script setup>
import PortalLayout from '@/components/layouts/PortalLayout.vue';
import BaseLabelBar from '@/components/base/BaseLabelBar.vue';

const actions = [
    {
        text: "Add configuration",
        onClick: () => alert('Add configuration clicked')
    }
];

//Methods

</script>
