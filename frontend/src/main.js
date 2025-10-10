import { createApp } from 'vue';
import App from './App.vue';
import router from './router';
import store from './store';
import vuetify from './plugins/vuetify';
import i18n from './plugins/i18n';

async function initializeApp() {
    await store.dispatch('user/initAuth');
    const app = createApp(App);
    app.use(i18n);
    app.use(vuetify);
    app.use(store);
    app.use(router);
    app.mount('#app');
}

initializeApp();
