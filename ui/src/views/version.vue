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
            <div class="zpk-page-header-actions">
                <a-button type="outline" :disabled="!info.manifest" @click="openYamlPreview">预览 YAML</a-button>
            </div>
        </div>
        <div class="content version-page-content pt-0">
        <a-tabs v-model:active-key="activeTab" @change="handleTabClick">
            <a-tab-pane key="version" title="版本管理">
                <div>
                    <div class="version-toolbar">
                        <a-button type="primary"
                            @click="form = { show: true, edit: false, version: '', description: '' }">
                            <template #icon><icon-plus /></template>
                            新建版本</a-button>
                        <a-checkbox v-model="crossUpgrade.enabled" class="cross-upgrade-check"
                            @change="handleCrossUpgradeEnabledChange">
                            跨应用升级
                        </a-checkbox>
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

                                        <a-tooltip v-if="audit_status > 1" content="应用已发布至制品市场" position="top">
                                            <a class="ml-10 cursor c-blue" target="_blank"
                                                :href="'https://zm.idc.w7.com/#/site-detail/' + goods_id">
                                                <IconCloud />
                                                <span class="ml-4">{{ {2: '待审核', 3: '审核失败', 4: '审核通过'}[audit_status] }}</span>
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
                            <div v-if="crossUpgrade.enabled" class="cross-upgrade-panel">
                                <div class="cross-upgrade-title">跨应用升级配置</div>
                                <div class="cross-upgrade-flow">
                                    <div class="cross-upgrade-app">
                                        <div class="cross-upgrade-icon">
                                            <img v-if="currentAppLogo" :src="currentAppLogo" alt="" />
                                            <span v-else>{{ currentAppInitial }}</span>
                                        </div>
                                        <div class="cross-upgrade-name">{{ currentAppName }}</div>
                                    </div>
                                    <div class="cross-upgrade-arrow">
                                        <IconArrowRight />
                                    </div>
                                    <div class="cross-upgrade-target-list">
                                        <div v-for="item in selectedCrossUpgradeTargets" :key="item.identifie"
                                            class="cross-upgrade-target-card">
                                            <button class="cross-upgrade-remove"
                                                @click.stop="removeCrossUpgradeTarget(item.identifie)">×</button>
                                            <div class="cross-upgrade-target-icon">
                                                <img v-if="getCrossUpgradeLogo(item)" :src="getCrossUpgradeLogo(item)"
                                                    alt="" />
                                                <span v-else>{{ getCrossUpgradeInitial(item) }}</span>
                                            </div>
                                            <div class="cross-upgrade-target-name">{{ item.title || item.name || item.identifie }}</div>
                                        </div>
                                        <div class="cross-upgrade-add-card">
                                            <button type="button" class="cross-upgrade-picker"
                                                @click="openCrossUpgradeDialog"></button>
                                            <div class="cross-upgrade-add-icon">+</div>
                                            <div class="cross-upgrade-target-label">选择应用</div>
                                        </div>
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
                        <div class="version-section-header">
                            <div class="c-16 b">开发版本</div>
                            <a-checkbox v-model="marketPublish.enabled" :disabled="marketPublish.saving"
                                @change="handleMarketPublishEnabledChange">
                                自动发布到制品市场
                            </a-checkbox>
                        </div>
                        <div class="mt-10">
                            <div v-for="(item, index) in list" :key="index" class="item version-dev-item">
                                <div class="version-dev-main">
                                    <div class="c-66">版本号</div>
                                    <div class="mt-20 version-title-row">
                                        <span class="lh-1 fs-20 b">{{ item.name }}</span>

                                        <a-tooltip
                                            v-if="item.id == version.id && goods_id && audit_status && audit_status < 4"
                                            content="应用已发布至微擎云市场，等待管理员审核" position="top">
                                            <a-tag class="version-status-tag" color="orange">待审核</a-tag>
                                        </a-tooltip>

                                        <a-tag class="version-status-tag publish-status"
                                            :color="getPublishStatusColor(item.publish_status)">
                                            {{ getPublishStatusText(item.publish_status) }}
                                        </a-tag>
                                        <a-tag v-if="item.id == version.id"
                                            class="version-status-tag" color="green">线上版本</a-tag>
                                        <a-button v-if="canPublishVersion(item)" class="publish-action-button"
                                            type="outline" @click="toPublish(item)" size="mini">点击发布</a-button>
                                        <template v-else-if="isPublishedVersion(item)">
                                            <a-button class="publish-action-button"
                                                status="danger"
                                                type="outline" @click="toUnpublish(item)" size="mini">点击下架</a-button>
                                            <a-button v-if="!isOnlineVersion(item)" class="publish-action-button"
                                                type="outline" @click="toPublish(item)" size="mini">点击发布</a-button>
                                        </template>

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
                <div class="version-paid-panel">
                    <a-form :model="instFee" ref="instFee" :rules="rules" label-align="left"
                        class="version-paid-form"
                        :label-col-props="{ flex: '0 0 96px' }" :wrapper-col-props="{ flex: '1' }">
                        <a-form-item label="授权费用" field="service_fee">
                            <div>
                                <a-input style="width: 200px;" v-model="instFee.service_fee" type="number" placeholder="请输入授权费用">
                                    <template #append>元</template>
                                </a-input>
                                <div class="version-form-help">用户获得该制品商业使用权的授权费用</div>
                            </div>
                        </a-form-item>

                        <a-form-item label="升级服务费用">
                            <div class="version-setting-block">
                                <a-alert type="info" show-icon class="zpk-primary-alert version-paid-alert mb-20" title="提示" :closable="false">
                                    <div class="registry-alert-item">根据升级规则，小版本升级时会跨版本跳过，升级服务只能指定大版本</div>
                                    <div class="registry-alert-item mt-6">当用户升级时遇到多个收费版本，只需要购买最后一个大版本</div>
                                    <div class="registry-alert-item mt-6">当设置所有版本付费后，用户每次大版本升级都需要付费</div>
                                    <div class="registry-alert-item mt-6">当设置周期服务费用后，用户如果在服务期内，所有版本升级不再次收费</div>
                                </a-alert>
                                <manifest-config-table :rows="instFee.version_prices" add-text="添加升级服务"
                                    @add="addVersionPrice">
                                    <template #columns>
                                        <manifest-config-table-column data-index="version" title="版本">
                                            <template #cell="{ record }">
                                                <a-select v-model="record.version" placeholder="请选择">
                                                    <a-option v-for="vl in instFee.can_upgrade_versions" :key="vl"
                                                        :value="vl"
                                                        :label="vl === 9999 ? '所有版本' : (vl + '.*.*')"></a-option>
                                                </a-select>
                                            </template>
                                        </manifest-config-table-column>
                                        <manifest-config-table-column data-index="price" title="价格" width="260px">
                                            <template #cell="{ record }">
                                                <a-input v-model="record.price" type="number" placeholder="请输入">
                                                    <template #append>元</template>
                                                </a-input>
                                            </template>
                                        </manifest-config-table-column>
                                        <manifest-config-table-column title="操作" width="100px">
                                            <template #cell="{ index }">
                                                <span class="c-blue cursor"
                                                    @click="instFee.version_prices.splice(index, 1)">删除</span>
                                            </template>
                                        </manifest-config-table-column>
                                    </template>
                                </manifest-config-table>
                            </div>
                        </a-form-item>

                        <a-form-item label="周期服务费用">
                            <div class="version-setting-block">
                                <div class="df ai-c">
                                    <a-switch v-model="instFee.enable_service_package_fee" />
                                    <span class="c-99" style="margin-left:10px;">开启后可配置周期费用。</span>
                                </div>
                                <div class="mt-6" style="line-height:18px;">
                                    <span class="warning-icon" style="display:inline-block; vertical-align:middle;">!</span>
                                    <span class="c-99"
                                        style="margin-left:4px; vertical-align:middle;">到期后无法维护更新,需要再次购买服务周期套餐才可以维护更新</span>
                                </div>
                                <manifest-config-table v-if="instFee.enable_service_package_fee" class="mt-10" :rows="instFee.service_packages"
                                    add-text="添加服务周期" @add="addServicePackage">
                                    <template #columns>
                                        <manifest-config-table-column data-index="price" title="价格" width="200px">
                                            <template #cell="{ record }">
                                                <a-input v-model="record.price" type="number" placeholder="请输入">
                                                    <template #append>元</template>
                                                </a-input>
                                            </template>
                                        </manifest-config-table-column>
                                        <manifest-config-table-column data-index="month" title="时长">
                                            <template #cell="{ record }">
                                                <a-select v-model="record.month" placeholder="请选择">
                                                    <a-option label="1年" :value="12"></a-option>
                                                    <a-option label="2年" :value="24"></a-option>
                                                    <a-option label="3年" :value="36"></a-option>
                                                    <a-option label="4年" :value="48"></a-option>
                                                    <a-option label="5年" :value="60"></a-option>
                                                </a-select>
                                            </template>
                                        </manifest-config-table-column>
                                        <manifest-config-table-column title="赠送" width="90px">
                                            <template #cell="{ record, index }">
                                                <a-tooltip content="勾选后选择当前套餐赠送，自动取消别套餐。">
                                                    <a-checkbox v-model="record.is_gift"
                                                        @change="() => setGiftServicePackage(index)"></a-checkbox>
                                                </a-tooltip>
                                            </template>
                                        </manifest-config-table-column>
                                        <manifest-config-table-column title="生效" width="90px">
                                            <template #cell="{ record }">
                                                <a-checkbox v-model="record.enabled"></a-checkbox>
                                            </template>
                                        </manifest-config-table-column>
                                        <manifest-config-table-column title="操作" width="100px">
                                            <template #cell="{ index }">
                                                <span class="c-blue cursor"
                                                    @click="instFee.service_packages.splice(index, 1)">删除</span>
                                            </template>
                                        </manifest-config-table-column>
                                    </template>
                                </manifest-config-table>
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
    <a-modal v-model:visible="dialogVisible" :footer="false">
        <img w-full :src="dialogImageUrl" alt="Preview Image" />
    </a-modal>
    <a-drawer v-model:visible="yamlPreview.show" :width="640" title="预览 YAML" :footer="false" unmount-on-close>
        <div class="yaml-preview-panel">
            <div class="yaml-preview-drawer" v-html="yamlPreview.html"></div>
            <div class="yaml-preview-actions">
                <button class="copybtn" @click="copyYamlPreview">一键复制</button>
                <a :href="yamlPreview.downloadUrl" download="manifest.yaml" class="copybtn">下载</a>
            </div>
        </div>
    </a-drawer>
    <a-modal v-model:visible="crossUpgrade.dialogVisible" title="选择应用" :width="720" :footer="false"
        modal-class="cross-upgrade-dialog" unmount-on-close>
        <a-spin :loading="crossUpgrade.loading">
            <div v-if="crossUpgrade.candidates.length" class="cross-upgrade-dialog-grid">
                <div v-for="item in crossUpgrade.candidates" :key="item.identifie"
                    class="cross-upgrade-dialog-item"
                    :class="{ active: crossUpgrade.identifies.includes(item.identifie) }"
                    @click="toggleCrossUpgradeTarget(item.identifie)">
                    <div class="cross-upgrade-dialog-icon">
                        <img v-if="getCrossUpgradeLogo(item)" :src="getCrossUpgradeLogo(item)" alt="" />
                        <span v-else>{{ getCrossUpgradeInitial(item) }}</span>
                    </div>
                    <div class="cross-upgrade-dialog-title">{{ item.title || item.name || item.identifie }}</div>
                </div>
            </div>
            <a-empty v-else description="暂无可选应用" />
        </a-spin>
    </a-modal>
