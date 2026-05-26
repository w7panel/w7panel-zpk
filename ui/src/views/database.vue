<template>
    <div class="database-page" v-loading="pageLoading">
        <div class="page-card">
            <div class="title">数据库设置</div>
            <div class="subtitle">MySQL数据库设置</div>

            <el-alert v-if="statusError" type="error" :closable="false" show-icon title="设置失败"
                :description="statusError" class="mb-20" />

            <el-alert v-else-if="migrating" type="primary" :closable="false" show-icon title="设置中"
                description="正在配置数据库，页面会自动轮询状态。" class="mb-20" />

            <el-form v-if="showForm" ref="ruleFormRef" :model="form" :rules="rules" label-width="110px"
                label-position="left" class="migration-form">
                <el-form-item label="MySQL 地址" prop="host">
                    <el-input v-model.trim="form.host" placeholder="请输入 MySQL Host" @input="resetChecked" />
                </el-form-item>

                <el-form-item label="MySQL 端口" prop="port">
                    <el-input v-model.trim="form.port" placeholder="请输入 MySQL Port" @input="resetChecked" />
                </el-form-item>

                <el-form-item label="数据库名" prop="database">
                    w7-cd-artifact
                </el-form-item>

                <el-form-item label="用户名" prop="username">
                    <el-input v-model.trim="form.username" placeholder="请输入用户名" @input="resetChecked" />
                </el-form-item>

                <el-form-item label="密码" prop="password">
                    <el-input v-model="form.password" type="password" show-password placeholder="请输入密码"
                        @input="resetChecked" />
                </el-form-item>

                <el-form-item>
                    <div class="action-row">
                        <el-button type="primary" :loading="checking" @click="checkMysqlStatus">测试连接</el-button>
                        <el-button type="success" :loading="submitting" :disabled="!checked" @click="switchToMysql">
                            提交
                        </el-button>
                    </div>
                    <div v-if="checked" class="checked-text">
                        连通性测试已通过，可以提交
                    </div>
                </el-form-item>
            </el-form>
        </div>
    </div>
</template>

<script>
import myAxios from '@/utils';

