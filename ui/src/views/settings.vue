<template>
    <div class="pd-20">
        <a-tabs v-model:active-key="activeTab" class="demo-tabs">
            <a-tab-pane key="database" title="数据库设置">
                <database-page></database-page>
            </a-tab-pane>
            <a-tab-pane key="storage" title="存储设置">
                <s3-page></s3-page>
            </a-tab-pane>
        </a-tabs>
    </div>
</template>

<script>
import S3Page from '@/views/s3.vue';
import DatabasePage from '@/views/database.vue';
export default {
    name: 'SettingsPage',
    components: {
        S3Page,
        DatabasePage
    },
    data() {
        return {
            activeTab: 'database'
        };
    }
}
</script>

<style scoped lang="scss">

</style>
