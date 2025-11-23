<template>
    <v-app>
        <router-view />
        <BaseOverlay v-model="overlay" />
        <BaseNotification v-model="notification.visible" :text="notification.text" :type="notification.type" :closable="notification.closable" />
    </v-app>
</template>

<script setup>
import { useStore } from 'vuex';
import { useTheme } from 'vuetify';
import { useI18n } from 'vue-i18n';
import { onMounted, onBeforeUnmount } from 'vue';
import { getAvailableGCPResources } from '@/services/provisioning.service';
import { getScaffolds } from './services/scaffolds.service';
import { GlobalNotificationHandler } from '@/composables/GlobalNotificationHandler';
import { GlobalOverlayHandler } from '@/composables/GlobalOverlayHandler';
import wsEventService from '@/services/wsEvent.service';

const store = useStore();
const theme = useTheme();
const i18n = useI18n();
const { notification } = GlobalNotificationHandler();
const { overlay } = GlobalOverlayHandler();

const savedTheme = store.getters['user/getTheme']
const savedLanguage = store.getters['user/getLanguage']
const isUserAuthenticated = store.getters['user/isAuthenticated']

theme.change(savedTheme);
i18n.locale.value = savedLanguage;

onMounted(async () => {
    wsEventService.connect();
    await getAvailableGCPResources().then(gcpResources => {
        store.dispatch('referenceLists/setCloudResources', gcpResources);
    });
    if(isUserAuthenticated) {
        await getScaffolds().then(scaffolds => {
            store.dispatch('referenceLists/setScaffolds', scaffolds);
        });
    }
});

onBeforeUnmount(() => {
    wsEventService.disconnect();
});
</script>

<style>
body {
    font-size: small;
}
</style>
