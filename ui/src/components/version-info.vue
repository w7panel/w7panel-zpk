<template>
    <div class="df">
        <div class="app-icon">
            <img v-if="logoimg" :src="logoimg" class="img df-s0" alt="" />
            <div v-else class="df df-c ai-c jc-c"
                style="width:100%;height:100%;border:1px solid #e7e7e7;border-radius:4px;">
                <el-icon>
                    <Plus />
                </el-icon>
                <span class="c-99 fs-12 lh-1 mt-8">点击上传</span>
            </div>
            <input type="file" @change="uplogo" accept="image/*" />
        </div>
        <el-form class="ml-40 fc version-info-form" @keydown.enter.prevent>
            <el-form-item label="名称" style="margin-bottom:10px;">
                <span v-if="edit.type != 'name'">{{ form.name || '-' }}</span>
                <span v-if="edit.type != 'name'" class="c-blue cursor editbtn"
                    @click="edit.type = 'name'; edit.name = form.name;">修改</span>
                <el-input v-if="edit.type == 'name'" v-model="edit.name" placeholder="请输入"
                    style="width:160px;"></el-input>
                <span v-if="edit.type == 'name'" class="c-blue cursor ml-20" @click="changeForm('name')">确定</span>
            </el-form-item>
            <el-form-item label="标识" style="margin-bottom:10px;">
                <span>{{ identifie }}</span>
            </el-form-item>
            <el-form-item label="描述" style="margin-bottom:10px;">
                <span v-if="edit.type != 'description'">{{ form.description || '-' }}</span>
                <span v-if="edit.type != 'description'" class="c-blue cursor editbtn"
                    @click="edit.type = 'description'; edit.description = form.description;">修改</span>
                <el-input v-if="edit.type == 'description'" v-model="edit.description" placeholder="请输入"
                    style="width:160px;"></el-input>
                <span v-if="edit.type == 'description'" class="c-blue cursor ml-20"
                    @click="changeForm('description')">确定</span>
            </el-form-item>
            <el-form-item label="标签" style="margin-bottom:10px;">

                <div class="df df-ww">
                    <el-tag type="primary" v-for="(item, index) in form.tags" :key="index" size="large"
                        :closable="edit.type == 'tags'" @close="deleteTag(index)" class="tag">{{ item.name }}</el-tag>
                    <div v-if="edit.type == 'tags'" class="df">
                        <el-select v-model="form.taginput" multiple collapse-tags size="default" style="width:140px;"
                            placeholder="添加新标签">
                            <el-option v-for="item in tags" :disabled="Boolean(form.tags.find(i => i.name == item.name))"
                                :key="item.id" :label="item.name" :value="item.id"></el-option>
                        </el-select>
                        <el-button type="primary" size="default" @click="addTag"
                            style="margin-left:10px;">确定</el-button>
                        <el-button @click="edit.type = ''" size="default">取消</el-button>
                    </div>
                </div>

                <span v-if="edit.type != 'tags' && (!form.tags || !form.tags.length)">-</span>
                <span v-if="edit.type != 'tags'" class="c-blue cursor editbtn" @click="edit.type = 'tags'">修改</span>
            </el-form-item>

            <el-form-item label="注解" style="margin-bottom:10px;">
                {{ annotationKeys }}<span class="c-blue cursor ml-20" @click="openAnnotationEdit">编辑</span>
            </el-form-item>
            <el-form-item label="属性">
                <div class="df df-c">
                    <el-checkbox v-model="edit.once" @change="changeForm('once')">仅安装一次</el-checkbox>
                    <el-checkbox v-model="edit.clusterPrivileges"
                        @change="changeForm('clusterPrivileges')">集群特权</el-checkbox>
                    <el-checkbox v-model="edit.registerSite" @change="changeForm('registerSite')">创建站点</el-checkbox>
                </div>
            </el-form-item>
        </el-form>


        <el-dialog v-model="annotationEdit.show" title="注解" width="1000px">
            <table class="table">
                <thead>
                    <tr>
                        <td>键</td>
                        <td>值</td>
                        <td>操作</td>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="(item, index) in annotationEdit.list" :key="index">
                        <td>
                            <el-input v-model="item.key" placeholder="请输入" style="width:200px;"></el-input>
                        </td>
                        <td>
                            <el-input v-model="item.value" placeholder="请输入" type="textarea" autosize
                                style="width: 600px;"></el-input>
                        </td>
                        <td>
                            <span class="c-blue cursor" @click="annotationEdit.list.splice(index, 1)">删除</span>
                        </td>
                    </tr>
                    <tr>
                        <td colspan="3" class="cursor txt-c" @click="annotationEdit.list.push({ key: '', value: '' })">
                            <span class="addmenu"><el-icon :size="14">
                                    <Plus />
                                </el-icon>添加注解</span>
                        </td>
                    </tr>
                </tbody>
            </table>
            <div class="df ai-c jc-c mt-20">
                <el-button @click="annotationEdit.show = false;">取消</el-button>
                <el-button @click="submitAnnotation" type="primary">确定</el-button>
            </div>
        </el-dialog>
    </div>
