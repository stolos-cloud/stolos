import { createApp } from 'vue'
import { createVuetify } from 'vuetify'
import * as def_components from 'vuetify/components'
import * as directives from 'vuetify/directives'
import 'vuetify/styles'
import App from './App.vue'
import router from './router'
import {VStepperVertical} from "vuetify/labs/components";

const vuetify = createVuetify({
    theme: {
        defaultTheme: 'system'
    },
    components: {
        ...def_components,
        VStepperVertical,
    },
    directives
})
createApp(App).use(vuetify).use(router).mount('#app')
