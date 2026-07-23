<template>
    <a-modal :visible="visible" title="编辑基础信息" :width="920" :footer="false" unmount-on-close
        modal-class="application-base-info-modal" @cancel="close">
        <a-spin :loading="loading || saving" class="application-base-info-spin">
            <a-form :model="form" label-align="left" :label-col-props="{ span: 4 }"
                :wrapper-col-props="{ span: 20 }">
                <a-form-item label="应用图标">
                    <label class="application-icon-upload">
                        <img v-if="iconPreview" :src="iconPreview" alt="" />
                        <span v-else class="application-icon-placeholder">
                            <span class="application-icon-plus">+</span>
                            <span>点击上传</span>
                        </span>
                        <input type="file" accept="image/*" @change="selectIcon" />
                        <span class="application-icon-mask">重新上传</span>
                    </label>
                </a-form-item>
                <a-form-item label="名称" required>
                    <a-input v-model="form.name" placeholder="请输入应用名称" :max-length="100" />
                </a-form-item>
                <a-form-item label="标识">
                    <a-input :model-value="identifie" disabled />
                </a-form-item>
                <a-form-item label="描述">
                    <a-textarea v-model="form.description" placeholder="请输入应用描述" :max-length="500"
                        show-word-limit :auto-size="{ minRows: 2, maxRows: 5 }" />
                </a-form-item>
                <a-form-item label="标签">
                    <a-select v-model="form.tagIds" multiple allow-search placeholder="请选择标签">
                        <a-option v-for="item in tags" :key="item.id" :value="item.id" :label="item.name" />
                    </a-select>
                </a-form-item>
                <a-form-item label="注解" class="application-annotation-item">
                    <manifest-config-table :rows="form.annotations" add-text="添加注解"
                        @add="form.annotations.push({ key: '', value: '' })">
                        <template #columns>
                            <manifest-config-table-column data-index="key" title="键" width="220px">
                                <template #cell="{ record }">
                                    <a-input v-model="record.key" placeholder="请输入键" />
                                </template>
                            </manifest-config-table-column>
                            <manifest-config-table-column data-index="value" title="值">
                                <template #cell="{ record }">
                                    <a-textarea v-model="record.value" placeholder="请输入值" auto-size />
                                </template>
                            </manifest-config-table-column>
                            <manifest-config-table-column title="操作" width="80px">
                                <template #cell="{ index }">
                                    <span class="c-blue cursor" @click="form.annotations.splice(index, 1)">删除</span>
                                </template>
                            </manifest-config-table-column>
                        </template>
                    </manifest-config-table>
                </a-form-item>
                <a-form-item label="属性">
                    <div class="application-property-list">
                        <a-checkbox v-model="form.once">仅安装一次</a-checkbox>
                        <a-checkbox v-model="form.clusterPrivileges">集群特权</a-checkbox>
                        <a-checkbox v-model="form.registerSite">创建站点</a-checkbox>
                    </div>
                </a-form-item>
                <a-form-item label="应用介绍" class="application-introduction-item">
                    <mavon-editor v-model="form.introduction" class="application-introduction-editor"
                        :subfield="false" :toolbarsFlag="true" defaultOpen="edit" previewBackground="#ffffff"
                        boxShadowStyle="" :xssOptions="false" placeholder="请输入应用介绍"
                        :toolbars="editorToolbars" />
                </a-form-item>
            </a-form>
            <div class="dialog-footer application-base-info-footer">
                <a-button :disabled="saving" @click="close">取消</a-button>
                <a-button type="primary" :loading="saving" :disabled="loading || loadFailed"
                    @click="submit">保存</a-button>
            </div>
        </a-spin>
    </a-modal>
</template>

<script>
import myAxios from '@/utils';
import ManifestConfigTable from '@/components/manifest-config-table.vue';
import ManifestConfigTableColumn from '@/components/manifest-config-table-column.vue';
import { messageError, messageSuccess } from '@/utils/ui-feedback';

