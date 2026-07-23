<template>
    <div id="zpk-description" class="description-page">
        <a-spin :loading="loading || saving" class="description-spin">
            <div class="content-box">
                <div class="sidebar df-s0">
                    <div class="sidebar-title df ai-c jc-b">
                        <span>文档目录</span>
                    </div>
                    <div class="sidebar-list">
                        <div v-for="item in docsFiles" :key="item.path" class="sidebar-item" :class="{
                            active: item.path === activeDocPath,
                            'is-dragging': item.path === draggingDocPath,
                            'is-drop-target': item.path === dragOverDocPath,
                        }" @click="selectDoc(item.path)">
                            <div class="sidebar-item-head df ai-c jc-b">
                                <div class="df ai-c sidebar-item-main" :draggable="!saving"
                                    @dragstart="onDocDragStart(item, $event)" @dragover.prevent="onDocDragOver(item)"
                                    @dragleave="onDocDragLeave(item)" @drop.prevent="onDocDrop(item)"
                                    @dragend="onDocDragEnd">
                                    <div class="sidebar-item-title ml-8">{{ item.title }}</div>
                                </div>
                                <div class="sidebar-item-actions df ai-c">
                                    <a-tooltip content="修改文档标题" position="top">
                                        <a-button class="sidebar-action sidebar-edit" :class="{ 'is-disabled': saving }"
                                            type="text" size="mini" :disabled="saving"
                                            @click.stop="openRenameDialog(item)">
                                            <template #icon><icon-edit /></template>
                                        </a-button>
                                    </a-tooltip>
                                    <span @click.stop>
                                        <a-popconfirm content="确认要删除吗" type="warning" ok-text="确定"
                                            cancel-text="取消" content-class="zpk-delete-popconfirm"
                                            :ok-button-props="{ status: 'danger' }"
                                            :cancel-button-props="{ type: 'secondary' }" @ok="deleteDoc(item.path)">
                                            <a-tooltip content="删除文件" position="top">
                                                <a-button class="sidebar-action sidebar-delete"
                                                    :class="{ 'is-disabled': saving }" type="text" size="mini"
                                                    :disabled="saving">
                                                    <template #icon><icon-delete /></template>
                                                </a-button>
                                            </a-tooltip>
                                        </a-popconfirm>
                                    </span>
                                </div>
                            </div>
                        </div>
                        <div class="sidebar-add">
                            <a-button type="text" :disabled="saving" @click="openCreateDialog">
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
                            <a-button type="primary" :loading="saveButtonLoading" :disabled="saving"
                                @click="saveDoc(currentEditingPath)">保存</a-button>
                        </div>
                    </div>

                    <div class="editor-body">
                        <mavon-editor ref="descriptionEditor" v-model="mdtxt" class="mavon-description-editor"
                            :subfield="false" :toolbarsFlag="true" defaultOpen="preview" previewBackground="#ffffff"
                            boxShadowStyle="" :xssOptions="false" placeholder="请输入文档内容" :toolbars="editorToolbars"
                            @click.capture="handleMavonToolbarClick" @change="patchMavonMarkdown" />
                    </div>
                </div>
                <div v-else class="editor-empty">
                    <a-empty description="暂无应用文档">
                        <a-button type="primary" :disabled="saving" @click="openCreateDialog">
                            <template #icon><icon-plus /></template>
                            添加文档
                        </a-button>
                    </a-empty>
                </div>
            </div>
        </a-spin>

        <a-modal v-model:visible="createDialog.show" title="添加文件" :width="560">
            <div class="create-form">
                <div class="form-row">
                    <div class="form-label">文件标题</div>
                    <a-input v-model="createDialog.title" :spellcheck="false"
                        placeholder="请输入文件标题，无需输入 .md"></a-input>
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
                    <a-input v-model="renameDialog.title" :spellcheck="false"
                        placeholder="请输入文件标题，无需输入 .md"></a-input>
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
import MarkdownIt from 'markdown-it';
import markdownItTaskLists from 'markdown-it-task-lists';
import myAxios from '@/utils';
import { IconDelete, IconEdit, IconPlus } from '@arco-design/web-vue/es/icon';
import { messageError, messageSuccess, messageWarning } from '@/utils/ui-feedback';

