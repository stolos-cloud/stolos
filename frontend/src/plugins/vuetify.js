import '@mdi/font/css/materialdesignicons.css';
import 'vuetify/styles';
import { createVuetify } from 'vuetify';
import * as components from 'vuetify/components';
import * as directives from 'vuetify/directives';

export default createVuetify({
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
                    background: '#262525ff',
                    surface: '#363535ff',
                    primary: '#f97316',
                    'primary-darken-1': '#ff9248',
                    secondary: '#ffb38a',
                    'secondary-darken-1': '#ffd7b5',
                    error: '#cf6679',
                    info: '#2196f3',
                    success: '#4caf50',
                    warning: '#fb8c00',
                },
            },
            light: {
                colors: {
                    background: '#ffffff',
                    surface: '#f5f5f5',
                    primary: '#ff7500',
                    'primary-darken-1': '#ff9248',
                    secondary: '#ffb38a',
                    'secondary-darken-1': '#ffd7b5',
                    error: '#b00020',
                    info: '#2196f3',
                    success: '#4caf50',
                    warning: '#fb8c00',
                },
            },
        },
        options: {
            customProperties: true,
        },
    },
    components,
    directives,
});
