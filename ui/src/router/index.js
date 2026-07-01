import { createRouter, createWebHashHistory } from 'vue-router';

const routes = [
    {
        path: '/',
        name: 'home',
        redirect: "/zpk"
    },
    {
        path: '/zpk',
        name: 'zpk',
        component: () => import("../views/respo.vue"),
    },
    {
        path: '/zpk-version',
        name: 'zpk-version',
        component: () => import("../views/version.vue"),
    },
    {
        path: '/zpk-detail/:id',
        name: 'zpk-detail',
        component: () => import("../views/respo-detail.vue"),
    },
    {
        path: '/zpk-editfront',
        name: 'zpk-editfront',
        component: () => import("../views/respo-editfront.vue"),
    },
    {
        path: '/zpk-edit',
        name: 'zpk-edit',
        component: () => import("../views/respo-edit.vue"),
    },
    {
        path: '/zpk-manifest',
        name: 'zpk-manifest',
        component: () => import("../views/respo-create.vue"),
    },
    {
        path: '/zpk-manifest-editor',
        name: 'zpk-manifest-editor',
        component: () => import("../components/manifest-editor.vue"),
    },
    {
        path: '/zpk-description',
        name: 'zpk-description',
        component: () => import("../views/description.vue"),
    },
    {
        path: '/zpk-registry',
        name: 'zpk-registry',
        component: () => import("../views/registry.vue"),
    },
    {
        path: '/zpk-registry/:id',
        name: 'zpk-registry-detail',
        component: () => import("../views/registry-detail.vue"),
    },
    {
        path: '/zpk-user',
        name: 'zpk-user',
        component: () => import("../views/user.vue"),
    },
    {
        path: '/zpk-namespace',
        name: 'zpk-namespace',
        component: () => import("../views/namespace.vue"),
    },
    {
        path: '/zpk-settings',
        name: 'zpk-settings',
        component: () => import("../views/settings.vue")
    },
    {
        path: "/:pathMatch(.*)*",
        redirect: "/zpk"
    },
];

const router = createRouter({
    history: createWebHashHistory(),
    routes
});

export default router;
