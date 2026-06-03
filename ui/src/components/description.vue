<template>
    <div class="description">
        <div class="tabs">
            <span class="active">应用介绍</span>
        </div>
        <a-empty v-if="!docEntries.length" :image-size="140" description="暂无应用介绍" />

        <div v-else class="description__layout" :class="{ 'is-single': !showSidebar }">
            <aside v-if="showSidebar" class="description__sidebar">
                <button v-for="item in docEntries" :key="item.path" type="button" class="description__nav-item"
                    :class="{ 'is-active': item.path === activePath }" @click="selectEntry(item.path)">
                    <span class="description__nav-title">{{ item.title === 'README' ? '简介' : item.title }}</span>
                </button>
            </aside>

            <section class="description__main">


                <div class="description__viewer" v-html="renderedActiveContent">
                </div>
            </section>
        </div>
    </div>
</template>

<script>
import MarkdownIt from 'markdown-it';
import markdownItTaskLists from 'markdown-it-task-lists';

const DOCS_ORDER_PATH = 'docs/.order';
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

const normalizePath = (path = '') => String(path || '')
    .replace(/\\/g, '/')
    .replace(/^\/+/, '')
    .replace(/\/+/g, '/');

const isReadmePath = (path = '') => /^readme\.md$/i.test(normalizePath(path));
const isDocMarkdownPath = (path = '') => /^docs\/[^/]+\.md$/i.test(normalizePath(path));
const parseDocsOrder = (content = '') => String(content || '')
    .split(',')
    .map(path => normalizePath(path))
    .filter(path => isDocMarkdownPath(path));

const getTitleFromPath = (path = '') => {
    const normalized = normalizePath(path);
    if (isReadmePath(normalized)) {
        return 'README';
    }

    const fileName = normalized.split('/').pop() || normalized;
    return fileName.replace(/\.md$/i, '') || normalized;
};

export default {
    name: 'SiteDescription',
    props: {
        files: {
            type: Object,
            default: () => ({}),
        },
    },
    data() {
        return {
            activePath: ''
        };
    },
    computed: {
        docEntries() {
            const source = this.files && typeof this.files === 'object' && !Array.isArray(this.files)
                ? this.files
                : {};
            const fileMap = {};
            Object.keys(source).forEach((path) => {
                const normalizedPath = normalizePath(path);
                if (!isReadmePath(normalizedPath) && !isDocMarkdownPath(normalizedPath)) { return }
                fileMap[normalizedPath] = source[path] == null ? '' : String(source[path]);
            });

            const orderPath = Object.keys(source).find(path => normalizePath(path) === DOCS_ORDER_PATH);
            const readmePath = Object.keys(fileMap).find(path => isReadmePath(path));
            const orderedDocs = parseDocsOrder(orderPath ? source[orderPath] : '')
                .filter(path => Object.prototype.hasOwnProperty.call(fileMap, path));
            const orderedDocSet = new Set(orderedDocs);
            const unorderedDocs = Object.keys(fileMap)
                .filter(path => isDocMarkdownPath(path) && !orderedDocSet.has(path))
                .sort((a, b) => a.localeCompare(b, 'zh-Hans-CN'));
            const paths = [
                ...(readmePath ? [readmePath] : []),
                ...orderedDocs,
                ...unorderedDocs,
            ];

            return paths.map((path) => {
                return {
                    path,
                    title: getTitleFromPath(path),
                    content: fileMap[path] || '',
                    isReadme: isReadmePath(path),
                };
            });
        },
        showSidebar() {
            return this.docEntries.length > 1;
        },
        activeEntry() {
            return this.docEntries.find(item => item.path === this.activePath) || this.docEntries[0] || {
                path: '',
                title: '',
                content: '',
            };
        },
        renderedActiveContent() {
            return renderMarkdownWithGithubAlerts(this.activeEntry.content);
        },
    },
    watch: {
        files: {
            immediate: true,
            deep: true,
            handler() {
                this.syncActivePath();
            },
        },
    },
    methods: {
        syncActivePath() {
            if (!this.docEntries.length) {
                this.activePath = '';
                return;
            }

            const exists = this.docEntries.some(item => item.path === this.activePath);
            if (!exists) {
                this.activePath = this.docEntries[0].path;
            }
        },
        selectEntry(path) {
            this.activePath = path;
        },
    },
};
</script>

