<template>
    <div>
        <a-tabs v-model:active-key="activeTab" type="card-gutter" editable show-add-button
            class="ingress-tabs" @add="addIngressEnd()" @delete="removeIngress">
            <a-tab-pane v-for="(item, index) in modelValue" :key="index"
                :title="item.name || '未命名配置'" closable>
                <div style="padding: 0 16px 16px;">
                    <div class="ingress-name-form-item">
                        <span class="ingress-name-label">名称</span>
                        <a-input v-model="item.name" :spellcheck="false" placeholder="请输入名称"
                            style="width:400px;" @change="emitUpdate"></a-input>
                    </div>
                    <div @click.stop>
                        <manifest-config-table :rows="item.routes" table-class="ingress-table"
                            add-text="添加配置" @add="addRoute(item)">
                            <template #columns>
                                <manifest-config-table-column data-index="match" title="匹配模式" width="16%">
                                    <template #cell="{ record }">
                                        {{ {Prefix: '前缀匹配', Exact: '精准匹配', ImplementationSpecific: '正则匹配'}[record.backend.match] || '-' }}
                                    </template>
                                </manifest-config-table-column>
                                <manifest-config-table-column data-index="path" title="目录" width="22%" />
                                <manifest-config-table-column data-index="backend.name" title="应用" width="22%">
                                    <template #cell="{ record }">
                                        <div v-if="record.backend && mainapp">
                                            {{ appNames.find(v => v.id == record.backend.name)?.title }}
                                        </div>
                                        <div v-else-if="record.backend && !mainapp">
                                            {{ appNamesFilter.find(v => v.id == record.backend.name)?.title }}
                                        </div>
                                    </template>
                                </manifest-config-table-column>
                                <manifest-config-table-column data-index="backend.port" title="端口" width="16%">
                                    <template #cell="{ record }">
                                        {{ record.backend.port }}
                                    </template>
                                </manifest-config-table-column>
                                <manifest-config-table-column title="操作" width="24%">
                                    <template #cell="{ record, index: ridx }">
                                        <span class="c-blue cursor handle" @click.stop="openEdit(record, index, ridx)">修改</span>
                                        <span class="c-blue cursor handle" @click.stop="openStrategy(record, index, ridx)">策略</span>
                                        <span class="c-blue cursor handle" @click.stop="removeRoute(item, ridx)">删除</span>
                                    </template>
                                </manifest-config-table-column>
                            </template>
                        </manifest-config-table>
                    </div>
                </div>
            </a-tab-pane>
        </a-tabs>
        <a-empty v-if="!modelValue.length" description="暂无域名配置，请点击上方 + 添加" />
    </div>
</template>

<script>
import ManifestConfigTable from '@/components/manifest-config-table.vue';
import ManifestConfigTableColumn from '@/components/manifest-config-table-column.vue';
import emitWujieEvent from '@/utils/wujie-event';

