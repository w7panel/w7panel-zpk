<template>
  <div style="min-height:100%;">
    <div style="height:100vh;box-sizing:border-box;">
      <div>
        <div class="bg-white bg-padding pb-24" style="border-top:1px solid #EEEEEE;">
          <div class="df jc-b">
            <el-input v-model="search.keyword" style="width:256px;" placeholder="输入制品名称搜索" @keydown.enter="getData(1)">
              <template #suffix>
                <el-icon @click="getData(1)">
                  <Search />
                </el-icon>
              </template>
            </el-input>
            <div>
              <el-button v-if="is_register" @click="openCloudApps">云端找回</el-button>
              <el-button type="primary" @click="add.show = true;">添加制品</el-button>
            </div>
          </div>
          <el-table class="mt-20 table-header table-respo-list" :data="list">
            <el-table-column prop="name" label="制品名称">
              <template #default="scope">
                <div style="display: flex;gap: 4px;align-items: center">
                  <img :src="getLogo(scope.row.icon)" style="width: 20px;height: 20px"
                    @error="(e) => { e.target.src = dfimg; }" alt="">
                  <span v-if="scope.row.remote_formula_info_url" class="c-99">{{ scope.row.name }}</span>
                  <span v-else class="c-blue cursor"
                    @click="$router.push('/zpk-version?id=' + scope.row.identifie + '&title=' + scope.row.name)">{{
                      scope.row.name }}</span>
                  <span v-if="scope.row.goods_id" style="color: red;margin: 0 4px;">[付费]</span>
                  <el-icon :class="{ 'star-show': scope.row.status === 99 }" color="rgb(248,85,91)" class="star-icon"
                    style="cursor: pointer;" size="16px">
                    <StarFilled v-if="scope.row.status === 99" @click="changeStatus(2, scope.row.identifie)" />
                    <Star v-else @click="changeStatus(99, scope.row.identifie)" />
                  </el-icon>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="identifie" label="标识" />
            <el-table-column prop="version.name" label="版本" />
            <el-table-column prop="audit_status" label="审核状态">
              <template #default="scope">
                <span v-if="scope.row.audit_status == 1" class="c-99">待审核</span>
                <span v-if="scope.row.audit_status == 2" class="c-red">不通过</span>
                <span v-if="scope.row.audit_status == 3" class="c-green">通过</span>
              </template>
            </el-table-column>
            <el-table-column prop="audit_remark" label="理由" />
            <el-table-column label="操作">
              <template #default="scope">
                <el-button type="text" @click="installEvent(scope.row)">安装</el-button>

                <el-button type="text" @click="deleteItem(scope.row)">删除</el-button>
                <el-popover placement="bottom" width="auto">
                  <template #reference>
                    <el-button type="text">分享</el-button>
                  </template>
                  <div class="df" style="padding:6px 0;">
                    <el-button type="primary"
                      @click.stop="toinstall(false, scope.row.version?.name, scope.row.identifie)">复制安装地址</el-button>
                    <el-button type="primary"
                      @click.stop="toinstall(true, scope.row.version?.name, scope.row.identifie)">复制发布地址</el-button>
                  </div>
                </el-popover>
                <el-button type="text" @click="changeStatus(2, scope.row.identifie)"
                  v-if="scope.row.status === 1">上架</el-button>
                <el-button type="text" @click="changeStatus(1, scope.row.identifie)" v-else>下架</el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="mt-20 df jc-e">
            <el-pagination background v-model:page-size="paginate" layout="sizes, prev, pager, next"
              :current-page="page" :total="total" :page-sizes="[10, 20, 30, 40]" @size-change="getData(1)"
              @current-change='getData'></el-pagination>
          </div>

        </div>
      </div>
    </div>
    <el-dialog v-model="add.show" title="添加应用" width="540px">
      <el-form ref="form" :model="add" label-width="60px" v-loading="add.loading">
        <el-form-item label="标识" prop="title" :rules="[{ required: true, message: '标识不能为空', trigger: 'manual' }]">
          <w7-identifie v-model="add.title" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="addRespo">确定</el-button>
        </el-form-item>
      </el-form>
    </el-dialog>

    <el-dialog v-model="ipt.show" title="导入制品库" width="600px">
      <el-form :model="ipt" label-width="100px" v-loading="ipt.loading">
        <el-form-item label="制品库地址">
          <el-input v-model="ipt.link" placeholder="请输入地址" :spellcheck="false" @input="iptInputlink"></el-input>
        </el-form-item>
        <el-form-item label="标识">
          <el-input v-model="ipt.identifie" :spellcheck="false" placeholder="author_app"></el-input>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="iptSubmit">确定</el-button>
        </el-form-item>
      </el-form>
    </el-dialog>


    <el-dialog v-model="cloudApp.show" title="选择云端应用" width="640px" modal-class="respo-cloudapp-dialog">
      <el-table v-loading="cloudApp.loading" :data="cloudApp.list" @row-click="v => cloudApp.selectId = [v.id]">
        <el-table-column label="选择应用" width="80px" align="center">
          <template #default="scope">
            <el-checkbox v-model="cloudApp.selectId" :label="scope.row.id" :value="scope.row.id"
              class="selectcloudapplabel"></el-checkbox>
          </template>
        </el-table-column>
        <el-table-column label="应用">
          <template #default="scope">
            <div class="df ai-c">
              <img :src="scope.row.cdn_logo" style="width:20px;height:20px;border-radius:4px;" class="icon" />
              <span class="ml-10 fs-14 one-hide">{{ scope.row.title || scope.row.name }}</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <div class="mt-10 df jc-e">
        <el-pagination background layout="prev, pager, next" v-model:current-page="cloudApp.page"
          :page-count="cloudApp.last_page" @current-change='openCloudApps'></el-pagination>
      </div>
      <div class="mt-20 df jc-c">
        <el-button :loading="cloudApp.loading" type="primary" @click="cloudAppUnpack">确定</el-button>
      </div>
    </el-dialog>
  </div>

