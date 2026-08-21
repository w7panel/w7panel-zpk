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
                    <a-table :loading="loading" :data="tags" style="width: 100%" class="table-header"
                        :pagination="false" row-key="TagName">
                        <template #columns>
                            <a-table-column title="镜像版本">
                                <template #cell="{ record }">
                                    <a-popover position="bl" content-class="registry-version-popover">
                                        <span class="c-blue cursor">{{ record.TagName }}</span>
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
                    <div v-if="tagPage.total > tagPage.pageSize" class="df jc-e mt-20">
                        <a-pagination v-model:current="tagPage.page" :total="tagPage.total"
                            :page-size="tagPage.pageSize" @change="getVersion" />
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
                <a-tab-pane key="build" title="镜像部署" v-if="hasAccess(data.user_id)">
                    <div class="zpk-toolbar-left">
                        <a-button type="primary" @click="openBuildForm()">
                            <template #icon><icon-plus /></template>
                            新增自动部署任务
                        </a-button>
                    </div>
                    <a-table :loading="loading" :data="builds" class="mt-20 table-header" style="width: 100%"
                        :pagination="false" row-key="id">
                        <template #columns>
                            <a-table-column title="应用">
                                <template #cell="{ record }">
                                    <span class="c-blue cursor" @click="showLog(record)">{{ record.k8s_app_name }}</span>
                                </template>
                            </a-table-column>

                            <a-table-column title="触发方式">
                                <template #cell="{ record }">{{ record.deploy_type_txt }}</template>
                            </a-table-column>
                            <a-table-column title="上次执行时间">
                                <template #cell="{ record }">{{ record.lastrun }}</template>
                            </a-table-column>
                            <a-table-column title="创建时间">
                                <template #cell="{ record }">{{ record.created }}</template>
                            </a-table-column>

                            <a-table-column title="操作">
                                <template #cell="{ record }">
                                    <a-button type="text" @click="getBuildDetail(record)">详情</a-button>
                                    <a-button type="text" @click="openBuildForm(record)">修改</a-button>
                                    <a-popconfirm content="确认要删除吗" type="warning" ok-text="确定"
                                        cancel-text="取消" content-class="zpk-delete-popconfirm"
                                        :ok-button-props="{ status: 'danger' }"
                                        :cancel-button-props="{ type: 'secondary' }" @ok="delBuild(record)">
                                        <a-button type="text">删除</a-button>
                                    </a-popconfirm>
                                </template>
                            </a-table-column>
                        </template>
                    </a-table>
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

        <a-modal v-model:visible="buildForm.show" :title="buildForm.id ? '修改自动部署任务' : '新增自动部署任务'"
            :width="850" :footer="false">
            <a-form ref="buildForm" :model="buildForm" :rules="rules" label-align="left"
                :label-col-props="{ span: 5, flex: '0 0 120px' }" :wrapper-col-props="{ span: 19, flex: '1' }"
                class="registry-detail-form registry-build-form">
                <a-form-item label="" style="margin-bottom:10px;">
                    <a-radio-group v-model="buildForm.selectType" style="margin-bottom:0;"
                        @change="changeBuildFormType">
                        <a-radio :value="1">当前集群</a-radio>
                        <a-radio :value="2">第三方集群</a-radio>
                    </a-radio-group>
                </a-form-item>

                <div v-if="buildForm.selectType == 1">
                    <a-form-item label="选择应用" field="k8s_container_name_arr">
                        <div class="df df-c" style="flex:1;">
                            <a-select v-model="buildForm.ccApp" placeholder="请选择应用" size="large" style="width:600px;"
                                @change="buildForm.selectCcapp">
                                <a-option v-if="!Object.keys(buildForm.appgroups || {}).length" disabled
                                    value="__empty__" label="暂无可选应用" />
                                <a-option v-for="(value, key) in buildForm.appgroups" :key="key" :label="value.title"
                                    :value="key" />
                            </a-select>

                            <div v-if="buildForm.treeLoading || buildForm.treeData.length" class="mt-10 buildformtree">
                                <a-spin :loading="buildForm.treeLoading">
                                    <div v-for="item in buildForm.treeData" :key="item.name" class="build-tree-group">
                                        <div class="build-tree-title">{{ item.label }}</div>
                                        <div v-for="child in item.children" :key="child.name" class="build-tree-container">
                                            <a-checkbox :model-value="buildForm.k8s_container_name_arr.includes(child.label)"
                                                @change="checked => toggleTreeContainer(child, checked)">
                                                {{ child.label }}
                                            </a-checkbox>
                                        </div>
                                    </div>
                                </a-spin>
                            </div>

                        </div>
                    </a-form-item>
                </div>

                <div v-if="buildForm.selectType == 2">
                    <a-form-item label="KUBECONFIG" field="k8s_config">
                        <div class="df df-c" style="flex:1;">
                            <a-textarea v-model="buildForm.k8s_config" size="large" :auto-size="{ minRows: 8, maxRows: 8 }"
                                @blur="buildForm.getNamespace()" style="width:600px;"
                                placeholder="请输入config" />
                            <div class="mt-10 df jc-e" style="width:600px;">
                                <div class="upfile df ml-10">
                                    <input type="file" class="fileinput" @change="upfile" />
                                    <a-button type="primary">导入</a-button>
                                </div>
                            </div>
                        </div>
                    </a-form-item>
                    <a-form-item label="命名空间" field="k8s_namespace">
                        <a-select v-model="buildForm.k8s_namespace" size="large" style="width:600px" placeholder="请选择"
                            @change="buildForm.getApps()">
                            <a-option v-for="item in namespaces" :key="item" :value="item" :label="item" />
                        </a-select>
                    </a-form-item>
                    <a-form-item label="应用类型" field="k8s_controller_type">
                        <a-select v-model="buildForm.k8s_controller_type" size="large" style="width:600px"
                            placeholder="请选择" @change="buildForm.getApps()">
                            <a-option v-for="(value, key) in controllerTypes" :key="key" :value="key"
                                :label="value" />
                        </a-select>
                    </a-form-item>
                    <div class="build-inline-form-row">
                        <a-form-item label="选择应用" field="k8s_app_name">
                                <a-select v-model="buildForm.k8s_app_name" size="large" placeholder="请选择"
                                    @change="buildForm.getContainer()">
                                    <a-option v-if="!apps.length" disabled value="__empty__" label="暂无可选应用" />
                                    <a-option v-for="item in apps" :key="item.name" :label="item.title"
                                        :value="item.name" />
                                </a-select>
                        </a-form-item>
                        <a-form-item label="选择容器" field="k8s_container_name_arr">
                            <a-select v-model="buildForm.k8s_container_name_arr" size="large" multiple
                                placeholder="请选择">
                                <a-option v-for="item in containers" :key="item.name" :label="item.title"
                                    :value="item.name" />
                            </a-select>
                        </a-form-item>
                    </div>
                </div>

                <a-form-item label="触发方式">
                    <div class="df df-c" style="flex:1;">
                        <a-radio-group v-model="buildForm.deploy_type" style="margin-bottom:0;">
                            <a-radio :value="1">更新版本</a-radio>
                            <a-radio :value="2">新增版本</a-radio>
                        </a-radio-group>
                        <span v-if="buildForm.deploy_type == 1" class="c-99 fs-12">对指定的版本名称更新镜像时，会自动执行部署任务</span>
                        <span v-if="buildForm.deploy_type == 2" class="c-99 fs-12">对匹配到的版本名称新增镜像版本时，会自动执行部署任务</span>
                    </div>
                </a-form-item>
                <a-form-item v-if="buildForm.deploy_type == 2" label="匹配方式" field="repository_tag">

                    <div class="build-match-rule-row">
                        <a-select v-model="buildForm.match_type" size="large" placeholder="请选择匹配方式"
                            class="build-match-type-select">
                            <a-option label="前缀匹配" :value="1" />
                            <a-option label="正则匹配" :value="2" />
                        </a-select>
                        <div class="tag-cpn build-match-rule-tags df df-ww">
                            <a-tag v-for="(tag, index) in buildForm.repository_tag_arr" :key="index" class="tag"
                                closable @close="deleteTag(index)">{{ tag }}</a-tag>
                            <div class="input fc">
                                <input type="text" placeholder="请输入匹配规则" v-model="buildForm.taginput"
                                    @keydown.enter="addTag" @blur="addTag" />
                            </div>
                        </div>
                    </div>
                </a-form-item>
                <a-form-item v-if="buildForm.deploy_type == 1"
                    :label="(buildForm.deploy_type == 2 && buildForm.match_type == 2) ? '匹配规则' : '版本名称'" field="repository_tag">
                    <div class="tag-cpn df df-ww">
                        <a-tag v-for="(tag, index) in buildForm.repository_tag_arr" :key="index" class="tag" closable
                            @close="deleteTag(index)">{{ tag }}</a-tag>
                        <div class="input fc">
                            <input type="text" placeholder="请输入" v-model="buildForm.taginput" @keydown.enter="addTag"
                                @blur="addTag" />
                        </div>
                    </div>
                </a-form-item>
            </a-form>
            <div class="dialog-footer">
                <a-button size="large" @click="buildForm.show = false">取消</a-button>
                <a-button type="primary" size="large" @click="submitBuildForm">确定</a-button>
            </div>
        </a-modal>

        <a-modal v-model:visible="buildDetail.show" title="自动部署任务详情" :width="750" :footer="false">
            <a-form :model="buildDetail" label-align="left" :label-col-props="{ span: 6, flex: '0 0 150px' }"
                :wrapper-col-props="{ span: 18, flex: '1' }" class="registry-detail-form">
                <a-form-item label="KUBECONFIG">
                    <span v-if="!buildDetail.k8s_config">-</span>
                    <a-textarea v-else v-model="buildDetail.k8s_config" :auto-size="{ minRows: 5, maxRows: 5 }" readonly />
                </a-form-item>
                <a-form-item label="命名空间">{{ buildDetail.k8s_namespace }}</a-form-item>
                <a-form-item label="应用">{{ buildDetail.k8s_app_name }}</a-form-item>
                <a-form-item label="容器">{{ buildDetail.k8s_container_name }}</a-form-item>
                <a-form-item label="应用类型">{{ buildDetail.k8s_controller_type_txt }}</a-form-item>
                <a-form-item label="触发方式">{{ buildDetail.deploy_type_txt }}</a-form-item>
                <a-form-item label="版本名称">{{ buildDetail.tag_name }}</a-form-item>
                <a-form-item label="创建时间">{{ buildDetail.created }}</a-form-item>
            </a-form>
        </a-modal>

        <a-modal v-model:visible="logls.show" :width="1000" title="执行记录" :footer="false">
            <a-spin :loading="logls.loading" class="registry-log-spin">
                <div v-if="!logls.list.length" class="registry-log-empty">
                    <a-empty description="暂无执行记录" />
                </div>
                <div v-else class="df">
                    <div>
                        <div class="df jc-e" style="height:400px; overflow:auto; max-width:300px;">
                            <a-tabs v-model:active-key="logls.act" position="left" class="logtabs" @change="termInit">
                                <a-tab-pane v-for="(item, index) in logls.list" :key="String(index)" :title="item.created" />
                            </a-tabs>
                        </div>
                    </div>
                    <div class="ml-20 fc">
                        <div ref="term" id="term" class="mt-10" style="height:400px;"></div>
                    </div>
                </div>
            </a-spin>
        </a-modal>

    </div>
