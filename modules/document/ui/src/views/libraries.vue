<template>
    <main class="document-page">
        <template v-if="selectedRef">
            <div class="document-page-header">
                <a-button @click="closeWorkbench">返回文档库</a-button><span>{{ selectedRef }}</span>
            </div>
            <workbench :key="selectedRef" :client="client" :library-ref="selectedRef" />
        </template>
        <template v-else>
            <div class="document-page-header">
                <h3>文档库</h3>
                <a-button v-if="user.id" type="primary" @click="openCreate">新建文档库</a-button>
                <a-select
                    v-model="category"
                    placeholder="全部分类"
                    allow-clear
                    style="width: 200px"
                    @change="loadLibraries(1)"
                >
                    <a-option v-for="item in categories" :key="item.identifie" :value="item.identifie">{{
                        item.name
                    }}</a-option>
                </a-select>
            </div>
            <a-table :data="libraries" :loading="loading" :pagination="false" row-key="id">
                <template #columns>
                    <a-table-column title="文档库"
                        ><template #cell="{ record }"
                            ><a-link @click="selectedRef = reference(record)">{{ record.name }}</a-link></template
                        ></a-table-column
                    >
                    <a-table-column title="标识"
                        ><template #cell="{ record }">{{ reference(record) }}</template></a-table-column
                    >
                    <a-table-column title="可见性"><template #cell="{ record }"><a-tag :color="record.public ? 'green' : 'gray'">{{ record.public ? '公有' : '私有' }}</a-tag></template></a-table-column>
                    <a-table-column data-index="user_id" title="所有者 ID" />
                    <a-table-column data-index="updated_at" title="更新时间" />
                    <a-table-column title="操作"
                        ><template #cell="{ record }">
                            <a-button type="text" @click="openEdit(record)">编辑</a-button>
                            <a-popconfirm content="确认删除文档库？" @ok="removeLibrary(record)"
                                ><a-button type="text" status="danger" :disabled="!!record.protected">删除</a-button></a-popconfirm
                            >
                        </template></a-table-column
                    >
                </template>
            </a-table>
            <a-pagination
                :current="page"
                :total="total"
                :page-size="20"
                class="document-pagination"
                @change="loadLibraries"
            />
        </template>
        <a-modal v-model:visible="form.visible" :title="form.edit ? '编辑文档库' : '新建文档库'" :footer="false">
            <a-form :model="form" layout="vertical">
                <a-form-item v-if="!form.edit" label="分类"
                    ><a-select v-model="form.category"
                        ><a-option v-for="item in categories" :key="item.identifie" :value="item.identifie">{{
                            item.name
                        }}</a-option></a-select
                    ></a-form-item
                >
                <a-form-item label="标识"
                    ><a-input
                        v-model="form.identifie"
                        :disabled="form.edit"
                        placeholder="小写字母、数字、下划线或短横线"
                /></a-form-item>
                <a-form-item label="名称"><a-input v-model="form.name" /></a-form-item>
                <a-form-item label="描述"><a-textarea v-model="form.description" /></a-form-item>
                <a-form-item label="可见性"><a-radio-group v-model="form.public"><a-radio :value="0">私有</a-radio><a-radio :value="1">公有</a-radio></a-radio-group></a-form-item>
                <a-alert>公有文档库允许所有人查看和下载；仅所有者和创始人可以修改。</a-alert>
                <a-button type="primary" :loading="saving" @click="save">保存</a-button>
            </a-form>
        </a-modal>
    </main>
</template>
<script>
import { Message } from '@arco-design/web-vue';
import Workbench from '../components/workbench.vue';
export default {
    name: 'DocumentLibraries',
    components: { Workbench },
    props: { client: { type: Object, required: true }, user: { type: Object, default: () => ({}) } },
    data: () => ({
        categories: [],
        libraries: [],
        category: '',
        page: 1,
        total: 0,
        loading: false,
        saving: false,
        selectedRef: '',
        form: {},
    }),
    async mounted() {
        if (!this.user.id) return;
        try {
            this.categories = (await this.client.post('/category/list', {})) || [];
            await this.loadLibraries(1);
        } catch (error) {
            Message.error(error.message);
        }
    },
    watch: {
        '$route.query.ref': {
            immediate: true,
            handler(reference) {
                this.selectedRef = String(reference || '');
            },
        },
        selectedRef(reference) {
            if (!this.$router || reference === String(this.$route.query.ref || '')) return;
            const query = { ...this.$route.query };
            if (reference) query.ref = reference;
            else delete query.ref;
            this.$router.replace({ query });
        },
    },
    methods: {
        reference(record) {
            return record.category + '/' + record.identifie;
        },
        async loadLibraries(page = 1) {
            this.loading = true;
            try {
                const result = await this.client.post('/library/list', {
                    category: this.category || '',
                    page,
                    limit: 20,
                });
                this.libraries = result.list || [];
                this.total = result.total;
                this.page = page;
            } catch (error) {
                Message.error(error.message);
            } finally {
                this.loading = false;
            }
        },
        async closeWorkbench() {
            this.selectedRef = '';
            await this.loadLibraries(this.page);
        },
        openCreate() {
            this.form = {
                visible: true,
                edit: false,
                category: this.category || this.categories[0]?.identifie || '',
                identifie: '',
                name: '',
                description: '',
                public: 0,
            };
        },
        openEdit(record) {
            this.form = { ...record, edit: true, visible: true };
        },
        async save() {
            if (this.saving) return;
            if (
                !this.form.name?.trim() ||
                !this.form.category ||
                !/^[a-z0-9][a-z0-9_-]{0,95}$/.test(this.form.identifie || '')
            ) {
                Message.warning('请填写分类、名称及有效标识');
                return;
            }
            this.saving = true;
            try {
                await this.client.post('/library/' + (this.form.edit ? 'edit' : 'create'), {
                    ref: this.reference(this.form),
                    name: this.form.name.trim(),
                    description: this.form.description || '',
                    public: this.form.public || 0,
                    revision: this.form.revision,
                });
                this.form.visible = false;
                await this.loadLibraries(1);
                Message.success('保存成功');
            } catch (error) {
                Message.error(error.message);
            } finally {
                this.saving = false;
            }
        },
        async removeLibrary(record) {
            try {
                await this.client.post('/library/delete', { ref: this.reference(record) });
                await this.loadLibraries(this.page);
            } catch (error) {
                Message.error(error.message);
            }
        },

    },
};
</script>
<style scoped>
.document-page {
    padding: 24px;
    background: white;
    min-height: calc(100vh - 2px);
    box-sizing: border-box;
}
.document-page-header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 24px;
}
.document-page-header h3 {
    margin: 0 auto 0 0;
    font-size: 18px;
}
.document-pagination {
    margin-top: 24px;
    justify-content: flex-end;
}
</style>
