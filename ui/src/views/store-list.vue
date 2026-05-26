<template>
    <div class="padding-20 bg-white">
        <div>
            <div style="margin-bottom: 20px;display: flex;justify-content: space-between;">
                <el-input v-if="activeTab === 'store'" class="dark-search" v-model="search.keyword"
                    :style="{ width: '256px' }" :placeholder="activeTab === 'store' ? '输入应用名称搜索' : '输入已购买应用名称搜索'"
                    @keyup.enter="toSearch">
                    <template #suffix>
                        <el-icon class="el-input__icon" :style="{ cursor: 'pointer' }" @click="toSearch">
                            <search />
                        </el-icon>
                    </template>
                </el-input>

                <div v-if="activeTab === 'purchased'">
                    <span style="margin-right: 10px;">应用状态</span>
                    <el-select v-model="search.enable_status" :style="{ width: '120px' }" placeholder="应用状态"
                        @change="toSearch">
                        <el-option label="全部" :value="null"></el-option>
                        <el-option label="有效" :value="1"></el-option>
                        <el-option label="失效" :value="0"></el-option>
                    </el-select>
                </div>
                <div class="tab-group" v-if="panelToken">
                    <span class="tab-item" :class="{ active: activeTab === 'store' }"
                        @click="switchTab('store')">制品商店</span>
                    <span class="tab-item" :class="{ active: activeTab === 'purchased' }"
                        @click="switchTab('purchased')">已购买应用</span>
                </div>
            </div>
            <div v-if="activeTab === 'store' && !hidetags" class="labelbox df">
                <div :class="allTag ? 'w-100 df df-ww' : 'one-hide f1'">
                    <span class="label" :class="{ active: search.tag == '' }"
                        @click="search.tag = ''; toSearch();">全部</span>
                    <span v-for="(item, index) in tags" :key="index" class="label"
                        :class="{ active: search.tag == item.name }" @click="search.tag = item.name; toSearch()">{{
                            item.name }}</span>
                    <div class="shouqi">
                        <span v-if="allTag" class="cursor alltag c-99 df ai-c ml-10" @click="allTag = !allTag">
                            <span class="va-middle">收起</span>
                        </span>
                    </div>
                </div>
                <span v-if="!allTag" class="cursor fa-14 alltag c-99 df ai-c ml-10" @click="allTag = !allTag">
                    <span class="va-middle">查看全部</span>
                </span>
            </div>
        </div>

        <template v-if="activeTab === 'store'">
            <el-row class="mt-24" :gutter="24">
                <el-col :span="8" v-for="(item, index) in list" :key="item.identifie" style="margin-bottom:24px;">
                    <respo-item :webUrl="webUrl" :data="item" @delete="list.splice(index, 1)" @refresh="getData"
                        @tagClick="tagClick" @goInstall="goInstall"></respo-item>
                </el-col>
            </el-row>
            <div v-if="list && list.length" class="df jc-c" style="padding-bottom:20px;">
                <el-pagination v-model:current-page="page" @current-change="getData" :page-size="limit" :total="total"
                    layout="prev, pager, next" />
            </div>
        </template>

        <template v-else>
            <el-table class="table-header table-respo-list" :data="purchasedList" style="width: 100%">
                <el-table-column prop="formula_name" label="应用名称">
                </el-table-column>

                <el-table-column prop="order_sn" label="订单编号">
                </el-table-column>
                <el-table-column prop="created_at" label="购买时间">
                </el-table-column>
                <el-table-column prop="service_expire_time" label="到期时间">
                    <template #default="scope">
                        {{ scope.row.service_expire_time === '0001-01-01 00:00:00' ? '-' :
                            scope.row.service_expire_time }}
                    </template>
                </el-table-column>
                <el-table-column prop="" label="操作" width="180">
                    <template #default="scope">
                        <template v-if="scope.row.product_type === 2">
                            <template v-if="scope.row.goods_id > 0">
                                <el-button type="text"
                                    v-if="scope.row.formula_latest_version > scope.row.formula_version && scope.row.is_free_upgrade > 0"
                                    @click="handleUpgrade(scope.row)">升级</el-button>
                            </template>
                            <el-tooltip v-if="scope.row.used_time" content="授权已被使用" placement="top">
                                <el-button type="text" disabled>安装</el-button>
                            </el-tooltip>
                            <el-button type="text" v-else
                                @click="handleInstall(scope.row.order_sn, scope.row.formula_identifie)">安装</el-button>
                        </template>
                        <template v-else>
                            <template v-if="scope.row.goods_id > 0">
                                <el-button type="text" v-if="scope.row.canUpgrade"
                                    @click="handleUpgrade(scope.row)">升级</el-button>
                                <el-button type="text"
                                    v-if="scope.row.service_packages && scope.row.service_packages.length > 0"
                                    @click="handleRenew(scope.row)">续费</el-button>
                            </template>

                            <el-tooltip v-if="scope.row.used_time" content="授权已被使用" placement="top">
                                <el-button type="text" disabled>安装</el-button>
                            </el-tooltip>
                            <el-button type="text" v-else
                                @click="handleInstall(scope.row.order_sn, scope.row.formula_identifie)">安装</el-button>
                        </template>
                    </template>
                </el-table-column>
            </el-table>
            <div v-if="purchasedList && purchasedList.length" class="df jc-c" style="padding-bottom:20px;">
                <el-pagination v-model:current-page="purchasedPage" @current-change="getPurchasedData"
                    :page-size="purchasedLimit" :total="purchasedTotal" layout="prev, pager, next" />
            </div>
        </template>
        <el-dialog v-model="renewDialogVisible" title="续费服务" width="520px" destroy-on-close>
            <div v-if="renewSite">
                <el-form label-width="90px" label-position="left">
                    <el-form-item label="服务周期" v-if="renewSite.service_packages && renewSite.service_packages.length">
                        <el-radio-group v-model="renewServicePackageId">
                            <template v-for="(item, key) in renewSite.service_packages" :key="key">
                                <el-radio class="el-radio--w7" :label="item.id" border>
                                    <span class="price-tag">{{ item.month / 12 }}年</span>
                                </el-radio>
                            </template>
                        </el-radio-group>
                    </el-form-item>
                    <el-form-item label="续费价格">
                        <span style="color: #2d5fff;font-size:24px;font-weight:500;">¥{{ renewPrice }}</span>
                    </el-form-item>
                </el-form>
            </div>
            <template #footer>
                <el-button @click="renewDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="submitRenew">确认续费</el-button>
            </template>
        </el-dialog>
        <el-dialog top="5vh" v-model="payDialogVisible" title="支付" width="800px" destroy-on-close>
            <iframe
                :src="`https://ip.w7.cc/pay/${ticket}?header=false&footer=false&paid_callback=https%3A%2F%2Fuser.w7.cc%2Forder`"
                width="100%" height="650px" frameborder="0"></iframe>
        </el-dialog>
        <el-dialog v-model="upgradeDialogVisible" title="升级版本" width="520px" destroy-on-close>
            <div v-if="upgradeSite">
                <el-form label-width="90px" label-position="left">
                    <el-form-item label="升级版本">
                        <el-radio-group v-model="upgradeTargetVersionId" @change="handleUpgradeVersionChange">
                            <el-radio v-for="vp in availableUpgradeVersions" :key="vp.id" class="el-radio--w7"
                                :label="vp.id" border>{{ vp.version === 9999 ? '其他版本' : 'V' + vp.version }}
                            </el-radio>
                        </el-radio-group>
                    </el-form-item>
                    <el-form-item label="升级价格">
                        <span style="color: #2d5fff;font-size:24px;font-weight:500;">¥{{ upgradePrice }}</span>
                    </el-form-item>
                </el-form>
            </div>
            <template #footer>
                <el-button @click="upgradeDialogVisible = false">取消</el-button>
                <el-button type="primary" @click="submitUpgrade">确认升级</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script>
