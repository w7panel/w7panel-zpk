<template>
    <div class="site-detail-page">
        <div class="content" :class="{ 'p-0': !isWujie }">
            <div class="top-panel">
                <div style="flex: 0 0 956px;">
                    <div style="display: flex;">
                        <div class="goods-info-box">
                            <div class="goods-base">
                                <img :src="appInfo.icon_url" class="goods-base__logo" alt="">
                                <div class="goods-base__info ">
                                    <div class="goods-base__title">
                                        <span>
                                            {{ appInfo.title }}
                                        </span>
                                    </div>
                                    <div class="goods-desc-shrink">
                                        <div style="">
                                            {{ appInfo.description }}
                                        </div>
                                    </div>
                                </div>
                            </div>
                            <div style="display: flex;align-items: start;padding-left: 16px;margin-top: 20px;">
                                <span class="detail-text-2">关联标签</span>
                                <div class="goods-base__label">
                                    <a v-for="label in appInfo.tags" class="label-item" :key="label.id"
                                        :href="label.url">
                                        <i class="wi wi-tag"></i>{{ label.name }}
                                    </a>
                                </div>
                            </div>
                            <div
                                style="border-radius: 8px;overflow: visible;background-color: #cfe2fb91;margin-top: 16px">
                                <div style="display: flex;overflow: hidden;padding: 16px 0 16px 16px;">
                                    <div style="flex: 1">
                                        <div style="display: flex;align-items: center;height: 32px">
                                            <span class="detail-text-2" style="margin-right: 32px;">当前价格</span>
                                            <span
                                                style="color: #2d5fff;font-size:24px;font-weight:500;vertical-align: middle"
                                                v-text="currentPrice === 0 ? '免费' : '¥' + currentPrice"></span>

                                        </div>
                                        <div style="display: flex;align-items: center;height: 32px;margin-top: 4px;"
                                            v-if="appInfo.service_packages && appInfo.service_packages.length">
                                            <span class="detail-text-2" style="margin-right: 32px;">续费价格
                                            </span>
                                            <template v-for="(item, key) in appInfo.service_packages" :key="key">
                                                <span class="detail-text-1" style="margin-right: 16px">{{
                                                    item.price
                                                }}元/{{ item.month / 12 }}年</span>
                                            </template>
                                        </div>
                                    </div>
                                    <div class="goods-comment-score">
                                        <div class="detail-text-1"
                                            v-text="appInfo.install_total > 9 ? (appInfo.install_total + '人在使用') : '<10人在使用'">
                                        </div>
                                        <div style="margin-top: 4px">
                                            <el-icon :size="14" color="#FF7D00" v-for="i in 5" :key="i">
                                                <StarFilled />
                                            </el-icon>
                                        </div>
                                    </div>
                                </div>
                            </div>
                            <el-form style="margin-top: 20px" class="goods-product" label-width="90px"
                                label-position="left">
                                <el-form-item style="margin-bottom: 16px"
                                    class="color-info el-form-item--support is-label" label="交付方式">
                                    <div class=" detail-text-1" style="line-height: 26px">
                                        微擎面板
                                    </div>
                                </el-form-item>
                                <el-form-item v-if="appInfo.service_packages && appInfo.service_packages.length"
                                    class="color-info">
                                    <template #label>
                                        <div style="display: flex;align-items: center;">
                                            服务周期<el-popover placement="top" trigger="hover">
                                                <div>服务周期内，应用可免费更新。</div>
                                                <template #reference>
                                                    <i class="wi wi-explain color-danger"></i>
                                                </template>
                                            </el-popover>
                                        </div>
                                    </template>
                                    <div>
                                        <el-radio-group v-model="activeServicePackageId">
                                            <template v-for="(item, key) in appInfo.service_packages" :key="key">
                                                <el-radio class="el-radio--w7" :label="item.id" border>
                                                    <span class="price-tag">￥{{ item.price }}/{{ item.month / 12
                                                    }}年</span><span v-if="item.is_gift">（赠送）</span></el-radio>
                                            </template>
                                        </el-radio-group>
                                    </div>
                                </el-form-item>
                                <el-form-item label="授权信息" class="color-info" v-if="uid > 0 && sites.length > 0">
                                    <div class="site-select-box" @click.stop="siteDialogVisible = true">
                                        <div class="text-over"
                                            style="position: relative;user-select: none;line-height: 22px;border: 1px solid #DCDCDC;box-sizing: border-box;padding: 5px 30px 5px 8px;border-radius: 3px;">
                                            <span
                                                style="margin-right: 10px;background: #E8F1FF;font-size: 12px;line-height: 20px;width: 40px;color: #2d5fff;display: inline-block;border-radius: 3px;text-align: center">
                                                {{ getSiteStatusText(activeSite.service_expire_time) }}
                                            </span>
                                            <span style="font-size: 14px;color: rgba(0, 0, 0, 0.9);">{{
                                                activeSite.order_sn }}</span>
                                            <i class="el-icon-arrow-down"
                                                style="font-weight: bold;position: absolute;right: 8px;top: 8px;color: rgba(0, 0, 0, 0.6);" />
                                        </div>
                                    </div>
                                    <el-dialog title="授权信息" custom-class="site-dialog" append-to-body
                                        v-model="siteDialogVisible" width="840px">
                                        <el-table :data="sites" style="width: 100%">
                                            <el-table-column prop="order_sn" label="订单编号" width="260">
                                            </el-table-column>
                                            <el-table-column prop="created_at" label="购买时间" width="180">
                                            </el-table-column>
                                            <el-table-column prop="service_expire_time" label="到期时间" width="180">
                                                <template #default="scope">
                                                    {{ scope.row.service_expire_time === '0001-01-01 00:00:00' ? '-' :
                                                        scope.row.service_expire_time }}
                                                </template>
                                            </el-table-column>
                                            <el-table-column prop="" label="操作" width="180">
                                                <template #default="scope">
                                                    <template v-if="appInfo.product_type === 2">
                                                        <template v-if="appInfo.goods_id > 0">
                                                            <el-button type="text"
                                                                v-if="scope.row.canUpgrade && appInfo.is_free_upgrade > 0"
                                                                @click="handleUpgrade(scope.row)">升级</el-button>
                                                        </template>
                                                        <el-tooltip v-if="scope.row.used_time" content="授权已被使用"
                                                            placement="top">
                                                            <el-button type="text" disabled>安装</el-button>
                                                        </el-tooltip>
                                                        <el-button type="text" v-else
                                                            @click="order_sn = scope.row.order_sn; confirmInstall()">安装</el-button>
                                                    </template>
                                                    <template v-else>
                                                        <template v-if="appInfo.goods_id > 0">
                                                            <el-button type="text" v-if="scope.row.canUpgrade"
                                                                @click="handleUpgrade(scope.row)">升级</el-button>
                                                            <el-button type="text"
                                                                v-if="appInfo.service_packages && appInfo.service_packages.length > 0"
                                                                @click="handleRenew(scope.row)">续费</el-button>
                                                        </template>

                                                        <el-tooltip v-if="scope.row.used_time" content="授权已被使用"
                                                            placement="top">
                                                            <el-button type="text" disabled>安装</el-button>
                                                        </el-tooltip>
                                                        <el-button type="text" v-else
                                                            @click="order_sn = scope.row.order_sn; confirmInstall()">安装</el-button>
                                                    </template>
                                                </template>
                                            </el-table-column>
                                        </el-table>
                                    </el-dialog>
                                </el-form-item>
                                <el-form-item class="is-label el-form-item--action">
                                    <div class="buy-button-group" style="position: relative;">
                                        <el-button
                                            v-if="appInfo.goods_id === 0 || (appInfo.goods_audit_status === 4 && appInfo.goods_onshef === 2)"
                                            type="primary" class="buy-button-item" @click="checkOrder">{{
                                                currentPrice === 0 ? '获取安装' :
                                                    '购买安装' }}</el-button>
                                        <el-button v-else disabled type="primary">商品已下架</el-button>
                                    </div>
                                </el-form-item>
                                <el-form-item class="color-info is-label">
                                    <el-checkbox class="color-info soe-checkbox" v-model="checkBuy">
                                        <div style="font-size: 12px">
                                            购买即同意<a href="https://wiki.w7.cc/chapter/887?id=3909"
                                                target="_blank">《微擎平台使用协议》</a>
                                        </div>
                                    </el-checkbox>
                                </el-form-item>
                            </el-form>
                        </div>
                    </div>
                </div>

                <div class="user-div" style="background-color: #fff">
                    <div class="top-info">

                        <div class="div-company-name" style="margin-bottom: 20px;align-items: center;">
                            <a :href="'https://ip.w7.cc/dev/' + developerId" target="_blank" v-if="developerId">
                                <div>
                                    {{ developerInfo.username || appInfo.founder_username }}
                                </div>
                            </a>
                            <div v-else>
                                {{ developerInfo.username || appInfo.founder_username }}
                            </div>
                        </div>
                        <template v-if="appInfo.founder_console_uid">
                            <div class="div-flex" style="height: 22px;margin-bottom: 16px;align-items: center;">
                                <p>资质荣誉</p>
                                <div>
                                    <a target="_blank" title="开发者实名认证" class="wi"
                                        href="https://s.w7.cc/store-static-promise.html"><img
                                            src="https://cdn.w7.cc/ued/shop/img/detail/person.png" /></a>
                                    <a target="_blank" v-if="developerInfo.is_company" title="开发者企业认证" class="wi"
                                        href="https://s.w7.cc/store-static-promise.html"><img
                                            src="https://cdn.w7.cc/ued/shop/img/detail/company.png" /></a>
                                    <template v-if="developerInfo.icons">
                                        <a target="_blank" v-for="(item, index) in developerInfo.icons" :key="index"
                                            :title="item.title" class="wi" href="javascript:void(0)"><img
                                                :src="item.logo" /></a>
                                    </template>
                                </div>
                            </div>
                            <div class="div-flex" style="height: 22px;margin-bottom: 16px;align-items: center;">
                                <p>
                                    入驻年限
                                </p>
                                <template v-if="developerInfo.join_year > 0">
                                    <div class="year">
                                        <div class="year-box" :data-content="developerInfo.join_year"></div>
                                    </div>
                                </template>
                                <template v-else>
                                    不足一年
                                </template>
                            </div>
                            <div class="div-flex" style="height: 22px;margin-bottom: 16px;align-items: center;">
                                <p>
                                    卖家等级
                                </p>
                                <div style="line-height: 24px">
                                    {{ developerInfo.rolename }}
                                </div>
                            </div>
                            <div class="div-flex" v-if="developerInfo.has_business_licence" style="height: 22px">
                                <p style="margin: 0;">经营凭证</p>
                                <div>
                                    <a target="_blank" title="店铺营业执照" class="wi"
                                        :href="'https://s.w7.cc/verify/company/' + developerInfo.dev_code"><img
                                            src="https://cdn.w7.cc/ued/shop/img/detail/license.png" /></a>
                                </div>
                            </div>
                            <div class="user-rate" v-if="developerInfo.reputation && developerInfo.reputation.stat">
                                <template v-for="(item, key) in developerInfo.reputation.stat" :key="key">
                                    <el-tooltip placement="left" effect="light" popper-class="reputation">
                                        <template #content>
                                            <div :class="['rate-we7', item.class + '-theme']">
                                                微擎开发者平均值：{{ item.all }}<span>{{ key !== 'active' ? item.differ : ''
                                                    }}</span>
                                            </div>
                                        </template>
                                        <a :href="'/developer/reputation/' + developerId" target="_blank" class="item">
                                            {{ item.title }}
                                            <span :class="item.class">
                                                {{ item.developer }}
                                            </span>
                                        </a>
                                    </el-tooltip>
                                </template>
                            </div>
                            <div class="user-im" v-if="im && im.url">
                                <div
                                    style="display: flex;justify-content: space-between;margin-bottom: 8px;flex-wrap: wrap">
                                    <a :href="im.url" target="_blank" class="im-button">
                                        <span>
                                            立即咨询
                                        </span>
                                    </a>
                                </div>
                                <div style="text-align: center;font-size: 14px;line-height: 22px">
                                    <div style="color: rgba(0, 0, 0, 0.9);">
                                        {{
                                            `卖家服务时间：${imService.weekStr}`
                                        }}
                                    </div>
                                    <div style="color: #FF7D00;font-weight: bolder;">
                                        {{ imService.workTimeStr }}
                                    </div>
                                </div>
                            </div>
                        </template>
                    </div>
                </div>
            </div>
            <div class="bottom-panel">
                <div class="description-box">
                    <SiteDescription :files="description" />
                </div>
            </div>
        </div>
        <el-dialog top="5vh" v-model="payDialogVisible" title="支付" width="800px" destroy-on-close>
            <iframe
                :src="`https://ip.w7.cc/pay/${ticket}?header=false&footer=false&paid_callback=https%3A%2F%2Fuser.w7.cc%2Forder`"
                width="100%" height="650px" frameborder="0"></iframe>
        </el-dialog>


        <el-dialog v-model="renewDialogVisible" title="续费服务" width="520px" destroy-on-close>
            <div v-if="renewSite">
                <el-form label-width="90px" label-position="left">
                    <el-form-item label="服务周期" v-if="appInfo.service_packages && appInfo.service_packages.length">
                        <el-radio-group v-model="renewServicePackageId">
                            <template v-for="(item, key) in appInfo.service_packages" :key="key">
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


        <el-dialog v-model="upgradeDialogVisible" title="升级版本" width="520px" destroy-on-close>
            <div v-if="upgradeSite">
                <el-form label-width="90px" label-position="left">
                    <el-form-item label="升级版本">
                        <el-radio-group v-model="upgradeTargetVersionId">
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

        <el-dialog v-model="installDialogVisible" title="支付成功" width="520px" destroy-on-close>
            <el-result icon="success" title="支付成功" subTitle="请点击下方按钮去安装">
                <template #extra>
                    <el-button type="primary" @click="confirmInstall">去安装</el-button>
                </template>
            </el-result>
        </el-dialog>
    </div>
