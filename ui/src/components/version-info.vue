<template>
    <div class="version-info-layout">
        <div class="app-icon">
            <img v-if="logoimg" :src="logoimg" class="img df-s0" alt="" />
            <div v-else class="df df-c ai-c jc-c"
                style="width:100%;height:100%;border:1px solid #e7e7e7;border-radius:4px;">
                <span class="c-99 fs-12 lh-1">暂无图标</span>
            </div>
        </div>
        <a-form class="fc version-info-form" :model="form" label-align="left"
            :label-col-props="{ span: 5, flex: '0 0 56px' }"
            :wrapper-col-props="{ span: 19, flex: '1' }" @keydown.enter.prevent>
            <a-form-item label="名称" style="margin-bottom:10px;">
                <span>{{ form.name || '-' }}</span>
            </a-form-item>
            <a-form-item label="标识" style="margin-bottom:10px;">
                <span>{{ identifie }}</span>
            </a-form-item>
            <a-form-item label="描述" style="margin-bottom:10px;">
                <span>{{ form.description || '-' }}</span>
            </a-form-item>
            <a-form-item label="标签" style="margin-bottom:10px;">

                <div class="df df-ww">
                    <a-tag color="blue" v-for="(item, index) in form.tags" :key="item.id || item.name"
                        :visible="true"
                        :closable="edit.type == 'tags' && item.name != requiredTagName"
                        @close="deleteTag(index)" class="tag">{{ item.name }}</a-tag>
                    <div v-if="edit.type == 'tags'" class="df">
                        <a-select v-model="form.taginput" multiple :max-tag-count="1" style="width:140px;"
                            placeholder="添加新标签">
                            <a-option v-for="item in editableTags" :disabled="Boolean(form.tags.find(i => i.name == item.name))"
                                :key="item.id" :label="item.name" :value="item.id"></a-option>
                        </a-select>
                        <a-button type="primary" @click="addTag" style="margin-left:10px;">确定</a-button>
                        <a-button @click="edit.type = ''">取消</a-button>
                    </div>
                </div>

                <span v-if="edit.type != 'tags' && (!form.tags || !form.tags.length)">-</span>
                <a-tooltip v-if="edit.type != 'tags'" content="修改">
                    <a-button class="editbtn" type="text" shape="circle" size="mini" @click="edit.type = 'tags'">
                        <template #icon><icon-edit /></template>
                    </a-button>
                </a-tooltip>
            </a-form-item>

            <a-form-item label="注解" style="margin-bottom:10px;">
                <span style="word-break: break-all;">
                    {{ annotationKeys }}
                </span>
                <a-tooltip content="编辑">
                    <a-button class="editbtn" type="text" shape="circle" size="mini"
                        @click="openAnnotationEdit">
                        <template #icon><icon-edit /></template>
                    </a-button>
                </a-tooltip>
            </a-form-item>
            <a-form-item label="属性">
                <div class="version-info-checks">
                    <a-checkbox v-model="edit.once" :disabled="isInstallOnlyOnceType"
                        @change="changeForm('once')">仅安装一次</a-checkbox>
                    <a-checkbox v-model="edit.clusterPrivileges"
                        @change="changeForm('clusterPrivileges')">集群特权</a-checkbox>
                    <a-checkbox v-model="edit.registerSite" :disabled="isRegisterSiteDisabled"
                        @change="changeForm('registerSite')">创建站点</a-checkbox>
                    <a-checkbox v-model="edit.officialApp" @change="changeForm('officialApp')">官方应用</a-checkbox>
                    <a-checkbox v-model="edit.denyDelete" @change="changeForm('denyDelete')">禁止卸载</a-checkbox>
                </div>
            </a-form-item>
        </a-form>


        <a-modal v-model:visible="annotationEdit.show" title="注解" :width="1000" :footer="false">
            <manifest-config-table :rows="annotationEdit.list" add-text="添加注解"
                @add="annotationEdit.list.push({ key: '', value: '' })">
                <template #columns>
                    <manifest-config-table-column data-index="key" title="键">
                        <template #cell="{ record }">
                            <a-input v-model="record.key" placeholder="请输入" style="width:200px;" />
                        </template>
                    </manifest-config-table-column>
                    <manifest-config-table-column data-index="value" title="值">
                        <template #cell="{ record }">
                            <a-textarea v-model="record.value" placeholder="请输入" auto-size style="width: 600px;" />
                        </template>
                    </manifest-config-table-column>
                    <manifest-config-table-column title="操作">
                        <template #cell="{ index }">
                            <span class="c-blue cursor" @click="annotationEdit.list.splice(index, 1)">删除</span>
                        </template>
                    </manifest-config-table-column>
                </template>
            </manifest-config-table>
            <div class="dialog-footer df jc-c mt-20">
                <a-button @click="annotationEdit.show = false;">取消</a-button>
                <a-button @click="submitAnnotation" type="primary">确定</a-button>
            </div>
        </a-modal>
    </div>