</template>

<script>
import myAxios from '@/utils'
import versionInfo from '@/components/version-info.vue';
import jsyaml from "js-yaml";
import description from './description.vue';
import publishSettings from './publish-settings.vue';
import ManifestConfigTable from '@/components/manifest-config-table.vue';
import ManifestConfigTableColumn from '@/components/manifest-config-table-column.vue';
import userMixin from "@/utils/user-mixin";
import { messageError, messageSuccess } from '@/utils/ui-feedback';
import { IconArrowLeft, IconArrowRight, IconPlus, IconCloud } from '@arco-design/web-vue/es/icon';

export default {
    components: {
        versionInfo,
        description,
        publishSettings,
        ManifestConfigTable,
        ManifestConfigTableColumn,
        IconArrowLeft,
        IconArrowRight,
        IconPlus,
        IconCloud,
    },
    mixins: [userMixin],
    data() {
        return {
            activeTab: 'version',
            identifie: '',
            title: '',
            baseurl: '',
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
                enable_service_package_fee: false,
                service_fee: '',
                service_packages: [],
                version_prices: [],
                can_upgrade_versions: [],
            },

            crossUpgrade: {
                enabled: false,
                loading: false,
                saving: false,
                settingLoading: false,
                dialogVisible: false,
                formulas: [],
                candidates: [],
                identifies: [],
                setting: {},
            },
            marketPublish: {
                enabled: false,
                saving: false,
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
            yamlPreview: {
                show: false,
                yaml: '',
                html: '',
                downloadUrl: '',
            },
        }
    },
    created() {
        this.is_register = window?.$wujie?.props?.isRegister;
        this.baseurl = window?.$wujie?.props?.url || '';
        this.identifie = this.$route.query.id;
        this.title = this.$route.query.title;
        this.getInfo();
        this.getList();
        this.getCrossUpgradeFormulaCandidates();
        this.getFormulaSetting();
    },
    computed: {
        currentAppName() {
            return this.info?.title || this.manifestAppName || this.title || this.identifie || '-';
        },
        manifestAppName() {
            if (!this.info?.manifest) { return '' }
            try {
                let json = jsyaml.load(this.info.manifest);
                return json?.platform?.baseInfo?.name || json?.application?.name || '';
            } catch {
                return '';
            }
        },
        currentAppLogo() {
            let logo = this.info?.icon_url || this.info?.icon || '';
            if (logo && !/^https?:\/\//.test(logo)) {
                return this.baseurl + logo;
            }
            return logo;
        },
        currentAppInitial() {
            return String(this.currentAppName || this.identifie || '应').slice(0, 1);
        },
        selectedCrossUpgradeTargets() {
            return this.crossUpgrade.identifies.map(identifie => {
                return this.crossUpgrade.candidates.find(item => item.identifie === identifie)
                    || this.crossUpgrade.formulas.find(item => item.identifie === identifie)
                    || { identifie };
            });
        },
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
        openYamlPreview() {
            let yaml = this.info?.manifest || '';
            if (!yaml) { return }
            this.yamlPreview.show = true;
            this.$nextTick(() => {
                this.yamlPreview.yaml = yaml;
                this.yamlPreview.html = `<pre class='pre'><code class='language-yaml'>${this.escapeHtml(yaml)}</code></pre>`;
                let file = new File([yaml], 'manifest.yaml', { type: 'text/plain' });
                this.yamlPreview.downloadUrl = URL.createObjectURL(file);
                this.$nextTick(() => {
                    window.hljs.highlightAll();
                });
            });
        },
        escapeHtml(text) {
            return String(text || '')
                .replace(/&/g, '&amp;')
                .replace(/</g, '&lt;')
                .replace(/>/g, '&gt;')
                .replace(/"/g, '&quot;')
                .replace(/'/g, '&#39;');
        },
        copyYamlPreview() {
            let text = this.yamlPreview.yaml || '';
            if (navigator.clipboard && window.isSecureContext) {
                navigator.clipboard.writeText(text);
            } else {
                let textarea = document.createElement('textarea');
                document.body.appendChild(textarea);
                textarea.style.position = 'fixed';
                textarea.style.clip = 'rect(0 0 0 0)';
                textarea.style.top = '10px';
                textarea.value = text;
                textarea.select();
                document.execCommand('copy', true);
                document.body.removeChild(textarea);
            }
            messageSuccess('复制成功');
        },
        getCuv() {
            myAxios.post('/respo/goods/can-upgrade-versions', {
                identifie: this.identifie,
            }).then(res => {
                this.instFee.can_upgrade_versions = res?.data?.data || [];
            })
        },
        getCrossUpgradeFormulaCandidates() {
            this.crossUpgrade.loading = true;
            myAxios.post('/respo/goods/cross-upgrade-formulas', {
                identifie: this.identifie,
            }).then(res => {
                this.crossUpgrade.candidates = res?.data?.data || [];
            }).finally(() => {
                this.crossUpgrade.loading = false;
            });
        },
        formatCrossUpgradeCandidate(item) {
            return `${item.title || item.name || item.identifie}（¥${item.price || 0}）`;
        },
        getFormulaSetting() {
            this.crossUpgrade.settingLoading = true;
            myAxios.post('/respo/setting/get', {
                identifie: this.identifie,
            }).then(res => {
                let setting = res?.data?.data || {};
                this.crossUpgrade.setting = setting;
                this.crossUpgrade.enabled = !!setting.support_cross_upgrade;
                this.marketPublish.enabled = !!(
                    setting.support_auto_publish_to_zpk_market
                    || setting.support_publish_to_zpk_market
                );
                this.instFee.enable_service_package_fee = !!setting.enable_service_package_fee;
            }).finally(() => {
                this.crossUpgrade.settingLoading = false;
            });
        },
        saveFormulaSetting() {
            this.crossUpgrade.settingLoading = true;
            this.marketPublish.saving = true;
            return myAxios.post('/respo/setting/set', {
                identifie: this.identifie,
                support_cross_upgrade: !!this.crossUpgrade.enabled,
                support_auto_publish_to_zpk_market: !!this.marketPublish.enabled,
                enable_service_package_fee: !!this.instFee.enable_service_package_fee,
            }).then(() => {
                this.crossUpgrade.setting = {
                    ...this.crossUpgrade.setting,
                    support_cross_upgrade: !!this.crossUpgrade.enabled,
                    support_auto_publish_to_zpk_market: !!this.marketPublish.enabled,
                    enable_service_package_fee: !!this.instFee.enable_service_package_fee,
                };
                messageSuccess('操作成功');
            }).finally(() => {
                this.crossUpgrade.settingLoading = false;
                this.marketPublish.saving = false;
            });
        },
        handleMarketPublishEnabledChange(checked) {
            this.marketPublish.enabled = checked;
            this.saveFormulaSetting();
        },
        normalizeCrossUpgradeIdentifiers(value) {
            return this.normalizeCrossUpgradeTargets(value).map(item => item.identifie).filter(Boolean);
        },
        normalizeCrossUpgradeTargets(value) {
            let list = value;
            if (typeof value === 'string') {
                try {
                    list = JSON.parse(value);
                } catch {
                    list = value ? [value] : [];
                }
            }
            if (!Array.isArray(list)) { return [] }
            return list.map(item => {
                if (typeof item === 'string') { return { identifie: item } }
                return item;
            }).filter(item => item?.identifie);
        },
        hydrateCrossUpgrade() {
            let formulas = this.normalizeCrossUpgradeTargets(this.info?.cross_upgrade_formulas);
            this.crossUpgrade.formulas = formulas;
            this.crossUpgrade.identifies = formulas.map(item => item.identifie);
        },
        handleCrossUpgradeEnabledChange(checked) {
            this.crossUpgrade.enabled = checked;
            if (checked) {
                this.saveFormulaSetting();
                this.getCrossUpgradeFormulaCandidates();
                return;
            }
            this.crossUpgrade.identifies = [];
            this.submitCrossUpgradeFormulas({ allowEmpty: true, silent: true }).finally(() => {
                this.saveFormulaSetting();
            });
        },
        openCrossUpgradeDialog() {
            this.crossUpgrade.dialogVisible = true;
            this.getCrossUpgradeFormulaCandidates();
        },
        toggleCrossUpgradeTarget(identifie) {
            if (this.crossUpgrade.identifies.includes(identifie)) {
                this.crossUpgrade.identifies = this.crossUpgrade.identifies.filter(item => item !== identifie);
            } else {
                this.crossUpgrade.identifies = [...this.crossUpgrade.identifies, identifie];
            }
            this.submitCrossUpgradeFormulas({ allowEmpty: true }).finally(() => {
                this.crossUpgrade.dialogVisible = false;
            });
        },
        removeCrossUpgradeTarget(identifie) {
            this.crossUpgrade.identifies = this.crossUpgrade.identifies.filter(item => item !== identifie);
            this.submitCrossUpgradeFormulas({ allowEmpty: true });
        },
        getCrossUpgradeLogo(item) {
            let logo = item?.icon || item?.Icon || item?.logo || item?.icon_url || item?.cdn_logo || '';
            if (logo && !/^https?:\/\//.test(logo)) {
                return this.baseurl + logo;
            }
            return logo;
        },
        getCrossUpgradeInitial(item) {
            return String(item?.title || item?.name || item?.identifie || '应').slice(0, 1);
        },
        submitCrossUpgradeFormulas(options = {}) {
            if (this.crossUpgrade.enabled && !this.crossUpgrade.identifies.length && !options.allowEmpty) {
                messageError('请选择可升级应用');
                return;
            }
            this.crossUpgrade.saving = true;
            return myAxios.post('/respo/goods/set-cross-upgrade-formulas', {
                identifie: this.identifie,
                cross_upgrade_formulas: this.crossUpgrade.enabled ? this.crossUpgrade.identifies : [],
            }).then(() => {
                if (!options.silent) {
                    messageSuccess('操作成功');
                }
                this.getInfo();
            }).finally(() => {
                this.crossUpgrade.saving = false;
            });
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


        addVersionPrice() {
            this.instFee.version_prices.push({ version: '', price: '' });
        },
        addServicePackage() {
            this.instFee.service_packages.push({ price: '', month: '', is_gift: false, enabled: false });
        },
        setGiftServicePackage(index) {
            this.instFee.service_packages.map((item, id) => {
                if (id != index) {
                    item.is_gift = false;
                }
            });
        },
        normalizeVersionPrices() {
            let o = {};
            let arr = [];
            this.instFee.version_prices.map(i => {
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
            return arr;
        },
        normalizeServicePackageRows(list) {
            return (list || []).map(item => ({
                ...item,
                enabled: item.enabled === true || item.enabled == 2,
                is_gift: item.is_gift === true || item.is_gift == 1,
            }));
        },
        normalizeServicePackages() {
            return this.instFee.service_packages?.filter(i => i.price !== '' && i.month)?.map(i => ({
                ...i,
                price: Number(i.price),
                month: Number(i.month),
                enabled: i.enabled ? 2 : 1,
            })) || [];
        },



        getInstFee() {
            let version_prices = (this.info?.version_prices || []).map(item => ({ ...item }));
            version_prices.map(i => {
                i.versionName = this.versionsKV[i.version];
            })
            this.instFee = {
                ...this.instFee,
                old_fee: this.info?.install_service_fee,
                service_fee: this.info?.install_service_fee || '',
                enable_service_package_fee: !!(this.info?.setting?.enable_service_package_fee ?? this.crossUpgrade.setting?.enable_service_package_fee),
                service_packages: this.normalizeServicePackageRows(this.info?.service_packages || []),
                version_prices: version_prices,
            }
        },
        submitInstFee() {
            this.$refs.instFee.validate((errors) => {
                if (errors) { return }
                let service_packages = this.normalizeServicePackages();
                let version_prices = this.normalizeVersionPrices();

                myAxios.post('/respo/goods/set-service-fee', {
                    identifie: this.identifie,
                    service_fee: Number(this.instFee.service_fee),
                    enable_service_package_fee: !!this.instFee.enable_service_package_fee,
                    service_packages,
                    version_prices,
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
                this.hydrateCrossUpgrade();
                if (!this.info.manifest) {
                    this.noPlatform = true;
                } else {
                    let json = jsyaml.load(this.info.manifest);
                    this.noPlatform = !json.platform;
                }

                let goodsid = res?.data?.data?.goods_id;
                if (!goodsid) { return }
                this.goods_id = goodsid;
                myAxios.post('/respo/goods/audit-status', { identifie: this.identifie }).then(res => {
                    this.audit_status = res?.data?.data?.audit_status;
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
            }).then(() => {
                this.publishGoods.loading = false;
                messageSuccess('操作成功')
                this.getInfo();
                this.getList();
            }).catch((error) => {
                this.publishGoods.loading = false;

                let str = !error?.response?.data?.error ? error?.response?.data?.message : '';
                if (str) {
                    messageError(str);
                }
            });
        },
        toUnpublish(item) {
            this.publishGoods.loading = true;
            myAxios.post('/respo/version-unpublish', {
                identifie: this.identifie,
                version: item.name
            }, {
                timeout: 0
            }).then(() => {
                this.publishGoods.loading = false;
                messageSuccess('操作成功')
                this.getInfo();
                this.getList();
            }).catch((error) => {
                this.publishGoods.loading = false;

                let str = !error?.response?.data?.error ? error?.response?.data?.message : '';
                if (str) {
                    messageError(str);
                }
            });
        },
        canPublishVersion(item) {
            return [-1, 3, '-1', '3'].includes(item.publish_status);
        },
        isOnlineVersion(item) {
            return item.id == this.version.id;
        },
        isPublishedVersion(item) {
            return [2, '2'].includes(item.publish_status);
        },
        getPublishStatusText(status) {
            return { '-1': '未发布', 1: '发布中', 2: '已发布', 3: '发布失败' }[status] || '未发布';
        },
        getPublishStatusColor(status) {
            return { '-1': 'gray', 1: 'blue', 2: 'green', 3: 'red' }[status] || 'gray';
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

.version-toolbar {
    display: flex;
    align-items: center;
    gap: 24px;
    flex-wrap: wrap;
}

.cross-upgrade-check {
    color: #333;
    font-size: 14px;
}

.cross-upgrade-panel {
    margin-top: 24px;
    padding-top: 22px;
    border-top: 1px solid #f0f0f0;
    background: #fff;
}

.cross-upgrade-title {
    color: #333;
    font-weight: 600;
    margin-bottom: 22px;
}

.cross-upgrade-flow {
    display: flex;
    align-items: flex-start;
    gap: 20px;
    min-width: 0;
}

.cross-upgrade-app {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
}

.cross-upgrade-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 64px;
    height: 64px;
    border: 1px solid #e7e7e7;
    border-radius: 4px;
    color: #666;
    font-size: 22px;
    font-weight: 600;
    background: #fff;
    overflow: hidden;
}

.cross-upgrade-icon img {
    width: 100%;
    height: 100%;
    object-fit: cover;
}

.cross-upgrade-name,
.cross-upgrade-target-name,
.cross-upgrade-target-label {
    max-width: 180px;
    color: #666;
    line-height: 20px;
    text-align: center;
    word-break: break-all;
}

.cross-upgrade-arrow {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 32px;
    width: 32px;
    height: 64px;
    color: #86909c;
    font-size: 20px;
}

.cross-upgrade-target-list {
    display: flex;
    align-items: flex-start;
    gap: 14px;
    min-width: 0;
    flex-wrap: wrap;
}

.cross-upgrade-target-card,
.cross-upgrade-add-card {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    min-width: 88px;
}

.cross-upgrade-target-icon,
.cross-upgrade-add-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 64px;
    height: 64px;
    border-radius: 4px;
    background: #f7f8fa;
    color: #999;
    font-size: 24px;
    overflow: hidden;
}

.cross-upgrade-target-icon {
    border: 1px solid #e7e7e7;
    background: #fff;
    color: #666;
    font-weight: 600;
}

.cross-upgrade-target-icon img {
    width: 100%;
    height: 100%;
    object-fit: cover;
}

.cross-upgrade-add-icon {
    border: 1px dashed #c9cdd4;
    cursor: pointer;
}

.cross-upgrade-picker {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    opacity: 0;
    border: 0;
    padding: 0;
    cursor: pointer;
    z-index: 2;
}

.cross-upgrade-remove {
    position: absolute;
    top: -7px;
    right: 4px;
    z-index: 1;
    width: 18px;
    height: 18px;
    padding: 0;
    border: 1px solid #d9d9d9;
    border-radius: 50%;
    background: #fff;
    color: #999;
    line-height: 16px;
    cursor: pointer;
}

.cross-upgrade-dialog-grid {
    display: grid;
    width: 100%;
    box-sizing: border-box;
    grid-template-columns: repeat(auto-fill, minmax(112px, 1fr));
    gap: 18px 12px;
    max-height: 480px;
    overflow: auto;
    padding: 2px;
}

.cross-upgrade-dialog :deep(.arco-spin) {
    display: block;
    width: 100%;
}

.cross-upgrade-dialog-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    padding: 8px 6px;
    cursor: pointer;
    border-radius: 8px;
}

.cross-upgrade-dialog-item:hover .cross-upgrade-dialog-icon,
.cross-upgrade-dialog-item.active .cross-upgrade-dialog-icon {
    border-color: #165dff;
    background: #f4f8ff;
}

.cross-upgrade-dialog-item.active .cross-upgrade-dialog-title {
    color: #165dff;
}

.cross-upgrade-dialog-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 56px;
    height: 56px;
    border: 1px solid #e7e7e7;
    border-radius: 8px;
    background: #fff;
    color: #666;
    font-size: 22px;
    font-weight: 600;
    overflow: hidden;
}

.cross-upgrade-dialog-icon img {
    width: 100%;
    height: 100%;
    object-fit: cover;
}

.cross-upgrade-dialog-title {
    max-width: 100px;
    color: #333;
    line-height: 20px;
    text-align: center;
    word-break: break-all;
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

.version-section-header {
    display: flex;
    align-items: center;
    justify-content: flex-start;
    gap: 12px;
    flex-wrap: wrap;
}

.version-title-row {
    display: flex;
    align-items: center;
    min-height: 32px;
    min-width: 0;
}

.version-status-tag {
    margin-left: 10px;
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

.version-paid-panel {
    width: 860px;
    max-width: 100%;
}

.version-paid-alert {
    margin-bottom: 20px;
}

.version-setting-block {
    width: 100%;
}

.version-form-help {
    margin-top: 6px;
    color: var(--color-text-3);
    font-size: 12px;
    line-height: 18px;
}

.version-paid-form :deep(.arco-form-item-label-col) {
    flex: 0 0 120px !important;
    width: 120px;
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

.publish-status {
    flex: 0 0 auto;
}

.publish-action-button {
    margin-left: 10px;
    display: none;
}

.gray-box .item:hover .publish-action-button {
    display: inline-flex;
    align-items: center;
    line-height: unset;
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

    .cross-upgrade-flow {
        align-items: flex-start;
        flex-wrap: wrap;
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

.cross-upgrade-dialog .arco-spin {
    display: block;
    width: 100%;
}
</style>