</template>

<script>
import siteAxios from '@/utils/site'
import panelAxios from '@/utils/panel'
import yaml from 'js-yaml'
import jsonp from 'jsonp'
import SiteDescription from '@/components/description.vue'

export default {
    name: 'SiteDetail',
    data() {
        return {
            installDialogVisible: false,
            order_sn: '',
            loading: false,
            errorMsg: '',
            info: {},
            manifestData: null,
            isLogin: false,
            activeServicePackageId: null,
            appInfo: {},
            sites: [],
            checkBuy: true,
            developerInfo: {},
            im: {
                url: ''
            },
            imService: {
                weekStr: '周一至周日',
                workTimeStr: '09:00 - 22:00',
            },
            description: {},
            siteDialogVisible: false,
            activeSite: {},
            payDialogVisible: false,
            ticket: '123123',
            renewDialogVisible: false,
            renewSite: null,
            renewServicePackageId: null,
            upgradeDialogVisible: false,
            upgradeSite: null,
            upgradeTargetVersionId: null,
            developerId: null,
            uid: null,
            fromHost: '',
            path: '',
            token: '',
            isWujie: false,
            webUrl: ''
        }
    },
    created() {
        this.init().then(() => {
            this.fetchDetail()
            this.getDescription()
        })
    },
    components: {
        SiteDescription,
    },
    computed: {
        name() {
            return this.$route.params.name || ''
        },
        currentPrice() {
            const appPrice = this.appInfo.goods_id > 0 ? (this.appInfo.install_service_fee > 0 ? this.appInfo.install_service_fee : 0) : 0
            const activeService = this.appInfo.service_packages?.find(item => item.id === this.activeServicePackageId)
            const servicePrice = activeService ? activeService.is_gift ? 0 : activeService.price : 0
            return appPrice + servicePrice
        },
        availableUpgradeVersions() {
            if (!this.upgradeSite || !this.appInfo.version_prices) return []
            const currentVersion = this.upgradeSite.formula_version?.split('.')?.[0]
            return this.appInfo.version_prices
                .filter(vp => vp.version > currentVersion)
                .sort((a, b) => a.version - b.version)
        },
        renewPrice() {
            const pkg = this.appInfo.service_packages?.find(item => item.id === this.renewServicePackageId)
            return pkg ? pkg.price : 0
        },
        upgradePrice() {
            const vp = this.availableUpgradeVersions.find(v => v.id === this.upgradeTargetVersionId)
            return vp ? vp.price : 0
        }
    },
    methods: {
        init() {
            return new Promise(async (res) => {
                this.isWujie = !!window.$wujie
                this.fromHost = this.$route.query.fromHost || ''
                this.path = this.$route.query.path || ''
                const token = this.$route.query.token || ''
                if (token !== 'undefined') {
                    this.token = token
                    window.localStorage.setItem('X-W7Panel-Token', token || '')
                }
                if (this.isWujie) {
                    await this.getWebUrl()
                }
                res()
            })
        },
        getWebUrl() {
            return panelAxios.get('/panel-api/v1/zpk/local-url').then(res => {
                this.webUrl = (res?.data?.isHttps ? 'https://' : 'http://') + res?.data?.host + '/zpk';
            })
        },
        requestPath(path) {
            return this.isWujie ? `${this.webUrl}${path}` : path
        },
        handleRenew(row) {
            this.renewSite = row
            this.renewServicePackageId = this.appInfo.service_packages?.[0]?.id || null
            this.siteDialogVisible = false
            this.renewDialogVisible = true
        },
        handleUpgrade(row) {
            this.upgradeSite = row
            if (this.appInfo.product_type === 2) {
                this.submitUpgrade()
                return
            }
            this.upgradeTargetVersionId = this.availableUpgradeVersions[0].id
            this.siteDialogVisible = false
            this.upgradeDialogVisible = true
        },
        handleInstall(order_sn) {
            this.order_sn = order_sn || ''
            this.installDialogVisible = true
        },
        confirmInstall() {
            this.installDialogVisible = false
            window.open(`${this.fromHost}/app/store-install?path=${encodeURIComponent(this.path + (this.order_sn ? '?order_sn=' + this.order_sn : ''))}&thirdpartyCDToken=${this.token}`)
        },
        submitRenew() {
            if (!this.renewServicePackageId) {
                ElMessage.warning('请选择服务周期')
                return
            }
            this.renewDialogVisible = false
            this.getTicket({
                order_sn: this.renewSite.order_sn,
                service_package_id: this.renewServicePackageId
            })
        },
        submitUpgrade() {
            if (!this.upgradeTargetVersionId) {
                ElMessage.warning('请选择升级版本')
                return
            }
            this.upgradeDialogVisible = false
            this.getTicket({
                order_sn: this.upgradeSite.order_sn,
                version_upgrade_id: this.upgradeTargetVersionId,
                needInstall: true
            })
        },
        getTicket({ order_sn, service_package_id, version_upgrade_id, needInstall }) {
            siteAxios.post(this.requestPath('/respo/order/pay'), {
                identifie: this.name,
                order_sn,
                service_package_id,
                version_upgrade_id
            }).then(res => {
                const data = res.data.data || {}
                this.ticket = data.ticket || ''
                this.checkOrderStatus(data.order_sn, needInstall)
                this.payDialogVisible = true
            })

        },
        checkOrderStatus(order_sn, needInstall) {
            let t = setInterval(() => {
                siteAxios.post(this.requestPath('/respo/order/info'), {
                    order_sn
                }).then(res => {
                    const data = res.data.data || {}
                    if (data.pay_status === 1) {
                        this.payDialogVisible = false
                        clearInterval(t)
                        this.getSites()
                        if (needInstall) {
                            this.handleInstall(order_sn)
                        }
                    }
                }).catch(() => {
                    clearInterval(t)
                })
            }, 1000)
        },
        getSiteStatusText(dateText) {
            if (dateText === '0001-01-01 00:00:00') {
                return '生效中'
            }
            const date = new Date(dateText)
            return date.getTime() > Date.now() ? '生效中' : '已过期'
        },
        getSites() {
            siteAxios.post(this.requestPath('/respo/order/list'), {
                formula_identify: this.name
            }).then(res => {
                const sites = res.data.data.list || []
                const maxPayVersion = this.appInfo.version_prices?.sort((a, b) => b.version - a.version)?.[0]?.version || 0
                this.sites = sites.map(item => {
                    item.statusText = this.getSiteStatusText(item.service_expire_time)
                    const version = item.formula_version.split('.')[0]
                    item.canUpgrade = version < maxPayVersion
                    return item
                })
                if (sites.length) {
                    this.activeSite = sites[0]
                }
            })
        },
        checkOrder() {
            if (!this.checkBuy) {
                ElMessage.warning('请先同意《微擎平台使用协议》')
                return
            }
            if (this.currentPrice === 0) {
                this.confirmInstall()
                return
            }
            this.getTicket({
                service_package_id: this.activeServicePackageId,
                needInstall: true
            })
        },
        async getAccountInfo() {
            const res = await siteAxios.get(this.requestPath('/respo/user/info'))
            this.uid = res.data.data.console_uid
        },
        async fetchDetail() {
            if (!this.name) {
                this.info = {}
                this.manifestData = null
                return
            }
            this.loading = true
            this.errorMsg = ''
            try {
                const res = await siteAxios.get(this.requestPath(`/respo/goods/info/${this.name}`))
                this.appInfo = res?.data?.data || {}
                if (this.appInfo.service_packages && this.appInfo.service_packages.length) {
                    this.activeServicePackageId = this.appInfo.service_packages[0].id
                }
                if (this.appInfo.founder_console_uid > 0) {
                    this.getDeveloperInfo(this.appInfo.founder_console_uid)
                }
                await this.getAccountInfo()
                if (this.uid > 0) {
                    this.getSites()
                }
            } finally {
                this.loading = false
            }
        },
        getDescription() {
            siteAxios.get(this.requestPath(`/respo/detail/${this.name}`)).then(res => {
                this.description = res.data.data.mds || {}
            })
        },
        getDeveloperInfo(uid) {
            this.developerId = uid
            jsonp(`https://s.w7.cc/developer/info/${uid}`, (err, data) => {
                if (err) {
                    return
                }
                if (data.reputation && data.reputation.stat) {
                    var Reputation = data.reputation.stat;
                    for (let i in Reputation) {
                        if (Reputation[i]['differ'] > 0) {
                            Reputation[i]['class'] = 'up'
                        } else if (Reputation[i]['differ'] < 0) {
                            Reputation[i]['class'] = 'down'
                        } else {
                            Reputation[i].class = "equal"
                        }
                        Reputation[i].differ = Math.abs(Reputation[i]['differ']).toFixed(2) + '分'
                        if (Reputation[i]['children']) {
                            var children = Reputation[i]['children']
                            for (let j in children) {
                                if (children[j]['differ'] > 0) {
                                    children[j]['class'] = 'up'
                                } else if (children[j]['differ'] < 0) {
                                    children[j]['class'] = 'down'
                                } else {
                                    children[j].class = "equal"
                                }
                                children[j].differ = Math.abs(children[j]['differ']).toFixed(2) + '分'
                            }
                        }
                    }
                }
                this.developerInfo = data
                this.getIm()
                this.getImServiceConfig()
            })
        },
        getIm() {
            this.im.url = `https://im.w7.cc/customer?remote_user_id=${this.developerInfo}&good_id=${this.appInfo.goods_id}&source=https%3A%2F%2Fip.w7.cc&source_type=shop_ip`
        },
        getImServiceConfig() {
            jsonp('https://im.w7.cc/apiOpen/im/getImOnline?remote_uid=' + this.developerId, {
                param: 'jsoncallback'
            }, (err, res) => {
                if (!err) {
                    if (res.im_config.config) {
                        this.imService = {
                            weekStr: `${res.im_config.config.start_week_str}至${res.im_config.config.end_week_str}`,
                            workTimeStr: `${res.im_config.config.start_time} - ${res.im_config.config.end_time}`,
                        }
                    } else {
                        this.imService = {
                            weekStr: '周一至周日',
                            workTimeStr: '09:00 - 22:00',
                        }
                    }
                } else {
                    this.imService = {
                        weekStr: '周一至周日',
                        workTimeStr: '09:00 - 22:00',
                    }
                }
            })
        },
        getCopyText() {
            return this.appInfo.title + '\n' + (this.$refs.shareButton && this.$refs.shareButton.shareUrl ? this.$refs.shareButton.shareUrl : window.location.href) + '\n(来自：微擎应用商城)'
        },
        parseManifest(text) {
            if (!text) {
                this.manifestData = null
                return
            }
            try {
                this.manifestData = yaml.load(text)
            } catch (error) {
                this.manifestData = null
            }
        },
        onLogoError(event) {
            const target = event && event.target
            if (target) {
                target.src = '/api/core/logo'
            }
        },
        formatDate(value) {
            if (!value) {
                return '-'
            }
            const time = new Date(value)
            if (Number.isNaN(time.getTime())) {
                return String(value)
            }
            const Y = time.getFullYear()
            const M = String(time.getMonth() + 1).padStart(2, '0')
            const D = String(time.getDate()).padStart(2, '0')
            const h = String(time.getHours()).padStart(2, '0')
            const m = String(time.getMinutes()).padStart(2, '0')
            const s = String(time.getSeconds()).padStart(2, '0')
            return `${Y}-${M}-${D} ${h}:${m}:${s}`
        },
        formatAmount(value) {
            if (value === null || value === undefined || value === '') {
                return '-'
            }
            const num = Number(value)
            if (!Number.isFinite(num)) {
                return String(value)
            }
            return `￥${num}`
        },
        boolText(value) {
            if (value === true || value === 1 || value === '1') {
                return '是'
            }
            if (value === false || value === 0 || value === '0') {
                return '否'
            }
            return this.displayValue(value)
        },
        displayValue(value) {
            if (value === null || value === undefined || value === '') {
                return '-'
            }
            return String(value)
        },
        getArrayLength(arr) {
            return Array.isArray(arr) ? arr.length : 0
        },
        getNested(source, path) {
            if (!source || !Array.isArray(path)) {
                return ''
            }
            let value = source
            for (let i = 0; i < path.length; i += 1) {
                if (!value || typeof value !== 'object') {
                    return ''
                }
                value = value[path[i]]
            }
            return value || ''
        }
    }
}
</script>

