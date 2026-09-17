import jsyaml from 'js-yaml';
import {
    isTraditionBuiltInStartParamName,
} from '@/components/manifest/tradition-manifest';
import {
    isPluginTraditionStartParamName,
    withPluginTraditionStartParams,
} from '@/components/manifest/plugin-app-manifest';
import {
    isSystemImageBuiltInStartParamName,
} from '@/components/manifest/system-image-manifest';

const isDerivedDependencyReleaseStartParam = item => Boolean(
    item?.hidden && /_RELEASE_NAME$/.test(item?.name || ''),
);

const cloneManifestValue = value => JSON.parse(JSON.stringify(value));

export function startParamsFormDefaults() {
    return {
        startParams: [],
        mysql8: false,
        mysql5: false,
        redis: false,
        mongodb6: false,
        storage: false,
        domain: false,
    };
}

export function startParamsManifestState() {
    return {
        startParamsOnlySource: null,
        initializingStartParamsOnly: false,
        spEdit: {
            show: false,
            values: '',
        },
        spDesc: {
            show: false,
            item: null,
            value: '',
        },
        disabledDomainStartParams: false,
    };
}

export const startParamsManifestWatch = {
    'form.storage'(value) {
        const params = this.form.type == 'tradition'
            ? []
            : [
                { mark: 'storage', name: 'global.cluster.storageRWmode', title: '读写模式', required: true, values_text: '%STORAGE_RW_MODE%', module_name: '' },
                { mark: 'storage', name: 'global.cluster.storageSize', title: '存储大小', required: true, values_text: '%STORAGE_SIZE%', module_name: '' },
                { mark: 'storage', name: 'global.cluster.storageClassName', title: '存储类', required: true, values_text: '%STORAGE_CLASS_NAME%', module_name: '' },
            ];
        this.checkStartParams(value, 'storage', params);
    },
    'form.mysql8'(value) {
        if (value && this.form.mysql5) this.form.mysql5 = false;
        this.checkStartParams(value, 'mysql8', [
            { mark: 'mysql8', name: 'MYSQL_DATABASE', title: 'mysql数据库', required: true, values_text: 'dbname_%RANDOM%', module_name: 'w7_mysql.DB_NAME' },
            { mark: 'mysql8', name: 'MYSQL_PASSWORD', title: 'mysql密码', required: true, values_text: '%MYSQL_ROOT_PASSWORD%', module_name: 'w7_mysql' },
            { mark: 'mysql8', name: 'MYSQL_USERNAME', title: 'mysql用户名', required: true, values_text: '%MYSQL_ROOT_USERNAME%', module_name: 'w7_mysql' },
            { mark: 'mysql8', name: 'MYSQL_PORT', title: 'mysql端口', required: true, values_text: '%PORT%', module_name: 'w7_mysql' },
            { mark: 'mysql8', name: 'MYSQL_HOST', title: 'mysql地址', required: true, values_text: '%HOST%', module_name: 'w7_mysql' },
        ]);
    },
    'form.mysql5'(value) {
        if (value && this.form.mysql8) this.form.mysql8 = false;
        this.checkStartParams(value, 'mysql5', [
            { mark: 'mysql5', name: 'MYSQL_DATABASE', title: 'mysql数据库', required: true, values_text: 'dbname_%RANDOM%', module_name: 'w7_mysql.DB_NAME' },
            { mark: 'mysql5', name: 'MYSQL_PASSWORD', title: 'mysql密码', required: true, values_text: '%MYSQL_ROOT_PASSWORD%', module_name: 'w7_mysql5' },
            { mark: 'mysql5', name: 'MYSQL_USERNAME', title: 'mysql用户名', required: true, values_text: '%MYSQL_ROOT_USERNAME%', module_name: 'w7_mysql5' },
            { mark: 'mysql5', name: 'MYSQL_PORT', title: 'mysql端口', required: true, values_text: '%PORT%', module_name: 'w7_mysql5' },
            { mark: 'mysql5', name: 'MYSQL_HOST', title: 'mysql地址', required: true, values_text: '%HOST%', module_name: 'w7_mysql5' },
        ]);
    },
    'form.redis'(value) {
        this.checkStartParams(value, 'redis', [
            { mark: 'redis', name: 'REDIS_PASSWORD', title: 'redis密码', required: true, values_text: '%REDIS_PASSWORD%', module_name: 'w7_redis' },
            { mark: 'redis', name: 'REDIS_PORT', title: 'redis端口', required: true, values_text: '%PORT%', module_name: 'w7_redis' },
            { mark: 'redis', name: 'REDIS_HOST', title: 'redis地址', required: true, values_text: '%HOST%', module_name: 'w7_redis' },
        ]);
    },
    'form.mongodb6'(value) {
        this.checkStartParams(value, 'mongodb6', [
            { mark: 'mongodb6', name: 'MONGO_PORT', title: '端口', required: true, values_text: '%PORT%', module_name: 'w7_mongodb' },
            { mark: 'mongodb6', name: 'MONGO_HOST', title: '内网域名', required: true, values_text: '%HOST%', module_name: 'w7_mongodb' },
            { mark: 'mongodb6', name: 'MONGO_INITDB_ROOT_PASSWORD', title: '密码', required: true, values_text: '%MONGO_INITDB_ROOT_PASSWORD%', module_name: 'w7_mongodb' },
            { mark: 'mongodb6', name: 'MONGO_INITDB_ROOT_USERNAME', title: '用户名', required: true, values_text: '%MONGO_INITDB_ROOT_USERNAME%', module_name: 'w7_mongodb' },
        ]);
    },
    'form.domain'(value) {
        this.checkStartParams(value, 'domain', [
            { mark: 'domain', name: 'DOMAIN_URL', title: '域名', required: true, values_text: '%DOMAIN_URL%', module_name: '' },
        ]);
    },
};

