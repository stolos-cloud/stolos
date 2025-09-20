<template>
    <v-navigation-drawer
      :width="sidebarWidth"
      app
      permanent
      :elevation="props.elevation"
    >
      <v-list-item class="sidebar-top" :style="{ height: props.toolbarHeight + 'px' }">
        <template #prepend>
            <v-icon>mdi-rocket</v-icon>
        </template>
        <v-list-item-title>{{ $t('applicationName') }}</v-list-item-title>
        <v-list-item-subtitle>Admin</v-list-item-subtitle> <!-- Role  to be added-->
      </v-list-item>

      <v-divider></v-divider>

      <v-list density="compact" nav>
        <v-list-item
          v-for="item in menuItems"
          :key="item.title"
          :to="item.route || '/'"
          link
          rounded="0"
          active-class="active-nav"
        >
          <template #prepend>
            <v-icon>{{ item.icon }}</v-icon>
          </template>

          <v-list-item-title v-if="drawer">
            {{ item.title }}
          </v-list-item-title>
        </v-list-item>
      </v-list>
    </v-navigation-drawer>
</template>

<script setup>
import { computed } from 'vue';
import { useI18n } from 'vue-i18n'

const { t } = useI18n();
const role = 'operator'; // This should be dynamically set based on the logged-in user's role

// Props
const props = defineProps({
    drawer: {
        type: Boolean,
        required: true,
    },
    toolbarHeight: {
        type: Number,
        default: 64,
    },
    elevation: {
        type: Number,
        default: 5,
    },
});

// Data
const developerMenu = [
    { title: t('dashboard.title'), icon: 'mdi-view-dashboard', route: '/dashboard' },
    { title: t('applications.title'), icon: 'mdi-apps', route: '/applications' },
    { title: t('deployments.title'), icon: 'mdi-rocket-launch', route: '/deployments' },
    { title: t('observability.title'),  icon: 'mdi-chart-line', route: '/observability' },
    { title: t('scaling.title'), icon: 'mdi-trending-up', route: '/scaling' },
    { title: t('workflows.title'), icon: 'mdi-file-tree', route: '/workflows' },
    { title: t('functionsVms.title'),  icon: 'mdi-desktop-classic', route: '/functions-vms' },
    { title: t('secrets.title'),  icon: 'mdi-key', route: '/secrets' }
];

const operatorMenu = [
    { title: t('dashboard.title'), icon: 'mdi-view-dashboard', route: '/dashboard' },
    { title: t('provisioning.title'), icon: 'mdi-server', route: '/provisioning' },
    { title: t('cloudProvider.title'), icon: 'mdi-cloud', route: '/cloud-provider' },
    { title: t('templates.title'), icon: 'mdi-layers-triple', route: '/templates' },
    { title: t('secretsSecurity.title'),  icon: 'mdi-shield', route: '/secrets-security' },
    // { title: t('observability.title'), icon: 'mdi-chart-line', route: '/observability' },
    // { title: t('policiesCompliance.title'), icon: 'mdi-file-document-check', route: '/policies-compliance' },
    // { title: t('gitopsSynchronization.title'),  icon: 'mdi-git', route: '/gitops-synchronization' }

];

// Computed
const sidebarWidth = computed(() => (props.drawer ? 200 : 60));
const menuItems = computed(() => role === 'developer' ? developerMenu : operatorMenu);
</script>

<style scoped>
.active-nav {
  border-left: 4px solid #f97316;
  color: #ffffff;
  font-weight: 600;
  text-shadow: 0 0 2px rgba(255, 255, 255, 0.7);
}
</style>