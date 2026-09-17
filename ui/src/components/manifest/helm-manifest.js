import myAxios from '@/utils/index';
import { messageError } from '@/utils/ui-feedback';

export function helmFormDefaults() {
    return {
        helmtype: '1',
        repository: '',
        chartName: 'default',
        chartName2: '',
        version: '',
        kv: [],
        depend_yamls: [],
        useHelm: false,
    };
}

export function helmManifestState() {
    return {
        helmCharts: [],
        helmChartVersions: [],
        helmChartKeyword: '',
        helmChartVersionKeyword: '',
        getChartInfoLoading: false,
    };
}

function filterAutocompleteOptions(list = [], keyword = '') {
    const normalizedKeyword = String(keyword || '').toLowerCase();
    const source = normalizedKeyword
        ? list.filter(item => String(item || '').toLowerCase().includes(normalizedKeyword))
        : list;
    return source.map(item => ({ label: item, value: item }));
}

export const helmManifestComputed = {
    helmChartOptions() {
        return filterAutocompleteOptions(this.helmCharts, this.helmChartKeyword);
    },
    helmChartVersionOptions() {
        return filterAutocompleteOptions(this.helmChartVersions, this.helmChartVersionKeyword);
    },
};

export const helmManifestMethods = {
    cleanupHelmType(previousType, nextType) {
        if (previousType == 'helm' && nextType != 'helm') {
            delete this.json.platform.helm;
        }
    },
    getChartInfo() {
        this.getChartInfoLoading = true;
        return myAxios.post('/helm/chart/list', {
            repository_url: this.form.helm.repository,
        }, { dontalert: true }).then(res => {
            this.helmCharts = res?.data?.data?.charts || [];
            this.helmChartVersions = [];
        }).catch(() => {
            this.helmCharts = [];
            this.helmChartVersions = [];
        }).finally(() => {
            this.getChartInfoLoading = false;
        });
    },
    getChartVersion() {
        return myAxios.post('/helm/chart/version/list', {
            repository_url: this.form.helm.repository,
            chart: this.form.helm.chartName,
        }, { dontalert: true }).then(res => {
            this.helmChartVersions = res?.data?.data?.versions || [];
        }).catch(() => {
            this.helmChartVersions = [];
        });
    },
    helmyamlsUpload(value, index) {
        const file = value?.target?.files?.[0] || value;
        if (!file) return;
        if (!/\.ya?ml$/.test(file.name)) {
            messageError('请上传yaml文件');
            return;
        }
        this.form.helm.depend_yamls[index].name = file.name;
        this.form.helm.depend_yamls[index].nameInput = file.name.replace(/\.ya?ml$/, '');
        const reader = new FileReader();
        reader.onload = event => {
            this.form.helm.depend_yamls[index].yaml = event.target.result;
            this.changeForm();
        };
        reader.readAsText(file, 'UTF-8');
    },
    helmUploadSuccess(data) {
        this.form.helm.chartName2 = data?.data?.url || '';
        this.changeForm();
    },
    applyHelmManifestForm(platform = {}) {
        const helm = platform?.helm || {};
        const dependYamls = (helm.depend_yamls || []).map(item => ({
            name: item?.name,
            nameInput: item?.name?.replace?.(/\.yaml$/, ''),
            yaml: item.yaml,
        }));
        this.form.helm = {
            ...helmFormDefaults(),
            helmtype: helm.repository ? '1' : '2',
            repository: helm.repository || '',
            chartName: helm.repository ? (helm.chartName || '') : '',
            chartName2: helm.repository ? '' : (helm.chartName || ''),
            version: helm.version || '',
            kv: helm.kv || [],
            depend_yamls: dependYamls,
            useHelm: Boolean(helm.chartName),
        };
        if (helm.repository) this.getChartInfo();
    },
    applyHelmPlatform() {
        if (this.form.type != 'helm') {
            delete this.json.platform.helm;
            return;
        }
        delete this.json.source;
        if (this.json.platform?.['container-v2']) {
            delete this.json.platform['container-v2'].image;
        }
        const dependYamls = (this.form.helm?.depend_yamls || [])
            .filter(item => item.nameInput && item.yaml)
            .map(item => ({
                name: item.nameInput + '.yaml',
                yaml: item.yaml,
            }));
        const kv = (this.form.helm?.kv || [])
            .filter(item => item.name && item.value)
            .map(item => ({ name: item.name, value: item.value }));
        this.json.platform.helm = this.form.helm.helmtype == '1'
            ? {
                repository: this.form.helm.repository,
                chartName: this.form.helm.chartName,
                version: this.form.helm.version,
                depend_yamls: dependYamls,
                kv,
            }
            : {
                chartName: this.form.helm.chartName2,
                kv,
                depend_yamls: dependYamls,
            };
        delete this.json.platform.shells;
    },
    cleanupHelmPlatformForSubmit() {
        if (this.form.type != 'helm') return;
        delete this.json.platform['container-v2'];
        delete this.json.platform.volumes;
        delete this.json.platform.volumeClaimTemplates;
        delete this.json.platform.ingress;
        delete this.json.platform.runtimeClassName;
    },
};
