<template>
    <div class="padding-20 bg-white">
        <div>
            <div style="margin-bottom: 20px;display: flex;justify-content: space-between;">
                <a-input v-if="activeTab === 'store'" class="dark-search" v-model="search.keyword"
                    :style="{ width: '256px' }" :placeholder="activeTab === 'store' ? '输入应用名称搜索' : '输入已购买应用名称搜索'"
                    @keyup.enter="toSearch">
                    <template #suffix>
                        <span class="store-search-action" @click="toSearch">
                            <icon-search :size="16" />
                        </span>
                    </template>
                </a-input>

                <div v-if="activeTab === 'purchased'">
                    <span style="margin-right: 10px;">应用状态</span>
                    <a-select v-model="search.enable_status" :style="{ width: '120px' }" placeholder="应用状态"
                        @change="toSearch">
                        <a-option label="全部" :value="null" />
                        <a-option label="有效" :value="1" />
                        <a-option label="失效" :value="0" />
                    </a-select>
                </div>
                <div class="tab-group" v-if="panelToken">
                    <button type="button" class="tab-item" :class="{ active: activeTab === 'store' }"
                        @click="switchTab('store')">制品商店</button>
                    <button type="button" class="tab-item" :class="{ active: activeTab === 'purchased' }"
                        @click="switchTab('purchased')">已购买应用</button>
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
            <div class="store-grid mt-24">
                <div class="store-grid-item" v-for="(item, index) in list" :key="item.identifie">
                    <respo-item :webUrl="webUrl" :data="item" @delete="list.splice(index, 1)" @refresh="getData"
                        @tagClick="tagClick" @goInstall="goInstall"></respo-item>
                </div>
            </div>
            <div v-if="list && list.length" class="df jc-c" style="padding-bottom:20px;">
                <a-pagination v-model:current="page" @change="getData" :page-size="limit" :total="total" />
            </div>
        </template>

        <template v-else>
            <a-table :loading="purchasedLoading" class="table-header table-respo-list" :data="purchasedList"
                :pagination="false" style="width: 100%" row-key="order_sn">
                <template #columns>
                    <a-table-column data-index="formula_name" title="应用名称" />

                    <a-table-column data-index="order_sn" title="订单编号" />
                    <a-table-column data-index="created_at" title="购买时间" />
                    <a-table-column data-index="service_expire_time" title="到期时间">
                        <template #cell="{ record }">
                            {{ record.service_expire_time === '0001-01-01 00:00:00' ? '-' :
                                record.service_expire_time }}
                        </template>
                    </a-table-column>
                    <a-table-column title="操作" :width="180">
                        <template #cell="{ record }">
                            <template v-if="record.product_type === 2">
                                <template v-if="record.goods_id > 0">
                                    <a-button type="text"
                                        v-if="record.formula_latest_version > record.formula_version && record.is_free_upgrade > 0"
                                        @click="handleUpgrade(record)">升级</a-button>
                                </template>
                                <a-tooltip v-if="record.used_time" content="授权已被使用" position="top">
                                    <a-button type="text" disabled>安装</a-button>
                                </a-tooltip>
                                <a-button type="text" v-else
                                    @click="handleInstall(record.order_sn, record.formula_identifie)">安装</a-button>
                            </template>
                            <template v-else>
                                <template v-if="record.goods_id > 0">
                                    <a-button type="text" v-if="record.canUpgrade"
                                        @click="handleUpgrade(record)">升级</a-button>
                                    <a-button type="text"
                                        v-if="record.service_packages && record.service_packages.length > 0"
                                        @click="handleRenew(record)">续费</a-button>
                                </template>

                                <a-tooltip v-if="record.used_time" content="授权已被使用" position="top">
                                    <a-button type="text" disabled>安装</a-button>
                                </a-tooltip>
                                <a-button type="text" v-else
                                    @click="handleInstall(record.order_sn, record.formula_identifie)">安装</a-button>
                            </template>
                        </template>
                    </a-table-column>
                </template>
            </a-table>
            <div v-if="purchasedList && purchasedList.length" class="df jc-c" style="padding-bottom:20px;">
                <a-pagination v-model:current="purchasedPage" @change="getPurchasedData"
                    :page-size="purchasedLimit" :total="purchasedTotal" />
            </div>
        </template>
        <a-modal v-model:visible="renewDialogVisible" title="续费服务" :width="520" :footer="false"
            unmount-on-close>
            <div v-if="renewSite">
                <a-form :model="renewSite" label-align="left" :label-col-props="{ span: 5, flex: '0 0 90px' }"
                    :wrapper-col-props="{ span: 19, flex: '1' }" class="store-modal-form">
                    <a-form-item label="服务周期" v-if="renewSite.service_packages && renewSite.service_packages.length">
                        <a-radio-group v-model="renewServicePackageId">
                            <template v-for="(item, key) in renewSite.service_packages" :key="key">
                                <a-radio class="store-radio-card" :value="item.id">
                                    <span class="price-tag">{{ item.month / 12 }}年</span>
                                </a-radio>
                            </template>
                        </a-radio-group>
                    </a-form-item>
                    <a-form-item label="续费价格">
                        <span style="color: #2d5fff;font-size:24px;font-weight:500;">¥{{ renewPrice }}</span>
                    </a-form-item>
                </a-form>
            </div>
            <div class="store-modal-footer">
                <a-button @click="renewDialogVisible = false">取消</a-button>
                <a-button type="primary" @click="submitRenew">确认续费</a-button>
            </div>
        </a-modal>
        <a-modal v-model:visible="payDialogVisible" title="支付" :width="800" :footer="false" unmount-on-close>
            <iframe
                :src="`https://ip.w7.cc/pay/${ticket}?header=false&footer=false&paid_callback=https%3A%2F%2Fuser.w7.cc%2Forder`"
                width="100%" height="650px" frameborder="0"></iframe>
        </a-modal>
        <a-modal v-model:visible="upgradeDialogVisible" title="升级版本" :width="520" :footer="false"
            unmount-on-close>
            <div v-if="upgradeSite">
                <a-form :model="upgradeSite" label-align="left" :label-col-props="{ span: 5, flex: '0 0 90px' }"
                    :wrapper-col-props="{ span: 19, flex: '1' }" class="store-modal-form">
                    <a-form-item label="升级版本">
                        <a-radio-group v-model="upgradeTargetVersionId" @change="handleUpgradeVersionChange">
                            <a-radio v-for="vp in availableUpgradeVersions" :key="vp.id" class="store-radio-card"
                                :value="vp.id">{{ vp.version === 9999 ? '其他版本' : 'V' + vp.version }}
                            </a-radio>
                        </a-radio-group>
                    </a-form-item>
                    <a-form-item label="升级价格">
                        <span style="color: #2d5fff;font-size:24px;font-weight:500;">¥{{ upgradePrice }}</span>
                    </a-form-item>
                </a-form>
            </div>
            <div class="store-modal-footer">
                <a-button @click="upgradeDialogVisible = false">取消</a-button>
                <a-button type="primary" @click="submitUpgrade">确认升级</a-button>
            </div>
        </a-modal>
    </div>
