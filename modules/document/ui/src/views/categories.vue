<template>
    <a-result v-if="!isAdmin" status="403" title="无分类管理权限" />
    <main v-else class="category-page">
        <div class="category-header">
            <h3>分类管理</h3>
            <a-button type="primary" @click="openCreate">新建分类</a-button>
        </div>
        <a-table :data="categories" :loading="loading" :pagination="false" row-key="id">
            <template #columns>
                <a-table-column data-index="name" title="分类名称" /><a-table-column
                    data-index="identifie"
                    title="分类标识"
                />
                <a-table-column data-index="description" title="描述" />
                <a-table-column title="操作"
                    ><template #cell="{ record }">
                        <a-button type="text" @click="openEdit(record)">编辑</a-button>
                        <a-button type="text" @click="openPermission(record)">授权</a-button>
                        <a-popconfirm content="仅允许删除空分类，确认删除？" @ok="remove(record)"
                            ><a-button type="text" status="danger" :disabled="record.identifie === 'applications'"
                                >删除</a-button
                            ></a-popconfirm
                        >
                    </template></a-table-column
                >
            </template>
        </a-table>
        <a-modal v-model:visible="form.visible" :title="form.edit ? '编辑分类' : '新建分类'" :footer="false">
            <a-form :model="form" layout="vertical">
                <a-form-item label="分类标识"
                    ><a-input
                        v-model="form.identifie"
                        :disabled="form.edit"
                        placeholder="小写字母、数字、下划线或短横线"
                /></a-form-item>
                <a-form-item label="分类名称"><a-input v-model="form.name" /></a-form-item>
                <a-form-item label="描述"><a-textarea v-model="form.description" /></a-form-item>
                <a-button type="primary" :loading="saving" @click="save">保存</a-button>
            </a-form>
        </a-modal>
        <a-modal v-model:visible="permission.visible" title="分类授权" :footer="false">
            <a-form :model="permission" layout="vertical">
                <a-form-item label="被授权用户 ID"><a-input v-model="permission.targetUserID" /></a-form-item>
                <a-form-item label="动作"
                    ><a-checkbox-group v-model="permission.actions" :options="['read', 'write', 'create', 'delete']"
                /></a-form-item>
                <a-button type="primary" :loading="saving" @click="savePermission">保存授权</a-button>
            </a-form>
        </a-modal>
    </main>
</template>
<script>
import { Message } from '@arco-design/web-vue';
export default {
    name: 'DocumentCategories',
    props: { client: { type: Object, required: true }, user: { type: Object, default: () => ({}) } },
    computed: {
        isAdmin() {
            return !!this.user.admin || ['founder', 'super'].includes(this.user.role);
        },
    },
    data: () => ({
        categories: [],
        loading: false,
        saving: false,
        form: {},
        permission: { visible: false, actions: [] },
    }),
    watch: {
        isAdmin: {
            immediate: true,
            handler(allowed) {
                if (allowed) this.load();
            },
        },
    },
    methods: {
        async load() {
            this.loading = true;
            try {
                this.categories = (await this.client.post('/category/list', {})) || [];
            } catch (error) {
                Message.error(error.message);
            } finally {
                this.loading = false;
            }
        },
        openCreate() {
            this.form = { visible: true, edit: false, identifie: '', name: '', description: '' };
        },
        openEdit(record) {
            this.form = { ...record, edit: true, visible: true };
        },
        async save() {
            if (this.saving) return;
            if (!this.form.name?.trim() || !/^[a-z0-9][a-z0-9_-]{0,95}$/.test(this.form.identifie || '')) {
                Message.warning('请填写分类名称及有效标识');
                return;
            }
            this.saving = true;
            try {
                await this.client.post('/category/' + (this.form.edit ? 'edit' : 'create'), {
                    identifie: this.form.identifie,
                    name: this.form.name.trim(),
                    description: this.form.description || '',
                    revision: this.form.revision,
                });
                this.form.visible = false;
                await this.load();
                Message.success('保存成功');
            } catch (error) {
                Message.error(error.message);
            } finally {
                this.saving = false;
            }
        },
        async remove(record) {
            try {
                await this.client.post('/category/delete', { identifie: record.identifie });
                await this.load();
            } catch (error) {
                Message.error(error.message);
            }
        },
        openPermission(record) {
            this.permission = { visible: true, record, targetUserID: '', actions: ['read'] };
        },
        async savePermission() {
            const targetUserID = Number(this.permission.targetUserID);
            if (!Number.isInteger(targetUserID) || targetUserID <= 0) {
                Message.warning('请输入有效的用户 ID');
                return;
            }
            this.saving = true;
            try {
                await this.client.post('/category/permission/set', {
                    identifie: this.permission.record.identifie,
                    target_user_id: targetUserID,
                    actions: this.permission.actions,
                });
                this.permission.visible = false;
                Message.success('授权已保存');
            } catch (error) {
                Message.error(error.message);
            } finally {
                this.saving = false;
            }
        },
    },
};
</script>
<style scoped>
.category-page {
    padding: 24px;
    min-height: calc(100vh - 2px);
    box-sizing: border-box;
    background: white;
}
.category-header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 24px;
}
.category-header h3 {
    margin: 0 auto 0 0;
    font-size: 18px;
}
</style>
