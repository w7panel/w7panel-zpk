<template>
    <div class="bg-white" style="min-height:100%;">
        <div class="com-back registry-detail-breadcrumb df ai-c">
            <span class="backbtn df ai-c" @click="$router.go(-1)">
                <icon-arrow-left class="backicon" />
            </span>
            <a-breadcrumb>
                <a-breadcrumb-item>
                    <router-link to="/zpk-registry">镜像仓库</router-link>
                </a-breadcrumb-item>
                <a-breadcrumb-item>
                    <span>{{ data.namespace }}/{{ data.name }}</span>
                </a-breadcrumb-item>
            </a-breadcrumb>
        </div>
        <div class="bg-white" style="padding: 0 24px 6px;">
            <a-tabs v-model:active-key="tabsActive">
                <a-tab-pane key="version" title="版本管理">
                    <div class="zpk-page-toolbar registry-version-toolbar">
                        <div class="zpk-toolbar-left">
                            <a-popconfirm :content="`确认删除选中的 ${selectedTagKeys.length} 个版本吗？`" type="warning" position="bl"
                                ok-text="确定" cancel-text="取消" content-class="zpk-delete-popconfirm"
                                :ok-button-props="{ status: 'danger', loading: batchDeleting }"
                                :cancel-button-props="{ type: 'secondary' }" @ok="batchDelTags">
                                <a-button status="danger" :disabled="selectedTagKeys.length === 0" :loading="batchDeleting">
                                    批量删除
                                </a-button>
                            </a-popconfirm>
                            <span v-if="selectedTagKeys.length" class="registry-version-selected">
                                已选择 {{ selectedTagKeys.length }} 项
                            </span>
                        </div>
                    </div>
                    <a-table :loading="loading || batchDeleting" :data="tags" style="width: 100%" class="table-header"
                        :pagination="false" row-key="TagName" v-model:selected-keys="selectedTagKeys"
                        :row-selection="{ type: 'checkbox', showCheckedAll: true, onlyCurrent: true }">
                        <template #columns>
                            <a-table-column title="镜像版本">
                                <template #cell="{ record }">
                                    <a-popover position="bl" content-class="registry-version-popover">
                                        <span class="c-blue cursor">{{ record.TagName }}</span>
                                        <a-tag v-if="record.Rebuilt" color="purple" size="small" class="ml-10">重构</a-tag>
                                        <template #content>
                                            <a-form :model="record" label-align="left" :label-col-props="{ span: 6, flex: '0 0 160px' }"
                                                :wrapper-col-props="{ span: 18, flex: '1' }" class="registry-detail-form">
                                                <a-form-item label="镜像ID(SHA256)" style="margin-bottom:0;">
                                                    <span class="registry-version-digest-line">
                                                        <span class="registry-version-digest">{{ record.Digest }}</span>
                                                        <a-tooltip content="复制">
                                                            <a-button class="icon-action" type="text" shape="circle" size="mini"
                                                                @click="onekeyCopy(record.Digest)">
                                                                <template #icon><icon-copy /></template>
                                                            </a-button>
                                                        </a-tooltip>
                                                    </span>
                                                </a-form-item>
                                                <a-form-item label="平台" style="margin-bottom:0;">{{ record.Platform }}</a-form-item>
                                                <a-form-item label="制品类型" style="margin-bottom:0;">{{ record.Type }}</a-form-item>
                                            </a-form>
                                        </template>
                                    </a-popover>
                                    <a-tooltip content="复制">
                                        <a-button class="icon-action registry-tag-copy" type="text" shape="circle" size="mini"
                                            style="color:#333333;"
                                            @click="onekeyCopy(record.TagName)">
                                            <template #icon><icon-copy /></template>
                                        </a-button>
                                    </a-tooltip>
                                </template>
                            </a-table-column>

                            <a-table-column title="大小" data-index="size" />

                            <a-table-column title="创建时间" data-index="CreatedAt" :width="200">
                                <template #cell="{ record }">
                                    {{ record.CreatedAt ? new Date(record.CreatedAt).toLocaleString() : '未知' }}
                                </template>
                            </a-table-column>
                            <a-table-column title="修改时间" data-index="CreatedAt" :width="200">
                                <template #cell="{ record }">
                                    {{ record.UpdatedAt ? new Date(record.UpdatedAt).toLocaleString() : '未知' }}
                                </template>
                            </a-table-column>
                            <a-table-column title="操作">
                                <template #cell="{ record }">
                                    <a-popconfirm content="确认要删除吗" type="warning" ok-text="确定"
                                        cancel-text="取消" content-class="zpk-delete-popconfirm"
                                        :ok-button-props="{ status: 'danger' }"
                                        :cancel-button-props="{ type: 'secondary' }" @ok="del(record)">
                                        <a-button type="text">删除</a-button>
                                    </a-popconfirm>
                                </template>
                            </a-table-column>
                        </template>
                    </a-table>
                    <div class="df jc-e mt-20">
                        <a-pagination v-model:current="tagPage.page" v-model:page-size="tagPage.pageSize"
                            :total="tagPage.total" :page-size-options="[10, 20, 30, 40, 50]" show-page-size
                            @page-size-change="handleTagPageSizeChange" @change="getVersion" />
                    </div>
                </a-tab-pane>
                <a-tab-pane key="info" title="仓库信息">
                    <a-spin :loading="loading" class="registry-info-spin">
                        <a-form :model="data" label-align="left" :label-col-props="{ flex: '0 0 96px' }"
                            :wrapper-col-props="{ flex: '1' }" class="registry-detail-form registry-info-form mt-24">
                            <a-form-item label="仓库名称">{{ data.namespace }}/{{ data.name }}</a-form-item>
                            <a-form-item label="仓库地址">
                                {{ data.registry }}/{{ data.namespace }}/{{ data.name }}
                                <a-tooltip content="复制">
                                    <a-button class="icon-action" type="text" shape="circle" size="mini"
                                        @click="onekeyCopy(`${data.registry}/${data.namespace}/${data.name}`)">
                                        <template #icon><icon-copy /></template>
                                    </a-button>
                                </a-tooltip>
                            </a-form-item>
                            <a-form-item label="命名空间">{{ data.namespace }}</a-form-item>
                            <a-form-item label="公共权限">
                                {{ data.visible_type === 3 ? namespaceTypeMap[data.namespace] :
                                    visibleTypeMap[data.visible_type]}}
                                <a-tooltip v-if="hasAccess(data.user_id)" content="编辑">
                                    <a-button class="icon-action" type="text" shape="circle" size="mini"
                                        @click="edit('visible_type')">
                                        <template #icon><icon-edit /></template>
                                    </a-button>
                                </a-tooltip>
                            </a-form-item>
                            <a-form-item label="描述">
                                {{ data.desc }}
                                <a-tooltip v-if="hasAccess(data.user_id)" content="编辑">
                                    <a-button class="icon-action" type="text" shape="circle" size="mini"
                                        @click="edit('desc')">
                                        <template #icon><icon-edit /></template>
                                    </a-button>
                                </a-tooltip>
                            </a-form-item>
                            <a-form-item label="创建时间">{{ data.created_at ? new Date(data.created_at).toLocaleString() : '' }}</a-form-item>
                        </a-form>
                    </a-spin>
                </a-tab-pane>
                <a-tab-pane key="build" title="镜像自动任务" v-if="hasAccess(data.user_id)">
                    <image-tasks v-if="data.id" :key="data.id" :repository-id="Number(data.id)" :user-info="userInfo" />
                </a-tab-pane>
            </a-tabs>
        </div>

        <a-modal v-model:visible="visible" title="编辑镜像" :width="500" :footer="false">
            <a-form ref="form" :model="form" label-align="left" :label-col-props="{ span: 4, flex: '0 0 80px' }"
                :wrapper-col-props="{ span: 20, flex: '1' }" class="registry-detail-form">
                <a-form-item label="描述" field="desc" v-if="editProp === 'desc'">
                    <a-textarea v-model="form.desc" :auto-size="{ minRows: 5, maxRows: 5 }" />
                </a-form-item>
                <a-form-item label="公共权限" field="visible_type" v-if="editProp === 'visible_type'">
                    <div class="df df-c" style="flex:1;">
                        <a-checkbox v-model="formVisibleType3" @change="v => form.visible_type = v ? 3 : 1">跟随命名空间</a-checkbox>
                        <a-radio-group v-model="form.visible_type" :disabled="formVisibleType3">
                            <a-radio :value="1">私有读写</a-radio>
                            <a-radio :value="4">公有读私有写</a-radio>
                            <a-radio :value="2">公有读写</a-radio>
                        </a-radio-group>
                    </div>
                </a-form-item>
            </a-form>
            <div class="dialog-footer">
                <a-button size="large" @click="visible = false">取消</a-button>
                <a-button type="primary" size="large" @click="onSubmit">确定</a-button>
            </div>
        </a-modal>


    </div>
