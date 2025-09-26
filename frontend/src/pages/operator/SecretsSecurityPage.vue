<template>
    <PortalLayout>
        <BaseLabelBar 
            :title="$t('secretsSecurity.title')"
            :subheading="$t('secretsSecurity.subheading')"
        />
    </PortalLayout>
</template>

<script setup>
import PortalLayout from '@/components/layouts/PortalLayout.vue';
import BaseLabelBar from '@/components/base/BaseLabelBar.vue';

</script>
