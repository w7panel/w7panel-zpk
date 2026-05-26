<template>
    <div style="min-height:100%;">
        <div>
            <div>
                <div class="bg-white bg-padding pb-24" style="border-top:1px solid #EEEEEE;">
                    <el-button type="primary" @click="add">新建用户</el-button>
                    <el-table :data="tableData" class="mt-20 table-header">
                        <el-table-column prop="username" label="名称">
                            <template #default="scope">
                                <div>
                                    <div>{{ scope.row.username }}</div>
                                    <div style="font-size: 12px; color: #999;">
                                        {{ scope.row.type === 1 ? '超级管理员' : (scope.row.desc || '暂无描述') }}
                                        <el-button v-if="scope.row.type !== 1" type="text" icon="edit"
                                            style="vertical-align: middle"
                                            @click="editProp = 'desc'; edit(scope.row)"></el-button>
                                    </div>
                                </div>
                            </template>
                        </el-table-column>

                        <el-table-column prop="created_at" label="创建时间">
                            <template #default="scope">{{ new Date(scope.row.created_at).toLocaleString() }}</template>
                        </el-table-column>
                        <el-table-column label="操作">
                            <template #default="scope">
                                <template v-if="scope.row.type !== 1">
                                    <el-button type="text" @click="editProp = ''; edit(scope.row)">修改</el-button>
                                    <el-button type="text" @click="del(scope.row)">删除</el-button>
                                </template>
                            </template>
                        </el-table-column>
                    </el-table>
                    <div class="mt-20 df jc-e">
                        <el-pagination background v-model:page-size="paginate" layout="sizes, prev, pager, next"
                            :current-page="page" :page-count="last_page" :page-sizes="[10, 20, 30, 40]"
                            @size-change="getData(1)" @current-change='getData'></el-pagination>
                    </div>
                </div>
            </div>
        </div>
        <el-dialog v-model="visible" :title="editId ? '编辑用户' : '添加用户'" :width="600">
            <el-form ref="form" :model="form" label-width="80px" label-position="left">
                <el-form-item label="用户名" prop="user_name" v-if="!editProp || editProp === 'user_name'"
                    :rules="[{ required: true, message: '用户名不能为空', trigger: 'manual' }]">
                    <el-input v-model="form.user_name" />
                </el-form-item>
                <template v-if="!editProp || editProp === 'password'">
                    <el-form-item label="密码" prop="password" v-if="editId">
                        <div v-if="lockPassword" class="df" style="width: 100%;gap: 10px">
                            <div class="df ai-c"
                                style="cursor:not-allowed;flex: 1;gap: 5px;box-shadow: rgb(220, 223, 230) 0px 0px 0px 1px inset;border-radius: 4px;padding: 0 5px;background-color: rgb(245, 247, 250);color: #a8a7a7">
                                <el-icon :size="20" color="#999">
                                    <Lock />
                                </el-icon>
                                <span>已隐藏</span>
                            </div>
                            <el-button type="primary" @click="lockPassword = false">修改密码</el-button>
                        </div>
                        <el-input v-model="form.password" type="password" show-password v-else />
                    </el-form-item>
                    <el-form-item label="密码" prop="password"
                        :rules="[{ required: true, message: '密码不能为空', trigger: 'manual' }]" v-else>
                        <el-input v-model="form.password" type="password" show-password />
                    </el-form-item>
                </template>
                <el-form-item label="描述" prop="desc" v-if="!editProp || editProp === 'desc'">
                    <el-input type="textarea" rows="5" v-model="form.desc" />
                </el-form-item>

                <el-form-item>
                    <el-button type="primary" size="large" @click="onSubmit">确定</el-button>
                    <el-button size="large" @click="visible = false">取消</el-button>
                </el-form-item>
            </el-form>
        </el-dialog>
    </div>
</template>

<script>
import myAxios from "@/utils";
import { ElMessageBox, ElMessage } from 'element-plus';

export default {
    name: "zpk_namespace",
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
            this.$refs.form.validate(async (valid) => {
                if (!valid) {
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
                    this.$message.success('操作成功');
                    this.getData(1, false)
                    this.visible = false
                }
            })
        },
        edit(row) {
            this.editProp = ''
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
            ElMessageBox.confirm('请确认删除', "提示", {
                confirmButtonText: "确定",
                cancelButtonText: "取消",
            }).then(() => {
                myAxios.post("/v2/api/user/del", { id: row.id }).then(res => {
                    if (!res) { return }
                    ElMessage({ message: '删除成功', type: 'success', })
                    this.getData(1);
                });
            }).catch(() => {
            });
        },
    }
}
</script>

<style scoped>
.ml-20 {
    margin-left: 20px;
}
</style>