</template>

<script>
import myAxios from "@/utils";
import ImageTasks from '@/components/workflow/image-tasks.vue';
import userMixin from "@/utils/user-mixin";
import { messageError, messageSuccess } from "@/utils/ui-feedback";
import { IconArrowLeft, IconCopy, IconEdit } from '@arco-design/web-vue/es/icon';

export default {
    components: {
        ImageTasks,
        IconArrowLeft,
        IconCopy,
        IconEdit,
    },
    data() {
        return {
            visibleTypeMap: {
                1: '私有读写',
                2: '公有读写',
                4: '公有读私有写'
            },
            editProp: '',
            visible: false,
            form: {
                name: '',
                namespace: '',
                desc: '',
                visible_type: 3
            },
            formVisibleType3: false,
            namespaceTypeMap: {},
            id: null,
            loading: true,
            data: {},
            tags: [],
            selectedTagKeys: [],
            batchDeleting: false,
            tagPage: {
                page: 1,
                pageSize: 10,
                total: 0,
            },

            tabsActive: 'version',






        }
    },
    mixins: [userMixin],
    created() {
        if (this.$route.params && this.$route.params.id) {
            this.id = parseInt(this.$route.params.id);
            this.getInfo();
            this.getData();
            this.getNamespace();
        }
    },
    watch: {
        tabsActive() {
            this.getData();
        },
    },
    methods: {

        edit(prop) {
            this.editProp = prop;
            this.visible = true
        },
        getNamespace() {
            myAxios.post("/v2/api/namespace/list").then(res => {
                let data = res.data?.data?.list ?? [];
                let map = {}
                data.forEach(i => {
                    map[i.name] = this.visibleTypeMap[i.visible_type]
                })
                this.namespaceTypeMap = map
            }).catch(() => { });
        },
        del(tag) {
            return myAxios.post("/v2/api/repository/tags/del", {
                id: parseInt(this.$route.params.id),
                tag: tag.TagName
            }).then(() => {
                messageSuccess("删除成功");
                this.reloadTagsAfterDelete(1);
            }).catch(() => { })
        },
        async batchDelTags() {
            const tags = [...this.selectedTagKeys];
            if (!tags.length || this.batchDeleting) { return Promise.resolve(); }

            this.batchDeleting = true;
            let deletedCount = 0;
            try {
                const res = await myAxios.post("/v2/api/repository/tags/batch-del", {
                    id: parseInt(this.$route.params.id),
                    tags,
                }, { timeout: 0 });
                const result = res?.data?.data;
                if (!Array.isArray(result?.deleted_tags) || !Array.isArray(result?.failed_tags)) {
                    messageError('未获取到删除结果，请刷新列表确认');
                    return;
                }
                deletedCount = result.deleted_tags.length;
                if (deletedCount > 0) {
                    messageSuccess(`已删除 ${deletedCount} 个版本`);
                }
                if (result.failed_tags.length > 0) {
                    messageError(`${result.failed_tags.length} 个版本删除失败，请重试`);
                }
            } catch {
                // The request layer displays the error; refresh partial remote changes.
            } finally {
                try {
                    this.selectedTagKeys = [];
                    await this.reloadTagsAfterDelete(deletedCount);
                } finally {
                    this.batchDeleting = false;
                }
            }
        },
        reloadTagsAfterDelete(deletedCount) {
            const remainingTotal = Math.max(0, this.tagPage.total - deletedCount);
            const lastPage = Math.max(1, Math.ceil(remainingTotal / this.tagPage.pageSize));
            if (this.tagPage.page > lastPage) {
                this.tagPage.page = lastPage;
            }
            return this.getVersion();
        },
        handleTagPageSizeChange() {
            this.tagPage.page = 1;
            this.selectedTagKeys = [];
            this.getVersion();
        },
        onSubmit() {
            myAxios.post('/v2/api/repository/edit', { id: parseInt(this.$route.params.id), ...this.form }).then(() => {
                messageSuccess('操作成功');
                this.getData();
                this.visible = false
            }).catch(() => { })
        },
        onekeyCopy(text) {

            var textarea = document.createElement('textarea');
            document.body.appendChild(textarea);
            textarea.style.position = 'fixed';
            textarea.style.clip = 'rect(0 0 0 0)';
            textarea.style.top = '10px';
            textarea.value = text;
            textarea.select();
            document.execCommand('copy', true);
            document.body.removeChild(textarea);
            messageSuccess("复制成功");
        },
        getData() {
            if (this.tabsActive == 'version') {
                this.getVersion();
            }
            if (this.tabsActive == 'info') {
                this.getInfo();
            }
        },

        getInfo() {
            this.loading = true;
            myAxios.post("/v2/api/repository/info", {
                id: this.id
            }).then(res => {
                let data = res.data.data || {};
                this.data = data;
                for (let key in this.form) {
                    this.form[key] = data[key] || ''
                }
                this.formVisibleType3 = this.form.visible_type == 3;
                this.loading = false;
            }).finally(() => { this.loading = false; });
        },

        getVersion() {
            this.loading = true;
            this.selectedTagKeys = [];
            return myAxios.post("/v2/api/repository/tags/list", {
                id: this.id,
                "page": this.tagPage.page,
                "page_size": this.tagPage.pageSize,
            }).then(res => {
                let data = res.data.data?.list || [];


                data.forEach(item => {
                    if (item?.Size) {
                        let B = item.Size;
                        let mb = Math.ceil(B / 1024 / 1024)
                        if (mb > 1024) {
                            let gb = mb / 1024
                            item.size = gb.toFixed(2) + 'GB'
                        } else {
                            item.size = mb.toFixed(2) + 'MB'
                        }
                    } else {
                        item.size = '未知'
                    }
                })
                this.tags = data
                this.tagPage.total = res.data.data?.total;
                const lastPage = Math.max(1, Math.ceil(this.tagPage.total / this.tagPage.pageSize));
                if (this.tagPage.page > lastPage) {
                    this.tagPage.page = lastPage;
                    return this.getVersion();
                }
            }).finally(() => { this.loading = false; });
        },


    }
}
</script>

