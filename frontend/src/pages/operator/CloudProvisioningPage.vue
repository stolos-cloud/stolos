<template>
    <PortalLayout>
        <BaseLabelBar 
            :title="$t('provisioning.cloud.title')"
            :subheading="$t('provisioning.cloud.subheading')"
        />
        
        <v-overlay class="d-flex align-center justify-center" v-model="overlay" persistent>
            <v-progress-circular
                indeterminate
            ></v-progress-circular>
        </v-overlay>
    </PortalLayout>
</template>

<script setup>
import PortalLayout from '@/components/layouts/PortalLayout.vue';
import BaseLabelBar from '@/components/base/BaseLabelBar.vue';
import { ref } from 'vue';

const overlay = ref(false);

</script>