</template>
<script>
import jsyaml from "js-yaml";
import myAxios from '@/utils';
import ManifestConfigTable from '@/components/manifest-config-table.vue';
import ManifestConfigTableColumn from '@/components/manifest-config-table-column.vue';
import { messageSuccess } from '@/utils/ui-feedback';
import { IconEdit } from '@arco-design/web-vue/es/icon';

const defaultManifest = `application:
    name: ''
    identifie: ''
    description: ''
    author: ''
    once: false
v: 2
`

const propertyAnnotationKeys = {
    officialApp: 'w7.cc/official-app',
    denyDelete: 'w7.cc/deny-delete',
};

export default {
    components: {
        ManifestConfigTable,
        ManifestConfigTableColumn,
        IconEdit,
    },
    props: ['identifie', 'info', 'iconCacheKey'],
    data() {
        return {
            baseurl: '',
            logoimg: '',
            json: null,

            form: {
                name: '',
                tags: [],
                once: false,
                clusterPrivileges: false,
                registerSite: false,
                officialApp: false,
                denyDelete: false,
                description: '',
                taginput: [],
            },
            edit: {
                type: '',
                once: false,
                clusterPrivileges: false,
                registerSite: false,
                officialApp: false,
                denyDelete: false,
            },
            annotationEdit: {
                show: false,
                list: [],
            },
            tags: [],
            ensuringRequiredTags: {},
        }
    },
    created() {
        this.baseurl = window?.$wujie?.props?.url || '';
        this.getTag();
        this.init();
    },
    watch: {
        identifie() {
            this.getTag();
        },
        info() {
            this.init();
        },
        iconCacheKey() {
            this.init();
        },
    },
    methods: {
        init() {
            if (!this.info) { return }
            let manifest = this.info?.manifest || defaultManifest;
            this.json = jsyaml.load(manifest);
            this.json.application.identifie = this.identifie;
            this.annotationEdit.list = this.json.application?.annotation || [];
            const annotation = this.json.application?.annotation || {};

            if (this.json.application) {
                this.form.once = this.isInstallOnlyOnceType ? true : (this.json.application?.once || false);
                this.form.name = this.json.application?.name || '';
                this.form.description = this.json?.application?.description || '';
                this.form.clusterPrivileges = this.json?.application?.clusterPrivileges || false;
                this.form.registerSite = this.isRegisterSiteDisabled ? false : (this.json?.application?.registerSite || false);
                this.form.officialApp = String(annotation[propertyAnnotationKeys.officialApp]).toLowerCase() == 'true';
                this.form.denyDelete = String(annotation[propertyAnnotationKeys.denyDelete]).toLowerCase() == 'true';
            }
            this.edit.once = this.isInstallOnlyOnceType ? true : (this.json?.application?.once || false);
            this.edit.clusterPrivileges = this.json?.application?.clusterPrivileges || false;
            this.edit.registerSite = this.isRegisterSiteDisabled ? false : (this.json?.application?.registerSite || false);
            this.edit.officialApp = this.form.officialApp;
            this.edit.denyDelete = this.form.denyDelete;

            this.logoimg = this.info?.icon_url;
            if (this.logoimg && !/^https?:\/\//.test(this.logoimg)) {
                this.logoimg = this.baseurl + this.logoimg;
            }
            if (this.logoimg && this.iconCacheKey) {
                this.logoimg += (this.logoimg.includes('?') ? '&' : '?') + 'time=' + this.iconCacheKey;
            }
            let tags = [...(this.info?.tags || [])];
            if (this.requiredTagName) {
                let requiredTagIndex = tags.findIndex(item => item.name == this.requiredTagName);
                let requiredTag = requiredTagIndex >= 0
                    ? tags.splice(requiredTagIndex, 1)[0]
                    : { name: this.requiredTagName };
                tags.unshift(requiredTag);
                if (requiredTagIndex < 0) {
                    this.ensureRequiredTag(this.requiredTagName);
                }
            }
            this.form.tags = tags;
        },

        ensureRequiredTag(name) {
            if (!name || !this.identifie || this.ensuringRequiredTags[name]) { return }
            this.ensuringRequiredTags[name] = true;
            myAxios.post('/respo/tag/add', {
                identifie: this.identifie,
                name,
            }).then(() => {
                this.$emit('refresh');
            }).catch(() => undefined).finally(() => {
                delete this.ensuringRequiredTags[name];
            });
        },

        async getTag() {
            if (!this.identifie) { return }

            await myAxios.post('/respo/tag/list', { limit: 999 }).then(res => {
                this.tags = res.data?.data?.list || [];
            }).catch(() => { })

        },

        deleteTag(index) {
            let tag = this.form.tags[index];
            if (!tag?.id || tag.name == this.requiredTagName) { return }
            let formulaId = this.info?.version?.formula_id;
            if (!formulaId) { return }
            myAxios.post('/respo/tag/delete', {
                tagId: tag.id,
                formulaId: formulaId,
            }).then(() => {
                messageSuccess('删除成功');
                let deletedIndex = this.form.tags.findIndex(item => item.id == tag.id);
                if (deletedIndex >= 0) {
                    this.form.tags.splice(deletedIndex, 1);
                }
                this.edit.type = '';
                this.$emit('refresh');
            })
        },

        async addTag() {
            if (!this.form.taginput?.length) { this.edit.type = ''; return }
            let ids = this.form.taginput;
            let names = ids?.map(id => {
                return this.tags.find(i => i.id == Number(id))?.name
            });
            for (let i in names) {
                let name = names[i];
                await myAxios.post('/respo/tag/add', {
                    identifie: this.identifie,
                    name: name,
                }).then(res => {
                    messageSuccess('添加成功');
                    this.form.tags.push({ name: name, id: res.data.id });
                    this.form.taginput = [];
                    this.edit.type = '';
                }).catch(() => { });
            }
            this.$emit('refresh');
        },
        openAnnotationEdit() {
            let list = Object.entries(this.json?.application?.annotation || {})
                .filter(([key]) => !Object.values(propertyAnnotationKeys).includes(key))
                .map(([k, v]) => ({ key: k, value: v }))
            this.annotationEdit = {
                show: true,
                list: list,
            }
        },
        submitAnnotation() {
            let currentAnnotation = this.json?.application?.annotation || {};
            let obj = Object.values(propertyAnnotationKeys).reduce((result, key) => {
                if (currentAnnotation[key] !== undefined) {
                    result[key] = currentAnnotation[key];
                }
                return result;
            }, {});
            this.annotationEdit.list.filter(i => i.key && i.value).map(i => {
                obj[i.key] = String(i.value);
            })
            this.json.application.annotation = obj;
            this.annotationEdit.show = false;
            this.submit();
        },
        changeForm(type) {
            if (type == 'once' && this.isInstallOnlyOnceType) {
                this.edit.once = true;
                return;
            }
            if (type == 'registerSite' && this.isRegisterSiteDisabled) {
                this.edit.registerSite = false;
                return;
            }
            if (this.edit[type] == this.form[type]) {
                this.edit.type = '';
                return;
            }
            if (propertyAnnotationKeys[type]) {
                this.json.application.annotation = this.json.application.annotation || {};
                if (this.edit[type]) {
                    this.json.application.annotation[propertyAnnotationKeys[type]] = 'true';
                } else {
                    delete this.json.application.annotation[propertyAnnotationKeys[type]];
                }
                this.edit.type = '';
                this.submit();
                return;
            }
            this.json.application[type] = this.edit[type];
            this.edit.type = '';
            this.submit();
        },
        async submit() {
            let settingRes = await myAxios.post('/respo/setting/get', {
                identifie: this.identifie,
            });
            let baseInfo = settingRes?.data?.data?.base_info || {};
            let annotation = {
                ...(this.json?.application?.annotation || {}),
            };
            await myAxios.post('/respo/setting/set', {
                identifie: this.identifie,
                base_info: {
                    ...baseInfo,
                    name: this.json?.application?.name || this.form.name || '',
                    description: this.json?.application?.description || this.form.description || '',
                    annotation,
                    once: this.isInstallOnlyOnceType ? true : Boolean(this.json?.application?.once),
                    cluster_privileges: Boolean(this.json?.application?.clusterPrivileges),
                    register_site: this.isRegisterSiteDisabled ? false : Boolean(this.json?.application?.registerSite),
                },
            }).then(() => {
                messageSuccess('操作成功');
                this.$emit('refresh');
            })
        },
    },
    computed: {
        applicationType() {
            return this.json?.application?.type || '';
        },
        isInstallOnlyOnceType() {
            return ['environment', 'gateway-plugin'].includes(this.applicationType);
        },
        isRegisterSiteDisabled() {
            return ['environment', 'gateway-plugin'].includes(this.applicationType);
        },
        requiredTagName() {
            if (this.applicationType == 'environment') { return '运行环境' }
            if (this.applicationType == 'gateway-plugin') { return '网关插件' }
            return '';
        },
        editableTags() {
            return this.tags.filter(item => item.name != this.requiredTagName);
        },
        annotationKeys() {
            const keys = Object.keys(this.json?.application?.annotation || [])
                .filter(key => !Object.values(propertyAnnotationKeys).includes(key));
            return keys.length === 0 ? '-' : keys.join(',');
        },
    },
}
</script>
<style scoped>
.version-info-layout {
    display: flex;
    align-items: flex-start;
    gap: 32px;
    min-width: 0;
}

