<template>
    <PortalLayout>
        <CCLabelBar 
            :title="$t('secretsSecurity.title')"
            :subheading="$t('secretsSecurity.subheading')"
        />
    </PortalLayout>
</template>

<script setup>
import PortalLayout from '@/components/layouts/PortalLayout.vue';
import CCLabelBar from '@/components/base/BaseLabelBar.vue';

</script>
