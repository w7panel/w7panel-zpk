<template>
    <mavon-editor
        v-if="!readonly"
        ref="editor"
        v-model="content"
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
    <div v-else class="markdown-body document-readonly" v-html="markdownToHtml(modelValue)"></div>
</template>

<script>
import { renderMarkdown } from '../utils/markdown';

const MAVON_ALIGN_TYPES = {
    alignleft: 'left',
    aligncenter: 'center',
    alignright: 'right',
};
const patchedMavonEditors = new WeakSet();
export default {
    name: 'DocumentMarkdownEditor',
    props: {
        modelValue: { type: String, default: '' },
        readonly: { type: Boolean, default: false },
        disabled: { type: Boolean, default: false },
    },
    emits: ['update:modelValue'],
    data() {
        return {
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
        content: {
            get() {
                return this.modelValue;
            },
            set(value) {
                this.$emit('update:modelValue', value);
            },
        },
    },
    watch: {
        modelValue() {
            this.$nextTick(this.patchMavonMarkdown);
        },
        readonly() {
            this.$nextTick(this.patchMavonMarkdown);
        },
    },
    mounted() {
        this.$nextTick(this.patchMavonMarkdown);
    },
    methods: {
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
            let editor = this.$refs.editor;
            let textarea = editor?.getTextareaDom?.();
            if (!align || !editor || !textarea || this.disabled) {
                return;
            }

            editor.insertText(textarea, {
                prefix: '<div align="' + align + '">',
                subfix: '</div>',
                str: '',
                type,
            });
            this.content = editor.d_value || textarea.value || this.content;
            this.$nextTick(() => {
                this.patchMavonMarkdown();
            });
        },
        patchMavonToolbar(editor = this.$refs.editor) {
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
            let editor = this.$refs.editor;
            if (!editor) {
                return;
            }
            this.patchMavonToolbar(editor);
            let markdownIt = editor.markdownIt;
            if (markdownIt && !markdownIt.__githubPreviewPatched) {
                markdownIt.render = (src = '') => this.markdownToHtml(src);
                markdownIt.__githubPreviewPatched = true;
            }
            editor.d_render = this.markdownToHtml(editor.d_value ?? this.content);
        },
    },
};
</script>

<style scoped>
.document-readonly {
    padding: 24px;
    overflow: auto;
    height: 100%;
    box-sizing: border-box;
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

</style>
