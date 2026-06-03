<template>
    <div style="min-height:100%;">
        <div>
            <div>
                <div class="bg-white bg-padding pb-24" style="border-top:1px solid #EEEEEE;">
                    <a-button type="primary" @click="add">新建用户</a-button>
                    <a-table :data="tableData" :pagination="false" class="mt-20 table-header" row-key="id">
                        <template #columns>
                            <a-table-column data-index="username" title="名称">
                                <template #cell="{ record }">
                                    <div>
                                        <div>{{ record.username }}</div>
                                        <div style="font-size: 12px; color: #999;">
                                            {{ record.type === 1 ? '超级管理员' : (record.desc || '暂无描述') }}
                                            <a-tooltip v-if="record.type !== 1" content="编辑">
                                                <a-button class="user-icon-action" type="text" shape="circle" size="mini"
                                                    @click="editProp = 'desc'; edit(record)">
                                                    <template #icon><icon-edit /></template>
                                                </a-button>
                                            </a-tooltip>
                                        </div>
                                    </div>
                                </template>
                            </a-table-column>

                            <a-table-column data-index="created_at" title="创建时间">
                                <template #cell="{ record }">{{ new Date(record.created_at).toLocaleString() }}</template>
                            </a-table-column>
                            <a-table-column title="操作">
                                <template #cell="{ record }">
                                    <template v-if="record.type !== 1">
                                        <a-button type="text" @click="editProp = ''; edit(record)">修改</a-button>
                                        <a-button type="text" status="danger" @click="del(record)">删除</a-button>
                                    </template>
                                </template>
                            </a-table-column>
                        </template>
                    </a-table>
                    <div class="mt-20 df jc-e">
                        <a-pagination v-model:current="page" v-model:page-size="paginate" :total="list.length"
                            :page-size-options="[10, 20, 30, 40]" show-page-size
                            @page-size-change="handlePageSizeChange" @change="getData" />
                    </div>
                </div>
            </div>
        </div>
        <a-modal v-model:visible="visible" :title="editId ? '编辑用户' : '添加用户'" :width="600" :footer="false">
            <a-form ref="form" :model="form" :label-col-props="{ span: 4, flex: '0 0 80px' }"
                :wrapper-col-props="{ span: 20, flex: '1' }" label-align="left" class="user-form">
                <a-form-item label="用户名" field="user_name" v-if="!editProp || editProp === 'user_name'"
                    :rules="[{ required: true, message: '用户名不能为空', trigger: 'manual' }]">
                    <a-input v-model="form.user_name" />
                </a-form-item>
                <template v-if="!editProp || editProp === 'password'">
                    <a-form-item label="密码" field="password" v-if="editId">
                        <div v-if="lockPassword" class="df" style="width: 100%;gap: 10px">
                            <a-input model-value="已隐藏" disabled />
                            <a-button type="primary" @click="lockPassword = false">修改密码</a-button>
                        </div>
                        <a-input-password v-model="form.password" v-else />
                    </a-form-item>
                    <a-form-item label="密码" field="password"
                        :rules="[{ required: true, message: '密码不能为空', trigger: 'manual' }]" v-else>
                        <a-input-password v-model="form.password" />
                    </a-form-item>
                </template>
                <a-form-item label="描述" field="desc" v-if="!editProp || editProp === 'desc'">
                    <a-textarea v-model="form.desc" :auto-size="{ minRows: 5, maxRows: 5 }" />
                </a-form-item>

                <a-form-item>
                    <a-button type="primary" size="large" @click="onSubmit">确定</a-button>
                    <a-button size="large" @click="visible = false">取消</a-button>
                </a-form-item>
            </a-form>
        </a-modal>
    </div>
</template>

<script>
import myAxios from "@/utils";
import { confirm, messageSuccess } from "@/utils/ui-feedback";
import { IconEdit } from '@arco-design/web-vue/es/icon';

export default {
    name: "zpk_namespace",
    components: {
        IconEdit,
    },
    data() {
        return {
            editProp: '',
            lockPassword: true,
            form: {
                user_name: '',
                password: '',
                desc: '',
                expire_type: 1,
                expire_days: -1,
            },
            editId: '',
            visible: false,
            page: 1,
            paginate: 10,
            last_page: 1,
            list: [],
        }
    },
    computed: {
        tableData() {
            return this.list.slice((this.page - 1) * this.paginate, this.page * this.paginate)
        }
    },
    created() {
        this.getData(1);
    },
    methods: {
        getData(p, notChangePage) {
            if (!notChangePage) {
                this.page = p;
            }
            if (p === 1) {
                myAxios.post("/v2/api/user/list").then(res => {
                    let data = res.data?.data?.list ?? [];
                    this.list = data;
                    this.last_page = Math.ceil(data.length / this.paginate);
                });
            }

        },
        handlePageSizeChange() {
            this.getData(1);
        },
        add() {
            this.editProp = ''
            this.editId = ''
            this.form = {
                user_name: '',
                password: '',
                desc: '',
                expire_type: 1,
                expire_days: -1,
            }
            this.visible = true
        },
        onSubmit() {
            this.$refs.form.validate(async (errors) => {
                if (errors) {
                    return
                }
                let postId = this.editId || ''
                let postData = {}
                for (let key in this.form) {
                    if (['expire_type', 'permission_detail'].includes(key)) {
                        continue
                    } else if (key === 'expire_days') {
                        if (this.form.expire_type === 1) {
                            postData[key] = -1
                        } else {
                            postData[key] = this.form.expire_days
                        }
                    } else if (key === 'permission_type') {
                        continue
                    } else {
                        postData[key] = this.form[key]
                    }
                }

                if (this.editId) {
                    await myAxios.post('/v2/api/user/edit', { id: this.editId, ...postData })
                } else {
                    const res = await myAxios.post('/v2/api/user/add', postData)
                    postId = res.data?.data?.user?.id
                }

                if (postId) {
                    messageSuccess('操作成功');
                    this.getData(1, false)
                    this.visible = false
                }
            })
        },
        edit(row) {
            this.editId = row.id
            this.lockPassword = true
            this.form = {
                user_name: row.username,
                password: '',
                desc: row.desc || '',
                expire_type: 1,
                expire_days: -1,
            }
            this.visible = true
        },
        del(row) {
            confirm({
                title: "提示",
                content: '请确认删除',
                confirmButtonText: "确定",
                cancelButtonText: "取消",
                onOk: () => {
                    return myAxios.post("/v2/api/user/del", { id: row.id }).then(res => {
                        if (!res) { return }
                        messageSuccess('删除成功')
                        this.getData(1);
                    });
                }
            });
        },
    }
}
</script>

<style scoped>
.ml-20 {
    margin-left: 20px;
}

.table-header :deep(.arco-table-th) {
    background: #F3F3F3;
    color: #666666;
    font-weight: 500;
}

.user-form :deep(.arco-form-item-label-col) {
    width: 80px;
}

.user-form :deep(.arco-form-item-label) {
    white-space: nowrap;
}

.user-form :deep(.arco-form-item-wrapper-col) {
    min-width: 0;
}

.user-icon-action {
    color: #3370ff;
    vertical-align: middle;
}

</style>
