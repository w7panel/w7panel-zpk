<template>
    <div style="height:100vh;">
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
                <a-breadcrumb-item><span class="c-33 fw-400">应用基础信息修改</span></a-breadcrumb-item>
            </a-breadcrumb>
            <div class="zpk-page-header-actions">
                <a-button type="outline" @click="openYamlPreview">预览 YAML</a-button>
            </div>
        </div>
        <a-spin :loading="deleteLoading" class="edit-spin">
            <div v-if="manifest && (!noPlatform || isCreate)">
                <div class="zpk-page-toolbar edit-package-toolbar">
                    <div class="zpk-toolbar-left">
                        <a-button @click="dependsIndex = -1;"
                            :type="dependsIndex == -1 ? 'primary' : 'secondary'">主应用</a-button>
                        <div v-for="(item, index) in depends" :key="item.identifie" style="position:relative;">
                            <a-button :type="dependsIndex == index ? 'primary' : 'secondary'"
                                @click="dependsIndex = index; edit({ stop: true })">{{ item.identifie }}</a-button>
                            <span v-if="!isManagedDependency(item)" @click="delDepend(index)"
                                class="depend-close c-red fs-20 cursor">×</span>
                        </div>
                        <a-button @click="openAddDepend">
                            <template #icon><icon-plus /></template>
                            添加子应用
                        </a-button>
                        <a-button @click="openImportDepend">
                            <template #icon><icon-download /></template>
                            导入子应用
                        </a-button>
                    </div>
                </div>
                <files-manifest v-show="dependsIndex == -1" :data="manifest" :version_id="version_id" ref="form"
                    :option="{ edit: true, imginstall: true, mainapp: true, app_ports: this.app_ports }"
                    :identifie="identifie" @addfile="addfileInside" @complete="complete"
                    @environment-nginx-gateway-change="handleEnvironmentNginxGatewayChange"
                    @structure="structure"></files-manifest>
                <files-manifest v-for="(item, index) in depends" :key="item.identifie" :ref="'depends' + index"
                    v-show="dependsIndex == index" :data="depends[index].manifest"
                    :option="{ pureManifest: true, imginstall: true, required: item.required, app_ports: this.app_ports }"
                    :identifie="item.identifie" @complete="dependsComplete" @structure="structure"></files-manifest>
            </div>
            <div v-else>
                <a-empty description="" class="manifest-empty">
                    <span class="c-99">暂无数据，点击</span>
                    <span class="cursor c-blue" @click="isCreate = true;">创建后端包配置</span>
                </a-empty>
            </div>
        </a-spin>

        <depend-picker v-model:visible="importPicker.show" title="导入子应用" action-text="导入"
            :action-loading="importPicker.importing" @select="importChild" @close="closeImportDepend" />
    </div>
</template>

<script>
import myAxios from '@/utils';
import filesManifest from '@/components/files-manifest.vue';
import dependPicker from '@/components/depend-picker.vue';
import jsyaml from "js-yaml";
import { confirm, messageError, messageSuccess } from '@/utils/ui-feedback';
import {
    childImportRepositoryBaseURL,
    importChildApplication,
    importedChildFilePath,
    getImportedChildIdentifies,
    saveImportedChildren,
    removeImportedChildren,
} from '@/utils/child-app-import';
import {
    environmentNginxDependency,
    withEnvironmentNginxPvcDependencySource,
} from '@/utils/environment-app';
import { IconArrowLeft, IconDownload, IconPlus } from '@arco-design/web-vue/es/icon';
const defaultManifest = `application:
    name: ''
    identifie: ''
    description: ''
    author: ''
platform:
    baseInfo:
        name: ''
        identifie: ''
        description: ''
    container:
        containerPort: 80
`;

