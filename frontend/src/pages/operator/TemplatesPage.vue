<template>
    <PortalLayout>
        <CCLabelBar 
            :title="$t('templates.title')"
            :subheading="$t('templates.subheading')"
        />
    </PortalLayout>
</template>

<script setup>
import PortalLayout from '@/components/layouts/PortalLayout.vue';
import CCLabelBar from '@/components/base/BaseLabelBar.vue';

</script>