<style scoped lang="scss">
$--color-primary: #2d5fff;
$--color-info: #606266;
$--border-color-base: #e8e9eb;

.goods-info-box {
    flex: 1;
    width: 0;
    padding: 20px 15px;
    background-color: #ffffff;
    box-sizing: border-box;
    z-index: 10;
    font-size: 0;
}

.goods-base {
    display: flex;
    font-size: 14px;
    position: relative;

    &__logo {
        width: 80px;
        height: 80px;
        border-radius: 8px;
        margin-right: 24px;
    }

    &__info {
        flex: 1;
        width: 0;
    }

    &__collect {
        cursor: pointer;
        float: right;
        font-size: 14px;
        color: #666666;
        line-height: 22px;

        * {
            vertical-align: middle;
        }
    }

    &__title {
        font-size: 20px;
        margin-bottom: 10px;
        vertical-align: top;
        line-height: 28px;

        &-own {
            margin-top: 6px;
            line-height: 20px;
            display: inline-block;
            vertical-align: top;
            width: 40px;
            text-align: center;
            border-radius: 3px;
            font-size: 12px;
            color: rgba(255, 255, 255, 0.9);
            background-color: #2d5fff;
        }
    }

    &__version {
        margin-bottom: 10px;
        line-height: 1;
    }

    &__label {
        flex: 1;
        margin-left: 22px;
        font-size: 12px;
        color: rgba(0, 0, 0, 0.6);
        display: flex;
        flex-wrap: wrap;

        .label-item {
            background: rgba(126, 134, 142, 0.08);
            color: rgba(0, 0, 0, 0.6);
            padding: 0px 6px;
            height: 22px;
            display: flex;
            align-items: center;
            border-radius: 4px;
            margin-left: 10px;
            margin-bottom: 10px;

            i {
                font-size: 14px;
                vertical-align: middle;
                margin-right: 5px;
            }

            &:hover {
                color: $--color-primary;
            }
        }
    }

    &__credit {
        position: relative;
        border: 1px solid #85b5fe;
        box-sizing: border-box;
        background: #85b5fe;
        line-height: 40px;
        height: 40px;
        display: flex;
        align-items: center;
        border-bottom: 1px solid #85b5fe;
    }
}

