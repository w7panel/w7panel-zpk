<template>
    <div class="storage-page" v-loading="pageLoading">
        <div>
            <el-alert v-if="storageType !== 's3'" type="info" style="color: #2d5fff" :closable="false" show-icon
                title="提示" class="mb-20">
                <template #default>
                    <div class="df jc-b">
                        <span style="color: var(--gray-800);">修改存储配置前，请先完成历史数据迁移，否则切换后旧镜像可能无法继续拉取。</span>
                        <a style="text-decoration: none;color: #2d5fff;"
                    href="https://wiki.w7.com/document/2575/8496" target="_blank">迁移文档</a>
                    </div>
                </template>
            </el-alert>

            <el-form ref="ruleFormRef" :model="form" :rules="rules" label-width="120px" label-position="left"
                class="setting-form">

                <el-form-item label="s3 服务地址" prop="s3_endpoint">
                    <el-input v-model.trim="form.s3_endpoint" placeholder="请输入 S3 Endpoint"
                        @input="handleEndpointInput">
                        <template #prepend>
                            <el-select v-model="form.s3_protocol" class="protocol-select" @change="resetChecked">
                                <el-option label="http" value="http" />
                                <el-option label="https" value="https" />
                            </el-select>
                        </template>
                    </el-input>
                </el-form-item>

                <el-form-item label="s3 存储地区" prop="s3_region">
                    <el-input v-model.trim="form.s3_region" placeholder="请输入 S3 Region" @input="resetChecked" />
                </el-form-item>

                <el-form-item label="s3 存储桶" prop="s3_bucket">
                    <el-input v-model.trim="form.s3_bucket" placeholder="请输入 Bucket" @input="resetChecked" />
                </el-form-item>

                <el-form-item label="根目录" prop="s3_root_directory">
                    <el-input v-model.trim="form.s3_root_directory" placeholder="请输入根目录，如 / 或 registry"
                        @input="resetChecked" />
                </el-form-item>

                <el-form-item label="Access Key" prop="s3_access_key">
                    <el-input v-model.trim="form.s3_access_key" placeholder="请输入 Access Key" @input="resetChecked" />
                </el-form-item>

                <el-form-item label="Secret Key" prop="s3_secret_key">
                    <el-input v-model="form.s3_secret_key" type="password" show-password placeholder="请输入 Secret Key"
                        @input="resetChecked" />
                </el-form-item>

                <el-form-item>
                    <div class="action-row">
                        <el-button type="primary" :loading="checking" @click="checkS3Status">测试连接</el-button>
                        <el-button type="success" :loading="submitting" :disabled="!checked" @click="saveStorageConfig">
                            保存设置
                        </el-button>
                    </div>
                    <div v-if="checked" class="checked-text">
                        连通性测试已通过，可以保存设置。
                    </div>
                </el-form-item>
            </el-form>
        </div>
    </div>
</template>

<script>
import myAxios from '@/utils';