</template>

<script>
import myAxios from '@/utils';
import dfimg from '@/assets/img/dfimg.png';
import w7Identifie from "@/components/w7-identifie.vue";

export default {
  name: 'respo-index',
  data() {
    return {
      starShowIndex: -1,
      paginate: 10,
      dfimg: dfimg,
      nozpk: false,
      page: 1,
      limit: 9,
      total: 1,
      list: [],
      tags: [],
      loading: false,
      add: {
        loading: false,
        show: false,
        title: '',
      },
      ipt: {
        loading: false,
        show: false,
        link: "",
        identifie: "",
      },
      search: {
        keyword: '',
        tag: '',
      },
      allTag: false,
      is_register: false,

      cloudApp: {
        show: false,
        list: [],
        page: 1,
        selectId: [],
        last_page: 1,
      },
      utoken: '',
      webUrl: '',
    }
  },
  created() {
    this.is_register = window?.$wujie?.props?.isRegister;
    this.getData(1);
  },
  components: { w7Identifie },
  methods: {
    cloudAppUnpack() {
      if (!this.cloudApp?.selectId?.[0]) {
        this.$message.warning('请选择应用');
        return;
      }
      this.cloudApp.loading = true;
      myAxios.post('/respo/cloud-app/notapp/unpack', {
        id: this.cloudApp.selectId[0]
      }).then(res => {
        this.$message.success('操作成功');
        this.cloudApp.show = false;
      }).catch((error) => {
        if (!error?.config?.dontalert) {
          if (error.response && !error.config.headers.cancelerror && error.response?.data?.error) {
            this.$message({
              message: error.response.data.error,
              duration: 3000,
              type: 'error',
            });
          }
        }
      }).finally(() => {
        this.cloudApp.loading = false;
        this.getData();
      })
    },
    openCloudApps() {
      this.cloudApp.list = [];
      this.cloudApp.show = true;
      this.cloudApp.loading = true;

      myAxios.post('/respo/cloud-app/notapp/list', {
        page: this.cloudApp.page,
        per_page: 10,
      }).then(res => {
        let list = res?.data?.data?.data || [];
        this.cloudApp = {
          ...this.cloudApp,
          show: true,
          loading: false,
          last_page: res?.data?.data?.last_page || 1,
          list: list,
          selectId: [],
        }
      }).catch(() => {
        this.cloudApp.loading = false;
      })
    },
    installEvent(data) {
      let url = this.webUrl + '/respo/info/' + data.identifie;
      if (data.audit_status !== 3) {
        url = url + '?utoken=' + data.utoken;
      }
      window.$wujie?.bus?.$emit?.('toStoreInstall', encodeURIComponent(url))
    },
    onekeyCopy(text, callback) {
      var createInput = document.createElement('textarea');
      createInput.value = text;
      document.body.appendChild(createInput);
      createInput.select();
      document.execCommand("Copy");
      createInput.className = 'createInput';
      createInput.style.display = 'none';
      callback && callback();
    },
    changeStatus(code, identifie) {
      myAxios.post('/respo/status', { identifie, status: code }).then(res => {
        if (res.data.code == 200) {
          this.$message.success('操作成功');
          this.getData();
        }
      });
    },
    toinstall(pure, version, identifie) {
      let url = this.webUrl + '/respo/v2/info/' + identifie + '/' + (version || '1.0.0');
      if (pure) {
        this.onekeyCopy(url, () => {
          this.$message.success({ message: '复制成功', duration: 4000 });
        });
        return;
      }
      url = 'https://console.w7.cc/api/deploy/thirdparty_cd/redirect?route=/zpk-install?path=' + encodeURIComponent(url);
      this.onekeyCopy(url, () => {
        this.$message.success({ message: '复制成功，访问此链接或是转发他人完成安装操作。', duration: 4000 });
      });
    },
    deleteItem(row) {
      this.$confirm('确定要删除该项吗', "提示", {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
      }).then(() => {
        myAxios.post('/respo/delete', { identifie: row.identifie }).then(res => {
          if (!res.data) { return }
          this.$message.success('删除成功');
          this.getData();
          this.$emit('delete');
        });
      });
    },
    getLogo(url) {
      let base = this.webUrl || '';
      if (url) {
        url = url + '?time=' + Date.now();
      }
      let icon = /^(https?:)?\/\//.test(url) ? url : (url ? base + url : dfimg);
      return icon
    },
    tagClick(item) {
      this.search.tag = item.name;
      this.toSearch();
    },
    toSearch() {
      this.getData(1);
    },
    getTags() {
      myAxios.post('/respo/tag/list', { limit: 999 }).then(res => {
        this.tags = res.data?.data?.list || [];
      });
    },
    iptInputlink() {
      if (!this.ipt.link) { return }
      this.ipt.loading = true;
      myAxios().get(this.ipt.link).then(res => {
        this.ipt.loading = false;
        this.ipt.identifie = this.ipt.link.replace(/^.+\/([^/]+)$/, '$1');
      }).catch(e => {
        this.$message.error("制品库请链接求失败");
        this.ipt.loading = false;
      });
    },
    iptSubmit() {
      let f = this.list.filter(i => i.identifie == this.ipt.identifie);
      if (f.length) {
        this.$message.warning('标识重复,请修改标识');
        return;
      }
      if (!/^[a-zA-Z0-9]+-[a-zA-Z0-9]+$/.test(this.ipt.identifie)) {
        this.$message.warning('标识中间必须用"-"分隔');
        return;
      }
      this.ipt.loading = true;
      myAxios.post('/respo/add', { identifie: this.ipt.identifie }).then(res => {
        let id = this.ipt.identifie;
        let link = this.ipt.link;
        this.getInfo(id, () => {
          this.ipt = { loading: false, show: true, link: "", identifie: "" };
          this.$router.push('/zpk-manifest?id=' + id + '&link=' + encodeURIComponent(link));
        });
      }).catch(() => {
        this.ipt.loading = false;
      });
    },
    getData(page) {
      if (page) {
        this.page = page;
      }
      this.loading = true;
      myAxios.get('/respo/list?status=1&status=2&status=99', {
        params: {
          page: this.page,
          limit: this.paginate,
          tag: this.search.tag,
          keyword: this.search.keyword,
          owner: true,
        },
      }).then(res => {
        this.webUrl = res.data?.data?.webUrl;
        this.list = res.data?.data?.list || [];
        this.list.map(i => {
          i.name = i.name || i.identifie;
        })
        this.total = res.data?.data?.total || 0;
      }).finally(() => {
        this.loading = false;
      });
    },
    addRespo() {
      this.$refs.form.validate((valid) => {
        if (!valid) {
          return
        }
        if (!/^[a-zA-Z0-9]+-[a-zA-Z0-9]+$/.test(this.add.title)) {
          this.$message.warning('只能使用数字和字母大小写');
          return;
        }
        this.add.loading = true;
        myAxios.post('/respo/add', { identifie: this.add.title }).then(res => {
          let id = this.add.title;
          this.getInfo(id, () => {
            this.add = { show: false, title: '', loading: false };
            this.$router.push('/zpk-version?id=' + id);
          });
        }).catch(() => {
          this.add.loading = false;
        });
      })
    },
    getInfo(id, callback, n) {
      n = n || 0;
      myAxios.get('/respo/v2/info/' + id + '/1.0.0').then(res => {
        callback && callback();
      }).catch(() => {
        if (n > 10) { return }
        setTimeout(() => {
          this.getInfo(id, callback, n + 1);
        }, 1000);
      });
    },
  }
}
</script>

