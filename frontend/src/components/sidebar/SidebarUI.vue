<template>
    <v-navigation-drawer v-model="drawerModel" :width="sidebarWidth" app :permanent="!temporary" :temporary="temporary" :elevation="elevation">
        <v-list-item :style="{ height: props.toolbarHeight + 'px' }">
            <v-list-item-title>{{ $t('ui.title') }}</v-list-item-title>
        </v-list-item>

        <v-divider></v-divider>
        <v-list density="compact" nav>
            <v-list-item 
                v-for="item in componentsList" :key="item.title" 
                :to="item.route || '/'" 
                link 
                rounded="0" 
                active-class="active-nav"
            >
                <template #prepend>
                    <v-icon>{{ item.icon }}</v-icon>
                </template>
                <v-list-item-title v-if="drawer">{{ $t(item.title) }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-navigation-drawer>
</template>

<script setup>
import { computed } from 'vue';

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
    temporary: {
        type: Boolean,
        default: false,
    },
    elevation: {
        type: Number,
        default: 8,
    },
});

// Data
const componentsList = [
  { title: 'ui.buttons.title', icon: 'mdi-gesture-tap-button', route: '/ui-components/buttons' },
  { title: 'ui.inputs.title', icon: 'mdi-form-textbox', route: '/ui-components/inputs' },
  { title: 'ui.selectionControls.title', icon: 'mdi-form-select', route: '/ui-components/selection-controls' },
  { title: 'ui.dialogs.title', icon: 'mdi-message-text', route: '/ui-components/dialogs' },
  { title: 'ui.tables.title', icon: 'mdi-table', route: '/ui-components/tables' },
  { title: 'ui.alerts.title', icon: 'mdi-bell', route: '/ui-components/alerts' }
];

// Emits
const emit = defineEmits(['update:drawer']);

// Computed
const sidebarWidth = computed(() => (props.drawer ? 230 : 60));
const drawerModel = computed({
    get: () => props.drawer,
    set: (value) => {
        emit('update:drawer', value);
    }
});
</script>

<style scoped> 
.active-nav {
  border-left: 4px solid #f97316;
  font-weight: 600;
  text-shadow: 0 0 2px rgba(255, 255, 255, 0.7);
}
</style>