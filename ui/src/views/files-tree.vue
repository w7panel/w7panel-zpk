<template>
    <div id="zpkfiltertree" class="bg-white" style="height:100%;">
        <div class="zpk-page-header">
            <span class="backbtn df ai-c" @click="$router.go(-1)">
                <icon-arrow-left class="backicon" />
            </span>
            <a-breadcrumb>
                <a-breadcrumb-item><router-link to="/zpk" class="c-99 fw-400">制品管理</router-link></a-breadcrumb-item>
                <a-breadcrumb-item>
                    <router-link :to="{ path: '/zpk-version', query: { id: identifie, title: vtitle } }"
                        class="c-99 fw-400">{{ vtitle || identifie }}</router-link>
                </a-breadcrumb-item>
                <a-breadcrumb-item>
                    <router-link :to="{ path: '/zpk-edit', query: { id: identifie, versionid } }"
                        class="c-99 fw-400">应用基础信息修改</router-link>
                </a-breadcrumb-item>
                <a-breadcrumb-item><span class="c-33 fw-400">文件编辑</span></a-breadcrumb-item>
            </a-breadcrumb>
        </div>
        <div class="df box">
            <div class="treebox df-s0">
                <div class="df jc-s ml-10" style="margin-bottom:10px;">
                    <a-button type="primary" class="create-file-btn" @click.stop="openCreatefile" size="small">
                        <template #icon><icon-plus /></template>
                        新建文件
                    </a-button>
                </div>
                <a-empty v-if="!tree.length" description="暂无数据" class="tree-empty" />
                <a-tree v-else v-model:selected-keys="selectedKeys" v-model:expanded-keys="treeExpanded" :data="tree"
                    :field-names="treeFieldNames" block-node class="tree" @select="nodeClick">
                    <template #title="data">
                        <span>{{ data.label }}</span>
                    </template>
                    <template #extra="data">
                        <span class="operation ml-10 c-blue" style="margin-right:6px;">
                            <span v-if="!data.children" @click.stop>
                                <a-popconfirm content="确认要删除吗" type="warning" ok-text="确定"
                                    cancel-text="取消" content-class="zpk-delete-popconfirm"
                                    :ok-button-props="{ status: 'danger' }"
                                    :cancel-button-props="{ type: 'secondary' }" @ok="deleteFile(data)">
                                    <span class="cursor">删除</span>
                                </a-popconfirm>
                            </span>
                            <span v-else @click.stop="createFile(data)" class="c-blue cursor">新建文件</span>
                        </span>
                    </template>
                </a-tree>
            </div>
            <div class="fc right df df-c" style="padding:0;">
                <div id="editor_file"></div>
            </div>
        </div>
        <div class="mt-16 df jc-c">
            <a-button type="primary" style="width:100px;" @click="save">保存</a-button>
        </div>
    </div>

    <a-modal v-model:visible="addfile.show" title="添加文件" :width="600" :footer="false"
        modal-class="zpk-version-dialog">
        <div class="mt-20 df ai-c ml-20">
            <div style="width:70px;">文件名</div>
            <a-input v-model="addfile.filename" placeholder="请输入文件名" :spellcheck="false" style="width:400px;" />
        </div>
        <div class="dialog-footer file-name-dialog-footer">
            <a-button @click="addfile.show = false;">取消</a-button>
            <a-button type="primary" @click="newFileName(addfile.filename)">确认添加</a-button>
        </div>
    </a-modal>
</template>

