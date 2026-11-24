<template>
    <div class="document-workbench">
        <a-spin :loading="loading || saving" class="workbench-spin">
            <div class="document-actions">
                <span class="document-reference">{{ libraryRef }}</span>
                <a-button @click="linksVisible = true">整体下载地址</a-button>
            </div>
            <div class="content-box">
                <div class="sidebar df-s0">
                    <div class="sidebar-title df ai-c jc-b">
                        <span>文档目录</span>
                    </div>
                    <div class="sidebar-list">
                        <div
                            v-for="item in docsFiles"
                            :key="item.path"
                            class="sidebar-item"
                            :class="{
                                active: item.path === activeDocPath,
                                'is-dragging': item.path === draggingDocPath,
                                'is-drop-target': item.path === dragOverDocPath,
                            }"
                            @click="selectDoc(item.path)"
                        >
                            <div class="sidebar-item-head df ai-c jc-b">
                                <div
                                    class="df ai-c sidebar-item-main"
                                    :draggable="!saving && canCreate"
                                    @dragstart="onDocDragStart(item, $event)"
                                    @dragover.prevent="onDocDragOver(item)"
                                    @dragleave="onDocDragLeave(item)"
                                    @drop.prevent="onDocDrop(item)"
                                    @dragend="onDocDragEnd"
                                >
                                    <div class="sidebar-item-title ml-8">{{ item.title }}</div>
                                </div>
                                <div class="sidebar-item-actions df ai-c">
                                    <a-tooltip content="修改文档标题" position="top">
                                        <a-button
                                            class="sidebar-action sidebar-edit"
                                            :class="{ 'is-disabled': saving }"
                                            type="text"
                                            size="mini"
                                            :disabled="saving || !item.can_write"
                                            @click.stop="openRenameDialog(item)"
                                        >
                                            <template #icon><icon-edit /></template>
                                        </a-button>
                                    </a-tooltip>
                                    <span @click.stop>
                                        <a-popconfirm
                                            content="确认要删除吗"
                                            type="warning"
                                            ok-text="确定"
                                            cancel-text="取消"
                                            position="lt"
                                            content-class="document-delete-popconfirm"
                                            :disabled="saving || !item.can_delete"
                                            :ok-button-props="{ status: 'danger' }"
                                            :cancel-button-props="{ type: 'secondary' }"
                                            @ok="deleteDoc(item.path)"
                                        >
                                            <a-tooltip content="删除文件" position="top">
                                                <a-button
                                                    class="sidebar-action sidebar-delete"
                                                    :class="{ 'is-disabled': saving }"
                                                    type="text"
                                                    size="mini"
                                                    :disabled="saving || !item.can_delete"
                                                >
                                                    <template #icon><icon-delete /></template>
                                                </a-button>
                                            </a-tooltip>
                                        </a-popconfirm>
                                    </span>
                                </div>
                            </div>
                        </div>
                        <div class="sidebar-add">
                            <a-button type="text" :disabled="saving || !canCreate" @click="openCreateDialog">
                                <template #icon><icon-plus /></template>
                                添加文档
                            </a-button>
                        </div>
                    </div>
                </div>

                <div v-if="docsFiles.length" class="editor-panel">
                    <div class="editor-header df ai-c jc-b">
                        <div>
                            <div class="editor-title">{{ currentTitle }}</div>
                        </div>
                        <div class="editor-actions df ai-c">
                            <a-button
                                type="primary"
                                :loading="saveButtonLoading"
                                :disabled="saving || !activeDoc?.can_write || !activeDoc?.loaded"
                                @click="saveDoc(activeDocPath)"
                                >保存</a-button
                            >
                        </div>
                    </div>

                    <div class="editor-body">
                        <markdown-editor
                            v-model="mdtxt"
                            :readonly="!activeDoc?.can_write"
                            :disabled="saving || !activeDoc?.loaded"
                        />
                    </div>
                </div>
                <div v-else class="editor-empty">
                    <a-empty description="暂无文档">
                        <a-button type="primary" :disabled="saving || !canCreate" @click="openCreateDialog">
                            <template #icon><icon-plus /></template>
                            添加文档
                        </a-button>
                    </a-empty>
                </div>
            </div>
        </a-spin>

        <a-modal v-model:visible="linksVisible" title="整体下载地址" :footer="false">
            <a-alert>公有文档库可直接下载；私有文档库需通过请求头携带所有者或创始人的登录凭据。</a-alert>
            <a-input :model-value="client.downloadURL(libraryRef)" readonly class="mt-16" />
        </a-modal>
        <a-modal v-model:visible="createDialog.show" title="添加文件" :width="560">
            <div class="create-form">
                <div class="form-row">
                    <div class="form-label">文件标题</div>
                    <a-input
                        v-model="createDialog.title"
                        :spellcheck="false"
                        placeholder="请输入文件标题，无需输入 .md"
                    ></a-input>
                </div>
            </div>
            <template #footer>
                <a-button @click="createDialog.show = false">取消</a-button>
                <a-button type="primary" @click="submitCreate">确认</a-button>
            </template>
        </a-modal>

        <a-modal v-model:visible="renameDialog.show" title="修改文档标题" :width="560">
            <div class="create-form">
                <div class="form-row">
                    <div class="form-label">文件标题</div>
                    <a-input
                        v-model="renameDialog.title"
                        :spellcheck="false"
                        placeholder="请输入文件标题，无需输入 .md"
                    ></a-input>
                </div>
            </div>
            <template #footer>
                <a-button @click="renameDialog.show = false">取消</a-button>
                <a-button type="primary" :loading="saving" @click="submitRename">确认</a-button>
            </template>
        </a-modal>
    </div>