</template>

<script>
import myAxios from "@/utils";
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import userMixin from "@/utils/user-mixin";
import { messageSuccess } from "@/utils/ui-feedback";
import { IconArrowLeft, IconCopy, IconEdit, IconPlus } from '@arco-design/web-vue/es/icon';

export default {
    components: {
        IconArrowLeft,
        IconCopy,
        IconEdit,
        IconPlus,
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
            tagPage: {
                page: 1,
                pageSize: 10,
                total: 0,
            },

            tabsActive: 'version',

            builds: [],

            controllerTypes: { deployments: '无状态应用', statefulsets: '有状态应用', daemonsets: '守护进程应用' },
            namespaces: [],
            apps: [],
            containers: [],

            buildForm: {
                show: false,
                id: 0,

                selectType: 0,

                defaultConfig: "",
                isDefaultConfig: false,

                k8s_config: '',
                k8s_namespace: '',
                k8s_controller_type: '',
                k8s_app_name: '',
                k8s_container_name: '',
                k8s_container_name_arr: [],
                deploy_type: 1,
                repository_tag_arr: [],
                repository_tag: '',
                match_type: 1,

                appgroups: [],
                treeData: [],
                treeLoading: false,
                ccApp: '',

                taginput: '',
            },

            rules: {
                k8s_config: [{ required: true, message: 'KUBECONFIG不能为空', trigger: 'blur' }],
                k8s_namespace: [{ required: true, message: '命名空间不能为空', trigger: 'blur' }],
                k8s_controller_type: [{ required: true, message: '内容不能为空', trigger: 'blur' }],
                k8s_app_name: [{ required: true, message: '请选择应用', trigger: 'blur' }],
                k8s_container_name_arr: [{ required: true, message: '请选择容器', trigger: 'blur' }],
                repository_tag: [{ required: true, message: '内容不能为空', trigger: 'blur' }],
            },

            buildDetail: {
                show: false,
            },

            logls: {
                show: false,
                loading: false,
                opened: false,
                act: '0',
                list: [],
            },
            term: null,
        }
    },
    mixins: [userMixin],
    created() {
        this.initBuildForm();
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
        'buildForm.selectType'(v) {
            if (v == 2) {
                this.buildForm.getNamespace();
            }
        },
        'buildForm.k8s_container_name_arr'(v) {
            this.buildForm.k8s_container_name = v.join(',');
        },
        'logls.show'(v) {
            if (!v) { return }
            this.logls.opened = true;
            this.$nextTick(() => this.termInit());
        },
    },
    methods: {
        showLog(row) {
            this.logls = {
                ...this.logls,
                show: true,
                act: '0',
                loading: true,
                opened: false,
                list: [],
            }
            myAxios.post('/v2/api/repository/deploy_rule/deploy-log', {
                id: row.id,
                page: 1,
                page_size: 9999,
            }).then(res => {
                let list = res?.data?.data?.list || [];
                list.map(i => {
                    i.created = this.formatDate(i.created_at);
                })
                this.logls.list = list;
            }).finally(() => {
                this.logls.loading = false;
                this.termInit();
            })
        },
        changeBuildFormType() {
            this.buildForm.k8s_app_name = '';
            this.buildForm.k8s_container_name_arr = [];
        },
        changeTreeContainer(data) {
            if (!this.buildForm.k8s_container_name_arr.includes(data.name)) { return }
            this.buildForm.k8s_app_name = data.app;
            this.buildForm.k8s_container_name_arr = [data.label];
            this.buildForm.k8s_controller_type = data.kind;
        },
        toggleTreeContainer(data, checked) {
            if (!checked) {
                this.buildForm.k8s_container_name_arr = this.buildForm.k8s_container_name_arr.filter(item => item !== data.label);
                return;
            }
            this.buildForm.k8s_container_name_arr = [data.label];
            this.changeTreeContainer(data);
        },

        upfile(file) {
            var file = file.target.files[0];
            const reader = new FileReader();
            reader.onload = () => {
                this.buildForm.k8s_config = reader.result;
            };
            reader.readAsText(file);
        },
        initBuildForm() {
            this.buildForm.treeData = []

            this.buildForm.getAppgroups = () => {
                myAxios.get('/v2/api/repository/deploy_rule/k8s/proxy/apis/appgroup.w7.cc/v1alpha1/namespaces/default/appgroups').then(res => {
                    let list = res?.data?.items || [];
                    let kvlist = {};
                    list = list.filter(i => !i?.metadata?.labels?.['w7.cc/parent']).map(i => {
                        let statusItem = i?.status?.items || [];
                        let childrenApp = statusItem.map(si => ({
                            title: si.title || si.name,
                            name: si.name,
                            kind: si.kind.toLowerCase() + 's',
                            ready: si.ready,
                            group: i.metadata.name,
                        }));
                        kvlist[i.metadata.name] = {
                            name: i.metadata.name,
                            title: i?.spec?.title || i.metadata.name,
                            childrenApp: childrenApp,
                        }
                        return {
                            name: i.metadata.name,
                            title: i?.spec?.title || i.metadata.name,
                            childrenApp: childrenApp,
                        }
                    })
                    this.buildForm.appgroups = kvlist;
                    if (this.buildForm.k8s_app_name) {
                        let find = list.find(i => i.childrenApp?.find(c => c.name == this.buildForm.k8s_app_name));
                        if (find) {
                            this.buildForm.selectType = 1;
                            this.buildForm.ccApp = find.name;
                            let k8s_container_name_arr = this.buildForm.k8s_container_name_arr || [];
                            this.buildForm.selectCcapp();
                            this.buildForm.k8s_container_name_arr = k8s_container_name_arr;
                        } else {
                            this.buildForm.selectType = 2;
                        }
                    } else {
                        this.buildForm.selectType = 1;
                    }
                }).catch(() => {
                    this.buildForm.selectType = 1;
                })
            };

            this.buildForm.selectCcapp = async () => {
                this.buildForm.k8s_container_name_arr = [];
                this.buildForm.treeLoading = true;
                let childrenApp = this.buildForm.appgroups[this.buildForm.ccApp]?.childrenApp || [];
                let tree = [];
                try {
                    for (let i = 0; i < childrenApp.length; i++) {
                        let c = childrenApp[i];
                        let { data } = await myAxios.get(`/v2/api/repository/deploy_rule/k8s/proxy/apis/apps/v1/namespaces/default/${c.kind}/${c.name}`);
                        let containers = data?.spec?.template?.spec?.containers || [];
                        containers = containers.map(i => {
                            return {
                                label: i.name,
                                name: i.name,
                                type: 'container',
                                app: c.name,
                                kind: c.kind,
                            }
                        })
                        tree.push({
                            label: c.title,
                            name: c.name,
                            kind: c.kind,
                            type: 'app',
                            children: containers,
                        })
                    }
                    this.buildForm.treeLoading = false;
                } catch {
                    this.buildForm.treeLoading = false;
                }

                this.buildForm.treeData = tree;
            }

            this.buildForm.getNamespace = () => {
                if (!this.buildForm.k8s_config) {
                    this.buildForm.getApps();
                    this.namespaces = [];
                    return;
                }
                myAxios.post('/v2/api/repository/deploy_rule/k8s/namespaces', {
                    k8s_config: this.buildForm.k8s_config,
                }).then(res => {
                    this.namespaces = res?.data?.data?.list || [];
                    if (!this.buildForm.k8s_namespace && this.namespaces.includes('default')) {
                        this.buildForm.k8s_namespace = 'default';
                    }
                    this.buildForm.getApps();
                }).catch(() => { })
            };
            this.buildForm.getApps = () => {
                if (!this.buildForm.k8s_controller_type || !this.buildForm.k8s_namespace || !this.buildForm.k8s_config) {
                    if (this.buildForm.selectType == 2) {
                        this.buildForm.k8s_app_name = '';
                    }
                    this.apps = [];
                    this.buildForm.getContainer();
                    return
                }
                myAxios.post('/v2/api/repository/deploy_rule/k8s/apps', {
                    k8s_config: this.buildForm.k8s_config,
                    k8s_namespace: this.buildForm.k8s_namespace,
                    k8s_controller_type: this.buildForm.k8s_controller_type,
                }).then(res => {
                    let list = res?.data?.data?.list || [];
                    list = list.map(i => ({
                        name: i,
                        title: i,
                    }))
                    this.apps = list;
                    if (this.buildForm.k8s_app_name) {
                        let find = this.apps?.find(i => i.name == this.buildForm.k8s_app_name);
                        if (!find) { this.buildForm.k8s_app_name = ''; }
                    }
                    this.buildForm.getContainer();
                }).catch(() => { })
            }
            this.buildForm.getContainer = () => {
                if (!this.buildForm.k8s_controller_type || !this.buildForm.k8s_namespace || !this.buildForm.k8s_app_name || !this.buildForm.k8s_config) {
                    if (this.buildForm.selectType == 2) {
                        this.buildForm.k8s_container_name = '';
                        this.buildForm.k8s_container_name_arr = [];
                    }
                    this.containers = [];
                    return
                }
                myAxios.post('/v2/api/repository/deploy_rule/k8s/app-containers', {
                    k8s_config: this.buildForm.k8s_config,
                    k8s_namespace: this.buildForm.k8s_namespace,
                    k8s_controller_type: this.buildForm.k8s_controller_type,
                    k8s_app_name: this.buildForm.k8s_app_name,
                }).then(res => {
                    let list = res?.data?.data?.list || [];
                    list = list.map(i => ({
                        name: i,
                        title: i,
                    }))
                    this.containers = list;
                    if (this.buildForm.k8s_container_name) {
                        let find = this.containers?.find(i => i.name == this.buildForm.k8s_container_name);
                        if (!find) { this.buildForm.k8s_container_name = ''; }
                    }
                }).catch(() => { })
            }
        },
        getBuildDetail(row) {
            this.buildDetail = {
                show: true,
                ...row,
            }
        },
        submitBuildForm() {
            this.$refs.buildForm.validate((errors) => {
                if (errors) { return }

                let bf = this.buildForm;
                let formdata = {
                    k8s_config: bf.k8s_config,
                    k8s_namespace: bf.k8s_namespace,
                    k8s_controller_type: bf.k8s_controller_type,
                    k8s_app_name: bf.k8s_app_name,
                    k8s_container_name: bf.k8s_container_name,
                    repository_id: this.id,
                    deploy_type: bf.deploy_type,
                    repository_tag: bf.repository_tag,
                    id: bf.id,
                    ...(bf.deploy_type == 2 ? {
                        match_type: bf.match_type,
                    } : {}),
                }
                if (this.buildForm.selectType == 1) {
                    formdata.k8s_config = '';
                    formdata.k8s_namespace = 'default';
                }

                if (bf.id) {
                    myAxios.post('/v2/api/repository/deploy_rule/edit', formdata).then(res => {
                        messageSuccess('操作成功');
                        this.getData();
                        this.buildForm.show = false;
                    }).catch(() => { })
                } else {
                    myAxios.post('/v2/api/repository/deploy_rule/add', formdata).then(res => {
                        messageSuccess('操作成功');
                        this.getData();
                        this.buildForm.show = false;
                    }).catch(() => { })
                }
            });
        },
        addTag() {
            if (!this.buildForm.taginput) { return }
            this.buildForm.repository_tag_arr.push(this.buildForm.taginput);
            this.buildForm.repository_tag = this.buildForm.repository_tag_arr.join(',');
            this.buildForm.taginput = '';
        },
        deleteTag(index) {
            this.buildForm.repository_tag_arr.splice(index, 1);
            this.buildForm.repository_tag = this.buildForm.repository_tag_arr.join(',');
        },
        openBuildForm(row) {
            if (row) {
                this.buildForm.repository_tag = row.tag_name || '';
                this.buildForm.repository_tag_arr = this.buildForm.repository_tag.split(',');
                this.buildForm.k8s_container_name = row?.k8s_container_name || '';
                this.buildForm.k8s_container_name_arr = this.buildForm.k8s_container_name.split(',');
            } else {
                this.buildForm.repository_tag = '';
                this.buildForm.repository_tag_arr = [];
                this.buildForm.k8s_container_name = '';
                this.buildForm.k8s_container_name_arr = [];
                this.buildForm.ccApp = '';
                this.buildForm.treeData = [];
            }
            this.buildForm = {
                ...this.buildForm,
                show: true,
                id: row?.id || 0,
                selectType: 0,
                k8s_config: row?.k8s_config || this.buildForm.defaultConfig,
                k8s_namespace: row?.k8s_namespace || '',
                k8s_controller_type: row?.k8s_controller_type || 'deployments',
                k8s_app_name: row?.k8s_app_name || '',
                deploy_type: row?.deploy_type || 1,
                match_type: row?.match_type || 1,
            }
            this.buildForm.getAppgroups();
        },
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
                this.getData();
            }).catch(() => { })
        },
        delBuild(row) {
            return myAxios.post('/v2/api/repository/deploy_rule/del', {
                id: row.id
            }).then(() => {
                messageSuccess('操作成功');
                this.getBuild();
            }).catch(() => { })
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
            if (this.tabsActive == 'build') {
                this.getBuild();
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
            myAxios.post("/v2/api/repository/tags/list", {
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
            }).finally(() => { this.loading = false; });
        },

        getBuild() {
            this.loading = true;
            myAxios.post("/v2/api/repository/deploy_rule/list", {
                repository_id: this.id,
            }).then(res => {
                let list = res?.data?.data?.list || [];
                list.map(i => {
                    i.k8s_controller_type_txt = this.controllerTypes[i.k8s_controller_type];
                    i.deploy_type_txt = { 1: '更新触发', 2: '新增触发' }[i.deploy_type];
                    i.created = this.formatDate(i.created_at);
                    i.lastrun = i.latest_trigger_at ? this.formatDate(i.latest_trigger_at) : '-'
                    if (!/^\d{4}\-\d{2}\-\d{2}\s\d{2}:\d{2}:\d{2}$/.test(i.lastrun)) {
                        i.lastrun = '-';
                    }
                })
                this.builds = list;
            }).finally(() => {
                this.loading = false;
            })
        },
        termInit() {
            if (this.logls.loading || !this.logls.opened) { return }
            if (!this.logls.list.length) {
                this.term = null;
                return;
            }
            let termEl = document.getElementById("term");
            if (!termEl) { return }
            termEl.innerHTML = "";
            this.term = null;
            this.term = new Terminal({
                rendererType: 'dom',
                cursorBlink: false,

            });
            this.term.open(termEl);
            this.fitAddon = new FitAddon()
            this.term.loadAddon(this.fitAddon);
            this.fitAddon.fit();

            this.termWrite();
        },
        termWrite() {
            let e = this.logls?.list?.[this.logls.act]?.k8s_log;
            if (e) {
                e = e.replace(/\x20+/g, ' ');
                e = e.replace(/(?<!\r)\n/g, '\r\n');
                this.term?.write(e);
            }
        },
        formatDate(date) {
            var d = new Date(date),
                month = '' + (d.getMonth() + 1),
                day = '' + d.getDate(),
                year = d.getFullYear();

            if (month.length < 2)
                month = '0' + month;
            if (day.length < 2)
                day = '0' + day;

            var hours = String(d.getHours());
            var minutes = String(d.getMinutes());
            var seconds = String(d.getSeconds());

            if (hours.length < 2)
                hours = '0' + hours;
            if (minutes.length < 2)
                minutes = '0' + minutes;
            if (seconds.length < 2)
                seconds = '0' + seconds;

            return [year, month, day].join('-') + ' ' + [hours, minutes, seconds].join(':');
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

.buildformtree {
    padding: 10px;
    background: rgb(247, 247, 250);
    width: 600px;
    box-sizing: border-box;
}

.build-tree-group + .build-tree-group {
    margin-top: 8px;
}

.build-tree-title {
    color: #333333;
    font-weight: 500;
    line-height: 28px;
}

.build-tree-container {
    padding-left: 18px;
    line-height: 28px;
}

.registry-log-empty {
    height: 400px;
    width: 100%;
    box-sizing: border-box;
    display: flex;
    align-items: center;
    justify-content: center;
}

.registry-log-spin,
.registry-log-spin :deep(.arco-spin-children) {
    display: block;
    width: 100%;
}

.build-inline-form-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    gap: 16px;
    width: 760px;
}

.build-inline-form-row :deep(.arco-form-item) {
    display: flex;
    flex-wrap: nowrap;
    align-items: flex-start;
    min-width: 0;
}

.build-inline-form-row :deep(.arco-form-item-label-col) {
    flex: 0 0 158px !important;
    width: 158px;
    max-width: 158px;
}

.build-inline-form-row :deep(.arco-form-item-wrapper-col) {
    flex: 1 1 auto !important;
    min-width: 0;
}

.build-inline-form-row :deep(.arco-select) {
    width: 100%;
}

.build-match-rule-row {
    display: flex;
    align-items: center;
    gap: 16px;
    width: 600px;
}

.build-match-type-select {
    flex: 0 0 128px;
    width: 128px;
}

.build-match-rule-tags {
    flex: 1 1 auto;
    min-width: 0;
    width: auto;
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

.tag-cpn {
    width: 600px;
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
