import App from './App.vue'
import router from './router'
import store from './store'
import vuetify from './plugins/vuetify'
import { createApp } from 'vue'

const app = createApp(App)
app.use(vuetify)
app.use(store)
app.use(router)
app.mount('#app')