</template>

<script>
import { IconDelete, IconEdit, IconPlus } from '@arco-design/web-vue/es/icon';
import { Message } from '@arco-design/web-vue';
import MarkdownEditor from './markdown-editor.vue';

const normalizePath = (path = '') =>
    path.replace(/\\/g, '/').replace(/^\/+/, '').replace(/\/+/g, '/').replace(/\/+$/, '');
const isDocMarkdownPath = (path = '') => /\.md$/i.test(normalizePath(path));
export default {
    name: 'DocumentWorkbench',
    components: { IconDelete, IconEdit, IconPlus, MarkdownEditor },
    props: {
        client: { type: Object, required: true },
        libraryRef: { type: String, required: true },
    },
    data() {
        return {
            loading: false,
            canCreate: false,
            selectionVersion: 0,
            disposed: false,
            linksVisible: false,
            saving: false,
            docsFiles: [],
            activeDocPath: '',
            draggingDocPath: '',
            dragOverDocPath: '',
            createDialog: {
                show: false,
                title: '',
            },
            renameDialog: {
                show: false,
                title: '',
                path: '',
            },
            mdtxt: '',
            savingPath: '',
        };
    },
    computed: {
        activeDoc() {
            return this.docsFiles.find((item) => item.path === this.activeDocPath) || null;
        },
        saveButtonLoading() {
            return this.savingPath === this.activeDocPath && !!this.activeDocPath;
        },
        currentTitle() {
            return this.activeDoc?.title || '文档';
        },
    },
    mounted() {
        this.getFiles().catch((error) => Message.error(error.message));
    },
    beforeUnmount() {
        this.disposed = true;
        this.selectionVersion++;
    },
    methods: {
        async getFiles() {
            this.loading = true;
            try {
                const result = await this.client.post('/file/list', { ref: this.libraryRef });
                if (this.disposed) return;
                this.persistEditorContent();
                const cache = new Map(this.docsFiles.map((file) => [file.path, file]));
                this.canCreate = result.can_create;
                this.docsFiles = (result.files || [])
                    .filter((file) => isDocMarkdownPath(file.path))
                    .map((file) => {
                        const previous = cache.get(file.path);
                        return previous && previous.dirty
                            ? { ...file, ...previous, can_write: file.can_write, can_delete: file.can_delete }
                            : { ...file, loaded: false, dirty: false };
                    });
                const nextPath =
                    this.docsFiles.find((file) => file.path === this.activeDocPath)?.path ||
                    this.docsFiles.find((file) => file.path === 'index.md')?.path ||
                    this.docsFiles[0]?.path;
                if (nextPath) await this.selectDoc(nextPath, true);
                else {
                    this.activeDocPath = '';
                    this.mdtxt = '';
                }
            } finally {
                this.loading = false;
            }
        },
        async saveDocsOrder() {
            if (this.saving) return;
            this.saving = true;
            try {
                await this.client.post('/file/order', {
                    ref: this.libraryRef,
                    paths: this.docsFiles.map((file) => file.path),
                });
                Message.success('排序已保存');
            } catch (error) {
                Message.error(error.message);
                await this.getFiles();
            } finally {
                this.saving = false;
            }
        },
        onDocDragStart(item, event) {
            if (!item || this.saving || !this.canCreate) {
                return;
            }
            this.draggingDocPath = item.path;
            this.dragOverDocPath = '';
            if (event?.dataTransfer) {
                event.dataTransfer.effectAllowed = 'move';
                event.dataTransfer.setData('text/plain', item.path);
            }
        },
        onDocDragOver(item) {
            if (!item || !this.draggingDocPath || item.path === this.draggingDocPath) {
                return;
            }
            this.dragOverDocPath = item.path;
        },
        onDocDragLeave(item) {
            if (item?.path === this.dragOverDocPath) {
                this.dragOverDocPath = '';
            }
        },
        async onDocDrop(item) {
            if (!item || !this.draggingDocPath || item.path === this.draggingDocPath) {
                this.onDocDragEnd();
                return;
            }
            let fromIndex = this.docsFiles.findIndex((doc) => doc.path === this.draggingDocPath);
            let toIndex = this.docsFiles.findIndex((doc) => doc.path === item.path);
            if (fromIndex < 0 || toIndex < 0) {
                this.onDocDragEnd();
                return;
            }

            this.persistEditorContent();
            let nextDocsFiles = [...this.docsFiles];
            let [moving] = nextDocsFiles.splice(fromIndex, 1);
            nextDocsFiles.splice(toIndex, 0, moving);
            this.docsFiles = nextDocsFiles;
            this.onDocDragEnd();
            await this.saveDocsOrder();
        },
        onDocDragEnd() {
            this.draggingDocPath = '';
            this.dragOverDocPath = '';
        },
        async selectDoc(path, force = false) {
            if (!path || (!force && path === this.activeDocPath)) return;
            this.persistEditorContent();
            const version = ++this.selectionVersion;
            this.activeDocPath = path;
            const file = this.activeDoc;
            if (!file) return;
            this.mdtxt = file.loaded ? file.content || '' : '';
            try {
                if (!file.loaded) {
                    const result = await this.client.post('/file/read', { ref: this.libraryRef, path });
                    if (this.disposed || version !== this.selectionVersion) return;
                    file.content = result.content;
                    file.baseContent = result.content;
                    file.loaded = true;
                }
                if (version !== this.selectionVersion) return;
                this.mdtxt = file.content || '';
            } catch (error) {
                Message.error(error.message);
            }
        },
        openCreateDialog() {
            this.persistEditorContent();
            this.createDialog.show = true;
            this.createDialog.title = '';
        },
        openRenameDialog(item) {
            if (!item || !item.path) {
                return;
            }
            this.persistEditorContent();
            this.renameDialog.show = true;
            this.renameDialog.title = item.title || '';
            this.renameDialog.path = item.path;
        },
        async submitCreate() {
            const title = (this.createDialog.title || '').trim().replace(/\.md$/i, '');
            if (!title || /[\\/]/.test(title)) {
                Message.warning('请输入不含路径分隔符的文件标题');
                return;
            }
            if (this.saving || !this.canCreate) return;
            const path = title.trim().replace(/\.md$/i, '').trim() + '.md';
            if (this.docsFiles.some((file) => file.path === path)) {
                Message.warning('同名文档已存在');
                return;
            }
            this.persistEditorContent();
            this.saving = true;
            try {
                await this.client.post('/file/save', { ref: this.libraryRef, path, title, content: '' });
                this.createDialog.show = false;
                await this.getFiles();
                await this.selectDoc(path);
                Message.success('添加成功');
            } catch (error) {
                Message.error(error.message);
            } finally {
                this.saving = false;
            }
        },
        async submitRename() {
            const file = this.docsFiles.find((item) => item.path === this.renameDialog.path);
            const title = (this.renameDialog.title || '').trim();
            if (!file?.can_write || !title || this.saving) return;
            this.saving = true;
            try {
                await this.client.post('/file/rename', {
                    ref: this.libraryRef,
                    path: file.path,
                    title,
                });
                file.title = title;
                this.renameDialog.show = false;
                await this.getFiles();
                Message.success('修改成功');
            } catch (error) {
                Message.error(error.message);
            } finally {
                this.saving = false;
            }
        },
        async deleteDoc(path) {
            const file = this.docsFiles.find((item) => item.path === path);
            if (!file?.can_delete || this.saving) return;
            this.saving = true;
            try {
                await this.client.post('/file/delete', { ref: this.libraryRef, path });
                this.docsFiles = this.docsFiles.filter((item) => item.path !== path);
                if (this.activeDocPath === path) this.activeDocPath = '';
                await this.getFiles();
                Message.success('删除成功');
            } catch (error) {
                Message.error(error.message);
            } finally {
                this.saving = false;
            }
        },
        persistEditorContent(activeDocPath = this.activeDocPath) {
            const file = this.docsFiles.find((item) => item.path === activeDocPath);
            if (file?.loaded && file.can_write) {
                file.content = this.mdtxt;
                file.dirty = file.content !== file.baseContent;
            }
        },
        async saveDoc(path) {
            const file = this.docsFiles.find((item) => item.path === path);
            if (!file?.can_write || !file.loaded || this.saving) return;
            this.persistEditorContent();
            this.saving = true;
            this.savingPath = path;
            try {
                await this.client.post('/file/save', {
                    ref: this.libraryRef,
                    path,
                    title: file.title,
                    content: file.content || '',
                });
                file.baseContent = file.content;
                file.dirty = false;
                await this.getFiles();
                Message.success('保存成功');
            } catch (error) {
                Message.error(error.message);
            } finally {
                this.saving = false;
                this.savingPath = '';
            }
        },
    },
};
</script>

