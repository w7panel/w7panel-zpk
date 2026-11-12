<template>
    <div>
        <a-space class="workflow-toolbar">
            <a-button type="primary" @click="editTask()" :disabled="loading || saving || !loaded">新增自动任务</a-button>
        </a-space>
        <a-spin :loading="loading" style="width:100%">
            <a-table :data="orderedTasks" :pagination="false" row-key="id">
                <template #columns>
                    <a-table-column title="执行时机" :width="240">
                        <template #cell="{ record }">{{ timingLabel(record) }}</template>
                    </a-table-column>
                    <a-table-column title="任务" data-index="name" />
                    <a-table-column title="执行动作"><template #cell="{ record }">{{ actionLabels[record.action] }}</template></a-table-column>
                    <a-table-column title="触发条件"><template #cell="{ record }">{{ record.deploy_type === 1 ? '更新版本' : '新增版本' }}：{{ record.tag_name }}</template></a-table-column>
                    <a-table-column title="镜像来源"><template #cell="{ record }">{{ sourceLabel(record) }}</template></a-table-column>
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
        <a-modal v-model:visible="runsVisible" :title="`${runsTask?.name || ''} · 执行记录`" :width="900" :footer="false" @close="closeRuns">
        <a-button :loading="loadingRuns" @click="loadRuns">刷新执行记录</a-button>
        <p class="c-99">显示此任务最近 50 次执行；“流程日志”可查看同一轮流程中所有串联、并行任务的日志。</p>
        <a-table :data="runs" :loading="loadingRuns" :pagination="false" row-key="id">
            <template #columns>
                <a-table-column title="版本" data-index="tag" />
                <a-table-column title="任务状态"><template #cell="{ record }"><a-tag :color="statusColors[record.status]">{{ statusLabels[record.status] || record.status }}</a-tag></template></a-table-column>
                <a-table-column title="触发时间"><template #cell="{ record }">{{ new Date(record.created_at).toLocaleString() }}</template></a-table-column>
                <a-table-column title="说明"><template #cell="{ record }">{{ record.error || (record.status === 'failed' ? '任务执行失败，请查看流程日志' : record.status === 'skipped' ? '前序任务失败或本任务未启动' : '') }}</template></a-table-column>
                <a-table-column title="操作"><template #cell="{ record }"><a-button type="text" @click="showLogs(record)">流程日志</a-button></template></a-table-column>
            </template>
        </a-table>
        </a-modal>
        <a-modal v-model:visible="visible" :title="editing ? '修改自动任务' : '新增自动任务'" :width="800" :footer="false" :mask-closable="false" :closable="!saving" :esc-to-close="!saving">
            <a-form :model="form" layout="vertical" :disabled="saving">
                <a-form-item label="任务名称"><a-input v-model="form.name" :max-length="120" /></a-form-item>
                <a-form-item label="触发方式"><a-radio-group v-model="form.deploy_type"><a-radio :value="1">更新版本</a-radio><a-radio :value="2">新增版本</a-radio></a-radio-group></a-form-item>
                <a-form-item v-if="form.deploy_type === 2" label="匹配方式"><a-radio-group v-model="form.match_type"><a-radio :value="1">前缀匹配</a-radio><a-radio :value="2">正则匹配</a-radio></a-radio-group></a-form-item>
                <a-form-item label="版本名称 / 匹配条件"><a-input v-model="form.tag_name" :placeholder="form.deploy_type === 1 ? '例如 latest,v1.0，多个版本用英文逗号分隔' : '填写前缀（英文逗号分隔）或一个正则表达式'" /></a-form-item>
                <a-form-item label="执行时机">
                    <a-select v-model="timing" :options="timingOptions" />
                    <template #extra>仅可选择触发方式、匹配方式和版本条件相同的任务；同组并行，“之后执行”等待整组全部成功。灰色选项会破坏镜像依赖。</template>
                </a-form-item>
                <p v-if="timingConflict" style="color:var(--color-danger-6)">{{ timingConflict }}</p>
                <a-form-item label="执行动作"><a-select v-model="form.action"><a-option value="deploy">部署应用</a-option><a-option value="build">镜像重构</a-option></a-select></a-form-item>
                <a-form-item label="镜像来源"><a-radio-group v-model="form.source"><a-radio value="original">原始镜像</a-radio><a-radio v-if="timingSourceTask || form.source === 'task'" value="task" :disabled="!timingSourceTask">指定构建任务的结果</a-radio></a-radio-group></a-form-item>
                <template v-if="form.action === 'build'">
                    <a-form-item label="Dockerfile 追加指令"><a-textarea v-model="form.dockerfile" :auto-size="{ minRows: 7, maxRows: 15 }" placeholder="例如 RUN apt-get update && apt-get install -y cloud-init" /></a-form-item>
                    <p class="c-99">系统自动添加 FROM；支持多条指令，不支持 FROM、ONBUILD、ADD、COPY。</p>
                    <a-form-item label="目标 tag 后缀"><a-input v-model="form.tag_suffix" placeholder="例如 -cloudinit；留空覆盖原 tag" /></a-form-item>
                </template>
                <template v-else>
                    <a-form-item label="目标集群 KUBECONFIG"><a-textarea v-model="form.k8s_config" @blur="loadTargets('namespaces')" :auto-size="{ minRows: 4, maxRows: 8 }" placeholder="管理员可留空使用当前集群；其他用户请填写目标集群配置" /></a-form-item>
                    <a-form-item label="目标命名空间"><a-select v-model="form.k8s_namespace" allow-create :loading="loadingTargets" @popup-visible-change="visible => visible && loadTargets('namespaces')" @change="loadTargets('apps')"><a-option v-for="namespace in namespaces" :key="namespace" :value="namespace" /></a-select></a-form-item>
                    <a-form-item label="应用类型"><a-select v-model="form.k8s_controller_type" @change="loadTargets('apps')"><a-option value="deployments">无状态应用</a-option><a-option value="statefulsets">有状态应用</a-option><a-option value="daemonsets">守护进程应用</a-option></a-select></a-form-item>
                    <a-form-item label="应用名称"><a-select v-model="form.k8s_app_name" allow-create :loading="loadingTargets" @popup-visible-change="visible => visible && loadTargets('apps')" @change="loadTargets('app-containers')"><a-option v-for="app in apps" :key="app" :value="app" /></a-select></a-form-item>
                    <a-form-item label="容器名称"><a-select v-model="containerNames" multiple allow-create :loading="loadingTargets" @popup-visible-change="visible => visible && loadTargets('app-containers')"><a-option v-for="container in containers" :key="container" :value="container" /></a-select></a-form-item>
                </template>
            </a-form>
            <div class="dialog-footer"><a-button :disabled="saving" @click="visible = false">取消</a-button><a-button type="primary" :loading="saving" @click="applyTask">保存</a-button></div>
        </a-modal>
        <a-modal v-model:visible="logsVisible" title="本轮流程日志" :width="900" :footer="false" @close="closeLogs">
            <a-space wrap>
                <span>日志行数：</span><a-select v-model="logTailLines" style="width:110px" @change="reloadLogs"><a-option v-for="count in [50, 100, 200, 500, 1000, 2000]" :key="count" :value="count">{{ count }} 行</a-option></a-select>
                <span>跟踪：</span><a-switch v-model="logFollow" :disabled="logArchived" @change="reloadLogs" />
                <a-button :loading="loadingLogs" @click="reloadLogs">刷新</a-button>
            </a-space>
            <p class="c-99">{{ logArchived ? 'Pod 已清理，当前显示保存的历史片段（末尾 100 行、最多 16 KiB），无法继续跟踪或恢复更多行。' : logFollow ? '每 3 秒更新本轮所有任务日志，自动滚动到底部。' : '当前为所选行数的日志快照。' }}</p>
            <p v-if="logTruncated" class="c-99">部分容器达到 1 MiB 显示上限，请减少日志行数。</p>
            <a-spin :loading="loadingLogs && !logs"><pre ref="workflowLogs" class="workflow-logs">{{ logs }}</pre></a-spin>
        </a-modal>
    </div>