<style scoped>
.w-100 {
  width: 100%;
}

.content {
  padding: 20px;
}

.labelbox {
  border: 1px solid #E7E7E7;
  border-radius: 8px;
  padding: 15px 15px 5px;
}

.labelbox .label {
  display: inline-flex;
  cursor: pointer;
  height: 30px;
  line-height: 30px;
  padding: 0 10px;
  margin: 0 16px 10px 0;
  border-radius: 2px;
  white-space: nowrap;
}

.labelbox .label:hover {
  background: #DCDCDC;
}

.labelbox .label.active {
  background: #2D62FF;
  color: #ffffff;
}

.alltag {
  display: inline-block;
  height: 30px;
  line-height: 30px;
  white-space: nowrap;
}
</style>
<style>
.table-respo-list .el-table__row .star-icon {
  display: none;
}

.table-respo-list .el-table__row .star-icon.star-show {
  display: inline-flex;
}

.table-respo-list .el-table__row:hover .star-icon {
  display: inline-flex;
}

.respo-cloudapp-dialog .el-table__row {
  cursor: pointer;
}

.respo-cloudapp-dialog .el-dialog__body {
  padding-top: 10px;
}

.respo-cloudapp-dialog .selectcloudapplabel .el-checkbox__label {
  display: none;
}
</style>