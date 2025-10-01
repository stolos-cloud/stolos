import LoginPage from '@/pages/LoginPage.vue'
import RegisterPage from '@/pages/RegisterPage.vue'
import DashboardPage from '@/pages/DashboardPage.vue'
import PremisesProvisioningPage from '@/pages/operator/PremisesProvisioningPage.vue'
import CloudProvisioningPage from '@/pages/operator/CloudProvisioningPage.vue'
import CloudProviderPage from '@/pages/operator/CloudProviderPage.vue'
import SecretsSecurityPage from '@/pages/operator/SecretsSecurityPage.vue'
import TemplatesPage from '@/pages/operator/TemplatesPage.vue'
import i18n from '@/plugins/i18n'
import { createRouter, createWebHistory } from 'vue-router/auto'

const routes = [
  {
    path: '/',
    name: 'home',
    redirect: '/login',
  },
  {
    path: '/login',
    name: 'login',
    component: LoginPage,
    meta: { title: 'login.title' }
  },
  {
    path: '/register',
    name: 'register',
    component: RegisterPage,
    meta: { title: 'register.title' }
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: DashboardPage,
    meta: { title: 'dashboard.title' }
  },
  {
    path: '/applications',
    name: 'applications',
    meta: { title: 'applications.title' }
  },
  {
    path: '/deployments',
    name: 'deployments',
    meta: { title: 'deployments.title' }
  },
  {
    path: '/secrets',
    name: 'secrets',
    meta: { title: 'secrets.title' }
  },
  {
    path: '/observability',
    name: 'observability',
    meta: { title: 'observability.title' }
  },
  {
    path: '/scalings',
    name: 'scalings',
    meta: { title: 'scalings.title' }
  },
  {
    path: '/workflows',
    name: 'workflows',
    meta: { title: 'workflows.title' }
  },
  {
    path: '/functions-vms',
    name: 'functions-vms',
    meta: { title: 'functionsVms.title' }
  },
  //operator routes
  {
    path: '/provisioning',
    name: 'provisioning',
    meta: { title: 'provisioning.title' },
    children: [
      {
        path: 'on-premises',
        name: 'provisioning-on-premises',
        component: PremisesProvisioningPage,
        meta: { title: 'provisioning.onPremises.title' }
      },
      {
        path: 'cloud',
        name: 'provisioning-cloud',
        component: CloudProvisioningPage,
        meta: { title: 'provisioning.cloud.title' }
      }
    ]
  },
  {
    path: '/cloud-provider',
    name: 'cloud-provider',
    component: CloudProviderPage,
    meta: { title: 'cloudProvider.title' }
  },
  {
    path: '/templates',
    name: 'templates',
    component: TemplatesPage,
    meta: { title: 'templates.title' }
  },
  {
    path: '/secrets-security',
    name: 'secrets-security',
    component: SecretsSecurityPage,
    meta: { title: 'secretsSecurity.title' }
  },
  {
    path: '/policies-compliance',
    name: 'policies-compliance',
    meta: { title: 'policiesCompliance.title' }
  },
  {
    path: '/gitops-synchronization',
    name: 'gitops-synchronization',
    meta: { title: 'gitopsSynchronization.title' }
  },
  {
    path: '/403',
    name: '403',
    meta: { title: 'errors.403.title' }
  },
  {
    path: '/404',
    name: '404',
    meta: { title: 'errors.404.title' }
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  document.title = i18n.global.t(to.meta?.title) || i18n.global.t('headerTitle.default');
  next();
});

export default router
