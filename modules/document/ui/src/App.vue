<template>
    <a-spin v-if="loading" :loading="true" style="width: 100%; padding: 80px" />
    <div v-else-if="user" class="document-shell">
        <aside class="document-navigation">
            <router-link to="/documents">文档库</router-link>
            <router-link v-if="user.admin" to="/categories">分类管理</router-link>
            <a-button v-if="!user.id" @click="user = null">登录</a-button>
        </aside>
        <router-view :client="client" :user="user" class="document-content" />
    </div>
    <div v-else class="document-login">
        <a-card title="文档管理登录" style="width: 420px">
            <a-alert v-if="error" type="error">{{ error }}</a-alert>
            <a-form :model="form" layout="vertical" style="margin-top: 20px" @submit-success="login">
                <a-form-item label="面板访问凭据"
                    ><a-input-password v-model="form.accessToken" placeholder="面板内打开可自动登录"
                /></a-form-item>
                <a-button type="primary" html-type="submit" :loading="loggingIn">登录</a-button>
            </a-form>
        </a-card>
    </div>
</template>
<script>
import { createStandaloneClient } from './utils/client';
export default {
    data: () => ({
        client: createStandaloneClient(),
        user: null,
        loading: true,
        loggingIn: false,
        form: { accessToken: '' },
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
    methods: {
        async login() {
            this.loggingIn = true;
            try {
                await this.client.login(this.form.accessToken);
                this.form.accessToken = '';
                this.user = await this.client.post('/user');
                this.error = '';
            } catch (error) {
                this.error = error.message;
            } finally {
                this.loggingIn = false;
            }
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
.document-navigation {
    width: 160px;
    padding: 20px 12px;
    box-sizing: border-box;
    border-right: 1px solid #eee;
}
.document-navigation a {
    display: block;
    padding: 12px;
    text-decoration: none;
    border-radius: 4px;
    color: #4e5969;
    margin-bottom: 8px;
}
.document-navigation .router-link-active {
    color: #165dff;
    background: #e8f3ff;
}
.document-login {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #f7f8fa;
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
