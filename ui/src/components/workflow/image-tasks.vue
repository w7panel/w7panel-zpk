<template>
    <div>
        <a-space class="workflow-toolbar">
            <a-button type="primary" @click="editTask()" :disabled="loading || saving || !loaded">新增自动任务</a-button>
            <a-button @click="openOrder" :disabled="loading || saving || !loaded || !config.tasks.length">调整执行顺序</a-button>
        </a-space>
        <a-spin :loading="loading" style="width:100%">
            <a-table :data="orderedTasks" :pagination="false" row-key="id">
                <template #columns>
                    <a-table-column title="任务"><template #cell="{ record }">{{ record.name }}</template></a-table-column>
                    <a-table-column title="执行动作"><template #cell="{ record }">{{ actionLabels[record.action] }}</template></a-table-column>
                    <a-table-column title="触发条件"><template #cell="{ record }">{{ record.deploy_type === 1 ? '更新版本' : '新增版本' }}：{{ record.tag_name }}</template></a-table-column>
                    <a-table-column title="使用镜像"><template #cell="{ record }">{{ sourceLabel(record) }}</template></a-table-column>
                    <a-table-column title="执行时机" :width="240">
                        <template #cell="{ record }">{{ timingLabel(record) }}</template>
                    </a-table-column>
                    <a-table-column title="操作" :width="240">
                        <template #cell="{ record }">
                            <a-button type="text" @click="editTask(record)" :disabled="saving || loading">修改</a-button>
                            <a-button type="text" @click="showRuns(record)">执行记录</a-button>
                            <a-popconfirm content="确认删除此任务？" type="warning" ok-text="确定" cancel-text="取消"
                                content-class="zpk-delete-popconfirm" :ok-button-props="{ status: 'danger', loading: saving }"
                                :cancel-button-props="{ type: 'secondary', disabled: saving }" @ok="removeTask(record)"><a-button type="text" status="danger" :disabled="saving || loading">删除</a-button></a-popconfirm>
                        </template>
                    </a-table-column>
                </template>
            </a-table>
        </a-spin>
        <a-modal v-model:visible="orderVisible" title="调整执行顺序" :width="960" :footer="false" :mask-closable="false" :closable="!saving" :esc-to-close="!saving">
            <section v-for="group in orderGroups" :key="group.key" class="workflow-order-group">
                <div class="workflow-order-trigger">{{ triggerLabel(group.tasks[0]) }}</div>
                <a-table :data="group.tasks" :pagination="false" row-key="id">
                    <template #columns>
                        <a-table-column title="任务" data-index="name" />
                        <a-table-column title="执行时机" :width="230"><template #cell="{ record }">
                            <a-select :model-value="record.stage" :options="orderStageOptions(group.tasks)" :disabled="saving" @update:model-value="changeOrderStage(record, $event)" />
                        </template></a-table-column>
                        <a-table-column title="使用镜像" :width="260"><template #cell="{ record }">
                            <a-select :model-value="orderSourceValue(record)" :options="orderSourceOptions(record)" placeholder="请选择使用镜像" :disabled="saving" @update:model-value="setOrderSource(record, $event)" />
                        </template></a-table-column>
                    </template>
                </a-table>
            </section>
            <a-space class="workflow-order-footer"><a-button @click="orderVisible = false" :disabled="saving">取消</a-button><a-button type="primary" :loading="saving" :disabled="!!orderError" @click="applyOrder">保存</a-button></a-space>
        </a-modal>
        <a-modal v-model:visible="runsVisible" :title="`${runsTask?.name || ''} · 执行记录`" :width="900" :footer="false" @close="closeRuns">
        <a-table :data="runs" :loading="loadingRuns" :pagination="false" row-key="id">
            <template #columns>
                <a-table-column title="流程 ID" :width="150"><template #cell="{ record }"><a-tooltip :content="record.run_id"><span class="workflow-run-id">{{ record.run_id?.slice(0, 12) }}</span></a-tooltip></template></a-table-column>
                <a-table-column title="版本" data-index="tag" />
                <a-table-column title="任务状态"><template #cell="{ record }"><a-tag :color="statusColors[record.status]">{{ statusLabels[record.status] || record.status }}</a-tag></template></a-table-column>
                <a-table-column title="触发时间"><template #cell="{ record }">{{ new Date(record.created_at).toLocaleString() }}</template></a-table-column>
                <a-table-column title="说明"><template #cell="{ record }">{{ record.error || (record.status === 'failed' ? '任务执行失败，请查看任务日志' : record.status === 'skipped' ? '本任务未执行，请查看任务日志' : record.status === 'unknown' ? '执行结果未能确认，请查看任务日志' : '') }}</template></a-table-column>
                <a-table-column title="操作"><template #cell="{ record }"><a-button type="text" @click="showLogs(record)">任务日志</a-button></template></a-table-column>
            </template>
        </a-table>
        </a-modal>
        <a-modal v-model:visible="visible" :title="editing ? '修改自动任务' : '新增自动任务'" :width="800" :footer="false" :mask-closable="false" :closable="!saving" :esc-to-close="!saving">
            <a-form :model="form" layout="vertical" :disabled="saving">
                <a-form-item label="任务名称"><a-input v-model="form.name" :max-length="120" /></a-form-item>
                <a-form-item label="触发方式"><a-radio-group v-model="form.deploy_type"><a-radio :value="1">更新版本</a-radio><a-radio :value="2">新增版本</a-radio></a-radio-group></a-form-item>
                <a-form-item v-if="form.deploy_type === 2" label="匹配方式"><a-radio-group v-model="form.match_type"><a-radio :value="1">前缀匹配</a-radio><a-radio :value="2">正则匹配</a-radio></a-radio-group></a-form-item>
                <a-form-item label="版本名称 / 匹配条件"><a-input v-model="form.tag_name" :placeholder="form.deploy_type === 1 ? '例如 latest,v1.0，多个版本用英文逗号分隔' : '填写前缀（英文逗号分隔）或一个正则表达式'" /></a-form-item>
                <a-form-item label="执行动作"><a-select v-model="form.action"><a-option value="deploy">部署应用</a-option><a-option value="build">镜像重构</a-option></a-select></a-form-item>
                <template v-if="form.action === 'build'">
                    <a-form-item label="Dockerfile 追加指令">
                        <a-textarea v-model="form.dockerfile" :auto-size="{ minRows: 7, maxRows: 15 }" placeholder="例如 RUN apt-get update && apt-get install -y cloud-init" />
                        <template #extra>系统自动添加 FROM；支持多条指令，不支持 FROM、ONBUILD、ADD、COPY。</template>
                    </a-form-item>
                    <a-form-item label="目标 tag 后缀"><a-input v-model="form.tag_suffix" placeholder="例如 -cloudinit；留空覆盖原 tag" /></a-form-item>
                </template>
                <template v-else>
                    <a-form-item label="目标集群">
                        <a-radio-group v-model="targetCluster" @change="changeCluster">
                            <a-radio v-if="isFounder" value="current">当前集群</a-radio>
                            <a-radio value="remote">第三方集群</a-radio>
                        </a-radio-group>
                    </a-form-item>
                    <a-form-item v-if="targetCluster === 'remote'" label="KUBECONFIG">
                        <div class="workflow-kubeconfig">
                            <a-textarea v-model="form.k8s_config" @input="changeKubeconfig" @blur="loadTargets('namespaces')" :auto-size="{ minRows: 8, maxRows: 8 }" placeholder="请输入 config 或导入配置文件" />
                            <div class="mt-10 df jc-e">
                                <div class="workflow-config-import">
                                    <a-button type="primary" :loading="importingConfig" :disabled="saving">导入</a-button>
                                    <input type="file" aria-label="导入 KUBECONFIG" :disabled="saving || importingConfig" @change="importKubeconfig" />
                                </div>
                            </div>
                        </div>
                    </a-form-item>
                    <a-form-item label="目标命名空间"><a-select v-model="form.k8s_namespace" allow-create :loading="loadingTargets" @popup-visible-change="visible => visible && loadTargets('namespaces')" @change="changeTarget('apps')"><a-option v-for="namespace in namespaces" :key="namespace" :value="namespace" :label="namespace" /></a-select></a-form-item>
                    <a-form-item label="应用类型"><a-select v-model="form.k8s_controller_type" @change="changeTarget('apps')"><a-option value="deployments">无状态应用</a-option><a-option value="statefulsets">有状态应用</a-option><a-option value="daemonsets">守护进程应用</a-option></a-select></a-form-item>
                    <a-form-item label="应用名称"><a-select v-model="form.k8s_app_name" allow-create :loading="loadingTargets" @popup-visible-change="visible => visible && loadTargets('apps')" @change="changeTarget('app-containers')"><a-option v-for="app in apps" :key="app" :value="app" :label="app" /></a-select></a-form-item>
                    <a-form-item label="容器名称"><a-select v-model="containerNames" multiple allow-create :loading="loadingTargets" @popup-visible-change="visible => visible && loadTargets('app-containers')"><a-option v-for="container in containers" :key="container" :value="container" :label="container" /></a-select></a-form-item>
                </template>
            </a-form>
            <div class="dialog-footer"><a-button :disabled="saving" @click="visible = false">取消</a-button><a-button type="primary" :loading="saving" :disabled="importingConfig" @click="applyTask">保存</a-button></div>
        </a-modal>
        <a-modal v-model:visible="logsVisible" :width="1100" :fullscreen="logsFullscreen" :footer="false" :closable="false" class="workflow-log-modal" @open="openLogTerminal" @close="closeLogs">
            <template #title>
                <div class="workflow-log-title">
                    <span>{{ logRun?.name || '任务' }} · 日志</span>
                    <a-space class="workflow-log-actions">
                        <a-button type="text" aria-label="切换日志全屏" @click="logsFullscreen = !logsFullscreen; fitLogTerminal()"><icon-fullscreen-exit v-if="logsFullscreen" /><icon-fullscreen v-else /></a-button>
                        <a-button type="text" aria-label="关闭任务日志" @click="closeLogs"><icon-close /></a-button>
                    </a-space>
                </div>
            </template>
            <div class="workflow-log-content">
                <div class="workflow-log-summary"><span>流程 ID：<span class="workflow-run-id">{{ logRun?.run_id }}</span></span><a-tag :color="statusColors[logStatus]">{{ statusLabels[logStatus] || logStatus }}</a-tag></div>
                <div class="workflow-log-toolbar">
                    <a-space wrap>
                        <span>是否跟踪：</span><a-switch v-model="logFollow" :disabled="logArchived" @change="reloadLogs" />
                        <span class="workflow-run-id">任务 ID：{{ logRun?.task_id }}</span>
                        <span>条数：</span><a-select v-model="logTailLines" style="width:100px" @change="reloadLogs"><a-option v-for="count in [50, 100, 200, 500, 1000, 2000]" :key="count" :value="count">{{ count }} 条</a-option></a-select>
                        <a-button :loading="loadingLogs" @click="reloadLogs">刷新</a-button>
                    </a-space>
                </div>
                <p v-if="logArchived" class="c-99">Pod 已清理，当前显示保存的历史片段（每个容器末尾 100 行、最多 16 KiB），无法恢复更多行。</p>
                <p v-if="logTruncated" class="c-99">部分容器达到 1 MiB 显示上限，请减少日志行数。</p>
                <a-spin :loading="loadingLogs && !logs" class="workflow-log-terminal-wrap"><div ref="workflowTerminal" class="workflow-log-terminal" :style="{ height: logsFullscreen ? 'calc(100vh - 220px)' : '480px' }"></div></a-spin>
            </div>
        </a-modal>
    </div>
