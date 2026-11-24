<template>
    <a-spin v-if="loading" :loading="true" style="width: 100%; padding: 80px" />
    <div v-else-if="user" class="document-shell">
        <router-view :client="client" :user="user" class="document-content" />
    </div>
    <a-result v-else status="error" title="无法加载文档管理" :subtitle="error" />
</template>
<script>
import { createStandaloneClient } from './utils/client';
export default {
    data: () => ({
        client: createStandaloneClient(),
        user: null,
        loading: true,
        error: '',
    }),
    async mounted() {
        await this.$router.isReady();
        try {
            this.user = await this.client.start();
        } catch (error) {
            if (this.$route.query.ref) this.user = { id: 0, admin: false };
            else this.error = error.message;
        } finally {
            this.loading = false;
        }
    },
    watch: {
        '$route.query.ref'(reference) {
            if (reference && !this.user && !this.loading) this.user = { id: 0, admin: false };
        },
    },
};
</script>
<style>
body {
    margin: 0;
    font-family: Arial, 'PingFang SC', 'Microsoft YaHei', sans-serif;
    color: #1d2129;
}
.document-shell {
    display: flex;
    min-height: 100vh;
}
.document-content {
    flex: 1;
    min-width: 0;
}
.document-delete-popconfirm .arco-popconfirm-footer {
    display: flex;
    flex-direction: row-reverse;
}
.document-delete-popconfirm .arco-popconfirm-footer > button {
    margin-left: 8px;
}
.document-delete-popconfirm .arco-btn-primary.arco-btn-status-danger {
    background-color: rgb(var(--danger-6));
    border-color: transparent;
}
</style>
