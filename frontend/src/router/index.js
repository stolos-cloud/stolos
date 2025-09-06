import HelloWorld from '@/components/HelloWorld.vue'
import { createRouter, createWebHistory } from 'vue-router/auto'

const routes = [
  {
    path: '/',
    name: 'Homepage',
    component: HelloWorld
  }
  // Add more routes here as needed
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