const DOCS_DIR = 'docs';
const DOCS_ORDER_PATH = DOCS_DIR + '/.order';
const MAVON_ALIGN_TYPES = {
    alignleft: 'left',
    aligncenter: 'center',
    alignright: 'right',
};
const patchedMavonEditors = new WeakSet();
const GITHUB_ALERTS = {
    note: 'Note',
    tip: 'Tip',
    important: 'Important',
    warning: 'Warning',
    caution: 'Caution',
};
const markdownItGithubAlerts = (md) => {
    md.core.ruler.after('inline', 'github_alerts', (state) => {
        state.tokens.forEach((token, index) => {
            if (token.type !== 'blockquote_open') { return }

            let inlineTokenIndex = state.tokens.findIndex((item, childIndex) => {
                return childIndex > index && item.type === 'inline'
                    && /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*(?:\n|$)/i.test(item.content || '');
            });
            if (inlineTokenIndex < 0) { return }

            let inlineToken = state.tokens[inlineTokenIndex];
            let matched = inlineToken.content.match(/^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*(?:\n|$)/i);
            if (!matched) { return }

            let type = matched[1].toLowerCase();
            token.attrJoin('class', 'markdown-alert markdown-alert-' + type);
            inlineToken.content = inlineToken.content.replace(/^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*(?:\n|$)/i, '');

            let children = inlineToken.children || [];
            let firstTextIndex = children.findIndex(item => item.type === 'text'
                && /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*$/i.test(item.content || ''));
            if (firstTextIndex >= 0) {
                children.splice(firstTextIndex, 1);
                if (['softbreak', 'hardbreak'].includes(children[firstTextIndex]?.type)) {
                    children.splice(firstTextIndex, 1);
                }
            }

            let titleToken = new state.Token('html_block', '', 0);
            titleToken.content = '<p class="markdown-alert-title">' + GITHUB_ALERTS[type] + '</p>\n';
            state.tokens.splice(index + 1, 0, titleToken);
        });
    });
};
const createInlineWrapRule = ({ marker, tag, name }) => {
    return (state, silent) => {
        let start = state.pos;
        if (!state.src.startsWith(marker, start)) { return false }
        if (marker === '~' && state.src[start + 1] === '~') { return false }

        let contentStart = start + marker.length;
        let close = contentStart;
        while ((close = state.src.indexOf(marker, close)) >= 0) {
            if (close === contentStart) {
                close += marker.length;
                continue;
            }
            if (marker === '~' && state.src[close + 1] === '~') {
                close += marker.length;
                continue;
            }

            if (!silent) {
                let oldPos = state.pos;
                let oldMax = state.posMax;
                state.push(name + '_open', tag, 1);
                state.pos = contentStart;
                state.posMax = close;
                state.md.inline.tokenize(state);
                state.pos = oldPos;
                state.posMax = oldMax;
                state.push(name + '_close', tag, -1);
            }

            state.pos = close + marker.length;
            return true;
        }

        return false;
    };
};
const markdownItMavonSyntax = (md) => {
    md.inline.ruler.before('emphasis', 'mavon_mark', createInlineWrapRule({
        marker: '==',
        tag: 'mark',
        name: 'mark',
    }));
    md.inline.ruler.before('emphasis', 'mavon_sup', createInlineWrapRule({
        marker: '^',
        tag: 'sup',
        name: 'sup',
    }));
    md.inline.ruler.before('emphasis', 'mavon_sub', createInlineWrapRule({
        marker: '~',
        tag: 'sub',
        name: 'sub',
    }));
};
const markdownParser = new MarkdownIt({
    html: true,
    linkify: true,
    breaks: false,
}).use(markdownItTaskLists, {
    enabled: true,
    label: true,
}).use(markdownItMavonSyntax).use(markdownItGithubAlerts);

const renderAlignedBlock = (align, content = '') => {
    let body = renderMarkdownWithGithubAlerts(content).trim();
    return '<div align="' + align + '">\n' + body + '\n</div>';
};

