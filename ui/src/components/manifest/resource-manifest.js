export const resourceTypes = [
    { type: 'cpu', title: 'CPU', unit: '核', name: 'W7_RESOURCE_CPU' },
    { type: 'memory', title: '内存', unit: 'MiB', name: 'W7_RESOURCE_MEMORY' },
    { type: 'gpu', title: 'GPU', unit: '个', name: 'W7_RESOURCE_GPU' },
    { type: 'bandwidth', title: '带宽', unit: 'Mbps', name: 'W7_RESOURCE_BANDWIDTH' },
];

export function resourceType(param) {
    return resourceTypes.find(item => param?.type === `resource-${item.type}`)?.type || '';
}

// Convert Kubernetes quantities to the units used by the resource editor.
export function resourceQuantity(value, type) {
    const match = String(value ?? '').match(/^([+]?(?:\d+(?:\.\d*)?|\.\d+))([a-zA-Z]*)$/);
    if (!match) return 0;
    const suffix = match[2];
    const factors = type === 'cpu' ? { '': 1, m: 0.001, u: 0.000001, n: 0.000000001 }
        : { '': 1 / 1048576, Ki: 1 / 1024, Mi: 1, Gi: 1024, Ti: 1048576, K: 1000 / 1048576, M: 1000000 / 1048576, G: 1000000000 / 1048576 };
    const result = Number(match[1]) * (type === 'gpu' && suffix === '' ? 1 : factors[suffix]);
    return Number.isFinite(result) && result > 0 ? result : 0;
}

export function containerResourceDefaults(containers = []) {
    const result = {};
    for (const container of containers) {
        if (container.isInitContainer) continue;
        const limits = container.resources?.limits || {};
        for (const [type, key] of [['cpu', 'cpu'], ['memory', 'memory'], ['gpu', 'nvidia.com/gpu']]) {
            const value = resourceQuantity(limits[key], type);
            if (value) result[type] = Number(((result[type] || 0) + value).toPrecision(12));
        }
    }
    return result;
}

export function resourceManifestState() {
    return { resourceSettings: { show: false, rows: [] } };
}

export const resourceManifestMethods = {
    resourceStartParams() {
        return (this.form.startParams || []).filter(item => resourceType(item));
    },
    resourceParam(definition, value = '') {
        return { name: definition.name, title: definition.title, unit: definition.unit,
            type: `resource-${definition.type}`, options: [], required: false,
            values_text: String(value), module_name: '', description: '', hidden: false, lock: false };
    },
    syncContainerResourceStartParams(previousContainers) {
        if (this.form.type !== 'docker' || this.option?.startParamsOnly) return;
        const defaults = containerResourceDefaults(this.json?.platform?.['container-v2']);
        const previous = containerResourceDefaults(previousContainers);
        for (const definition of resourceTypes) {
            const value = defaults[definition.type];
            if (!value) continue;
            const param = this.form.startParams.find(item => resourceType(item) === definition.type);
            if (!param) {
                if (this.form.startParams.some(item => item.name === definition.name)) continue;
                this.form.startParams.push(this.resourceParam(definition, value));
            } else if (previousContainers && String(param.values_text) === String(previous[definition.type])) {
                param.values_text = String(value);
            }
        }
    },
    openResourceSettings() {
        const defaults = this.form.type === 'docker' ? containerResourceDefaults(this.json?.platform?.['container-v2']) : {};
        this.resourceSettings.rows = resourceTypes.map(definition => {
            const param = this.resourceStartParams().find(item => resourceType(item) === definition.type);
            return { ...definition, enabled: Boolean(param), value: param?.values_text || '', fromContainer: Boolean(defaults[definition.type]) };
        });
        this.resourceSettings.show = true;
    },
    saveResourceSettings() {
        const rows = this.resourceSettings.rows;
        for (const row of rows.filter(item => item.enabled)) {
            if (row.value !== '' && (!Number.isFinite(Number(row.value)) || Number(row.value) <= 0
                || (row.type === 'gpu' && !Number.isInteger(Number(row.value))))) {
                this.$message.error(`${row.title}默认值必须为正${row.type === 'gpu' ? '整数' : '数'}，或留空表示不限制`);
                return;
            }
            if (this.form.startParams.some(item => !resourceType(item) && item.name === row.name)) {
                this.$message.error(`启动参数 ${row.name} 已存在，请先调整其标识`);
                return;
            }
        }
        const existing = this.resourceStartParams();
        this.form.startParams = this.form.startParams.filter(item => !resourceType(item));
        for (const row of rows.filter(item => item.enabled)) {
            const param = existing.find(item => resourceType(item) === row.type);
            this.form.startParams.push({ ...(param || this.resourceParam(row)), unit: row.unit, values_text: String(row.value).trim() });
        }
        this.resourceSettings.show = false;
        this.getStart();
    },
};