<style scoped lang="scss">
.description {
    background-color: #fff;
    padding: 20px;
}

.description__layout {
    display: flex;
    gap: 20px;
    align-items: flex-start;
}

.description__layout.is-single {
    display: block;
}

.description__sidebar {
    width: 240px;
    flex: 0 0 240px;
    background: #fff;
    border-radius: 12px;
    padding: 20px 0;
    position: sticky;
    top: 20px;
}

.description__sidebar-title {
    padding: 0 20px 12px;
    font-size: 16px;
    font-weight: 600;
    line-height: 24px;
    color: rgba(0, 0, 0, 0.9);
}

.description__nav-item {
    width: 100%;
    border: 0;
    background: transparent;
    text-align: left;
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 12px 20px;
    cursor: pointer;
    transition: background-color 0.2s ease, color 0.2s ease;
}

.description__nav-item:hover {
    background: #f5f8ff;
}

.description__nav-item.is-active {
    background: #edf3ff;
}

.description__nav-title {
    font-size: 14px;
    line-height: 22px;
    color: rgba(0, 0, 0, 0.88);
    word-break: break-word;
}

.description__nav-item.is-active .description__nav-title {
    color: #2d5fff;
}

.description__nav-path {
    font-size: 12px;
    line-height: 18px;
    color: rgba(0, 0, 0, 0.45);
    word-break: break-all;
}

.description__main {
    min-width: 0;
    flex: 1;
    background: #fff;
    border-radius: 12px;
    overflow: hidden;
}

.description__header {
    padding: 24px 28px 0;
}

.description__title {
    font-size: 24px;
    font-weight: 600;
    line-height: 32px;
    color: rgba(0, 0, 0, 0.9);
    word-break: break-word;
}

.description__path {
    margin-top: 8px;
    font-size: 13px;
    line-height: 20px;
    color: rgba(0, 0, 0, 0.45);
    word-break: break-all;
}

.description__viewer {
    min-height: 240px;
    color: rgba(0, 0, 0, 0.88);
    line-height: 1.75;
    word-break: break-word;
}

.description__viewer :deep(h1),
.description__viewer :deep(h2),
.description__viewer :deep(h3) {
    margin: 20px 0 12px;
    color: rgba(0, 0, 0, 0.9);
    line-height: 1.35;
}

.description__viewer :deep(h1) {
    font-size: 26px;
}

.description__viewer :deep(h2) {
    font-size: 22px;
}

.description__viewer :deep(h3) {
    font-size: 18px;
}

.description__viewer :deep([align="left"]) {
    text-align: left;
}

.description__viewer :deep([align="center"]) {
    text-align: center;
}

.description__viewer :deep([align="right"]) {
    text-align: right;
}

.description__viewer :deep([align="justify"]) {
    text-align: justify;
}

.description__viewer :deep(p) {
    margin: 0 0 12px;
}

.description__viewer :deep(a) {
    color: #2d5fff;
    text-decoration: none;
}

.description__viewer :deep(a:hover) {
    color: #1f49d8;
    text-decoration: underline;
}

.description__viewer :deep(p:empty) {
    display: none;
}

.description__viewer :deep(hr) {
    height: 0;
    margin: 15px 0;
    overflow: hidden;
    background: transparent;
    border: 0;
    border-bottom: 1px solid #dfe2e5;
}

.description__viewer :deep(ul),
.description__viewer :deep(ol) {
    margin: 0 0 12px 22px;
    padding: 0;
}

.description__viewer :deep(blockquote) {
    margin: 0 0 12px;
    padding: 0 1em;
    border-left: .25em solid #dfe2e5;
    color: #6a737d;
    background: transparent;
}

