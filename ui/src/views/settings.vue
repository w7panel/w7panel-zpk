<template>
    <div class="pd-20">
        <el-tabs v-model="activeTab" class="demo-tabs">
            <el-tab-pane label="数据库设置" name="database">
                <database-page></database-page>
            </el-tab-pane>
            <el-tab-pane label="存储设置" name="storage">
                <s3-page></s3-page>
            </el-tab-pane>
        </el-tabs>
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
