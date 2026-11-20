<template>
    <div id="zpk-description" class="description-page">
        <a-spin :loading="loading || saving" class="description-spin">
            <div class="document-actions">
                <span class="document-reference">{{ libraryRef }}</span>
                <a-button @click="download('zip')">下载可读文档</a-button>
                <a-button @click="download('markdown')">下载 AI 上下文</a-button>
                <a-button
                    @click="
                        linksVisible = true;
                        loadLinks();
                    "
                    >下载地址</a-button
                >
                <a-button v-if="isAdmin && activeDoc" @click="permissionVisible = true">文件授权</a-button>
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
                                    :draggable="!saving"
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
                                            content-class="zpk-delete-popconfirm"
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
                        <mavon-editor
                            v-if="activeDoc?.can_write"
                            ref="descriptionEditor"
                            v-model="mdtxt"
                            class="mavon-description-editor"
                            :subfield="false"
                            :toolbarsFlag="true"
                            defaultOpen="preview"
                            previewBackground="#ffffff"
                            boxShadowStyle=""
                            :xssOptions="false"
                            placeholder="请输入文档内容"
                            :toolbars="editorToolbars"
                            @click.capture="handleMavonToolbarClick"
                            @change="patchMavonMarkdown"
                        />
                        <div v-else class="markdown-body document-readonly" v-html="markdownToHtml(mdtxt)"></div>
                    </div>
                </div>
                <div v-else class="editor-empty">
                    <a-empty description="暂无应用文档">
                        <a-button type="primary" :disabled="saving || !canCreate" @click="openCreateDialog">
                            <template #icon><icon-plus /></template>
                            添加文档
                        </a-button>
                    </a-empty>
                </div>
            </div>
        </a-spin>

        <a-modal v-model:visible="linksVisible" title="整体下载地址" :footer="false">
            <a-alert>链接仅包含签发时可读的文件，有效期 24 小时；后续撤权会影响下载。</a-alert>
            <a-button class="mt-16" @click="createLink">生成下载地址</a-button>
            <a-input v-if="linkURL" :model-value="linkURL" readonly class="mt-16" />
            <div v-for="link in links" :key="link.id" class="document-link-row">
                <span>#{{ link.id }} · {{ link.expires_at }} · {{ link.revoked ? '已撤销' : '有效' }}</span>
                <a-button v-if="!link.revoked" type="text" status="danger" @click="revokeLink(link.id)">撤销</a-button>
            </div>
        </a-modal>
        <a-modal v-model:visible="permissionVisible" title="设置文件授权" @ok="setPermission">
            <a-alert>文件独立授权优先于文档库授权；未获授权的用户将不能继承上层权限。</a-alert>
            <a-input v-model="targetUserID" placeholder="被授权用户 ID" class="mt-16" />
            <a-checkbox-group v-model="permissionActions" :options="['read', 'write', 'delete']" class="mt-16" />
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
import { renderMarkdown } from '../utils/markdown';

const DOCS_DIR = 'docs';
const MAVON_ALIGN_TYPES = {
    alignleft: 'left',
    aligncenter: 'center',
    alignright: 'right',
};
const patchedMavonEditors = new WeakSet();
const normalizePath = (path = '') =>
    path.replace(/\\/g, '/').replace(/^\/+/, '').replace(/\/+/g, '/').replace(/\/+$/, '');
