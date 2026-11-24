<template>
    <a-result v-if="!isAdmin" status="403" title="无分类管理权限" />
    <main v-else class="category-page">
        <div class="category-header">
            <h3>分类管理</h3>
            <a-button type="primary" @click="openCreate">新建分类</a-button>
        </div>
        <a-alert class="category-notice">分类关联公共命名空间。新空间默认从镜像列表隐藏；标识修改和删除会同步到公共命名空间。</a-alert>
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
                        <a-popconfirm
                            content="确认删除分类及对应公共命名空间？仅允许删除没有资源的空间。"
                            position="lt"
                            type="warning"
                            content-class="document-delete-popconfirm"
                            ok-text="确定"
                            cancel-text="取消"
                            :disabled="record.identifie === 'app'"
                            :ok-button-props="{ status: 'danger' }"
                            :cancel-button-props="{ type: 'secondary' }"
                            @ok="remove(record)"
                            ><a-button type="text" status="danger" :disabled="record.identifie === 'app'"
                                >删除</a-button
                            ></a-popconfirm
                        >
                    </template></a-table-column
                >
            </template>
        </a-table>
        <a-modal v-model:visible="form.visible" :title="form.edit ? '编辑分类' : '新建分类'">
            <a-form :model="form" layout="vertical">
                <a-form-item label="命名空间标识"
                    ><a-input
                        v-model="form.identifie"
                        :disabled="form.edit && form.originalIdentifie === 'app'"
                        placeholder="小写字母、数字、下划线或短横线"
                /></a-form-item>
                <a-form-item label="分类名称"><a-input v-model="form.name" /></a-form-item>
                <a-form-item label="描述"><a-textarea v-model="form.description" /></a-form-item>
            </a-form>
            <template #footer>
                <a-button @click="form.visible = false">取消</a-button>
                <a-button type="primary" :loading="saving" @click="save">保存</a-button>
            </template>
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
            this.form = { ...record, originalIdentifie: record.identifie, edit: true, visible: true };
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
                    identifie: this.form.edit ? this.form.originalIdentifie : this.form.identifie,
                    ...(this.form.edit ? { new_identifie: this.form.identifie } : {}),
                    name: this.form.name.trim(),
                    description: this.form.description || '',
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
.category-notice {
    margin-bottom: 20px;
}
</style>