export default {
    name: 'DatebasePage',
    data() {
        return {
            pageLoading: false,
            checking: false,
            submitting: false,
            migrating: false,
            checked: false,
            canSwitch: null,
            statusError: '',
            statusMessage: '',
            pollTimer: null,
            form: {
                host: '',
                port: '',
                username: '',
                password: '',
            },
            rules: {
                host: [{ required: true, message: '请输入 MySQL 地址', trigger: 'blur' }],
                port: [{ required: true, message: '请输入 MySQL 端口', trigger: 'blur' }],
                username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
                password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
            },
        };
    },
    computed: {
        showForm() {
            return (this.canSwitch === true) && !this.migrating;
        },
    },
    created() {
        this.getStatus();
    },
    beforeUnmount() {
        this.clearPollTimer();
    },
    methods: {
        getStatus(showLoading = true) {
            if (showLoading) {
                this.pageLoading = true;
            }
            return myAxios.get(`/system/util/db/switch/status`).then((res) => {
                const status = this.normalizeStatus(res?.data);
                this.applyStatus(status);
                return status;
            }).finally(() => {
                this.pageLoading = false;
            });
        },
        checkMysqlStatus() {
            this.validateForm(() => {
                this.checking = true;
                this.statusError = '';
                myAxios.post(`/system/util/db/switch/mysql/test`, this.buildPayload()).then(() => {
                    this.checked = true;
                    this.$message.success('MySQL 连通性测试成功');
                }).finally(() => {
                    this.checking = false;
                });
            });
        },
        switchToMysql() {
            this.validateForm(() => {
                if (!this.checked) {
                    this.$message.warning('请先完成 MySQL 连通性测试');
                    return;
                }
                this.submitting = true;
                this.statusError = '';
                myAxios.post(`/system/util/db/switch/mysql/run`, this.buildPayload()).then(() => {
                    this.migrating = true;
                    this.statusMessage = 'MySQL 连接信息已提交'
                    this.startPolling();
                }).finally(() => {
                    this.submitting = false;
                });
            });
        },
        startPolling() {
            this.clearPollTimer();
            const poll = () => {
                this.getStatus(false).then((status) => {
                    if (status?.error) {
                        this.clearPollTimer();
                        return;
                    }
                    if (status?.can_switch === false) {
                        this.clearPollTimer();
                        return;
                    }
                    if (this.migrating) {
                        this.pollTimer = setTimeout(poll, 3000);
                    }
                }).catch(() => {
                    this.clearPollTimer();
                });
            };
            this.pollTimer = setTimeout(poll, 3000);
        },
        clearPollTimer() {
            if (!this.pollTimer) {
                return;
            }
            clearTimeout(this.pollTimer);
            this.pollTimer = null;
        },
        applyStatus(status = {}) {
            const canSwitch = typeof status.can_switch === 'boolean' ? status.can_switch : null;
            const progressText = `${status.message || ''} ${status.status || ''}`;
            this.canSwitch = canSwitch;
            this.statusMessage = '';

            if (status.error) {
                this.statusError = status.error;
                this.migrating = false;
                this.clearPollTimer();
                return;
            }

            this.statusError = '';
            if (canSwitch === false) {
                this.applyConnectionInfo(status);
                this.migrating = false;
                this.clearPollTimer();
                return;
            }

            this.applyConnectionInfo(status);
            if (/迁移中|running|processing|switching|migrat/i.test(progressText)) {
                this.migrating = true;
                if (!this.pollTimer) {
                    this.startPolling();
                }
                return;
            }

            this.migrating = false;
        },
        normalizeStatus(data) {
            if (!data || typeof data !== 'object') {
                return {};
            }
            if (data.data && typeof data.data === 'object' && !Object.prototype.hasOwnProperty.call(data, 'can_switch')) {
                return data.data;
            }
            return data;
        },
        applyConnectionInfo(status = {}) {
            const source = this.findConnectionSource(status);
            if (!source) {
                return;
            }

            const nextForm = {
                host: this.pickConnectionValue(source, ['host', 'old_host', 'mysql_host', 'db_host']),
                port: this.pickConnectionValue(source, ['port', 'old_port', 'mysql_port', 'db_port']),
                username: this.pickConnectionValue(source, ['username', 'user_name', 'old_username', 'old_user_name', 'mysql_username', 'db_username']),
                password: this.pickConnectionValue(source, ['password', 'old_password', 'mysql_password', 'db_password']),
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
        findConnectionSource(status = {}) {
            const candidates = [
                status,
                status.data,
                status.mysql,
                status.mysql_config,
                status.config,
                status.db,
                status.database,
                status.connection,
                status.old,
                status.old_mysql,
                status.old_config,
            ];

            return candidates.find((item) => {
                return item && typeof item === 'object' && [
                    'host',
                    'old_host',
                    'mysql_host',
                    'db_host',
                    'port',
                    'old_port',
                    'mysql_port',
                    'db_port',
                    'username',
                    'user_name',
                    'old_username',
                    'old_user_name',
                    'password',
                    'old_password',
                ].some((key) => Object.prototype.hasOwnProperty.call(item, key));
            });
        },
        pickConnectionValue(source, keys = []) {
            const key = keys.find((item) => Object.prototype.hasOwnProperty.call(source, item));
            return key ? source[key] : undefined;
        },
        buildPayload() {
            const host = this.form.host;
            const port = this.form.port;
            const username = this.form.username;
            const password = this.form.password;
            return {
                host,
                port,
                user_name: username,
                password,
            };
        },
        validateForm(callback) {
            this.$refs.ruleFormRef.validate((valid) => {
                if (!valid) {
                    return false;
                }
                callback && callback();
            });
        },
        resetChecked() {
            this.checked = false;
        },
    },
}
</script>

<style scoped lang="scss">
.database-page {
    padding: 20px;
}

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

.migration-form {
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
</style>
