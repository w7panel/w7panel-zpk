export function shellFormDefaults() {
    return {
        shell: [],
    };
}

export function shellManifestState() {
    return {
        shellConfig: {
            show: false,
            editIndex: -1,
            item: null,
        },
    };
}

export const shellManifestComputed = {
    shellTypeOptions() {
        return [
            { label: '安装前执行', value: 'requireinstall' },
            { label: '安装后执行', value: 'install' },
            { label: '升级前执行', value: 'pre-upgrade' },
            { label: '升级后执行', value: 'upgrade' },
            { label: '卸载后执行', value: 'uninstall' },
            { label: '手动触发', value: 'custom' },
        ];
    },
    shellContainerOptions() {
        const containers = this.form.containers?.length
            ? this.form.containers
            : (this.json?.platform?.['container-v2'] || []);
        return (containers || [])
            .filter(item => item && !item.isInitContainer && item.name)
            .map(item => ({
                label: item.name,
                value: item.name,
            }));
    },
    hasShellContainerOptions() {
        return this.form.type != 'app-plugin' && this.shellContainerOptions.length > 0;
    },
    defaultShellContainer() {
        return this.hasShellContainerOptions ? this.shellContainerOptions[0].value : '';
    },
};

export const shellManifestWatch = {
    'form.shell': {
        deep: true,
        handler() {
            this.changeForm();
        },
    },
};

export const shellManifestMethods = {
    initShellManifest(platform = {}) {
        this.form.shell = JSON.parse(JSON.stringify(
            platform.shells || platform?.['container-v2']?.[0]?.shells || [],
        ));
        this.fillDefaultShellContainer();
    },
    addShellTask() {
        this.form.shell.push({ title: '', type: '', container: this.defaultShellContainer, shell: '' });
        this.changeForm();
    },
    openShellConfig(record, index) {
        if (!record) return;
        const item = { ...record };
        if (item.container === undefined) item.container = '';
        if (this.form.type != 'app-plugin' && !item.container && this.defaultShellContainer) {
            item.container = this.defaultShellContainer;
        }
        if (item.shell === undefined) item.shell = '';
        this.shellConfig = {
            show: true,
            editIndex: index,
            item,
        };
    },
    getShellTaskStatus(record) {
        if (!record?.shell) return '未配置';
        if (this.form.type == 'app-plugin') return '所选传统应用容器';
        return record.container || this.defaultShellContainer || '默认执行容器';
    },
    fillDefaultShellContainer() {
        if (this.form.type == 'app-plugin' || !this.defaultShellContainer) return;
        (this.form.shell || []).forEach(item => {
            if (item && !item.container) item.container = this.defaultShellContainer;
        });
    },
    resetShellConfig() {
        this.shellConfig = {
            show: false,
            editIndex: -1,
            item: null,
        };
    },
    saveShellConfig() {
        if (this.shellConfig.editIndex >= 0 && this.shellConfig.item) {
            this.form.shell[this.shellConfig.editIndex] = {
                ...(this.form.shell[this.shellConfig.editIndex] || {}),
                ...this.shellConfig.item,
            };
        }
        this.resetShellConfig();
        this.changeForm();
    },
    getValidShells() {
        return (this.form.shell || []).filter(item => item.type && item.shell);
    },
    applyPlatformShells() {
        this.json.platform = this.json.platform || {};
        const shells = this.getValidShells();
        if (!shells.length) {
            delete this.json.platform.shells;
            return;
        }
        this.json.platform.shells = shells;
        if (this.json.platform?.['container-v2']?.[0]) {
            delete this.json.platform['container-v2'][0].shells;
        }
    },
};
