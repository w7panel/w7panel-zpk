<template>
    <div class="bg-f2" style="min-height:100%;">
        <div class="top bg-white df ai-c">
            <div class="df ai-c cursor" @click="$router.go(-1);">
                <el-icon :size="14" color="#666666">
                    <ArrowLeft />
                </el-icon>
                <span class="c-66 fs-16" style="margin-left:4px;">返回</span>
            </div>
        </div>
        <div class="pd-20">
            <div class="bg-white pd-20">
                <div class="df">
                    <div class="logo-box">
                        <img :src="info.icon" alt="" class="logo" />
                    </div>
                    <div class="ml-20">
                        <div class="fs-18 b">{{ info.title }}</div>
                        <div class="mt-10 fs-14 c-66">{{ info.description }}</div>
                        <div class="mt-20 df ai-c">
                            <span class="fs-14 c-66 lh-1">版本</span>
                            <el-select v-model="version" class="ml-10">
                                <el-option v-for="item in versionlist" :key="item.id" :value="item.name"
                                    :label="item.name"></el-option>
                            </el-select>
                        </div>
                        <div class="mt-20">
                            <el-button v-if="iniframe" type="primary" @click="installurl">安装</el-button>
                            <div v-else>
                                <el-button type="primary" @click="toinstall">复制地址</el-button>
                                <a :href="url" target="_blank" style="margin-left:10px;">
                                    <el-button type="primary" @click="toinstall">安装</el-button>
                                </a>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="topline mt-20">
                    <mavon-editor v-model="mdtxt" :subfield="false" :toolbarsFlag="false" defaultOpen="preview"
                        previewBackground="#ffffff" boxShadowStyle="" style="min-height:100px;" />
                </div>
            </div>
        </div>
    </div>
</template>

<script>
import axios from 'axios';

export default {
    data() {
        return {
            identifie: '',
            info: {},
            version: '',
            mdtxt: '',
            versionlist: [],
            iniframe: false,

            url: '',
        }
    },
    created() {
        this.iniframe = window.top != window;
        this.identifie = this.$route.params.id;
        this.getData();

        let host = window?.$wujie?.props?.url || window.location.origin;
        let url = host + '/respo/v2/info/' + this.identifie + '/1.0.0';
        this.url = 'https://console.w7.cc/api/deploy/thirdparty_cd/redirect?route=/zpk-install?path=' + encodeURIComponent(url);
    },
    methods: {
        installurl() {
            let host = window?.$wujie?.props?.url || window.location.origin;
            let url = host + '/respo/v2/info/' + this.identifie + '/' + this.version;
            window.parent.postMessage({ type: "zpkinstall", url: url }, "*");
        },
        getData() {
            axios.get('/respo/v2/detail/' + this.identifie + '/1.0.0', {
                baseURL: window?.$wujie?.props?.url,
            }).then(res => {
                this.info = res.data?.data || {};
                this.mdtxt = res.data?.data?.content;

                this.versionlist = res.data?.data?.version_list || [];
                if (this.versionlist.length) {
                    this.version = this.versionlist[0].name;
                }
            })
        },
        toinstall() {
            let host = window?.$wujie?.props?.url || window.location.origin;
            let url = host + '/respo/v2/info/' + this.identifie + '/' + this.version;
            url = 'https://console.w7.cc/api/deploy/thirdparty_cd/redirect?route=/zpk-install?path=' + encodeURIComponent(url);
            this.onekeyCopy(url);
        },
        onekeyCopy(text) {
            if (0 && navigator.clipboard) {
                navigator.clipboard.writeText(text);
            } else {
                var createInput = document.createElement('input');
                createInput.value = text;
                document.body.appendChild(createInput);
                createInput.select();
                document.execCommand("Copy");
                createInput.className = 'createInput';
                createInput.style.display = 'none';
            }
            this.$message.success("复制成功")
        },
    }
}
</script>

<style scoped>
.content {
    padding: 20px;
}

.top {
    padding: 20px;
}

.logo-box {
    width: 168px;
    height: 168px;
    padding: 5px;
    border: 1px solid #eaeaea;
    border-radius: 8px;
}

.logo-box .logo {
    border-radius: 4px;
    width: 100%;
    height: 100%;
}

.topline {
    border-top: 1px solid #f2f2f2;
}
</style>