.description__viewer :deep(blockquote > :first-child) {
    margin-top: 0;
}

.description__viewer :deep(blockquote > :last-child) {
    margin-bottom: 0;
}

.description__viewer :deep(.markdown-alert) {
    padding: 8px 16px;
    border-left-width: 4px;
    color: rgba(0, 0, 0, 0.88);
    background: #fff;
}

.description__viewer :deep(.markdown-alert-title) {
    margin: 0 0 8px;
    font-weight: 600;
}

.description__viewer :deep(.markdown-alert-note) {
    border-left-color: #0969da;
}

.description__viewer :deep(.markdown-alert-note .markdown-alert-title) {
    color: #0969da;
}

.description__viewer :deep(.markdown-alert-tip) {
    border-left-color: #1a7f37;
}

.description__viewer :deep(.markdown-alert-tip .markdown-alert-title) {
    color: #1a7f37;
}

.description__viewer :deep(.markdown-alert-important) {
    border-left-color: #8250df;
}

.description__viewer :deep(.markdown-alert-important .markdown-alert-title) {
    color: #8250df;
}

.description__viewer :deep(.markdown-alert-warning) {
    border-left-color: #9a6700;
}

.description__viewer :deep(.markdown-alert-warning .markdown-alert-title) {
    color: #9a6700;
}

.description__viewer :deep(.markdown-alert-caution) {
    border-left-color: #cf222e;
}

.description__viewer :deep(.markdown-alert-caution .markdown-alert-title) {
    color: #cf222e;
}

.description__viewer :deep(pre) {
    margin: 0 0 12px;
    padding: 16px;
    overflow: auto;
    border-radius: 3px;
    color: #24292e;
    font-size: 85%;
    line-height: 1.45;
    word-wrap: normal;
    background-color: #f6f8fa;
}

.description__viewer :deep(code) {
    padding: .2em 0;
    margin: 0;
    border-radius: 3px;
    font-size: 85%;
    background-color: rgba(27, 31, 35, .05);
}

.description__viewer :deep(code::before),
.description__viewer :deep(code::after) {
    letter-spacing: -.2em;
    content: "\00a0";
}

.description__viewer :deep(pre code) {
    padding: 0;
    margin: 0;
    border: 0;
    color: inherit;
    font-size: 100%;
    line-height: inherit;
    word-break: normal;
    white-space: pre;
    background: transparent;
}

.description__viewer :deep(pre code::before),
.description__viewer :deep(pre code::after) {
    content: normal;
}

.description__viewer :deep(img) {
    max-width: 100%;
    box-sizing: content-box;
    background-color: #fff;
}

.description__viewer :deep(table) {
    width: 100%;
    margin: 0 0 16px;
    overflow: auto;
    border-spacing: 0;
    border-collapse: collapse;
    table-layout: auto;
}

.description__viewer :deep(table tr) {
    background-color: #fff;
    border-top: 1px solid #c6cbd1;
}

.description__viewer :deep(table tr:nth-child(2n)) {
    background-color: #f6f8fa;
}

.description__viewer :deep(th),
.description__viewer :deep(td) {
    padding: 6px 13px;
    border: 1px solid #dfe2e5;
    vertical-align: top;
}

.description__viewer :deep(th) {
    font-weight: 600;
}

.description__viewer :deep(input[type="checkbox"]) {
    margin-right: 8px;
}

.description__viewer :deep(.contains-task-list) {
    margin-left: 0;
    list-style: none;
}

.description__viewer :deep(.task-list-item) {
    list-style: none;
}


.tabs {
    display: flex;
    margin-bottom: 20px;

    span {
        cursor: pointer;
        line-height: 18px;
        font-size: 14px;
        text-align: center;
        padding: 10px 0;
        margin: 0 12px;
        position: relative;

        &:first-child {
            margin-left: 0;
        }

        &.active {
            border-bottom: 2px solid #2d5fff;
        }

        font {
            color: #2d5fff;
            font-weight: bold;
            margin-left: 5px;
        }
    }
}
</style>
