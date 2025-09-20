<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('cloudProvider.title')"
            :subheading="$t('cloudProvider.subheading')"
        />
    </PortalLayout>
</template>

<script setup>
import PortalLayout from '@/components/layouts/PortalLayout.vue';
import BaseLabelBar from '@/components/base/BaseLabelBar.vue';

</script>
