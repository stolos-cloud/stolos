<template>
    <v-app>
        <router-view />
        <BaseOverlay v-model="overlay" />
        <BaseNotification v-model="notification.visible" :text="notification.text" :type="notification.type" />
    </v-app>
</template>

<script setup>
import { useStore } from 'vuex';
import { useTheme } from 'vuetify';
import { useI18n } from 'vue-i18n';
import { onMounted } from 'vue';
import { getAvailableGCPResources } from '@/services/provisioning.service';
import { GlobalNotificationHandler } from '@/composables/GlobalNotificationHandler';
import { GlobalOverlayHandler } from '@/composables/GlobalOverlayHandler';

const store = useStore();
const theme = useTheme();
const i18n = useI18n();
const { notification } = GlobalNotificationHandler();
const { overlay } = GlobalOverlayHandler();

const savedTheme = store.getters['user/getTheme']
const savedLanguage = store.getters['user/getLanguage']

theme.change(savedTheme);
i18n.locale.value = savedLanguage;

onMounted(async () => {
    await getAvailableGCPResources().then(gcpResources => {
        store.dispatch('referenceLists/setCloudResources', gcpResources);
    });
});
</script>

<style>
body {
    font-size: small;
}
</style>