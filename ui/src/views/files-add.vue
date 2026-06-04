<template>
    <div class="bg-white" style="min-height:100vh;">
        <div class="zpk-page-header">
            <span class="backbtn df ai-c" @click="$router.go(-1)">
                <icon-arrow-left class="backicon" />
            </span>
            <a-breadcrumb>
                <a-breadcrumb-item><router-link to="/zpk" class="c-99 fw-400">制品管理</router-link></a-breadcrumb-item>
                <a-breadcrumb-item>
                    <router-link :to="{ path: '/zpk-version', query: { id: identifie, title: vtitle } }"
                        class="c-99 fw-400">{{ vtitle || identifie }}</router-link>
                </a-breadcrumb-item>
                <a-breadcrumb-item>
                    <router-link :to="{ path: '/zpk-edit', query: { id: identifie, versionid: version_id } }"
                        class="c-99 fw-400">应用基础信息修改</router-link>
                </a-breadcrumb-item>
                <a-breadcrumb-item><span class="c-33 fw-400">文件编辑</span></a-breadcrumb-item>
            </a-breadcrumb>
        </div>
        <div class="bg-padding pb-24">
            <files-editor :filename="filename" :version_id="version_id" :filecont="filecont" @complete="complete"></files-editor>
        </div>
    </div>
</template>

<script>
import myAxios from '@/utils';
import filesEditor from '@/components/files-editor.vue';
import { messageSuccess } from '@/utils/ui-feedback';
import { IconArrowLeft } from '@arco-design/web-vue/es/icon';
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
    components: { filesEditor, IconArrowLeft },
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
                    messageSuccess('操作成功');
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
