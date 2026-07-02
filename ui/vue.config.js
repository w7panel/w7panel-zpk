const { defineConfig } = require('@vue/cli-service')
const path = require('path')

const proxyTarget = process.env.VUE_PROXY_TARGET || 'http://172.16.1.137:8000'
const devServerProxy = proxyTarget ? {
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
} : undefined

module.exports = defineConfig({
  transpileDependencies: true,
  productionSourceMap: false,
  outputDir: 'dist',
  lintOnSave: false,
  publicPath: process.env.NODE_ENV === 'production' ? '' : '/',
  chainWebpack: config => {
    config.module
      .rule('js')
      .use('babel-loader')
      .tap(options => ({
        ...options,
        cacheDirectory: path.resolve(__dirname, '../.cache/babel-loader')
      }))
  },
  devServer: {
    host: '0.0.0.0',
    port: 8080,
    client: {
      overlay: false
    },
    proxy: devServerProxy,
    headers: {
      'Access-Control-Allow-Origin': '*',
    },
  }
})
