<template>
    <div>
        <div class="top zpk-page-header">
            <span class="backbtn df ai-c" @click="$router.go(-1)">
                <icon-arrow-left class="backicon" />
            </span>
            <div class="zpk-page-header-actions" v-if="manifest && !noPlatform">
                <a-button type="outline" @click="openYamlPreview">预览 YAML</a-button>
            </div>
        </div>
        <a-spin :loading="loading" class="create-spin">
            <div v-if="noPlatform">
                <a-empty :image-size="200" description="" class="manifest-empty">
                    <span class="c-99">暂无数据，点击</span>
                    <span class="cursor c-blue" @click="setPlatform">创建后端包配置</span>
                </a-empty>
            </div>
            <files-manifest v-if="manifest && !noPlatform" :data="manifest" ref="form" :version_id="version_id"
                :identifie="identifie" :option="{ create: true }" @addfile="addfileInside" @complete="complete">
            </files-manifest>
        </a-spin>
    </div>
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
import { messageError, messageSuccess } from '@/utils/ui-feedback';
import { IconArrowLeft } from '@arco-design/web-vue/es/icon';

export default {
    components: { filesManifest, IconArrowLeft },
    data() {
        return {
            version_id: '',
            identifie: '',
            manifest: '',
            loading: false,
            tree: [],
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
        openYamlPreview() {
            this.$refs.form?.openYamlPreview?.();
        },
        addfileInside(json, yaml, data) {

            myAxios.post('/respo/manifest/file', {
                identifie: this.identifie,
                filename: 'manifest.yaml',
                content: yaml,
                version: this.version_id,
            }).then((res) => {
                if (this.tree.find(i => i.label == path)) { return }
                myAxios.post('/respo/manifest/file', {
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
            myAxios.post('/respo/manifest/path-tree', {
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
        complete(json, yaml, otherData) {
            myAxios.post('/respo/manifest/file', {
                identifie: this.identifie,
                filename: 'manifest.yaml',
                content: yaml,
                version: this.version_id,
            }).then((res) => {
                if (otherData?.editfile) {
                    if (/\.yaml$/.test(otherData.editfile)) {
                        this.$router.push('/zpk-manifest-editor?version_id=' + this.version_id + '&identifie=' + this.identifie + '&filename=' + otherData.editfile);
                        return;
                    }
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
