import '@mdi/font/css/materialdesignicons.css';
import 'vuetify/styles';
import { createVuetify } from 'vuetify';
import * as components from 'vuetify/components';
import * as directives from 'vuetify/directives';
import { aliases, mdi } from 'vuetify/iconsets/mdi';

export default createVuetify({
    defaults: {
        global: {
            density: 'comfortable'
        }
    },
    icons: {
        defaultSet: 'mdi',
        aliases: {
            ...aliases,
            success: 'mdi-check-circle-outline',
            info: 'mdi-information-outline',
            warning: 'mdi-alert-outline',
            error: 'mdi-close-circle-outline',
        },
        sets: {
            mdi,
        },
    },
    theme: {
        // #ffffff
        // #ffd7b5
        // #ffb38a
        // #ff9248
        // #ff7500
        defaultTheme: 'dark',
        themes: {
            dark: {
                colors: {
                    background: '#040609', //040609
                    surface: '#000018ff', //0d0f15ff
                    'surface-variant': '#000009ff', //9c9a9aff
                    primary: '#5a86d8ff', //f97316
                    'primary-darken-1': '#5a86d8a7', //ff9248
                    secondary: '#5a73f5ff',
                    'secondary-darken-1': '#5a73f5a7',
                    error: '#cf6679',
                    info: '#2196f3',
                    success: '#4caf50',
                    warning: '#fbe600ff',
                    title: '#ffffff',
                    subtitlte: '#dddddd',
                    text: '#bbbbbb',
                    'list-item': '#1e1e1eff',
                },
            },
            light: {
                colors: {
                    background: '#ffffff',
                    surface: '#f5f5f5',
                    'surface-variant': '#9a9898ff',
                    primary: '#5a86d8ff',
                    'primary-darken-1': '#4274d0b6',
                    secondary: '#5a73f5ff',
                    'secondary-darken-1': '#4a62c4b6',
                    error: '#b00020',
                    info: '#2196f3',
                    success: '#429845ff',
                    warning: '#fbe600ff',
                    title: '#000000',
                    subtitle: '#222222',
                    text: '#444444',
                    'list-item': '#e0e0e0',
                },
            },
        },
        options: {
            customProperties: true,
        },
    },
    components,
    directives
});
