<template>
    <div>
        <v-list-item>
          <template #prepend>
            <v-icon>mdi-theme-light-dark</v-icon>
          </template>
          <v-list-item-title class="text-body-2">
            {{ isDark ? $t('userPreferences.theme.dark') : $t('userPreferences.theme.light') }}
          </v-list-item-title>
          <template #append>
              <v-switch v-model="isDark" hide-details color="primary" />
          </template>
        </v-list-item>
        <v-list-item>
          <template #prepend>
              <v-icon>mdi-account-circle</v-icon>
          </template>
          <v-list-item-title class="text-body-2">{{ userEmail }}</v-list-item-title>
          <v-list-item-subtitle class="text-caption">{{ userRole }}</v-list-item-subtitle>
          <template #append>
            <v-menu :offset="[18, 10]" location="right">
              <template #activator="{ props }">
                <v-btn v-bind="props" icon="mdi-dots-vertical" size="x-small" variant="text"></v-btn>
              </template>
              <v-list>
                <v-menu offset-y location="right">
                  <template #activator="{ props }">
                    <v-list-item v-bind="props" link>
                      <v-list-item-title class="text-body-2">{{ $t('userPreferences.language') }}</v-list-item-title>
                    </v-list-item>
                  </template>

                  <v-list>
                    <v-list-item @click="changeLanguage('en')">
                      <v-list-item-title class="text-body-2">{{ $t('userPreferences.languages.en') }}</v-list-item-title>
                    </v-list-item>
                    <v-list-item @click="changeLanguage('fr')">
                      <v-list-item-title class="text-body-2">{{ $t('userPreferences.languages.fr') }}</v-list-item-title>
                    </v-list-item>
                  </v-list>
                </v-menu>
                <v-list-item link @click="logout">
                  <v-list-item-title class="text-body-2">{{ $t('userPreferences.logout') }}</v-list-item-title>
                </v-list-item>
              </v-list>
            </v-menu>
          </template>
        </v-list-item>
    </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n';
import { useTheme } from 'vuetify';
import { computed } from 'vue';
import { useStore } from "vuex";
import router from '@/router';

const i18n = useI18n();
const theme = useTheme();
const store = useStore();

const userEmail = computed(() => store.getters['user/getEmail']);
const userRole = computed(() => {
    const role = store.getters['user/getRole'];
    return role ? role.charAt(0).toUpperCase() + role.slice(1) : '';
});
const isDark = computed({
    get: () => store.getters['user/getTheme'] === "dark",
    set: (value) => {
        const themeValue = value ? 'dark' : 'light';
        theme.change(themeValue);
        store.dispatch('user/setTheme', themeValue);
    }
});
const language = computed({
    get: () => store.getters['user/getLanguage'],
    set: (value) => {
        store.dispatch('user/setLanguage', value);
    }
});

// Methods
function changeLanguage(lang) {
    language.value = lang;
    i18n.locale.value = lang;
}

function logout() {
    store.dispatch('user/logoutUser').then(() => {
        router.push('/login');
    });
}
</script>