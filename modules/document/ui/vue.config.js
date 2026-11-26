const { defineConfig } = require('@vue/cli-service');
module.exports = defineConfig({
    publicPath: '',
    productionSourceMap: false,
    lintOnSave: false,
    devServer: {
        port: 8081,
        proxy: process.env.VUE_PROXY_TARGET
            ? { '/doc': { target: process.env.VUE_PROXY_TARGET, changeOrigin: true } }
            : undefined,
    },
});
