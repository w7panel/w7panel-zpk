<template>
    <a-spin :loading="loading" class="upload-spin">
        <a-upload class="upload-components" :accept="accept" :show-file-list="false" :custom-request="httpRequest"
            :on-before-upload="beforeUpload">
            <template #upload-button>
                <slot>
                    <a-button type="primary">点击上传</a-button>
                </slot>
            </template>
        </a-upload>
    </a-spin>
</template>
<script>
import myAxios from '../utils';
import JSZip from 'jszip';
import SparkMD5 from 'spark-md5';

export default {
    props: {
        file: { default: null },
        testDockerfile: { default: false },
        accept: { default: '.zip' },
    },
    emits: ['success', 'error', 'progress', 'testDockerfile'],
    data() {
        return {
            loading: false,
            chunkSize: 1024 * 1024,
        }
    },
    created() {
        if (this.file) {
            if (this.testDockerfile) {
                this.beforeDockerfile(this.file);
            } else {
                this.upfile(this.file)
            }
        }
    },
    watch: {
        file() {
            if (this.testDockerfile) {
                this.beforeDockerfile(this.file);
            } else {
                this.upfile(this.file)
            }
        }
    },
    methods: {

        calcFileMd5(file) {
            return new Promise((resolve, reject) => {
                const spark = new SparkMD5.ArrayBuffer();
                const reader = new FileReader();
                reader.onload = (e) => {
                    spark.append(e.target.result);
                    resolve(spark.end());
                };
                reader.onerror = (e) => {
                    reject(e);
                };
                reader.readAsArrayBuffer(file);
            });
        },
        beforeUpload() {
            return true;
        },
        beforeDockerfile(file) {
            let zip = new JSZip();
            zip.loadAsync(file).then(async () => {
                let hasDockerfile = false;
                let hasroot = true;
                let root = "";
                zip.forEach((path, o) => {
                    if (path.match(/^([^/]+)\//)) {
                        if (root) {
                            if (!RegExp('^' + root + '/').test(path)) { hasroot = false; }
                        } else {
                            root = path.match(/^[^\/]+/)[0];
                        }
                        if (path.match(/^([^/]+)\/Dockerfile/)) {
                            hasDockerfile = true;
                        }
                    } else {
                        hasroot = false;
                    }
                });
                if (!hasDockerfile || !hasroot) {
                    this.$emit('testDockerfile', false);
                } else {
                    this.$emit('testDockerfile', true);
                    this.upfile(file)
                }
            });
        },
        httpRequest(data) {
            let file = data.fileItem?.file || data.file?.file || data.file;
            if (this.testDockerfile) {
                this.beforeDockerfile(file);
            } else {
                this.upfile(file)
            }
            return {
                abort() { }
            };
        },
        async upfile(file) {
            this.loading = true;
            let time = Date.now();

            let md5;
            try {
                md5 = await this.calcFileMd5(file);
            } catch (e) {
                this.loading = false;
                this.$emit("error", e);
                return;
            }
            let slice = (count, over) => {
                let start = count * this.chunkSize;
                let end = start + this.chunkSize >= file.size ? file.size : start + this.chunkSize;

                const chunkFile = file.slice(start, end);
                const chunkTotal = Math.ceil(file.size / this.chunkSize);

                let filename = file.name.replace(/\.zip$/, () => {
                    return '_' + time + '.zip';
                })

                const formData = new FormData();
                formData.append('filename', filename);
                formData.append('md5', md5);
                formData.append('totalChunks', chunkTotal);
                formData.append('chunkNumber', count + 1);
                formData.append('finish', (over || chunkTotal == 1) ? 1 : 0);
                if (!over || chunkTotal == 1) {
                    formData.append('file', chunkFile);
                }

                let url = "/zip/upload";

                myAxios.post(url, formData, {
                    headers: { "Content-Type": "multipart/form-data" },
                    timeout: 90000,
                }).then((res) => {
                    if (!res.data) { this.loading = false; return }
                    if (chunkTotal > count + 1) {
                        this.$emit('progress', { value: parseInt(count / chunkTotal * 100), over: false });
                        slice(count + 1);
                    } else {
                        if (!over && chunkTotal != 1) { slice(count, true); return; }
                        this.$emit('progress', { value: 100, over: true });
                        this.$emit("success", res.data, file.name);
                        this.loading = false;
                    }
                }).catch((error) => {
                    this.loading = false;
                    this.$emit("error", error);
                });
            }
            slice(0);
        },
    }
}
</script>
