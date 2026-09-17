import jsyaml from 'js-yaml';
import emitWujieEvent from '@/utils/wujie-event';

export function applicationConfigFormDefaults() {
    return {
        containers: [],
        ingress: [],
        cmd: [''],
    };
}

export function applicationConfigManifestState() {
    return {
        domainConfig: {
            show: false,
            ingress: [],
        },
        containerPluginData: {},
        app_ports: [],
        app_names: [],
    };
}

export const applicationConfigManifestWatch = {
    'form.ingress': {
        deep: true,
        handler() {
            this.changeForm();
        },
    },
    'option.app_ports'() {
        this.computedAppPort();
    },
};

export const applicationConfigManifestMethods = {
    initApplicationConfig(platform = {}) {
        if (this.form.type == 'system-image') {
            const command = platform?.['container-v2']?.[0]?.command;
            this.form.cmd = Array.isArray(command) && command.length
                ? command.map(item => String(item))
                : [''];
        } else {
            this.form.cmd = [''];
        }
        this.form.ingress = JSON.parse(JSON.stringify(platform.ingress || []));
        this.form.containers = platform?.['container-v2'] || [];
        this.containerPluginData = {
            ...this.containerPluginData,
            runtimeClassName: platform.runtimeClassName || '',
            kind: platform?.workload?.type || '',
        };
    },
    openDomainConfig() {
        this.domainConfig = {
            show: true,
            ingress: JSON.parse(JSON.stringify(this.form.ingress || [])),
        };
    },
    resetDomainConfig() {
        this.domainConfig = {
            show: false,
            ingress: [],
        };
    },
    saveDomainConfig() {
        const ingress = JSON.parse(JSON.stringify(this.domainConfig.ingress || []));
        this.form.ingress = ingress;
        this.checkDomainStartParams(Boolean(ingress.length));
        this.changeForm();
        this.domainConfig.show = false;
    },
    formatIngressRoutes(routes = []) {
        return routes.filter(route => route.path && route.backend?.port).map(route => ({
            ...route,
            backend: {
                ...route.backend,
                port: Number(route.backend.port),
            },
        }));
    },
    openAppset() {
        const volumes = this.json?.platform?.volumes;
        const volumeClaimTemplates = this.json?.platform?.volumeClaimTemplates;
        const containers = this.json?.platform?.['container-v2']?.filter(item => !item.isInitContainer);
        const initContainers = this.json?.platform?.['container-v2']?.filter(item => item.isInitContainer);

        emitWujieEvent('containerPlugin', {
            volumes,
            volumeClaimTemplates,
            containers,
            initContainers,
            isTemplate: true,
            pluginData: this.containerPluginData,
        }, data => {
            const nextInitContainers = data?.initContainers || [];
            nextInitContainers.forEach(item => { item.isInitContainer = true; });
            const nextContainers = data?.containers || [];
            const allContainers = nextContainers.concat(nextInitContainers);
            this.form.containers = allContainers;
            this.json.platform['container-v2'] = allContainers;
            this.json.platform.volumes = data.volumes;
            this.json.platform.volumeClaimTemplates = data.volumeClaimTemplates;
            this.fillDefaultShellContainer();

            this.form.storage = Boolean(data?.volumeClaimTemplates?.length);
            this.json.platform.runtimeClassName = this.form.type == 'system-image'
                ? 'sysbox-runc'
                : (data?.pluginData?.gpu ? 'nvidia' : '');
            this.json.platform.workload = this.json.platform.workload || {};
            this.json.platform.workload.type = data?.pluginData?.kind || 'deployments';

            this.applyPlatformShells();
            if (this.form.type == 'tradition') {
                this.syncTraditionIngress();
            } else {
                this.json.platform.ingress = (this.form.ingress || []).map(item => ({
                    name: item.name,
                    routes: this.formatIngressRoutes(item.routes),
                }));
            }

            this.syncApplicationBuiltInStartParams();
            this.json.platform.startParams = this.serializeStartParams();
            this.yaml = jsyaml.dump(this.json);
            this.computedAppPort(this.json.platform?.['container-v2']);
        });
    },
    computedAppPort(containers = this.form.containers) {
        const normalizePorts = ports => {
            const values = Array.isArray(ports) ? ports : (ports ? [ports] : []);
            return [...new Set(values.map(item => {
                if (typeof item === 'object' && item !== null) {
                    return item.port ?? item.containerPort ?? '';
                }
                return item;
            }).filter(item => item !== '' && item !== undefined && item !== null).map(String))];
        };
        let appPorts = [];
        (containers || []).forEach(item => {
            appPorts = appPorts.concat(normalizePorts(item?.ports || []));
            if (item?.port || item?.containerPort) {
                appPorts = appPorts.concat(normalizePorts([item.port ?? item.containerPort]));
            }
        });
        const legacyContainer = this.json?.platform?.container;
        if (legacyContainer) {
            appPorts = appPorts.concat(normalizePorts(legacyContainer.ports || []));
            if (legacyContainer.port || legacyContainer.containerPort) {
                appPorts = appPorts.concat(normalizePorts([
                    legacyContainer.port ?? legacyContainer.containerPort,
                ]));
            }
        }
        appPorts = [...new Set(appPorts.concat(normalizePorts(this.json?.platform?.port || [])))];
        const ports = { [this.identifie]: appPorts };

        (this.option?.app_ports || []).forEach(item => {
            if (item.name != this.identifie) {
                ports[item.name] = normalizePorts(item.port || item.ports || []);
            }
        });
        this.app_ports = ports;

        const names = [{
            id: this.identifie,
            name: this.identifie,
            title: this.form.name || this.identifie,
        }];
        this.app_names = names.concat((this.option?.app_ports || [])
            .filter(item => item.name != this.identifie)
            .map(item => ({
                id: item.name,
                name: item.name,
                title: item.title || item.name,
            })));
    },
    validateSysboxContainerName() {
        if (this.form.type == 'system-image') {
            const containerName = this.json?.platform?.['container-v2']?.[0]?.name;
            return String(containerName || '').trim()
                ? ''
                : '系统镜像主容器名称不能为空';
        }
        if (this.form.type == 'tradition' && !this.form.traditionSystemRebootRestore) {
            const containerName = this.json?.platform?.['container-v2']
                ?.find(item => !item?.isInitContainer)?.name;
            return String(containerName || '').trim()
                ? ''
                : '传统应用主容器名称不能为空';
        }
        return '';
    },
};
