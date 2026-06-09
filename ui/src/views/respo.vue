<template>
  <div style="min-height:100%;">
    <div style="height:100vh;box-sizing:border-box;">
      <div>
        <div class="bg-white bg-padding pb-24" style="border-top:1px solid #EEEEEE;">
          <div class="zpk-page-toolbar">
            <div class="zpk-toolbar-left">
              <a-button type="primary" @click="add.show = true;">
                <template #icon><icon-plus /></template>
                添加制品
              </a-button>
              <a-button v-if="is_register" @click="openCloudApps">云端找回</a-button>
            </div>
            <div class="zpk-toolbar-right">
              <a-input v-model="search.keyword" style="width:256px;" placeholder="输入制品名称搜索" @keydown.enter="getData(1)">
                <template #suffix>
                  <span class="respo-search-action" @click="getData(1)">
                    <icon-search :size="16" />
                  </span>
                </template>
              </a-input>
            </div>
          </div>
          <a-table :loading="loading" class="mt-20 table-header table-respo-list" :data="list" :pagination="false"
            row-key="identifie">
            <template #columns>
              <a-table-column data-index="name" title="制品名称">
                <template #cell="{ record }">
                  <div class="respo-name-cell">
                    <img :src="getLogo(record.icon)" class="respo-name-icon"
                      @error="(e) => { e.target.src = dfimg; }" alt="">
                    <div class="respo-name-main">
                      <div class="respo-name-line">
                        <span v-if="record.remote_formula_info_url" class="c-99">{{ record.name }}</span>
                        <span v-else class="c-blue cursor"
                          @click="$router.push('/zpk-version?id=' + record.identifie + '&title=' + record.name)">{{
                            record.name }}</span>
                        <span v-if="record.goods_id" style="color: red;margin: 0 4px;">[付费]</span>
                        <span :class="{ 'star-show': record.status === 99 }" class="respo-star-icon"
                          @click="changeStatus(record.status === 99 ? 2 : 99, record.identifie)">
                          {{ record.status === 99 ? '★' : '☆' }}
                        </span>
                      </div>
                      <div class="respo-url-line">
                        <span class="respo-url-text">{{ getArtifactInfoUrl(record) }}</span>
                        <a-tooltip content="复制地址" position="top">
                          <a-button type="text" size="mini" @click.stop="copyArtifactInfoUrl(record)">
                            <template #icon><icon-copy /></template>
                          </a-button>
                        </a-tooltip>
                      </div>
                    </div>
                  </div>
                </template>
              </a-table-column>
              <a-table-column data-index="identifie" title="标识" />
              <a-table-column title="版本">
                <template #cell="{ record }">
                  {{ record.version?.name }}
                </template>
              </a-table-column>
              <a-table-column data-index="audit_status" title="审核状态">
                <template #cell="{ record }">
                  <span v-if="getAuditStatus(record) === 0" class="c-99">-</span>
                  <span v-else-if="getAuditStatus(record) === 2" class="c-99">待审核</span>
                  <a-tooltip v-else-if="getAuditStatus(record) === 3 && record.audit_remark"
                    :content="record.audit_remark" position="top">
                    <span class="c-red respo-audit-fail">不通过 <icon-exclamation-circle-fill class="respo-audit-fail-icon" /></span>
                  </a-tooltip>
                  <span v-else-if="getAuditStatus(record) === 4" class="c-green">通过</span>
                  <span v-else class="c-99">-</span>
                </template>
              </a-table-column>
              <a-table-column title="操作">
                <template #cell="{ record }">
                  <a-button type="text" @click="installEvent(record)">安装</a-button>

                  <a-popconfirm content="确认要删除吗" type="warning" ok-text="确定" cancel-text="取消"
                    content-class="zpk-delete-popconfirm"
                    :ok-button-props="{ status: 'danger' }" :cancel-button-props="{ type: 'secondary' }"
                    @ok="deleteItem(record)">
                    <a-button type="text">删除</a-button>
                  </a-popconfirm>
                  <a-button type="text" @click="changeStatus(2, record.identifie)"
                    v-if="record.status === 1">上架</a-button>
                  <a-button type="text" @click="changeStatus(1, record.identifie)" v-else>下架</a-button>
                </template>
              </a-table-column>
            </template>
          </a-table>
          <div class="mt-20 df jc-e">
            <a-pagination v-model:current="page" v-model:page-size="paginate" :total="total"
              :page-size-options="[10, 20, 30, 40]" show-page-size @page-size-change="getData(1)"
              @change='getData' />
          </div>

        </div>
      </div>
    </div>
    <a-modal v-model:visible="add.show" title="添加制品" :width="540" :footer="false">
      <a-spin :loading="add.loading">
      <a-form ref="form" :model="add" label-align="left" :label-col-props="{ span: 4, flex: '0 0 60px' }"
        :wrapper-col-props="{ span: 20, flex: '1' }" class="respo-form respo-add-form">
        <a-form-item label="标识" field="title" :rules="[{ required: true, message: '标识不能为空', trigger: 'manual' }]">
          <w7-identifie v-model="add.title" />
        </a-form-item>
      </a-form>
      <div class="dialog-footer">
        <a-button @click="add.show = false">取消</a-button>
        <a-button type="primary" @click="addRespo">确定</a-button>
      </div>
      </a-spin>
    </a-modal>

    <a-modal v-model:visible="cloudApp.show" title="选择云端应用" :width="640" :footer="false">
      <a-table :loading="cloudApp.loading" :data="cloudApp.list" :pagination="false" row-key="id"
        class="respo-cloudapp-table" @row-click="v => cloudApp.selectId = [v.id]">
        <template #columns>
          <a-table-column title="选择应用" :width="80" align="center">
            <template #cell="{ record }">
              <a-checkbox :model-value="cloudApp.selectId[0] === record.id"
                @change="() => cloudApp.selectId = [record.id]" />
            </template>
          </a-table-column>
          <a-table-column title="应用">
            <template #cell="{ record }">
              <div class="df ai-c">
                <img :src="record.cdn_logo" style="width:20px;height:20px;border-radius:4px;" class="icon" />
                <span class="ml-10 fs-14 one-hide">{{ record.title || record.name }}</span>
              </div>
            </template>
          </a-table-column>
        </template>
      </a-table>
      <div class="mt-10 df jc-e">
        <a-pagination v-model:current="cloudApp.page" :page-size="10" :total="cloudApp.last_page * 10"
          @change='openCloudApps' />
      </div>
      <div class="dialog-footer">
        <a-button @click="cloudApp.show = false">取消</a-button>
        <a-button :loading="cloudApp.loading" type="primary" @click="cloudAppUnpack">确定</a-button>
      </div>
    </a-modal>
  </div>

