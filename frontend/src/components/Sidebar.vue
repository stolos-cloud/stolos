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
        <template v-for="item in menuItems" :key="item.title">
          <v-list-group v-if="item.children" :prepend-icon="item.icon">
            <template #activator="{ props }">
              <v-list-item v-bind="props">
                <v-list-item-title>{{ $t(item.title) }}</v-list-item-title>
              </v-list-item>
            </template>

            <v-list-item
              v-for="child in item.children"
              :key="child.title"
              :to="child.route"
              link
              active-class="active-nav"
            >
              <v-list-item-title>{{ $t(child.title) }}</v-list-item-title>
            </v-list-item>
          </v-list-group>

          <v-list-item v-else :to="item.route || '/'" link rounded="0" active-class="active-nav">
            <template #prepend>
              <v-icon>{{ item.icon }}</v-icon>
            </template>
            <v-list-item-title v-if="drawer">{{ $t(item.title) }}</v-list-item-title>
          </v-list-item>
        </template>
      </v-list>
    </v-navigation-drawer>
</template>

<script setup>
import { computed } from 'vue';
import { useStore } from 'vuex';

const store = useStore();

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
    { title: 'dashboard.title', icon: 'mdi-view-dashboard', route: '/dashboard' },
    { title: 'applications.title', icon: 'mdi-apps', route: '/applications' },
    { title: 'deployments.title', icon: 'mdi-rocket-launch', route: '/deployments' },
    { title: 'observability.title',  icon: 'mdi-chart-line', route: '/observability' },
    { title: 'scaling.title', icon: 'mdi-trending-up', route: '/scaling' },
    { title: 'workflows.title', icon: 'mdi-file-tree', route: '/workflows' },
    { title: 'functionsVms.title',  icon: 'mdi-desktop-classic', route: '/functions-vms' },
    { title: 'secrets.title',  icon: 'mdi-key', route: '/secrets' }
];

const operatorMenu = [
  { title: 'dashboard.title', icon: 'mdi-view-dashboard', route: '/dashboard' },
  { title: 'provisioning.title', icon: 'mdi-server', children: [
      { title: 'provisioning.onPremises.sidemenuTitle', route: '/provisioning/on-premises' },
      { title: 'provisioning.cloud.sidemenuTitle', route: '/provisioning/cloud' },
    ]
  },
  { title: 'cloudProvider.title', icon: 'mdi-cloud', route: '/cloud-provider' },
  { title: 'templates.title', icon: 'mdi-layers-triple', route: '/templates' },
  { title: 'secretsSecurity.title',  icon: 'mdi-shield', route: '/secrets-security' },
];

// Computed
const role = computed(() => store.getters['user/getRole']);
const sidebarWidth = computed(() => (props.drawer ? 230 : 60));
const menuItems = computed(() => role.value === 'developer' ? developerMenu : operatorMenu);

</script>

<style scoped> 
.active-nav {
  border-left: 4px solid #f97316;
  font-weight: 600;
  text-shadow: 0 0 2px rgba(255, 255, 255, 0.7);
}
</style>