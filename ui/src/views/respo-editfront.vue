<template>
    <div style="height:100vh;">
        <div style="padding:20px; border-bottom:1px solid #E7E7E7;">
            <a-breadcrumb>
                <a-breadcrumb-item><router-link to="/zpk" class="c-99 fw-400">我的制品库</router-link></a-breadcrumb-item>
                <a-breadcrumb-item>
                    <router-link :to="{ path: '/zpk-version', query: { id: identifie, title: vtitle } }"
                        class="c-99 fw-400">版本管理</router-link>
                </a-breadcrumb-item>
                <a-breadcrumb-item><span class="c-33 fw-400">编辑前端托管</span></a-breadcrumb-item>
            </a-breadcrumb>
        </div>
        <a-spin :loading="deleteLoading" class="edit-spin">
            <div v-if="manifest">
                <div class="df jc-b" style="padding:20px 20px 0">
                    <div class="df">
                        <a-button @click="dependsIndex = -1;"
                            :type="dependsIndex == -1 ? 'primary' : 'secondary'">主应用</a-button>
                        <div v-for="(item, index) in depends" :key="item.identifie"
                            style="margin-left:10px;position:relative;">
                            <a-button :type="dependsIndex == index ? 'primary' : 'secondary'"
                                @click="dependsIndex = index; edit({ stop: true })">{{ item.identifie }}</a-button>
                        </div>
                    </div>
                </div>
                <files-manifestfront v-show="dependsIndex == -1" :data="manifest" :version_id="version_id" ref="form"
                    :option="{ edit: true, imginstall: true, mainapp: true, app_ports: this.app_ports }"
                    :identifie="identifie" @addfile="addfileInside" @complete="complete"
                    @structure="structure"></files-manifestfront>
                <files-manifestfront v-for="(item, index) in depends" :key="item.identifie" :ref="'depends' + index"
                    v-show="dependsIndex == index" :data="depends[index].manifest"
                    :option="{ pureManifest: true, imginstall: true }" :identifie="item.identifie"
                    :manifest-info="item" @complete="dependsComplete" @structure="structure"></files-manifestfront>
            </div>
        </a-spin>
    </div>

    <a-modal v-model:visible="impt.show" title="导入制品库" :width="560" :footer="false"
        modal-class="zpk-version-dialog">
        <div class="df" style="padding:10px;">
            <a-auto-complete v-model="impt.title" ref="imptzpk" :data="impt.options" placeholder="请选择制品库"
                style="width:400px;" size="large" allow-clear :filter-option="false" @search="addQuerySearch"
                @change="handleImportTitleChange" @select="addSelect" :spellcheck="false" />
            <a-button type="primary" @click="imptSubmit()" class="ml-10" size="large">确定</a-button>
        </div>
    </a-modal>
</template>

<script>
import myAxios from '@/utils';
import filesManifestfront from '@/components/files-manifestfront.vue';
import jsyaml from "js-yaml";
import { confirm, messageError, messageSuccess, messageWarning } from '@/utils/ui-feedback';
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

