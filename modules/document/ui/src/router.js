import { createRouter, createWebHashHistory } from 'vue-router';
export default createRouter({
    history: createWebHashHistory(),
    routes: [
        { path: '/', redirect: '/documents' },
        { path: '/documents', component: () => import('./views/libraries.vue') },
        { path: '/categories', component: () => import('./views/categories.vue') },
    ],
});
