<template>
    <v-app-bar :height="props.toolbarHeight" app color="primary" dark :elevation="props.elevation">
      <v-btn icon @click="toggleSidebar">
        <v-icon>mdi-menu</v-icon>
      </v-btn>

      <v-spacer></v-spacer>

      <v-menu offset-y>
        <template #activator="{ props }">
          <v-btn icon="mdi-account" v-bind="props" aria-label="User menu">
          </v-btn>
        </template>

        <v-list>
          <v-list-item @click="goToProfile">
            <v-list-item-title>{{ $t('userPreferences.myAccount') }}</v-list-item-title>
          </v-list-item>

          <v-menu offset-y location="left">
            <template #activator="{ props }">
              <v-list-item v-bind="props" link>
                <v-list-item-title>{{ $t('userPreferences.language') }}</v-list-item-title>
              </v-list-item>
            </template>

            <v-list>
              <v-list-item @click="changeLanguage('en')">
                <v-list-item-title>{{ $t('userPreferences.languages.en') }}</v-list-item-title>
              </v-list-item>
              <v-list-item @click="changeLanguage('fr')">
                <v-list-item-title>{{ $t('userPreferences.languages.fr') }}</v-list-item-title>
              </v-list-item>
            </v-list>
          </v-menu>

          <v-list-item @click="logout">
            <v-list-item-title>{{ $t('userPreferences.logout') }}</v-list-item-title>
          </v-list-item>
        </v-list>
      </v-menu>
    </v-app-bar>
</template>

<script setup>
import { useI18n } from 'vue-i18n';
import router from '@/router';

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

// Emits
const emit = defineEmits(['update:drawer', 'go-to-profile', 'change-language', 'logout']);

const { locale } = useI18n();

// Methods
function toggleSidebar() {
  emit('update:drawer', !props.drawer);
}

function goToProfile() {
  router.push({ name: 'my-account' });
}

function changeLanguage(lang) {
  locale.value = lang;
}

function logout() {
  emit('logout');
}
</script>