<style scoped>
.back {
    margin-bottom: 30px;
}

.registry-detail-breadcrumb {
    height: 56px;
    padding: 0 24px;
}

.registry-version-toolbar {
    margin: 12px 0 16px;
}

.registry-version-selected {
    margin-left: 12px;
    color: var(--color-text-3);
}

.icon-action {
    margin-left: 6px;
    color: #3370ff;
    vertical-align: middle;
    width: 24px;
    height: 24px;
    font-size: 16px;
}

.icon-action :deep(.arco-icon) {
    font-size: 16px;
}

.registry-tag-copy,
.registry-tag-copy :deep(.arco-icon) {
    color: #333333 !important;
}

:deep(.arco-btn.registry-tag-copy) {
    color: #333333 !important;
}

.registry-info-form {
    width: 100%;
}

.registry-info-spin {
    display: block;
    width: 100%;
}

.registry-info-form :deep(.arco-form-item-label-col) {
    flex: 0 0 96px !important;
    width: 96px;
    max-width: 96px;
}

.registry-info-form :deep(.arco-form-item-wrapper-col) {
    flex: 1 1 auto !important;
    min-width: 0;
    max-width: calc(100% - 96px);
}

.registry-info-form :deep(.arco-form-item-content-wrapper),
.registry-info-form :deep(.arco-form-item-content) {
    width: 100%;
    min-width: 0;
}

