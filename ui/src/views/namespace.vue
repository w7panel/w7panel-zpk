<template>
  <div style="min-height:100%;">
    <div>
      <div>
        <div class="bg-white bg-padding pb-24" style="border-top:1px solid #EEEEEE;">
          <el-button type="primary" @click="add">
            新建命名空间
          </el-button>
          <el-table :data="tableData" class="mt-20 table-header">
            <el-table-column label="名称" prop="name" />
            <el-table-column label="仓库数量">
              <template #default="scope">
                {{ namespaceCountMap[scope.row.name]?.count || 0 }}
              </template>
            </el-table-column>
            <el-table-column label="访问级别">
              <template #default="scope">
                {{ visibleTypeMap[scope.row.visible_type] }}
              </template>
            </el-table-column>
            <el-table-column label="创建时间" prop="created_at">
              <template #default="scope">
                {{ new Date(scope.row.created_at).toLocaleString() }}
              </template>
            </el-table-column>
            <el-table-column label="操作">
              <template #default="scope">
                <template v-if="hasAccess(scope.row.user_id)">
                  <el-button type="text" @click="edit(scope.row)">修改</el-button>
                  <el-button v-if="scope.row.name != 'zpk_oci'" type="text" @click="del(scope.row)">删除</el-button>
                </template>
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
    <el-dialog v-model="visible" :title="editId ? '编辑命名空间' : '添加命名空间'" :width="800">
      <el-form ref="form" :model="form" label-position="left" label-width="80px">
        <el-form-item :rules="[{ required: true, message: '名称不能为空', trigger: 'manual' }]" label="名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="公共权限" prop="visible_type">
          <el-radio-group v-model="form.visible_type">
            <el-radio :label="1">私有读写</el-radio>
            <el-radio :label="4">公有读私有写</el-radio>
            <el-radio :label="2">公有读写</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="userRole != 'normal'" label="用户权限">
          <table class="table">
            <thead>
              <tr>
                <td>用户</td>
                <td>权限</td>
                <td style="width:60px;">操作</td>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(item, index) in form.permissions" :key="index">
                <td>
                  <el-select v-model="item.user_id" placeholder="请选择">
                    <el-option v-for="(item, index) in userList" :key="index" :label="item.username"
                      :value="item.id"></el-option>
                  </el-select>
                </td>
                <td>
                  <el-radio-group v-model="item.permission">
                    <el-radio :label="1">只读</el-radio>
                    <el-radio :label="2">读写</el-radio>
                  </el-radio-group>
                </td>
                <td>
                  <span class="cursor c-blue" @click="form.permissions.splice(index, 1)">删除</span>
                </td>
              </tr>
              <tr>
                <td @click="form.permissions.push({ username: '', permission: 1 })" class="cursor txt-c" colspan="3">
                  <span class="addmenu"><el-icon :size="14">
                      <Plus />
                  </el-icon>添加用户</span>
                </td>
              </tr>
            </tbody>
          </table>
        </el-form-item>
        <el-form-item>
          <el-button size="large" type="primary" @click="onSubmit">确定</el-button>
          <el-button size="large" @click="visible = false">取消</el-button>
        </el-form-item>
      </el-form>
    </el-dialog>
  </div>
</template>

<script>
import myAxios from "@/utils";
import { ElMessageBox, ElMessage } from 'element-plus';
import userMixin from "@/utils/user-mixin";

export default {
  name: "zpk_namespace",
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
    add() {
      this.editId = ''
      this.form.name = ''
      this.form.visible_type = 1
      this.form.permissions = [];
      this.visible = true
    },
    onSubmit() {
      this.$refs.form.validate(async (valid) => {
        if (!valid) { return }
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
            this.$message.success('操作成功');
            this.getData(1, false)
            this.visible = false
          })
        } else {
          this.$message.success('操作成功');
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
      ElMessageBox.confirm('请确认删除', "提示", {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
      }).then(() => {
        myAxios.post("/v2/api/namespace/del", { id: row.id }).then(res => {
          if (!res) { return }
          ElMessage({
            message: '删除成功',
            type: 'success',
          })
          this.getData(1);
        });
      }).catch(() => { });
    },
  }
}
</script>

<style scoped>
.ml-20 {
  margin-left: 20px;
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