export default {
    inheritAttrs: false,
    components: { filesManifestfront },
    data() {
        return {
            identifie: '',
            version_id: '',
            manifest: '',
            tree: [],
            addfile: {
                show: false,
                filename: '',
            },
            list: {},

            vtitle: '',
            json: {},
            depends: [],
            dependsIndex: -1,
            deleteLoading: false,

            impt: {
                show: false,
                title: '',
                data: null,
                options: [],
            },

            app_ports: []

        }
    },
    async created() {
        this.identifie = this.$route.query.id;
        this.version_id = this.$route.query.versionid;
        await this.getFile();
        this.getManifest();
    },
    beforeUnmount() {
        window.removeEventListener('message', this.winMessage);
    },
    watch: {
        manifest() {
            this.refreshAppPorts();
        },
        depends: {
            handler() {
                this.refreshAppPorts();
            },
            deep: true,
        },
    },
    methods: {
        getManifestAppPorts(manifest) {
            let json = typeof manifest == 'string' ? (jsyaml.load(manifest) || {}) : (manifest || {});
            let ports = [];
            let addContainerPorts = (container) => {
                if (!container) { return }
                if (Array.isArray(container)) {
                    container.forEach(addContainerPorts);
                    return;
                }
                ports = ports.concat(container?.ports?.map?.(i => i?.port ?? i?.containerPort) || []);
                if (container.containerPort) {
                    ports.push(container.containerPort);
                }
            }
            addContainerPorts(json?.platform?.container);
            addContainerPorts(json?.platform?.['container-v2']);
            return {
                json,
                ports: [...new Set(ports.filter(i => i !== '' && i !== undefined && i !== null))],
            };
        },
        getManifestIdentifie(json, fallback) {
            let identifie = json?.application?.identifie || fallback || '';
            let author = json?.application?.author || '';
            if (identifie && author && !/^[^-]+-.+$/.test(identifie)) {
                return author + '-' + identifie;
            }
            return identifie;
        },
        refreshAppPorts() {
            let arr = [];
            let main = this.getManifestAppPorts(this.manifest);
            if (main.ports.length) {
                arr.push({
                    name: this.getManifestIdentifie(main.json, this.identifie),
                    title: main.json?.application?.name,
                    port: main.ports,
                });
            }
            for (let i = 0; i < this.depends.length; i++) {
                let manifest = this.depends[i]?.manifest || '';
                let { json, ports } = this.getManifestAppPorts(manifest);
                if (ports.length) {
                    arr.push({
                        name: this.getManifestIdentifie(json, this.depends[i].identifie),
                        title: json?.application?.name,
                        port: ports
                    })
                }
            }
            this.app_ports = arr;
        },
        structure() {
            let app_name = '';
            if (window.__MICRO_APP_ENVIRONMENT__) {
                let data = window.microApp?.getData();
                let info = data?.appInfo() || {};
                app_name = info?.app_name || '';
            }
            let url = window?.$wujie?.props?.url;
            let data = {
                app_name: app_name,
                zip_url: url + '/respo/v2/info/' + this.identifie + '/' + this.version_id,
            }
            window.parent.postMessage({ type: "structure", data: data }, "*");
            window.addEventListener('message', this.winMessage);
        },
        winMessage(e) {
            let msg = e.data;
            if (msg.type == "image_build_ok") {
                let image = msg.data;
                let ref = this.$refs.form;
                if (this.dependsIndex > -1) { ref = this.$refs['depends' + this.dependsIndex][0]; }
                ref?.getCreateImg(image);
                window.parent.postMessage({ type: "getImgval" }, "*");
            }
        },
        imptSubmit() {
            if (!this.impt.data) { messageWarning('请选择制品库'); return; }
            this.jsonp('https://console.w7.cc/zpk?path=/respo/v2/info/' + this.impt.data.identifie + '/' + this.version_id, 'getimportmanifest', data => {
                let manifest = data?.data?.manifest || defaultManifest;
                let json = jsyaml.load(manifest);
                let ref = this.$refs.form;
                if (this.dependsIndex > -1) { ref = this.$refs['depends' + this.dependsIndex][0]; }
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
            })
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
            }).then(async (res) => {
                if (this.tree.find(i => i.label == data.file)) {
                    await this.getFile();
                    this.getManifest();
                    return
                }
                myAxios.post('/respo/file', {
                    identifie: this.identifie,
                    filename: data.file,
                    content: data.cont,
                    version: this.version_id,
                }).then(async () => {
                    await this.getFile();
                    this.getManifest();
                });
            }).catch((error) => {
                if (error?.response?.data?.error) {
                    messageError(error.response.data.error);
                }
            });
        },
        getManifest() {
            return myAxios.get('/respo/v2/info/' + this.identifie + '/' + this.version_id).then(res => {
                this.manifest = res?.data?.manifest || res?.data?.data?.manifest || defaultManifest;
                this.json = jsyaml.load(this.manifest);
                this.vtitle = this.json?.application?.name || '';

                if (this.json?.platform?.depends) {
                    this.dependsIndex = -1;
                    let depends = this.json?.platform?.depends || [];
                    depends = depends.filter(i => i.type !== 'out');
                    depends.forEach(i => {
                        i.manifest = this.list[i.identifie + '/manifest.yaml'] || defaultManifest;
                        i.title = i.identifie + '/manifest.yaml';
                    });
                    this.depends = JSON.parse(JSON.stringify(depends))
                }
            });
        },

        dependsComplete(json, yaml) {
            myAxios.post('/respo/file', {
                identifie: this.identifie,
                filename: this.depends[this.dependsIndex].title,
                content: yaml,
                version: this.version_id,
            }).then(() => {
                this.getInfo(this.identifie, () => {
                    messageSuccess('操作成功');
                });
                this.getFile();
            });
        },

        complete(json, yaml, otherData, callback) {
            myAxios.post('/respo/file', {
                identifie: this.identifie,
                filename: 'manifest.yaml',
                content: yaml,
                version: this.version_id,
            }).then((res) => {
                if (otherData?.editfile) {
                    (typeof callback == 'function') && callback();
                    if (/\.yaml$/.test(otherData.editfile)) {
                        this.$router.push('/zpk-manifest-editor?version_id=' + this.version_id + '&identifie=' + this.identifie + '&filename=' + otherData.editfile + '&vtitle=' + this.vtitle);
                        return;
                    }
                    this.$router.push('/zpk-fileadd?version_id=' + this.version_id + '&identifie=' + this.identifie + '&filename=' + otherData.editfile + '&vtitle=' + this.vtitle);
                    return;
                }
                if (otherData?.stop) {
                    (typeof callback == 'function') && callback();
                    return;
                }
                messageSuccess('修改成功');
                (typeof callback == 'function') && callback();
                this.getInfo(this.identifie, () => {
                    this.publish(true).then(() => {
                        this.$router.push('/zpk-version?id=' + this.identifie + '&title=' + (json?.application?.name) || '')
                    });
                });
            }).then(res => {

            }).catch((error) => {
                if (error?.response?.data?.error) {
                    messageError(error.response.data.error);
                }
            });
        },
        publish(noback) {
            return myAxios.post('/respo/publish', { identifie: this.identifie, version: this.version_id }).then(res => {
                if (noback) { return; }
                if (res.data?.message || res.data?.data?.message) {
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
            myAxios.get('/respo/v2/info/' + id + '/' + this.version_id).then(res => {
                callback && callback();
            }).catch(() => {
                if (n > 10) { return }
                setTimeout(() => {
                    this.getInfo(id, callback, n + 1);
                }, 1000);
            });
        },
        getFile() {

            return myAxios.post('/respo/path-tree', {
                identifie: this.identifie,
                version: this.version_id,
            }).then(res => {
                this.list = res?.data?.data?.list || {};
                let tree = [];
                for (let i in this.list) {
                    tree.push({ label: i })
                }
                this.tree = tree;
                this.depends = this.json?.platform?.depends || [];
                this.depends.forEach(i => {
                    i.manifest = this.list[i.identifie + '/manifest.yaml'] || defaultManifest;
                    i.title = i.identifie + '/manifest.yaml';
                });
                this.depends = JSON.parse(JSON.stringify(this.depends));
            })
        },
        edit(data) {
            this.$refs.form.submit(data);
        },
        del(row) {
            confirm({
                title: '提示',
                content: '确定要删除"' + row.label + '"吗',
                confirmButtonText: "确定",
                cancelButtonText: "取消",
                onOk: () => myAxios.post('/respo/file', {
                    identifie: this.identifie,
                    filename: row.label,
                    content: '',
                    version: this.version_id,
                }).then(res => {
                    messageSuccess('删除成功');
                    this.getInfo(this.identifie, () => {
                        this.publish(1);
                        setTimeout(() => {
                            this.getFile();
                        }, 300)
                    })
                })
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
.edit-spin {
    display: block;
    min-height: calc(100vh - 65px);
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

.table thead tr:first-child td {
    background: #f3f3f3;
    border-top: 0;
}
</style>
<style>
.zpk-version-dialog .arco-modal-header {
    border-bottom: 1px solid #dcdcdc;
}

.zpk-version-dialog .arco-modal-footer {
    text-align: center;
    border-top: 1px solid #dcdcdc;
}
</style>