export default {
    name: 'ApplicationBaseInfoModal',
    components: {
        ManifestConfigTable,
        ManifestConfigTableColumn,
    },
    props: {
        visible: {
            type: Boolean,
            default: false,
        },
        identifie: {
            type: String,
            default: '',
        },
        info: {
            type: Object,
            default: () => ({}),
        },
        iconCacheKey: {
            type: [String, Number],
            default: '',
        },
    },
    emits: ['update:visible', 'saved'],
    data() {
        return {
            loading: false,
            loadFailed: false,
            saving: false,
            selectedIcon: null,
            objectIconUrl: '',
            iconPreview: '',
            tags: [],
            initialTagIds: [],
            form: {
                name: '',
                description: '',
                introduction: '',
                tagIds: [],
                annotations: [],
                once: false,
                clusterPrivileges: false,
                registerSite: false,
            },
            editorToolbars: {
                bold: true,
                italic: true,
                header: true,
                underline: true,
                strikethrough: true,
                mark: true,
                superscript: true,
                subscript: true,
                quote: true,
                ol: true,
                ul: true,
                link: true,
                imagelink: false,
                code: false,
                table: true,
                undo: true,
                redo: true,
                trash: false,
                save: false,
                alignleft: true,
                aligncenter: true,
                alignright: true,
                navigation: false,
                subfield: false,
                fullscreen: false,
                readmodel: false,
                htmlcode: false,
                help: false,
                preview: true,
            },
        };
    },
    watch: {
        visible(show) {
            if (show) {
                this.load();
            } else {
                this.resetSelectedIcon();
            }
        },
    },
    beforeUnmount() {
        this.resetSelectedIcon();
    },
    methods: {
        getIconUrl(icon = '') {
            if (!icon) { return '' }
            let url = /^https?:\/\//.test(icon)
                ? icon
                : (window?.$wujie?.props?.url || '') + icon;
            if (this.iconCacheKey) {
                url += (url.includes('?') ? '&' : '?') + 'time=' + this.iconCacheKey;
            }
            return url;
        },
        async load() {
            this.loading = true;
            this.loadFailed = false;
            this.resetSelectedIcon();
            try {
                let [infoRes, settingRes, filesRes, tagsRes] = await Promise.all([
                    myAxios.get('/respo/info/' + this.identifie),
                    myAxios.post('/respo/setting/get', { identifie: this.identifie }),
                    myAxios.post('/respo/share-file/path-tree', { identifie: this.identifie }),
                    myAxios.post('/respo/tag/list', { limit: 999 }),
                ]);
                let latestInfo = infoRes?.data?.data || this.info || {};
                let baseInfo = settingRes?.data?.data?.base_info || {};
                this.form.name = baseInfo.name || '';
                this.form.description = baseInfo.description || '';
                this.form.annotations = Object.entries(baseInfo.annotation || {})
                    .map(([key, value]) => ({ key, value: String(value) }));
                this.form.once = Boolean(baseInfo.once);
                this.form.clusterPrivileges = Boolean(baseInfo.cluster_privileges);
                this.form.registerSite = Boolean(baseInfo.register_site);
                this.iconPreview = this.getIconUrl(latestInfo?.icon_url || this.info?.icon_url || '');

                this.form.introduction = filesRes?.data?.data?.list?.['readme.md'] || '';
                this.tags = tagsRes?.data?.data?.list || [];
                this.initialTagIds = (latestInfo?.tags || this.info?.tags || []).map(item => Number(item.id));
                this.form.tagIds = [...this.initialTagIds];
            } catch (error) {
                this.loadFailed = true;
                messageError(error?.response?.data?.error || '基础信息加载失败');
            } finally {
                this.loading = false;
            }
        },
        selectIcon(event) {
            let file = event?.target?.files?.[0];
            if (!file) { return }
            this.resetSelectedIcon();
            this.selectedIcon = file;
            this.objectIconUrl = URL.createObjectURL(file);
            this.iconPreview = this.objectIconUrl;
            event.target.value = '';
        },
        resetSelectedIcon() {
            if (this.objectIconUrl) {
                URL.revokeObjectURL(this.objectIconUrl);
            }
            this.objectIconUrl = '';
            this.selectedIcon = null;
        },
        close() {
            if (this.saving) { return }
            this.$emit('update:visible', false);
        },
        async submit() {
            if (this.loading || this.loadFailed) {
                messageError('基础信息尚未加载完成，请关闭后重试');
                return;
            }
            let name = (this.form.name || '').trim();
            if (!name) {
                messageError('请输入应用名称');
                return;
            }

            this.saving = true;
            try {
                let latestRes = await myAxios.get('/respo/info/' + this.identifie);
                let latestInfo = latestRes?.data?.data || this.info || {};

                let annotation = this.form.annotations.reduce((result, item) => {
                    let key = (item?.key || '').trim();
                    if (key && item?.value !== undefined && item?.value !== null && String(item.value) !== '') {
                        result[key] = String(item.value);
                    }
                    return result;
                }, {});

                await myAxios.post('/respo/setting/set', {
                    identifie: this.identifie,
                    base_info: {
                        name,
                        description: this.form.description || '',
                        annotation,
                        once: Boolean(this.form.once),
                        cluster_privileges: Boolean(this.form.clusterPrivileges),
                        register_site: Boolean(this.form.registerSite),
                    },
                });
                await myAxios.post('/respo/share-file/file', {
                    identifie: this.identifie,
                    filename: 'readme.md',
                    content: this.form.introduction || '',
                });

                let nextTagIds = this.form.tagIds.map(id => Number(id));
                let deletedTagIds = this.initialTagIds.filter(id => !nextTagIds.includes(id));
                let addedTagIds = nextTagIds.filter(id => !this.initialTagIds.includes(id));
                let formulaId = latestInfo?.version?.formula_id || this.info?.version?.formula_id;
                for (let tagId of deletedTagIds) {
                    await myAxios.post('/respo/tag/delete', { tagId, formulaId });
                }
                for (let tagId of addedTagIds) {
                    let tag = this.tags.find(item => Number(item.id) === tagId);
                    if (tag?.name) {
                        await myAxios.post('/respo/tag/add', { identifie: this.identifie, name: tag.name });
                    }
                }

                let iconUpdated = false;
                if (this.selectedIcon) {
                    let formData = new FormData();
                    formData.append('identifie', this.identifie);
                    formData.append('file', this.selectedIcon);
                    await myAxios.post('/respo/icon', formData);
                    iconUpdated = true;
                }

                messageSuccess('保存成功');
                this.$emit('saved', { iconUpdated });
                this.$emit('update:visible', false);
            } catch (error) {
                messageError(error?.response?.data?.error || '保存失败');
            } finally {
                this.saving = false;
            }
        },
    },
};
</script>