.goods-other {
    display: flex;
    font-size: 14px;
    margin-top: 30px;
    margin-bottom: 30px;
    line-height: 1;

    .item {
        margin-right: 10px;
        padding-left: 10px;

        .color-base {
            font-weight: 600;
        }

        &:first-child {
            text-align: center;
            cursor: pointer;
            width: 76px;
            padding-left: 0;
            margin-right: 0px;
        }
    }
}

.goods-product,
.goods-prices {
    padding: 0 10px 0px 15px;

    .el-form-item {
        margin-bottom: 20px;

        &:last-child {
            margin-bottom: 0;
        }

        &.is-label {
            .el-form-item__label {
                line-height: 1;
            }

            .el-form-item__content {
                line-height: 1;
            }
        }

        &__label {
            color: rgba(0, 0, 0, 0.6);
            ;
            line-height: 38px;
        }

        &__content {
            line-height: 38px;
        }
    }

    .el-checkbox {
        &__label {
            color: $--color-info;
        }
    }

    .el-radio-group,
    .el-checkbox-group {
        display: flex;
        flex-wrap: wrap;
        margin-right: -12px;
        margin-bottom: -12px;

        .el-radio,
        .el-checkbox {
            margin-left: 0;
            margin-right: 12px;
            margin-top: 3px;
            margin-bottom: 9px;
            height: 32px;
            box-sizing: border-box;
            border-radius: 2px !important;
            padding-left: 16px;
            padding-right: 16px;
            display: flex;
            align-items: center;

            &.is-bought {
                padding-right: 40px;

                .given-tag {
                    font-weight: bold;
                    transform: scale(0.8);
                    transform-origin: right top;
                }
            }

            .price-tag {
                color: #000000aa;
            }

            .given-tag {
                position: absolute;
                right: -1px;
                top: -1px;
                font-size: 12px;
                height: 20px;
                line-height: 20px;
                vertical-align: top;
                color: #fff;
                padding: 0 4px;
                border-top-right-radius: 3px;
                border-bottom-left-radius: 6px;
            }

            &__label {
                line-height: 32px;
                padding-left: 1px;
                padding-right: 1px;
            }

            &.is-checked {
                position: relative;
                border-color: #2d5fff;
                border-width: 1px;

                .given-tag {
                    background-color: #2d5fff;
                    top: -2px;
                    right: -2px;
                }

                .price-tag {
                    color: #2d5fff;
                }

                &::before {
                    content: '';
                    position: absolute;
                    bottom: 0;
                    right: 0;
                    width: 22px;
                    height: 16px;
                    background-image: url(data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABYAAAAQCAMAAAAlM38UAAAAAXNSR0IB2cksfwAAAAlwSFlzAAALEwAACxMBAJqcGAAAAFFQTFRFAAAAQ3PtQXPyQXX3QXPyQnTzQnTyQXPyQHPyQnTzQXPyQnTzL2n8QnTyP3P1QnTzQXPyQnTzY4z1zNn7TXzz1+H8bJP11uH8V4P0l7L4Q3XzsDHLHAAAABt0Uk5TABCdMd3/dvkZuUbmBokizWLv//////////9j1thQKAAAAGFJREFUeJxtzkkOgCAQBVFAv/MEDqD3P6gChnTSXcu3KqWEtJG0qiFo04Jz1wOchxGcpxngvKwQeLOJ3E5ZGxxJT0f4m718+LVwnMXtQ1bQ2ehZQWeju3JSZkl0lvJjBcUL3NIFzmiP9QMAAAAASUVORK5CYII=)
                }

                &::after {
                    display: none;
                }

                .el-radio__label,
                .el-checkbox__label {
                    color: #F53F3F;
                    padding-left: 0;
                    padding-right: 0;
                }
            }
        }
    }

    .margin-right-sm {
        margin-right: 10px;
    }

    a {
        &:hover {
            color: $--color-primary;
        }
    }
}

.goods-prices {
    padding: 20px 15px;

    .el-form-item--price {
        .el-form-item__label {
            line-height: 24px !important;
        }
    }

}

.goods-desc {
    &-shrink {
        overflow: hidden;
        font-size: 12px;
        text-overflow: ellipsis;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        text-align: left;
        word-break: break-all;
        line-height: 22px;
        letter-spacing: 1px;
        color: rgba(0, 0, 0, 0.6);
    }
}

.detail-text-1 {
    font-size: 14px;
    line-height: 22px;
    color: rgba(0, 0, 0, 0.9);
}

.detail-text-2 {
    font-size: 14px;
    line-height: 22px;
    color: rgba(0, 0, 0, 0.6);
}

.goods-comment-score {
    border-left: 1px solid #FDCDC5;
    flex: 0 0 154px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-direction: column;
}

.top-panel {
    display: flex;
    gap: 20px;
}

.site-detail-page {
    min-height: 100vh;
    background-color: #f2f3f5;

    .content {
        width: 1200px;
        margin: 0 auto;

        &.p-0 {
            padding: 0;
        }
    }
}

.user-div {
    width: 224px;
    flex: 0 0 224px;
    position: relative;

    .top-info {
        padding: 20px 25px;
        font-size: 14px;

        .title {
            font-size: 16px;
            margin-bottom: 15px;
        }

        .year {
            text-align: center;
            display: inline-block;
        }

        .year-box {
            display: inline-block;
            border: 1px solid #C3D9FF;
            font-size: 12px;
            border-radius: 2px;
            line-height: 20px;

            &::before {
                content: attr(data-content);
                background-color: $--color-primary;
                color: #fff;
                width: 20px;
                border-radius: 2px 0px 0px 2px;
                display: inline-block;
            }

            &:after {
                content: '微擎' attr(data-content)'年店';
                color: $--color-primary;
                padding: 0 5px;
                display: inline-block;
            }
        }

        .div-company-avatar {
            font-size: 0;
            text-align: center;
            margin-bottom: 12px;

            img {
                height: 40px;
                width: 40px;
                border-radius: 20px;
            }
        }

        .div-developer-name {
            line-height: 1;
            margin-bottom: 15px;
            text-align: center;
        }

        .div-company-name {
            overflow: hidden;
            text-overflow: ellipsis;
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
            text-align: left;
            word-break: break-all;
            line-height: 20px;
            max-height: 40px;
            margin-bottom: 12px;
            font-weight: bold;
            cursor: pointer;

            &:hover {
                a {
                    display: block;
                    ;
                    color: #3296fa;
                }
            }
        }

        .div-flex {
            margin-bottom: 12px;
            margin-right: -25px;
            display: flex;
            word-break: break-all;
            width: 100%;
            align-items: flex-start;

            p {
                flex: 0 0 70px;
                line-height: 24px;
            }

            .develop-name {
                align-self: center;
                width: 123px;
                overflow: hidden;
                white-space: nowrap;
                text-overflow: ellipsis;
            }

            div {
                display: flex;
                flex-wrap: wrap;
                align-items: center;
            }

            img {
                width: 62px;
                flex-shrink: 0;
                height: 62px;
                border-radius: 100%;
                margin-right: 15px;
            }

            span {
                p {
                    margin: 7.5px 0px;
                }
            }

            a {
                margin-right: 5px;
                font-size: 0;

                img {
                    height: auto;
                    width: auto;
                    max-height: 20px;
                    border-radius: 0;
                    max-width: 20px;
                    margin-right: 0;
                    vertical-align: middle;
                }
            }
        }

        .btn {
            padding: 10px 35px;
        }
    }

    .user315 {
        border-top: 1px dashed $--border-color-base;
        position: absolute;
        text-align: center;
        bottom: 0;
        left: 10px;
        right: 10px;
        height: 46px;
        line-height: 42px;
        font-size: 12px;

        img {
            vertical-align: middle;
        }

        .wi {
            color: #d10f0f;
            font-size: 20px;
            margin-right: 10px;
        }

        a {
            &:hover {
                color: $--color-primary;
            }
        }
    }

    .user-rate {
        display: block !important;
        border-top: 1px dashed $--border-color-base;
        margin: 24px -10px 0;
        padding: 24px 10px 0;

        .item {
            font-size: 14px;
            color: #4d4d4d;
            margin-bottom: 17px;
            display: block;

            &>span {
                color: $--color-primary;
                float: right;
                margin-right: 2px;
                padding-right: 5px;
                position: relative;

                &.down {
                    &:after {
                        content: url('~@/assets/img/down.png');
                        position: absolute;
                        right: -5px;
                        top: -1px;
                    }
                }

                &.equal {
                    &:after {
                        content: '';
                        position: absolute;
                        right: -5px;
                        top: 50%;
                        width: 7px;
                        height: 2px;
                        background-color: #ffd064;
                        margin-top: -1px;
                    }
                }

                &.up {
                    &:after {
                        content: url('~@/assets/img/up.png');
                        position: absolute;
                        right: -5px;
                        top: -1px;
                    }
                }
            }
        }

        a.btn {
            line-height: 1;
            font-size: 14px;
            text-align: center;
            color: #4d4d4d;
            border: 1px solid #e8e9eb;
            border-radius: 4px;
            padding: 12px 15px;
            background-color: #fafafa;
            background-repeat: no-repeat;
            background-position: center;
            text-decoration: none;
            transition: border-color .15s, background-color .15s, opacity .15s;
            cursor: pointer;
            overflow: visible;
            display: block;
            background-color: #fff;
            margin-top: 20px;
        }
    }
}

.rate-we7 {
    background-color: #fff;
    border: 1px solid $--border-color-base;
    height: 32px;
    line-height: 32px;
    display: inline-block;
    padding-left: 10px;
    color: #4d4d4d;

    span {
        margin-left: 40px;
        background-color: $--color-primary;
        display: inline-block;
        padding: 0 10px;
        color: #fff;
        max-height: 100%;
    }

    &.up-theme {
        span {
            background-color: #e23232;

            &::before {
                content: '高 '
            }
        }
    }

    &.down-theme {
        span {
            background-color: $--color-primary;

            &::before {
                content: '低 '
            }
        }
    }

    &.equal-theme {
        span {
            font-size: 0;
            background-color: #ffd064;

            &::before {
                content: '持平';
                font-size: 14px;
            }
        }
    }
}

.user-im {
    border-top: 1px dashed #e8e9eb;
    margin: 24px -10px 0;
    padding: 20px 10px 0;
}

.im-button {
    border: 1px solid #2d5fff;
    color: #2d5fff;
    border-radius: 3px;
    display: block;
    position: relative;
    width: 100%;
    text-align: center;
    line-height: 34px;
    font-size: 14px;
}

.bottom-panel {
    display: flex;
    margin-top: 20px;

    .description-box {
        flex: 0 0 956px;
        overflow: hidden;
    }
}

.site-dialog {
    .el-dialog {
        &__header {
            padding: 20px;
            line-height: 1;
        }

        &__title {
            font-size: 14px;
            line-height: 1;
        }

        &__body {
            padding: 0;
        }
    }
}
</style>
