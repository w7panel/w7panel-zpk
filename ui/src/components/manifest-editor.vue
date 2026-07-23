<template>
    <div ref="content" style="height:100vh;overflow:auto;">
        <div>
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
                        <router-link :to="{ path: '/zpk-edit', query: { id: identifie, versionid: version_id } }"
                            class="c-99 fw-400">应用基础信息修改</router-link>
                    </a-breadcrumb-item>
                    <a-breadcrumb-item><span class="c-33 fw-400">{{ title }}</span></a-breadcrumb-item>
                </a-breadcrumb>
                <div class="zpk-page-header-actions">
                    <a-button type="outline" @click="openYamlPreview">预览 YAML</a-button>
                </div>
            </div>
            <files-manifest ref="form" :data="manifest" :option="{ pureManifest: true }" @complete="complete"></files-manifest>
        </div>
    </div>
</template>

<script>
import myAxios from '@/utils';
import { messageSuccess } from '@/utils/ui-feedback';
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
`;

import filesManifest from './files-manifest.vue';
import { IconArrowLeft } from '@arco-design/web-vue/es/icon';


export default {
    data() {
        return {
            vtitle: '',
            version_id: '',
            manifest: '',
            identifie: '',
            title: '',
        }
    },
    components: { filesManifest, IconArrowLeft },
    created() {
        this.vtitle = this.$route.query.vtitle;
        this.identifie = this.$route.query.identifie;
        this.version_id = this.$route.query.version_id;
        this.manifest = defaultManifest;
        this.title = this.$route.query.filename || '';
        this.getFile();
    },
    mounted() {
        this.$refs.content.scrollTo(0, 0);
    },
    methods: {
        getFile() {
            myAxios.post('/respo/manifest/path-tree', {
                identifie: this.identifie,
                version: this.version_id,
            }).then(res => {
                this.list = res?.data?.data?.list || {};
                if (this.title) {
                    this.manifest = this.list[this.title] || defaultManifest;
                }
            });
        },
        complete(json, yaml) {
            myAxios.post('/respo/manifest/file', {
                identifie: this.identifie,
                filename: this.title,
                content: yaml,
                version: this.version_id,
            }).then(() => {
                this.getInfo(this.identifie, () => {
                    messageSuccess('操作成功');
                    setTimeout(() => {
                        this.$router.go(-1);
                    }, 500);
                })
            });
        },
        openYamlPreview() {
            this.$refs.form?.openYamlPreview?.();
        },
        getInfo(id, callback, n) {
            n = n || 0;
            myAxios.get('/respo/v2/info/' + id).then(res => {
                callback && callback();
            }).catch(() => {
                if (n > 10) { return }
                setTimeout(() => {
                    this.getInfo(id, callback, n + 1);
                }, 1000);
            });
        },
    },
}
</script>
