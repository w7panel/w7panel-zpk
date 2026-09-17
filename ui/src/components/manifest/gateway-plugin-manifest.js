import { messageWarning } from '@/utils/ui-feedback';

export const gatewayPluginAnnotationPrefix = 'w7.cc/plugin-';

export const gatewayPluginCategoryOptions = Object.freeze([
    { label: '路由', value: 'route' },
    { label: 'AI', value: 'ai' },
    { label: '认证', value: 'auth' },
    { label: '安全', value: 'security' },
    { label: '流量', value: 'traffic' },
    { label: '转换', value: 'transform' },
    { label: '可观测性', value: 'o11y' },
    { label: '自定义', value: 'custom' },
]);

export function gatewayPluginFormDefaults() {
    return {
        gatewayPluginCategory: 'custom',
        gatewayPluginDriver: 'higress-wasm/v1',
        gatewayPluginUrl: '',
        gatewayPluginPhase: 'UNSPECIFIED_PHASE',
        gatewayPluginPriority: 0,
        gatewayPluginSupportGlobal: true,
        gatewayPluginSupportRule: false,
        gatewayPluginDefaultEnabled: true,
        gatewayPluginDefaultConfig: '{}',
    };
}

export function gatewayPluginManifestState() {
    return {
        gatewayPluginCategoryOptions,
    };
}

export function gatewayPluginValidationRules() {
    return {
        gatewayPluginUrl: [
            { required: true, message: '请输入插件镜像地址', trigger: 'blur' },
        ],
        gatewayPluginCategory: [
            { required: true, message: '请选择插件分类', trigger: 'change' },
        ],
        gatewayPluginDriver: [
            { required: true, message: '请选择运行时驱动', trigger: 'change' },
        ],
        gatewayPluginPhase: [
            { required: true, message: '请选择执行阶段', trigger: 'change' },
        ],
        gatewayPluginPriority: [
            { required: true, message: '请输入优先级', trigger: 'change' },
        ],
    };
}

export function gatewayPluginFormValues(gatewayPlugin = {}) {
    const runtime = gatewayPlugin.runtime || {};
    const runtimeConfig = runtime.config || {};
    const supports = gatewayPlugin.supports || {};
    return {
        gatewayPluginCategory: gatewayPlugin.category || 'custom',
        gatewayPluginDriver: runtime.driver || 'higress-wasm/v1',
        gatewayPluginUrl: runtimeConfig.url || '',
        gatewayPluginPhase: runtimeConfig.phase || 'UNSPECIFIED_PHASE',
        gatewayPluginPriority: Number(runtimeConfig.priority || 0),
        gatewayPluginSupportGlobal: supports.global !== false,
        gatewayPluginSupportRule: supports.rule === true,
        gatewayPluginDefaultEnabled: gatewayPlugin.defaultEnabled !== false,
        gatewayPluginDefaultConfig: JSON.stringify(gatewayPlugin.defaultConfig || {}, null, 2),
    };
}

export function gatewayPluginPlatform(form, currentGatewayPlugin = {}, dependencies = []) {
    let defaultConfig;
    try {
        defaultConfig = JSON.parse(form.gatewayPluginDefaultConfig || '{}');
        if (defaultConfig === null || typeof defaultConfig !== 'object' || Array.isArray(defaultConfig)) {
            defaultConfig = currentGatewayPlugin.defaultConfig || {};
        }
    } catch {
        defaultConfig = currentGatewayPlugin.defaultConfig || {};
    }
    const configSchema = currentGatewayPlugin.configSchema || {};
    return {
        gatewayPlugin: {
            category: form.gatewayPluginCategory || 'custom',
            defaultEnabled: Boolean(form.gatewayPluginDefaultEnabled),
            supports: {
                global: Boolean(form.gatewayPluginSupportGlobal),
                rule: Boolean(form.gatewayPluginSupportRule),
            },
            defaultConfig,
            ...(Object.keys(configSchema).length ? { configSchema } : {}),
            runtime: {
                driver: form.gatewayPluginDriver || 'higress-wasm/v1',
                config: {
                    url: form.gatewayPluginUrl,
                    phase: form.gatewayPluginPhase || 'UNSPECIFIED_PHASE',
                    priority: Number(form.gatewayPluginPriority || 0),
                },
            },
        },
        depends: dependencies,
    };
}

export const gatewayPluginManifestMethods = {
    initGatewayPluginManifest(platform = {}) {
        Object.assign(this.form, gatewayPluginFormValues(platform.gatewayPlugin || {}));
    },
    applyGatewayPluginManifest() {
        if (this.form.type == 'gateway-plugin') {
            this.json.platform = gatewayPluginPlatform(
                this.form,
                this.json.platform?.gatewayPlugin || {},
                this.form.dependsIn.concat(this.form.depends),
            );
            delete this.json.source;
        } else if (this.json.platform) {
            delete this.json.platform.gatewayPlugin;
        }
    },
    filterGatewayPluginAnnotations(annotation = {}, type = this.form.type) {
        const filtered = { ...(annotation || {}) };
        Object.keys(filtered)
            .filter(key => type != 'gateway-plugin' && key.startsWith(gatewayPluginAnnotationPrefix))
            .forEach(key => delete filtered[key]);
        return filtered;
    },
    cleanupGatewayPluginType(previousType, nextType) {
        if (previousType == 'gateway-plugin' && nextType != 'gateway-plugin') {
            delete this.json.platform.gatewayPlugin;
        }
    },
    parseGatewayPluginDefaultConfig(showMessage = false) {
        try {
            const config = JSON.parse(this.form.gatewayPluginDefaultConfig || '{}');
            if (config === null || typeof config !== 'object' || Array.isArray(config)) {
                throw new Error('默认配置必须是 JSON 对象');
            }
            return config;
        } catch (error) {
            if (showMessage) messageWarning(error?.message || '默认配置 JSON 格式错误');
            return null;
        }
    },
};
