import LoginPage from '@/pages/LoginPage.vue';
import RegisterPage from '@/pages/RegisterPage.vue';
import DashboardPage from '@/pages/DashboardPage.vue';
import PremisesProvisioningPage from '@/pages/operator/PremisesProvisioningPage.vue';
import CloudProvisioningPage from '@/pages/operator/CloudProvisioningPage.vue';
import CloudProviderPage from '@/pages/operator/CloudProviderPage.vue';
import SecretsSecurityPage from '@/pages/operator/SecretsSecurityPage.vue';
import TemplatesPage from '@/pages/operator/TemplatesPage.vue';
import UsersManagementPage from '@/pages/operator/UsersManagementPage.vue';
import TeamsManagementPage from '@/pages/operator/TeamsManagementPage.vue';
import Error403Page from '@/pages/errors/Error403Page.vue';
import Error404Page from '@/pages/errors/Error404Page.vue';
import i18n from '@/plugins/i18n';
import { createRouter, createWebHistory } from 'vue-router/auto';
import store from '@/store';
import DeployTemplate from "@/pages/developer/DeployTemplate.vue";

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
        props: route => ({ message: route.query.message }),
        meta: { title: 'login.title' },
    },
    {
        path: '/register',
        name: 'register',
        component: RegisterPage,
        meta: { title: 'register.title' },
    },
    {
        path: '/dashboard',
        name: 'dashboard',
        component: DashboardPage,
        meta: { title: 'dashboard.title', requiresAuth: true, roles: ['admin', 'developer'] },
    },
    {
        path: '/applications',
        name: 'applications',
        meta: { title: 'applications.title', requiresAuth: true, roles: ['developer'] },
    },
    {
        path: '/deployments',
        name: 'deployments',
        meta: { title: 'deployments.title', requiresAuth: false },
        children: [
          {
            path: 'new',
            name: 'new-deployment',
            meta: {title: 'new-deployment.title', requiresAuth: false},
            component: DeployTemplate
          }
        ]
    },
    {
        path: '/secrets',
        name: 'secrets',
        meta: { title: 'secrets.title', requiresAuth: true },
    },
    {
        path: '/observability',
        name: 'observability',
        meta: { title: 'observability.title', requiresAuth: true },
    },
    {
        path: '/scalings',
        name: 'scalings',
        meta: { title: 'scalings.title', requiresAuth: true },
    },
    {
        path: '/workflows',
        name: 'workflows',
        meta: { title: 'workflows.title', requiresAuth: true },
    },
    {
        path: '/functions-vms',
        name: 'functions-vms',
        meta: { title: 'functionsVms.title', requiresAuth: true },
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
                meta: {
                    title: 'provisioning.onPremises.title',
                    requiresAuth: true,
                    roles: ['admin'],
                },
            },
            {
                path: 'cloud',
                name: 'provisioning-cloud',
                component: CloudProvisioningPage,
                meta: { title: 'provisioning.cloud.title', requiresAuth: true, roles: ['admin'] },
            },
        ],
    },
    {
        path: '/cloud-provider',
        name: 'cloud-provider',
        component: CloudProviderPage,
        meta: { title: 'cloudProvider.title', requiresAuth: true, roles: ['admin'] },
    },
    {
        path: '/templates',
        name: 'templates',
        component: TemplatesPage,
        meta: { title: 'templates.title', requiresAuth: true, roles: ['admin'] },
    },
    {
        path: '/secrets-security',
        name: 'secrets-security',
        component: SecretsSecurityPage,
        meta: { title: 'secretsSecurity.title', requiresAuth: true, roles: ['admin'] },
    },
    {
        path: '/administration',
        name: 'administration',
        meta: { title: 'administration.title' },
        children: [
            {
                path: 'users',
                name: 'administration-users',
                component: UsersManagementPage,
                meta: { title: 'administration.users.title', requiresAuth: true, roles: ['admin'] },
            },
            {
                path: 'teams',
                name: 'administration-teams',
                component: TeamsManagementPage,
                meta: { title: 'administration.teams.title', requiresAuth: true, roles: ['admin'] },
            },
        ],
    },
    {
        path: '/policies-compliance',
        name: 'policies-compliance',
        meta: { title: 'policiesCompliance.title', requiresAuth: true, roles: ['admin'] },
    },
    {
        path: '/gitops-synchronization',
        name: 'gitops-synchronization',
        meta: { title: 'gitopsSynchronization.title', requiresAuth: true, roles: ['admin'] },
    },
    {
        path: '/403',
        name: '403',
        component: Error403Page,
        meta: { title: 'errors.403.title' },
    },
    {
        path: '/404',
        name: '404',
        component: Error404Page,
        meta: { title: 'errors.404.title' },
    },
    {
        path: '/:pathMatch(.*)*',
        redirect: '/404',
    },
];

const router = createRouter({
    history: createWebHistory(),
    routes,
});

router.beforeEach((to, from, next) => {
    document.title = i18n.global.t(to.meta?.title) || i18n.global.t('headerTitle.default');
    const user = store.state.user;

    if (to.meta?.requiresAuth && !user.isAuthenticated) {
        next({ name: 'login', query: { message: 'sessionExpired' } });
    } else if (to.meta?.roles && !to.meta.roles.includes(user.role)) {
        next('/403');
    } else {
        next();
    }
});

export default router;
