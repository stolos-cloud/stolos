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
                    background: '#040609',//131212ff //040609
                    surface: '#0d0f15ff',
                    'surface-variant': '#9c9a9aff',
                    primary: '#f97316',
                    'primary-darken-1': '#ff9248',
                    secondary: '#ffb38a',
                    'secondary-darken-1': '#ffd7b5',
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
                    primary: '#ff7500',
                    'primary-darken-1': '#ff9248',
                    secondary: '#ffb38a',
                    'secondary-darken-1': '#ffd7b5',
                    error: '#b00020',
                    info: '#2196f3',
                    success: '#4caf50',
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
