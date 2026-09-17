import { confirm, messageSuccess } from '@/utils/ui-feedback';

export const traditionToolDependency = Object.freeze({
    identifie: 'w7-traditiontool',
    name: 'w7-traditiontool',
    source: 'https://zpk.fan.b2.sz.w7.com',
});

export const traditionStorageVolumeName = 'site-storage';
export const traditionToolIdentifie = traditionToolDependency.identifie;
export const traditionSystemRebootRestoreAnnotation = 'w7.cc/system-reboot-restore';
export const traditionSysboxRootfsAnnotation = 'sysbox/rootfs-rw-layer';
export const traditionSysboxRuntimeClassName = 'sysbox-runc-lite';

const traditionGatewayStartParamName = 'gatewayEnabled';

const traditionBuiltInStartParamNames = Object.freeze([
    'IMAGE_VERSION',
    'DOMAIN_URL',
    traditionGatewayStartParamName,
    'PVC_NAME',
    'global.cluster.storageRWmode',
    'global.cluster.storageSize',
    'global.cluster.storageClassName',
]);

export function isTraditionBuiltInStartParamName(name) {
    const normalizedName = String(name || '').trim().toUpperCase();
    return traditionBuiltInStartParamNames
        .some(item => item.toUpperCase() === normalizedName);
}

export function withTraditionStartParams(
    startParams = [],
    versions = [],
    gatewayEnabled = false,
) {
    const customParams = (startParams || [])
        .filter(item => !isTraditionBuiltInStartParamName(item?.name));
    return [
        { mark: 'tradition', name: 'IMAGE_VERSION', title: '传统应用版本', required: true, values_text: (versions || []).join('|'), module_name: '', description: '选择要安装的传统应用版本', type: 'select' },
        { mark: 'environment-site', name: 'DOMAIN_URL', title: '站点域名', required: true, values_text: '%DOMAIN_URL%', module_name: '', description: '用于站点访问', type: 'text' },
        { mark: 'tradition-gateway', name: traditionGatewayStartParamName, title: '开启网关工具', required: true, values_text: String(Boolean(gatewayEnabled)), module_name: '', description: '启用传统应用网关工具', type: 'text', hidden: true },
        { name: 'PVC_NAME', title: '存储', required: true, values_text: '%PVC_NAME%', module_name: '', description: '安装时选择的 PVC 名称', type: 'text', hidden: false },
        ...customParams,
    ];
}

export function isTraditionAppDependency(record) {
    return record?.identifie == traditionToolDependency.identifie;
}

export function removeTraditionAppCodeStorage(json) {
    const platform = json?.platform;
    if (!platform) return;
    platform.volumes = (platform.volumes || [])
        .filter(item => item?.name != traditionStorageVolumeName);
    (platform['container-v2'] || []).forEach(container => {
        container.volumeMounts = (container.volumeMounts || [])
            .filter(item => item?.name != traditionStorageVolumeName);
    });
}

export function traditionAppRootfsAnnotation(
    applicationIdentifie = '',
    containerName = '',
) {
    const identifie = String(applicationIdentifie || '').trim()
        .replace(/[^A-Za-z0-9._-]+/g, '-')
        .replace(/^-+|-+$/g, '');
    const name = String(containerName || '').trim().replaceAll('_', '-');
    if (!identifie || !name) return '';
    const version = '${IMAGE_VERSION}';
    return JSON.stringify([{
        name,
        volumeName: traditionStorageVolumeName,
        path: `www/server/${identifie}-${version}/system`,
        persistentSpecialMounts: true,
    }]);
}

