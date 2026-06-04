<template>
    <div class="access-page" :class="{ 'is-embedded': embedded }">
        <div class="access-panel">
            <a-form ref="form" :model="form" :label-col-style="{ width: '80px' }" label-align="left"
                class="access-form">
                <a-form-item label="用户名">
                    {{ userInfo.username }}
                </a-form-item>
                <a-form-item label="密码" field="password"
                    :rules="[{ required: true, message: '密码不能为空', trigger: 'manual' }]" v-if="editMode">
                    <a-input-password v-model="form.password" />
                </a-form-item>
                <a-form-item label="密码" field="password" v-else>
                    <div class="access-password-row">
                        <a-input class="access-hidden-password" model-value="已隐藏" disabled>
                            <template #prefix>
                                <icon-lock />
                            </template>
                        </a-input>
                        <a-button type="primary" @click="editMode = true">重置密码</a-button>
                    </div>
                </a-form-item>


                <a-form-item v-if="editMode">
                    <div class="access-action-row">
                        <a-button @click="editMode = false">取消</a-button>
                        <a-button type="primary" @click="onSubmit">确定</a-button>
                    </div>
                </a-form-item>
            </a-form>
        </div>
    </div>
</template>

<script>
import myAxios from "@/utils";
import { messageSuccess } from "@/utils/ui-feedback";
import { IconLock } from '@arco-design/web-vue/es/icon';

export default {
    name: "zpk_namespace",
    components: {
        IconLock,
    },
    props: {
        userInfo: {
            type: Object,
            default: () => ({})
        },
        embedded: {
            type: Boolean,
            default: false
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
.access-page {
    min-height: 100%;
}

.access-panel {
    padding: 24px 30px;
    background: #ffffff;
    border-top: 1px solid #eeeeee;
}

.access-page.is-embedded .access-panel {
    padding: 0;
    border-top: 0;
}

.access-form {
    max-width: 520px;
}

.access-page.is-embedded .access-form {
    max-width: none;
}

.access-password-row {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 320px;
}

.access-hidden-password {
    width: 210px;
}

.access-hidden-password :deep(.arco-icon) {
    color: #86909c;
    font-size: 16px;
}

.access-action-row {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
}
</style>