.table-header :deep(.arco-table-th) {
    background: #f2f3f5;
    color: var(--color-text-1);
    font-weight: 400;
}

.table-header :deep(.arco-table-container) {
    border-left: 0;
    border-right: 0;
}

.table-header :deep(.arco-table-th:first-child),
.table-header :deep(.arco-table-td:first-child) {
    border-left: 0;
}

.table-header :deep(.arco-table-th:last-child),
.table-header :deep(.arco-table-td:last-child) {
    border-right: 0;
}

.registry-detail-form :deep(.arco-form-item-label) {
    white-space: nowrap;
}

.registry-detail-form :deep(.arco-form-item-wrapper-col) {
    min-width: 0;
}

.pd-20 {
    padding: 20px;
}

.title {
    font-size: 14px;
    font-weight: bold;
}

.title::before {
    display: block;
    content: " ";
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #0052D9;
    margin-right: 12px;
}

.upfile {
    position: relative;
}

.upfile .fileinput {
    width: 100%;
    min-width: 100px;
    position: absolute;
    z-index: 9;
    left: 0;
    right: 0;
    top: 0;
    bottom: 0;
    opacity: 0;
    cursor: pointer;
}

.upfile .fileinput::file-selector-button {
    display: none;
}
</style>
<style>
.registry-version-popover {
    max-width: 520px;
    white-space: normal;
}

.registry-version-popover .arco-popover-content,
.registry-version-popover .registry-version-digest-line,
.registry-version-popover .arco-form-item-content {
    min-width: 0;
    white-space: normal;
    overflow-wrap: anywhere;
    word-break: break-all;
}

.registry-version-popover .registry-version-digest-line {
    display: inline;
    line-height: 24px;
}

.registry-version-popover .registry-version-digest {
    display: inline;
}

.registry-version-popover .registry-version-digest-line .icon-action {
    display: inline-flex;
    margin-left: 4px;
    vertical-align: middle;
}

.registry-tag-copy,
.registry-tag-copy .arco-icon {
    color: #333333 !important;
}
</style>