<style scoped>
.application-base-info-spin {
    display: block;
}

.application-icon-upload {
    position: relative;
    display: block;
    width: 80px;
    height: 80px;
    overflow: hidden;
    border: 1px solid #e5e6eb;
    border-radius: 8px;
    cursor: pointer;
}

.application-icon-upload img {
    width: 100%;
    height: 100%;
    object-fit: cover;
}

.application-icon-upload input {
    display: none;
}

.application-icon-placeholder {
    display: flex;
    width: 100%;
    height: 100%;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 6px;
    color: #999;
    font-size: 12px;
}

.application-icon-plus {
    color: #666;
    font-size: 24px;
    line-height: 1;
}

.application-icon-mask {
    position: absolute;
    right: 0;
    bottom: 0;
    left: 0;
    display: none;
    padding: 5px 0;
    color: #fff;
    background: rgba(0, 0, 0, .55);
    font-size: 12px;
    line-height: 1;
    text-align: center;
}

.application-icon-upload:hover .application-icon-mask {
    display: block;
}

.application-introduction-item :deep(.arco-form-item-content-flex) {
    display: block;
}

.application-annotation-item :deep(.arco-form-item-content-flex) {
    display: block;
}

.application-property-list {
    display: flex;
    align-items: center;
    gap: 24px;
    flex-wrap: wrap;
}

.application-introduction-editor {
    width: 100%;
    min-height: 280px;
}

.application-base-info-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    padding-top: 8px;
}
</style>

<style>
.application-base-info-modal .v-note-wrapper {
    min-height: 280px;
    z-index: 1;
}

.application-base-info-modal .v-note-op {
    overflow: visible;
}

.application-base-info-modal .v-note-op .v-left-item {
    flex: 1 1 auto;
    min-width: 0;
    white-space: nowrap;
}

.application-base-info-modal .v-note-op .v-right-item {
    flex: 0 0 34px;
    width: 34px;
    max-width: 34px;
    padding-right: 0;
    white-space: nowrap;
}

.application-base-info-modal {
    max-width: calc(100vw - 48px);
}

.application-base-info-modal .arco-modal-body {
    max-height: calc(100vh - 120px);
}
</style>
