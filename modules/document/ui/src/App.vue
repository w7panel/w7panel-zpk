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
    created() {
        this.panelRouteBus = window.$wujie?.bus;
        this.panelRouteBus?.$on('routeChange', this.onPanelRouteChange);
    },
    beforeUnmount() {
        this.panelRouteBus?.$off('routeChange', this.onPanelRouteChange);
    },
    methods: {
        onPanelRouteChange(path) {
            this.$router.push(path);
        },
    },
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
    padding: 0;
    font-size: 14px;
    background: #ffffff;
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
.arco-table {
	color: #1d2129;
}

.arco-table .arco-table-container,
.arco-table-border .arco-table-container,
.arco-table-border:not(.arco-table-border-cell) .arco-table-container {
	border: 0 !important;
}

.arco-table .arco-table-th,
.table-header .arco-table-th {
	color: #1d2129 !important;
	font-weight: 400 !important;
	background: #f2f3f5 !important;
	border-bottom: 1px solid #f2f3f5 !important;
}

.arco-table .arco-table-td {
	color: #1d2129;
	background: transparent;
	border-bottom: 1px solid #f2f3f5 !important;
}

.arco-table .arco-table-cell {
	min-height: 22px;
	padding: 10px 16px !important;
	line-height: 22px;
	font-size: 14px;
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
