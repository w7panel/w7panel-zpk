<template>
    <a-spin :loading="publishGoods.loading" class="version-page-spin"
        style="min-height:100vh;box-sizing:border-box;">
        <div class="zpk-page-header">
            <span class="backbtn df ai-c" @click="$router.go(-1)">
                <icon-arrow-left class="backicon" />
            </span>
            <a-breadcrumb>
                <a-breadcrumb-item>
                    <router-link to="/zpk" class="c-99 fw-400">制品管理</router-link>
                </a-breadcrumb-item>
                <a-breadcrumb-item>{{ title || identifie }}</a-breadcrumb-item>
            </a-breadcrumb>
        </div>
        <div class="content version-page-content pt-0">
        <a-tabs v-model:active-key="activeTab" @change="handleTabClick">
            <a-tab-pane key="version" title="版本管理">
                <div>
                    <div>
                        <a-button type="primary"
                            @click="form = { show: true, edit: false, version: '', description: '' }">
                            <template #icon><icon-plus /></template>
                            新建版本</a-button>
                    </div>
                    <div v-if="list.length" class="mt-20 df version-summary">
                        <div class="white-box version-current-card">
                            <div class="c-16 b">当前线上版本</div>
                            <div class="version-current-content">
                                <div class="version-current-primary">
                                    <div class="c-66">版本号</div>
                                    <div class="mt-20">
                                        <span class="lh-1 fs-20 b">{{ version.name }}</span>
                                    </div>
                                </div>
                                <div class="version-current-fields">
                                    <div class="version-detail-row">
                                        <span class="c-66 version-detail-label">发布状态</span>
                                        <span class="version-detail-value">已发布</span>

                                        <a-tooltip v-if="goods_id" content="应用已发布至微擎云市场" position="top">
                                            <a class="ml-10 cursor c-blue" target="_blank"
                                                :href="'https://dev.w7.cc/publishgoods/' + goods_id">
                                                <IconCloud />
                                                <span class="ml-4">微擎云市场</span>
                                            </a>
                                        </a-tooltip>
                                    </div>
                                    <div class="version-detail-row">
                                        <span class="c-66 version-detail-label">交付方式</span>
                                        <span v-if="noPlatform" class="version-detail-value">在线使用（无服务器）</span>
                                        <span v-else class="version-detail-value">安装部署（有服务器）</span>
                                    </div>
                                    <div class="version-detail-row">
                                        <span class="c-66 version-detail-label">创建时间</span>
                                        <span class="version-detail-value">{{ version.created_at }}</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <div class="white-box version-base-card">
                            <div class="c-16 b">基础信息</div>
                            <div class="mt-20">
                                <version-info :identifie="identifie" :info="info"
                                    @refresh="() => { getInfo() }"></version-info>
                            </div>
                        </div>
                    </div>
                    <div class="mt-20 gray-box">
                        <div class="c-16 b">开发版本</div>
                        <div class="mt-10">
                            <div v-for="(item, index) in list" :key="index" class="item version-dev-item">
                                <div class="version-dev-main">
                                    <div class="c-66">版本号</div>
                                    <div class="mt-20" style="display: flex; align-items: center;height: 24px;">
                                        <span class="lh-1 fs-20 b">{{ item.name }}</span>

                                        <a-tooltip
                                            v-if="item.id == version.id && goods_id && audit_status && audit_status < 4"
                                            content="应用已发布至微擎云市场，等待管理员审核" position="top">
                                            <span class="ml-10 cursor" style="color:#E6A23C;">待审核</span>
                                        </a-tooltip>

                                        <span v-if="item.id == version.id" class="c-blue"
                                            style="border: 1px solid #0052d9;margin-left: 12px;padding: 1px;">线上版本</span>
                                        <span class="publish-status" @click="toPublish(item)"
                                            :class="{ '-1': 'c-99', 1: 'c-blue', 2: 'c-green', 3: 'c-red' }[item.publish_status]">{{
                                                { '-1': '未发布', 1: '发布中', 2: '已发布', 3: '发布失败' }[item.publish_status]
                                            }}</span>
                                        <span class="publish-status-button c-blue" @click="toPublish(item)"
                                            style="border:1px solid;padding:1px 3px;">点击发布</span>

                                        <template v-if="item.publish_status == 3">
                                            <a-tooltip :content="item.publish_fail_reason" position="top">
                                                <span class="warning-icon va-middle" style="margin-left:4px;">!</span>
                                            </a-tooltip>
                                        </template>
                                    </div>
                                </div>
                                <div class="version-dev-meta">
                                    <div class="mt-20">
                                        <span class="c-66">创建时间</span>
                                        <span class="ml-20">{{ item.created_at }}</span>
                                    </div>
                                </div>


                                <div class="version-dev-actions">
                                    <a-button @click="edit(item)">
                                        后端包管理
                                    </a-button>
                                    <a-button @click="editfront(item)">
                                        前端包管理
                                    </a-button>
                                    <a-button @click="editVersion(item)">
                                        版本说明
                                    </a-button>
                                </div>


                            </div>
                        </div>
                        <div v-if="total > 10" class="mt-20 df jc-c">
                            <a-pagination v-model:current="currentPage" :page-size="10" :total="total"
                                @change="getList" />
                        </div>
                    </div>
                </div>
            </a-tab-pane>
            <a-tab-pane key="paidset" title="付费设置">
                <div>
                    <a-form :model="instFee" ref="instFee" :rules="rules" label-align="left"
                        class="version-paid-form"
                        :label-col-props="{ flex: '0 0 72px' }" :wrapper-col-props="{ flex: '1' }">
                        <a-form-item label="">
                            <div class="df df-c" style="flex:1;">
                                <a-radio-group v-model="instFee.product_type">
                                    <a-radio value="1">按授权付费</a-radio>
                                    <a-radio value="2">按安装付费</a-radio>
                                </a-radio-group>
                                <span v-if="instFee.product_type == '1'" class="c-99">仅针对项目拥有所有权的商家，可按项目授权出售</span>
                                <span v-if="instFee.product_type == '2'"
                                    class="c-99">对该项目熟悉并打包成可用安装包的技术人员，可按安装付费出售</span>
                            </div>
                        </a-form-item>

                        <a-form-item label="售价" field="service_fee">
                            <a-input v-model="instFee.service_fee" type="number" placeholder="请输入服务费">
                                <template #append>元</template>
                            </a-input>
                        </a-form-item>

                        <a-form-item v-if="instFee.product_type == '1'" label="升级服务">
                            <div class="version-setting-block">
                                <table class="table mt-10">
                                    <thead>
                                        <tr>
                                            <td>版本号</td>
                                            <td>价格</td>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        <tr v-for="(item, index) in instFee.version_prices" :key="index">
                                            <td>{{ item.version === 9999 ? '其他版本' : item.version ? (item.version +
                                                '.*.*')
                                                : '' }}</td>
                                            <td>￥{{ item.price }}</td>
                                        </tr>
                                        <tr v-if="!instFee.version_prices.length">
                                            <td colspan="3" class="c-99 txt-c">暂无数据</td>
                                        </tr>
                                    </tbody>
                                </table>
                                <a-button style="width:100%;margin-top:8px;" type="primary"
                                    @click="openVersionPrices(instFee.version_prices)">设置升级服务</a-button>
                            </div>
                        </a-form-item>
                        <a-form-item v-if="instFee.product_type == '2'" label="付费升级">
                            <div>
                                <a-switch v-model="instFee.is_free_upgrade" />
                                <span class="c-99" style="margin-left:10px;">用户想升级到指定版本，需要付费。</span>
                            </div>
                        </a-form-item>

                        <a-form-item v-if="instFee.product_type == '1'" label="服务周期">
                            <div class="version-setting-block">
                                <div class="mt-6" style="line-height:18px;">
                                    <span class="warning-icon" style="display:inline-block; vertical-align:middle;">!</span>
                                    <span class="c-99"
                                        style="margin-left:4px; vertical-align:middle;">到期后无法维护更新,需要再次购买服务周期套餐才可以维护更新</span>
                                </div>
                                <table class="table mt-10">
                                    <thead>
                                        <tr>
                                            <td>价格</td>
                                            <td>时间</td>
                                            <td>生效</td>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        <tr v-for="(item, index) in instFee.service_packages" :key="index">
                                            <td>￥{{ item.price }}</td>
                                            <td>{{ item.month / 12 }}年</td>
                                            <td>{{ item.enabled == 2 ? '是' : '否' }}</td>
                                        </tr>
                                        <tr v-if="!instFee.service_packages.length">
                                            <td colspan="3" class="c-99 txt-c">暂无数据</td>
                                        </tr>
                                    </tbody>
                                </table>
                                <a-button style="width:100%;margin-top:8px;" type="primary"
                                    @click="openServicePackages(instFee.service_packages)">设置服务周期</a-button>
                            </div>
                        </a-form-item>

                        <a-form-item label="">
                            <div>
                                <a-button @click="instFee.show = false;">取消</a-button>
                                <a-button type="primary" @click="submitInstFee">确定</a-button>
                            </div>
                        </a-form-item>
                    </a-form>
                </div>

            </a-tab-pane>
            <a-tab-pane key="appinfo" title="应用介绍">
                <description v-if="activeTab == 'appinfo'" :identifie="identifie"></description>
            </a-tab-pane>
            <a-tab-pane key="publish" title="发布设置">
                <publish-settings v-if="activeTab == 'publish'" :identifie="identifie" :userInfo="userInfo"></publish-settings>
            </a-tab-pane>
        </a-tabs>
        </div>
    </a-spin>
    <a-modal v-model:visible="form.show" :width="640" :title="form.edit ? '编辑版本' : '新建版本'" :footer="false"
        modal-class="createversiondialog">
        
        <a-alert type="info" class="zpk-primary-alert" :closable="false">
            <div class="arco-alert-title">版本分类标准</div>
            <div class="registry-alert-item">1，格式：主版本号 . 次版本号 . 修订号</div>
            <div class="registry-alert-item">2，小版本：仅修订号变更（例：1.0.0、1.0.2、1.0.10）</div>
            <div class="registry-alert-item">3，大版本：主 / 次版本号变更（例：1.0.0、1.1.2、1.10.2、2.0.0）</div>
            <div class="arco-alert-title mt-6">升级规则</div>
            <div class="registry-alert-item">1，小版本：支持直接跨版升级（例：1.0.0 → 1.0.10 可直接完成）</div>
            <div class="registry-alert-item">2，跨大版本：需逐次升级路径中所有大版本，不可跳过（例：1.0.0 → 1.1.2 → 1.10.2 → 2.0.0，不可跳过中间大版本直接升级）</div>
        </a-alert>
        <div style="margin-top:20px; padding-left:20px;">
            <a-form :model="form" ref="newversionform" :rules="rules" label-align="left" class="version-create-form"
                :label-col-props="{ span: 5, flex: '0 0 130px' }" :wrapper-col-props="{ span: 19, flex: '1' }">
                <a-form-item field="version" label="输入版本号">
                    <a-input :disabled="form.edit" v-model="form.version" placeholder="请输入版本号" style="width:400px;" />
                </a-form-item>
                <a-form-item field="description" label="版本说明">
                    <a-textarea v-model="form.description" placeholder="请输入版本说明" :rows="3" style="width:400px;" />
                </a-form-item>
            </a-form>
        </div>
        <div class="dialog-footer version-create-footer">
            <a-button @click="form.show = false;">取消</a-button>
            <a-button type="primary" @click="addVersion">确定</a-button>
        </div>
    </a-modal>


    <a-modal v-model:visible="versionPrices.show" :width="840" title="设置升级服务" :footer="false">
        <table class="table">
            <thead>
                <tr>
                    <td>版本</td>
                    <td>价格</td>
                    <td>操作</td>
                </tr>
            </thead>
            <tbody>
                <tr v-for="(item, index) in versionPrices.list" :key="index">
                    <td>
                        <a-select v-model="item.version" placeholder="请选择">
                            <a-option v-for="vl in instFee.can_upgrade_versions" :key="vl" :value="vl"
                                :label="vl === 9999 ? '其他版本' : (vl + '.*.*')"></a-option>
                        </a-select>
                    </td>
                    <td style="width:300px;">
                        <a-input v-model="item.price" type="number" placeholder="请输入">
                            <template #append>元</template>
                        </a-input>
                    </td>
                    <td>
                        <span class="c-blue cursor" @click="versionPrices.list.splice(index, 1)">删除</span>
                    </td>
                </tr>
                <tr>
                    <td colspan="3">
                        <div class="df ai-c jc-c cursor" @click="versionPrices.list.push({ version: '', price: '' })">
                            <span class="addmenu"><icon-plus />添加</span>
                        </div>
                    </td>
                </tr>
            </tbody>
        </table>
        <div class="dialog-footer" style="justify-content: center;margin-top: 20px;">
            <a-button @click="versionPrices.show = false;">取消</a-button>
            <a-button type="primary" @click="submitVersionPrices">确定</a-button>
        </div>
    </a-modal>


    <a-modal v-model:visible="service_packages.show" :width="840" title="设置服务周期" :footer="false">

        <div class="df service_packages">
            <a-form v-for="(item, index) in service_packages.list" :model="item" label-align="left"
                :label-col-props="{ span: 6, flex: '0 0 60px' }" :wrapper-col-props="{ span: 18, flex: '1' }"
                class="fc" :key="index">
                <a-form-item label="" style="margin-bottom:10px;"><span class="fs-16">套餐{{ index + 1
                        }}</span></a-form-item>
                <a-form-item label="价格" style="margin-bottom:10px;">
                    <a-input v-model="item.price" type="number" placeholder="请输入" />
                </a-form-item>
                <a-form-item label="时长" style="margin-bottom:10px;">

                    <a-select v-model="item.month" placeholder="请选择">
                        <a-option label="1年" :value="12"></a-option>
                        <a-option label="2年" :value="24"></a-option>
                        <a-option label="3年" :value="36"></a-option>
                        <a-option label="4年" :value="48"></a-option>
                        <a-option label="5年" :value="60"></a-option>
                    </a-select>
                </a-form-item>
                <a-form-item label="" style="margin-bottom:0;">
                    <a-tooltip content="勾选后选择当前套餐赠送，自动取消别套餐。">
                        <a-checkbox v-model="item.is_gift"
                            @change="() => service_packages.list.map((i, id) => { (id != index) ? (i.is_gift = false) : null })">赠送</a-checkbox>
                    </a-tooltip>
                    <a-checkbox v-model="item.enabled">生效</a-checkbox>
                </a-form-item>
            </a-form>
        </div>
        <div class="dialog-footer" style="justify-content: center;margin-top: 20px;">
            <a-button @click="service_packages.show = false;">取消</a-button>
            <a-button type="primary" @click="submitServicePackages">确定</a-button>
        </div>
    </a-modal>
    <a-modal v-model:visible="dialogVisible" :footer="false">
        <img w-full :src="dialogImageUrl" alt="Preview Image" />
    </a-modal>
