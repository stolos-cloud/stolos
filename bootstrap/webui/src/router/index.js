import { createRouter, createWebHistory } from 'vue-router'
import Wizard from '../views/Wizard.vue'

const routes = [
    { path: '/', redirect: '/wizard' },
    { path: '/wizard', name: 'Wizard', component: Wizard },
]

export default createRouter({
    history: createWebHistory(),
    routes,
})
