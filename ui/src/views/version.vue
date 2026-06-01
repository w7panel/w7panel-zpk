<template>
    <div v-loading="publishGoods.loading" class="content" style="min-height:100vh;box-sizing:border-box;">
        <div>
            <el-breadcrumb separator="/">
                <el-breadcrumb-item :to="{ path: '/zpk' }" style="height:30px;">
                    <template #default><span class="c-99 fw-400">{{ title || '制品库' }}</span></template>
                </el-breadcrumb-item>
                <el-breadcrumb-item style="height:30px;">
                    <template #default><span>版本管理</span></template>
                </el-breadcrumb-item>
            </el-breadcrumb>
        </div>
        <el-tabs v-model="activeTab" class="mt-10" @tab-click="handleTabClick">
            <el-tab-pane label="版本管理" name="version">
                <div>
                    <div>
                        <el-button type="primary"
                            @click="form = { show: true, edit: false, version: '', description: '' }">+
                            新建版本</el-button>
                    </div>
                    <div v-if="list.length" class="mt-20 df">
                        <div class="white-box" style="flex:18;">
                            <div class="c-16 b">当前线上版本</div>
                            <div class="df mt-20">
                                <div style="width:400px;">
                                    <div class="c-66">版本号</div>
                                    <div class="mt-20">
                                        <span class="lh-1 fs-20 b">{{ version.name }}</span>
                                    </div>
                                </div>
                                <div>
                                    <div>
                                        <span class="c-66">发布状态</span>
                                        <span class="ml-20">已发布</span>

                                        <el-tooltip v-if="goods_id" effect="dark" content="应用已发布至微擎云市场"
                                            placement="top-start">
                                            <a class="ml-10 cursor c-blue" target="_blank"
                                                :href="'https://dev.w7.cc/publishgoods/' + goods_id">
                                                <el-icon class="va-middle">
                                                    <MostlyCloudy />
                                                </el-icon>
                                                <span class="ml-4">微擎云市场</span>
                                            </a>
                                        </el-tooltip>
                                    </div>
                                    <div class="mt-20">
                                        <span class="c-66">交付方式</span>
                                        <span v-if="noPlatform" class="ml-20">在线使用（无服务器）</span>
                                        <span v-else class="ml-20">安装部署（有服务器）</span>
                                    </div>
                                    <div class="mt-20">
                                        <span class="c-66">创建时间</span>
                                        <span class="ml-20">{{ version.created_at }}</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <div class="white-box ml-20" style="flex:12;">
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
                            <div v-for="(item, index) in list" :key="index" class="item df">
                                <div style="width:400px;">
                                    <div class="c-66">版本号</div>
                                    <div class="mt-20" style="display: flex; align-items: center;height: 24px;">
                                        <span class="lh-1 fs-20 b">{{ item.name }}</span>

                                        <el-tooltip
                                            v-if="item.id == version.id && goods_id && audit_status && audit_status < 4"
                                            effect="dark" content="应用已发布至微擎云市场，等待管理员审核" placement="top-start">
                                            <span class="ml-10 cursor" style="color:#E6A23C;">待审核</span>
                                        </el-tooltip>

                                        <span v-if="item.id == version.id" class="c-blue"
                                            style="border: 1px solid #0052d9;margin-left: 12px;padding: 1px;">线上版本</span>
                                        <span class="publish-status" @click="toPublish(item)"
                                            :class="{ '-1': 'c-99', 1: 'c-blue', 2: 'c-green', 3: 'c-red' }[item.publish_status]">{{
                                                { '-1': '未发布', 1: '发布中', 2: '已发布', 3: '发布失败' }[item.publish_status]
                                            }}</span>
                                        <span class="publish-status-button c-blue" @click="toPublish(item)"
                                            style="border:1px solid;padding:1px 3px;">点击发布</span>

                                        <template v-if="item.publish_status == 3">
                                            <el-tooltip effect="dark" :content="item.publish_fail_reason"
                                                placement="top">
                                                <el-icon class="va-middle" color="#D00805" :size="16"
                                                    style="margin-left:4px;">
                                                    <Warning />
                                                </el-icon>
                                            </el-tooltip>
                                        </template>
                                    </div>
                                </div>
                                <div class="fc">
                                    <div class="mt-20">
                                        <span class="c-66">创建时间</span>
                                        <span class="ml-20">{{ item.created_at }}</span>
                                    </div>
                                </div>


                                <el-button @click="edit(item)">后端包管理</el-button>
                                <el-button @click="editfront(item)">前端包管理</el-button>
                                <el-button @click="editVersion(item)">版本说明</el-button>


                            </div>
                        </div>
                        <div class="mt-20 df jc-c">
                            <el-pagination v-model:current-page="currentPage" :page-size="10" :total="total"
                                layout="prev, pager, next" background @current-change="getList" />
                        </div>
                    </div>
                </div>
            </el-tab-pane>
            <el-tab-pane label="付费设置" name="paidset">
                <div>
                    <el-form :model="instFee" ref="instFee" label-width="80px" :rules="rules">
                        <el-form-item label="">
                            <div class="df df-c" style="flex:1;">
                                <el-radio-group v-model="instFee.product_type">
                                    <el-radio label="1">按授权付费</el-radio>
                                    <el-radio label="2">按安装付费</el-radio>
                                </el-radio-group>
                                <span v-if="instFee.product_type == '1'" class="c-99">仅针对项目拥有所有权的商家，可按项目授权出售</span>
                                <span v-if="instFee.product_type == '2'"
                                    class="c-99">对该项目熟悉并打包成可用安装包的技术人员，可按安装付费出售</span>
                            </div>
                        </el-form-item>

                        <el-form-item label="售价" prop="service_fee">
                            <el-input v-model="instFee.service_fee" type="number" placeholder="请输入服务费">
                                <template #append>元</template>
                            </el-input>
                        </el-form-item>

                        <el-form-item v-if="instFee.product_type == '1'" label="升级服务">

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
                            <el-button style="width:100%;margin-top:8px;" type="primary"
                                @click="openVersionPrices(instFee.version_prices)">设置升级服务</el-button>
                        </el-form-item>
                        <el-form-item v-if="instFee.product_type == '2'" label="付费升级">
                            <div>
                                <el-switch v-model="instFee.is_free_upgrade"></el-switch>
                                <span class="c-99" style="margin-left:10px;">用户想升级到指定版本，需要付费。</span>
                            </div>
                        </el-form-item>

                        <el-form-item v-if="instFee.product_type == '1'" label="服务周期">
                            <div class="mt-6" style="line-height:18px;">
                                <el-icon class="c-red" style="display:inline-block; vertical-align:middle;">
                                    <Warning />
                                </el-icon>
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
                            <el-button style="width:100%;margin-top:8px;" type="primary"
                                @click="openServicePackages(instFee.service_packages)">设置服务周期</el-button>
                        </el-form-item>

                        <el-form-item label="">
                            <div>
                                <el-button @click="instFee.show = false;">取消</el-button>
                                <el-button type="primary" @click="submitInstFee">确定</el-button>
                            </div>
                        </el-form-item>
                    </el-form>
                </div>

            </el-tab-pane>
            <el-tab-pane label="应用介绍" name="appinfo">
                <description v-if="activeTab == 'appinfo'" :identifie="identifie"></description>
            </el-tab-pane>
            <el-tab-pane label="发布设置" name="publish">
                <publish-settings v-if="activeTab == 'publish'" :identifie="identifie" :userInfo="userInfo"></publish-settings>
            </el-tab-pane>
        </el-tabs>
    </div>
    <el-dialog v-model="form.show" modal-class="createversiondialog" width="640px" title="新建版本">
        <template #header>
            <div class="fs-16 b">{{ form.edit ? '编辑版本' : '新建版本' }}</div>
        </template>
        <el-alert type="warning" class="primary-el-warning" :closable="false" style="line-height:1.5;">
            <div class="b">版本分类标准</div>
            <div>1，格式：主版本号 . 次版本号 . 修订号</div>
            <div>2，小版本：仅修订号变更（例：1.0.0、1.0.2、1.0.10）</div>
            <div>3，大版本：主 / 次版本号变更（例：1.0.0、1.1.2、1.10.2、2.0.0）</div>
            <div class="b mt-6">升级规则</div>
            <div>1，小版本：支持直接跨版升级（例：1.0.0 → 1.0.10 可直接完成）</div>
            <div>2，跨大版本：需逐次升级路径中所有大版本，不可跳过（例：1.0.0 → 1.1.2 → 1.10.2 → 2.0.0，不可跳过中间大版本直接升级）</div>
        </el-alert>
        <div style="margin-top:20px; padding-left:20px;">
            <el-form :model="form" ref="newversionform" label-width="130px" :rules="rules" label-position="left">
                <el-form-item prop="version" label="输入版本号">
                    <el-input :disabled="form.edit" v-model="form.version" placeholder="请输入版本号"
                        style="width:400px;"></el-input>
                </el-form-item>
                <el-form-item prop="description" label="版本说明">
                    <el-input v-model="form.description" placeholder="请输入版本说明" type="textarea" :rows="3"
                        style="width:400px;"></el-input>
                </el-form-item>
            </el-form>
        </div>
        <template #footer>
            <el-button @click="form.show = false;">取消</el-button>
            <el-button type="primary" @click="addVersion">确定</el-button>
        </template>
    </el-dialog>


    <el-dialog v-model="versionPrices.show" width="840px" title="设置升级服务">
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
                        <el-select v-model="item.version" placeholder="请选择">
                            <el-option v-for="vl in instFee.can_upgrade_versions" :key="vl" :value="vl"
                                :label="vl === 9999 ? '其他版本' : (vl + '.*.*')"></el-option>
                        </el-select>
                    </td>
                    <td style="width:300px;">
                        <el-input v-model="item.price" type="number" placeholder="请输入">
                            <template #append>元</template>
                        </el-input>
                    </td>
                    <td>
                        <span class="c-blue cursor" @click="versionPrices.list.splice(index, 1)">删除</span>
                    </td>
                </tr>
                <tr>
                    <td colspan="3">
                        <div class="df ai-c jc-c cursor" @click="versionPrices.list.push({ version: '', price: '' })">添加
                        </div>
                    </td>
                </tr>
            </tbody>
        </table>
        <template #footer>
            <el-button @click="versionPrices.show = false;">取消</el-button>
            <el-button type="primary" @click="submitVersionPrices">确定</el-button>
        </template>
    </el-dialog>


    <el-dialog v-model="service_packages.show" width="840px" title="设置服务周期">

        <div class="df service_packages mt-20">
            <el-form v-for="(item, index) in service_packages.list" label-width="60px" class="fc" :key="index">
                <el-form-item label="" style="margin-bottom:10px;"><span class="fs-16">套餐{{ index + 1
                        }}</span></el-form-item>
                <el-form-item label="价格" style="margin-bottom:10px;">
                    <el-input v-model="item.price" type="number" placeholder="请输入" />
                </el-form-item>
                <el-form-item label="时长" style="margin-bottom:10px;">

                    <el-select v-model="item.month" placeholder="请选择">
                        <el-option label="1年" :value="12"></el-option>
                        <el-option label="2年" :value="24"></el-option>
                        <el-option label="3年" :value="36"></el-option>
                        <el-option label="4年" :value="48"></el-option>
                        <el-option label="5年" :value="60"></el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="" style="margin-bottom:0;">
                    <el-tooltip content="勾选后选择当前套餐赠送，自动取消别套餐。">
                        <el-checkbox v-model="item.is_gift"
                            @change="() => service_packages.list.map((i, id) => { (id != index) ? (i.is_gift = false) : null })">赠送</el-checkbox>
                    </el-tooltip>
                    <el-checkbox v-model="item.enabled">生效</el-checkbox>
                </el-form-item>
            </el-form>
        </div>
        <template #footer>
            <el-button @click="service_packages.show = false;">取消</el-button>
            <el-button type="primary" @click="submitServicePackages">确定</el-button>
        </template>
    </el-dialog>
    <el-dialog v-model="dialogVisible">
        <img w-full :src="dialogImageUrl" alt="Preview Image" />
    </el-dialog>