</template>

<script>
import myAxios from '@/utils'
import versionInfo from '@/components/version-info.vue';
import jsyaml from "js-yaml";
import description from './description.vue';
import publishSettings from './publish-settings.vue';
import userMixin from "@/utils/user-mixin";
import { messageError, messageSuccess } from '@/utils/ui-feedback';
import { IconArrowLeft, IconPlus, IconCloud } from '@arco-design/web-vue/es/icon';

export default {
    components: {
        versionInfo,
        description,
        publishSettings,
        IconArrowLeft,
        IconPlus,
        IconCloud,
    },
    mixins: [userMixin],
    data() {
        return {
            activeTab: 'version',
            identifie: '',
            list: [],
            versionsKV: {},
            form: {
                show: false,
                edit: false,
                version: '',
                description: '',
            },
            version: {},
            rules: {
                version: [
                    { required: true, message: '请输入版本号', trigger: 'blur' },
                    {
                        validator: (value, callback) => {
                            if (/^\d+\.\d+\.\d+$/.test(value)) {
                                callback()
                            } else {
                                callback('版本格式有误')
                            }
                        }, trigger: 'blur'
                    },
                ],
                description: [{ required: true, message: '请输入版本描述', trigger: 'blur' },],
                service_fee: [{ required: true, message: '请输入费用', trigger: 'blur' },],
            },
            pgRules: {
                label_ids: [{ required: true, message: '请选择标签', trigger: 'blur' },],
                goods_imgs: [{ required: true, message: '请上传图片', trigger: 'blur' },],
            },

            info: {},

            instFee: {
                show: false,
                is_free_upgrade: false,
                product_type: '1',
                service_fee: '',
                service_packages: [],
                version_prices: [],
                can_upgrade_versions: [],
            },
            versionPrices: {
                show: false,
                list: [],
            },
            service_packages: {
                show: false,
                list: [
                    { price: '', month: '', is_gift: false, enabled: false },
                    { price: '', month: '', is_gift: false, enabled: false },
                    { price: '', month: '', is_gift: false, enabled: false },
                ],
            },







            publishGoods: {
                show: false,
                loading: false,
                loadingTxt: '',

                identifie: '',
                version: '',
                logo: '',
                goods_imgs: [],
                label_ids: [],
                backend_attach_md5: '',
                frontend_attach_md5: '',
            },


            is_register: false,

            goods_id: '',
            audit_status: 0,

            noPlatform: true,
            currentPage: 1,
            total: 0,
            dialogVisible: false,
            dialogImageUrl: '',
        }
    },
    created() {
        this.is_register = window?.$wujie?.props?.isRegister;
        this.identifie = this.$route.query.id;
        this.title = this.$route.query.title;
        this.getInfo();
        this.getList();
    },
    methods: {
        handleTabClick(key) {
            if (key == 'paidset') {
                this.getInstFee();
            } else if (key == 'appinfo') {
                this.getInfo();
            } else if (key == 'publish') {
                this.getPublishInfo();
            }
        },
        getCuv() {
            myAxios.post('/respo/goods/can-upgrade-versions', {
                identifie: this.identifie,
            }).then(res => {
                this.instFee.can_upgrade_versions = res?.data?.data || [];
            })
        },


        openPublishGoods(row) {
            this.publishGoods = {
                show: true,
                loading: false,
                loadingTxt: '',

                identifie: this.identifie,
                version: row.name,
                logo: '',
                goods_imgs: [],
                label_ids: [],
                backend_attach_md5: '',
                frontend_attach_md5: '',
            }
            this.submitPublishGoods();
        },
        async submitPublishGoods() {
            this.publishGoods.loading = true;
            try {
                let icon = await myAxios.get('/respo/v2/info/' + this.identifie + '/' + this.publishGoods.version).then(res => res?.data?.data?.icon_url);
                let iconfile = await this.downloadFileAsFile(icon);
                let uploadIcon = await this.uploadImg({ file: iconfile });

                await myAxios.post('/respo/goods/publish', {
                    identifie: this.identifie,
                    version: this.publishGoods.version,
                    logo: uploadIcon
                }).then(res => {
                    messageSuccess('操作成功');
                    this.publishGoods.loading = false;
                    this.publishGoods.show = false;

                    this.getInfo();
                    this.getList();
                }).catch(() => {
                    this.publishGoods.loading = false;
                })
            } catch {
                messageError('操作失败');
                this.publishGoods.loading = false;
            }
        },


        openVersionPrices(data) {
            let list = [];
            if (data) {
                data = JSON.parse(JSON.stringify(data));
                list = data || [];
            }
            this.versionPrices = {
                show: true,
                list: list,
            };
        },
        submitVersionPrices() {
            let o = {};
            let arr = [];
            this.versionPrices.list.map(i => {
                if (!i.version || !i.price) { return }
                o[i.version] = {
                    version: Number(i.version),
                    versionName: this.versionsKV[i.version],
                    price: Number(i.price),
                };
            });
            for (let i in o) {
                arr.push(o[i]);
            }
            this.instFee.version_prices = arr;
            this.versionPrices.show = false;
        },


        openServicePackages(data) {
            let list = [
                { price: '', month: '', enabled: false },
                { price: '', month: '', enabled: false },
                { price: '', month: '', enabled: false },
            ]
            if (data) {
                data = JSON.parse(JSON.stringify(data));
                data?.map((item, index) => {
                    list[index] = {
                        ...item,
                        enabled: Boolean(item.enabled == 2),
                    }
                });
            }
            this.service_packages = {
                show: true,
                list: list,
            }
        },
        submitServicePackages() {
            let list = this.service_packages?.list?.filter(i => i.price !== '' && i.month)?.map(i => ({
                ...i,
                price: Number(i.price),
                month: Number(i.month),
                enabled: i.enabled ? 2 : 1,
            }));
            this.instFee.service_packages = list;
            this.service_packages.show = false;
        },



        getInstFee() {
            let version_prices = this.info?.version_prices || [];
            version_prices.map(i => {
                i.versionName = this.versionsKV[i.version];
            })
            let product_type = 1;
            if (this.info?.product_type == 1 || this.info?.product_type == 2) {
                product_type = this.info?.product_type;
            }
            this.instFee = {
                ...this.instFee,
                product_type: String(product_type),
                old_fee: this.info?.install_service_fee,
                service_fee: this.info?.install_service_fee || '',
                service_packages: this.info?.service_packages || [],
                is_free_upgrade: this.info?.is_free_upgrade || false,
                version_prices: version_prices,
            }
        },
        submitInstFee() {
            this.$refs.instFee.validate((errors) => {
                if (errors) { return }

                myAxios.post('/respo/goods/set-service-fee', {
                    identifie: this.identifie,
                    product_type: Number(this.instFee.product_type),
                    service_fee: Number(this.instFee.service_fee),
                    is_free_upgrade: this.instFee.is_free_upgrade ? 1 : 0,

                    ...(this.instFee.product_type == '1' ? {
                        service_packages: this.instFee.service_packages,
                        version_prices: this.instFee.version_prices,
                    } : {
                        version_prices: [],
                    })
                }).then(res => {
                    if (this.goods_id && (Number(this.instFee.service_fee) != Number(this.instFee.old_fee)) && this.list.length) {
                        let item = this.list.find(i => i.id === this.version.id);
                        if (item) { this.toPublish(item); }
                    } else {
                        messageSuccess('操作成功');
                    }
                    this.instFee.show = false;
                    this.getInfo();
                });
            });
        },

        edit(item) {
            this.$router.push('/zpk-edit?id=' + this.identifie + '&versionid=' + (item.name || ''));
        },
        editfront(item) {
            this.$router.push('/zpk-editfront?id=' + this.identifie + '&versionid=' + (item.name || ''));
        },
        editDescription() {
            this.$router.push('/zpk-description?id=' + this.identifie);
        },
        editPublish() {
            this.$router.push('/zpk-publish?id=' + this.identifie);
        },
        getInfo() {
            myAxios.get('/respo/info/' + this.identifie).then(res => {

                this.version = res?.data?.data?.version || {};
                this.version.created_at = this.version.created_at || "";
                this.version.created_at = new Date(new Date(this.version.created_at || "").getTime()).toISOString().replace(/T|Z/g, ' ').replace(/\.\d+/g, '').trim()

                this.info = res?.data?.data;
                this.info.is_free_upgrade = !!this.info.is_free_upgrade;
                if (!this.info.manifest) {
                    this.noPlatform = true;
                } else {
                    let json = jsyaml.load(this.info.manifest);
                    this.noPlatform = !json.platform;
                }

                let goodsid = res?.data?.data?.goods_id;
                if (!goodsid) { return }
                this.goods_id = goodsid;
                myAxios.post('/respo/goods/info', { identifie: this.identifie }).then(res => {
                    let audit_status = res?.data?.data?.audit_status;
                    this.audit_status = audit_status;
                })
            })
        },
        getList(page = 1) {
            this.currentPage = page
            myAxios.post('/respo/version-list', { identifie: this.identifie, page: this.currentPage }).then(res => {
                this.list = res?.data?.data?.list || [];
                this.total = res?.data?.data?.total || 0
                this.list.forEach(i => {
                    i.created_at = i.created_at || "";
                    i.created_at = i.created_at.replace(/[a-zA-Z]/g, ' ').replace(/\s+$/, '');
                });

                let versions = {};
                this.list.map(i => {
                    versions[i.id] = i.name;
                })
                this.versionsKV = versions;
            })
            this.getCuv();
        },



        editVersion(item) {
            this.form.show = true;
            this.form.edit = true;
            this.form.version = item.name;
            this.form.description = item.description;
        },
        addVersion() {
            this.$refs.newversionform.validate((errors) => {
                if (errors) { return }

                myAxios.post('/respo/version-add', {
                    identifie: this.identifie,
                    version: this.form.version,
                    description: this.form.description,
                }).then(res => {
                    if (res?.data) {
                        messageSuccess('操作成功');
                        this.getList();
                        this.form.show = false;
                    }
                });

            });

        },

        toPublish(item) {
            this.publishGoods.loading = true;
            myAxios.post('/respo/publish', {
                identifie: this.identifie,
                version: item.name
            }, {
                timeout: 0
            }).then(res => {
                this.publishGoods.loading = false;
                if (this.is_register && this.info.install_service_fee !== 0) {
                    this.openPublishGoods(item);
                } else {
                    messageSuccess('操作成功')
                    this.getInfo();
                    this.getList();
                }
            }).catch((error) => {
                this.publishGoods.loading = false;
                if (error?.response?.data?.error) {
                    messageError(error.response.data.error);
                }

                let str = error?.response?.data?.message;
                if (str) {
                    messageError(str);
                }
            });
        },
        async uploadImg({ file, js_ticket, host }) {
            let formData = new FormData();
            formData.append('file', file);
            formData.append('file_name', file.name);
            let img = await myAxios.post('/respo/attach/upload-img', formData).then(res => {
                if (!res?.data) { return }
                let img = res.data?.data?.attach?.path || '';
                return img;
            });
            return img;
        },
        async downloadFileAsFile(url, filename = '') {
            const response = await fetch(url);
            if (!response.ok) throw new Error(`下载失败: ${response.status}`);

            const contentDisposition = response.headers.get('Content-Disposition');
            const contentType = response.headers.get('Content-Type');


            const imageMimeTypes = {
                'image/jpeg': '.jpg',
                'image/png': '.png',
                'image/gif': '.gif',
                'image/webp': '.webp',
                'image/svg+xml': '.svg',
                'image/bmp': '.bmp',
            };


            if (!filename && contentDisposition) {
                const matches = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
                if (matches?.[1]) {
                    filename = matches[1].replace(/['"]/g, '');
                }
            }


            if (!filename) {
                filename = new URL(url).pathname.split('/').pop() || 'image';
            }

            const blob = await response.blob();


            if (!filename.includes('.')) {

                if (contentType && imageMimeTypes[contentType]) {
                    filename += imageMimeTypes[contentType];
                }

                else {
                    const ext = await this.detectImageExtension(blob);
                    if (ext) {
                        filename += ext;
                    } else {

                        filename += '.jpg';
                    }
                }
            }
            return new File([blob], filename, { type: contentType });
        },
        async detectImageExtension(blob) {
            return new Promise((resolve) => {
                const reader = new FileReader();
                reader.onloadend = () => {
                    const buffer = reader.result;
                    const view = new DataView(buffer);


                    if (view.byteLength >= 3 &&
                        view.getUint8(0) === 0xFF &&
                        view.getUint8(1) === 0xD8 &&
                        view.getUint8(2) === 0xFF) {
                        resolve('.jpg');
                    }

                    else if (view.byteLength >= 8 &&
                        view.getUint32(0) === 0x89504E47 &&
                        view.getUint32(4) === 0x0D0A1A0A) {
                        resolve('.png');
                    }

                    else if (view.byteLength >= 6 &&
                        view.getUint32(0) === 0x47494638 &&
                        (view.getUint16(4) === 0x3761 || view.getUint16(4) === 0x3961)) {
                        resolve('.gif');
                    }

                    else if (view.byteLength >= 12 &&
                        view.getUint32(0) === 0x52494646 &&
                        view.getUint32(8) === 0x57454250) {
                        resolve('.webp');
                    }

                    else if (view.byteLength >= 4) {
                        const text = new TextDecoder().decode(buffer.slice(0, 200));
                        if (text.includes('<svg')) {
                            resolve('.svg');
                        }
                    }

                    resolve('');
                };
                reader.readAsArrayBuffer(blob.slice(0, 200));
            });
        },
    },
}
</script>

<style scoped>
.version-page-spin {
    display: block;
    width: 100%;
}

.gray-box {
    border: 1px solid #E7E7E7;
    background: #fff;
    padding: 20px;
    border-radius: 8px;
}

.version-summary {
    align-items: stretch;
    gap: 24px;
}

.version-current-card {
    flex: 1 1 620px;
}

.version-base-card {
    flex: 0 0 480px;
}

.version-current-content {
    display: grid;
    grid-template-columns: minmax(220px, 1fr) minmax(300px, 1.2fr);
    gap: 32px;
    margin-top: 24px;
}

.version-current-primary {
    min-width: 0;
}

.version-current-fields {
    display: flex;
    flex-direction: column;
    gap: 18px;
    min-width: 0;
}

.version-detail-row {
    display: flex;
    align-items: flex-start;
    min-width: 0;
    line-height: 22px;
}

.version-detail-label {
    flex: 0 0 72px;
    white-space: nowrap;
}

.version-detail-value {
    min-width: 0;
    margin-left: 24px;
    word-break: keep-all;
    white-space: nowrap;
}

.gray-box .item,
.version-dev-item {
    padding: 20px 0;
    border-bottom: 1px solid #E7E7E7;
}

.version-dev-item {
    display: flex;
    align-items: center;
    gap: 28px;
}

.version-dev-main {
    flex: 0 0 400px;
    min-width: 0;
}

.version-dev-meta {
    flex: 1 1 auto;
    min-width: 260px;
}

.version-dev-actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    flex: 0 0 auto;
    margin-left: auto;
}

.gray-box .item:last-child {
    border-bottom: 0;
}

.gray-box .item:hover .publish-status {
    display: none;
}

.gray-box .item:hover .publish-status-button {
    display: inline;
}

.white-box {
    border: 1px solid #E7E7E7;
    background: #fff;
    padding: 22px 24px;
    border-radius: 8px;
    min-width: 0;
}

.table {
    width: 100%;
}

.version-setting-block {
    width: 100%;
}

.version-paid-form :deep(.arco-form-item-label-col) {
    flex: 0 0 72px !important;
    width: 72px;
}

.version-paid-form :deep(.arco-form-item-wrapper-col) {
    flex: 1 1 auto;
    min-width: 0;
}

.table td {
    padding: 10px;
    line-height: 1.4;
    border: 1px solid #cccccc;
    border-left: 0;
    border-right: 0;
}

.table thead tr:first-child td {
    background: #f3f3f3;
    border-top: 0;
}

.service_packages .fc {
    padding: 0 16px;
}

.service_packages .fc+.fc {
    border-left: 1px solid #f1f1f1;
}

.publish-status {
    margin-left: 10px;
    cursor: pointer;
}

.publish-status-button {
    margin-left: 10px;
    cursor: pointer;
    display: none;
}

.warning-icon {
    color: #D00805;
    font-size: 16px;
    font-weight: 700;
    line-height: 1;
}

.cloud-icon {
    font-size: 14px;
    line-height: 1;
}

@media (max-width: 1040px) {
    .version-summary {
        flex-direction: column;
    }

    .version-base-card {
        flex-basis: auto;
    }

    .version-current-content {
        grid-template-columns: 1fr;
    }
}
</style>
<style>
.primary-arco-warning {
    background-color: #eef4ff !important;
    color: #0052D9 !important;
}

.primary-arco-warning .arco-alert-content {
    color: #0052D9 !important;
}
</style>