.app-icon {
    width: 72px;
    height: 72px;
    flex: 0 0 72px;
    position: relative;
    border-radius: 8px;
}

.app-icon .img {
    width: 72px;
    height: 72px;
    display: block;
    border-radius: 8px;
}

.version-info-form {
    min-width: 0;
}

.version-info-form :deep(.arco-form-item) {
    margin-bottom: 14px !important;
}

.version-info-form :deep(.arco-form-item-content) {
    min-width: 0;
}

.version-info-checks {
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.tag {
    height: 24px;
    margin-right: 6px;
    margin-bottom: 3px;
    margin-top: 3px;
}

.tag-cpn {
    width: 100%;
    box-sizing: border-box;
    border-radius: 4px;
    padding: 7px 7px 1px;
    border: 1px solid #dcdfe6;
}

.tag-cpn.active {
    border-color: #409eff;
}

.tag-cpn .tag {
    height: 24px;
    margin-right: 6px;
    margin-bottom: 6px;
    margin-top: 0;
}

.tag-cpn .input {
    width: -webkit-min-content;
    width: min-content;
    height: 24px;
    min-width: 60px;
    margin-bottom: 6px;
}

.tag-cpn .input input {
    width: 100%;
    height: 100%;
    display: block;
    border: 0;
    outline: 0;
    padding: 0 6px;
}

.tag-cpn .input input::placeholder {
    color: #999;
}

.table {
    width: 100%;
}

.table td {
    padding: 10px;
    line-height: 1.4;
    border: 1px solid #cccccc;
    border-left: 0;
    border-right: 0;
    background: #F0F3FA;
}

.table tr:last-child td {
    background: transparent;
}

.table.nolast tr:last-child td {
    background: #f0f3fa;
}

.table thead tr:first-child td {
    background: #f3f3f3;
    border-top: 0;
}
</style>
<style>
.version-info-form .arco-form-item .editbtn {
    display: none;
    margin-left: 8px;
    color: #3370ff;
    vertical-align: middle;
    width: 24px;
    height: 24px;
    font-size: 16px;
}

.version-info-form .arco-form-item:hover .editbtn {
    display: inline-flex;
}

.version-info-form .inline-icon-action {
    margin-left: 8px;
    color: #3370ff;
    vertical-align: middle;
    width: 24px;
    height: 24px;
    font-size: 16px;
}

.version-info-form .editbtn .arco-icon,
.version-info-form .inline-icon-action .arco-icon {
    font-size: 16px;
}
</style>
