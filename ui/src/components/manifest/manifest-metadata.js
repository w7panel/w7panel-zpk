import myAxios from '@/utils/index';

export function manifestMetadataFormDefaults() {
    return {
        name: '',
        author: '',
        description: '',
        identifie: '',
    };
}

export function manifestMetadataValidationRules(component) {
    return {
        name: [
            { required: true, message: '内容不能为空', trigger: 'blur' },
        ],
        identifie: [
            { required: true, message: '内容不能为空', trigger: 'blur' },
            {
                required: true,
                trigger: 'blur',
                validator: (value, callback) => {
                    if (component.form.author) callback();
                    else callback('请输入完整');
                },
            },
            {
                required: true,
                trigger: 'blur',
                validator: (value, callback) => {
                    if (/^[a-zA-Z0-9]+$/.test(value)) callback();
                    else callback('标识格式有误');
                },
            },
            {
                required: true,
                trigger: 'blur',
                validator: (value, callback) => {
                    if (/^[a-zA-Z0-9]+$/.test(component.form.author)) callback();
                    else callback('标识格式有误');
                },
            },
        ],
    };
}

export function manifestMetadataState() {
    return {
        formulaSettingLoading: false,
        formulaSettingLoaded: false,
        formulaBaseInfo: null,
    };
}