<style scoped>
.df {
    display: flex;
}
.df-s0 {
    flex-shrink: 0;
}
.ai-c {
    align-items: center;
}
.jc-b {
    justify-content: space-between;
}
.ml-8 {
    margin-left: 8px;
}
.mt-16 {
    margin-top: 16px;
}

.document-actions {
    display: flex;
    gap: 10px;
    align-items: center;
    padding: 12px 0;
    flex-wrap: wrap;
}
.document-reference {
    color: #86909c;
    margin-right: auto;
}

.document-workbench {
    min-height: 100vh;
    background: #fff;
}

.workbench-spin {
    display: block;
}

.content-box {
    display: flex;
    border: 1px solid #dcdfe6;
    border-radius: 8px;
    overflow: hidden;
    min-height: 720px;
}

.sidebar {
    width: 280px;
    border-right: 1px solid #ebeef5;
    background: #fafafa;
    display: flex;
    flex-direction: column;
}

.sidebar-title {
    padding: 19px 18px;
    font-size: 14px;
    line-height: 1;
    color: #303133;
    border-bottom: 1px solid #ebeef5;
}

.sidebar-list {
    padding: 10px;
    overflow: auto;
    flex: 1;
}

.sidebar-item {
    padding: 12px;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s;
    margin-bottom: 8px;
    border: 1px solid transparent;
}

