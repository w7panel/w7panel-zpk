<template>
  <div style="min-height:100%;">
    <div>
      <div>
        <div class="bg-white bg-padding pb-24" style="border-top:1px solid #EEEEEE;">
          <div class="df jc-b">
            <div>
              <span>命名空间：</span>
              <el-select v-model="activeNamespace" @change="changeActiveNamespace">
                <el-option :value="-1" label="所有命名空间" />
                <el-option v-for="item in namespace" :key="item.id" :value="item.name" />
              </el-select>
              <span style="margin-left:10px;">二级命名空间：</span>
              <el-select v-model="subNamespace" @change="getData(1)">
                <el-option :value="-1" label="所有命名空间" />
                <el-option v-for="item in subNamespaceData" :key="item" :value="item" />
              </el-select>
            </div>
            <el-button type="primary" @click="add">
              新建镜像
            </el-button>
          </div>
          <el-alert type="warning" class="registry-primary-el-warning mt-20 fs-14" :closable="false">
            <div>docker login命令登录信息可在 <span class="cursor" style="border-bottom: 1px solid var(--el-color-primary);display: inline-block;line-height: 1;" @click="accessDialogShow = true"><ArcoIcon name="icon-244" :size="16" color="#3370ff" style="vertical-align: text-top;line-height: 1;"/>访问凭证</span>中获取
            </div>
            <div class="mt-6">外网地址：<span class="cursor txt-line"
                @click="serverdomain.external_domain ? onekeyCopy(serverdomain.external_domain) : null">{{ serverdomain.external_domain || '' }}</span>
            </div>
            <div class="mt-6">内网地址：<span class="cursor txt-line"
                @click="serverdomain.intranet_domain ? onekeyCopy(serverdomain.intranet_domain) : null">{{ serverdomain.intranet_domain || '' }}</span>
            </div>
          </el-alert>
          <el-table :data="tableData" class="mt-20 table-header">
            <el-table-column label="名称" prop="name" width="150">
              <template #default="scope">
                <router-link :to="'/zpk-registry/' + scope.row.id" class="el-button el-button--primary is-text"
                  style="padding: 0">
                  {{ scope.row.name }}
                </router-link>
              </template>
            </el-table-column>
            <el-table-column label="镜像地址">
              <template #default="scope">
                {{ scope.row.registry }}/{{ scope.row.namespace }}/{{ scope.row.name }}
                <el-icon :size="12" style="cursor:pointer"
                  @click="onekeyCopy(`${scope.row.registry}/${scope.row.namespace}/${scope.row.name}`)">
                  <DocumentCopy />
                </el-icon>
              </template>
            </el-table-column>
            <el-table-column label="类型" width="100">
              <template #default="scope">
                {{ scope.row.visible_type === 3 ? namespaceTypeMap[scope.row.namespace] :
                  visibleTypeMap[scope.row.visible_type]}}
              </template>
            </el-table-column>

            <el-table-column label="命名空间" prop="namespace" width="150" />
            <el-table-column label="创建时间" prop="created_at" width="200">
              <template #default="scope">
                {{ new Date(scope.row.created_at).toLocaleString() }}
              </template>
            </el-table-column>
            <el-table-column align="left" label="操作" width="300">
              <template #default="scope">
                <template v-if="hasAccess(scope.row.user_id)">
                  <el-button type="text" @click="edit(scope.row)" style="padding-left: 0">修改</el-button>
                  <el-button type="text" @click="del(scope.row)">删除</el-button>
                </template>
                <el-button type="text" @click="shortcut(scope.row)">快捷指令</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="mt-20 df jc-e">
            <el-pagination v-model:page-size="paginate" :current-page="page" :page-count="last_page"
              :page-sizes="[10, 20, 30, 40]" background layout="sizes, prev, pager, next" @size-change="getData(1)"
              @current-change='getData'></el-pagination>
          </div>
        </div>
      </div>
    </div>
    <el-dialog v-model="visible" :title="editId ? '编辑镜像' : '添加镜像'" :width="500">
      <el-form ref="form" :model="form" label-position="left" label-width="80px">
        <el-form-item :rules="[{ required: true, message: '名称不能为空', trigger: 'manual' }]" label="名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item :rules="[{ required: true, message: '命名空间不能为空', trigger: 'manual' }]" label="命名空间" prop="namespace">
          <el-select v-model="form.namespace">
            <el-option v-for="item in namespace" :key="item.id" :value="item.name">
              {{ item.name }}
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="描述" prop="namespace">
          <el-input v-model="form.desc" :rows="5" type="textarea" />
        </el-form-item>
        <el-form-item label="公共权限" prop="namespace">
          <div class="df df-c" style="flex:1;">
            <el-checkbox v-model="formVisibleType3" @change="v => form.visible_type = v ? 3 : 1">跟随命名空间</el-checkbox>
            <el-radio-group v-model="form.visible_type" :disabled="formVisibleType3">

              <el-radio :label="1">私有读写</el-radio>
              <el-radio :label="4">公有读私有写</el-radio>
              <el-radio :label="2">公有读写</el-radio>
            </el-radio-group>
          </div>
        </el-form-item>
        <el-form-item>
          <el-button size="large" type="primary" @click="onSubmit">确定</el-button>
          <el-button size="large" @click="visible = false">取消</el-button>
        </el-form-item>
      </el-form>
    </el-dialog>
    <el-dialog v-model="shortcutVisible" :width="800" title="快捷指令">
      <div class="shortcut-header">登录容器镜像服务 Docker Registry</div>
      <div class="shortcut-content">
        <div>{{ `docker login ${shortcutData.registry} --username=${userInfo.username}` }}</div>
        <el-icon :size="12" style="cursor:pointer"
          @click="onekeyCopy(`docker login ${shortcutData.registry} --username=${userInfo.username}`)">
          <DocumentCopy />
        </el-icon>
      </div>
      <div class="shortcut-header">从 Registry 拉取镜像</div>
      <div class="shortcut-content">
        <div>{{ `docker pull ${shortcutData.registry}/${shortcutData.namespace}/${shortcutData.name}:[tag]` }}</div>
        <el-icon :size="12" style="cursor:pointer"
          @click="onekeyCopy(`docker pull ${shortcutData.registry}/${shortcutData.namespace}/${shortcutData.name}:[tag]`)">
          <DocumentCopy />
        </el-icon>
      </div>
      <div class="shortcut-desc">其中［tag］请根据您需要拉取镜像的具体版本镜像替换，如latest。</div>
      <div class="shortcut-header">向 Registry 中推送镜像</div>
      <div class="shortcut-content">
        <div>{{ `docker tag [imageId] ${shortcutData.registry}/${shortcutData.namespace}/${shortcutData.name}:[tag]` }}
        </div>
        <el-icon :size="12" style="cursor:pointer"
          @click="onekeyCopy(`docker tag [imageId] ${shortcutData.registry}/${shortcutData.namespace}/${shortcutData.name}:[tag]`)">
          <DocumentCopy />
        </el-icon>
      </div>
      <div class="shortcut-content">
        <div>{{ `docker push ${shortcutData.registry}/${shortcutData.namespace}/${shortcutData.name}:[tag]` }}</div>
        <el-icon :size="12" style="cursor:pointer"
          @click="onekeyCopy(`docker push ${shortcutData.registry}/${shortcutData.namespace}/${shortcutData.name}:[tag]`)">
          <DocumentCopy />
        </el-icon>
      </div>
      <div class="shortcut-desc">其中［ImageId］请替换力您所要推送的实际镜像ID，或使用本地镜像的完整路径，［tag］请替换为您期待的镜像版本。</div>
    </el-dialog>
    <el-dialog v-model="accessDialogShow" title="访问凭证" :width="800">
      <div>
        <Access :userInfo="userInfo" />
      </div>
    </el-dialog>
  </div>
