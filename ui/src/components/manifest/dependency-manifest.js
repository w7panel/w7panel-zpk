import jsyaml from 'js-yaml';
import myAxios from '@/utils/index';
import { messageWarning } from '@/utils/ui-feedback';
import {
    isTraditionAppDependency,
    traditionSysboxRootfsAnnotation,
    traditionSysboxRuntimeClassName,
} from '@/components/manifest/tradition-manifest';
import {
    applyPluginTraditionDependencyStartParams,
} from '@/components/manifest/plugin-app-manifest';

export function dependencyFormDefaults() {
    return {
        dependsIn: [],
        depends: [],
    };
}

export function dependencyManifestState() {
    return {
        dependForm: {
            show: false,
            editIndex: -1,
            identifie: '',
            identifie_before: '',
            identifie_last: '',
            name: '',
            required: true,
            from: '',
        },
        dependsList: {},
        subDependsList: {},
        dependPicker: {
            show: false,
            editIndex: -1,
        },
        addRules: {
            identifie: [
                { required: true, message: '内容不能为空', trigger: 'blur' },
                {
                    required: true,
                    trigger: 'blur',
                    validator: (value, callback) => {
                        if (/^[a-zA-Z0-9]+-[a-zA-Z0-9]+$/.test(value)) callback();
                        else callback('标识格式有误');
                    },
                },
            ],
            name: [{ required: true, message: '内容不能为空', trigger: 'blur' }],
        },
        sysboxDependencyManaged: false,
    };
}

export const dependencyManifestWatch = {
    'dependForm.identifie_before'() {
        this.dependForm.identifie = `${this.dependForm.identifie_before}-${this.dependForm.identifie_last}`;
    },
    'dependForm.identifie_last'() {
        this.dependForm.identifie = `${this.dependForm.identifie_before}-${this.dependForm.identifie_last}`;
    },
};

