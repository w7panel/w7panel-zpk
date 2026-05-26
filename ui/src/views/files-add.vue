<template>
    <div class="bg-white" style="min-height:100vh;">
        <div style="padding:20px; border:1px solid #E7E7E7;">
            <el-breadcrumb separator="/">
                <el-breadcrumb-item :to="{path:'/zpk'}"><template #default><span class="c-99 fw-400">我的制品库</span></template></el-breadcrumb-item>
                <el-breadcrumb-item :to="{path:'/zpk-version',query:{id:this.identifie,title:vtitle}}"><template #default><span class="c-99 fw-400">版本管理</span></template></el-breadcrumb-item>
                <el-breadcrumb-item :to="{path:'/zpk-edit',query:{id:identifie,versionid:version_id}}"><template #default><span class="c-99 fw-400">应用基础信息修改</span></template></el-breadcrumb-item>
                <el-breadcrumb-item><template #default><span class="c-33 fw-400">文件编辑</span></template></el-breadcrumb-item>
            </el-breadcrumb>
        </div>
        <div class="bg-padding pb-24">
            <files-editor :filename="filename" :version_id="version_id" :filecont="filecont" @complete="complete"></files-editor>
        </div>
    </div>
</template>

<script>
import myAxios from '@/utils';
import filesEditor from '@/components/files-editor.vue';
export default {
    data(){
        return {
            vtitle: '',
            identifie: '',
            list: {},
            filename: '',
            filecont: '',
        }
    },
    components: {filesEditor},
    created(){
        this.vtitle = this.$route.query.vtitle;
        this.identifie = this.$route.query.identifie;
        this.filename = this.$route.query.filename || '';
        this.version_id = this.$route.query.version_id || '',
        this.getFile();
    },
    methods:{
        getFile(){
            myAxios.post('/respo/path-tree',{
                identifie:this.identifie,
                version: this.version_id,
            }).then(res=>{
                this.list = res?.data?.data?.list || {};
                if(this.filename){
                    this.filecont = this.list[this.filename] || '';
                }
            });
        },
        complete(form){
            myAxios.post('/respo/file',{
                identifie: this.identifie,
                filename: form.title,
                content: form.content,
                version: this.version_id,
            }).then(()=>{
                this.getInfo(this.identifie, ()=>{
                    this.$message.success('操作成功');
                    setTimeout(()=>{
                        this.$router.go(-1);
                    },500);
                })
            });
        },
        getInfo(id,callback,n){
            n = n || 0;
            myAxios.get('/respo/v2/info/'+id + '/' + this.version_id).then(res=>{
                callback && callback();
            }).catch(()=>{
                if(n>10){return}
                setTimeout(()=>{
                    this.getInfo(id,callback,n+1);
                },1000);
            });
        },
    }
}
</script>

<style scoped>
.top{padding:10px 20px; height:32px;}
</style>