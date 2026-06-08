<template>
    <div>
        <div class="top zpk-page-header">
            <span class="backbtn df ai-c" @click="$router.go(-1)">
                <icon-arrow-left class="backicon" />
            </span>
        </div>
        <a-spin :loading="loading" class="create-spin">
            <div class="zpk-toolbar-left" style="padding:0 20px;">
                <a-button type="primary" @click="impt.show = true; impt.data = null;">
                    导入制品库
                </a-button>
            </div>
            <div v-if="noPlatform">
                <a-empty :image-size="200" description="" class="manifest-empty">
                    <span class="c-99">暂无数据，点击</span>
                    <span class="cursor c-blue" @click="setPlatform">创建后端包配置</span>
                </a-empty>
            </div>
            <files-manifest v-if="manifest && !noPlatform" :data="manifest" ref="form" :version_id="version_id"
                :identifie="identifie" :option="{ create: true }" @addfile="addfileInside" @complete="complete">
            </files-manifest>
            <files-upload v-if="uploadzipfile" :file="uploadzipfile" @error="loading = false;"
                @success="uploadSuccess"></files-upload>
        </a-spin>
    </div>

    <a-modal v-model:visible="addfile.show" title="添加文件" :width="600" :footer="false"
        modal-class="zpk-version-dialog">
        <div class="mt-20 df ai-c ml-20">
            <div style="width:70px;">文件名</div>
            <a-input v-model="addfile.filename" placeholder="请输入文件名" style="width:400px;"></a-input>
        </div>
        <div class="dialog-footer file-name-dialog-footer">
            <a-button @click="addfile.show = false;">取消</a-button>
            <a-button type="primary"
                @click="$router.push('/zpk-fileadd?identifie=' + identifie + '&filename=' + addfile.filename)">确认添加</a-button>
        </div>
    </a-modal>

    <a-modal v-model:visible="impt.show" title="导入制品库" :width="560" :footer="false">
        <div class="zpk-modal-content import-zpk-content">
            <a-auto-complete v-model="impt.title" ref="imptzpk" :data="impt.options" placeholder="请选择制品库"
                style="width:400px;" size="large" allow-clear :filter-option="false" @search="addQuerySearch"
                @change="handleImportTitleChange" @select="addSelect" :spellcheck="false" />
        </div>
        <div class="dialog-footer">
            <a-button @click="impt.show = false">取消</a-button>
            <a-button type="primary" @click="imptSubmit()" size="large">确定</a-button>
        </div>
    </a-modal>
</template>

<script>
const defaultManifest = `application:
    name: ''
    identifie: ''
    description: ''
    author: ''
platform:
    container:
        containerPort: 80
        minNum: 1
        maxNum: 10
        cpu: 1
        mem: 2
        policyType: cpu
        policyThreshold: 80
        customLogs: stdout
        initialDelaySeconds: 2
    supports:
        - notapp
`;

import jsyaml from "js-yaml";
import filesManifest from '@/components/files-manifest.vue';
import myAxios from '@/utils';
import axios from 'axios';
import JSZip from 'jszip';
import JSZipUtils from "jszip-utils";
import filesUpload from '@/components/files-upload.vue';
import { messageError, messageSuccess, messageWarning } from '@/utils/ui-feedback';
import { IconArrowLeft } from '@arco-design/web-vue/es/icon';

