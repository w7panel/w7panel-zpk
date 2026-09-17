import jsyaml from 'js-yaml';
import hljs from 'highlight.js';
import { messageSuccess } from '@/utils/ui-feedback';

export function manifestYamlState() {
    return {
        showYaml: false,
        yamlDom: '',
        downloadUrl: '',
    };
}

export const manifestYamlMethods = {
    initManifestYaml() {
        hljs.configure({ ignoreUnescapedHTML: true });
    },
    cleanupManifestYaml() {
        if (this.downloadUrl) URL.revokeObjectURL(this.downloadUrl);
    },
    setYaml() {
        this.yaml = jsyaml.dump(this.json, {
            indent: 2,
            sortKeys: (first, second) => {
                if (second == 'menu') return -1;
                return first > second ? 1 : -1;
            },
        });
        this.yamlDom = `<pre class='pre'><code class='language-yaml'>${this.escapeHtml(this.yaml)}</code></pre>`;
        this.$nextTick(() => {
            hljs.highlightAll();
            this.download();
        });
    },
    escapeHtml(text) {
        return String(text || '')
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#39;');
    },
    openYamlPreview() {
        this.showYaml = true;
        this.$nextTick(() => this.setYaml());
    },
    download() {
        if (this.downloadUrl) URL.revokeObjectURL(this.downloadUrl);
        const file = new File([this.yaml], 'manifest.yaml', { type: 'text/plain' });
        this.downloadUrl = URL.createObjectURL(file);
    },
    onekeyCopy(text) {
        const textarea = document.createElement('textarea');
        document.body.appendChild(textarea);
        textarea.style.position = 'fixed';
        textarea.style.clip = 'rect(0 0 0 0)';
        textarea.style.top = '10px';
        textarea.value = text;
        textarea.select();
        document.execCommand('copy', true);
        document.body.removeChild(textarea);
        messageSuccess('复制成功');
    },
};
