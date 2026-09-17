import { confirm } from '@/utils/ui-feedback';

export function manifestSourceState() {
    return {
        zip: {
            codetype: 'zip',
            name: '',
            url: '',
            hasDockerfile: true,
        },
        web: {
            type: 'zip',
            name: '',
            url: '',
        },
    };
}

export const manifestSourceMethods = {
    initManifestSource(manifest = {}) {
        if (manifest?.source?.type == 'zip') {
            this.zip.codetype = 'zip';
            this.zip.url = manifest.source.url;
            this.zip.name = manifest.source.url.replace(/.*\//, '');
        }
        if (manifest?.web?.type == 'zip') {
            this.web.url = manifest.web.url;
            this.web.name = manifest.web.url.replace(/.*\//, '');
        }
    },
    syncManifestSourceForType() {
        if (!['light', 'system-image'].includes(this.form.type) && this.zip.url) {
            this.json.source = this.json.source || {};
            this.json.source.type = 'zip';
            this.json.source.url = this.zip.url;
        }
        if (!['app-plugin', 'system-image'].includes(this.form.type) && this.web.url) {
            this.json.web = this.json.web || {};
            this.json.web.type = 'zip';
            this.json.web.url = this.web.url;
        }
        if (this.form.type == 'light') delete this.json.source;
        if (this.form.type == 'app-plugin') delete this.json.web;
    },
    uploadSuccess(data) {
        if (!data?.data?.url) return;
        const url = data.data.url;
        this.zip.name = url.match(/[^/]+$/)[0];
        this.json.source = this.json.source || {};
        this.json.source.type = 'zip';
        this.json.source.url = url;
        this.zip.url = url;
        if (this.form.type == 'tradition') {
            this.form.startParams = this.traditionStartParams();
            this.ensureTraditionContainerDefaults();
            this.json.platform.startParams = this.serializeStartParams();
        }
        this.setYaml();
    },
    deleteUpload() {
        confirm({
            title: '提示',
            content: '确定要删除吗',
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            onOk: () => {
                this.zip.name = '';
                this.zip.url = '';
                delete this.json.source;
                if (this.form.type == 'tradition') {
                    this.form.startParams = this.traditionStartParams();
                    this.ensureTraditionContainerDefaults();
                    this.json.platform.startParams = this.serializeStartParams();
                }
                this.setYaml();
            },
        });
    },
};