</template>

<script>
import myAxios from "@/utils";
import { ElMessageBox, ElMessage } from 'element-plus';
import userMixin from "@/utils/user-mixin";
import Access from '@/views/access.vue';
import ArcoIcon from "@/components/arco-icon.vue";

export default {
  name: "zpk_registry",
  components: {
    Access,
    ArcoIcon
  },
  mixins: [userMixin],
  data() {
    return {
      accessDialogShow: false,
      formVisibleType3: false,
      visibleTypeMap: {
        1: '私有读写',
        2: '公有读写',
        4: '公有读私有写'
      },
      shortcutData: {},
      shortcutVisible: false,
      namespaceTypeMap: {},
      activeNamespace: -1,
      subNamespace: -1,
      subNamespaceData: [],
      domainVisible: false,
      form: {
        name: '',
        namespace: '',
        desc: '',
        visible_type: 3
      },
      domainForm: {
        registry_host: '',
        registry_username: '',
        registry_password: ''
      },
      namespace: [],
      editId: '',
      visible: false,
      page: 1,
      paginate: 10,
      last_page: 1,
      list: [],

      serverdomain: {
        external_domain: '',
        intranet_domain: '',
      },
    }
  },
  computed: {
    tableData() {
      return this.list.slice((this.page - 1) * this.paginate, this.page * this.paginate)
    },
  },
  created() {
    this.getRegistryInfo();
    this.getData(1);
    this.getNamespace()
  },
  methods: {
    changeActiveNamespace() {
      this.subNamespace = -1;
      this.subNamespaceData = [];
      if (this.activeNamespace !== -1) {
        myAxios.post('/v2/api/namespace/sub_namespace/list', { namespace: this.activeNamespace }).then(res => {
          this.subNamespaceData = res.data?.data?.list || [];
        })
      }
      this.getData(1);
    },
    getRegistryInfo() {
      myAxios.get(`/v2/api/registry/info`).then(res => {
        let data = res?.data?.data?.server || {};
        this.serverdomain = {
          ...this.serverdomain,
          ...data,
        };
      })
    },
    shortcut(data) {
      this.shortcutData = data || {}
      this.shortcutVisible = true
    },
    onekeyCopy(text) {
      var textarea = document.createElement('textarea');
      document.body.appendChild(textarea);
      textarea.style.position = 'fixed';
      textarea.style.clip = 'rect(0 0 0 0)';
      textarea.style.top = '10px';
      textarea.value = text;
      textarea.select();
      document.execCommand('copy', true);
      document.body.removeChild(textarea);
      this.$message.success("复制成功");
    },
    getNamespace() {
      myAxios.post("/v2/api/namespace/list").then(res => {
        let data = res.data?.data?.list ?? [];
        this.namespace = data;
        let map = {}
        data.forEach(i => {
          map[i.name] = this.visibleTypeMap[i.visible_type]
        })
        this.namespaceTypeMap = map
      });
    },
    getData(p, notChangePage) {
      if (!notChangePage) {
        this.page = p
      }
      if (p === 1) {
        myAxios.post("/v2/api/repository/list", {
          namespace: this.activeNamespace !== -1 ? this.activeNamespace : '',
          sub_namespace: this.subNamespace !== -1 ? this.subNamespace : '',
        }).then(res => {
          let data = res.data?.data?.list ?? [];
          this.list = data;
          this.last_page = Math.ceil(data.length / this.paginate);
        });
      }
    },
    add() {
      this.editId = ''
      this.form.name = ''
      this.form.namespace = ''
      this.form.desc = ''
      this.form.visible_type = 3
      this.formVisibleType3 = this.form.visible_type == 3
      this.visible = true
    },
    onSubmit() {
      this.$refs.form.validate((valid) => {
        if (!valid) {
          return
        }

        if (this.editId) {
          myAxios.post('/v2/api/repository/edit', { id: this.editId, ...this.form }).then(() => {
            this.$message.success('操作成功');
            this.getData(1, true);
            this.visible = false
          })
          return
        }
        myAxios.post('/v2/api/repository/add', this.form).then(() => {
          this.$message.success('操作成功');
          this.getData(1, false);
          this.visible = false
        })
      })
    },
    edit(row) {
      this.editId = row.id
      this.form.name = row.name
      this.form.namespace = row.namespace
      this.form.desc = row.desc
      this.form.visible_type = row.visible_type
      this.formVisibleType3 = this.form.visible_type == 3
      this.visible = true
    },
    del(row) {
      ElMessageBox.confirm('请确认删除', "提示", {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
      }).then(() => {
        myAxios.post("/v2/api/repository/del", { id: row.id }).then(res => {
          if (!res) { return }
          ElMessage({
            message: '删除成功',
            type: 'success',
          })
          this.getData(1, true);
        });
      }).catch(() => { });
    },
    see(row) {
      this.$router.push({ path: "/registry-detail/" + row.id });
    },
  }
}
</script>

<style>
.shortcut-header {
  font-weight: bold;
  font-size: 16px;
  margin-bottom: 10px;
}

.shortcut-content {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
  justify-content: space-between;
  background: #f0f3fa;
  padding: 10px;
  border-radius: 4px;
}

.shortcut-desc {
  color: #999;
}

.shortcut-desc+.shortcut-header {
  margin-top: 30px;
}

.registry-primary-el-warning {
  background-color: var(--el-color-primary-light-9) !important;
  color: var(--el-color-primary) !important;
}

.registry-primary-el-warning .el-alert__description {
  margin: 0;
  padding: 5px 0;
  color: var(--el-color-primary) !important;
}
</style>