export default {
    name: 'S3Page',
    data() {
        return {
            pageLoading: false,
            checking: false,
            submitting: false,
            checked: false,
            statusError: '',
            storageType: '',
            configInfo: {},
            form: {
                s3_protocol: 'https',
                s3_access_key: '',
                s3_bucket: '',
                s3_endpoint: '',
                s3_region: '',
                s3_root_directory: '',
                s3_secret_key: '',
            },
            rules: {
                s3_endpoint: [{ required: true, message: '请输入 S3 Endpoint', trigger: 'blur' }],
                s3_region: [{ required: true, message: '请输入 S3 Region', trigger: 'blur' }],
                s3_bucket: [{ required: true, message: '请输入 Bucket', trigger: 'blur' }],
                s3_root_directory: [{ required: true, message: '请输入根目录，如 / 或 registry', trigger: 'blur' }],
                s3_access_key: [{ required: true, message: '请输入 Access Key', trigger: 'blur' }],
                s3_secret_key: [{ required: true, message: '请输入 Secret Key', trigger: 'blur' }],
            },
        };
    },
    created() {
        this.getStatus();
    },
    methods: {
        getStatus(showLoading = true) {
            if (showLoading) {
                this.pageLoading = true;
            }
            return myAxios.get(`/system/util/registry/storage/config/get`).then((res) => {
                const config = this.normalizeConfig(res?.data);
                this.applyConfig(config);
                return config;
            }).finally(() => {
                this.pageLoading = false;
            });
        },
        checkS3Status() {
            this.validateForm(() => {
                this.checking = true;
                this.statusError = '';
                myAxios.post(`/system/util/registry/storage/s3/test`, this.buildS3Payload()).then(() => {
                    this.checked = true;
                    this.$message.success('连通性测试成功');
                }).finally(() => {
                    this.checking = false;
                });
            });
        },
        saveStorageConfig() {
            this.validateForm(() => {
                if (!this.checked) {
                    this.$message.warning('请先完成连通性测试');
                    return;
                }
                this.submitting = true;
                this.statusError = '';
                myAxios.post(`/system/util/registry/storage/config/update`, this.buildUpdatePayload()).then(() => {
                    this.$message.success('存储设置成功');
                    this.storageType = 's3';
                    this.checked = false;
                    this.getStatus(false);
                }).finally(() => {
                    this.submitting = false;
                });
            });
        },
        normalizeConfig(data) {
            if (!data || typeof data !== 'object') {
                return {};
            }
            if (data.info && typeof data.info === 'object') {
                return data.info;
            }
            if (data.data?.info && typeof data.data.info === 'object') {
                return data.data.info;
            }
            if (data.data && typeof data.data === 'object') {
                return data.data;
            }
            return data;
        },
        applyConfig(config = {}) {
            this.configInfo = config;
            this.storageType = config.storage_type || '';
            this.statusError = '';
            this.applyS3Config(config);
        },
        applyS3Config(config = {}) {
            const source = this.findS3Source(config);
            if (!source) {
                return;
            }

            const endpointInfo = this.normalizeEndpointValue(this.pickConfigValue(source, ['s3_endpoint', 'endpoint']));
            const protocol = endpointInfo.protocol || this.normalizeProtocolValue(this.pickConfigValue(source, ['s3_protocol', 'protocol', 'scheme', 'endpoint_protocol']));
            const nextForm = {
                s3_protocol: protocol,
                s3_access_key: this.pickConfigValue(source, ['s3_access_key', 'access_key']),
                s3_bucket: this.pickConfigValue(source, ['s3_bucket', 'bucket']),
                s3_endpoint: endpointInfo.endpoint,
                s3_region: this.pickConfigValue(source, ['s3_region', 'region']),
                s3_root_directory: this.pickConfigValue(source, ['s3_root_directory', 'root_directory']) || '/registry',
                s3_secret_key: this.pickConfigValue(source, ['s3_secret_key', 'secret_key']),
            };
            let hasChanged = false;

            Object.keys(nextForm).forEach((key) => {
                const value = nextForm[key];
                if (value === undefined || value === null) {
                    return;
                }
                const normalizedValue = String(value);
                if (this.form[key] !== normalizedValue) {
                    this.form[key] = normalizedValue;
                    hasChanged = true;
                }
            });

            if (hasChanged) {
                this.checked = false;
            }
        },
        findS3Source(config = {}) {
            const candidates = [
                config.s3,
                config.data?.s3,
                config.info?.s3,
                config,
            ];

            return candidates.find((item) => {
                return item && typeof item === 'object' && [
                    's3_access_key',
                    'access_key',
                    's3_bucket',
                    'bucket',
                    's3_endpoint',
                    'endpoint',
                    's3_region',
                    'region',
                    's3_root_directory',
                    'root_directory',
                    's3_secret_key',
                    'secret_key',
                ].some((key) => Object.prototype.hasOwnProperty.call(item, key));
            });
        },
        pickConfigValue(source, keys = []) {
            const key = keys.find((item) => Object.prototype.hasOwnProperty.call(source, item));
            return key ? source[key] : undefined;
        },
        normalizeEndpointValue(value) {
            if (value === undefined || value === null) {
                return {};
            }
            const endpoint = String(value).trim();
            const matched = endpoint.match(/^(https?):\/\/(.*)$/i);
            if (!matched) {
                return { endpoint };
            }
            return {
                protocol: matched[1].toLowerCase(),
                endpoint: matched[2],
            };
        },
        normalizeProtocolValue(value) {
            if (value === undefined || value === null) {
                return undefined;
            }
            const protocol = String(value).trim().replace(/:\/\//, '').toLowerCase();
            return ['http', 'https'].includes(protocol) ? protocol : undefined;
        },
        getFullEndpoint() {
            const endpoint = this.form.s3_endpoint.replace(/^https?:\/\//i, '');
            return `${this.form.s3_protocol}://${endpoint}`;
        },
        buildS3Payload() {
            return {
                s3_access_key: this.form.s3_access_key,
                s3_bucket: this.form.s3_bucket,
                s3_endpoint: this.getFullEndpoint(),
                s3_region: this.form.s3_region,
                s3_root_directory: this.form.s3_root_directory,
                s3_secret_key: this.form.s3_secret_key,
            };
        },
        buildUpdatePayload() {
            return this.buildS3Payload();
        },
        validateForm(callback) {
            this.$refs.ruleFormRef.validate((valid) => {
                if (!valid) {
                    return false;
                }
                callback && callback();
            });
        },
        handleEndpointInput(value) {
            const endpointInfo = this.normalizeEndpointValue(value);
            if (endpointInfo.protocol && endpointInfo.endpoint !== this.form.s3_endpoint) {
                this.form.s3_protocol = endpointInfo.protocol;
                this.form.s3_endpoint = endpointInfo.endpoint;
            }
            this.resetChecked();
        },
        resetChecked() {
            this.checked = false;
        },
    },
}
</script>

<style scoped lang="scss">
.page-card {
    max-width: 760px;
    background: #ffffff;
    box-sizing: border-box;
}

.title {
    font-size: 20px;
    font-weight: 600;
    color: #1d2129;
    line-height: 28px;
}

.subtitle {
    margin-top: 8px;
    margin-bottom: 20px;
    font-size: 14px;
    line-height: 22px;
    color: #4e5969;
}

.setting-form {
    max-width: 560px;
}

.action-row {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
}

.checked-text {
    font-size: 13px;
    color: #00a870;
    line-height: 32px;
    margin-left: 10px;
}

.status-text {
    padding: 12px 14px;
    border-radius: 8px;
    color: #4e5969;
    line-height: 22px;
    font-size: 14px;
    word-break: break-all;
}

.mb-20 {
    margin-bottom: 20px;
}

.protocol-select {
    width: 96px;
}
</style>