export default {
    components: { filesManifest, filesUpload, IconArrowLeft },
    data() {
        return {
            version_id: '',
            identifie: '',
            manifest: '',
            iptmanifest: '',
            link: '',
            loading: false,
            uploadzipfile: null,
            tree: [],
            addfile: {
                show: false,
                filename: '',
            },
            impt: {
                show: false,
                title: '',
                data: null,
                options: [],
            },
            noPlatform: false,
            noManifest: false,
        }
    },
    async created() {
        this.identifie = this.$route.query.id;
        await myAxios.post('/respo/version-list', { identifie: this.identifie }).then(res => {
            this.version_id = res?.data?.data?.list[0]?.name || '';
        })
        await this.next();
        if (this.$route.query.link) {
            this.link = decodeURIComponent(this.$route.query.link);
            this.iptzpk();
        }
        this.getFile();
    },
    methods: {
        setPlatform() {
            if (this.noManifest) {
                this.noManifest = false;
                this.noPlatform = false;
            } else {
                this.json.platform = {
                    container: {
                        containerPort: 80,
                        minNum: 1,
                        maxNum: 10,
                        cpu: 1,
                        mem: 2,
                        policyType: 'cpu',
                        policyThreshold: 80,
                        customLogs: 'stdout',
                        initialDelaySeconds: 2
                    },
                    supports: ['notapp']
                }
                this.manifest = jsyaml.dump(this.json);
                this.noPlatform = false;
            }
        },
        imptSubmit() {
            if (!this.impt.data) { messageWarning('请选择制品库'); return; }
            this.jsonp('https://console.w7.cc/zpk?path=/respo/v2/info/' + this.impt.data.identifie + '/' + this.version_id, 'getimportmanifest', data => {
                let manifest = data?.data?.manifest || defaultManifest;
                let json = jsyaml.load(manifest);
                let ref = this.$refs.form;
                ref?.replaceZpk(json);
                this.impt.show = false;
            })
        },

        addQuerySearch(query = '') {
            this.jsonp('https://console.w7.cc/zpk?status=2&page=1&limit=999&keyword=' + encodeURIComponent(query) + '&path=/respo/list', 'getimport', (data) => {
                let list = data?.data?.list || [];
                list = list.splice(0, 20);
                this.impt.options = list.map(item => ({
                    label: item.name,
                    value: item.name,
                    raw: item,
                }));
            });
        },

        handleImportTitleChange(value) {
            if (this.impt.data?.name == value) { return; }
            let option = this.impt.options.find(item => item.value == value);
            this.impt.data = option?.raw || null;
        },

        addSelect(value) {
            let option = this.impt.options.find(item => item.value == value);
            this.impt.data = option?.raw || null;
            this.$refs.imptzpk?.blur?.();
        },

        addfileInside(json, yaml, data) {

            myAxios.post('/respo/file', {
                identifie: this.identifie,
                filename: 'manifest.yaml',
                content: yaml,
                version: this.version_id,
            }).then((res) => {
                if (this.tree.find(i => i.label == path)) { return }
                myAxios.post('/respo/file', {
                    identifie: this.identifie,
                    filename: data.file,
                    content: data.cont,
                    version: this.version_id,
                }).then(() => {
                    this.getFile();
                });
            }).catch((error) => {
                if (error?.response?.data?.error) {
                    messageError(error.response.data.error);
                }
            });
        },
        getFile() {
            myAxios.post('/respo/path-tree', {
                identifie: this.identifie,
                version: this.version_id,
            }).then(res => {
                this.list = res?.data?.data?.list || {};
                let tree = [];
                for (let i in this.list) {
                    tree.push({ label: i })
                }
                this.tree = tree;
            })
        },
        uploadSuccess(data) {
            if (data?.url || data?.data?.url) {
                let url = data?.url || data?.data?.url;

                this.json = jsyaml.load(this.iptmanifest);
                if (!this.json.application) { this.json.application = {}; }
                this.json.application.identifie = this.identifie;
                if (!this.json.source) { this.json.source = {}; }
                this.json.source.type = 'zip';
                this.json.source.url = url;
                this.manifest = jsyaml.dump(this.json);

                this.loading = false;
            }
        },
        getOnlineZip(url, callback) {
            var promise = new JSZip.external.Promise((resolve, reject) => {
                JSZipUtils.getBinaryContent(url, function (err, data) {
                    err ? reject(err) : resolve(data);
                });
            });
            promise.then(JSZip.loadAsync).then(zip => {
                callback && callback(zip);
            }).catch(() => {
                this.loading = false;
            });
        },
        iptzpk() {
            this.loading = true;
            axios.get(this.link).then(res => {
                this.iptmanifest = res.data.manifest;
                this.getOnlineZip(res.data?.data?.zip_url, (zip) => {
                    zip.generateAsync({ type: "blob" }).then((content) => {
                        let file = new File([content], this.identifie + '_' + Date.now() + '.zip', { type: "application/x-zip-compressed" });
                        this.uploadzipfile = file;
                    }).catch(() => {
                        this.loading = false;
                    });
                });
            }).catch(() => {
                this.loading = false;
                messageError('制品库链接请求失败');
            })
        },
        complete(json, yaml, otherData) {
            myAxios.post('/respo/file', {
                identifie: this.identifie,
                filename: 'manifest.yaml',
                content: yaml,
                version: this.version_id,
            }).then((res) => {
                if (otherData?.editfile) {
                    if (/\.yaml$/.test(otherData.editfile)) {
                        this.$router.push('/zpk-manifest-editor?identifie=' + this.identifie + '&filename=' + otherData.editfile);
                        return;
                    }
                    this.$router.push('/zpk-fileadd?identifie=' + this.identifie + '&filename=' + otherData.editfile);
                    return;
                }
                messageSuccess('添加manifest成功');
                this.getInfo(this.identifie, () => {
                    this.publish();
                });
            }).then(res => {

            }).catch((error) => {
                if (error?.response?.data?.error) {
                    messageError(error.response.data.error);
                }
            });
        },
        publish() {
            myAxios.post('/respo/publish', { identifie: this.identifie, version: this.version_id }).then(res => {
                if (res.data?.message || res.data?.data?.message) {
                    messageSuccess(res.data?.message || res.data?.data?.message);
                    this.$router.go(-1);
                }
            }).catch((error) => {
                if (error?.response?.data?.error) {
                    messageError(error.response.data.error);
                }
            });
        },
        getInfo(id, callback, n) {
            n = n || 0;
            myAxios.get('/respo/v2/info/' + id + '/1.0.0').then(res => {
                callback && callback();
            }).catch(() => {
                if (n > 10) { return }
                setTimeout(() => {
                    this.getInfo(id, callback, n + 1);
                }, 1000);
            });
        },
        next() {
            return myAxios.get('/respo/v2/info/' + this.identifie + '/1.0.0').then(res => {
                let m = res?.data?.data?.manifest;

                if (!m) {
                    this.noManifest = true;
                    this.noPlatform = true;
                    m = defaultManifest;
                    this.json = jsyaml.load(m);
                } else {
                    this.noManifest = false;
                    this.json = jsyaml.load(m);
                    this.noPlatform = Boolean(!this.json.platform);
                }

                if (!this.json.application) { this.json.application = {}; }
                this.json.application.identifie = this.identifie;
                this.manifest = jsyaml.dump(this.json);
            });
        },
        jsonp(url, name, callback) {
            var win = window?.rawWindow || window;
            win[name] = (data) => {
                callback(data);
                win[name] = null;
            };
            let u = new URL(url);
            u.searchParams.append('callback', name);
            let script = document.createElement("script");
            script.type = "text/javascript";
            script.setAttribute('ignore', 'true')
            script.async = true;
            script.src = u.href;
            script.onload = function () { document.body.removeChild(this); };
            script.onerror = function () { document.body.removeChild(this); };
            document.body.append(script);
        },

    },
}
</script>

<style scoped>
.create-spin {
    display: block;
    min-height: 240px;
}

.treebox {
    width: 100px;
    height: 500px;
    border: 1px solid;
    overflow: auto;
}

.table {
    width: 100%;
}

.table td {
    padding: 10px;
    line-height: 1.4;
    border: 1px solid #e7e7e7;
    border-left: 0;
    border-right: 0;
    background: #F0F3FA;
}

.table tr:last-child td {
    background: transparent;
}

.table thead tr:first-child td {
    background: #f3f3f3;
    border-top: 0;
}
</style>
