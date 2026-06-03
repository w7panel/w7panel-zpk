<template>
  <div style="min-height:100%;">
    <div>
      <div>
        <div class="bg-white bg-padding pb-24" style="border-top:1px solid #EEEEEE;">
          <div class="df jc-b">
            <div>
              <span>命名空间：</span>
              <a-select v-model="activeNamespace" class="registry-filter-select" @change="changeActiveNamespace">
                <a-option :value="-1" label="所有命名空间" />
                <a-option v-for="item in namespace" :key="item.id" :label="item.name" :value="item.name" />
              </a-select>
              <span style="margin-left:10px;">二级命名空间：</span>
              <a-select v-model="subNamespace" class="registry-filter-select" @change="getData(1)">
                <a-option :value="-1" label="所有命名空间" />
                <a-option v-for="item in subNamespaceData" :key="item" :label="item" :value="item" />
              </a-select>
            </div>
            <a-button type="primary" @click="add">
              新建镜像
            </a-button>
          </div>
          <a-alert type="warning" class="registry-primary-warning mt-20 fs-14" :closable="false">
            <div>docker login命令登录信息可在 <span class="credential-link" @click="accessDialogShow = true"><ArcoIcon name="icon-244" :size="16" color="#3370ff" style="vertical-align: text-top;line-height: 1;"/>访问凭证</span>中获取
            </div>
            <div class="mt-6">外网地址：<span class="cursor txt-line"
                @click="serverdomain.external_domain ? onekeyCopy(serverdomain.external_domain) : null">{{ serverdomain.external_domain || '' }}</span>
            </div>
            <div class="mt-6">内网地址：<span class="cursor txt-line"
                @click="serverdomain.intranet_domain ? onekeyCopy(serverdomain.intranet_domain) : null">{{ serverdomain.intranet_domain || '' }}</span>
            </div>
          </a-alert>
          <a-table :data="tableData" :pagination="false" class="mt-20 table-header" row-key="id">
            <template #columns>
              <a-table-column title="名称" data-index="name" :width="150">
                <template #cell="{ record }">
                  <span class="registry-link" @click="goDetail(record)">
                    {{ record.name }}
                  </span>
                </template>
              </a-table-column>
              <a-table-column title="镜像地址">
                <template #cell="{ record }">
                  {{ record.registry }}/{{ record.namespace }}/{{ record.name }}
                  <a-tooltip content="复制">
                    <a-button class="registry-icon-action" type="text" shape="circle" size="mini"
                      @click="onekeyCopy(`${record.registry}/${record.namespace}/${record.name}`)">
                      <template #icon><icon-copy /></template>
                    </a-button>
                  </a-tooltip>
                </template>
              </a-table-column>
              <a-table-column title="类型" :width="130" class="registry-type-column">
                <template #cell="{ record }">
                  {{ record.visible_type === 3 ? namespaceTypeMap[record.namespace] :
                    visibleTypeMap[record.visible_type]}}
                </template>
              </a-table-column>

              <a-table-column title="命名空间" data-index="namespace" :width="150" />
              <a-table-column title="创建时间" data-index="created_at" :width="200">
                <template #cell="{ record }">
                  {{ new Date(record.created_at).toLocaleString() }}
                </template>
              </a-table-column>
              <a-table-column align="left" title="操作" :width="300">
                <template #cell="{ record }">
                  <template v-if="hasAccess(record.user_id)">
                    <a-button type="text" @click="edit(record)" style="padding-left: 0">修改</a-button>
                    <a-button type="text" status="danger" @click="del(record)">删除</a-button>
                  </template>
                  <a-button type="text" @click="shortcut(record)">快捷指令</a-button>
                </template>
              </a-table-column>
            </template>
          </a-table>
          <div class="mt-20 df jc-e">
            <a-pagination v-model:current="page" v-model:page-size="paginate" :total="list.length"
              :page-size-options="[10, 20, 30, 40]" show-page-size
              @page-size-change="handlePageSizeChange" @change="getData" />
          </div>
        </div>
      </div>
    </div>
    <a-modal v-model:visible="visible" :title="editId ? '编辑镜像' : '添加镜像'" :width="500" :footer="false">
      <a-form ref="form" :model="form" label-align="left" :label-col-props="{ span: 4, flex: '0 0 80px' }"
        :wrapper-col-props="{ span: 20, flex: '1' }" class="registry-form">
        <a-form-item :rules="[{ required: true, message: '名称不能为空', trigger: 'manual' }]" label="名称" field="name">
          <a-input v-model="form.name" />
        </a-form-item>
        <a-form-item :rules="[{ required: true, message: '命名空间不能为空', trigger: 'manual' }]" label="命名空间" field="namespace">
          <a-select v-model="form.namespace">
            <a-option v-for="item in namespace" :key="item.id" :value="item.name">
              {{ item.name }}
            </a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="描述" field="desc">
          <a-textarea v-model="form.desc" :auto-size="{ minRows: 5, maxRows: 5 }" />
        </a-form-item>
        <a-form-item label="公共权限" field="visible_type">
          <div class="df df-c" style="flex:1;">
            <a-checkbox v-model="formVisibleType3" @change="v => form.visible_type = v ? 3 : 1">跟随命名空间</a-checkbox>
            <a-radio-group v-model="form.visible_type" :disabled="formVisibleType3">

              <a-radio :value="1">私有读写</a-radio>
              <a-radio :value="4">公有读私有写</a-radio>
              <a-radio :value="2">公有读写</a-radio>
            </a-radio-group>
          </div>
        </a-form-item>
        <a-form-item>
          <a-button size="large" type="primary" @click="onSubmit">确定</a-button>
          <a-button size="large" @click="visible = false">取消</a-button>
        </a-form-item>
      </a-form>
    </a-modal>
    <a-modal v-model:visible="shortcutVisible" :width="800" title="快捷指令" :footer="false">
      <div class="shortcut-header">登录容器镜像服务 Docker Registry</div>
      <div class="shortcut-content">
        <div>{{ `docker login ${shortcutData.registry} --username=${userInfo.username}` }}</div>
        <a-tooltip content="复制">
          <a-button class="registry-icon-action" type="text" shape="circle" size="mini"
            @click="onekeyCopy(`docker login ${shortcutData.registry} --username=${userInfo.username}`)">
            <template #icon><icon-copy /></template>
          </a-button>
        </a-tooltip>
      </div>
      <div class="shortcut-header">从 Registry 拉取镜像</div>
      <div class="shortcut-content">
        <div>{{ `docker pull ${shortcutData.registry}/${shortcutData.namespace}/${shortcutData.name}:[tag]` }}</div>
        <a-tooltip content="复制">
          <a-button class="registry-icon-action" type="text" shape="circle" size="mini"
            @click="onekeyCopy(`docker pull ${shortcutData.registry}/${shortcutData.namespace}/${shortcutData.name}:[tag]`)">
            <template #icon><icon-copy /></template>
          </a-button>
        </a-tooltip>
      </div>
      <div class="shortcut-desc">其中［tag］请根据您需要拉取镜像的具体版本镜像替换，如latest。</div>
      <div class="shortcut-header">向 Registry 中推送镜像</div>
      <div class="shortcut-content">
        <div>{{ `docker tag [imageId] ${shortcutData.registry}/${shortcutData.namespace}/${shortcutData.name}:[tag]` }}
        </div>
        <a-tooltip content="复制">
          <a-button class="registry-icon-action" type="text" shape="circle" size="mini"
            @click="onekeyCopy(`docker tag [imageId] ${shortcutData.registry}/${shortcutData.namespace}/${shortcutData.name}:[tag]`)">
            <template #icon><icon-copy /></template>
          </a-button>
        </a-tooltip>
      </div>
      <div class="shortcut-content">
        <div>{{ `docker push ${shortcutData.registry}/${shortcutData.namespace}/${shortcutData.name}:[tag]` }}</div>
        <a-tooltip content="复制">
          <a-button class="registry-icon-action" type="text" shape="circle" size="mini"
            @click="onekeyCopy(`docker push ${shortcutData.registry}/${shortcutData.namespace}/${shortcutData.name}:[tag]`)">
            <template #icon><icon-copy /></template>
          </a-button>
        </a-tooltip>
      </div>
      <div class="shortcut-desc">其中［ImageId］请替换力您所要推送的实际镜像ID，或使用本地镜像的完整路径，［tag］请替换为您期待的镜像版本。</div>
    </a-modal>
    <a-modal v-model:visible="accessDialogShow" title="访问凭证" :width="800" :footer="false">
      <div>
        <Access :userInfo="userInfo" />
      </div>
    </a-modal>
  </div>