import myAxios from '@/utils';
import panelAxios from '@/utils/panel';
import respoItem from '@/components/respo-item.vue'

export default {
    name: "store",
    data() {
        return {
            search: {
                keyword: "",
                tag: "",
                enable_status: 1
            },
            activeTab: 'store',
            list: [],
            total: 1,
            page: 1,
            limit: 9,
            tags: [],
            allTag: true,
            hidetags: false,
            webUrl: '',
            purchasedList: [],
            purchasedTotal: 0,
            purchasedPage: 1,
            purchasedLimit: 10,
            purchasedLoading: false,
            renewDialogVisible: false,
            renewSite: null,
            renewServicePackageId: null,
            upgradeDialogVisible: false,
            upgradeSite: null,
            upgradeTargetVersionId: null,
            developerId: null,
            availableUpgradeVersions: [],
            upgradePrice: 0,
            payDialogVisible: false,
            panelToken: ''
        }
    },
    components: {
        respoItem,
    },
    created() {
        if (this.$route.query.tag) { this.search.tag = this.$route.query.tag; }
        if (this.$route.query.keyword) { this.search.keyword = this.$route.query.keyword; }
        if (this.$route.query.hidetags) { this.hidetags = true; }
        this.getZpk();
        this.panelToken = window.$wujie?.props?.paneltoken
    },
    watch: {
        'search.tag'(v) {
            this.$router.push({
                query: {
                    ...this.$route.query,
                    tag: v
                }
            })
        },
    },
    methods: {
        handleUpgradeVersionChange(v) {
            const targetVersion = this.availableUpgradeVersions.find(item => item.id === v)
            this.upgradePrice = targetVersion?.price || 0
        },
        handleRenew(row) {
            this.renewSite = row
            this.renewServicePackageId = row.service_packages?.[0]?.id || null
            this.renewDialogVisible = true
        },
        handleUpgrade(row) {
            this.upgradeSite = row
            if (row.product_type === 2) {
                this.submitUpgrade()
                return
            }
            const version = row.formula_version.split('.')[0] || 0
            this.availableUpgradeVersions = row.version_prices?.filter(item => item.version > version).sort((a, b) => a.version - b.version) || []
            this.upgradeTargetVersionId = this.availableUpgradeVersions[0].id
            this.handleUpgradeVersionChange(this.upgradeTargetVersionId)
            this.upgradeDialogVisible = true
        },
        handleInstall(order_sn, identifie) {
            let path = this.webUrl + '/respo/info/' + identifie + (order_sn ? '?order_sn=' + order_sn : '');
            window.open(`/app/store-install?path=${encodeURIComponent(path)}&thirdpartyCDToken=${window.$wujie?.props?.paneltoken}`)
        },
        goInstall(identifie) {

            let path = this.webUrl + '/respo/info/' + identifie
            window.open(`/app/store-install?path=${encodeURIComponent(path)}&thirdpartyCDToken=${window.$wujie?.props?.paneltoken}`)
        },
        submitRenew() {
            if (!this.renewServicePackageId) {
                ElMessage.warning('请选择服务周期')
                return
            }
            this.renewDialogVisible = false
            this.getTicket({
                identifie: this.renewSite.formula_identifie,
                order_sn: this.renewSite.order_sn,
                service_package_id: this.renewServicePackageId
            })
        },
        getTicket({ identifie, order_sn, service_package_id, version_upgrade_id }) {
            myAxios.post(`${this.webUrl}/respo/order/pay`, {
                identifie,
                order_sn,
                service_package_id,
                version_upgrade_id
            }).then(res => {
                const data = res.data.data || {}
                this.ticket = data.ticket || ''
                this.checkOrderStatus(data.order_sn)
                this.payDialogVisible = true
            })

        },
        checkOrderStatus(order_sn) {
            let t = setInterval(() => {
                myAxios.post(`${this.webUrl}/respo/order/info`, {
                    order_sn
                }).then(res => {
                    const data = res.data.data || {}
                    if (data.pay_status === 1) {
                        this.payDialogVisible = false
                        clearInterval(t)
                        this.getPurchasedData()
                    }
                }).catch(() => {
                    clearInterval(t)
                })
            }, 1000)
        },
        submitUpgrade() {
            if (!this.upgradeTargetVersionId) {
                ElMessage.warning('请选择升级版本')
                return
            }
            this.upgradeDialogVisible = false
            this.getTicket({
                identifie: this.upgradeSite.formula_identifie,
                order_sn: this.upgradeSite.order_sn,
                version_upgrade_id: this.upgradeTargetVersionId
            })
        },
        switchTab(tab) {
            if (this.activeTab === tab) return;
            this.activeTab = tab;
            this.search.keyword = '';
            this.page = 1;
            this.purchasedPage = 1;
            if (tab === 'purchased') {
                this.getPurchasedData();
            } else {
                this.getData();
            }
        },
        getZpk() {
            if (window.$wujie) {
                panelAxios.get('/panel-api/v1/zpk/local-url').then(res => {
                    this.webUrl = (res?.data?.isHttps ? 'https://' : 'http://') + res?.data?.host + '/zpk';
                    this.getData();
                    this.getTags();
                })
            } else {
                this.webUrl = ''
                this.getData();
                this.getTags();
            }

        },
        tagClick(item) {
            this.search.tag = item.name;
            this.toSearch();
        },
        getTags() {
            myAxios.post(`${this.webUrl}/respo/tag/list`, { limit: 999 }).then(res => {
                this.tags = res.data?.data?.list || [];
            });
        },
        getData() {
            myAxios.get(`${this.webUrl}/respo/list?status=2&status=99`, {
                params: {
                    page: this.page,
                    limit: this.limit,
                    tag: this.search.tag,
                    keyword: this.search.keyword,
                },
            }).then(res => {
                this.list = res.data?.data?.list || []
                this.total = res.data?.data?.total || 1;
            });
        },
        toSearch() {
            if (this.activeTab === 'purchased') {
                this.purchasedPage = 1;
                this.getPurchasedData();
            } else {
                this.page = 1;
                this.getData();
            }
        },
        getPurchasedData() {
            this.purchasedLoading = true;
            myAxios.post(`${this.webUrl}/respo/order/list`, {
                keyword: this.search.keyword,
                page: this.purchasedPage,
                per_page: this.purchasedLimit,
                enable_status: this.search.enable_status,
            }).then(res => {
                const list = res.data?.data?.list || []
                list.forEach(item => {
                    if (item.version_prices?.length) {
                        const version = item.formula_version.split('.')[0]
                        const maxPayVersion = item.version_prices?.sort((a, b) => b.version - a.version)?.[0]?.version || 0
                        item.canUpgrade = version < maxPayVersion
                    }
                })
                this.purchasedList = list;
                this.purchasedTotal = res.data?.data?.total || 0;
            }).finally(() => {
                this.purchasedLoading = false;
            });
        },
    },
    computed: {
        renewPrice() {
            const pkg = this.renewSite?.service_packages?.find(item => item.id === this.renewServicePackageId)
            return pkg ? pkg.price : 0
        },
    }
}
</script>

