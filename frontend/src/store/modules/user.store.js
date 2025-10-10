import {
    login,
    logout,
    refreshToken,
    initAuthenticationExistingToken,
} from '@/services/auth.service';
import { StorageService } from '@/services/storage.service';

export const user = {
    namespaced: true,
    state: {
        email: null,
        role: null,
        id: null,
        token: null,
        teams: [],
        isAuthenticated: false,
        theme: StorageService.get('theme') || 'dark',
        language: StorageService.get('language') || 'en',
    },
    mutations: {
        SET_USER(state, { email, role, id, token, teams }) {
            state.email = email;
            state.role = role;
            state.id = id;
            state.token = token;
            state.teams = teams;
            state.isAuthenticated = true;
        },
        SET_EMAIL(state, email) {
            state.email = email;
        },
        SET_ROLE(state, role) {
            state.role = role;
        },
        SET_TEAMS(state, teams) {
            state.teams = teams;
        },
        SET_THEME(state, theme) {
            state.theme = theme;
            StorageService.set('theme', theme);
        },
        SET_LANGUAGE(state, language) {
            state.language = language;
            StorageService.set('language', language);
        },
        CLEAR_USER(state) {
            state.email = null;
            state.role = null;
            state.id = null;
            state.token = null;
            state.teams = [];
            state.isAuthenticated = false;
        },
    },
    actions: {
        setEmail({ commit }, email) {
            commit('SET_EMAIL', email);
        },
        setRole({ commit }, role) {
            commit('SET_ROLE', role);
        },
        setTeams({ commit }, teams) {
            commit('SET_TEAMS', teams);
        },
        setTheme({ commit }, theme) {
            commit('SET_THEME', theme);
        },
        setLanguage({ commit }, language) {
            commit('SET_LANGUAGE', language);
        },
        async initAuth({ commit }) {
            await initAuthenticationExistingToken().then(response => {
                if (response == null) {
                    commit('CLEAR_USER');
                    return;
                }
                const { token, user } = response;
                commit('SET_USER', {
                    email: user.email,
                    role: user.role,
                    id: user.id,
                    teams: user.teams,
                    token,
                });
            });
        },
        async loginUser({ commit }, { email, password }) {
            await login(email, password).then(({ token, user }) => {
                commit('SET_USER', {
                    email: user.email,
                    role: user.role,
                    id: user.id,
                    teams: user.teams,
                    token,
                });
            });
        },
        async logoutUser({ commit }) {
            await logout().then(() => {
                commit('CLEAR_USER');
            });
        },
        async refreshTokenUser({ commit, state }) {
            await refreshToken().then(({ token }) => {
                commit('SET_USER', {
                    email: state.email,
                    role: state.role,
                    id: state.id,
                    teams: state.teams,
                    token,
                });
            });
        },
    },
    getters: {
        getEmail: state => state.email,
        getRole: state => state.role,
        getId: state => state.id,
        getToken: state => state.token,
        isAuthenticated: state => state.isAuthenticated,
        getTeams: state => state.teams,
        getTheme: state => state.theme,
        getLanguage: state => state.language,
    },
};
