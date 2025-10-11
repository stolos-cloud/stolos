import { createApp } from 'vue'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import 'vuetify/styles'
import App from './App.vue'
import router from './router'

const vuetify = createVuetify({
    theme: {
        defaultTheme: 'system'
    },
    components,
    directives
})
createApp(App).use(vuetify).use(router).mount('#app')