</template>

<script>
import myAxios from '@/utils';
import { messageError, messageSuccess } from '@/utils/ui-feedback';

const taskDefaults = () => ({ id: '', name: '', stage: 1, action: 'build', deploy_type: 1, match_type: 1, tag_name: '', source: 'original', source_task: '', dockerfile: '', tag_suffix: '', k8s_config: '', k8s_namespace: 'default', k8s_controller_type: 'deployments', k8s_app_name: '', k8s_container_name: '' });

export default {
    props: { repositoryId: { type: Number, required: true } },
    data() {
        return {
            config: { namespace: 'default', tasks: [] }, loading: false, saving: false, loaded: false,
            namespaces: [], apps: [], containers: [], loadingTargets: false, targetSequence: 0,
            visible: false, editing: false, form: taskDefaults(), timing: 'first', runs: [], loadingRuns: false,
            runsVisible: false, runsTask: null, runSequence: 0,
            logsVisible: false, logs: '', loadingLogs: false, disposed: false, logSequence: 0,
            logTailLines: 100, logFollow: true, logArchived: false, logTruncated: false,
            logRun: null, logTimer: null, logController: null,
            actionLabels: { build: '镜像重构', deploy: '部署应用' },
            statusLabels: { pending: '排队中', running: '执行中', succeeded: '成功', failed: '失败', skipped: '已跳过' },
            statusColors: { pending: 'gray', running: 'blue', succeeded: 'green', failed: 'red', skipped: 'gray' },
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
        timingOptions() {
            const options = [{ value: 'first', label: '最先执行' }, ...this.orderedTasks.filter(task => task.id !== this.form.id && this.sameTrigger(task, this.form)).flatMap(task => [
                { value: `after:${task.id}`, label: `在“${task.name}”之后执行` },
                { value: `with:${task.id}`, label: `和“${task.name}”一起执行` },
            ])];
            return options.map(option => {
                const reason = this.imageDependencyError(this.tasksForTiming(option.value));
                return { ...option, disabled: !!reason, label: reason ? `${option.label}（${reason}）` : option.label };
            });
        },
        previewTasks() { return this.tasksForTiming(this.timing); },
        timingConflict() { return this.imageDependencyError(this.previewTasks); },
        executionStage() {
            return this.previewTasks.find(task => task.id === this.form.id)?.stage;
        },
        timingDescription() {
            if (this.timing === 'first') return '';
            const [mode, id] = this.timing.split(':');
            const task = this.config.tasks.find(task => task.id === id && this.sameTrigger(task, this.form));
            if (!task) return '请选择执行时机。';
            const names = this.config.tasks.filter(item => item.id !== this.form.id && this.sameTrigger(item, this.form) && item.stage === task.stage).map(item => `“${item.name}”`).join('、');
            return mode === 'after' ? `等待${names}全部成功后执行。` : `与${names}同组并行；后续任务等待整组全部成功。`;
        },
        timingSourceTask() { return this.sourceForTiming(this.timing); },
    },
    watch: {
        'form.deploy_type'() { this.resetTiming(); },
        'form.match_type'() { this.resetTiming(); },
        'form.tag_name'() { this.resetTiming(); },
    },
    mounted() { this.load(); },
    beforeUnmount() { this.disposed = true; this.closeLogs(); this.targetSequence++; this.runSequence++; },
    methods: {
        triggerKey(task) {
            const type = Number(task.deploy_type || 1);
            return JSON.stringify([type, type === 2 ? Number(task.match_type || 1) : 0, (task.tag_name || '').trim()]);
        },
        sameTrigger(a, b) { return this.triggerKey(a) === this.triggerKey(b); },
        resetTiming() {
            if (this.timing === 'first') return;
            const id = this.timing.split(':')[1];
            if (!this.config.tasks.some(task => task.id === id && this.sameTrigger(task, this.form))) this.timing = 'first';
        },
        sourceForTiming(timing) {
            const original = this.config.tasks.find(task => task.id === this.form.id);
            if (original?.source === 'task' && this.sameTrigger(original, this.form) && timing === this.taskTiming(original)) return original.source_task;
            if (timing === 'first') return '';
            const [mode, id] = timing.split(':');
            const target = this.config.tasks.find(task => task.id === id && task.id !== this.form.id && this.sameTrigger(task, this.form));
            if (!target) return '';
            return mode === 'after' && target.action === 'build' ? target.id : target.source === 'task' ? target.source_task : '';
        },
        tasksForTiming(timing) {
            const original = this.config.tasks.find(task => task.id === this.form.id);
            const unchanged = original && this.sameTrigger(original, this.form) && timing === this.taskTiming(original);
            const tasks = this.config.tasks.filter(task => task.id !== this.form.id).map(task => ({ ...task }));
            let stage;
            if (unchanged) stage = original.stage;
            else if (timing === 'first') {
                stage = 1;
                tasks.forEach(task => { if (this.sameTrigger(task, this.form)) task.stage++; });
            } else {
                const [mode, id] = timing.split(':');
                const target = tasks.find(task => task.id === id && this.sameTrigger(task, this.form));
                stage = target ? target.stage + (mode === 'after' ? 1 : 0) : null;
            }
            const task = { ...this.form, stage, source_task: this.form.source === 'task' ? this.sourceForTiming(timing) : '' };
            const index = this.config.tasks.findIndex(item => item.id === task.id);
            tasks.splice(index >= 0 ? index : tasks.length, 0, task);
            return tasks;
        },
        imageDependencyError(tasks) {
            for (const task of tasks) {
                if (task.source !== 'task') continue;
                const source = tasks.find(item => item.id === task.source_task && item.action === 'build');
                if (source && !this.sameTrigger(source, task)) return `“${task.name || '当前任务'}”与来源构建任务的触发条件必须相同`;
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
        closeLogs() { this.stopLogRequest(); this.loadingLogs = false; this.logsVisible = false; this.logs = ''; this.logRun = null; },
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
            this.targetSequence++; this.namespaces = []; this.apps = []; this.containers = []; this.loadingTargets = false; this.editing = !!task;
            this.form = task ? { ...task } : { ...taskDefaults(), id: `task-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}` };
            this.timing = task ? this.taskTiming(task) : 'first'; this.visible = true;
        },
        async applyTask() {
            if (this.saving || this.loading) return;
            if (!this.form.name.trim() || !this.form.tag_name.trim()) { messageError('请填写任务名称和版本匹配条件'); return; }
            if (!this.executionStage || this.executionStage > 1000) { messageError('请选择有效的执行时机'); return; }
            if (this.form.source === 'task' && !this.timingSourceTask) { messageError('当前执行时机没有可用的构建镜像，请选择原始镜像'); return; }
            if (this.timingConflict) { messageError(this.timingConflict); return; }
            const tasks = this.previewTasks.map(task => task.id === this.form.id && task.action === 'build' ? { ...task, k8s_config: '' } : task);
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
        async loadTargets(action) {
            if (action !== 'namespaces' && !this.form.k8s_namespace) return;
            if (action === 'app-containers' && !this.form.k8s_app_name) return;
            const sequence = ++this.targetSequence;
            this.loadingTargets = true;
            try {
                const response = await myAxios.post(`/v2/api/repository/deploy_rule/k8s/${action}`, {
                    k8s_config: this.form.k8s_config, k8s_namespace: this.form.k8s_namespace,
                    k8s_controller_type: this.form.k8s_controller_type, k8s_app_name: this.form.k8s_app_name,
                });
                if (!response?.data?.data || response.data.error) throw new Error('读取部署目标失败');
                if (!this.disposed && sequence === this.targetSequence) {
                    const key = { namespaces: 'namespaces', apps: 'apps', 'app-containers': 'containers' }[action];
                    this[key] = response.data.data.list || [];
                }
            } catch (error) { if (!this.disposed && sequence === this.targetSequence) messageError(error?.response?.data?.error || error.message || '读取部署目标失败'); }
            finally { if (!this.disposed && sequence === this.targetSequence) this.loadingTargets = false; }
        },
        showLogs(run) {
            this.stopLogRequest(); this.logRun = { run_id: run.run_id }; this.logsVisible = true; this.logs = '';
            this.logArchived = false; this.logTruncated = false; this.logFollow = true;
            return this.reloadLogs();
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
                const data = await this.request('logs', { run_id: this.logRun.run_id, tail_lines: this.logTailLines }, { signal: controller.signal, dontalert: true });
                if (this.disposed || sequence !== this.logSequence) return;
                this.logs = data.log; this.logArchived = !!data.archived; this.logTruncated = !!data.truncated;
                if (this.logArchived) this.logFollow = false;
                if (this.logFollow) this.$nextTick(() => { if (sequence === this.logSequence && this.$refs.workflowLogs) this.$refs.workflowLogs.scrollTop = this.$refs.workflowLogs.scrollHeight; });
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
.workflow-toolbar { margin-bottom: 16px; }
.workflow-logs { white-space: pre-wrap; overflow-wrap: anywhere; max-height: 60vh; overflow: auto; }
</style>
