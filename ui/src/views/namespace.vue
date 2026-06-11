<template>
  <div style="min-height:100%;">
    <div>
      <div>
        <div class="bg-white bg-padding pb-24" style="border-top:1px solid #EEEEEE;">
          <a-button type="primary" @click="add">
            <template #icon><icon-plus /></template>
            新建命名空间
          </a-button>
          <a-table :data="tableData" :pagination="false" class="mt-20 table-header" row-key="id">
            <template #columns>
              <a-table-column title="名称" data-index="name" />
              <a-table-column title="仓库数量">
                <template #cell="{ record }">
                  {{ namespaceCountMap[record.name]?.count || 0 }}
                </template>
              </a-table-column>
              <a-table-column title="访问级别">
                <template #cell="{ record }">
                  {{ visibleTypeMap[record.visible_type] }}
                </template>
              </a-table-column>
              <a-table-column title="创建时间" data-index="created_at">
                <template #cell="{ record }">
                  {{ new Date(record.created_at).toLocaleString() }}
                </template>
              </a-table-column>
              <a-table-column title="操作">
                <template #cell="{ record }">
                  <template v-if="hasAccess(record.user_id)">
                    <a-button type="text" @click="edit(record)">修改</a-button>
                    <a-popconfirm v-if="record.name != 'zpk_oci'" content="确认要删除吗" type="warning" ok-text="确定"
                      cancel-text="取消" content-class="zpk-delete-popconfirm" :ok-button-props="{ status: 'danger' }"
                      :cancel-button-props="{ type: 'secondary' }" @ok="del(record)">
                      <a-button type="text" status="danger">删除</a-button>
                    </a-popconfirm>
                  </template>
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
    <a-modal v-model:visible="visible" :title="editId ? '编辑命名空间' : '添加命名空间'" :width="800" :footer="false">
      <a-form ref="form" :model="form" label-align="left" :label-col-props="{ span: 3, flex: '0 0 80px' }"
        :wrapper-col-props="{ span: 21, flex: '1' }" class="namespace-form">
        <a-form-item :rules="[{ required: true, message: '名称不能为空', trigger: 'manual' }]" label="名称" field="name">
          <a-input v-model="form.name" />
        </a-form-item>
        <a-form-item label="公共权限" field="visible_type">
          <a-radio-group v-model="form.visible_type">
            <a-radio :value="1">私有读写</a-radio>
            <a-radio :value="4">公有读私有写</a-radio>
            <a-radio :value="2">公有读写</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item v-if="userRole != 'normal'" label="用户权限">
          <manifest-config-table :rows="form.permissions" add-text="添加用户"
            @add="form.permissions.push({ username: '', permission: 1 })">
            <template #columns>
              <manifest-config-table-column data-index="user_id" title="用户">
                <template #cell="{ record }">
                  <a-select v-model="record.user_id" placeholder="请选择" class="permission-select">
                    <a-option v-for="(item, index) in userList" :key="index" :label="item.username"
                      :value="item.id"></a-option>
                  </a-select>
                </template>
              </manifest-config-table-column>
              <manifest-config-table-column data-index="permission" title="权限">
                <template #cell="{ record }">
                  <a-radio-group v-model="record.permission">
                    <a-radio :value="1">只读</a-radio>
                    <a-radio :value="2">读写</a-radio>
                  </a-radio-group>
                </template>
              </manifest-config-table-column>
              <manifest-config-table-column title="操作" width="60px">
                <template #cell="{ index }">
                  <span class="cursor c-blue" @click="form.permissions.splice(index, 1)">删除</span>
                </template>
              </manifest-config-table-column>
            </template>
          </manifest-config-table>
        </a-form-item>
      </a-form>
      <div class="dialog-footer">
        <a-button size="large" @click="visible = false">取消</a-button>
        <a-button size="large" type="primary" @click="onSubmit">确定</a-button>
      </div>
    </a-modal>
  </div>
</template>

<script>
import myAxios from "@/utils";
import ManifestConfigTable from '@/components/manifest-config-table.vue';
import ManifestConfigTableColumn from '@/components/manifest-config-table-column.vue';
import { messageSuccess } from "@/utils/ui-feedback";
import userMixin from "@/utils/user-mixin";
import { IconPlus } from '@arco-design/web-vue/es/icon';