export const manifestMetadataMethods = {
    ensureManifestBaseInfo(manifest) {
        if (this.option?.pureManifest) return;
        manifest.platform = manifest.platform || {};
        manifest.platform.baseInfo = manifest.platform.baseInfo || {};
        manifest.platform.baseInfo.identifie = manifest.platform.baseInfo.identifie
            || manifest.application?.identifie
            || '';
        manifest.platform.baseInfo.name = manifest.platform.baseInfo.name
            || manifest.application?.name
            || '';
        manifest.platform.baseInfo.description = manifest.platform.baseInfo.description
            || manifest.application?.description
            || '';
    },
    initManifestMetadata(manifest = {}) {
        if (/^[^-]+-.+$/.test(manifest.application?.identifie)) {
            manifest.application.author = manifest.application.identifie.match(/^([^-]+)-(.+)$/)[1];
        }
        this.form.once = manifest.application?.once || false;
        this.form.name = manifest.application?.name;
        if (/^[^-]+-.+$/.test(manifest.application?.identifie)) {
            this.form.identifie = manifest.application.identifie.match(/^([^-]+)-(.+)$/)[2];
        } else {
            this.form.identifie = manifest.application?.identifie;
        }
        this.form.author = manifest.application?.author;
        if (!this.form.identifie && !this.form.author && this.identifie) {
            const identifyParts = this.identifie.match(/^([^-]+)-(.+)$/);
            if (identifyParts?.length) {
                this.form.identifie = identifyParts[2];
                this.form.author = identifyParts[1];
            }
        }
        this.form.description = manifest.application?.description;

        if (!this.option?.pureManifest && this.form.type != 'gateway-plugin') {
            this.form.name = manifest.platform?.baseInfo?.name || '';
            const identifie = manifest.platform?.baseInfo?.identifie || '';
            this.form.identifie = identifie.match(/^([^-]+)-(.+)$/)?.[2] || '';
            this.form.author = identifie.match(/^([^-]+)-(.+)$/)?.[1] || '';
            this.form.description = manifest.platform?.baseInfo?.description || '';
        }
    },
    applyApplicationMetadata(application) {
        if (!application) return;
        if (this.option?.pureManifest) {
            application.name = this.form.name;
            application.identifie = `${this.form.author}-${this.form.identifie}`;
            application.description = this.form.description;
        }
        application.author = this.form.author;
        application.theme = this.form.theme;
        application.type = this.form.type;
        if (this.form.type == 'gateway-plugin') {
            this.form.once = true;
        } else if (['tradition', 'system-image'].includes(this.form.type)) {
            this.form.once = false;
        }
        application.once = ['tradition', 'system-image'].includes(this.form.type)
            ? false
            : Boolean(this.form.once);
    },
    applyPlatformBaseInfo(platform) {
        if (this.option?.pureManifest) return;
        platform.baseInfo = platform.baseInfo || {};
        platform.baseInfo.name = this.form.name;
        platform.baseInfo.identifie = this.form.author && this.form.identifie
            ? `${this.form.author}-${this.form.identifie}`
            : '';
        platform.baseInfo.description = this.form.description;
    },
    loadFormulaSetting({ applyToForm = true } = {}) {
        if (this.option?.pureManifest || !this.identifie) return Promise.resolve(null);
        if (this.formulaSettingLoaded) {
            return Promise.resolve(applyToForm
                ? this.applyFormulaSetting(this.formulaBaseInfo)
                : this.formulaBaseInfo);
        }
        if (!this._formulaSettingPromise) {
            this.formulaSettingLoading = true;
            this._formulaSettingPromise = myAxios.post('/respo/setting/get', {
                identifie: this.identifie,
            }).then(response => {
                const baseInfo = response?.data?.data?.base_info;
                if (!baseInfo) throw new Error('制品基础信息为空');
                this.formulaBaseInfo = JSON.parse(JSON.stringify(baseInfo));
                this.formulaSettingLoaded = true;
                return this.formulaBaseInfo;
            }).finally(() => {
                this.formulaSettingLoading = false;
                this._formulaSettingPromise = null;
            });
        }
        return this._formulaSettingPromise.then(baseInfo => (
            applyToForm ? this.applyFormulaSetting(baseInfo) : baseInfo
        ));
    },
    applyFormulaSetting(baseInfo) {
        if (!baseInfo) return null;
        this.applyTraditionAnnotationForm(baseInfo.annotation || {});
        this.applySystemImageAnnotationForm(baseInfo.annotation || {});
        if (this.form.type == 'gateway-plugin') this.form.once = true;
        return baseInfo;
    },
    normalizeApplicationVersions(value) {
        const values = Array.isArray(value) ? value : String(value || '').split(/[,，\n]/);
        return [...new Set(values
            .map(item => String(item || ''))
            .flatMap(item => item.split(/[,，\n]/))
            .map(item => item.trim())
            .filter(Boolean))];
    },
    shouldSaveFormulaTypeSetting() {
        return Boolean(this.identifie);
    },
    async saveFormulaTypeSetting() {
        if (this.option?.pureManifest || !this.shouldSaveFormulaTypeSetting()) return;
        const baseInfo = await this.loadFormulaSetting({ applyToForm: false });
        if (!baseInfo) throw new Error('制品基础信息加载失败');
        const nextBaseInfo = {
            ...baseInfo,
            annotation: this.filterAnnotationsForType(
                this.json.application?.annotation || baseInfo.annotation || {},
            ),
            once: this.form.type == 'gateway-plugin'
                ? true
                : (['tradition', 'system-image'].includes(this.form.type)
                    ? false
                    : Boolean(baseInfo.once)),
        };
        await myAxios.post('/respo/setting/set', {
            identifie: this.identifie,
            base_info: nextBaseInfo,
        });
        if (this.form.type == 'gateway-plugin') {
            this.form.once = true;
        } else if (['tradition', 'system-image'].includes(this.form.type)) {
            this.form.once = false;
        }
        this.formulaBaseInfo = JSON.parse(JSON.stringify(nextBaseInfo));
    },
    getDefaultTypeTagName(type = this.form.type) {
        if (type == 'app-plugin') return '应用插件';
        if (type == 'tradition') return '传统应用';
        if (type == 'system-image') return '系统镜像';
        if (type == 'gateway-plugin') return '网关插件';
        return '';
    },
    async ensureDefaultTypeTags() {
        if (this.option?.pureManifest || !this.identifie) return;
        const names = [...new Set([
            this.getDefaultTypeTagName(this.initialApplicationType),
            this.getDefaultTypeTagName(this.form.type),
        ].filter(Boolean))];
        await Promise.all(names.map(name => myAxios.post('/respo/tag/add', {
            identifie: this.identifie,
            name,
        })));
    },
};
