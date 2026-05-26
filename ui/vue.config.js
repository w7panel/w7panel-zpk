const { defineConfig } = require('@vue/cli-service')

const proxyTarget = process.env.VUE_PROXY_TARGET || ''

module.exports = defineConfig({
  transpileDependencies: true,
  productionSourceMap: false,
  outputDir: 'dist',
  lintOnSave: false,
  publicPath: process.env.NODE_ENV === 'production' ? '' : '/',
  devServer: {
    host: '0.0.0.0',
    port: 8080,
    client: {
      overlay: false
    },
    proxy: {
      '/apis': {
        target: proxyTarget,
        changeOrigin: true
      },
      '/zpk': {
        target: proxyTarget,
        changeOrigin: true,
      },
      '/api': {
        target: proxyTarget,
        changeOrigin: true,
        pathRewrite: {
          '^/api': '/api'
        }
      },
      '/respo': {
        target: proxyTarget,
        changeOrigin: true,
        pathRewrite: {},
      },
      '/zip': {
        target: proxyTarget,
        changeOrigin: true,
        pathRewrite: {},
      },
      '/oauth': {
        target: proxyTarget,
        changeOrigin: true,
        pathRewrite: {},
      },
      '/attach': {
        target: proxyTarget,
        changeOrigin: true,
        pathRewrite: {
          '^/attach': '/attach'
        }
      },
    },
    headers: {
      'Access-Control-Allow-Origin': '*',
    },
  }
})
