<template>
    <div style="padding:20px;">
        <el-tabs v-model="tabActive" @tab-change="changeTab">
            <el-tab-pane label="未审核" name="0"></el-tab-pane>
            <el-tab-pane label="已审核" name="1"></el-tab-pane>
        </el-tabs>

        <el-table v-loading="loading"  class="table-header" :data="list">
            <el-table-column prop="title" label="名称">
                <template #default="scope">
                    <div style="display: flex;gap: 4px;align-items: center">
                        <img :src="getLogo(scope.row.icon)" style="width: 20px;height: 20px" @error="(e) => {e.target.src = dfimg;}" alt="">
                        <span class="c-blue cursor" @click="installEvent(scope.row)">{{scope.row.title}}</span>
                    </div>
                </template>
            </el-table-column>
            <el-table-column prop="identifie" label="标识" />
            <el-table-column v-if="tabActive=='1'" prop="audit_status" label="审核状态">
                <template #default="scope">
                    <span v-if="scope.row.audit_status==2" class="c-red">不通过</span>
                    <span v-if="scope.row.audit_status==3" class="c-green">通过</span>
                    <span v-if="scope.row.audit_status==3 && scope.row.publish_official_store_status!=0 && scope.possTxt">（官方制品库{{scope.possTxt}}）</span>
                </template>
            </el-table-column>
            <el-table-column v-if="tabActive=='1'" prop="audit_remark" label="理由" />
            <el-table-column label="操作">
                <template #default="scope">
                    <el-button v-if="tabActive=='0'" type="text" @click="toReview(scope.row)">审核</el-button>
                    <el-button v-if="tabActive=='1'" type="text" @click="revoke(scope.row)">撤销审核</el-button>
                    <el-button v-if="scope.row.audit_status==3 && can_push_official_store && scope.row.publish_official_store_status==0" type="text" @click="addTozpk(scope.row)">添加到官方制品仓库</el-button>
                </template>
            </el-table-column>
        </el-table>
        <div class="mt-10 df jc-e">
            <el-pagination :current-page="page" background layout="prev, pager, next" @current-change="v=>{page=v;getList()}" :total="total" />
        </div>


        <el-dialog v-model="review.show" title="审核" width="700px">
            <el-form ref="form" :model="review" label-width="80px" style="margin-right:10px;">
                <el-form-item label="名称">{{ review.title }}</el-form-item>
                <el-form-item label="审核">
                    <el-radio-group v-model="review.audit_status">
                        <el-radio :label="3">通过</el-radio>
                        <el-radio :label="2">不通过</el-radio>
                    </el-radio-group>
                </el-form-item>
                <el-form-item label="原因" prop="audit_remark" :rules="[{required: review.audit_status==2, message: '内容不能为空' }]">
                    <el-input v-model="review.audit_remark" type="textarea" placeholder="请输入原因" :rows="5"></el-input>
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="review.show = false">取消</el-button>
                    <el-button type="primary" @click="reviewSubmit">确定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>
<script>
import myAxios from '@/utils'
import dfimg from '@/assets/img/dfimg.png';
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
                this.$message.success('操作成功');
                this.getList();
            })
        },

        revoke(row){
            myAxios.post('/respo/audit/audit',{
                identifie: row.identifie,
                audit_status: 1,
                audit_remark: '',
            }).then(res=>{
                this.$message.success('操作成功');
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
            this.$refs.form.validate((valid) => {
                if(!valid){return}
                myAxios.post('/respo/audit/audit',{
                    identifie: this.review.identifie,
                    audit_status: this.review.audit_status,
                    audit_remark: this.review.audit_remark,
                }).then(res=>{
                    this.$message.success('操作成功');
                    this.review.show = false;
                    this.getList();
                })
            });
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
</style>