.sidebar-item:hover {
    background: #f4f4f5;
}

.sidebar-item.active {
    background: #ecf5ff;
    border-color: #d9ecff;
}

.sidebar-item.is-dragging {
    opacity: 0.45;
}

.sidebar-item.is-drop-target {
    background: #ecf5ff;
    border-color: #409eff;
}

.sidebar-item-head {
    gap: 12px;
}

.sidebar-item-main {
    flex: 1;
    min-width: 0;
    cursor: grab;
}

.sidebar-item-main[draggable='false'] {
    cursor: default;
}

.sidebar-item-main:active {
    cursor: grabbing;
}

.sidebar-item-title {
    font-size: 14px;
    color: #303133;
    word-break: break-all;
}

.sidebar-item-actions {
    gap: 8px;
    flex-shrink: 0;
    opacity: 0;
    transition: opacity 0.2s;
}

.sidebar-item:hover .sidebar-item-actions {
    opacity: 1;
}

.sidebar-action {
    width: 22px;
    height: 22px;
    padding: 0;
}

.sidebar-edit,
.sidebar-delete {
    cursor: pointer;
    flex-shrink: 0;
    transition: color 0.2s;
}

.sidebar-edit {
    color: #409eff;
}

.sidebar-delete {
    color: #f56c6c;
}

.sidebar-edit:hover {
    color: #66b1ff;
}

.sidebar-delete:hover {
    color: #f78989;
}

.sidebar-edit.is-disabled,
.sidebar-delete.is-disabled {
    opacity: 0.35 !important;
    cursor: not-allowed;
}

.sidebar-add .arco-btn {
    width: 100%;
}

.editor-panel {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-width: 0;
}

.editor-empty {
    display: flex;
    flex: 1;
    align-items: center;
    justify-content: center;
    min-width: 0;
}

.editor-header {
    padding: 10px 20px;
    height: 32px;
    border-bottom: 1px solid #ebeef5;
}

.editor-actions {
    flex-shrink: 0;
}

.editor-title {
    font-size: 18px;
    color: #303133;
}

.editor-body {
    flex: 1;
    min-height: 640px;
    position: relative;
    display: flex;
    flex-direction: column;
}

.form-row {
    display: flex;
    align-items: center;
    margin-bottom: 16px;
}

.form-label {
    width: 80px;
    color: #606266;
}
</style>
