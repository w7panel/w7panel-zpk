<template>
    <div class="bg-white" style="min-height:100%;">
        <div class="com-back df ai-c ">
            <span class="backbtn df ai-c" @click="$router.go(-1)">
                <el-icon class="backicon" color="#0052D9" :size="20">
                    <Back />
                </el-icon>
                <span class="fs-18">{{ data.namespace }}/{{ data.name }}</span>
            </span>
        </div>
        <div class="bg-white" style="padding: 0 24px 6px;">
            <el-tabs v-model="tabsActive">
                <el-tab-pane label="版本管理" name="version">
                    <el-table v-loading="loading" :data="tags" style="width: 100%"
                        :header-cell-style="{ background: '#F5F7FA', color: '#909399' }">
                        <el-table-column label="镜像版本">
                            <template #default="scope">
                                <el-popover placement="bottom-start" width="800px">
                                    <template #reference>
                                        <span class="c-blue cursor">{{ scope.row.TagName }}</span>
                                    </template>
                                    <el-form label-width="160px">
                                        <el-form-item label="镜像ID(SHA256)" style="margin-bottom:0;">
                                            <span>{{ scope.row.Digest }}</span>
                                            <el-icon :size="12" @click="onekeyCopy(scope.row.Digest)"
                                                style="cursor:pointer;margin-left:6px;">
                                                <DocumentCopy />
                                            </el-icon>
                                        </el-form-item>
                                        <el-form-item label="平台"
                                            style="margin-bottom:0;">{{ scope.row.Platform }}</el-form-item>
                                        <el-form-item label="制品类型"
                                            style="margin-bottom:0;">{{ scope.row.Type }}</el-form-item>
                                    </el-form>
                                </el-popover>
                                <el-icon :size="12" @click="onekeyCopy(scope.row.TagName)"
                                    style="cursor:pointer;margin-left:6px;">
                                    <DocumentCopy />
                                </el-icon>
                            </template>
                        </el-table-column>

                        <el-table-column label="大小" prop="size"></el-table-column>


                        <el-table-column label="创建时间" prop="CreatedAt" width="200">
                            <template #default="scope">
                                {{ scope.row.CreatedAt ? new Date(scope.row.CreatedAt).toLocaleString() : '未知' }}
                            </template>
                        </el-table-column>
                        <el-table-column label="修改时间" prop="CreatedAt" width="200">
                            <template #default="scope">
                                {{ scope.row.UpdatedAt ? new Date(scope.row.UpdatedAt).toLocaleString() : '未知' }}
                            </template>
                        </el-table-column>
                        <el-table-column label="操作">
                            <template #default="scope">
                                <el-button type="text" size="mini" @click="del(scope.row)">删除</el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                    <div class="df jc-e mt-20">
                        <el-pagination background layout="prev, pager, next" v-model:current-page="tagPage.page"
                            :total="tagPage.total" :page-size="tagPage.pageSize"
                            @current-change="getVersion"></el-pagination>
                    </div>
                </el-tab-pane>
                <el-tab-pane label="仓库信息" name="info">
                    <el-form v-loading="loading" label-suffix="：" label-width="150px" class="mt-24">
                        <el-form-item label="仓库名称">{{ data.namespace }}/{{ data.name }}</el-form-item>
                        <el-form-item label="仓库地址">{{ data.registry }}/{{ data.namespace }}/{{ data.name }}<el-icon
                                class="ml-10" :size="12"
                                @click="onekeyCopy(`${data.registry}/${data.namespace}/${data.name}`)"
                                style="cursor:pointer">
                                <DocumentCopy />
                            </el-icon></el-form-item>
                        <el-form-item label="命名空间">{{ data.namespace }}</el-form-item>
                        <el-form-item label="公共权限">{{ data.visible_type === 3 ? namespaceTypeMap[data.namespace] :
                            visibleTypeMap[data.visible_type]}}<el-icon class="ml-10" :size="12"
                                @click="edit('visible_type')" style="cursor:pointer" v-if="hasAccess(data.user_id)">
                                <EditPen />
                            </el-icon></el-form-item>
                        <el-form-item label="描述">{{ data.desc }}<el-icon class="ml-10" :size="12" @click="edit('desc')"
                                style="cursor:pointer" v-if="hasAccess(data.user_id)">
                                <EditPen />
                            </el-icon></el-form-item>
                        <el-form-item label="创建时间">{{ new Date(data.created_at).toLocaleString() }}</el-form-item>
                    </el-form>
                </el-tab-pane>
                <el-tab-pane label="镜像部署" name="build" v-if="hasAccess(data.user_id)">
                    <div>
                        <el-button type="primary" @click="openBuildForm()">新增自动部署任务</el-button>
                    </div>
                    <el-table v-loading="loading" :data="builds" class="mt-20" style="width: 100%"
                        :header-cell-style="{ background: '#F5F7FA', color: '#909399' }">
                        <el-table-column label="应用">
                            <template #default="scope">
                                <span class="c-blue cursor"
                                    @click="showLog(scope.row)">{{ scope.row.k8s_app_name }}</span>
                            </template>
                        </el-table-column>

                        <el-table-column label="触发方式">
                            <template #default="scope">{{ scope.row.deploy_type_txt }}</template>
                        </el-table-column>
                        <el-table-column label="上次执行时间">
                            <template #default="scope">{{ scope.row.lastrun }}</template>
                        </el-table-column>
                        <el-table-column label="创建时间">
                            <template #default="scope">{{ scope.row.created }}</template>
                        </el-table-column>

                        <el-table-column label="操作">
                            <template #default="scope">
                                <el-button type="text" size="mini" @click="getBuildDetail(scope.row)">详情</el-button>
                                <el-button type="text" size="mini" @click="openBuildForm(scope.row)">修改</el-button>
                                <el-button type="text" size="mini" @click="delBuild(scope.row)">删除</el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                </el-tab-pane>
            </el-tabs>
        </div>

        <el-dialog v-model="visible" title="编辑镜像" :width="500">
            <el-form ref="form" :model="form" label-width="80px" label-position="left">
                <el-form-item label="描述" prop="desc" v-if="editProp === 'desc'">
                    <el-input v-model="form.desc" type="textarea" :rows="5" />
                </el-form-item>
                <el-form-item label="公共权限" prop="visible_type" v-if="editProp === 'visible_type'">
                    <el-checkbox v-model="formVisibleType3" @change="v => form.visible_type = v ? 3 : 1">跟随命名空间</el-checkbox>
                    <el-radio-group v-model="form.visible_type" :disabled="formVisibleType3">
                        <el-radio :label="1">私有读写</el-radio>
                        <el-radio :label="4">公有读私有写</el-radio>
                        <el-radio :label="2">公有读写</el-radio>
                    </el-radio-group>
                </el-form-item>
                <el-form-item>
                    <el-button type="primary" size="large" @click="onSubmit">确定</el-button>
                    <el-button size="large" @click="visible = false">取消</el-button>
                </el-form-item>
            </el-form>
        </el-dialog>

        <el-dialog v-model="buildForm.show" :title="buildForm.id ? '修改自动部署任务' : '新增自动部署任务'" :width="800">
            <el-form ref="buildForm" :model="buildForm" :rules="rules" label-width="120px">
                <el-form-item label="" style="margin-bottom:10px;">
                    <el-radio-group v-model="buildForm.selectType" style="margin-bottom:0;"
                        @change="changeBuildFormType">
                        <el-radio :value="1" :label="1" size="large">当前集群</el-radio>
                        <el-radio :value="2" :label="2" size="large">第三方集群</el-radio>
                    </el-radio-group>
                </el-form-item>

                <div v-if="buildForm.selectType == 1">
                    <el-form-item label="选择应用" prop="k8s_container_name_arr">
                        <div class="df df-c" style="flex:1;">
                            <el-select v-model="buildForm.ccApp" placeholder="请选择应用" size="large" style="width:600px;"
                                @change="buildForm.selectCcapp">
                                <el-option v-for="(value, key) in buildForm.appgroups" :key="key" :label="value.title"
                                    :value="key"></el-option>
                            </el-select>

                            <div class="mt-10"
                                style="padding:10px;background:rgb(247,247,250);width:600px;box-sizing:border-box;">
                                <el-tree style="width:580px" class="buildformtree" :data="buildForm.treeData"
                                    v-loading="buildForm.treeLoading" default-expand-all>
                                    <template #default="{ node, data }">
                                        <el-checkbox v-if="data.type == 'container'"
                                            v-model="buildForm.k8s_container_name_arr"
                                            @change="changeTreeContainer(data)" :label="node.label" />
                                        <span v-else>{{ node.label }}</span>
                                    </template>
                                </el-tree>
                            </div>

                        </div>
                    </el-form-item>
                </div>

                <div v-if="buildForm.selectType == 2">
                    <el-form-item label="KUBECONFIG" prop="k8s_config">
                        <div class="df df-c" style="flex:1;">
                            <el-input type="textarea" v-model="buildForm.k8s_config" size="large" :rows="8"
                                @blur="buildForm.getNamespace()" style="width:600px;"
                                placeholder="请输入config"></el-input>
                            <div class="mt-10 df jc-e" style="width:600px;">
                                <div class="upfile df ml-10">
                                    <input type="file" class="fileinput" @change="upfile" />
                                    <el-button type="primary">导入</el-button>
                                </div>
                            </div>
                        </div>
                    </el-form-item>
                    <el-form-item label="命名空间" prop="k8s_namespace">
                        <el-select v-model="buildForm.k8s_namespace" size="large" style="width:600px" placeholder="请选择"
                            @change="buildForm.getApps()">
                            <el-option v-for="item in namespaces" :key="item" :value="item" :label="item"></el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item label="应用类型" prop="k8s_controller_type">
                        <el-select v-model="buildForm.k8s_controller_type" size="large" style="width:600px"
                            placeholder="请选择" @change="buildForm.getApps()">
                            <el-option v-for="(value, key) in controllerTypes" :key="key" :value="key"
                                :label="value"></el-option>
                        </el-select>
                    </el-form-item>
                    <div class="df" style="width:720px;">
                        <div class="f1">
                            <el-form-item label="选择应用" prop="k8s_app_name">
                                <el-select v-model="buildForm.k8s_app_name" size="large" placeholder="请选择"
                                    style="flex:1;" @change="buildForm.getContainer()">
                                    <el-option v-for="item in apps" :key="item.name" :label="item.title"
                                        :value="item.name"></el-option>
                                </el-select>
                            </el-form-item>
                        </div>
                        <div class="f1">
                            <el-form-item label="选择容器" prop="k8s_container_name_arr">
                                <el-select v-model="buildForm.k8s_container_name_arr" size="large" multiple
                                    placeholder="请选择" style="flex:1;">
                                    <el-option v-for="item in containers" :key="item.name" :label="item.title"
                                        :value="item.name"></el-option>
                                </el-select>
                            </el-form-item>
                        </div>
                    </div>
                </div>

                <el-form-item label="触发方式">
                    <div class="df df-c" style="flex:1;">
                        <el-radio-group v-model="buildForm.deploy_type" style="margin-bottom:0;">
                            <el-radio :value="1" :label="1" size="large">更新版本</el-radio>
                            <el-radio :value="2" :label="2" size="large">新增版本</el-radio>
                        </el-radio-group>
                        <span v-if="buildForm.deploy_type == 1" class="c-99 fs-12">对指定的版本名称更新镜像时，会自动执行部署任务</span>
                        <span v-if="buildForm.deploy_type == 2" class="c-99 fs-12">对匹配到的版本名称新增镜像版本时，会自动执行部署任务</span>
                    </div>
                </el-form-item>
                <el-form-item v-if="buildForm.deploy_type == 2" label="匹配方式" prop="repository_tag">

                    <div class="df ai-c" style="width:600px;">
                        <el-select v-model="buildForm.match_type" size="large" placeholder="请选择匹配方式">
                            <el-option label="前缀匹配" :value="1"></el-option>
                            <el-option label="正则匹配" :value="2"></el-option>
                        </el-select>
                        <div class="tag-cpn df df-ww ml-20" style="flex:1;">
                            <el-tag v-for="(tag, index) in buildForm.repository_tag_arr" :key="index" class="tag"
                                closable @close="deleteTag(index)">{{ tag }}</el-tag>
                            <div class="input fc">
                                <input type="text" placeholder="请输入匹配规则" v-model="buildForm.taginput"
                                    @keydown.enter="addTag" @blur="addTag" />
                            </div>
                        </div>
                    </div>
                </el-form-item>
                <el-form-item v-if="buildForm.deploy_type == 1"
                    :label="(buildForm.deploy_type == 2 && buildForm.match_type == 2) ? '匹配规则' : '版本名称'" prop="repository_tag">
                    <div class="tag-cpn df df-ww">
                        <el-tag v-for="(tag, index) in buildForm.repository_tag_arr" :key="index" class="tag" closable
                            @close="deleteTag(index)">{{ tag }}</el-tag>
                        <div class="input fc">
                            <input type="text" placeholder="请输入" v-model="buildForm.taginput" @keydown.enter="addTag"
                                @blur="addTag" />
                        </div>
                    </div>
                </el-form-item>
                <el-form-item>
                    <el-button type="primary" size="large" @click="submitBuildForm">确定</el-button>
                    <el-button size="large" @click="buildForm.show = false">取消</el-button>
                </el-form-item>
            </el-form>
        </el-dialog>

        <el-dialog v-model="buildDetail.show" title="自动部署任务详情" :width="750">
            <el-form ref="buildForm" :model="buildDetail" :rules="rules" label-width="150px">
                <el-form-item label="KUBECONFIG">
                    <span v-if="!buildDetail.k8s_config">-</span>
                    <el-input v-else type="textarea" v-model="buildDetail.k8s_config" rows="5" readonly />
                </el-form-item>
                <el-form-item label="命名空间">{{ buildDetail.k8s_namespace }}</el-form-item>
                <el-form-item label="应用">{{ buildDetail.k8s_app_name }}</el-form-item>
                <el-form-item label="容器">{{ buildDetail.k8s_container_name }}</el-form-item>
                <el-form-item label="应用类型">{{ buildDetail.k8s_controller_type_txt }}</el-form-item>
                <el-form-item label="触发方式">{{ buildDetail.deploy_type_txt }}</el-form-item>
                <el-form-item label="版本名称">{{ buildDetail.tag_name }}</el-form-item>
                <el-form-item label="创建时间">{{ buildDetail.created }}</el-form-item>
            </el-form>
        </el-dialog>

        <el-dialog v-model="logls.show" :width="1000" title="执行记录" @opened="logls.opened = true; termInit();">
            <div v-loading="logls.loading" class="df">
                <div>
                    <div class="df jc-e" style="height:400px; overflow:auto; max-width:300px;">
                        <el-tabs v-model="logls.act" tab-position="left" class="logtabs" @tab-change="termInit">
                            <el-tab-pane v-for="(item, index) in logls.list" :key="index">
                                <template #label>
                                    <span>{{ item.created }}</span>
                                </template>
                            </el-tab-pane>
                        </el-tabs>
                    </div>
                </div>
                <div class="ml-20 fc">
                    <div ref="term" id="term" class="mt-10" style="height:400px;"></div>
                </div>
            </div>
        </el-dialog>

    </div>
