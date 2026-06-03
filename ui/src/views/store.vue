<template>
    <div style="padding:20px;">
        <a-tabs v-model:active-key="tabActive" @change="changeTab">
            <a-tab-pane key="0" title="未审核"></a-tab-pane>
            <a-tab-pane key="1" title="已审核"></a-tab-pane>
        </a-tabs>

        <a-table :loading="loading" class="table-header" :data="list" :pagination="false" row-key="identifie">
            <template #columns>
                <a-table-column data-index="title" title="名称">
                    <template #cell="{ record }">
                        <div style="display: flex;gap: 4px;align-items: center">
                            <img :src="getLogo(record.icon)" style="width: 20px;height: 20px" @error="(e) => {e.target.src = dfimg;}" alt="">
                            <span class="c-blue cursor" @click="installEvent(record)">{{record.title}}</span>
                        </div>
                    </template>
                </a-table-column>
                <a-table-column data-index="identifie" title="标识" />
                <a-table-column v-if="tabActive=='1'" data-index="audit_status" title="审核状态">
                    <template #cell="{ record }">
                        <span v-if="record.audit_status==2" class="c-red">不通过</span>
                        <span v-if="record.audit_status==3" class="c-green">通过</span>
                        <span v-if="record.audit_status==3 && record.publish_official_store_status!=0 && record.possTxt">（官方制品库{{record.possTxt}}）</span>
                    </template>
                </a-table-column>
                <a-table-column v-if="tabActive=='1'" data-index="audit_remark" title="理由" />
                <a-table-column title="操作">
                    <template #cell="{ record }">
                        <a-button v-if="tabActive=='0'" type="text" @click="toReview(record)">审核</a-button>
                        <a-button v-if="tabActive=='1'" type="text" @click="revoke(record)">撤销审核</a-button>
                        <a-button v-if="record.audit_status==3 && can_push_official_store && record.publish_official_store_status==0" type="text" @click="addTozpk(record)">添加到官方制品仓库</a-button>
                    </template>
                </a-table-column>
            </template>
        </a-table>
        <div class="mt-10 df jc-e">
            <a-pagination v-model:current="page" :page-size="limit" :total="total" @change="handlePageChange" />
        </div>


        <a-modal v-model:visible="review.show" title="审核" :width="700" :footer="false">
            <a-form ref="form" :model="review" :label-col-props="{ span: 3, flex: '0 0 80px' }"
                :wrapper-col-props="{ span: 21, flex: '1' }" label-align="left" class="review-form">
                <a-form-item label="名称">{{ review.title }}</a-form-item>
                <a-form-item label="审核">
                    <a-radio-group v-model="review.audit_status">
                        <a-radio :value="3">通过</a-radio>
                        <a-radio :value="2">不通过</a-radio>
                    </a-radio-group>
                </a-form-item>
                <a-form-item label="原因" field="audit_remark" :rules="[{required: review.audit_status==2, message: '内容不能为空' }]">
                    <a-textarea v-model="review.audit_remark" placeholder="请输入原因" :auto-size="{ minRows: 5, maxRows: 5 }" />
                </a-form-item>
                <a-form-item>
                    <a-button @click="review.show = false">取消</a-button>
                    <a-button type="primary" @click="reviewSubmit">确定</a-button>
                </a-form-item>
            </a-form>
        </a-modal>
    </div>
</template>
<script>
import myAxios from '@/utils'
import dfimg from '@/assets/img/dfimg.png';
import { messageSuccess } from '@/utils/ui-feedback';
export default{
    data(){
        return {
            tabActive: '0',
            page: 1,
            limit: 10,
            total: 0,
            list: [],
            loading: false,
            can_push_official_store: false,
            dfimg: dfimg,

            review: {
                show: false,
            },
            webUrl: '',
        }
    },
    created(){
        this.getList();
    },
    methods: {
        getLogo(url) {
            let base = this.webUrl;
            if(url){
                url = url + '?time=' + Date.now();
            }
            let icon = /^(https?:)?\/\//.test(url)? url : (url?base+url:dfimg);
            return icon
        },
        addTozpk(row){
            myAxios.post('/respo/add-to-official',{
                identifie: row.identifie,
            }).then(res=>{
                messageSuccess('操作成功');
                this.getList();
            })
        },

        revoke(row){
            myAxios.post('/respo/audit/audit',{
                identifie: row.identifie,
                audit_status: 1,
                audit_remark: '',
            }).then(res=>{
                messageSuccess('操作成功');
                this.getList();
            })
        },
        toReview(row){
            this.review = {
                show: true,
                title: row.title,
                identifie: row.identifie,
                audit_status: 3,
                audit_remark: '',
            }
        },
        reviewSubmit(){
            this.$refs.form.validate((errors) => {
                if(errors){return}
                myAxios.post('/respo/audit/audit',{
                    identifie: this.review.identifie,
                    audit_status: this.review.audit_status,
                    audit_remark: this.review.audit_remark,
                }).then(res=>{
                    messageSuccess('操作成功');
                    this.review.show = false;
                    this.getList();
                })
            });
        },
        handlePageChange(v) {
            this.page = v;
            this.getList();
        },
        changeTab(v){
            this.page = 1;
            this.total = 0;
            this.list = [];
            this.getList();
        },
        getList(){
            this.loading = true;
            myAxios.post('/respo/audit/list',{
                audit_status: this.tabActive=='0'?[1]:[2,3],
                page: this.page,
                limit: this.limit,
            }).then(res=>{
                this.webUrl =res.data?.data?.webUrl;
                let list = res.data?.data?.list || [];
                this.total = res.data?.data?.total;
                this.can_push_official_store = res.data?.data?.can_push_official_store;
                this.list = list?.map(i=>{
                    return {
                        title: i.name || i.identifie,
                        identifie: i.identifie,
                        publish_official_store_status: i.publish_official_store_status,
                        possTxt: {1:'审核中',2:'不通过',3:'通过'}[i.publish_official_store_status],
                        audit_remark: i.audit_remark,
                        audit_status: i.audit_status,
                        icon: i.icon,
                    }
                })
            }).finally(()=>{
                this.loading = false;
            })
        },
        installEvent(data){
            let url = this.webUrl + '/respo/info/' + data.identifie;
            window.$wujie?.bus?.$emit?.('toStoreInstall', url)
        },
    }
}
</script>
<style scoped>
.table-header :deep(.arco-table-th) {
    background: #F3F3F3;
    color: #666666;
    font-weight: 500;
}

.review-form :deep(.arco-form-item-label-col) {
    width: 80px;
}

.review-form :deep(.arco-form-item-label) {
    white-space: nowrap;
}

.review-form :deep(.arco-form-item-wrapper-col) {
    min-width: 0;
}
</style>