<script>
import myAxios from '@/utils'
import { basicSetup } from "codemirror"
import { indentWithTab } from "@codemirror/commands"
import { EditorView, keymap } from "@codemirror/view"
import { IconArrowLeft, IconPlus } from '@arco-design/web-vue/es/icon';
import { messageSuccess, messageWarning } from '@/utils/ui-feedback';
export default {
    components: {
        IconArrowLeft,
        IconPlus,
    },
    data() {
        return {
            vtitle: '',
            versionid: '',
            identifie: '',
            tree: [],
            treeFieldNames: {
                key: 'id',
                title: 'label',
                children: 'children',
            },
            treeExpanded: [],
            selectedKeys: [],
            addfile: {
                show: false,
                filename: '',
            },
            editor: null,
            activePath: '',
        }
    },
    created() {
        this.vtitle = this.$route.query.vtitle || '';
        this.identifie = this.$route.query.id;
        this.versionid = this.$route.query.versionid;
    },
    mounted() {
        this.init();
        this.getZipFileList();
    },
    methods: {
        openCreatefile() {
            this.addfile.show = true;
            this.addfile.filename = this.activePath || '';
        },
        newFileName(path) {
            path = (path || '').replace(/^\/+/, '');
            if (!path) { return }
            this.addfile.show = false;
            this.eachTree(path, '');

            this.activePath = path;
            this.selectedKeys = [path];
            this.treeExpanded = this.getParentKeys(path);

            this.$nextTick(() => {
                let node = this.findTreeNode(path);
                if (node?.hasOwnProperty('content')) {
                    this.inputContent(node.content);
                    return;
                }
            });
        },
        createFile(data) {
            this.addfile.show = true;
            this.addfile.filename = data.id || '';
        },
        deleteFile(data) {
            return myAxios.post('/respo/file', {
                identifie: this.identifie,
                filename: data.id,
                content: '',
                version: this.versionid,
            }).then(() => {
                this.getInfo(this.identifie, () => {
                    messageSuccess('操作成功');
                    this.activePath = '';
                    this.selectedKeys = [];
                    this.tree = [];
                    this.inputContent();
                    this.getZipFileList();
                });
            });
        },
        save() {
            let txt = this.editor.state.doc.toString();
            if (!this.activePath) { return; }
            if (!txt) { messageWarning('文件内容不能为空'); return; }
            myAxios.post('/respo/file', {
                identifie: this.identifie,
                filename: this.activePath,
                content: txt,
                version: this.versionid,
            }).then(() => {
                let node = this.findTreeNode(this.activePath);
                if (node) {
                    node.content = txt;
                }
                this.getInfo(this.identifie, () => {
                    messageSuccess('操作成功');
                });
            });
        },
        getInfo(id, callback, n) {
            n = n || 0;
            myAxios.get('/respo/v2/info/' + id + '/' + this.versionid).then(res => {
                callback && callback();
            }).catch(() => {
                if (n > 10) { return }
                setTimeout(() => {
                    this.getInfo(id, callback, n + 1);
                }, 1000);
            });
        },
        getZipFileList() {
            myAxios.post('/respo/get-zip-file-list', { identifie: this.identifie, version: this.versionid }).then(res => {
                let path = res?.data?.data?.list || [];
                for (let i in path) {
                    this.eachTree(path[i]);
                }
            });
            myAxios.post('/respo/path-tree', { identifie: this.identifie, version: this.versionid }).then(res => {
                let path = res?.data?.data?.list || {};
                for (let i in path) {
                    let cont = path[i] || '';
                    this.eachTree(i, cont);
                }
            });
        },

        inputContent(file = '') {
            if (!this.editor) { return }
            let txt = this.editor.state.doc.toString();
            this.editor.dispatch({ changes: { from: 0, to: txt.length, insert: file } });
        },

        nodeClick(selectedKeys, event) {
            let clicknode = event?.node;
            if (!clicknode) { return }
            if (clicknode.children) {
                this.selectedKeys = this.activePath ? [this.activePath] : [];
                return
            }
            let path = clicknode.id;
            this.activePath = path;
            this.selectedKeys = [path];
            if (clicknode.hasOwnProperty('content')) {
                this.inputContent(clicknode.content);
                return;
            }
            this.getContent(path);
        },
        findTreeNode(key, nodes = this.tree) {
            for (let item of nodes) {
                if (item.id === key) {
                    return item;
                }
                if (item.children?.length) {
                    let found = this.findTreeNode(key, item.children);
                    if (found) { return found }
                }
            }
            return null;
        },
        getParentKeys(path) {
            let parts = path.replace(/^\/+|\/+$/g, '').split('/').filter(Boolean);
            let keys = [];
            for (let i = 0; i < parts.length - 1; i++) {
                keys.push(parts.slice(0, i + 1).join('/') + '/');
            }
            return keys;
        },

        getContent(path) {
            myAxios.post('/respo/get-zip-file-content', {
                identifie: this.identifie,
                version: this.versionid,
                path: path,
            }).then(res => {
                let cont = res?.data?.data?.content || '';
                let node = this.findTreeNode(path);
                if (node) {
                    node.content = cont;
                }
                this.inputContent(cont);
            })
        },

        eachTree(path, cont) {
            path = path.replace(/^\/+/, '');
            let dir = path.split('/');
            let arr = this.tree;
            let current = null;
            for (let d = 0; d < dir.length; d++) {
                if (d == 0) {
                    let find = arr.find(i => i.label == dir[d]);
                    if (!find) {
                        find = { label: dir[d], id: dir[d] + (dir.length == 1 ? '' : '/'), }
                        arr.push(find);
                    }
                    if (d == dir.length - 1 && cont != undefined) { find.content = cont; }
                    current = find;
                } else {
                    current.children = current.children || [];
                    if (!dir[d]) { break; }
                    let find = current.children.find(i => i.label == dir[d]);
                    if (!find) {
                        find = { label: dir[d], id: dir.slice(0, d + 1).join('/') + (d + 1 == dir.length ? '' : '/'), }
                        current.children.push(find);
                    }
                    if (d == dir.length - 1 && cont !== undefined) { find.content = cont; }
                    current = find;
                }
            }
            this.tree = JSON.parse(JSON.stringify(this.tree))
        },

        init(callback) {
            const ivory = "#abb2bf",
                stone = "#7d8799",
                darkBackground = "#21252b",
                highlightBackground = "#2c313a",
                background = "#282c34",
                tooltipBackground = "#353a42",
                selection = "#3E4451",
                cursor = "#528bff";

            let myTheme = EditorView.theme({
                "&": {
                    color: ivory,
                    backgroundColor: background,
                    height: Math.max(document.getElementById('zpkfiltertree').offsetHeight - 120, 500) + "px"
                },
                ".cm-content": { caretColor: cursor },
                ".cm-cursor, .cm-dropCursor": { borderLeftColor: cursor },
                "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection": { backgroundColor: selection },

                ".cm-panels": { backgroundColor: darkBackground, color: ivory },
                ".cm-panels.cm-panels-top": { borderBottom: "2px solid black" },
                ".cm-panels.cm-panels-bottom": { borderTop: "2px solid black" },

                ".cm-searchMatch": { backgroundColor: "#72a1ff59", outline: "1px solid #457dff" },
                ".cm-searchMatch.cm-searchMatch-selected": { backgroundColor: "#6199ff2f" },

                ".cm-activeLine": { backgroundColor: highlightBackground },
                ".cm-selectionMatch": { backgroundColor: "#aafe661a" },

                "&.cm-focused .cm-matchingBracket, &.cm-focused .cm-nonmatchingBracket": { backgroundColor: "#bad0f847", outline: "1px solid #515a6b" },

                ".cm-gutters": { backgroundColor: background, color: stone, border: "none" },

                ".cm-activeLineGutter": { backgroundColor: highlightBackground },

                ".cm-foldPlaceholder": { backgroundColor: "transparent", border: "none", color: "#ddd" },

                ".cm-tooltip": { border: "none", backgroundColor: tooltipBackground },
                ".cm-tooltip .cm-tooltip-arrow:before": { borderTopColor: "transparent", borderBottomColor: "transparent" },
                ".cm-tooltip .cm-tooltip-arrow:after": { borderTopColor: tooltipBackground, borderBottomColor: tooltipBackground },
                ".cm-tooltip-autocomplete": {
                    "& > ul > li[aria-selected]": {
                        backgroundColor: highlightBackground,
                        color: ivory
                    }
                },
                ".cm-scroller": { overflow: "auto" },
                ".cm-content, .cm-gutter": { minHeight: "500px" },
            }, { dark: true });

            this.$nextTick(() => {
                document.getElementById("editor_file").innerHTML = "";
                this.editor = new EditorView({
                    doc: "",
                    extensions: [
                        basicSetup,
                        myTheme,
                        keymap.of([indentWithTab]),
                    ],
                    parent: document.getElementById("editor_file"),
                });
                callback && callback();
            })
        },
    },
}
</script>