const isDocMarkdownPath = (path = '') => /\.md$/i.test(normalizePath(path));
export default {
    name: 'DocumentWorkbench',
    components: { IconDelete, IconEdit, IconPlus },
    props: {
        client: { type: Object, required: true },
        libraryRef: { type: String, required: true },
        isAdmin: { type: Boolean, default: false },
    },
    data() {
        return {
            loading: false,
            libraryRevision: 0,
            canCreate: false,
            selectionVersion: 0,
            disposed: false,
            linksVisible: false,
            links: [],
            linkURL: '',
            permissionVisible: false,
            targetUserID: '',
            permissionActions: ['read'],
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
            editorToolbars: {
                bold: true,
                italic: true,
                header: true,
                underline: true,
                strikethrough: true,
                mark: true,
                superscript: true,
                subscript: true,
                quote: true,
                ol: true,
                ul: true,
                link: true,
                imagelink: false,
                code: false,
                table: true,
                undo: true,
                redo: true,
                trash: false,
                save: false,
                alignleft: true,
                aligncenter: true,
                alignright: true,
                navigation: false,
                subfield: false,
                fullscreen: false,
                readmodel: false,
                htmlcode: false,
                help: false,
                preview: true,
            },
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
        this.$nextTick(() => {
            this.patchMavonMarkdown();
        });
    },
    beforeUnmount() {
        this.disposed = true;
        this.selectionVersion++;
    },
    methods: {
        async download(format) {
            try {
                await this.client.download(this.libraryRef, format);
            } catch (error) {
                Message.error(error.message);
            }
        },
        async loadLinks() {
            try {
                this.links = (await this.client.post('/link/list', { ref: this.libraryRef })) || [];
            } catch (error) {
                Message.error(error.message);
            }
        },
        async createLink() {
            try {
                const result = await this.client.post('/link/create', { ref: this.libraryRef, hours: 24 });
                this.linkURL = this.client.linkURL(result.token);
                await this.loadLinks();
            } catch (error) {
                Message.error(error.message);
            }
        },
        async revokeLink(id) {
            try {
                await this.client.post('/link/revoke', { id });
                this.linkURL = '';
                await this.loadLinks();
            } catch (error) {
                Message.error(error.message);
            }
        },
        async setPermission() {
            const targetUserID = Number(this.targetUserID);
            if (!Number.isInteger(targetUserID) || targetUserID <= 0) {
                Message.warning('请输入被授权用户 ID');
                return;
            }
            try {
                await this.client.post('/permission/set', {
                    ref: this.libraryRef,
                    path: this.activeDocPath,
                    target_user_id: targetUserID,
                    actions: this.permissionActions,
                });
                this.permissionVisible = false;
                await this.getFiles();
                Message.success('授权已保存');
            } catch (error) {
                Message.error(error.message);
            }
        },
        async getFiles() {
            this.loading = true;
            try {
                const result = await this.client.post('/file/list', { ref: this.libraryRef });
                if (this.disposed) return;
                const cache = new Map(this.docsFiles.map((file) => [file.path, file]));
                this.libraryRevision = result.revision;
                this.canCreate = result.can_create;
                this.docsFiles = (result.files || [])
                    .filter((file) => isDocMarkdownPath(file.path))
                    .map((file) => {
                        const previous = cache.get(file.path);
                        return previous && (previous.revision === file.revision || previous.dirty)
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
                    revision: this.libraryRevision,
                });
                this.libraryRevision++;
                Message.success('排序已保存');
            } catch (error) {
                Message.error(error.message);
                await this.getFiles();
            } finally {
                this.saving = false;
            }
        },
        onDocDragStart(item, event) {
            if (!item || this.saving) {
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
                    file.revision = result.revision;
                    file.loaded = true;
                }
                if (version !== this.selectionVersion) return;
                this.mdtxt = file.content || '';
                this.$nextTick(() => this.patchMavonMarkdown());
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
            this.persistEditorContent();
            this.saving = true;
            const path = DOCS_DIR + '/' + title.trim().replace(/\.md$/i, '').trim() + '.md';
            try {
                await this.client.post('/file/save', { ref: this.libraryRef, path, title, content: '', revision: 0 });
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
                    revision: file.revision,
                });
                file.title = title;
                file.revision++;
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
                await this.client.post('/file/delete', { ref: this.libraryRef, path, revision: file.revision });
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
        markdownToHtml: renderMarkdown,
        handleMavonToolbarClick(event) {
            let target = event?.target;
            if (!target?.classList) {
                return;
            }

            let alignType = '';
            if (target.classList.contains('fa-mavon-align-left')) {
                alignType = 'alignleft';
            } else if (target.classList.contains('fa-mavon-align-center')) {
                alignType = 'aligncenter';
            } else if (target.classList.contains('fa-mavon-align-right')) {
                alignType = 'alignright';
            }
            if (!alignType) {
                return;
            }

            event.preventDefault();
            event.stopPropagation();
            event.stopImmediatePropagation && event.stopImmediatePropagation();
            this.insertMavonAlignBlock(alignType);
        },
        insertMavonAlignBlock(type) {
            let align = MAVON_ALIGN_TYPES[type];
            let editor = this.$refs.descriptionEditor;
            let textarea = editor?.getTextareaDom?.();
            if (!align || !editor || !textarea || this.saving) {
                return;
            }

            editor.insertText(textarea, {
                prefix: '<div align="' + align + '">',
                subfix: '</div>',
                str: '',
                type,
            });
            this.mdtxt = editor.d_value || textarea.value || this.mdtxt;
            this.$nextTick(() => {
                this.patchMavonMarkdown();
            });
        },
        patchMavonToolbar(editor = this.$refs.descriptionEditor) {
            if (!editor || patchedMavonEditors.has(editor) || typeof editor.toolbar_left_click !== 'function') {
                return;
            }
            let originalToolbarClick = editor.toolbar_left_click.bind(editor);
            try {
                editor.toolbar_left_click = (type) => {
                    if (MAVON_ALIGN_TYPES[type]) {
                        this.insertMavonAlignBlock(type);
                        return;
                    }
                    originalToolbarClick(type);
                };
                patchedMavonEditors.add(editor);
            } catch (error) {
                patchedMavonEditors.add(editor);
            }
        },
        patchMavonMarkdown() {
            let editor = this.$refs.descriptionEditor;
            if (!editor) {
                return;
            }
            this.patchMavonToolbar(editor);
            let markdownIt = editor.markdownIt;
            if (markdownIt && !markdownIt.__githubPreviewPatched) {
                markdownIt.render = (src = '') => this.markdownToHtml(src);
                markdownIt.__githubPreviewPatched = true;
            }
            editor.d_render = this.markdownToHtml(editor.d_value || this.mdtxt || '');
        },
        async saveDoc(path) {
            const file = this.docsFiles.find((item) => item.path === path);
            if (!file?.can_write || !file.loaded || this.saving) return;
            this.persistEditorContent();
            this.saving = true;
            this.savingPath = path;
            try {
                const result = await this.client.post('/file/save', {
                    ref: this.libraryRef,
                    path,
                    title: file.title,
                    content: file.content || '',
                    revision: file.revision,
                });
                file.revision = result.revision;
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
.document-link-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: 12px;
}
.document-readonly {
    padding: 24px;
    overflow: auto;
    height: 100%;
    box-sizing: border-box;
}

.description-page {
    min-height: 100vh;
    background: #fff;
}

.description-spin {
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

.mavon-description-editor {
    flex: 1;
    min-height: 640px;
    border: 0;
    z-index: 1;
}

.mavon-description-editor :deep(.v-note-op) {
    display: flex !important;
    align-items: center;
    flex-wrap: nowrap;
    min-height: 36px;
    padding: 0 8px;
    overflow-x: auto;
    overflow-y: visible;
    white-space: nowrap;
}

.mavon-description-editor :deep(.op-icon) {
    width: 24px;
    min-width: 24px;
    height: 28px;
    padding: 6px 4px;
    margin: 0 1px;
}

.mavon-description-editor :deep(.op-icon-divider) {
    display: none !important;
}

.mavon-description-editor :deep(.v-left-item),
.mavon-description-editor :deep(.v-right-item) {
    float: none !important;
    display: flex !important;
    align-items: center;
    flex-wrap: nowrap;
    flex: 0 0 auto;
    height: 100%;
    min-height: 36px;
    padding: 0 !important;
}

.mavon-description-editor :deep(.v-right-item) {
    margin-left: 4px;
}

.mavon-description-editor :deep(.v-note-panel) {
    min-height: 580px;
}

.mavon-description-editor :deep([align='left']) {
    text-align: left;
}

.mavon-description-editor :deep([align='center']) {
    text-align: center;
}

.mavon-description-editor :deep([align='right']) {
    text-align: right;
}

.mavon-description-editor :deep([align='justify']) {
    text-align: justify;
}

.mavon-description-editor :deep(.markdown-alert) {
    padding: 8px 16px;
    border-left-width: 4px;
    color: #303133;
    background: #fff;
}

.mavon-description-editor :deep(.markdown-alert-title) {
    margin: 0 0 8px;
    font-weight: 600;
}

.mavon-description-editor :deep(.markdown-alert-note) {
    border-left-color: #0969da;
}

.mavon-description-editor :deep(.markdown-alert-note .markdown-alert-title) {
    color: #0969da;
}

.mavon-description-editor :deep(.markdown-alert-tip) {
    border-left-color: #1a7f37;
}

.mavon-description-editor :deep(.markdown-alert-tip .markdown-alert-title) {
    color: #1a7f37;
}

.mavon-description-editor :deep(.markdown-alert-important) {
    border-left-color: #8250df;
}

.mavon-description-editor :deep(.markdown-alert-important .markdown-alert-title) {
    color: #8250df;
}

.mavon-description-editor :deep(.markdown-alert-warning) {
    border-left-color: #9a6700;
}

.mavon-description-editor :deep(.markdown-alert-warning .markdown-alert-title) {
    color: #9a6700;
}

.mavon-description-editor :deep(.markdown-alert-caution) {
    border-left-color: #cf222e;
}

.mavon-description-editor :deep(.markdown-alert-caution .markdown-alert-title) {
    color: #cf222e;
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
