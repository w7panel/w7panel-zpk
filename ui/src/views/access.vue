<template>
    <div style="min-height:100%;">
        <div>
            <div>
                <div class="bg-white bg-padding pb-24" style="border-top:1px solid #EEEEEE;">
                    <el-form ref="form" :model="form" label-width="80px" label-position="left">
                        <el-form-item label="用户名">
                            {{ userInfo.username }}
                        </el-form-item>
                        <el-form-item label="密码" prop="password"
                            :rules="[{ required: true, message: '密码不能为空', trigger: 'manual' }]" v-if="editMode">
                            <el-input v-model="form.password" type="password" show-password />
                        </el-form-item>
                        <el-form-item label="密码" prop="password" v-else>
                            <div class="df" style="width: 200px;gap: 10px">
                                <div class="df ai-c"
                                    style="cursor:not-allowed;flex: 1;gap: 5px;box-shadow: rgb(220, 223, 230) 0px 0px 0px 1px inset;border-radius: 4px;padding: 0 5px;background-color: rgb(245, 247, 250);color: #a8a7a7">
                                    <el-icon :size="20" color="#999">
                                        <Lock />
                                    </el-icon>
                                    <span>已隐藏</span>
                                </div>
                                <el-button type="primary" @click="editMode = true">重置密码</el-button>
                            </div>
                        </el-form-item>


                        <el-form-item v-if="editMode">
                            <el-button type="primary" @click="onSubmit">确定</el-button>
                            <el-button @click="editMode = false">取消</el-button>
                        </el-form-item>
                    </el-form>
                </div>
            </div>
        </div>
    </div>
</template>

<script>
import myAxios from "@/utils";

export default {
    name: "zpk_namespace",
    props: ['userInfo'],
    data() {
        return {
            form: {
                user_name: '',
                password: '',
            },
            editMode: false,
        }
    },
    methods: {
        onSubmit() {
            this.$refs.form.validate(async (valid) => {
                if (!valid) {
                    return
                }
                await myAxios.post('/v2/api/user/cur-user/edit', { password: this.form.password })

                this.editMode = false
                this.form.password = ''
                this.$message.success('操作成功');

            })
        },
    }
}
</script>

<style scoped>
.ml-20 {
    margin-left: 20px;
}
</style>