</template>

<script>
import myAxios from '@/utils'
import versionInfo from '@/components/version-info.vue';
import jsyaml from "js-yaml";
import description from './description.vue';
import publishSettings from './publish-settings.vue';
import userMixin from "@/utils/user-mixin";

export default {
    components: {
        versionInfo,
        description,
        publishSettings,
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
                        validator: (rule, value, callback) => {
                            if (/^\d+\.\d+\.\d+$/.test(value)) {
                                callback()
                            } else {
                                callback(new Error('版本格式有误'))
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
            total: 0
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
        handleTabClick(tab) {
            if (tab.props.name == 'paidset') {
                this.getInstFee();
            } else if (tab.props.name == 'appinfo') {
                this.getInfo();
            } else if (tab.props.name == 'publish') {
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
                    this.$message.success('操作成功');
                    this.publishGoods.loading = false;
                    this.publishGoods.show = false;

                    this.getInfo();
                    this.getList();
                }).catch(() => {
                    this.publishGoods.loading = false;
                })
            } catch {
                this.$message.error('操作失败');
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
            this.$refs.instFee.validate((valid) => {
                if (!valid) { return }

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
                        this.$message.success('操作成功');
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
            this.$refs.newversionform.validate((valid) => {
                if (!valid) { return }

                myAxios.post('/respo/version-add', {
                    identifie: this.identifie,
                    version: this.form.version,
                    description: this.form.description,
                }).then(res => {
                    if (res?.data) {
                        this.$message.success('操作成功');
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
                    this.$message.success('操作成功')
                    this.getInfo();
                    this.getList();
                }
            }).catch((error) => {
                this.publishGoods.loading = false;
                if (error?.response?.data?.error) {
                    this.$message.error(error.response.data.error);
                }

                let str = error?.response?.data?.message;
                if (str) {
                    this.$message.error(str);
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
.gray-box {
    border: 1px solid #E7E7E7;
    background: #FAFAFA;
    padding: 20px;
    border-radius: 8px;
}

.gray-box .item {
    padding: 20px 0;
    border-bottom: 1px solid #E7E7E7;
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
    padding: 20px;
    border-radius: 8px;
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
</style>
<style>
.createversiondialog .el-dialog__body {
    border-top: 1px solid #E7E7E7;
}

.createversiondialog .el-dialog__footer {
    border-top: 1px solid #E7E7E7;
    padding: 16px;
    text-align: left;
    display: flex;
    justify-content: center;
}

.publishigoods-uploadimg .el-upload-list__item-preview {
    display: none !important;
}

.publishigoods-uploadimg .el-upload-list__item-delete {
    margin-left: 0 !important;
}

.primary-el-warning {
    background-color: var(--el-color-primary-light-9) !important;
    color: var(--el-color-primary) !important;
}

.primary-el-warning .el-alert__description {
    color: var(--el-color-primary) !important;
}
</style>