export default {
    components: { filesManifest, dependPicker, IconArrowLeft, IconDownload, IconPlus },
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

            importPicker: {
                show: false,
                importing: false,
            },

            app_ports: [],

            noPlatform: true,
            isCreate: false,
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
            this.updateAppPorts();
        },
        depends: {
            handler() {
                this.updateAppPorts();
            },
            deep: true,
        },
    },
    methods: {
        getManifestIdentifie(json, fallback) {
            let identifie = json?.platform?.baseInfo?.identifie || json?.application?.identifie || fallback || '';
            let author = json?.application?.author || '';
            if (identifie && author && !/^[^-]+-.+$/.test(identifie)) {
                return author + '-' + identifie;
            }
            return identifie;
        },
        extractManifestPorts(manifest, fallback) {
            let json = jsyaml.load(manifest || '') || {};
            let ports = [];
            json?.platform?.['container-v2']?.forEach?.(container => {
                ports = ports.concat(container?.ports?.map?.(item => item?.port ?? item?.containerPort) || []);
                if (container?.port || container?.containerPort) {
                    ports.push(container.port ?? container.containerPort);
                }
            });
            let legacyContainer = json?.platform?.container;
            if (legacyContainer) {
                ports = ports.concat(legacyContainer?.ports?.map?.(item => item?.port ?? item?.containerPort) || []);
                if (legacyContainer?.port || legacyContainer?.containerPort) {
                    ports.push(legacyContainer.port ?? legacyContainer.containerPort);
                }
            }
            if (Array.isArray(json?.platform?.port)) {
                ports = ports.concat(json.platform.port.map(item => item?.port ?? item?.containerPort ?? item));
            } else if (json?.platform?.port) {
                ports.push(json.platform.port);
            }
            return {
                name: this.getManifestIdentifie(json, fallback),
                title: json?.platform?.baseInfo?.name || json?.application?.name || '',
                port: [...new Set(ports.filter(item => item !== '' && item !== undefined && item !== null).map(item => String(item)))]
            };
        },
        updateAppPorts() {
            let arr = [];
            let main = this.extractManifestPorts(this.manifest, this.identifie);
            if (main.name && main.port.length) {
                arr.push(main);
            }
            for (let i = 0; i < this.depends.length; i++) {
                let item = this.extractManifestPorts(this.depends[i]?.manifest || '', this.depends[i]?.identifie);
                if (item.name && item.port.length) {
                    arr.push(item);
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
        getActiveManifestRef() {
            if (this.dependsIndex > -1) {
                return this.$refs['depends' + this.dependsIndex]?.[0];
            }
            return this.$refs.form;
        },
        openYamlPreview() {
            this.getActiveManifestRef()?.openYamlPreview?.();
        },
        delDepend(index) {
            if (this.deleteLoading) { return }
            if (this.isManagedDependency(this.depends[index])) { return }
            let file = this.tree.find(i => i.label == (this.depends[index]?.identifie) + '/manifest.yaml');
            if (file) {
                myAxios.post('/respo/manifest/file', {
                    identifie: this.identifie,
                    filename: file.label,
                    content: '',
                    version: this.version_id,

                }).then(res => {
                    this.getInfo(this.identifie, () => {
                        this.publish(1);
                        setTimeout(() => {
                            this.getFile();
                        }, 300)
                    })
                });
            }
            this.deleteLoading = true;
            this.$refs.form?.delDepend(index);
            this.$nextTick(() => {
                this.$refs.form.submit({ stop: true }, () => {
                    this.getManifest().finally(() => {
                        this.deleteLoading = false;
                    });
                });
            })
        },
        openAddDepend() {
            this.dependsIndex = -1;
            this.$refs.form?.openAddDepend();
        },
        isManagedDependency(item) {
            return this.$refs.form?.isEnvironmentFixedDependency?.(item) || false;
        },
        async persistEnvironmentManifest() {
            const rootRef = this.$refs.form;
            if (rootRef?.form?.type != 'environment' || !rootRef?.json) {
                return;
            }
            // app_ports is rebuilt from the imported child manifests, so the
            // gateway backend can use the dependency's declared service port.
            this.updateAppPorts();
            rootRef.syncEnvironmentIngress?.(this.app_ports);
            const content = jsyaml.dump(rootRef.json);
            await myAxios.post('/respo/manifest/file', {
                identifie: this.identifie,
                filename: 'manifest.yaml',
                content,
                version: this.version_id,
            });
        },
        prepareEnvironmentNginxEntry(entry) {
            if (!entry) { return { manifest: {}, changed: false }; }
            let source = entry.data || entry.manifest || {};
            if (typeof source == 'string') {
                try {
                    source = jsyaml.load(source) || {};
                } catch {
                    source = {};
                }
            }
            const before = JSON.stringify(source);
            const manifest = withEnvironmentNginxPvcDependencySource(
                source,
                this.getManifestIdentifie(this.$refs.form?.json, this.identifie),
            );
            entry.data = manifest;
            entry.manifest = jsyaml.dump(manifest);
            return {
                manifest,
                changed: JSON.stringify(manifest) != before,
            };
        },
        openImportDepend() {
            this.importPicker.show = true;
        },
        closeImportDepend() {
            this.importPicker.show = false;
        },
        async handleEnvironmentNginxGatewayChange({ enabled, finish } = {}) {
            const done = typeof finish == 'function' ? finish : () => { };
            try {
                if (enabled) {
                    const rootDependencies = this.$refs.form?.json?.platform?.depends || [];
                    const hasDependency = rootDependencies.some(item =>
                        item?.identifie == environmentNginxDependency.identifie
                        && String(item?.from || '').trim())
                        && Object.prototype.hasOwnProperty.call(
                            this.list || {},
                            importedChildFilePath(environmentNginxDependency.identifie),
                        );
                    if (!hasDependency) {
                        const dependency = {
                            identifie: environmentNginxDependency.identifie,
                            name: environmentNginxDependency.name,
                            subidentifie: '',
                            subname: '',
                            required: true,
                            type: 'in',
                            from: environmentNginxDependency.source,
                        };
                        const entries = await importChildApplication(myAxios, { dependency });
                        const nginxEntry = entries.find(entry =>
                            entry?.identifie == environmentNginxDependency.identifie);
                        if (!nginxEntry) {
                            throw new Error('导入结果中缺少 NGINX 子应用 manifest');
                        }
                        this.prepareEnvironmentNginxEntry(nginxEntry);
                        const result = await saveImportedChildren(myAxios, {
                            rootRef: this.$refs.form,
                            rootIdentifie: this.identifie,
                            versionId: this.version_id,
                            entries,
                            sourceDependency: dependency,
                            existingDependencies: this.depends,
                        });
                        this.applyImportedChildrenResult(result);
                    }
                    done(true);
                    await this.persistEnvironmentManifest();
                    messageSuccess('NGINX 网关及其子应用导入成功');
                    return;
                }

                const identifies = getImportedChildIdentifies(
                    environmentNginxDependency.identifie,
                    this.list,
                    this.depends,
                );
                const result = await removeImportedChildren(myAxios, {
                    rootRef: this.$refs.form,
                    rootIdentifie: this.identifie,
                    versionId: this.version_id,
                    list: this.list,
                    identifies,
                });
                this.applyRemovedChildrenResult(result);
                done(true);
                await this.persistEnvironmentManifest();
                messageSuccess('NGINX 网关及其子应用已删除');
            } catch (error) {
                done(false);
                messageError(error?.response?.data?.error || error?.message || (enabled
                    ? '导入 NGINX 网关失败'
                    : '删除 NGINX 网关失败'));
            }
        },
        async importChild(record, tab = 'local') {
            if (!record?.identifie || this.importPicker.importing) { return; }
            const dependency = {
                identifie: record.identifie,
                goodsId: Number(record.goods_id || record.goodsId || record.id || 0),
                name: record.name || record.identifie,
                subidentifie: '',
                subname: '',
                required: true,
                type: 'in',
                from: childImportRepositoryBaseURL(tab),
                version: record.version?.name || '',
            };
            this.importPicker.importing = true;
            try {
                const entries = await importChildApplication(myAxios, {
                    dependency,
                });
                const result = await saveImportedChildren(myAxios, {
                    rootRef: this.$refs.form,
                    rootIdentifie: this.identifie,
                    versionId: this.version_id,
                    entries,
                    sourceDependency: dependency,
                    existingDependencies: this.depends,
                });
                this.applyImportedChildrenResult(result);
                this.importPicker.show = false;
                messageSuccess('子应用导入成功');
            } catch (error) {
                messageError(error?.response?.data?.error || error?.message || '导入子应用失败');
            } finally {
                this.importPicker.importing = false;
            }
        },
        applyImportedChildrenResult({ imported = [], dependencies = [], rootManifest = '' } = {}) {
            if (this.$refs.form?.form?.type != 'environment') {
                this.manifest = rootManifest;
            }
            this.json = this.$refs.form?.json || {};
            imported.forEach((entry, index) => {
                const file = importedChildFilePath(entry.identifie);
                this.list[file] = entry.manifest;
                if (!this.tree.some(item => item.label === file)) {
                    this.tree.push({ label: file });
                }
                this.depends.push({
                    ...dependencies[index],
                    manifest: entry.manifest,
                    title: file,
                });
            });
            // Keep the editor on the main application after importing child
            // manifests; the user can switch to a child explicitly.
            this.dependsIndex = -1;
        },
        applyRemovedChildrenResult({ existingFiles = [], rootManifest = '', identifies = [] } = {}) {
            if (this.$refs.form?.form?.type != 'environment') {
                this.manifest = rootManifest;
            }
            this.json = this.$refs.form?.json || {};
            existingFiles.forEach(file => {
                delete this.list[file];
                this.tree = this.tree.filter(item => item.label != file);
            });
            const removed = new Set(identifies);
            this.depends = this.depends.filter(item => !removed.has(item?.identifie));
            this.dependsIndex = -1;
        },
        addfileInside(json, yaml, data) {
            this.deleteLoading = true;
            myAxios.post('/respo/manifest/file', {
                identifie: this.identifie,
                filename: 'manifest.yaml',
                content: yaml,
                version: this.version_id,
            }).then(async (res) => {
                if (this.tree.find(i => i.label == data.file)) {
                    await this.getFile();
                    await this.getManifest();
                    this.dependsIndex = this.depends.length - 1;
                    return
                }
                myAxios.post('/respo/manifest/file', {
                    identifie: this.identifie,
                    filename: data.file,
                    content: data.cont,
                    version: this.version_id,

                }).then(async () => {
                    this.deleteLoading = false;
                    await this.getFile();
                    await this.getManifest();
                    this.dependsIndex = this.depends.length - 1;
                }).catch(() => {
                    this.deleteLoading = false;
                });
            }).catch((error) => {
                this.deleteLoading = false;
                if (error?.response?.data?.error) {
                    messageError(error.response.data.error);
                }
            });
        },
        getManifest() {
            this.deleteLoading = true;
            return myAxios.get('/respo/v2/info/' + this.identifie + '/' + this.version_id).then(res => {
                let nativeManifest = res?.data?.data?.manifest
                this.manifest = nativeManifest || defaultManifest;

                this.json = jsyaml.load(this.manifest);

                if (nativeManifest) {
                    this.noPlatform = !this.json.platform;
                    if (this.json?.platform) {
                        this.json.platform.baseInfo = this.json.platform.baseInfo || {};
                        this.json.platform.baseInfo.identifie = this.json.platform.baseInfo.identifie || this.identifie;
                        this.json.platform.baseInfo.name = this.json.platform.baseInfo.name || this.json?.application?.name || '';
                        this.json.platform.baseInfo.description = this.json.platform.baseInfo.description || this.json?.application?.description || '';
                    }
                    this.manifest = jsyaml.dump(this.json);
                } else {
                    this.noPlatform = true;
                    this.json.platform.baseInfo.identifie = this.identifie;
                    this.json.application.identifie = this.identifie;
                    this.manifest = jsyaml.dump(this.json);
                }
                this.vtitle = this.json?.application?.name || '';
                if (this.json?.platform?.depends) {
                    this.dependsIndex = -1;
                    let depends = this.json?.platform?.depends || [];
                    depends = depends.filter(i => i.type !== 'out'
                        && (!String(i.from || '').trim()
                            || this.list[importedChildFilePath(i.identifie)] !== undefined));
                    depends = depends.map(i => {
                        i.manifest = this.list[i.identifie + '/manifest.yaml'] || defaultManifest;
                        i.title = i.identifie + '/manifest.yaml';
                        return i;
                    });
                    this.depends = depends;
                }
                this.updateAppPorts();
            }).finally(() => {
                this.deleteLoading = false;
            });
        },

        dependsComplete(json, yaml, otherData) {

            myAxios.post('/respo/manifest/file', {
                identifie: this.identifie,
                filename: this.depends[this.dependsIndex].title,
                content: yaml,
                version: this.version_id,
            }).then(() => {
                this.depends.forEach((item, index) => {
                    if (item.identifie == json.application.identifie) {
                        item.name = json.application.name || '';
                        let v = otherData?.required ?? item.required;
                        if (v != item.required) {
                            let o = {
                                type: 'in',
                                identifie: item.identifie,
                                name: item.name,
                                required: v,
                            }
                            this.$refs.form.changeDepend(index, o)
                        }
                    }
                })

                this.getInfo(this.identifie, () => {
                    messageSuccess('操作成功');
                });
                this.getFile();
                this.getManifest();
            });
        },

        complete(json, yaml, otherData, callback) {
            myAxios.post('/respo/manifest/file', {
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
            this.deleteLoading = true;
            return myAxios.post('/respo/manifest/path-tree', {
                identifie: this.identifie,
                version: this.version_id
            }).then(res => {
                this.list = res?.data?.data?.list || {};
                let tree = [];
                for (let i in this.list) {
                    tree.push({ label: i })
                }
                this.tree = tree;
            }).finally(() => {
                this.deleteLoading = false;
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
                onOk: () => myAxios.post('/respo/manifest/file', {
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

.edit-package-toolbar {
    padding: 20px 20px 0;
}

.depend-close {
    position: absolute;
    top: -10px;
    right: -10px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    line-height: 1;
    background: #fff;
    border-radius: 50%;
}

.manifest-empty {
    padding-top: 80px;
}

.manifest-empty :deep(.arco-empty-image) {
    height: 200px;
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
