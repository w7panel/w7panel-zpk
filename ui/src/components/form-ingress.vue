<template>
    <div>
        <div>
            <a-checkbox v-model="checked" :disabled="checked && modelValue.length"
                @change="v => v ? addIngressEnd('default') : $emit('checkDomainStartParams', false)">启用域名</a-checkbox>
        </div>

        <div v-for="(item, index) in modelValue" :key="index" class="mt-10" style="width:100%; margin-bottom:10px;">
            <div class="df ai-c jc-b">
                <div class="df ai-c">
                    <a-input v-if="ingressEditIndex == index" :spellcheck="false" v-model="ingressEdit"
                        @blur="ingressEditIndex = -1; ingressEdit && (item.name = ingressEdit); emitUpdate()"></a-input>
                    <div v-else @click="ingressEditIndex = index; ingressEdit = item.name;" class="df ai-c cursor">
                        <span class="lh-1">{{ item.name }}</span>
                        <icon-edit class="ingress-edit-icon" />
                    </div>
                    <div class="ml-40 c-blue cursor df-s0" @click="ingressEditIndex = -1; removeIngress(index);">删除业务端
                    </div>
                </div>
            </div>

            <manifest-config-table class="mt-10" :rows="item.routes" table-class="ingress-table"
                add-text="添加配置" @add="addRoute(item)">
                <template #columns>
                    <manifest-config-table-column data-index="match" title="匹配模式" width="16%">
                        <template #cell="{ record }">
                            <a-select v-if="record.backend" v-model="record.backend.match" placeholder="请选择">
                                <a-option label="前缀匹配" value="Prefix" />
                                <a-option label="精准匹配" value="Exact" />
                                <a-option label="正则匹配" value="ImplementationSpecific" />
                            </a-select>
                        </template>
                    </manifest-config-table-column>
                    <manifest-config-table-column data-index="path" title="目录" width="22%">
                        <template #cell="{ record }">
                            <a-input v-model="record.path" placeholder="请输入路径"></a-input>
                        </template>
                    </manifest-config-table-column>
                    <manifest-config-table-column data-index="name" title="应用" width="22%">
                        <template #cell="{ record }">
                            <a-select v-if="record.backend && mainapp" v-model="record.backend.name"
                                @change="value => changeBackendName(record, value)" placeholder="请选择应用">
                                <a-option v-for="p in appNames" :key="p.id" :label="p.title" :value="p.id"></a-option>
                            </a-select>
                            <a-select v-if="record.backend && !mainapp" v-model="record.backend.name"
                                @change="value => changeBackendName(record, value)" placeholder="请选择应用">
                                <a-option v-for="p in appNamesFilter" :key="p.id" :label="p.title"
                                    :value="p.id"></a-option>
                            </a-select>
                        </template>
                    </manifest-config-table-column>
                    <manifest-config-table-column data-index="port" title="端口" width="16%">
                        <template #cell="{ record }">
                            <a-select v-if="record.backend" v-model="record.backend.port" placeholder="请选择端口">
                                <a-option v-for="p in getBackendPorts(record.backend.name)" :key="p" :label="p"
                                    :value="p"></a-option>
                            </a-select>
                        </template>
                    </manifest-config-table-column>
                    <manifest-config-table-column title="操作" width="24%">
                        <template #cell="{ record, index: ridx }">
                            <span class="c-blue cursor handle" @click="openEdit(record, index, ridx)">修改</span>
                            <span class="c-blue cursor handle" @click="openStrategy(record, index, ridx)">策略</span>
                            <span class="c-blue cursor handle" @click="item.routes.splice(ridx, 1);">删除</span>
                        </template>
                    </manifest-config-table-column>
                </template>
            </manifest-config-table>


        </div>

        <div v-if="checked" class="mt-10 lh-1 addrole df ai-c jc-c cursor" style="width:100%;"
            @click="addIngressEnd();">
            <span class="addmenu"><icon-plus />添加业务端</span>
        </div>

    </div>
</template>

<script>
import ManifestConfigTable from '@/components/manifest-config-table.vue';
import ManifestConfigTableColumn from '@/components/manifest-config-table-column.vue';
import { IconEdit, IconPlus } from '@arco-design/web-vue/es/icon';

export default {
    components: { ManifestConfigTable, ManifestConfigTableColumn, IconEdit, IconPlus },
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
            checked: false,
            ingressEditIndex: -1,
            ingressEdit: '',
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
                if (newVal && newVal.length > 0 && !this.checked) {
                    this.checked = true;
                    this.$emit('checkDomainStartParams', true);
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
            item.routes = item.routes || [];
            item.routes.push(this.createRoute());
            this.emitUpdate();
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
            window.$wujie?.bus.$emit("ingressEdit", {
                ingress: row,
                appList: this.appNames,
                appPorts: this.appPorts,
                callback: (data) => {
                    this.modelValue[index]['routes'][ridx] = data;
                },
            })
        },
        openStrategy(row, index, ridx) {
            window.$wujie?.bus.$emit("ingressStrategy", {
                ingress: row,
                callback: (data) => {
                    this.modelValue[index]['routes'][ridx].backend.strategy = data;
                },
            })
        },
        emitUpdate() {
            this.$emit('update:modelValue', this.modelValue);
        },
        addIngressEnd(name) {
            if (name == 'default') {
                this.$emit('checkDomainStartParams', true)
            }
            this.modelValue.push({
                name: name || 'web_' + Math.random().toString(36).slice(2, 8),
                routes: [this.createRoute()],
            });
            this.emitUpdate();
        },
        removeIngress(index) {
            this.modelValue.splice(index, 1);
            this.emitUpdate();
        },

    },
}
</script>
<style scoped>
.addrole {
    border: 1px dashed #2d5fff;
    background: rgb(240, 243, 250);
    padding: 10px;
    box-sizing: border-box;
}

.ingress-edit-icon {
    color: #333333;
    font-size: 14px;
    margin-left: 4px;
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