</template>

<script>
import myAxios from "@/utils";
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import userMixin from "@/utils/user-mixin";

export default {
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
    },
    methods: {
        showLog(row) {
            this.logls = {
                ...this.logls,
                show: true,
                act: '0',
                loading: true,
                opened: false,
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
            this.$refs.buildForm.validate((valid) => {
                if (!valid) { return }

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
                        this.$message.success('操作成功');
                        this.getData();
                        this.buildForm.show = false;
                    }).catch(() => { })
                } else {
                    myAxios.post('/v2/api/repository/deploy_rule/add', formdata).then(res => {
                        this.$message.success('操作成功');
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
            this.$confirm('此操作将永久删除该版本, 是否继续?', '提示', {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning'
            }).then(() => {
                myAxios.post("/v2/api/repository/tags/del", {
                    id: parseInt(this.$route.params.id),
                    tag: tag.TagName
                }).then(res => {
                    this.$message.success("删除成功");
                    this.getData();
                }).catch(() => { });
            })
        },
        delBuild(row) {
            this.$confirm('确定要删除吗, 是否继续?', '提示', {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning'
            }).then(() => {
                myAxios.post('/v2/api/repository/deploy_rule/del', {
                    id: row.id
                }).then(res => {
                    this.$message.success('操作成功');
                    this.getBuild();
                }).catch(() => { })
            }).catch(() => { });
        },
        onSubmit() {
            myAxios.post('/v2/api/repository/edit', { id: parseInt(this.$route.params.id), ...this.form }).then(() => {
                this.$message.success('操作成功');
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
            this.$message.success("复制成功");
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
                this.loading = false;
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
            document.getElementById("term").innerHTML = "";
            this.term = null;
            this.term = new Terminal({
                rendererType: 'dom',
                cursorBlink: false,

            });
            this.term.open(document.getElementById("term"));
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

.backbtn {
    cursor: pointer;
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