export default {
    components: { ManifestConfigTable, ManifestConfigTableColumn },
    props: {
        modelValue: {
            type: Array,
            default: () => [],
        },
        appNames: {
            type: Array,
            default: () => [],
        },
        appPorts: {
            type: Object,
            default: () => ({}),
        },
        mainapp: {
            type: Boolean,
            default: false,
        },
        identifie: {
            type: String,
            default: '',
        },




    },
    emits: ['update:modelValue', 'checkDomainStartParams'],
    data() {
        return {
            activeTab: 0,
            ingressRewrite: {
                show: false,
                ingressIndex: 0,
                routesIndex: 0,
                path: '',
                host: '',
            },
            ingressMoreMatch: {
                show: false,
                ingressIndex: 0,
                routesIndex: 0,
                method: [],
                query: [],
                header: [],
            },
        }
    },
    watch: {
        modelValue: {
            handler(newVal) {
                if (!newVal?.length) {
                    this.activeTab = undefined;
                } else if (this.activeTab === undefined || Number(this.activeTab) >= newVal.length) {
                    this.activeTab = 0;
                }
                this.fillDefaultPorts();
            },
            immediate: true
        },
        appPorts: {
            handler() {
                this.fillDefaultPorts();
            },
            deep: true,
            immediate: true
        }
    },
    computed: {
        appNamesFilter() {
            return this.appNames?.filter?.(v => v.id == this.identifie) || [];
        },
    },
    mounted() {
        if (!this.modelValue.length) {
            this.addIngressEnd('default');
        }
    },
    methods: {
        getDefaultBackendName() {
            if (this.mainapp) {
                return this.identifie || this.appNames?.[0]?.id || '';
            }
            return this.identifie || 'current';
        },
        createRoute() {
            let name = this.getDefaultBackendName();
            return {
                path: '/',
                backend: {
                    port: this.getBackendPorts(name)[0] || '',
                    name,
                    match: 'Prefix',
                },
            };
        },
        addRoute(item) {
            emitWujieEvent("ingressEdit", {
                ingress: this.createRoute(),
                appList: this.appNames,
                appPorts: this.appPorts,
                callback: (data) => {
                    item.routes = item.routes || [];
                    item.routes.push(data);
                    this.emitUpdate();
                },
            })

        },
        changeBackendName(route, name) {
            route.backend.name = name;
            route.backend.port = this.getBackendPorts(name)[0] || '';
            this.emitUpdate();
        },
        getBackendPorts(name) {
            if (!name) { return [] }
            let ports = this.appPorts?.[name] || [];
            return [...new Set((ports.length ? ports : [80]).map(p => String(p)))];
        },
        fillDefaultPorts() {
            let changed = false;
            this.modelValue?.forEach?.(item => {
                item.routes?.forEach?.(route => {
                    let backend = route.backend;
                    if (!backend) { return }
                    if (backend.port !== '' && backend.port !== undefined && backend.port !== null) {
                        let normalizedPort = String(backend.port);
                        if (backend.port !== normalizedPort) {
                            backend.port = normalizedPort;
                            changed = true;
                        }
                        return;
                    }
                    let ports = this.getBackendPorts(backend.name);
                    if (!ports.length) { return }
                    backend.port = ports[0];
                    changed = true;
                });
            });
            if (changed) {
                this.emitUpdate();
            }
        },
        openEdit(row, index, ridx) {
            emitWujieEvent("ingressEdit", {
                ingress: row,
                appList: this.appNames,
                appPorts: this.appPorts,
                callback: (data) => {
                    this.modelValue[index]['routes'][ridx] = data;
                    this.emitUpdate();
                },
            })
        },
        openStrategy(row, index, ridx) {
            emitWujieEvent("ingressStrategy", {
                ingress: row,
                callback: (data) => {
                    this.modelValue[index]['routes'][ridx].backend.strategy = data;
                    this.emitUpdate();
                },
            })
        },
        removeRoute(item, index) {
            item.routes.splice(index, 1);
            this.emitUpdate();
        },
        emitUpdate() {
            this.$emit('update:modelValue', this.modelValue);
        },
        addIngressEnd(name = '') {
            let wasEmpty = this.modelValue.length === 0;
            if (wasEmpty) {
                this.$emit('checkDomainStartParams', true)
            }
            this.modelValue.push({
                name: name || (wasEmpty ? 'default' : 'web_' + Math.random().toString(36).slice(2, 8)),
                routes: [this.createRoute()],
            });
            this.activeTab = this.modelValue.length - 1;
            this.emitUpdate();
        },
        removeIngress(key) {
            let index = Number(key);
            if (!Number.isInteger(index) || index < 0 || index >= this.modelValue.length) { return }
            this.modelValue.splice(index, 1);
            if (!this.modelValue.length) {
                this.activeTab = undefined;
                this.$emit('checkDomainStartParams', false);
            } else {
                this.activeTab = Math.min(index, this.modelValue.length - 1);
            }
            this.emitUpdate();
        },

    },
}
</script>
<style scoped>
.ingress-tabs {
    width: 100%;
}

.ingress-name-form-item {
    display: flex;
    align-items: center;
    margin-bottom: 20px;
}

.ingress-name-label {
    flex: 0 0 80px;
}

.table {
    width: 100%;
}

.ingress-table {
    table-layout: fixed;
}

.ingress-match-col {
    width: 18%;
}

.ingress-path-col {
    width: 28%;
}

.ingress-app-col {
    width: 26%;
}

.ingress-port-col {
    width: 16%;
}

.ingress-action-col {
    width: 12%;
}

.table td {
    box-sizing: border-box;
    padding: 10px;
    line-height: 1.4;
    border: 1px solid #cccccc;
    border-left: 0;
    border-right: 0;
    background: #F0F3FA;
}

.table tr:last-child td {
    background: transparent;
}

.table.nolast tr:last-child td {
    background: #f0f3fa;
}

.table thead tr:first-child td {
    background: #f3f3f3;
    border-top: 0;
}

.ingress-table :deep(.arco-select),
.ingress-table :deep(.arco-input-wrapper) {
    width: 100%;
}

.ingress-table td:last-child {
    white-space: nowrap;
}

.handle+.handle {
    padding-left: 10px;
}
</style>
