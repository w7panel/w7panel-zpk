<template>
    <div id="zpkfiltertree" class="bg-white" style="height:100%;">
        <div style="padding:20px; border-bottom:1px solid #E7E7E7;">
            <el-breadcrumb separator="/">
                <el-breadcrumb-item :to="{ path: '/zpk' }"><template #default><span
                            class="c-99 fw-400">我的制品库</span></template></el-breadcrumb-item>
                <el-breadcrumb-item :to="{ path: '/zpk-version', query: { id: this.identifie, title: vtitle } }"><template
                        #default><span class="c-99 fw-400">版本管理</span></template></el-breadcrumb-item>
                <el-breadcrumb-item :to="{ path: '/zpk-edit', query: { id: identifie, versionid: versionid } }"><template
                        #default><span class="c-99 fw-400">应用基础信息修改</span></template></el-breadcrumb-item>
                <el-breadcrumb-item><template #default><span
                            class="c-33 fw-400">文件编辑</span></template></el-breadcrumb-item>
            </el-breadcrumb>
        </div>
        <div class="df box">
            <div class="treebox df-s0">
                <div class="df jc-s ml-10" style="margin-bottom:10px;">
                    <el-button @click.stop="openCreatefile" size="small">+ 新建文件</el-button>
                </div>
                <el-tree ref="eltree" :data="tree" @node-click="nodeClick" :default-expanded-keys="treeExpanded"
                    :props="treeProps" node-key="id" class="tree">
                    <template #default="{ node, data }">
                        <div class="custom-tree-node df ai-c jc-b f1 fs-14">
                            <span>{{ node.label }}</span>
                            <span class="operation ml-10 c-blue" style="margin-right:6px;">
                                <span v-if="!data.children" @click.stop="deleteFile(node, data)" class="cursor">删除</span>
                                <span v-else @click.stop="createFile(node, data)" class="c-blue cursor">新建文件</span>
                            </span>
                        </div>
                    </template>
                </el-tree>
            </div>
            <div class="fc right df df-c" style="padding:0;">
                <div id="editor_file"></div>
            </div>
        </div>
        <div class="mt-16 df jc-c">
            <el-button type="primary" style="width:100px;" @click="save">保存</el-button>
        </div>
    </div>

    <el-dialog v-model="addfile.show" title="添加文件" width="600px" modal-class="zpk-version-dialog">
        <div class="mt-20 df ai-c ml-20">
            <div style="width:70px;">文件名</div>
            <el-input v-model="addfile.filename" placeholder="请输入文件名" :spellcheck="false"
                style="width:400px;"></el-input>
        </div>
        <template #footer>
            <el-button @click="addfile.show = false;">取消</el-button>
            <el-button type="primary" @click="newFileName(addfile.filename)">确认添加</el-button>
        </template>
    </el-dialog>
</template>

<script>
import myAxios from '@/utils'
import { basicSetup } from "codemirror"
import { indentWithTab } from "@codemirror/commands"
import { EditorView, keymap } from "@codemirror/view"
const customNodeClass = (data, node) => data.isActive ? 'active' : null;
export default {
    data() {
        return {
            vtitle: '',
            versionid: '',
            identifie: '',
            tree: [],
            treeProps: {
                label: 'label',
                children: 'children',
                isLeaf: 'isLeaf',
                class: customNodeClass
            },
            treeExpanded: [],
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
            this.addfile.filename = this.$refs.eltree.getCurrentKey();
        },
        newFileName(path) {
            path = path.replace(/^\/+/, '');
            this.addfile.show = false;
            this.eachTree(path, '');


            for (let i in this.tree) {
                this.clearActive(this.tree[i]);
            }
            this.activePath = path;

            this.$nextTick(() => {
                this.treeExpanded = [path];
                let node = this.$refs.eltree.getNode(path);
                for (let i in this.tree) {
                    this.clearActive(this.tree[i]);
                }
                node.data.isActive = true;
                if (node?.data?.hasOwnProperty('content')) {
                    this.inputContent(node.data.content);
                    return;
                }
            });
        },
        createFile(data) {
            let path = data.data.id;
            this.$msgbox.prompt('文件名', '新建文件').then((data) => {
                let value = data.value.replace(/(^\/+)|(\/+$)/, '')
                if (!value.value) { return; }
                this.newFileName(path + value);
            }).catch(() => { })
        },
        deleteFile(node, data) {
            this.$confirm('确定要删除"' + data.label + '"吗', "提示", {
                confirmButtonText: "确定",
                cancelButtonText: "取消",
            }).then(() => {
                let nd = node;
                let path = data.label;
                while (nd?.parent?.data?.label) {
                    path = nd.parent.data.label + '/' + path;
                    nd = nd.parent;
                }

                myAxios.post('/respo/file', {
                    identifie: this.identifie,
                    filename: path,
                    content: '',
                    version: this.versionid,
                }).then(() => {
                    this.getInfo(this.identifie, () => {
                        this.$message.success('操作成功');
                        this.activePath = '';
                        this.tree = [];
                        this.inputContent();
                        this.getZipFileList();
                    });
                });
            });
        },
        save() {
            let txt = this.editor.state.doc.toString();
            if (!this.activePath) { return; }
            if (!txt) { this.$message.warning('文件内容不能为空'); return; }
            myAxios.post('/respo/file', {
                identifie: this.identifie,
                filename: this.activePath,
                content: txt,
                version: this.versionid,
            }).then(() => {
                let node = this.$refs.eltree.getNode(this.activePath);
                node.data.content = txt;
                this.getInfo(this.identifie, () => {
                    this.$message.success('操作成功');
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

        inputContent(file) {
            if (!this.editor) { return }
            let txt = this.editor.state.doc.toString();
            this.editor.dispatch({ changes: { from: 0, to: txt.length, insert: file } });
        },

        nodeClick(clicknode, info) {
            if (clicknode.children) { return }
            let path = clicknode.label;
            for (let i in this.tree) {
                this.clearActive(this.tree[i]);
            }
            clicknode.isActive = true;
            let node = info;
            while (node?.parent?.data?.label) {
                path = node.parent.data.label + '/' + path;
                node = node.parent;
            }
            this.activePath = path;
            if (clicknode.hasOwnProperty('content')) {
                this.inputContent(clicknode.content);
                return;
            }
            this.getContent(path);
        },
        clearActive(obj) {
            if (obj.isActive) { obj.isActive = false; }
            if (obj.children?.length) {
                for (let i in obj.children) {
                    this.clearActive(obj.children[i]);
                }
            }
        },

        getContent(path) {
            myAxios.post('/respo/get-zip-file-content', {
                identifie: this.identifie,
                version: this.versionid,
                path: path,
            }).then(res => {
                let cont = res?.data?.data?.content || '';
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

.right {
    box-sizing: border-box;
    max-height: 700px;
    padding: 10px;
    overflow: hidden;
}
</style>
<style>
.treebox .active>.el-tree-node__content {
    background: #3296fa !important;
    color: #ffffff !important;
}

.treebox .el-tree-node__children {
    overflow: visible !important;
}

.treebox .custom-tree-node .operation {
    display: none;
}

.treebox .custom-tree-node:hover .operation {
    display: inline;
}

.treebox .active .custom-tree-node:hover .operation {
    color: #ffffff;
}
</style>