const renderGithubAlertBlock = (type, content = '') => {
    let body = markdownParser.render(content.trim());
    return '<blockquote class="markdown-alert markdown-alert-' + type + '">\n'
        + '<p class="markdown-alert-title">' + GITHUB_ALERTS[type] + '</p>\n'
        + body
        + '</blockquote>';
};
const normalizeEscapedBlockquoteMarkers = (content = '') => String(content || '').replace(/^(\s*)&gt;\s?/gm, '$1> ');
const renderMarkdownWithGithubAlerts = (content = '') => {
    let lines = normalizeEscapedBlockquoteMarkers(content).split('\n');
    let output = [];

    for (let index = 0; index < lines.length; index++) {
        let alignMatched = lines[index].replace(/\s+$/, '').match(/^:::\s*hljs-(left|center|right)\s*$/i);
        if (alignMatched) {
            let align = alignMatched[1].toLowerCase();
            let alignLines = [];
            index++;

            while (index < lines.length && !/^:::\s*$/.test(lines[index].trim())) {
                alignLines.push(lines[index]);
                index++;
            }

            output.push(renderAlignedBlock(align, alignLines.join('\n')));
            continue;
        }

        let matched = lines[index].replace(/\s+$/, '').match(/^>\s*\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*$/i);
        if (!matched) {
            output.push(lines[index]);
            continue;
        }

        let type = matched[1].toLowerCase();
        let alertLines = [];
        index++;

        while (index < lines.length && /^>\s?/.test(lines[index])) {
            alertLines.push(lines[index].replace(/^>\s?/, ''));
            index++;
        }
        index--;

        output.push(renderGithubAlertBlock(type, alertLines.join('\n')));
    }

    return markdownParser.render(output.join('\n'));
};
const normalizePath = (path = '') => path.replace(/\\/g, '/').replace(/^\/+/, '').replace(/\/+/g, '/').replace(/\/+$/, '');
const isDocMarkdownPath = (path = '') => /^docs\/[^/]+\.md$/i.test(normalizePath(path));
const createDocFile = (title, content = '') => {
    let cleanTitle = (title || '').trim().replace(/\.md$/i, '').trim();
    return {
        title: cleanTitle,
        path: DOCS_DIR + '/' + cleanTitle + '.md',
        content,
    };
};

