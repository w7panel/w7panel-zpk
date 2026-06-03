<template>
    <div style="min-height:100%;">
          <div class="shortcut-header">1. 下载命令行工具</div>
          <div class="shortcut-desc">前往以下地址下载对应平台的命令行工具：</div>
          <a class="shortcut-link" target="_blank" href="https://github.com/w7panel/w7panel-zpk/releases/tag/latest">https://github.com/w7panel/w7panel-zpk/releases/tag/latest</a>
          <div class="shortcut-desc">Linux 平台示例工具名：</div>
          <div class="shortcut-content">
            <pre>zpk_linux_amd64</pre>
            <span class="copy-action" @click="onekeyCopy(`zpk_linux_amd64`)">复制</span>
          </div>
          <div class="shortcut-desc">如果下载后无法直接执行，请先添加执行权限：</div>
          <div class="shortcut-content">
            <pre>chmod +x zpk_linux_amd64</pre>
            <span class="copy-action" @click="onekeyCopy(`chmod +x zpk_linux_amd64`)">复制</span>
          </div>

          <div class="shortcut-header">2. 登录镜像仓库</div>
          <div class="shortcut-desc">在镜像仓库页面中查看访问凭证，获取<code>password</code>。</div>
          <div class="shortcut-desc">执行登录命令：</div>
          <div class="shortcut-content">
            <pre>./zpk_linux_amd64 login --username={{ userInfo.username }} --password=xxx --host={{ host }}</pre>
            <span class="copy-action" @click="onekeyCopy(`./zpk_linux_amd64 login --username=${userInfo.username} --password=xxx --host=${host}`)">复制</span>
          </div>
          <div class="shortcut-desc">参数说明：</div>
          <ul class="shortcut-list">
            <li><code>--username</code>：镜像仓库用户名</li>
            <li><code>--password</code>：镜像仓库密码</li>
            <li><code>--host</code>：镜像仓库地址</li>
          </ul>

          <div class="shortcut-header">3. 选择制品</div>
          <div class="shortcut-desc">登录成功后，选择需要操作的制品：</div>
          <div class="shortcut-content">
            <pre>./zpk_linux_amd64 use {{ $route.query.id }}</pre>
            <span class="copy-action" @click="onekeyCopy(`./zpk_linux_amd64 use ${$route.query.id}`)">复制</span>
          </div>

          <div class="shortcut-header">4. 添加附件</div>
          <div class="shortcut-desc">使用 <code>attach add</code> 命令添加附件：</div>
          <div class="shortcut-content">
            <pre>./zpk_linux_amd64 attach add --path=./xxx --type=helm</pre>
            <span class="copy-action" @click="onekeyCopy(`./zpk_linux_amd64 attach add --path=./xxx --type=helm`)">复制</span>
          </div>
          <div class="shortcut-desc">也可以根据附件类型选择不同的 <code>--type</code>：</div>
          <div class="shortcut-content">
            <pre>./zpk_linux_amd64 attach add --path=./xxx --type=helm
./zpk_linux_amd64 attach add --path=./xxx --type=backend
./zpk_linux_amd64 attach add --path=./xxx --type=frontend</pre>
            <span class="copy-action" @click="onekeyCopy(`./zpk_linux_amd64 attach add --path=./xxx --type=helm
./zpk_linux_amd64 attach add --path=./xxx --type=backend
./zpk_linux_amd64 attach add --path=./xxx --type=frontend`)">复制</span>
          </div>
          <div class="shortcut-desc">参数说明：</div>
          <ul class="shortcut-list">
            <li><code>--path</code>：附件路径，可以是当前目录下的文件或目录</li>
            <li><code>--type</code>：附件类型，支持 <code>helm</code>、<code>backend</code>、<code>frontend</code></li>
          </ul>
          <div class="shortcut-desc">如果需要为子应用添加附件，可以使用 <code>--sub_artifact</code> 参数：</div>
          <div class="shortcut-content">
            <pre>./zpk_linux_amd64 attach add --path=./xxx --type=backend --sub_artifact=sub_app_name</pre>
            <span class="copy-action" @click="onekeyCopy(`./zpk_linux_amd64 attach add --path=./xxx --type=backend --sub_artifact=sub_app_name`)">复制</span>
          </div>
          <div class="shortcut-desc">参数说明：</div>
          <ul class="shortcut-list">
            <li><code>--sub_artifact</code>：子应用标识</li>
          </ul>

          <div class="shortcut-header">5. 推送制品</div>
          <div class="shortcut-desc">所有附件添加完成后，执行推送命令：</div>
          <div class="shortcut-content">
            <pre>./zpk_linux_amd64 push</pre>
            <span class="copy-action" @click="onekeyCopy(`./zpk_linux_amd64 push`)">复制</span>
          </div>
    </div>
</template>

<script>
import myAxios from '@/utils/index';
import { messageSuccess } from '@/utils/ui-feedback';
export default {
    name: "publish-settings",
    props:{
      userInfo:{
        type:Object,
        default:()=>{
          return {
            username:''
          }
        }
      }
    },
    data() {
      return {
        host: ''
      }
    },
    created() {
      myAxios.get(`/v2/api/registry/info`).then(res => {
        let data = res?.data?.data?.server || {};
        this.host = data.external_domain || '';
      })
    },
    methods: {
      onekeyCopy(text){
        var textarea = document.createElement('textarea');
        document.body.appendChild(textarea);
        textarea.style.position = 'fixed';
        textarea.style.clip = 'rect(0 0 0 0)';
        textarea.style.top = '10px';
        textarea.value = text;
        textarea.select();
        document.execCommand('copy', true);
        document.body.removeChild(textarea);
        messageSuccess("复制成功");
      },
    }
}
</script>

<style scoped>
.shortcut-header{
  font-weight:bold;
  font-size: 16px;
  margin:30px 0 10px;
}
.shortcut-header:first-child{
  margin-top:0;
}
.shortcut-content{
  display:flex; align-items:center; margin:10px 0;
  justify-content: space-between;
  background:#f0f3fa; padding:10px; border-radius:4px;
  gap: 12px;
}
.copy-action {
  color: #3370ff;
  cursor: pointer;
  flex-shrink: 0;
}
.shortcut-content pre{
  margin:0;
  white-space:pre-wrap;
  word-break:break-all;
  line-height:1.6;
  flex:1;
}
.shortcut-desc{
  color:#999;
  line-height:1.8;
}
.shortcut-link{
  color:#409eff;
  margin:8px 0 12px;
  word-break:break-all;
}
.shortcut-list{
  color:#999;
  line-height:1.8;
  margin:8px 0 0;
  padding-left:20px;
}
.shortcut-desc code,
.shortcut-list code{
  color:#666;
  background:#f0f3fa;
  border-radius:3px;
  padding:2px 4px;
}
</style>