export function withTraditionAppSysbox(
    platform = {},
    sourceAnnotations = {},
    restoreOverride,
    applicationIdentifie = '',
) {
    const nextPlatform = { ...(platform || {}) };
    const annotations = { ...(sourceAnnotations || {}) };
    let restore = restoreOverride;
    if (restore === undefined) {
        restore = true;
        const value = annotations[traditionSystemRebootRestoreAnnotation];
        if (value !== undefined) {
            restore = typeof value === 'boolean'
                ? value
                : String(value).trim().toLowerCase() !== 'false';
        }
    }

    delete annotations[traditionSysboxRootfsAnnotation];
    delete nextPlatform.hostUsers;
    if (restore) {
        if (nextPlatform.runtimeClassName === traditionSysboxRuntimeClassName) {
            delete nextPlatform.runtimeClassName;
        }
        return { platform: nextPlatform, annotations };
    }

    nextPlatform.runtimeClassName = traditionSysboxRuntimeClassName;
    const containers = nextPlatform['container-v2'] || [];
    const container = containers.find(item => !item?.isInitContainer);
    const rootfs = traditionAppRootfsAnnotation(applicationIdentifie, container?.name);
    if (rootfs) {
        annotations[traditionSysboxRootfsAnnotation] = rootfs;
    }
    return { platform: nextPlatform, annotations };
}

export function withTraditionAppStorage(platform = {}) {
    const nextPlatform = { ...(platform || {}) };
    const containers = Array.isArray(nextPlatform['container-v2'])
        ? nextPlatform['container-v2'].map(container => ({
            ...container,
            volumeMounts: [...(container?.volumeMounts || [])],
        }))
        : [];
    const volumes = (nextPlatform.volumes || [])
        .filter(item => item?.name !== traditionStorageVolumeName);
    volumes.push({
        name: traditionStorageVolumeName,
        persistentVolumeClaim: { claimName: '' },
    });
    nextPlatform.volumes = volumes;

    const container = containers.find(item => !item?.isInitContainer);
    if (container) {
        container.volumeMounts = container.volumeMounts
            .filter(item => item?.name !== traditionStorageVolumeName);
        container.volumeMounts.push({
            name: traditionStorageVolumeName,
            mountPath: '/www/wwwroot',
            subPath: 'nginx-web-dir',
        }, {
            name: traditionStorageVolumeName,
            mountPath: '/www/server',
            subPath: 'server-dir',
        });
    }
    nextPlatform['container-v2'] = containers;
    return nextPlatform;
}

function firstPort(value) {
    if (value === undefined || value === null || value === '') return 0;
    if (typeof value === 'number' || typeof value === 'string') {
        const port = Number(value);
        return Number.isFinite(port) && port > 0 ? port : 0;
    }
    if (Array.isArray(value)) {
        for (const item of value) {
            const port = firstPort(item);
            if (port) return port;
        }
        return 0;
    }
    if (typeof value !== 'object') return 0;
    for (const candidate of [
        value.containerPort,
        value.port,
        value.targetPort,
        value.servicePort,
    ]) {
        const port = firstPort(candidate);
        if (port) return port;
    }
    return firstPort(value.ports);
}

export function traditionAppDefaultContainerPort(source = {}, fallback = 80) {
    const value = source?.platform || source || {};
    for (const container of value['container-v2'] || []) {
        if (container?.isInitContainer) continue;
        const port = firstPort(container);
        if (port) return port;
    }
    const legacyPort = firstPort(value.container) || firstPort(value.port);
    return legacyPort || fallback;
}

export function withTraditionAppIngress(
    platform = {},
    applicationIdentifie = '',
    nginxGateway = false,
) {
    const nextPlatform = { ...(platform || {}) };
    const containerPort = traditionAppDefaultContainerPort(nextPlatform);
    const defaultBackend = nginxGateway
        ? { name: traditionToolIdentifie, port: 80, match: 'Prefix' }
        : { name: applicationIdentifie, port: containerPort, match: 'Prefix' };
    nextPlatform.ingress = (platform?.ingress || []).map(item => ({
        ...item,
        routes: (item?.routes || []).map(route => ({
            ...route,
            backend: { ...(route?.backend || {}) },
        })),
    }));
    if (!nextPlatform.ingress.length) {
        nextPlatform.ingress = [{ name: '/', routes: [{ path: '/', backend: { ...defaultBackend } }] }];
        return nextPlatform;
    }

    nextPlatform.ingress.forEach(item => {
        if (!item.routes.length) {
            item.routes.push({ path: '/', backend: { ...defaultBackend } });
        }
        item.routes.forEach(route => {
            const backend = route.backend;
            if (nginxGateway) {
                backend.name = traditionToolIdentifie;
                backend.port = 80;
                backend.match = backend.match || 'Prefix';
                return;
            }
            backend.name = applicationIdentifie;
            backend.port = containerPort;
            backend.match = backend.match || 'Prefix';
        });
    });
    return nextPlatform;
}