</template>

<script>
import myAxios from "@/utils";
import { confirm, messageSuccess } from "@/utils/ui-feedback";
import userMixin from "@/utils/user-mixin";
import Access from '@/views/access.vue';
import ArcoIcon from "@/components/arco-icon.vue";
import { IconCopy } from '@arco-design/web-vue/es/icon';

export default {
  name: "zpk_registry",
  components: {
    Access,
    ArcoIcon,
    IconCopy
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
      messageSuccess("复制成功");
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
    handlePageSizeChange() {
      this.getData(1);
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
      this.$refs.form.validate((errors) => {
        if (errors) {
          return
        }

        if (this.editId) {
          myAxios.post('/v2/api/repository/edit', { id: this.editId, ...this.form }).then(() => {
            messageSuccess('操作成功');
            this.getData(1, true);
            this.visible = false
          })
          return
        }
        myAxios.post('/v2/api/repository/add', this.form).then(() => {
          messageSuccess('操作成功');
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
      confirm({
        title: "提示",
        content: '请确认删除',
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        onOk: () => {
          return myAxios.post("/v2/api/repository/del", { id: row.id }).then(res => {
            if (!res) { return }
            messageSuccess('删除成功')
            this.getData(1, true);
          });
        }
      });
    },
    goDetail(row) {
      this.$router.push({ name: 'zpk-registry-detail', params: { id: row.id } });
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

.registry-primary-warning {
  background-color: #e8f3ff !important;
  color: #3370ff !important;
}

.registry-primary-warning .arco-alert-content {
  margin: 0;
  padding: 5px 0;
  color: #3370ff !important;
}

.credential-link {
  display: inline-block;
  line-height: 1;
  color: #3370ff;
  border-bottom: 1px solid #3370ff;
  cursor: pointer;
}

.registry-filter-select {
  width: 180px;
}

.table-header .arco-table-th {
  background: #F3F3F3;
  color: #666666;
  font-weight: 500;
}

.table-header .arco-table-tr .arco-table-th:nth-child(3),
.table-header .arco-table-tr .arco-table-td:nth-child(3) {
  white-space: nowrap;
}

.registry-link {
  color: #3370ff;
  cursor: pointer;
}

.registry-link:hover {
  text-decoration: underline;
}

.registry-icon-action {
  margin-left: 4px;
  color: #3370ff;
  vertical-align: middle;
}

.registry-form .arco-form-item-label-col {
  width: 80px;
}

.registry-form .arco-form-item-label {
  white-space: nowrap;
}

.registry-form .arco-form-item-wrapper-col {
  min-width: 0;
}
</style>