export const startParamsManifestMethods = {
    beginStartParamsInitialization(manifest) {
        this.initializingStartParamsOnly = Boolean(this.option?.startParamsOnly);
        this.startParamsOnlySource = this.option?.startParamsOnly
            ? cloneManifestValue(manifest || {})
            : null;
    },
    finishStartParamsInitialization() {
        if (!this.option?.startParamsOnly) {
            this.initializingStartParamsOnly = false;
            return false;
        }
        this.json = cloneManifestValue(this.startParamsOnlySource || {});
        this.json.platform = this.json.platform || {};
        this.form.startParams = this.markStartParamsForEditor(cloneManifestValue(
            this.startParamsOnlySource?.platform?.startParams || [],
        ));
        this.json.platform.startParams = this.serializeStartParamsOnly();
        this.yaml = jsyaml.dump(this.json);
        this.$nextTick(() => {
            this.initializingStartParamsOnly = false;
        });
        return true;
    },
    submitStartParamsOnly(otherData, callback) {
        if (!this.option?.startParamsOnly) return false;
        const manifest = cloneManifestValue(this.startParamsOnlySource || {});
        manifest.platform = manifest.platform || {};
        manifest.platform.startParams = this.serializeStartParamsOnly();
        this.json = manifest;
        this.yaml = jsyaml.dump(manifest);
        this.$emit('complete', manifest, this.yaml, otherData, callback);
        return true;
    },
    syncApplicationBuiltInStartParams() {
        if (this.option?.startParamsOnly) return;
        if (this.form.type == 'app-plugin') {
            this.form.startParams = withPluginTraditionStartParams(
                this.form.startParams,
                this.form.traditionName,
            );
        } else if (this.form.type == 'tradition') {
            this.form.startParams = this.traditionStartParams();
        } else if (this.form.type == 'system-image') {
            this.form.startParams = this.systemImageStartParams();
        } else if (this.form.type == 'docker') {
            this.syncDockerBuiltInStartParams();
        }
    },
    initStartParams(platform = {}) {
        this.form.startParams = JSON.parse(JSON.stringify(platform.startParams || []))
            .filter(item => item?.mark !== 'environment-release'
                && !isDerivedDependencyReleaseStartParam(item));

        if (this.form.type == 'system-image') {
            this.form.startParams = this.systemImageStartParams();
            this.form.storage = true;
            this.ensureSystemImageContainer();
        } else if (this.form.type == 'tradition') {
            this.ensureTraditionContainerDefaults();
            this.form.startParams = this.traditionStartParams();
        }

        this.syncApplicationBuiltInStartParams();
        this.markStartParamsForEditor(this.form.startParams).forEach(item => {
            if (item.mark == 'mysql8' || item.mark == 'mysql5') this.form[item.mark] = true;
            if (item.mark == 'domain') this.form.domain = true;
            if (item.mark == 'redis') this.form.redis = true;
            if (item.mark == 'mongodb6') this.form.mongodb6 = true;
            if (item.mark == 'storage') this.form.storage = true;
        });
        if (this.form.type != 'helm' && this.form.type != 'app-plugin' && this.form.ingress?.length) {
            this.form.domain = true;
            this.disabledDomainStartParams = true;
        }
    },
    isDomainStartParam(item) {
        return item?.mark === 'domain'
            || (item?.name === 'DOMAIN_URL' && ['%DOMAIN_URL%', '%DOMAIN_SSL_URL%'].includes(item?.values_text));
    },
    isBuiltInStartParam(item) {
        const name = String(item?.name || '').trim();
        if (this.form.type == 'tradition') return isTraditionBuiltInStartParamName(name);
        if (this.form.type == 'system-image') return isSystemImageBuiltInStartParamName(name);
        if (this.form.type == 'app-plugin') return isPluginTraditionStartParamName(name);
        return this.form.type == 'docker' && this.isDockerBuiltInStartParam(name);
    },
    isFixedStartParam(item) {
        return this.isBuiltInStartParam(item);
    },
    computedSpDisabled(item) {
        return this.isFixedStartParam(item)
            || ((this.disabledDomainStartParams || this.form.type == 'app-plugin') && this.isDomainStartParam(item))
            || (this.json?.platform?.volumeClaimTemplates?.length && item.mark === 'storage');
    },
    serializeStartParams() {
        this.syncApplicationBuiltInStartParams();
        const params = (this.form.startParams || []).filter(item => item?.mark !== 'environment-release'
            && !isDerivedDependencyReleaseStartParam(item));
        const startParams = params.filter(item => item.name).map(item => ({
            type: 'text',
            name: item.name,
            title: item.title,
            required: item.required,
            values_text: item.values_text,
            module_name: item.module_name,
            description: item.description || '',
            hidden: Boolean(item.hidden),
        }));
        return this.form.type == 'app-plugin'
            ? withPluginTraditionStartParams(startParams, this.form.traditionName)
            : startParams;
    },
    serializeStartParamsOnly() {
        return (this.form.startParams || [])
            .filter(item => item?.name)
            .map(item => {
                const startParam = { ...item };
                delete startParam.mark;
                return {
                    ...startParam,
                    type: item.type || 'text',
                    description: item.description || '',
                    hidden: Boolean(item.hidden),
                };
            });
    },
    markStartParamsForEditor(params = []) {
        params.forEach((item, index) => {
            if (item.module_name == 'w7_mysql' || item.module_name == 'w7_mysql5') {
                item.mark = item.module_name == 'w7_mysql' ? 'mysql8' : 'mysql5';
                const next = params[index + 1];
                if (next?.name == 'MYSQL_DATABASE' && !next.module_name) next.mark = item.mark;
            }
            if (item.name == 'DOMAIN_URL') item.mark = 'domain';
            if (item.module_name == 'w7_redis') item.mark = 'redis';
            if (item.module_name == 'w7_mongodb') item.mark = 'mongodb6';
            if (['global.cluster.storageRWmode', 'global.cluster.storageSize', 'global.cluster.storageClassName'].includes(item.name)) {
                item.mark = 'storage';
            }
        });
        return params;
    },
    checkDomainStartParams(value) {
        this.form.domain = value;
        this.disabledDomainStartParams = Boolean(value);
    },
    openSpEdit() {
        this.spEdit.show = true;
        const values = this.form.startParams
            .filter(item => !this.isFixedStartParam(item))
            .map(item => `${item.name || ''}=${item.values_text || ''} #${item.title || ' '}:${item.description || ' '}:${item.required ? 1 : 0}:${item.module_name || ''}`);
        this.spEdit.values = values.join('\n');
    },
    submitSpEdit() {
        const fixedParams = this.form.startParams.filter(item => this.isFixedStartParam(item));
        const params = [];
        this.spEdit.values.split('\n').forEach(value => {
            const match = value.match(/^([^\s=#]+)\s*=\s*([^\s=#]+)\s*(#([^:：\s]*)\s*[:：]\s*([^:：\s]*)\s*[:：]\s*([^:：\s]*)\s*[:：]\s*([^:：\s]*))?$/);
            if (!match) return;
            const item = {
                name: match[1],
                values_text: match[2],
                title: match[4] || '',
                description: match[5] || '',
                required: match[6] == '1',
                module_name: match[7] || '',
                type: 'text',
            };
            if (!this.isFixedStartParam(item)) params.push(item);
        });
        this.form.startParams = [...fixedParams, ...params];
        this.markStartParamsForEditor(this.form.startParams).forEach(item => {
            if (item.mark == 'mysql8' || item.mark == 'mysql5') this.form[item.mark] = true;
            if (item.mark == 'redis') this.form.redis = true;
            if (item.mark == 'mongodb6') this.form.mongodb6 = true;
        });
        this.spEdit.show = false;
        this.getStart();
    },
    openSpDesc(item) {
        this.spDesc.show = true;
        this.spDesc.item = item;
        this.spDesc.value = item.description || '';
    },
    submitSpDesc() {
        if (this.spDesc.item) {
            this.spDesc.item.description = this.spDesc.value || '';
            this.getStart();
        }
        this.spDesc.show = false;
    },
    checkStartParams(check, mark, params) {
        if (this.initializingStartParamsOnly) return;
        if (check) {
            const exists = this.form.startParams.some(item => item.mark == mark);
            if (!exists) params.forEach(item => this.form.startParams.unshift(item));
        } else {
            this.form.startParams = this.form.startParams.filter(item => item.mark != mark);
        }
        this.getStart();
    },
    getStart() {
        this.json.platform.startParams = this.option?.startParamsOnly
            ? this.serializeStartParamsOnly()
            : this.serializeStartParams();
        this.setYaml();
    },
};