export const traditionAnnotationKeys = Object.freeze({
    imageLanguage: 'w7.cc/image_language',
    imageTemplate: 'w7.cc/image_template',
    imageVersion: 'w7.cc/image_version',
    nginxVhostTemplate: 'w7.cc/nginx_vhost_template',
    nginxGateway: 'w7.cc/nginx-gateway',
    systemRebootRestore: traditionSystemRebootRestoreAnnotation,
});

const traditionLanguagePresets = Object.freeze([
    { label: 'PHP', value: 'php' },
    { label: 'Java', value: 'java' },
    { label: 'Node.js', value: 'nodejs' },
    { label: 'Python', value: 'python' },
    { label: 'Go', value: 'go' },
    { label: '.NET', value: 'dotnet' },
    { label: 'Ruby', value: 'ruby' },
    { label: 'Rust', value: 'rust' },
]);

export const nginxTemplatePlaceholders = Object.freeze([
    '{SERVER_NAME}',
    '{LOG_DIR}',
    '{ROOT_DIR}',
    '{K8S_DOMAIN}',
    '{UPSTREAM_APP_NAME}',
]);

export const nginxTemplateExample = String.raw`server {
    listen 80;
    server_name {SERVER_NAME};

    root {ROOT_DIR};
    index index.php index.html index.htm;

    access_log /dev/stdout;
    error_log /dev/stderr;

    # 静态文件处理
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
        try_files $uri =404;
    }

    # 隐藏文件保护
    location ~ /\. {
        deny all;
        access_log off;
        log_not_found off;
    }

    # PHP 代理配置
    location ~ ^/(.+\.php)(/.*)?$ {
        # 使用 FastCGI 连接到 PHP-FPM
        fastcgi_pass {K8S_DOMAIN}:9000;

        set $real_scheme $http_x_forwarded_proto;
        if ($real_scheme = "") {
            set $real_scheme $scheme;
        }

        # 关键 FastCGI 参数
        include fastcgi_params;
        fastcgi_split_path_info ^(.+\.php)(/.+)$;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        fastcgi_param SCRIPT_NAME $fastcgi_script_name;
        fastcgi_param PATH_INFO $fastcgi_path_info;
        fastcgi_param PATH_TRANSLATED $document_root$fastcgi_path_info;
        fastcgi_index index.php;

        # 必要的请求头
        fastcgi_param HTTP_X_REAL_IP $remote_addr;
        fastcgi_param HTTP_X_FORWARDED_FOR $proxy_add_x_forwarded_for;
        fastcgi_param HTTP_X_FORWARDED_PROTO $real_scheme;
        fastcgi_param REQUEST_SCHEME $real_scheme;
        fastcgi_param HTTPS $real_scheme;

        # FastCGI 超时设置
        fastcgi_connect_timeout 30s;
        fastcgi_send_timeout 60s;
        fastcgi_read_timeout 60s;
    }

    # 主路由配置（支持前端控制器）
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }
}`;

export function traditionFormDefaults() {
    return {
        traditionImageLanguage: '',
        traditionImageTemplate: '',
        traditionImageVersion: [],
        traditionSystemRebootRestore: true,
        traditionNginxGateway: false,
        traditionNginxVhostTemplate: '',
    };
}

export function traditionManifestState() {
    return {
        traditionNginxGatewayChanging: false,
        nginxTemplatePlaceholders,
        nginxTemplateExample,
        nginxTemplateExampleVisible: false,
    };
}

export function traditionValidationRules(component) {
    return {
        traditionImageLanguage: [{
            required: true,
            message: '请选择应用语言',
            trigger: 'change',
        }],
        traditionImageTemplate: [{
            required: true,
            message: '请输入镜像地址',
            trigger: 'blur',
            validator: (value, callback) => component.validateTraditionImageTemplate(value, callback),
        }],
        traditionImageVersion: [{
            required: true,
            message: '请输入传统应用版本',
            trigger: 'change',
            validator: (value, callback) => component.validateTraditionImageVersions(value, callback),
        }],
        traditionNginxVhostTemplate: [{
            required: false,
            message: '请输入 NGINX 模板',
            trigger: 'blur',
            validator: (value, callback) => {
                callback(!component.form.traditionNginxGateway || String(value || '').trim()
                    ? undefined
                    : '请输入 NGINX 模板');
            },
        }],
    };
}