<style scoped>
.padding-20 {
    padding: 20px;
}

.labelbox {
    border: 1px solid #E7E7E7;
    border-radius: 8px;
    padding: 15px 15px 5px;
}

.labelbox .label {
    display: inline-flex;
    cursor: pointer;
    height: 30px;
    line-height: 30px;
    padding: 0 10px;
    margin: 0 16px 10px 0;
    border-radius: 2px;
    white-space: nowrap;
}

.labelbox .label:hover {
    background: #DCDCDC;
}

.labelbox .label.active {
    background: #2D62FF;
    color: #ffffff;
}

.alltag {
    display: inline-block;
    height: 30px;
    line-height: 30px;
    white-space: nowrap;
}

.tab-group {
    display: inline-flex;
    border: 3px solid #f2f3f5;
    border-radius: 4px;
    overflow: hidden;
}

.tab-item {
    display: inline-block;
    padding: 0 20px;
    height: 26px;
    line-height: 26px;
    font-size: 14px;
    cursor: pointer;
    background: #f2f3f5;
    color: #666;
    transition: all 0.2s;
}

.tab-item:last-child {
    border-left: 3px solid #f2f3f5;
}

.tab-item:hover {
    background: #fff;
}

.tab-item.active {
    color: #2D62FF;
    background: #fff;
}

.purchased-table {
    width: 100%;
    border-collapse: collapse;
}

.purchased-table td {
    padding: 12px 10px;
    border: 1px solid #f2f2f2;
    border-left: 0;
    border-right: 0;
    line-height: 1.4;
}

.purchased-table thead tr:first-child td {
    background: #f3f3f3;
    border-top: 0;
    color: #999;
}

.purchased-logo {
    width: 32px;
    height: 32px;
    margin-right: 8px;
}

.purchased-name {
    color: #2D62FF;
}
</style>
