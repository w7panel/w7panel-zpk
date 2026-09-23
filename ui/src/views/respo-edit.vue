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
                        <div class="application-tab-item">
                            <a-button @click="dependsIndex = -1;"
                                :type="dependsIndex == -1 ? 'primary' : 'secondary'">
                                主应用
                                <span class="application-index-label">app[{{ getManifestOrder(manifest) ?? 0 }}]</span>
                            </a-button>
                        </div>
                        <div v-for="(item, index) in depends" :key="item.identifie" class="depend-item">
                            <div class="depend-tab">
                                <a-button :type="dependsIndex == index ? 'primary' : 'secondary'"
                                    @click="selectChildApplication(index)">
                                    {{ item.identifie }}
                                    <span v-if="getManifestOrder(item.manifest) !== null" class="application-index-label">
                                        app[{{ getManifestOrder(item.manifest) }}]
                                    </span>
                                </a-button>
                                <div v-if="!isManagedDependency(item) || isImportedDependency(item)
                                    || childUpdates[item.identifie]?.available" class="application-tab-corner">
                                    <span v-if="isImportedDependency(item)"
                                        class="application-tab-badge application-imported-badge">导入</span>
                                    <a-popover v-if="childUpdates[item.identifie]?.available" position="bottom"
                                        trigger="click" :content-style="{ padding: '6px 10px 16px' }">
                                        <div class="child-update-trigger" @click.stop>
                                            <icon-exclamation-circle-fill />
                                            <span>新版本</span>
                                        </div>
                                        <template #content>
                                            <div class="child-update-popover">
                                                <div class="child-update-title">
                                                    <icon-exclamation-circle-fill />
                                                    <span>新版本</span>
                                                </div>
                                                <div class="child-update-description">
                                                    当前子应用有新版发布，可更新至
                                                    {{ childUpdates[item.identifie].latestVersion }}
                                                </div>
                                                <div class="child-update-actions">
                                                    <a-button size="small" type="primary"
                                                        :loading="childUpdateLoading == item.identifie"
                                                        @click="updateImportedChild(item)">立即更新</a-button>
                                                </div>
                                            </div>
                                        </template>
                                    </a-popover>
                                    <span v-if="!isManagedDependency(item)" class="depend-close cursor"
                                        title="删除子应用" @click.stop="delDepend(index)">×</span>
                                </div>
                            </div>
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
                    @tradition-nginx-gateway-change="handleTraditionNginxGatewayChange"
                    @structure="structure"></files-manifest>
                <files-manifest v-for="(item, index) in depends" :key="item.identifie" :ref="'depends' + index"
                    v-show="dependsIndex == index" :data="depends[index].manifest"
                    :option="{ pureManifest: true, imginstall: true, required: item.required, app_ports: this.app_ports, startParamsOnly: isImportedDependency(item) }"
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
import { confirm, messageError, messageSuccess, messageWarning } from '@/utils/ui-feedback';
import {
    childImportRepositoryBaseURL,
    fetchChildImportListFromSource,
    importChildApplication,
    getImportedChildIdentifies,
    importedChildFilePath,
    isChildImportVersionNewer,
    removeImportedChildren,
    saveImportedChildren,
} from '@/utils/child-app-import';
import {
    traditionToolDependency,
} from '@/components/manifest/tradition-manifest';
import {
    IconArrowLeft,
    IconDownload,
    IconExclamationCircleFill,
    IconPlus,
} from '@arco-design/web-vue/es/icon';
const defaultManifest = `application:
    name: ''
    identifie: ''
    order: 0
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
    components: {
        filesManifest,
        dependPicker,
        IconArrowLeft,
        IconDownload,
        IconExclamationCircleFill,
        IconPlus,
    },
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
            childUpdates: {},
            childUpdateLoading: '',
            childUpdateCheckToken: 0,

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
        this.childUpdateCheckToken++;
        window.removeEventListener('message', this.winMessage);
    },
    computed: {
        importedChildIdentifySet() {
            const identifies = new Set();
            this.depends
                .filter(item => String(item?.from || '').trim())
                .forEach(item => {
                    getImportedChildIdentifies(item.identifie, this.list, this.depends)
                        .forEach(identifie => identifies.add(identifie));
                });
            return identifies;
        },
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
        getManifestOrder(manifest) {
            try {
                const json = typeof manifest == 'string'
                    ? (jsyaml.load(manifest) || {})
                    : (manifest || {});
                const order = json?.application?.order;
                return order === undefined || order === null || order === ''
                    ? null
                    : order;
            } catch {
                return null;
            }
        },
        sortChildApplications(dependencies = []) {
            return dependencies
                .map((dependency, index) => ({
                    dependency,
                    index,
                    order: this.getManifestOrder(dependency?.manifest),
                }))
                .sort((first, second) => {
                    const firstOrder = Number(first.order);
                    const secondOrder = Number(second.order);
                    const firstHasOrder = first.order !== null && Number.isFinite(firstOrder);
                    const secondHasOrder = second.order !== null && Number.isFinite(secondOrder);
                    if (firstHasOrder && secondHasOrder && firstOrder !== secondOrder) {
                        return firstOrder - secondOrder;
                    }
                    if (firstHasOrder !== secondHasOrder) return firstHasOrder ? -1 : 1;
                    return first.index - second.index;
                })
                .map(item => item.dependency);
        },
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
        ensureMainManifestSaved() {
            if (!this.$refs.form?.hasUnsavedChanges?.()) return true;
            this.dependsIndex = -1;
            messageWarning('请先保存主应用配置');
            return false;
        },
        selectChildApplication(index) {
            if (this.dependsIndex == -1 && !this.ensureMainManifestSaved()) return;
            this.dependsIndex = index;
        },
        delDepend(index) {
            if (this.deleteLoading || !this.ensureMainManifestSaved()) { return }
            const dependency = this.depends[index];
            if (!dependency || this.isManagedDependency(dependency)) { return }
            confirm({
                title: '删除子应用',
                content: `确定要删除“${dependency.name || dependency.identifie}”吗？`,
                confirmButtonText: '确定删除',
                cancelButtonText: '取消',
                onOk: () => this.removeChildDependency(dependency),
            });
        },
        async removeChildDependency(dependency) {
            if (this.deleteLoading || !dependency?.identifie
                || !this.ensureMainManifestSaved()) return;
            const rootRef = this.$refs.form;
            if (!rootRef?.json) {
                messageError('主应用 manifest 尚未加载完成');
                return;
            }
            this.deleteLoading = true;
            try {
                const result = await removeImportedChildren(myAxios, {
                    rootRef,
                    rootIdentifie: this.identifie,
                    versionId: this.version_id,
                    list: this.list,
                    identifies: [dependency.identifie],
                    orderedIdentifies: this.depends.map(item => item.identifie),
                });
                this.applyRemovedChildrenResult(result);
                await this.getFile();
                await this.getManifest();
                await this.$nextTick();
                this.$refs.form?.markManifestSaved?.();
                await this.publish(1);
                messageSuccess('子应用已删除');
            } catch (error) {
                messageError(error?.response?.data?.error || error?.message || '删除子应用失败');
            } finally {
                this.deleteLoading = false;
            }
        },
        openAddDepend() {
            if (!this.ensureMainManifestSaved()) return;
            this.dependsIndex = -1;
            this.$refs.form?.openAddDepend();
        },
        isManagedDependency(item) {
            return this.$refs.form?.isTraditionFixedDependency?.(item) || false;
        },
        isImportedDependency(item) {
            return Boolean(item?.identifie)
                && this.importedChildIdentifySet.has(item.identifie);
        },
        async persistTraditionManifest() {
            const rootRef = this.$refs.form;
            if (rootRef?.form?.type != 'tradition' || !rootRef?.json) {
                return;
            }
            rootRef.syncTraditionIngress?.();
            const content = jsyaml.dump(rootRef.json);
            await myAxios.post('/respo/manifest/file', {
                identifie: this.identifie,
                filename: 'manifest.yaml',
                content,
                version: this.version_id,
            });
            this.manifest = content;
            rootRef.markManifestSaved?.();
        },
        async ensureTraditionToolDependency() {
            const rootRef = this.$refs.form;
            if (rootRef?.form?.type != 'tradition' || !rootRef?.json) {
                return;
            }
            rootRef.syncTraditionDependency?.();
            rootRef.syncImportedDependenciesToManifest?.();
            const file = importedChildFilePath(traditionToolDependency.identifie);
            const dependencyExists = (rootRef.json?.platform?.depends || []).some(item =>
                item?.identifie == traditionToolDependency.identifie
                && String(item?.from || '').trim());
            const existingContent = this.list?.[file];
            if (dependencyExists && existingContent !== undefined) {
                return;
            }

            const dependency = {
                identifie: traditionToolDependency.identifie,
                name: traditionToolDependency.name,
                subidentifie: '',
                subname: '',
                required: true,
                type: 'in',
                from: traditionToolDependency.source,
            };
            const entries = await importChildApplication(myAxios, { dependency });
            const toolEntry = entries.find(entry =>
                entry?.identifie == traditionToolDependency.identifie);
            if (!toolEntry) {
                throw new Error('导入结果中缺少传统应用工具子应用 manifest');
            }
            const replaceExisting = (this.depends || []).some(item =>
                item?.identifie == traditionToolDependency.identifie)
                || existingContent !== undefined;
            const result = await saveImportedChildren(myAxios, {
                rootRef,
                rootIdentifie: this.identifie,
                versionId: this.version_id,
                entries,
                sourceDependency: dependency,
                existingDependencies: this.depends,
                existingManifests: this.list,
                replaceExisting,
            });
            this.applyImportedChildrenResult(result);
            await this.persistChildOrders();
        },
        openImportDepend() {
            if (!this.ensureMainManifestSaved()) return;
            this.importPicker.show = true;
        },
        closeImportDepend() {
            this.importPicker.show = false;
        },
        async handleTraditionNginxGatewayChange({ enabled, finish } = {}) {
            const done = typeof finish == 'function' ? finish : () => { };
            const changeGateway = async () => {
                try {
                    await this.ensureTraditionToolDependency();
                    done(true);
                    await this.$refs.form?.saveFormulaTypeSetting?.();
                    await this.persistTraditionManifest();
                    messageSuccess(enabled ? 'NGINX 网关已开启' : 'NGINX 网关已关闭');
                } catch (error) {
                    done(false);
                    messageError(error?.response?.data?.error || error?.message || (enabled
                        ? '开启 NGINX 网关失败'
                        : '关闭 NGINX 网关失败'));
                }
            };
            const rootRef = this.$refs.form;
            if (rootRef?.hasUnsavedChanges?.()) {
                rootRef.submit(
                    { stop: true },
                    changeGateway,
                    () => done(false),
                );
                return;
            }
            await changeGateway();
        },
        async importChild(record, tab = 'local') {
            if (!record?.identifie || this.importPicker.importing
                || !this.ensureMainManifestSaved()) { return; }
            const sourceRepositoryURL = String(
                record.sourceRepositoryURL || childImportRepositoryBaseURL(tab),
            ).trim().replace(/\/+$/, '');
            const dependency = {
                identifie: record.identifie,
                goodsId: Number(record.goods_id || record.goodsId || record.id || 0),
                name: record.name || record.identifie,
                subidentifie: '',
                subname: '',
                required: true,
                type: 'in',
                from: sourceRepositoryURL,
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
                await this.persistChildOrders();
                this.$refs.form?.markManifestSaved?.();
                this.importPicker.show = false;
                messageSuccess('子应用导入成功');
            } catch (error) {
                messageError(error?.response?.data?.error || error?.message || '导入子应用失败');
            } finally {
                this.importPicker.importing = false;
            }
        },
        getImportedChildVersion(item) {
            if (item?.version) { return String(item.version); }
            try {
                const manifest = jsyaml.load(item?.manifest || '') || {};
                return String(manifest?.application?.version || '');
            } catch {
                return '';
            }
        },
        async checkImportedChildUpdates() {
            const checkToken = ++this.childUpdateCheckToken;
            const dependencies = this.depends.filter(item => String(item?.from || '').trim());
            this.childUpdates = {};
            if (!dependencies.length) { return; }
            const groups = new Map();
            dependencies.forEach(item => {
                const source = String(item.from).trim();
                if (!groups.has(source)) { groups.set(source, []); }
                groups.get(source).push(item);
            });
            const updates = {};
            await Promise.all([...groups.entries()].map(async ([source, items]) => {
                try {
                    const list = await fetchChildImportListFromSource(myAxios, source, {
                        page: 1,
                        limit: 999,
                    });
                    const latestByIdentifie = new Map(list.map(record => [record.identifie, record]));
                    items.forEach(item => {
                        const record = latestByIdentifie.get(item.identifie);
                        const latestVersion = String(record?.version?.name || '');
                        const currentVersion = this.getImportedChildVersion(item);
                        if (isChildImportVersionNewer(latestVersion, currentVersion)) {
                            updates[item.identifie] = { available: true, latestVersion };
                        }
                    });
                } catch {
                    // 更新检测失败不影响 manifest 编辑。
                }
            }));
            if (checkToken === this.childUpdateCheckToken) {
                this.childUpdates = updates;
            }
        },
        async updateImportedChild(item) {
            const update = this.childUpdates[item?.identifie];
            if (!item?.from || !update?.available || this.childUpdateLoading
                || !this.ensureMainManifestSaved()) { return; }
            const dependency = {
                identifie: item.identifie,
                name: item.name || item.identifie,
                subidentifie: '',
                subname: '',
                required: item.required ?? true,
                type: 'in',
                from: item.from,
                version: update.latestVersion,
            };
            this.childUpdateLoading = item.identifie;
            try {
                const entries = await importChildApplication(myAxios, { dependency });
                const rootEntry = entries.find(entry => entry?.identifie == item.identifie);
                if (!rootEntry) {
                    throw new Error(`导入结果中缺少 ${item.identifie} 子应用 manifest`);
                }
                const result = await saveImportedChildren(myAxios, {
                    rootRef: this.$refs.form,
                    rootIdentifie: this.identifie,
                    versionId: this.version_id,
                    entries,
                    sourceDependency: dependency,
                    existingDependencies: this.depends,
                    existingManifests: this.list,
                    replaceExisting: true,
                });
                this.applyImportedChildrenResult(result);
                await this.persistChildOrders();
                if (item.identifie == traditionToolDependency.identifie) {
                    await this.persistTraditionManifest();
                }
                this.$refs.form?.markManifestSaved?.();
                await this.checkImportedChildUpdates();
                messageSuccess(`子应用已更新到 ${update.latestVersion}`);
            } catch (error) {
                messageError(error?.response?.data?.error || error?.message || '更新子应用失败');
            } finally {
                this.childUpdateLoading = '';
            }
        },
        applyImportedChildrenResult({ imported = [], dependencies = [], rootManifest = '' } = {}) {
            if (this.$refs.form?.form?.type != 'tradition') {
                this.manifest = rootManifest;
            }
            this.json = this.$refs.form?.json || {};
            imported.forEach((entry, index) => {
                const file = importedChildFilePath(entry.identifie);
                this.list[file] = entry.manifest;
                if (!this.tree.some(item => item.label === file)) {
                    this.tree.push({ label: file });
                }
                const nextDependency = {
                    ...(this.depends.find(item => item.identifie === entry.identifie) || {}),
                    ...dependencies[index],
                    manifest: entry.manifest,
                    title: file,
                };
                const existingIndex = this.depends.findIndex(item => item.identifie === entry.identifie);
                if (existingIndex >= 0) {
                    this.depends.splice(existingIndex, 1, nextDependency);
                } else {
                    this.depends.push(nextDependency);
                }
            });
            // Keep the editor on the main application after importing child
            // manifests; the user can switch to a child explicitly.
            this.dependsIndex = -1;
        },
        applyRemovedChildrenResult({
            existingFiles = [],
            orderedFiles = {},
            rootManifest = '',
            identifies = [],
        } = {}) {
            if (this.$refs.form?.form?.type != 'tradition') {
                this.manifest = rootManifest;
            }
            this.json = this.$refs.form?.json || {};
            existingFiles.forEach(file => {
                delete this.list[file];
                this.tree = this.tree.filter(item => item.label != file);
            });
            Object.entries(orderedFiles).forEach(([file, content]) => {
                this.list[file] = content;
                const dependency = this.depends.find(item => item?.title == file);
                if (dependency) dependency.manifest = content;
            });
            const removed = new Set(identifies);
            this.depends = this.depends.filter(item => !removed.has(item?.identifie));
            this.dependsIndex = -1;
        },
        async persistChildOrders() {
            const writes = [];
            this.depends.forEach((item, index) => {
                let manifest;
                try {
                    manifest = typeof item.manifest == 'string'
                        ? (jsyaml.load(item.manifest) || {})
                        : (item.manifest || {});
                } catch {
                    return;
                }
                if (!manifest.application) { return; }
                const order = index + 1;
                manifest.application.order = order;
                const content = jsyaml.dump(manifest);
                const filename = item.title || importedChildFilePath(item.identifie);
                item.manifest = content;
                this.list[filename] = content;
                writes.push(myAxios.post('/respo/manifest/file', {
                    identifie: this.identifie,
                    filename,
                    content,
                    version: this.version_id,
                }));
            });
            await Promise.all(writes);
        },
        async addfileInside(json, yaml, data) {
            if (!this.ensureMainManifestSaved()) return;
            this.deleteLoading = true;
            const childExists = this.tree.some(item => item.label == data.file);
            let childCreated = false;
            try {
                if (!childExists) {
                    await myAxios.post('/respo/manifest/file', {
                        identifie: this.identifie,
                        filename: data.file,
                        content: data.cont,
                        version: this.version_id,
                    });
                    childCreated = true;
                }
                try {
                    await myAxios.post('/respo/manifest/file', {
                        identifie: this.identifie,
                        filename: 'manifest.yaml',
                        content: yaml,
                        version: this.version_id,
                    });
                } catch (error) {
                    if (childCreated) {
                        await myAxios.post('/respo/manifest/file', {
                            identifie: this.identifie,
                            filename: data.file,
                            content: '',
                            version: this.version_id,
                        }).catch(() => undefined);
                    }
                    throw error;
                }
                await this.getFile();
                await this.getManifest();
                await this.persistChildOrders();
                await this.$nextTick();
                this.$refs.form?.markManifestSaved?.();
                this.dependsIndex = this.depends.length - 1;
            } catch (error) {
                messageError(error?.response?.data?.error || error?.message || '新增子应用失败');
            } finally {
                this.deleteLoading = false;
            }
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
                    this.depends = this.sortChildApplications(depends);
                }
                this.updateAppPorts();
                this.checkImportedChildUpdates();
            }).finally(() => {
                this.deleteLoading = false;
            });
        },

        async dependsComplete(json, yaml, otherData) {
            const dependency = this.depends[this.dependsIndex];
            if (!dependency) {
                messageError('未找到当前子应用');
                return;
            }
            const imported = this.isImportedDependency(dependency);
            const required = otherData?.required ?? dependency.required;
            if (required != dependency.required && !this.ensureMainManifestSaved()) return;
            let savedJSON = json;
            let savedYAML = yaml;
            let rootManifestYAML = '';
            if (imported) {
                try {
                    const original = jsyaml.load(dependency?.manifest || '') || {};
                    original.platform = original.platform || {};
                    original.platform.startParams = JSON.parse(JSON.stringify(
                        json?.platform?.startParams || [],
                    ));
                    savedJSON = original;
                    savedYAML = jsyaml.dump(original);
                } catch (error) {
                    messageError(error?.message || '导入子应用配置解析失败');
                    return;
                }
            }

            try {
                await myAxios.post('/respo/manifest/file', {
                    identifie: this.identifie,
                    filename: dependency.title,
                    content: savedYAML,
                    version: this.version_id,
                });
                dependency.manifest = savedYAML;
                if (!imported) {
                    dependency.name = savedJSON.application.name || '';
                }
                if (required != dependency.required) {
                    const rootDependencies = this.$refs.form?.form?.dependsIn || [];
                    const rootIndex = rootDependencies.findIndex(item =>
                        item?.identifie == dependency.identifie);
                    if (rootIndex >= 0) {
                        rootDependencies.splice(rootIndex, 1, {
                            ...rootDependencies[rootIndex],
                            name: dependency.name,
                            required,
                        });
                        this.$refs.form.syncImportedDependenciesToManifest?.();
                        const rootManifest = this.$refs.form?.getSavedManifest?.()
                            || jsyaml.load(this.manifest || '') || {};
                        rootManifest.platform = rootManifest.platform || {};
                        rootManifest.platform.depends = JSON.parse(JSON.stringify(
                            this.$refs.form.json?.platform?.depends || [],
                        ));
                        rootManifestYAML = jsyaml.dump(rootManifest);
                        dependency.required = required;
                    }
                }
                if (rootManifestYAML) {
                    await myAxios.post('/respo/manifest/file', {
                        identifie: this.identifie,
                        filename: 'manifest.yaml',
                        content: rootManifestYAML,
                        version: this.version_id,
                    });
                }
                await this.getFile();
                await this.getManifest();
                await this.$nextTick();
                this.$refs.form?.markManifestSaved?.();
                this.getInfo(this.identifie, () => {
                    messageSuccess('操作成功');
                });
            } catch (error) {
                messageError(error?.response?.data?.error || error?.message || '子应用配置保存失败');
            }
        },

        complete(json, yaml, otherData, callback, errorCallback) {
            const prepare = json?.application?.type == 'tradition'
                ? this.ensureTraditionToolDependency().then(() => {
                    json = this.$refs.form?.json || json;
                    yaml = jsyaml.dump(json);
                })
                : Promise.resolve();
            prepare.then(() => myAxios.post('/respo/manifest/file', {
                identifie: this.identifie,
                filename: 'manifest.yaml',
                content: yaml,
                version: this.version_id,

            })).then(() => {
                this.manifest = yaml;
                this.$refs.form?.markManifestSaved?.();
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
            }).catch((error) => {
                messageError(error?.response?.data?.error || error?.message || '制品配置保存失败');
                (typeof errorCallback == 'function') && errorCallback();
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
            myAxios.get('/respo/v2/info/' + id + '/' + this.version_id).then(() => {
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

.edit-package-toolbar .zpk-toolbar-left {
    align-items: center;
}

.application-tab-item,
.depend-item {
    position: relative;
    display: inline-flex;
    align-items: center;
}

.depend-tab {
    position: relative;
    display: inline-flex;
    align-items: center;
}

.application-tab-corner {
    position: absolute;
    top: -14px;
    right: -6px;
    z-index: 2;
    display: flex;
    align-items: center;
    gap: 4px;
}

.application-tab-badge {
    display: inline-flex;
    align-items: center;
    height: 16px;
    box-sizing: border-box;
    padding: 0 5px;
    font-size: 10px;
    font-weight: 500;
    line-height: 16px;
    white-space: nowrap;
    border-radius: 4px;
    box-shadow: 0 0 0 2px #fff;
}

.application-index-label {
    display: inline-flex;
    align-items: center;
    height: 16px;
    box-sizing: border-box;
    margin-left: 6px;
    padding: 0 4px;
    color: #fff;
    font-family: SFMono-Regular, Consolas, "Liberation Mono", monospace;
    font-size: 10px;
    font-weight: 500;
    line-height: 16px;
    white-space: nowrap;
    background: rgb(var(--red-6));
    border-radius: 4px;
}

.application-imported-badge {
    color: rgb(var(--red-6));
    background: #fff;
    border: 1px solid rgb(var(--red-2));
}

.child-update-trigger,
.child-update-title {
    display: inline-flex;
    align-items: center;
    color: rgb(var(--red-7));
}

.child-update-trigger {
    gap: 2px;
    height: 16px;
    box-sizing: border-box;
    padding: 0 5px;
    font-size: 10px;
    font-weight: 500;
    line-height: 16px;
    white-space: nowrap;
    background: rgb(var(--red-1));
    border-radius: 4px;
    box-shadow: 0 0 0 2px #fff;
    cursor: pointer;
}

.child-update-trigger svg {
    font-size: 11px;
}

.child-update-title {
    gap: 4px;
    font-size: 16px;
    font-weight: 600;
}

.child-update-description {
    margin-top: 10px;
    color: rgba(0, 0, 0, 0.6);
    text-align: center;
}

.child-update-actions {
    display: flex;
    justify-content: center;
    margin-top: 10px;
}

.depend-close {
    position: relative;
    top: 1px;
    left: 2px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    box-sizing: border-box;
    padding-bottom: 1px;
    color: rgb(var(--red-6));
    font-size: 15px;
    line-height: 1;
    background: #fff;
    border: 1px solid rgb(var(--red-2));
    border-radius: 50%;
    box-shadow: 0 0 0 2px #fff;
}

.depend-close:hover {
    color: #fff;
    background: rgb(var(--red-6));
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