export default {
  name: "zpk_namespace",
  components: {
    ManifestConfigTable,
    ManifestConfigTableColumn,
    IconPlus,
  },
  data() {
    return {
      visibleTypeMap: {
        1: '私有读写',
        2: '公有读写',
        4: '公有读私有写'
      },
      namespaceCountMap: {},
      form: {
        name: '',
        namespace: '',
        desc: '',
        visible_type: 1,
        permissions: [],
      },
      userList: [],
      editId: '',
      visible: false,
      page: 1,
      paginate: 10,
      last_page: 1,
      list: [],

      userRole: '',
    }
  },
  mixins: [userMixin],
  computed: {
    tableData() {
      return this.list.slice((this.page - 1) * this.paginate, this.page * this.paginate)
    }
  },
  async created() {
    await myAxios.get('/v2/api/user/cur-user/info').then(res => {
      this.userRole = res.data?.data?.user?.role;
    })
    if (this.userRole !== 'normal') {
      this.getUserList();
    }
    this.getData(1);
  },
  methods: {
    getUserList() {
      myAxios.post('/v2/api/user/list').then(res => {
        this.userList = res.data?.data?.list || [];
      })
    },
    getData(p, notChangePage) {
      if (!notChangePage) {
        this.page = p;
      }
      if (p === 1) {
        myAxios.post("/v2/api/namespace/list").then(res => {
          let data = res.data?.data?.list ?? [];
          this.list = data;
          this.last_page = Math.ceil(data.length / this.paginate);
          this.namespaceCountMap = res.data?.data?.namespace_registry_count || {}
        });
      }

    },
    handlePageSizeChange() {
      this.getData(1);
    },
    add() {
      this.editId = ''
      this.form.name = ''
      this.form.visible_type = 1
      this.form.permissions = [];
      this.visible = true
    },
    onSubmit() {
      this.$refs.form.validate(async (errors) => {
        if (errors) { return }
        if (this.editId) {
          await myAxios.post('/v2/api/namespace/edit', { id: this.editId, ...this.form }).then(() => { })
        } else {
          await myAxios.post('/v2/api/namespace/add', this.form).then(() => { })
        }
        if (this.userRole != 'normal') {
          let permissions = this.form.permissions.filter(i => i.user_id);
          permissions = permissions.map(i => {
            return {
              user_id: i.user_id,
              actions: {
                1: ['pull'],
                2: ['pull', 'push'],
              }[i.permission] || [],
            }
          })
          myAxios.post('/v2/api/permission/namespace/set', {
            namespace: this.form.name,
            permissions: permissions,
          }).then(res => {
            messageSuccess('操作成功');
            this.getData(1, false)
            this.visible = false
          })
        } else {
          messageSuccess('操作成功');
          this.getData(1, false)
          this.visible = false
        }
      })
    },
    async edit(row) {
      let permissions = [];
      if (this.userRole != 'normal') {
        permissions = await myAxios.post('/v2/api/permission/namespace/get', { namespace: row.name }).then(res => {
          return res.data?.data?.permissions || [];
        }).catch(() => ([]));
        permissions = permissions.map(item => {
          return {
            user_id: item.user_id,
            permission: item.actions?.includes?.('push') ? 2 : 1
          }
        });
        this.form.permissions = permissions;
      }

      this.editId = row.id
      this.form.name = row.name
      this.form.visible_type = row.visible_type
      this.visible = true
    },
    del(row) {
      return myAxios.post("/v2/api/namespace/del", { id: row.id }).then(res => {
        if (!res) { return }
        messageSuccess('删除成功')
        this.getData(1);
      });
    },
  }
}
</script>

<style scoped>
.ml-20 {
  margin-left: 20px;
}

.table-header :deep(.arco-table-th) {
  background: #F3F3F3;
  color: #666666;
  font-weight: 500;
}

.namespace-form :deep(.arco-form-item-label-col) {
  width: 80px;
}

.namespace-form :deep(.arco-form-item-label) {
  white-space: nowrap;
}

.namespace-form :deep(.arco-form-item-wrapper-col) {
  min-width: 0;
}

.permission-select {
  width: 220px;
}

.table {
  width: 100%;
}

.table td {
  padding: 10px;
  line-height: 1.4;
  border: 1px solid #eee;
  border-left: 0;
  border-right: 0;
}

.table tr:last-child td {
  background: transparent;
}

.table thead tr:first-child td {
  background: #f3f3f3;
  border-top: 0;
}
</style>
