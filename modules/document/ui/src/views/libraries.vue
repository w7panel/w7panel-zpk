<template>
    <main class="document-page">
        <template v-if="selectedRef">
            <div class="document-page-header">
                <a-button @click="closeWorkbench">返回文档库</a-button>
                <span class="document-reference">{{ selectedRef }}</span>
            </div>
            <workbench :key="selectedRef" :client="client" :library-ref="selectedRef" />
        </template>
        <template v-else>
            <div class="document-page-toolbar">
                <div class="document-toolbar-left">
                    <a-button v-if="user.id" type="primary" @click="openCreate">新建文档库</a-button>
                </div>
                <div class="document-toolbar-right">
                    <span>分类：</span>
                    <a-select
                        v-model="category"
                        placeholder="全部分类"
                        allow-clear
                        class="document-category-select"
                        @change="loadLibraries(1)"
                    >
                        <a-option v-for="item in categories" :key="item.identifie" :value="item.identifie">{{
                            item.name
                        }}</a-option>
                    </a-select>
                </div>
            </div>
            <a-table :data="libraries" :loading="loading" :pagination="false" row-key="id">
                <template #columns>
                    <a-table-column title="文档库"
                        ><template #cell="{ record }"
                            ><a-link @click="selectedRef = reference(record)">{{ record.name }}</a-link></template
                        ></a-table-column
                    >
                    <a-table-column title="分类">
                        <template #cell="{ record }">{{ categoryName(record.category) }}</template>
                    </a-table-column>
                    <a-table-column title="标识"
                        ><template #cell="{ record }">{{ reference(record) }}</template></a-table-column
                    >
                    <a-table-column title="可见性"><template #cell="{ record }"><a-tag :color="record.public ? 'green' : 'gray'">{{ record.public ? '公有' : '私有' }}</a-tag></template></a-table-column>
                    <a-table-column data-index="username" title="所有者" />
                    <a-table-column title="更新时间">
                        <template #cell="{ record }">{{ new Date(record.updated_at).toLocaleString() }}</template>
                    </a-table-column>
                    <a-table-column title="操作"
                        ><template #cell="{ record }">
                            <a-button type="text" @click="showDownloadURL(record)">整体下载地址</a-button>
                            <a-button type="text" @click="openEdit(record)">编辑</a-button>
                            <a-popconfirm
                                content="确认删除文档库？"
                                position="lt"
                                type="warning"
                                content-class="document-delete-popconfirm"
                                ok-text="确定"
                                cancel-text="取消"
                                :disabled="!!record.protected"
                                :ok-button-props="{ status: 'danger' }"
                                :cancel-button-props="{ type: 'secondary' }"
                                @ok="removeLibrary(record)"
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
        <a-modal v-model:visible="download.visible" title="整体下载地址" :footer="false">
            <a-alert>公有文档库可直接下载；私有文档库需通过请求头携带所有者或创始人的登录凭据。</a-alert>
            <a-input :model-value="download.url" readonly style="margin-top: 16px">
                <template #append><a-button @click="copyDownloadURL">复制</a-button></template>
            </a-input>
        </a-modal>
        <a-modal v-model:visible="form.visible" :title="form.edit ? '编辑文档库' : '新建文档库'">
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
import Workbench from '../components/workbench.vue';
export default {
    name: 'DocumentLibraries',
    components: { Workbench },
    props: { client: { type: Object, required: true }, user: { type: Object, default: () => ({}) } },
    data: () => ({
        categories: [],
        libraries: [],
        download: { visible: false, url: '' },
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
        async showDownloadURL(record) {
            try {
                const result = await this.client.post('/download-url');
                const url = new URL(result.url);
                url.searchParams.set('ref', this.reference(record));
                this.download = { visible: true, url: url.href };
            } catch (error) {
                Message.error(error.message);
            }
        },
        async copyDownloadURL() {
            try {
                await navigator.clipboard.writeText(this.download.url);
                Message.success('下载地址已复制');
            } catch {
                Message.error('复制失败，请手动复制下载地址');
            }
        },
        categoryName(identifie) {
            return this.categories.find((category) => category.identifie === identifie)?.name || identifie;
        },
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
    padding: 24px 30px;
    border-top: 1px solid #eeeeee;
    background: white;
    min-height: calc(100vh - 2px);
    box-sizing: border-box;
}
.document-page-header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 20px;
}
.document-reference {
    color: #86909c;
}
.document-page-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 16px;
    margin-bottom: 20px;
}
.document-toolbar-left,
.document-toolbar-right {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 10px;
    min-width: 0;
}
.document-toolbar-right {
    flex-wrap: nowrap;
    flex-shrink: 0;
    justify-content: flex-end;
    margin-left: auto;
    color: var(--color-text-2);
    font-size: 14px;
}
.document-toolbar-right :deep(.document-category-select) {
    width: 180px;
    flex-shrink: 0;
}
.document-pagination {
    margin-top: 24px;
    justify-content: flex-end;
}
</style>