</template>

<script>
import myAxios from '@/utils';
import panelAxios from '@/utils/panel';
import respoItem from '@/components/respo-item.vue'
import { messageWarning } from '@/utils/ui-feedback';
import { getPanelToken } from '@/utils/panel-token';
import { IconSearch } from '@arco-design/web-vue/es/icon';

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
        IconSearch,
    },
    created() {
        this.panelToken = getPanelToken();
        if (this.panelToken && this.$route.query.tab === 'purchased') {
            this.activeTab = 'purchased';
        }
        if (this.$route.query.tag) { this.search.tag = this.$route.query.tag; }
        if (this.$route.query.keyword) { this.search.keyword = this.$route.query.keyword; }
        if (this.$route.query.hidetags) { this.hidetags = true; }
        this.getZpk();
    },
    watch: {
        '$route.query.tab'(tab) {
            const nextTab = this.panelToken && tab === 'purchased' ? 'purchased' : 'store';
            if (this.activeTab === nextTab) return;
            this.activeTab = nextTab;
            this.search.keyword = '';
            this.page = 1;
            this.purchasedPage = 1;
            this.loadActiveTabData();
        },
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
            window.open(`/app/store-install?path=${encodeURIComponent(path)}&thirdpartyCDToken=${encodeURIComponent(this.panelToken)}`)
        },
        goInstall(identifie) {

            let path = this.webUrl + '/respo/info/' + identifie
            window.open(`/app/store-install?path=${encodeURIComponent(path)}&thirdpartyCDToken=${encodeURIComponent(this.panelToken)}`)
        },
        submitRenew() {
            if (!this.renewServicePackageId) {
                messageWarning('请选择服务周期')
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
                messageWarning('请选择升级版本')
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
            this.$router.replace({
                query: {
                    ...this.$route.query,
                    tab: tab === 'purchased' ? 'purchased' : undefined,
                }
            });
            if (tab === 'purchased') {
                this.getPurchasedData();
            } else {
                this.getData();
            }
        },
        loadActiveTabData() {
            if (this.activeTab === 'purchased') {
                this.getPurchasedData();
            } else {
                this.getData();
            }
        },
        getZpk() {
            if (window.$wujie) {
                panelAxios.get('/panel-api/v1/zpk/local-url').then(res => {
                    this.webUrl = (res?.data?.isHttps ? 'https://' : 'http://') + res?.data?.host + '/zpk';
                    this.loadActiveTabData();
                    this.getTags();
                })
            } else {
                this.webUrl = ''
                this.loadActiveTabData();
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
    margin: 0;
    padding: 0 20px;
    height: 26px;
    line-height: 26px;
    font-size: 14px;
    font-family: inherit;
    cursor: pointer;
    background: #f2f3f5;
    border: 0;
    color: #666;
    appearance: none;
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

.store-search-action {
    color: #86909c;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
    transition: color .2s;
}

.store-search-action:hover {
    color: #4e5969;
}

.store-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 24px;
}

.store-grid-item {
    min-width: 0;
}

.table-header :deep(.arco-table-th) {
    background: #F3F3F3;
    color: #666666;
    font-weight: 500;
}

.store-modal-form :deep(.arco-form-item-label) {
    white-space: nowrap;
}

.store-modal-form :deep(.arco-form-item-wrapper-col) {
    min-width: 0;
}

.store-radio-card {
    margin-right: 12px;
}

.store-modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 20px;
}

@media (max-width: 960px) {
    .store-grid {
        grid-template-columns: repeat(2, minmax(0, 1fr));
    }
}

@media (max-width: 640px) {
    .store-grid {
        grid-template-columns: 1fr;
    }
}
</style>
