<template>
    <div>
        <div>
            <el-checkbox v-model="checked" :disabled="checked && modelValue.length"
                @change="v => v ? addIngressEnd('default') : $emit('checkDomainStartParams', false)" label="启用域名" />
        </div>

        <div v-for="(item, index) in modelValue" :key="index" class="mt-10" style="width:100%; margin-bottom:10px;">
            <div class="df ai-c jc-b">
                <div class="df ai-c">
                    <el-input v-if="ingressEditIndex == index" :spellcheck="false" v-model="ingressEdit"
                        @blur="ingressEditIndex = -1; ingressEdit && (item.name = ingressEdit); emitUpdate()"></el-input>
                    <div v-else @click="ingressEditIndex = index; ingressEdit = item.name;" class="df ai-c cursor">
                        <span class="lh-1">{{ item.name }}</span>
                        <el-icon color="#333333" :size="14" style="margin-left:4px;">
                            <Edit />
                        </el-icon>
                    </div>
                    <div class="ml-40 c-blue cursor df-s0" @click="ingressEditIndex = -1; removeIngress(index);">删除业务端
                    </div>
                </div>
            </div>

            <table class="table mt-10">
                <thead>
                    <tr>
                        <td>匹配模式</td>
                        <td>目录</td>
                        <td>应用</td>
                        <td>端口</td>
                        <td>操作</td>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="(route, ridx) in item.routes" :key="ridx">
                        <td>
                            <el-select v-if="route.backend" v-model="route.backend.match" placeholder="请选择">
                                <el-option label="前缀匹配" value="Prefix" />
                                <el-option label="精准匹配" value="Exact" />
                                <el-option label="正则匹配" value="ImplementationSpecific" />
                            </el-select>
                        </td>
                        <td>
                            <el-input v-model="route.path" placeholder="请输入路径"></el-input>
                        </td>
                        <td>
                            <el-select v-if="route.backend && mainapp" v-model="route.backend.name"
                                @change="route.backend.port = '';" placeholder="请选择应用">
                                <el-option v-for="p in appNames" :key="p.id" :label="p.title" :value="p.id"></el-option>
                            </el-select>
                            <el-select v-if="route.backend && !mainapp" v-model="route.backend.name"
                                @change="route.backend.port = '';" placeholder="请选择应用">
                                <el-option v-for="p in appNamesFilter" :key="p.id" :label="p.title"
                                    :value="p.id"></el-option>
                            </el-select>
                        </td>
                        <td>
                            <el-select v-if="route.backend" v-model="route.backend.port" placeholder="请选择端口">
                                <el-option v-for="p in appPorts[route.backend.name] || []" :key="p" :label="p"
                                    :value="p"></el-option>
                            </el-select>
                        </td>
                        <td>
                            <span class="c-blue cursor handle" @click="openEdit(route, index, ridx)">修改</span>
                            <span class="c-blue cursor handle" @click="openStrategy(route, index, ridx)">策略</span>
                            <span class="c-blue cursor handle" @click="item.routes.splice(ridx, 1);">删除</span>
                        </td>
                    </tr>
                    <tr>
                        <td colspan="8" class="cursor txt-c"
                            @click="item.routes.push({ path: '/', backend: { port: '', name: '', match: 'Prefix' } })">
                            <span class="addmenu">添加配置</span>
                        </td>
                    </tr>
                </tbody>

            </table>


        </div>

        <div v-if="checked" class="mt-10 lh-1 addrole df ai-c jc-c cursor" style="width:100%;"
            @click="addIngressEnd();">添加业务端</div>

    </div>
</template>

<script>

export default {
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
            },
            immediate: true
        }
    },
    computed: {
        appNamesFilter() {
            return this.appNames?.filter?.(v => v.id == this.identifie) || [];
        },
    },
    methods: {
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
                routes: [
                    {
                        path: '/',
                        backend: { port: '', name: this.mainapp ? '' : 'current' },
                    }
                ],
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

.table {
    width: 100%;
}

.table td {
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

.handle+.handle {
    padding-left: 10px;
}
</style>