export const traditionManifestComputed = {
    traditionLanguageOptions() {
        const currentLanguage = String(this.form.traditionImageLanguage || '').trim();
        if (currentLanguage && !traditionLanguagePresets.some(option => option.value == currentLanguage)) {
            return [
                { label: `${currentLanguage}（已有值）`, value: currentLanguage },
                ...traditionLanguagePresets,
            ];
        }
        return traditionLanguagePresets;
    },
    traditionIcon() {
        return icon => {
            if (!icon) return '';
            return /^https?:\/\//i.test(icon)
                ? icon
                : `https://img.w7.cc${icon.startsWith('/') ? '' : '/'}${icon}`;
        };
    },
};

export const traditionManifestMethods = {
    applyTraditionManifest() {
        if (this.form.type != 'tradition') return;
        this.form.traditionImageVersion = this.normalizeApplicationVersions(this.form.traditionImageVersion);
        this.form.startParams = this.traditionStartParams();
        this.ensureTraditionContainerDefaults();
        this.applyTraditionRebootRestoreConfig();
        this.syncTraditionIngress();
    },
    filterTraditionAnnotations(annotation = {}, type = this.form.type) {
        const filtered = { ...(annotation || {}) };
        if (type != 'tradition' && type != 'system-image') {
            Object.values(traditionAnnotationKeys).forEach(key => delete filtered[key]);
        } else if (type == 'system-image') {
            Object.values(traditionAnnotationKeys)
                .filter(key => key != traditionAnnotationKeys.imageVersion)
                .forEach(key => delete filtered[key]);
        }
        if (type == 'tradition') Object.assign(filtered, this.getTraditionAnnotations());
        return filtered;
    },
    cleanupTraditionType(previousType, nextType) {
        if (previousType != 'tradition' || nextType == 'tradition') return;
        this.form.startParams = (this.form.startParams || [])
            .filter(item => !isTraditionBuiltInStartParamName(item?.name));
        removeTraditionAppCodeStorage(this.json);
        this.form.dependsIn = (this.form.dependsIn || [])
            .filter(item => !isTraditionAppDependency(item));
        this.form.depends = (this.form.depends || [])
            .filter(item => !isTraditionAppDependency(item));
    },
    useNginxTemplateExample() {
        const applyExample = () => {
            this.form.traditionNginxVhostTemplate = this.nginxTemplateExample;
            this.nginxTemplateExampleVisible = false;
            messageSuccess('已填入 NGINX 模板示例');
        };
        const currentTemplate = String(this.form.traditionNginxVhostTemplate || '').trim();
        if (!currentTemplate || currentTemplate == this.nginxTemplateExample.trim()) {
            applyExample();
            return;
        }
        confirm({
            title: '使用 NGINX 模板示例',
            content: '当前模板内容将被示例覆盖，是否继续？',
            confirmButtonText: '覆盖并使用',
            cancelButtonText: '取消',
            onOk: applyExample,
        });
    },
    toggleTraditionNginxGateway(value) {
        if (this.form.type != 'tradition' || this.option?.pureManifest
            || this.traditionNginxGatewayChanging) {
            return;
        }
        const enabled = Boolean(value);
        const previous = !enabled;
        this.form.traditionNginxGateway = previous;
        const finish = success => {
            this.traditionNginxGatewayChanging = false;
            this.form.traditionNginxGateway = success ? enabled : previous;
            this.syncTraditionGatewayStartParam();
            this.syncTraditionIngress();
        };
        const request = () => {
            this.traditionNginxGatewayChanging = true;
            this.$emit('tradition-nginx-gateway-change', { enabled, finish });
        };
        if (enabled) {
            request();
            return;
        }
        confirm({
            title: '关闭 NGINX 网关',
            content: '关闭后将停止使用 NGINX 网关，是否继续？',
            confirmButtonText: '关闭',
            cancelButtonText: '取消',
            onOk: request,
            onCancel: () => {
                this.form.traditionNginxGateway = previous;
            },
        });
    },
    validateTraditionImageTemplate(value, callback) {
        const template = String(value || '').trim();
        if (!template) { callback('请输入镜像地址'); return; }
        if (/\s/.test(template)) { callback('镜像地址不能包含空格或换行'); return; }
        if (!template.includes('{version}')) { callback('镜像地址必须包含 {version} 占位符'); return; }
        const unsupported = [...new Set(template.match(/\{[^{}]+\}/g) || [])]
            .filter(placeholder => placeholder != '{version}');
        callback(unsupported.length
            ? '镜像地址包含不支持的占位符：' + unsupported.join('、')
            : undefined);
    },
    validateTraditionImageVersions(value, callback) {
        const versions = this.normalizeApplicationVersions(value);
        if (!versions.length) { callback('请输入至少一个传统应用版本'); return; }
        const invalid = [...new Set(versions
            .filter(version => !/^[A-Za-z0-9_][A-Za-z0-9._-]{0,127}$/.test(version)))];
        let invalidText = invalid.slice(0, 3).join('、');
        if (invalid.length > 3) invalidText += ' 等';
        callback(invalid.length ? '以下语言版本格式不正确：' + invalidText : undefined);
    },
    traditionStartParams() {
        return withTraditionStartParams(
            this.form.startParams,
            this.normalizeApplicationVersions(this.form.traditionImageVersion),
            this.form.traditionNginxGateway,
        );
    },
    syncTraditionGatewayStartParam() {
        if (this.form.type != 'tradition') return;
        this.form.startParams = this.traditionStartParams();
        this.json.platform = this.json.platform || {};
        this.json.platform.startParams = this.serializeStartParams();
        this.json.application = this.json.application || {};
        this.json.application.annotation = this.json.application.annotation || {};
        this.json.application.annotation[traditionAnnotationKeys.nginxGateway] = String(
            Boolean(this.form.traditionNginxGateway),
        );
    },
    syncTraditionVersionConfig() {
        if (this.form.type != 'tradition') return;
        this.form.traditionImageVersion = this.normalizeApplicationVersions(this.form.traditionImageVersion);
        this.form.startParams = this.traditionStartParams();
        this.changeForm();
    },
    ensureTraditionContainerDefaults(forceImage = false) {
        if (this.form.type != 'tradition') return;
        this.json.platform = this.json.platform || {};
        delete this.json.platform.container;
        const containers = Array.isArray(this.json.platform['container-v2'])
            ? this.json.platform['container-v2']
            : [];
        let mainContainer = containers.find(item => !item?.isInitContainer);
        if (!mainContainer) {
            mainContainer = {
                name: (this.form.author && this.form.identifie)
                    ? `${this.form.author}-${this.form.identifie}`
                    : (this.json?.application?.identifie || 'tradition'),
                image: '',
                imagePullPolicy: 'IfNotPresent',
            };
            containers.unshift(mainContainer);
        }
        const imageTemplate = String(this.form.traditionImageTemplate || '').trim();
        if (forceImage || !String(mainContainer.image || '').trim()) {
            mainContainer.image = imageTemplate;
        }
        if (!mainContainer.imagePullPolicy) mainContainer.imagePullPolicy = 'IfNotPresent';
        this.json.platform['container-v2'] = containers;
        this.form.containers = containers;
        this.json.platform.workload = this.json.platform.workload || {};
        if (!this.json.platform.workload.type) this.json.platform.workload.type = 'Deployment';
        if (!this.containerPluginData.kind) this.containerPluginData.kind = 'Deployment';
        this.fillDefaultShellContainer();
        this.json.platform = withTraditionAppStorage(this.json.platform);
    },
    syncTraditionRuntimeConfig() {
        if (this.form.type != 'tradition') return;
        this.ensureTraditionContainerDefaults(true);
        this.applyTraditionRebootRestoreConfig();
        this.syncSysboxDependency();
        this.changeForm();
    },
    applyTraditionRebootRestoreConfig() {
        if (this.form.type != 'tradition' || !this.json?.platform) return;
        const result = withTraditionAppSysbox(
            this.json.platform,
            this.json.application?.annotation || {},
            Boolean(this.form.traditionSystemRebootRestore),
            this.json.application?.identifie || this.form.identifie,
        );
        this.json.platform = result.platform;
        this.json.application = this.json.application || {};
        this.json.application.annotation = result.annotations;
    },
    syncTraditionIngress() {
        if (this.form.type != 'tradition' || !this.json?.platform) {
            return this.json?.platform?.ingress || [];
        }
        const applicationIdentifie = this.json?.application?.identifie
            || ((this.form.author && this.form.identifie)
                ? `${this.form.author}-${this.form.identifie}`
                : this.identifie || '');
        this.json.platform = withTraditionAppIngress(
            this.json.platform,
            applicationIdentifie,
            Boolean(this.form.traditionNginxGateway),
        );
        const ingress = JSON.parse(JSON.stringify(this.json.platform.ingress || []));
        if (JSON.stringify(this.form.ingress || []) != JSON.stringify(ingress)) {
            this.form.ingress = ingress;
        }
        return ingress;
    },
    applyTraditionAnnotationForm(annotation = {}) {
        const hasGatewayAnnotation = Object.prototype.hasOwnProperty.call(
            annotation || {}, traditionAnnotationKeys.nginxGateway,
        );
        if (hasGatewayAnnotation) {
            const value = annotation[traditionAnnotationKeys.nginxGateway];
            this.form.traditionNginxGateway = value === true || String(value).toLowerCase() == 'true';
        }
        this.form.traditionImageLanguage = String(annotation[traditionAnnotationKeys.imageLanguage] || '');
        this.form.traditionImageTemplate = String(annotation[traditionAnnotationKeys.imageTemplate] || '');
        this.form.traditionImageVersion = this.normalizeApplicationVersions(annotation[traditionAnnotationKeys.imageVersion]);
        this.form.traditionNginxVhostTemplate = String(annotation[traditionAnnotationKeys.nginxVhostTemplate] || '');
        const restoreValue = annotation[traditionAnnotationKeys.systemRebootRestore];
        this.form.traditionSystemRebootRestore = restoreValue === undefined || restoreValue === null || restoreValue === ''
            ? String(this.form.traditionImageLanguage || '').toLowerCase() != 'php'
            : String(restoreValue).toLowerCase() == 'true';
        if (this.form.type == 'tradition') {
            this.ensureTraditionContainerDefaults();
            this.form.startParams = this.traditionStartParams();
            this.applyTraditionRebootRestoreConfig();
        }
    },
    changeTraditionLanguage(value) {
        this.form.traditionSystemRebootRestore = String(value || '').toLowerCase() != 'php';
    },
    getTraditionAnnotations() {
        const versions = this.normalizeApplicationVersions(this.form.traditionImageVersion);
        this.form.traditionImageVersion = versions;
        return {
            [traditionAnnotationKeys.imageLanguage]: String(this.form.traditionImageLanguage || '').trim(),
            [traditionAnnotationKeys.imageTemplate]: String(this.form.traditionImageTemplate || '').trim(),
            [traditionAnnotationKeys.imageVersion]: versions.join(','),
            [traditionAnnotationKeys.nginxVhostTemplate]: this.form.traditionNginxGateway
                ? String(this.form.traditionNginxVhostTemplate || '')
                : '',
            [traditionAnnotationKeys.nginxGateway]: String(Boolean(this.form.traditionNginxGateway)),
            [traditionAnnotationKeys.systemRebootRestore]: String(Boolean(this.form.traditionSystemRebootRestore)),
        };
    },
    syncTraditionDependency() {
        if (this.form.type != 'tradition' || this.traditionNginxGatewayChanging) return;
        const existing = [...(this.form.dependsIn || []), ...(this.form.depends || [])]
            .find(item => isTraditionAppDependency(item));
        const dependency = {
            ...(existing || {}),
            identifie: traditionToolDependency.identifie,
            name: traditionToolDependency.name,
            subidentifie: '',
            subname: '',
            required: true,
            type: 'in',
            from: traditionToolDependency.source,
        };
        this.form.depends = (this.form.depends || []).filter(item => !isTraditionAppDependency(item));
        this.form.dependsIn = (this.form.dependsIn || []).filter(item => !isTraditionAppDependency(item));
        this.form.dependsIn.push(dependency);
    },
};