</template>

<script>
import myAxios from '@/utils';
import userMixin from '@/utils/user-mixin';
import { markRaw } from 'vue';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { IconFullscreen, IconFullscreenExit, IconClose } from '@arco-design/web-vue/es/icon';
import { messageError, messageSuccess } from '@/utils/ui-feedback';

const taskDefaults = () => ({ id: '', name: '', stage: 1, action: 'build', deploy_type: 1, match_type: 1, tag_name: '', source: 'original', source_task: '', dockerfile: '', tag_suffix: '', k8s_config: '', k8s_namespace: 'default', k8s_controller_type: 'deployments', k8s_app_name: '', k8s_container_name: '' });

export default {
    mixins: [userMixin],
    components: { IconFullscreen, IconFullscreenExit, IconClose },
    props: { repositoryId: { type: Number, required: true } },
    data() {
        return {
            config: { namespace: 'default', tasks: [] }, loading: false, saving: false, loaded: false,
            namespaces: [], apps: [], containers: [], loadingTargets: false, targetSequence: 0,
            targetCluster: 'remote', targetLoaded: {}, targetRequest: '', importingConfig: false,
            visible: false, editing: false, form: taskDefaults(), orderVisible: false, orderTasks: [], runs: [], loadingRuns: false,
            runsVisible: false, runsTask: null, runSequence: 0,
            logsFullscreen: false, logStatus: '', logTerminal: null, logFit: null,
            logsVisible: false, logs: '', loadingLogs: false, disposed: false, logSequence: 0,
            logTailLines: 100, logFollow: true, logArchived: false, logTruncated: false,
            logRun: null, logTimer: null, logController: null,
            actionLabels: { build: '镜像重构', deploy: '部署应用' },
            statusLabels: { pending: '排队中', running: '执行中', succeeded: '成功', failed: '失败', skipped: '已跳过', unknown: '状态未知' },
            statusColors: { pending: 'gray', running: 'blue', succeeded: 'green', failed: 'red', skipped: 'gray', unknown: 'orange' },
        };
    },
    computed: {
        containerNames: {
            get() { return this.form.k8s_container_name.split(',').filter(Boolean); },
            set(value) { this.form.k8s_container_name = value.join(','); },
        },
        orderedTasks() {
            const groups = new Map();
            for (const task of this.config.tasks) {
                const key = this.triggerKey(task);
                if (!groups.has(key)) groups.set(key, []);
                groups.get(key).push(task);
            }
            return [...groups.values()].flatMap(tasks => tasks.sort((a, b) => a.stage - b.stage));
        },
        orderGroups() {
            const groups = new Map();
            for (const task of this.orderTasks) {
                const key = this.triggerKey(task);
                if (!groups.has(key)) groups.set(key, { key, tasks: [] });
                groups.get(key).tasks.push(task);
            }
            return [...groups.values()];
        },
        orderError() { return this.imageDependencyError(this.orderTasks); },
    },
    mounted() { this.load(); },
    beforeUnmount() { this.disposed = true; this.closeLogs(); this.targetSequence++; this.runSequence++; },
    methods: {
        triggerKey(task) {
            const type = Number(task.deploy_type || 1);
            return JSON.stringify([type, type === 2 ? Number(task.match_type || 1) : 0, (task.tag_name || '').trim()]);
        },
        sameTrigger(a, b) { return this.triggerKey(a) === this.triggerKey(b); },
        triggerLabel(task) {
            return `${Number(task.deploy_type || 1) === 1 ? '更新版本' : `新增版本 · ${Number(task.match_type || 1) === 1 ? '前缀匹配' : '正则匹配'}`}：${task.tag_name}`;
        },
        openOrder() {
            if (this.loading || this.saving || !this.loaded) return;
            this.orderTasks = this.orderedTasks.map(task => ({ ...task }));
            this.normalizeOrderSources();
            this.orderVisible = true;
        },
        orderStageOptions(tasks) {
            const stages = [...new Set([...Array.from({ length: tasks.length }, (_, i) => i + 1), ...tasks.map(task => task.stage)])].sort((a, b) => a - b);
            return stages.map(stage => ({ value: stage, label: `第 ${stage} 组` }));
        },
        orderSourceOptions(task) {
            return [{ value: '', label: '原始镜像' }, ...this.orderTasks.filter(source => source.id !== task.id && source.action === 'build' && this.sameTrigger(source, task) && source.stage < task.stage).map(source => ({ value: source.id, label: source.name }))];
        },
        orderSourceValue(task) {
            const value = task.source === 'task' ? task.source_task : '';
            return this.orderSourceOptions(task).some(option => option.value === value) ? value : undefined;
        },
        setOrderSource(task, value) {
            const selected = this.orderSourceOptions(task).some(option => option.value === value) ? value : '';
            task.source = selected ? 'task' : 'original'; task.source_task = selected || '';
        },
        normalizeOrderSources() {
            for (const task of this.orderTasks) {
                if (task.source === 'task' && !this.orderSourceOptions(task).some(option => option.value === task.source_task)) this.setOrderSource(task, '');
            }
        },
        changeOrderStage(task, stage) { task.stage = stage; this.normalizeOrderSources(); },
        async applyOrder() {
            if (this.saving || this.loading) return;
            this.normalizeOrderSources();
            if (this.orderError) { messageError(this.orderError); return; }
            const tasks = this.config.tasks.map(task => ({ ...this.orderTasks.find(item => item.id === task.id) }));
            if (await this.save({ ...this.config, tasks }) && !this.disposed) this.orderVisible = false;
        },
        imageDependencyError(tasks, sources = tasks) {
            for (const task of tasks) {
                if (task.source !== 'task') continue;
                const source = sources.find(item => item.id === task.source_task && item.action === 'build');
                if (source && !this.sameTrigger(source, task)) {
                    return `“${task.name || '当前任务'}”与来源构建任务的触发条件必须相同`;
                }
                if (!source || source.stage >= task.stage) return `“${task.name || '当前任务'}”须在来源构建任务“${source?.name || '未选择'}”完成后执行`;
            }
            return '';
        },
        async request(action, params = {}, options = {}) {
            const response = await myAxios.post(`/v2/api/repository/workflow/${action}`, { repository_id: this.repositoryId, ...params }, options);
            if (!response || !response.data || response.data.error || (response.data.code !== undefined && ![0, 200, '0', '200'].includes(response.data.code))) throw new Error(response?.data?.error || '请求未成功，请重试');
            return response.data.data;
        },
        async load() {
            this.loading = true;
            try { const data = await this.request('get'); if (!this.disposed) { this.config = data.config; this.loaded = true; } }
            catch (error) { if (!this.disposed) messageError(error?.response?.data?.error || error.message || '请求失败'); }
            finally { if (!this.disposed) this.loading = false; }
        },
        showRuns(task) {
            this.runsTask = { id: task.id, name: task.name }; this.runs = []; this.runsVisible = true;
            this.closeLogs(); this.loadRuns();
        },
        closeRuns() { this.runSequence++; this.loadingRuns = false; this.runsVisible = false; },
        stopLogRequest() {
            this.logSequence++; clearTimeout(this.logTimer); this.logTimer = null;
            this.logController?.abort(); this.logController = null;
        },
        closeLogs() { this.stopLogRequest(); this.loadingLogs = false; this.logsVisible = false; this.logs = ''; this.logRun = null; this.logTerminal?.dispose(); this.logTerminal = null; this.logFit = null; },
        async loadRuns() {
            if (!this.runsTask || !this.runsVisible) return;
            const sequence = ++this.runSequence;
            this.loadingRuns = true;
            try { const data = await this.request('runs', { task_id: this.runsTask.id }); if (!this.disposed && sequence === this.runSequence) this.runs = data.list || []; }
            catch (error) { if (!this.disposed && sequence === this.runSequence) messageError(error?.response?.data?.error || error.message || '请求失败'); }
            finally { if (!this.disposed && sequence === this.runSequence) this.loadingRuns = false; }
        },
        sourceLabel(task) { return task.source === 'original' ? '原始镜像' : this.config.tasks.find(item => item.id === task.source_task)?.name || '来源任务已移除'; },
        taskTiming(task) {
            const peers = this.orderedTasks.filter(item => this.sameTrigger(item, task) && item.stage === task.stage);
            if (peers[0]?.id !== task.id) return `with:${peers[0].id}`;
            const previous = this.orderedTasks.filter(item => this.sameTrigger(item, task) && item.stage < task.stage).pop();
            const source = this.config.tasks.find(item => item.id === task.source_task && this.sameTrigger(item, task) && item.stage === previous?.stage);
            if (task.source === 'task' && source) return `after:${source.id}`;
            return previous ? `after:${previous.id}` : 'first';
        },
        timingLabel(task) {
            const timing = this.taskTiming(task);
            if (timing === 'first') return '最先执行';
            const [mode, id] = timing.split(':');
            const name = this.config.tasks.find(item => item.id === id)?.name;
            return mode === 'with' ? `和“${name}”一起执行` : `在“${name}”所在组成功后执行`;
        },
        editTask(task) {
            this.resetTargets('cluster'); this.editing = !!task;
            this.form = task ? { ...task } : { ...taskDefaults(), id: `task-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}` };
            this.targetCluster = this.isFounder && !this.form.k8s_config ? 'current' : 'remote';
            this.visible = true;
        },
        async applyTask() {
            if (this.saving || this.loading || this.importingConfig) return;
            if (this.form.action === 'deploy' && this.targetCluster === 'remote' && !this.form.k8s_config.trim()) { messageError('请填写或导入目标集群 KUBECONFIG'); return; }
            if (!this.form.name.trim() || !this.form.tag_name.trim()) { messageError('请填写任务名称和版本匹配条件'); return; }
            const original = this.config.tasks.find(task => task.id === this.form.id);
            const candidate = { ...this.form, stage: original?.stage || 1, source: original?.source || 'original', source_task: original?.source_task || '' };
            if (candidate.action === 'build') candidate.k8s_config = '';
            const tasks = this.config.tasks.map(task => task.id === candidate.id ? candidate : { ...task });
            if (!original) tasks.push(candidate);
            const error = this.imageDependencyError(tasks);
            if (error) { messageError(`${error}，请先在“调整执行顺序”中修改使用镜像`); return; }
            if (await this.save({ ...this.config, tasks }) && !this.disposed) this.visible = false;
        },
        async removeTask(task) {
            if (this.saving || this.loading) return;
            if (this.config.tasks.some(item => item.source === 'task' && item.source_task === task.id)) { messageError('请先修改引用此任务的镜像来源'); return; }
            await this.save({ ...this.config, tasks: this.config.tasks.filter(item => item.id !== task.id) });
        },
        async save(config) {
            if (this.saving) return false;
            this.saving = true;
            try {
                await this.request('save', { config });
                if (this.disposed) return false;
                this.config = config;
                messageSuccess('任务配置已保存');
                return true;
            } catch (error) {
                if (!this.disposed) messageError(error?.response?.data?.error || error.message || '请求失败');
                return false;
            } finally { if (!this.disposed) this.saving = false; }
        },
        resetTargets(scope) {
            this.targetSequence++; this.loadingTargets = false; this.targetRequest = ''; this.importingConfig = false;
            if (scope === 'cluster') { this.namespaces = []; this.targetLoaded = {}; }
            if (scope !== 'app-containers') { this.apps = []; delete this.targetLoaded.apps; }
            this.containers = []; delete this.targetLoaded['app-containers'];
        },
        changeKubeconfig() {
            this.resetTargets('cluster');
            this.form.k8s_namespace = 'default'; this.form.k8s_app_name = ''; this.form.k8s_container_name = '';
        },
        changeCluster() {
            this.form.k8s_config = '';
            this.resetTargets('cluster');
            this.form.k8s_namespace = 'default'; this.form.k8s_app_name = ''; this.form.k8s_container_name = '';
            if (this.targetCluster === 'current') this.loadTargets('namespaces');
        },
        changeTarget(action) {
            this.resetTargets(action);
            if (action === 'apps') this.form.k8s_app_name = '';
            this.form.k8s_container_name = '';
            this.loadTargets(action);
        },
        async importKubeconfig(event) {
            const input = event.target, file = input.files?.[0];
            if (!file || this.saving) return;
            const sequence = ++this.targetSequence;
            this.loadingTargets = false; this.targetRequest = ''; this.importingConfig = true;
            try {
                const content = await file.text();
                if (this.disposed || !this.visible || sequence !== this.targetSequence || this.targetCluster !== 'remote') return;
                this.form.k8s_config = content;
                this.resetTargets('cluster');
                this.form.k8s_namespace = 'default'; this.form.k8s_app_name = ''; this.form.k8s_container_name = '';
                await this.loadTargets('namespaces');
            } catch (error) { if (!this.disposed && sequence === this.targetSequence) messageError('读取 KUBECONFIG 文件失败，请重试'); }
            finally {
                input.value = '';
                if (!this.disposed && sequence === this.targetSequence) this.importingConfig = false;
            }
        },
        async loadTargets(action) {
            if (this.importingConfig || (this.targetCluster === 'remote' && !this.form.k8s_config.trim())) return;
            if (action !== 'namespaces' && !this.form.k8s_namespace) return;
            if (action === 'app-containers' && !this.form.k8s_app_name) return;
            const params = {
                k8s_config: this.form.k8s_config, k8s_namespace: this.form.k8s_namespace,
                k8s_controller_type: this.form.k8s_controller_type, k8s_app_name: this.form.k8s_app_name,
            };
            const requestKey = JSON.stringify([action, params.k8s_config, ...(action === 'namespaces' ? [] : [params.k8s_namespace, params.k8s_controller_type]), ...(action === 'app-containers' ? [params.k8s_app_name] : [])]);
            if (this.targetLoaded[action] === requestKey || this.targetRequest === requestKey) return;
            const sequence = ++this.targetSequence;
            this.targetRequest = requestKey; this.loadingTargets = true;
            try {
                const response = await myAxios.post(`/v2/api/repository/deploy_rule/k8s/${action}`, params);
                if (!response?.data?.data || response.data.error) throw new Error('读取部署目标失败');
                if (!this.disposed && sequence === this.targetSequence) {
                    const key = { namespaces: 'namespaces', apps: 'apps', 'app-containers': 'containers' }[action];
                    this[key] = response.data.data.list || [];
                    this.targetLoaded[action] = requestKey;
                }
            } catch (error) { if (!this.disposed && sequence === this.targetSequence) messageError(error?.response?.data?.error || error.message || '读取部署目标失败'); }
            finally { if (!this.disposed && sequence === this.targetSequence) { this.loadingTargets = false; this.targetRequest = ''; } }
        },
        showLogs(run) {
            this.stopLogRequest(); this.logRun = { run_id: run.run_id, task_id: run.task_id || this.runsTask?.id || '', name: this.runsTask?.name || '' }; this.logsVisible = true; this.logs = ''; this.logStatus = run.status || '';
            this.logArchived = false; this.logTruncated = false; this.logFollow = true;
            return this.reloadLogs();
        },
        openLogTerminal() {
            if (!this.$refs.workflowTerminal || this.logTerminal) return;
            this.logTerminal = markRaw(new Terminal({ disableStdin: true, convertEol: true, fontSize: 13, lineHeight: 1.4, scrollback: 30000, theme: { background: '#000000', foreground: '#e5e5e5' } }));
            this.logFit = markRaw(new FitAddon());
            this.logTerminal.loadAddon(this.logFit); this.logTerminal.open(this.$refs.workflowTerminal);
            this.renderLogTerminal();
        },
        fitLogTerminal() { this.$nextTick(() => { this.logFit?.fit(); }); },
        renderLogTerminal() {
            this.$nextTick(() => {
                if (!this.logsVisible || !this.logTerminal) return;
                this.logTerminal.reset();
                this.logFit?.fit();
                this.logTerminal.write(this.logs, () => { if (this.logFollow) this.logTerminal?.scrollToBottom(); });
            });
        },
        reloadLogs() {
            this.stopLogRequest();
            return this.loadLogs();
        },
        async loadLogs() {
            if (!this.logsVisible || !this.logRun || this.disposed) return;
            const sequence = ++this.logSequence;
            const controller = new AbortController(); this.logController = controller; this.loadingLogs = true;
            let succeeded = false;
            try {
                const data = await this.request('logs', { run_id: this.logRun.run_id, task_id: this.logRun.task_id, tail_lines: this.logTailLines }, { signal: controller.signal, dontalert: true });
                if (this.disposed || sequence !== this.logSequence) return;
                this.logs = data.log; this.logStatus = data.status || this.logStatus; this.renderLogTerminal(); this.logArchived = !!data.archived; this.logTruncated = !!data.truncated;
                if (this.logArchived) this.logFollow = false;
                succeeded = true;
            } catch (error) {
                if (!this.disposed && sequence === this.logSequence && !controller.signal.aborted) { this.logFollow = false; messageError(error?.response?.data?.error || error.message || '读取日志失败，请刷新重试'); }
            } finally {
                if (!this.disposed && sequence === this.logSequence) {
                    this.loadingLogs = false; this.logController = null;
                    if (succeeded && this.logFollow && this.logsVisible) this.logTimer = setTimeout(() => this.loadLogs(), 3000);
                }
            }
        },
    },
};
</script>

<style scoped>
.workflow-kubeconfig { width: 100%; }
.workflow-config-import { position: relative; }
.workflow-config-import input { position: absolute; inset: 0; width: 100%; height: 100%; opacity: 0; cursor: pointer; }
.workflow-order-group { margin-bottom: 20px; }
.workflow-order-trigger { margin-bottom: 12px; font-weight: 500; }
.workflow-order-footer { display: flex; justify-content: flex-end; margin-top: 20px; }
.workflow-toolbar { margin-bottom: 16px; }
.workflow-run-id { font-family: ui-monospace, monospace; }
.workflow-log-title { position: relative; width: 100%; min-height: 32px; display: flex; align-items: center; justify-content: center; }
.workflow-log-actions { position: absolute; right: 0; }
.workflow-log-summary { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; overflow-wrap: anywhere; }
.workflow-log-toolbar { margin-bottom: 12px; }
.workflow-log-terminal-wrap { display: block; width: 100%; }
.workflow-log-terminal { background: #000; padding: 10px; box-sizing: border-box; width: 100%; overflow: hidden; border: 1px solid var(--color-neutral-3); }
.workflow-log-terminal :deep(.xterm) { height: 100%; }
</style>
