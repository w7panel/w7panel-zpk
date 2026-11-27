<template>
    <main class="document-page" :class="{ 'document-page-editing': selectedRef }">
        <template v-if="selectedRef">
            <div class="document-page-header">
                <button class="document-back" type="button" aria-label="返回文档库" @click="closeWorkbench">
                    <icon-arrow-left />
                </button>
                <a-breadcrumb>
                    <a-breadcrumb-item><a class="document-back-label" @click="closeWorkbench">文档库</a></a-breadcrumb-item>
                    <a-breadcrumb-item>{{ selectedName }}</a-breadcrumb-item>
                </a-breadcrumb>
            </div>
            <div class="document-editor-content">
                <workbench :key="selectedRef" :client="client" :library-ref="selectedRef" @library-loaded="selectedName = $event" />
            </div>
        </template>
        <template v-else>
            <div class="document-page-toolbar">
                <div class="document-toolbar-left">
                    <a-button v-if="user.id" type="primary" @click="openCreate">新建文档库</a-button>
                </div>
            </div>
            <a-table :data="libraries" :loading="loading" :pagination="false" row-key="id">
                <template #columns>
                    <a-table-column title="文档库" :width="360">
                        <template #cell="{ record }">
                            <a-link @click="selectedRef = reference(record)">{{ record.name }}</a-link>
                            <div v-if="downloadBaseURL" class="document-download-address">
                                <span :title="downloadURL(record)">{{ downloadURL(record) }}</span>
                                <a-button type="text" size="mini" aria-label="复制下载地址" @click="copyDownloadURL(record)">
                                    <template #icon><icon-copy /></template>
                                </a-button>
                            </div>
                        </template>
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
        <a-modal v-model:visible="form.visible" :title="form.edit ? '编辑文档库' : '新建文档库'">
            <a-form :model="form" layout="vertical">
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
import { IconArrowLeft, IconCopy } from '@arco-design/web-vue/es/icon';
import Workbench from '../components/workbench.vue';
export default {
    name: 'DocumentLibraries',
    components: { Workbench, IconArrowLeft, IconCopy },
    props: { client: { type: Object, required: true }, user: { type: Object, default: () => ({}) } },
    data: () => ({
        libraries: [],
        downloadBaseURL: '',
        page: 1,
        total: 0,
        loading: false,
        saving: false,
        selectedRef: '',
        selectedName: '',
        form: {},
    }),
    async mounted() {
        if (!this.user.id) return;
        try {
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
            this.selectedName = this.libraries.find((library) => this.reference(library) === reference)?.name || '';
            if (!this.$router || reference === String(this.$route.query.ref || '')) return;
            const query = { ...this.$route.query };
            if (reference) query.ref = reference;
            else delete query.ref;
            this.$router.replace({ query });
        },
    },
    methods: {
        downloadURL(record) {
            const url = new URL(this.downloadBaseURL);
            url.searchParams.set('ref', this.reference(record));
            return url.href;
        },
        async copyDownloadURL(record) {
            try {
                await navigator.clipboard.writeText(this.downloadURL(record));
                Message.success('下载地址已复制');
            } catch {
                Message.error('复制失败，请手动复制下载地址');
            }
        },
        reference(record) {
            return record.namespace + '/' + record.identifie;
        },
        async loadLibraries(page = 1) {
            this.loading = true;
            try {
                const result = await this.client.post('/library/list', {
                    page,
                    limit: 20,
                });
                this.libraries = result.list || [];
                this.total = result.total;
                this.page = page;
                if (!this.downloadBaseURL) {
                    this.downloadBaseURL = (await this.client.post('/download-url')).url;
                }
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
                namespace: 'app_doc',
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
                !/^[a-z0-9][a-z0-9_-]{0,95}$/.test(this.form.identifie || '')
            ) {
                Message.warning('请填写名称及有效标识');
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
.document-page-editing {
    padding: 0;
}
.document-page-header {
    display: flex;
    align-items: center;
    min-height: 56px;
    padding: 0 24px;
    box-sizing: border-box;
    background: #fff;
}
.document-back {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    margin-right: 4px;
    padding: 0;
    border: 0;
    background: transparent;
    color: #0052d9;
    font-size: 20px;
    cursor: pointer;
}
.document-back-label {
    color: #999;
    font-weight: 400;
    cursor: pointer;
}
.document-page-header :deep(.arco-breadcrumb-item) {
    color: #333;
    font-weight: 400;
}
.document-editor-content {
    padding: 0 20px 20px;
}
.document-download-address {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-top: 4px;
    color: #86909c;
    font-size: 12px;
}
.document-download-address span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}
.document-download-address .arco-btn {
    flex-shrink: 0;
}
.document-page-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 16px;
    margin-bottom: 20px;
}
.document-toolbar-left {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 10px;
    min-width: 0;
}
.document-pagination {
    margin-top: 24px;
    justify-content: flex-end;
}
</style>