</template>

<script>
import myAxios from '@/utils';
import dfimg from '@/assets/img/dfimg.png';
import w7Identifie from "@/components/w7-identifie.vue";
import { messageError, messageSuccess, messageWarning } from '@/utils/ui-feedback';
import { IconPlus, IconSearch, IconCopy, IconExclamationCircleFill } from '@arco-design/web-vue/es/icon';

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
  components: { w7Identifie, IconCopy, IconExclamationCircleFill, IconPlus, IconSearch },
  methods: {
    cloudAppUnpack() {
      if (!this.cloudApp?.selectId?.[0]) {
        messageWarning('请选择应用');
        return;
      }
      this.cloudApp.loading = true;
      myAxios.post('/respo/cloud-app/notapp/unpack', {
        id: this.cloudApp.selectId[0]
      }).then(res => {
        messageSuccess('操作成功');
        this.cloudApp.show = false;
      }).catch((error) => {
        if (!error?.config?.dontalert) {
          if (error.response && !error.config.headers.cancelerror && error.response?.data?.error) {
            messageError(error.response.data.error);
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
          messageSuccess('操作成功');
          this.getData();
        }
      });
    },
    getArtifactInfoUrl(record) {
      return 'https://zpk.w7.cc/zpk/respo/info/' + record.identifie;
    },
    copyArtifactInfoUrl(record) {
      this.onekeyCopy(this.getArtifactInfoUrl(record), () => {
        messageSuccess('复制成功', { duration: 4000 });
      });
    },
    getAuditStatus(record) {
      return Number(record?.audit_status || 0);
    },
    deleteItem(row) {
      return myAxios.post('/respo/delete', { identifie: row.identifie }).then(res => {
        if (!res.data) { return }
        messageSuccess('删除成功');
        this.getData();
        this.$emit('delete');
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
      this.$refs.form.validate((errors) => {
        if (errors) {
          return
        }
        if (!/^[a-zA-Z0-9]+-[a-zA-Z0-9]+$/.test(this.add.title)) {
          messageWarning('只能使用数字和字母大小写');
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

.respo-search-action {
  color: #86909c;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
  transition: color .2s;
}

.respo-search-action:hover {
  color: #4e5969;
}

.table-header :deep(.arco-table-th) {
  background: #f2f3f5;
  color: var(--color-text-1);
  font-weight: 400;
}

.table-respo-list .respo-star-icon {
  color: rgb(248, 85, 91);
  cursor: pointer;
  display: inline-flex;
  font-size: 16px;
  line-height: 1;
  visibility: hidden;
}

.table-respo-list .respo-star-icon.star-show {
  visibility: visible;
}

.table-respo-list .respo-name-cell {
  align-items: flex-start;
  display: flex;
  gap: 10px;
  min-width: 0;
}

.table-respo-list .respo-name-icon {
  border-radius: 6px;
  flex: 0 0 auto;
  height: 42px;
  object-fit: cover;
  width: 42px;
}

.table-respo-list .respo-name-main {
  min-width: 0;
}

.table-respo-list .respo-name-line {
  align-items: center;
  display: flex;
  gap: 4px;
  line-height: 20px;
  min-width: 0;
}

.table-respo-list .respo-url-line {
  align-items: center;
  display: flex;
  line-height: 20px;
  margin-top: 4px;
  min-width: 0;
}

.table-respo-list .respo-url-text {
  color: #86909c;
  font-size: 12px;
  max-width: 360px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.table-respo-list .respo-audit-fail {
  align-items: center;
  display: inline-flex;
  gap: 4px;
}

.table-respo-list .respo-audit-fail-icon {
  color: #D00805;
  font-size: 14px;
  line-height: 1;
}

.table-respo-list :deep(.arco-table-tr:hover) .respo-star-icon {
  visibility: visible;
}

.respo-cloudapp-table :deep(.arco-table-tr) {
  cursor: pointer;
}

.respo-form :deep(.arco-form-item-label) {
  white-space: nowrap;
}

.respo-form :deep(.arco-form-item-wrapper-col) {
  min-width: 0;
}
</style>