export const dependencyManifestMethods = {
    cleanupInactiveSysboxRuntime(platform = {}) {
        delete platform.hostUsers;
        if (platform.runtimeClassName == traditionSysboxRuntimeClassName) {
            delete platform.runtimeClassName;
        }
    },
    applyManifestDependencies(platform = {}) {
        let dependencies = this.form.dependsIn.concat(this.form.depends);
        if (this.form.type == 'app-plugin') {
            dependencies = applyPluginTraditionDependencyStartParams(
                dependencies,
                this.form.traditionName,
                this.form.traditionVersion,
            );
        }
        platform.depends = dependencies;
    },
    initDependencies(platform = {}) {
        this.form.dependsIn = (platform.depends || [])
            .filter(item => item.type != 'out' && !String(item.from || '').trim());
        this.form.dependsIn.forEach(item => { item.type = 'in'; });
        this.form.depends = (platform.depends || [])
            .filter(item => item.type == 'out' || String(item.from || '').trim());

        this.form.depends.forEach((item, index) => {
            if (this.isPluginTraditionDependency(item)) {
                item.temporary = true;
                item.required = true;
                item.multipleInstances = true;
            }
            if (this.form.type == 'app-plugin' && this.traditionList?.length) {
                const tradition = this.traditionList.find(record => (
                    record.identifie == this.form.traditionName
                ));
                if (tradition && item.identifie == tradition.identifie && item.name == tradition.name) {
                    item.temporary = true;
                }
            }
            this.getSubDepends(index);
        });
    },
    getDependName(record) {
        if (!record) return '';
        return record.name || this.dependsList?.[record.identifie] || record.identifie || '';
    },
    getSubDependsOptions(index) {
        return this.subDependsList?.[index] || { '': '无' };
    },
    getSubDependName(index, identifie) {
        return this.getSubDependsOptions(index)?.[identifie] || '';
    },
    normalizeDependList(list = []) {
        return (list || []).filter(item => item?.install_only_once).map(item => ({
            ...item,
            name: item.name || item.identifie,
        }));
    },
    mergeDependsList(list = []) {
        const dependsList = { ...this.dependsList };
        list.forEach(item => {
            if (item?.identifie) dependsList[item.identifie] = item.name || item.identifie;
        });
        this.dependsList = dependsList;
    },
    openDependPicker(index = -1) {
        if (this.form.type == 'tradition') return;
        this.dependPicker.show = true;
        this.dependPicker.editIndex = index;
    },
    closeDependPicker() {
        this.dependPicker.editIndex = -1;
    },
    selectDependFromPicker(record, tab = 'local') {
        if (!record?.identifie) return;
        const dependency = {
            identifie: record.identifie,
            goodsId: Number(record.goods_id || record.goodsId || record.id || 0),
            name: record.name || record.identifie,
            subidentifie: '',
            subname: '',
            required: false,
            type: 'out',
            from: tab == 'official' ? 'https://zpk.w7.cc' : '',
        };
        let index = this.dependPicker.editIndex;
        if (index >= 0 && this.form.depends[index]) {
            dependency.required = Boolean(this.form.depends[index].required);
            this.form.depends.splice(index, 1, dependency);
        } else {
            this.form.depends.push(dependency);
            index = this.form.depends.length - 1;
        }
        this.dependPicker.show = false;
        this.getSubDepends(index, tab);
        this.changeForm();
    },
    getSubDepends(index, source = '') {
        this.subDependsList[index] = { '': '无' };
        const identifie = this.form.depends?.[index]?.identifie;
        if (!identifie) return Promise.resolve();
        const localUrl = `/respo/v2/info/${identifie}/1.0.0`;
        const officialUrl = `https://zpk.w7.cc/zpk/respo/v2/info/${identifie}/1.0.0`;
        const urls = source == 'official' ? [officialUrl, localUrl] : [localUrl, officialUrl];
        const load = (urlIndex = 0) => {
            const url = urls[urlIndex];
            if (!url) return Promise.resolve();
            return myAxios.get(url, {
                headers: { cancelerror: true },
                dontalert: true,
            }).then(response => {
                this.setSubDepends(index, response);
            }).catch(() => load(urlIndex + 1));
        };
        return load();
    },
    setSubDepends(index, response) {
        try {
            const manifest = jsyaml.load(response?.data?.data?.manifest);
            (manifest?.platform?.depends || []).forEach(item => {
                this.subDependsList[index][item.identifie] = item.name;
            });
        } catch {
            // Ignore malformed dependency manifests returned by legacy records.
        }
    },
    getDependsList() {
        myAxios.get('/respo/list?limit=999').then(response => {
            const list = response.data?.data?.list || [];
            this.mergeDependsList(this.normalizeDependList(list));
        });
    },
    delDepend(index) {
        if (this.form.dependsIn.length - 1 < index) {
            this.form.depends.splice(index - (this.form.dependsIn.length || 0), 1);
        } else {
            this.form.dependsIn.splice(index, 1);
        }
        this.changeForm();
    },
    openAddDepend() {
        this.dependForm.show = true;
        const identifie = this.json?.platform?.baseInfo?.identifie || this.json?.application?.identifie;
        this.dependForm.identifie_before = identifie.match(/^([^-]+)-(.+)$/)?.[1] || '';
    },
    addDepend() {
        this.$refs.depend.validate(errors => {
            if (errors) return;
            if (!this.dependForm.identifie_before || !this.dependForm.identifie_last) {
                messageWarning('标识请填写完整');
                return;
            }
            const dependency = {
                identifie: this.dependForm.identifie,
                required: this.dependForm.required,
                name: this.dependForm.name,
            };
            const source = this.getSavedManifest
                ? this.getSavedManifest()
                : (typeof this.data == 'string'
                    ? (jsyaml.load(this.data) || {})
                    : JSON.parse(JSON.stringify(this.data || {})));
            source.platform = source.platform || {};
            const dependencies = Array.isArray(source.platform.depends)
                ? source.platform.depends.slice()
                : [];
            if (dependencies.some(item => item?.identifie == dependency.identifie)) {
                messageWarning('该子应用已存在');
                return;
            }
            const nextDependency = { ...dependency, type: 'in' };
            const outIndex = dependencies.findIndex(item => item?.type === 'out');
            dependencies.splice(outIndex < 0 ? dependencies.length : outIndex, 0, nextDependency);
            source.platform.depends = dependencies;
            this.dependForm.show = false;

            const file = `${dependency.identifie}/manifest.yaml`;
            const childOrder = dependencies
                .filter(item => item?.type !== 'out')
                .findIndex(item => item?.identifie == dependency.identifie) + 1;
            const content = jsyaml.dump({
                application: {
                    name: dependency.name,
                    identifie: dependency.identifie,
                    order: childOrder,
                    description: '',
                    author: '',
                },
                platform: {
                    container: { containerPort: 80 },
                },
            });
            const yaml = jsyaml.dump(source);
            this.$emit('addfile', source, yaml, {
                file,
                cont: content,
            });
        });
    },
    addImportedDependencies(dependencies = []) {
        const imported = Array.isArray(dependencies) ? dependencies : [];
        const importedIdentifies = new Set(imported.map(item => item?.identifie).filter(Boolean));
        if (!importedIdentifies.size) return;
        this.form.depends = (this.form.depends || [])
            .filter(item => !importedIdentifies.has(item?.identifie));
        const existing = new Set((this.form.dependsIn || [])
            .map(item => item?.identifie).filter(Boolean));
        imported.forEach(item => {
            if (!item?.identifie || existing.has(item.identifie)) return;
            this.form.dependsIn.push(item);
            existing.add(item.identifie);
        });
        this.syncImportedDependenciesToManifest();
    },
    replaceImportedDependencies(dependencies = []) {
        const replacements = new Map((dependencies || [])
            .filter(item => item?.identifie)
            .map(item => [item.identifie, item]));
        if (!replacements.size) return;
        const replaced = new Set();
        const replace = items => (items || []).map(item => {
            const replacement = replacements.get(item?.identifie);
            if (!replacement) return item;
            replaced.add(item.identifie);
            return { ...item, ...replacement };
        });
        this.form.dependsIn = replace(this.form.dependsIn);
        this.form.depends = replace(this.form.depends);
        replacements.forEach((item, identifie) => {
            if (!replaced.has(identifie)) this.form.dependsIn.push(item);
        });
        this.syncImportedDependenciesToManifest();
    },
    removeImportedDependencies(identifies = []) {
        const importedIdentifies = new Set((identifies || []).filter(Boolean));
        if (!importedIdentifies.size) return;
        this.form.dependsIn = (this.form.dependsIn || [])
            .filter(item => !importedIdentifies.has(item?.identifie));
        this.form.depends = (this.form.depends || [])
            .filter(item => !importedIdentifies.has(item?.identifie));
        this.syncImportedDependenciesToManifest();
    },
    removeChildDependencies(identifies = []) {
        const childIdentifies = new Set((identifies || []).filter(Boolean));
        if (!childIdentifies.size) return;
        this.form.dependsIn = (this.form.dependsIn || [])
            .filter(item => !childIdentifies.has(item?.identifie));
        this.form.depends = (this.form.depends || [])
            .filter(item => item?.type === 'out' || !childIdentifies.has(item?.identifie));
        this.syncImportedDependenciesToManifest();
    },
    syncImportedDependenciesToManifest() {
        this.json.platform = this.json.platform || {};
        this.json.platform.depends = (this.form.dependsIn || []).concat(this.form.depends || []);
        this.setYaml();
    },
    isTraditionFixedDependency(record) {
        return (this.form.type == 'tradition' && isTraditionAppDependency(record))
            || (this.requiresSysboxDependency() && record?.identifie == 'w7panel-sysbox')
            || this.isPluginTraditionDependency(record);
    },
    requiresSysboxDependency() {
        if (this.form.type == 'system-image') return true;
        if (this.form.type != 'tradition') return false;
        return Boolean(this.json.application?.annotation?.[traditionSysboxRootfsAnnotation]);
    },
    syncSysboxDependency() {
        const dependencies = Array.isArray(this.form.depends) ? this.form.depends : [];
        if (!this.requiresSysboxDependency()) {
            if (this.sysboxDependencyManaged) {
                this.form.depends = dependencies.filter(item => item?.identifie != 'w7panel-sysbox');
                this.sysboxDependencyManaged = false;
            }
            return;
        }

        this.sysboxDependencyManaged = true;
        const dependency = {
            identifie: 'w7panel-sysbox',
            name: '微擎sysbox',
            subidentifie: '',
            subname: '',
            required: true,
            type: 'out',
            from: 'https://zpk.w7.cc',
        };
        let inserted = false;
        this.form.depends = dependencies.reduce((result, item) => {
            if (item?.identifie != dependency.identifie) {
                result.push(item);
            } else if (!inserted) {
                result.push({ ...item, ...dependency });
                inserted = true;
            }
            return result;
        }, []);
        if (!inserted) this.form.depends.unshift(dependency);
    },
};