</template>
<script>
import jsyaml from "js-yaml";
import myAxios from '@/utils';

const defaultManifest = `application:
    name: ''
    identifie: ''
    description: ''
    author: ''
    once: false
v: 2
`

export default {
    props: ['identifie', 'info'],
    data() {
        return {
            baseurl: '',
            logoimg: '',
            logofile: null,
            json: null,

            form: {
                name: '',
                tags: [],
                once: false,
                clusterPrivileges: false,
                registerSite: false,
                description: '',
                taginput: '',
            },
            edit: {
                type: '',
                once: false,
                clusterPrivileges: false,
                registerSite: false,
            },
            annotationEdit: {
                show: false,
                list: [],
            },
            tags: [],
        }
    },
    created() {
        this.baseurl = window?.$wujie?.props?.url || '';
        this.getTag();
        this.init();
    },
    watch: {
        identifie() {
            this.getTag();
        },
        info() {
            this.init();
        },
    },
    methods: {
        init() {
            if (!this.info) { return }
            let manifest = this.info?.manifest || defaultManifest;
            this.json = jsyaml.load(manifest);
            this.json.application.identifie = this.identifie;
            this.annotationEdit.list = this.json.application?.annotation || [];

            if (this.json.application) {
                this.form.once = this.json.application?.once || false;
                this.form.name = this.json.application?.name || '';
                this.form.description = this.json?.application?.description || '';
                this.form.clusterPrivileges = this.json?.application?.clusterPrivileges || false;
                this.form.registerSite = this.json?.application?.registerSite || false;
            }
            this.edit.once = this.json?.application?.once || false;
            this.edit.clusterPrivileges = this.json?.application?.clusterPrivileges || false;
            this.edit.registerSite = this.json?.application?.registerSite || false;

            this.logoimg = this.info?.icon_url;
            if (this.logoimg && !/^https?:\/\//.test(this.logoimg)) {
                this.logoimg = this.baseurl + this.logoimg;
            }
            this.form.tags = this.info?.tags || [];
        },

        async getTag() {
            if (!this.identifie) { return }

            await myAxios.post('/respo/tag/list', { limit: 999 }).then(res => {
                this.tags = res.data?.data?.list || [];
            }).catch(() => { })

        },

        uplogo(event) {
            this.logofile = event.target.files[0];
            if (!this.logofile) { return }
            let formdata = new FormData();
            formdata.append('identifie', this.identifie);
            formdata.append('file', this.logofile);
            myAxios.post('/respo/icon', formdata).then(res => {
                this.$message.success('添加成功');
                this.logoimg = res.data?.data?.url;
                if (!/^https?:\/\//.test(this.logoimg)) {
                    this.logoimg = this.baseurl + this.logoimg;
                }
                this.logoimg = this.logoimg + '?time=' + Date.now();
            });
        },

        deleteTag(index) {
            let formulaId = this.info?.version?.formula_id;
            if (!formulaId) { return }
            myAxios.post('/respo/tag/delete', {
                tagId: this.form.tags[index].id,
                formulaId: formulaId,
            }).then(res => {
                this.$message.success('删除成功');
                this.form.tags.splice(index, 1);
                this.edit.type = '';
                this.$emit('refresh');
            })
        },

        async addTag() {
            if (!this.form.taginput?.length) { this.edit.type = ''; return }
            let ids = this.form.taginput;
            let names = ids?.map(id => {
                return this.tags.find(i => i.id == Number(id))?.name
            });
            for (let i in names) {
                let name = names[i];
                await myAxios.post('/respo/tag/add', {
                    identifie: this.identifie,
                    name: name,
                }).then(res => {
                    this.$message.success('添加成功');
                    this.form.tags.push({ name: name, id: res.data.id });
                    this.form.taginput = '';
                    this.edit.type = '';
                }).catch(() => { });
            }
            this.$emit('refresh');
        },
        openAnnotationEdit() {
            let list = Object.entries(this.json?.application?.annotation || {}).map(([k, v]) => ({ key: k, value: v }))
            this.annotationEdit = {
                show: true,
                list: list,
            }
        },
        submitAnnotation() {
            let obj = {};
            this.annotationEdit.list.filter(i => i.key && i.value).map(i => {
                obj[i.key] = String(i.value);
            })
            this.json.application.annotation = obj;
            this.annotationEdit.show = false;
            this.submit();
        },
        changeForm(type) {
            if (this.edit[type] == this.form[type]) {
                this.edit.type = '';
                return;
            }
            this.json.application[type] = this.edit[type];
            this.edit.type = '';
            this.submit();
        },
        submit() {
            let yaml = jsyaml.dump(this.json, {
                indent: 4,
                sortKeys: (a, b) => {
                    if (b == 'menu') { return -1; }
                    return a > b ? 1 : -1;
                },
            });
            myAxios.post('/respo/file', {
                identifie: this.identifie,
                filename: 'manifest.yaml',
                content: yaml,
                version: String(this.info?.version?.id),
            }).then((res) => {
                this.$message.success('操作成功');
                this.$emit('refresh');
            })
        },
    },
    computed: {
        annotationKeys() {
            const keys = Object.keys(this.json?.application?.annotation || []);
            return keys.length === 0 ? '-' : keys.join(',');
        }
    }
}
</script>
<style scoped>
.app-icon {
    width: 64px;
    height: 64px;
    position: relative;
    border-radius: 8px;
}