<style scoped>
.cont {
    padding: 0;
}

.box {
    border: 1px solid #cccccc;
}

.treebox {
    box-sizing: border-box;
    padding: 10px 10px 10px 0;
    min-width: 300px;
    max-width: 300px;
    max-height: 700px;
    overflow: auto;
    border-right: 1px solid #cccccc;
    background: #f1f1f1;
}

.tree {
    background: #f1f1f1;
}

.tree-empty {
    margin-top: 28px;
}

.tree-empty :deep(.arco-empty-image) {
    height: 56px;
}

.right {
    box-sizing: border-box;
    max-height: 700px;
    padding: 10px;
    overflow: hidden;
}
</style>
<style>
.treebox .arco-tree-node-children {
    overflow: visible !important;
}

.treebox .operation {
    display: none;
}

.treebox .arco-tree-node:hover .operation {
    display: inline;
}

.treebox .arco-tree-node {
    padding-left: 0;
}

.treebox .arco-tree-node-selected,
.treebox .arco-tree-node-selected:hover {
    background: #3296fa !important;
}

.treebox .arco-tree-node-selected .arco-tree-node-title,
.treebox .arco-tree-node-selected .arco-tree-node-title:hover,
.treebox .arco-tree-node-selected .arco-tree-node-title-block,
.treebox .arco-tree-node-selected .arco-tree-node-title-block:hover {
    background: transparent !important;
    color: #ffffff !important;
    border-radius: 0;
}

.treebox .arco-tree-node-selected .arco-tree-node-title-text,
.treebox .arco-tree-node-selected .arco-tree-node-switcher,
.treebox .arco-tree-node-selected .operation,
.treebox .arco-tree-node-selected .operation .c-blue {
    color: #ffffff !important;
}

.treebox .arco-tree-node-selected .operation .arco-btn,
.treebox .arco-tree-node-selected .operation .arco-btn:hover {
    background: transparent !important;
    border-color: transparent !important;
    color: #ffffff !important;
}
</style>