export default {
    name: 'ProductDescriptionPage',
    components: { IconDelete, IconEdit, IconPlus },
    props: {
        identifie: {
            type: String,
            default: '',
        },
    },
    data() {
        return {
            vtitle: '',
            loading: false,
            saving: false,
            source: 'remote',
            docsFiles: [],
            docsOrderContent: '',
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
            isEditing: true,
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
            xssOptions: {
                stripIgnoreTag: false,
                stripIgnoreTagBody: false,
            }
        }
    },
    computed: {
        formulaIdentifie() {
            return this.identifie || this.$route.query.id || '';
        },
        hasDocsFiles() {
            return this.docsFiles.length > 0;
        },
        activeDoc() {
            return this.docsFiles.find(item => item.path === this.activeDocPath) || null;
        },
        currentEditingPath() {
            return this.activeDocPath;
        },
        saveButtonLoading() {
            return this.savingPath === this.currentEditingPath && !!this.currentEditingPath;
        },
        currentTitle() {
            return this.activeDoc?.title || '文档';
        },
        toolbarDescription() {
            return '默认显示编辑器内置预览，可通过工具栏切换编辑。';
        },
        currentContent() {
            return this.activeDoc?.content || '';
        },
    },
    created() {
        this.vtitle = this.$route.query.vtitle || '';
    },
    mounted() {
        this.getFiles();
        this.$nextTick(() => {
            this.patchMavonMarkdown();
        });
    },
    methods: {
        async getFiles() {

            this.loading = true;
            try {
                let res = await myAxios.post('/respo/share-file/path-tree', {
                    identifie: this.formulaIdentifie,
                });
                let list = res?.data?.data?.list || {};

                this.docsOrderContent = list[DOCS_ORDER_PATH] || '';
                this.docsFiles = this.parseDocsFiles(list);
                let previousPath = this.activeDocPath || '';
                let nextPath = this.docsFiles.find(item => item.path === previousPath)?.path
                    || this.docsFiles[0]?.path
                    || '';
                this.activeDocPath = nextPath;
                this.inputContent(this.activeDoc?.content || '');
                this.$nextTick(() => {
                    this.patchMavonMarkdown();
                });
            } finally {
                this.loading = false;
            }
        },
        parseDocsFiles(list) {
            let fileMap = {};
            Object.keys(list || {}).forEach(path => {
                let normalized = normalizePath(path);
                if (!isDocMarkdownPath(normalized)) { return }
                fileMap[normalized] = list[path] || '';
            });

            let docsOrder = this.parseDocsOrder(list?.[DOCS_ORDER_PATH] || '');
            let pathSet = new Set(Object.keys(fileMap));
            let orderedPaths = docsOrder.filter(path => pathSet.has(path));
            let orderedPathSet = new Set(orderedPaths);
            let unorderedPaths = Object.keys(fileMap)
                .filter(path => !orderedPathSet.has(path))
                .sort((a, b) => a.localeCompare(b, 'zh-Hans-CN'));
            orderedPaths = [...orderedPaths, ...unorderedPaths];

            return orderedPaths.map(path => {
                let title = path.replace(/^docs\//, '').replace(/\.md$/i, '');
                return {
                    title,
                    path,
                    content: fileMap[path] || '',
                };
            });
        },
        parseDocsOrder(content = '') {
            return String(content || '')
                .split(',')
                .map(path => normalizePath(path))
                .filter(path => isDocMarkdownPath(path));
        },
        getDocsOrderContent(files = this.docsFiles) {
            return files.map(item => normalizePath(item.path)).filter(Boolean).join(',');
        },
        postDocsOrderFile(content = this.getDocsOrderContent()) {
            this.docsOrderContent = content;
            return myAxios.post('/respo/share-file/file', {
                identifie: this.formulaIdentifie,
                filename: DOCS_ORDER_PATH,
                content,
            });
        },
        async saveDocsOrder(successText = '排序已保存') {
            if (this.saving) { return }
            this.persistEditorContent();
            this.saving = true;
            this.savingPath = DOCS_ORDER_PATH;
            try {
                await this.postDocsOrderFile(this.getDocsOrderContent());
                if (successText) {
                    messageSuccess(successText);
                }
            } catch (error) {
                await this.getFiles();
            } finally {
                this.saving = false;
                this.savingPath = '';
            }
        },
        onDocDragStart(item, event) {
            if (!item || this.saving) { return }
            this.draggingDocPath = item.path;
            this.dragOverDocPath = '';
            if (event?.dataTransfer) {
                event.dataTransfer.effectAllowed = 'move';
                event.dataTransfer.setData('text/plain', item.path);
            }
        },
        onDocDragOver(item) {
            if (!item || !this.draggingDocPath || item.path === this.draggingDocPath) { return }
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
            let fromIndex = this.docsFiles.findIndex(doc => doc.path === this.draggingDocPath);
            let toIndex = this.docsFiles.findIndex(doc => doc.path === item.path);
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
            await this.saveDocsOrder('排序已保存');
        },
        onDocDragEnd() {
            this.draggingDocPath = '';
            this.dragOverDocPath = '';
        },
        selectDoc(path) {
            if (!path || path === this.activeDocPath) { return }
            this.persistEditorContent();
            this.activeDocPath = path;
            this.inputContent(this.activeDoc?.content || '');
            this.$nextTick(() => {
                this.patchMavonMarkdown();
            });
        },
        startEditDoc(path) {
            if (!path) { return }
            if (path !== this.activeDocPath) {
                this.selectDoc(path);
            }
            let content = this.currentContent;
            this.isEditing = true;
            this.inputContent(content);
        },
        cancelEdit() {
            this.isEditing = true;
            this.inputContent(this.currentContent);
            this.$nextTick(() => {
                this.patchMavonMarkdown();
            });
        },
        openCreateDialog() {
            this.persistEditorContent();
            this.createDialog.show = true;
            this.createDialog.title = '';
        },
        openRenameDialog(item) {
            if (!item || !item.path) { return }
            this.persistEditorContent();
            this.renameDialog.show = true;
            this.renameDialog.title = item.title || '';
            this.renameDialog.path = item.path;
        },
        async submitCreate() {
            let title = (this.createDialog.title || '').trim().replace(/\.md$/i, '').trim();
            if (!title) {
                messageWarning('请输入文件标题');
                return;
            }
            if (/[\\/]/.test(title)) {
                messageWarning('文件标题不能包含“/”或“\\”');
                return;
            }

            let file = createDocFile(title, '');
            let targetPath = normalizePath(file.path);
            let exists = this.docsFiles.some(item => normalizePath(item.path) === targetPath || item.title === title);
            if (exists) {
                messageError('已存在同名文档');
                return;
            }

            if (this.saving) { return }
            this.saving = true;
            try {
                let latest = await myAxios.post('/respo/share-file/path-tree', {
                    identifie: this.formulaIdentifie,
                });
                let latestList = latest?.data?.data?.list || {};
                let remoteExists = Object.keys(latestList).some(path => normalizePath(path) === targetPath);
                if (remoteExists) {
                    messageError('已存在同名文档');
                    await this.getFiles();
                    return;
                }

                await myAxios.post('/respo/share-file/file', {
                    identifie: this.formulaIdentifie,
                    filename: file.path,
                    content: ' ',
                });
                this.docsFiles.push(file);
                this.docsFiles = [...this.docsFiles];
                await this.postDocsOrderFile(this.getDocsOrderContent());
                this.createDialog.show = false;
                messageSuccess('添加成功');

                await this.$nextTick();
                this.selectDoc(file.path);
                this.startEditDoc(file.path);
            } finally {
                this.saving = false;
            }
        },
        async submitRename() {
            let oldPath = normalizePath(this.renameDialog.path || '');
            let title = (this.renameDialog.title || '').trim().replace(/\.md$/i, '').trim();
            if (!oldPath) {
                this.renameDialog.show = false;
                return;
            }
            if (!title) {
                messageWarning('请输入文件标题');
                return;
            }
            if (/[\\/]/.test(title)) {
                messageWarning('文件标题不能包含“/”或“\\”');
                return;
            }

            let current = this.docsFiles.find(item => normalizePath(item.path) === oldPath);
            if (!current) {
                messageError('文档不存在');
                this.renameDialog.show = false;
                await this.getFiles();
                return;
            }

            let targetFile = createDocFile(title, current.content || '');
            let targetPath = normalizePath(targetFile.path);
            if (targetPath === oldPath) {
                this.renameDialog.show = false;
                return;
            }

            let exists = this.docsFiles.some(item => {
                let path = normalizePath(item.path);
                return path !== oldPath && (path === targetPath || item.title === title);
            });
            if (exists) {
                messageError('已存在同名文档');
                return;
            }

            if (this.saving) { return }
            this.persistEditorContent();
            current = this.docsFiles.find(item => normalizePath(item.path) === oldPath);
            this.saving = true;
            this.savingPath = oldPath;
            try {
                let latest = await myAxios.post('/respo/share-file/path-tree', {
                    identifie: this.formulaIdentifie,
                });
                let latestList = latest?.data?.data?.list || {};
                let remoteExists = Object.keys(latestList).some(path => {
                    let normalized = normalizePath(path);
                    return normalized !== oldPath && normalized === targetPath;
                });
                if (remoteExists) {
                    messageError('已存在同名文档');
                    await this.getFiles();
                    return;
                }

                await myAxios.post('/respo/share-file/file', {
                    identifie: this.formulaIdentifie,
                    filename: targetFile.path,
                    content: current?.content || '',
                });
                await myAxios.post('/respo/share-file/file', {
                    identifie: this.formulaIdentifie,
                    filename: oldPath,
                    content: '',
                });

                this.docsFiles = this.docsFiles.map(item => {
                    if (normalizePath(item.path) !== oldPath) { return item }
                    return {
                        title,
                        path: targetPath,
                        content: current?.content || '',
                    };
                });
                await this.postDocsOrderFile(this.getDocsOrderContent());

                if (normalizePath(this.activeDocPath) === oldPath) {
                    this.activeDocPath = targetPath;
                }
                this.renameDialog.show = false;
                this.renameDialog.path = '';
                this.renameDialog.title = '';
                messageSuccess('修改成功');
                await this.getFiles();
            } finally {
                this.saving = false;
                this.savingPath = '';
            }
        },
        async deleteDoc(path) {
            if (!path) { return }
            this.persistEditorContent();
            let current = this.docsFiles.find(item => item.path === path);
            if (!current) { return }
            if (this.saving) { return }
            let currentIndex = this.docsFiles.findIndex(item => item.path === path);
            if (currentIndex < 0) { return }
            this.docsFiles.splice(currentIndex, 1);
            this.docsFiles = [...this.docsFiles];
            let nextDoc = this.docsFiles[currentIndex] || this.docsFiles[currentIndex - 1] || null;
            this.isEditing = true;
            if (this.docsFiles.length) {
                this.activeDocPath = nextDoc?.path || '';
                this.inputContent(nextDoc?.content || '');
            } else {
                this.activeDocPath = '';
                this.inputContent('');
            }
            await this.saveMultipleDocs('', [path], '删除成功');
        },
        persistEditorContent(activeDocPath = this.activeDocPath) {
            let content = this.mdtxt;
            if (!activeDocPath) { return }
            let current = this.docsFiles.find(item => item.path === activeDocPath);
            if (current) {
                current.content = content;
            }
        },
        inputContent(content = '') {
            this.mdtxt = content;
        },
        markdownToHtml(content = '') {
            return renderMarkdownWithGithubAlerts(content);
        },
        handleMavonToolbarClick(event) {
            let target = event?.target;
            if (!target?.classList) { return }

            let alignType = '';
            if (target.classList.contains('fa-mavon-align-left')) {
                alignType = 'alignleft';
            } else if (target.classList.contains('fa-mavon-align-center')) {
                alignType = 'aligncenter';
            } else if (target.classList.contains('fa-mavon-align-right')) {
                alignType = 'alignright';
            }
            if (!alignType) { return }

            event.preventDefault();
            event.stopPropagation();
            event.stopImmediatePropagation && event.stopImmediatePropagation();
            this.insertMavonAlignBlock(alignType);
        },
        insertMavonAlignBlock(type) {
            let align = MAVON_ALIGN_TYPES[type];
            let editor = this.$refs.descriptionEditor;
            let textarea = editor?.getTextareaDom?.();
            if (!align || !editor || !textarea || this.saving) { return }

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
            if (!editor || patchedMavonEditors.has(editor) || typeof editor.toolbar_left_click !== 'function') { return }
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
            if (!editor) { return }
            this.patchMavonToolbar(editor);
            let markdownIt = editor.markdownIt;
            if (markdownIt && !markdownIt.__githubPreviewPatched) {
                markdownIt.render = (src = '') => this.markdownToHtml(src);
                markdownIt.__githubPreviewPatched = true;
            }
            editor.d_render = this.markdownToHtml(editor.d_value || this.mdtxt || '');
        },
        async saveDoc(path) {
            if (!path || this.saving) { return }
            await this.saveMultipleDocs(path, [], '保存成功');
        },
        async saveMultipleDocs(triggerPath = '', deletePaths = [], successText = '操作成功') {
            if (this.saving) { return }
            this.persistEditorContent();
            this.saving = true;
            this.savingPath = triggerPath;
            try {
                let currentDoc = this.docsFiles.find(item => item.path === triggerPath);
                let uniqueDeletePaths = deletePaths.filter((path, index, list) => path && list.indexOf(path) === index);

                if (currentDoc) {
                    await myAxios.post('/respo/share-file/file', {
                        identifie: this.formulaIdentifie,
                        filename: currentDoc.path,
                        content: currentDoc.content || '',
                    });
                }

                for (let i = 0; i < uniqueDeletePaths.length; i++) {
                    await myAxios.post('/respo/share-file/file', {
                        identifie: this.formulaIdentifie,
                        filename: uniqueDeletePaths[i],
                        content: '',
                    });
                }

                if (uniqueDeletePaths.length) {
                    await this.postDocsOrderFile(this.getDocsOrderContent());
                }

                messageSuccess(successText);
                await this.getFiles();
                this.isEditing = true;
            } finally {
                this.saving = false;
                this.savingPath = '';
            }
        },
    }
}
</script>

<style scoped>
.description-page {
    min-height: 100vh;
    background: #fff;
}

.description-spin {
    display: block;
}

.page-header {
    padding: 20px;
    border-bottom: 1px solid #e7e7e7;
}

.page-body {
    padding: 20px;
}

.env-toolbar {
    padding: 0 0 16px;
}

.env-label {
    margin-right: 12px;
    font-size: 13px;
    color: #606266;
}

.env-path {
    font-size: 13px;
    color: #909399;
}

.toolbar {
    padding: 0 0 16px;
}

.toolbar-title {
    font-size: 18px;
    color: #303133;
    line-height: 1;
}

.toolbar-desc {
    margin-top: 8px;
    font-size: 13px;
    color: #909399;
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
    transition: all .2s;
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
    opacity: .45;
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

.sidebar-item-main[draggable="false"] {
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
    transition: opacity .2s;
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
    transition: color .2s;
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
    opacity: .35 !important;
    cursor: not-allowed;
}

.sidebar-add .arco-btn {
    width: 100%;
}

.sidebar-item-path {
    margin-top: 6px;
    font-size: 12px;
    color: #909399;
    word-break: break-all;
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

.editor-path {
    margin-top: 6px;
    font-size: 12px;
    color: #909399;
    word-break: break-all;
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

.mavon-description-editor :deep([align="left"]) {
    text-align: left;
}

.mavon-description-editor :deep([align="center"]) {
    text-align: center;
}

.mavon-description-editor :deep([align="right"]) {
    text-align: right;
}

.mavon-description-editor :deep([align="justify"]) {
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

.form-value {
    flex: 1;
    padding: 10px 12px;
    background: #f5f7fa;
    border-radius: 4px;
    word-break: break-all;
    color: #303133;
}

.form-tip {
    font-size: 12px;
    color: #909399;
    line-height: 1.6;
}
</style>
