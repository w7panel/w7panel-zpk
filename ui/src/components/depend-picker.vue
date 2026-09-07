<template>
    <a-modal v-model:visible="proxyVisible" :title="title" :width="840" :footer="false"
        modal-class="depend-picker-modal" @close="handleClose">
        <div class="depend-picker-toolbar">
            <a-input v-model="keyword" placeholder="搜索制品名称或标识" allow-clear
                @keydown.enter="search">
                <template #suffix>
                    <span class="depend-picker-search-action" @click="search">
                        <icon-search :size="16" />
                    </span>
                </template>
            </a-input>
        </div>
        <a-tabs v-model:active-key="activeTab" @change="changeTab">
            <a-tab-pane key="local" title="本地仓库"></a-tab-pane>
            <a-tab-pane key="official" title="官方仓库"></a-tab-pane>
        </a-tabs>
        <a-table :loading="loading" :data="rows" :pagination="false"
            row-key="identifie" class="depend-picker-table" @row-click="select">
            <template #columns>
                <a-table-column title="制品">
                    <template #cell="{ record }">
                        <div class="depend-picker-product">
                            <div class="depend-picker-product-name">{{ record.name || record.identifie }}</div>
                            <div class="depend-picker-product-identifie">{{ record.identifie }}</div>
                        </div>
                    </template>
                </a-table-column>
                <a-table-column title="版本" :width="120">
                    <template #cell="{ record }">
                        {{ record.version?.name || '-' }}
                    </template>
                </a-table-column>
                <a-table-column title="操作" :width="100" align="center">
                    <template #cell="{ record }">
                        <a-button type="text" size="small" :loading="actionLoading"
                            @click.stop="select(record)">
                            {{ actionText }}
                        </a-button>
                    </template>
                </a-table-column>
            </template>
        </a-table>
        <div class="depend-picker-footer">
            <a-pagination v-model:current="page" :page-size="pageSize"
                :total="total" @change="changePage" />
        </div>
    </a-modal>
</template>

<script>
import { IconSearch } from '@arco-design/web-vue/es/icon';
import { fetchChildImportList } from '@/utils/child-app-import';
import myAxios from '@/utils';

export default {
    name: 'DependPicker',
    components: { IconSearch },
    props: {
        visible: {
            type: Boolean,
            default: false,
        },
        title: {
            type: String,
            default: '选择安装依赖',
        },
        actionText: {
            type: String,
            default: '选择',
        },
        actionLoading: {
            type: Boolean,
            default: false,
        },
    },
    emits: ['update:visible', 'select', 'close'],
    data() {
        return {
            activeTab: 'local',
            keyword: '',
            page: 1,
            pageSize: 8,
            loading: false,
            lists: {
                local: [],
                official: [],
            },
        };
    },
    computed: {
        proxyVisible: {
            get() {
                return this.visible;
            },
            set(value) {
                this.$emit('update:visible', value);
            },
        },
        currentList() {
            return this.lists[this.activeTab] || [];
        },
        rows() {
            const start = (this.page - 1) * this.pageSize;
            return this.currentList.slice(start, start + this.pageSize);
        },
        total() {
            return this.currentList.length;
        },
    },
    watch: {
        visible(value) {
            if (value) {
                this.page = 1;
                this.load();
            } else {
                this.keyword = '';
                this.page = 1;
            }
        },
    },
    methods: {
        load() {
            this.loading = true;
            return fetchChildImportList(myAxios, this.activeTab, {
                page: 1,
                limit: 999,
                keyword: this.keyword,
            }).then(list => {
                this.lists[this.activeTab] = list;
            }).catch(() => {
                this.lists[this.activeTab] = [];
            }).finally(() => {
                this.loading = false;
            });
        },
        search() {
            this.page = 1;
            this.load();
        },
        changeTab() {
            this.page = 1;
            this.load();
        },
        changePage(page) {
            this.page = page;
        },
        select(record) {
            if (!record?.identifie || this.actionLoading) { return; }
            this.$emit('select', record, this.activeTab);
        },
        handleClose() {
            this.$emit('close');
        },
    },
};
</script>

<style scoped>
.depend-picker-toolbar {
    width: 320px;
    margin-bottom: 12px;
}

.depend-picker-search-action {
    display: inline-flex;
    align-items: center;
    color: #666;
    cursor: pointer;
}

.depend-picker-table {
    margin-top: 8px;
}

.depend-picker-product {
    min-width: 0;
    cursor: pointer;
}

.depend-picker-product-name,
.depend-picker-product-identifie {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.depend-picker-product-name {
    color: #333;
}

.depend-picker-product-identifie {
    margin-top: 2px;
    color: #999;
    font-size: 12px;
}

.depend-picker-footer {
    display: flex;
    justify-content: flex-end;
    margin-top: 16px;
}
</style>