.app-icon .img {
    width: 64px;
    height: 64px;
    display: block;
    border-radius: 8px;
}

.app-icon input[type='file'] {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 2;
    min-width: 0;
    opacity: 0;
    cursor: pointer;
}

.app-icon input[type='file']::-webkit-file-upload-button {
    display: none;
}

.tag {
    height: 24px;
    margin-right: 6px;
    margin-bottom: 3px;
    margin-top: 3px;
}

.tag-cpn {
    width: 100%;
    box-sizing: border-box;
    border-radius: 4px;
    padding: 7px 7px 1px;
    border: 1px solid #dcdfe6;
}

.tag-cpn.active {
    border-color: #409eff;
}

.tag-cpn .tag {
    height: 24px;
    margin-right: 6px;
    margin-bottom: 6px;
    margin-top: 0;
}

.tag-cpn .input {
    width: -webkit-min-content;
    width: min-content;
    height: 24px;
    min-width: 60px;
    margin-bottom: 6px;
}

.tag-cpn .input input {
    width: 100%;
    height: 100%;
    display: block;
    border: 0;
    outline: 0;
    padding: 0 6px;
}

.tag-cpn .input input::placeholder {
    color: #999;
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
</style>
<style>
.version-info-form .el-form-item .editbtn {
    display: none;
    margin-left: 20px;
}

.version-info-form .el-form-item:hover .editbtn {
    display: inline;
}
</style>