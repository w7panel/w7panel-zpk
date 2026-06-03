<template>
    <div style="min-height:100%;">
        <div>
            <div>
                <div class="bg-white bg-padding pb-24" style="border-top:1px solid #EEEEEE;">
                    <a-form ref="form" :model="form" :label-col-style="{ width: '80px' }" label-align="left">
                        <a-form-item label="用户名">
                            {{ userInfo.username }}
                        </a-form-item>
                        <a-form-item label="密码" field="password"
                            :rules="[{ required: true, message: '密码不能为空', trigger: 'manual' }]" v-if="editMode">
                            <a-input-password v-model="form.password" />
                        </a-form-item>
                        <a-form-item label="密码" field="password" v-else>
                            <div class="df" style="width: 200px;gap: 10px">
                                <a-input model-value="已隐藏" disabled />
                                <a-button type="primary" @click="editMode = true">重置密码</a-button>
                            </div>
                        </a-form-item>


                        <a-form-item v-if="editMode">
                            <a-button type="primary" @click="onSubmit">确定</a-button>
                            <a-button @click="editMode = false">取消</a-button>
                        </a-form-item>
                    </a-form>
                </div>
            </div>
        </div>
    </div>
</template>

<script>
import myAxios from "@/utils";
import { messageSuccess } from "@/utils/ui-feedback";

export default {
    name: "zpk_namespace",
    props: {
        userInfo: {
            type: Object,
            default: () => ({})
        }
    },
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
            this.$refs.form.validate(async (errors) => {
                if (errors) {
                    return
                }
                await myAxios.post('/v2/api/user/cur-user/edit', { password: this.form.password })

                this.editMode = false
                this.form.password = ''
                messageSuccess('操作成功');

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
