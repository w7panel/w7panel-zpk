<template>
    <div class="df content">
        <div class="fc" style="padding-left:0; overflow:auto;">
            <div>
                <a-form ref="formref" :model="form" :rules="rules" label-align="left"
                    :label-col-props="{ span: 5, flex: '0 0 120px' }"
                    :wrapper-col-props="{ span: 19, flex: '1' }" class="form manifest-form"
                    :class="{ 'manifest-start-params-only': option?.startParamsOnly }">
                    <div class="bg-white com-line df manifest-base-section">
                        <div class="fc">
                            <div class="c-00-6 df ai-c">基础配置</div>
                            <a-form-item class="mt-16 manifest-start-params-readonly" label="名称"
                                :field="option?.pureManifest ? 'name' : ''"
                                :inert="Boolean(option?.startParamsOnly)">
                                <a-input v-model="form.name" size="large" style="width:500px;" @change="changeForm"
                                    placeholder="请输入"></a-input>
                            </a-form-item>
                            <a-form-item class="manifest-start-params-readonly" label="标识"
                                :field="option?.pureManifest ? 'identifie' : ''"
                                :inert="Boolean(option?.startParamsOnly)">
                                <div class="df jc-b" style="width:500px;">
                                    <w7-identifie v-model:author="form.author" v-model:identifie="form.identifie"
                                        @change="changeForm" disabled />
                                </div>
                            </a-form-item>
                            <a-form-item class="manifest-start-params-readonly" label="描述" field="description"
                                :inert="Boolean(option?.startParamsOnly)">
                                <div class="df df-c">
                                    <a-input v-model="form.description" size="large" style="width:500px;"
                                        placeholder="请输入应用描述" @change="changeForm"></a-input>
                                </div>
                            </a-form-item>
                            <a-form-item v-if="option.pureManifest" label="可选安装">
                                <a-switch v-model="otherData.required" :checked-value="false"
                                    :unchecked-value="true"></a-switch>
                            </a-form-item>
                        </div>
                    </div>
                    <div class="bg-white mt-20 pb-24" :inert="Boolean(option?.startParamsOnly)">
                        <div class="c-00-6 df ai-c">
                            <span class="">代码配置</span>
                        </div>
                        <div class="mt-16">
                            <a-form-item label="类型">
                                <div class="df df-c fc">

                                    <a-radio-group v-model="form.type" @change="changeFormtype">
                                        <a-radio value="docker">原生应用</a-radio>
                                        <a-radio value="helm">K8sYaml</a-radio>
                                        <a-radio value="tradition">传统应用</a-radio>
                                        <a-radio value="app-plugin">应用插件</a-radio>
                                        <a-radio value="system-image">系统镜像</a-radio>
                                        <a-radio value="gateway-plugin">网关插件</a-radio>
                                    </a-radio-group>

                                    <a-form-item v-if="form.type == 'helm'" class="mt-20" style="margin-bottom:10px;"
                                        label="启用helm配置">
                                        <a-switch v-model="form.helm.useHelm" @change="changeForm"></a-switch>
                                    </a-form-item>

                                    <div v-if="form.type == 'helm' && form.helm.useHelm" class="greybox"
                                        style="margin-bottom:20px;">
                                        <div class="greybox-title">Helm配置</div>

                                        <a-form-item label="Chart包来源" style="margin-bottom:18px;">
                                            <a-radio-group v-model="form.helm.helmtype" @change="changeForm">
                                                <a-radio value="1">helm仓库</a-radio>
                                                <a-radio value="2">helm下载包</a-radio>
                                            </a-radio-group>
                                        </a-form-item>

                                        <a-form-item v-if="form.type == 'helm' && form.helm.helmtype == '1'"
                                            label="helm仓库地址" field="repository" style="margin-bottom:18px;">
                                            <a-input v-model="form.helm.repository" size="large"
                                                @change="getChartInfo(); changeForm();" style="width:500px;"
                                                placeholder="请输入" />
                                            <icon-loading v-if="getChartInfoLoading" class="is-loading fs-24 ml-10" />
                                        </a-form-item>

                                        <a-form-item v-if="form.type == 'helm' && form.helm.helmtype == '1'"
                                            label="chart名称" field="chartName" style="margin-bottom:18px;">
                                            <a-auto-complete ref="formhelmchartname" v-model="form.helm.chartName"
                                                :data="helmChartOptions" placeholder="请输入内容" :filter-option="false"
                                                @search="helmChartKeyword = $event"
                                                @change="getChartVersion(); changeForm();"
                                                @select="getChartVersion(); changeForm(); $refs.formhelmchartname?.blur?.()"
                                                style="width:240px;"></a-auto-complete>

                                            <a-auto-complete ref="formhelmversion" v-model="form.helm.version"
                                                :data="helmChartVersionOptions" placeholder="请输入内容"
                                                :filter-option="false" @search="helmChartVersionKeyword = $event"
                                                @change="changeForm"
                                                @select="changeForm(); $refs.formhelmversion?.blur?.()"
                                                style="width:240px;margin-left:20px;"></a-auto-complete>
                                        </a-form-item>

                                        <a-form-item v-if="form.type == 'helm' && form.helm.helmtype == '2'"
                                            label="Chart包地址" field="chartName2" style="margin-bottom:18px;">
                                            <a-input v-model="form.helm.chartName2" size="large" style="width:500px;"
                                                @change="changeForm" placeholder="请输入" />
                                            <files-upload accept=".tgz" @success="helmUploadSuccess">
                                                <a-button type="primary" size="large"
                                                    style="margin-left:10px;">上传</a-button>
                                            </files-upload>
                                        </a-form-item>

                                        <a-form-item v-if="form.type == 'helm'" label="安装配置">
                                            <manifest-config-table :rows="form.helm.kv" add-text="添加配置"
                                                @add="form.helm.kv.push({ name: '', value: '' }); changeForm();">
                                                <template #columns>
                                                    <manifest-config-table-column data-index="name" title="键">
                                                        <template #cell="{ record }">
                                                            <a-input v-model="record.name" size="large"
                                                                style="width:200px;" @change="changeForm"
                                                                placeholder="请输入" />
                                                        </template>
                                                    </manifest-config-table-column>
                                                    <manifest-config-table-column data-index="value" title="值">
                                                        <template #cell="{ record }">
                                                            <a-input v-model="record.value" size="large"
                                                                style="width:200px;" @change="changeForm"
                                                                placeholder="请输入" />
                                                        </template>
                                                    </manifest-config-table-column>
                                                    <manifest-config-table-column title="操作">
                                                        <template #cell="{ record, index }">
                                                            <span v-if="!record.disabled" class="c-blue cursor"
                                                                @click="form.helm.kv.splice(index, 1); changeForm();">删除</span>
                                                        </template>
                                                    </manifest-config-table-column>
                                                </template>
                                            </manifest-config-table>
                                        </a-form-item>

                                    </div>
                                    <div v-if="form.type == 'helm'" class="greybox">
                                        <div class="greybox-title">YAML配置</div>
                                        <div>
                                            <div v-for="(item, index) in form.helm.depend_yamls" :key="index"
                                                style="margin-bottom:30px;" class="show-on-hover-container">
                                                <div class="df yaml-header">
                                                    <a-upload
                                                        :on-before-upload="fileItem => { helmyamlsUpload(fileItem.file, index); return false }"
                                                        :show-file-list="false">
                                                        <a-button type="primary">上传文件</a-button>
                                                    </a-upload>
                                                    <div class="show-on-hover yaml-delete">
                                                        <icon-close :size="20" class="cursor"
                                                            @click="form.helm.depend_yamls.splice(index, 1); changeForm();">
                                                        </icon-close>
                                                    </div>

                                                </div>
                                                <a-form-item label="标题" style="margin-bottom: 20px">
                                                    <a-input v-model="item.nameInput" placeholder="标题"
                                                        @change="changeForm" style="width:500px;"><template
                                                            #append>.yaml</template></a-input>
                                                </a-form-item>
                                                <a-form-item label="YAML">
                                                    <a-textarea v-model="item.yaml" :rows="5"
                                                        placeholder="请输入YAML" class="fc" @change="changeForm" />
                                                    <div class="command-upfile ml-20">
                                                        <input type="file" @change="e => helmyamlsUpload(e, index)" />
                                                    </div>
                                                </a-form-item>
                                            </div>
                                            <div class="df ai-c jc-c cursor"
                                                @click="form.helm.depend_yamls.push({ nameInput: '', name: '', yaml: '' })">
                                                <span class="addmenu"><icon-plus />添加YAML</span>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </a-form-item>

                            <a-form-item v-if="form.type == 'gateway-plugin'" label="网关插件配置">
                                <div class="greybox" style="margin-bottom:0;">
                                    <a-form-item label="插件分类" field="gatewayPluginCategory"
                                        style="margin-bottom:18px;">
                                        <a-select v-model="form.gatewayPluginCategory" style="width:240px;"
                                            @change="changeForm">
                                            <a-option v-for="item in gatewayPluginCategoryOptions" :key="item.value"
                                                :value="item.value">{{ item.label }}</a-option>
                                        </a-select>
                                    </a-form-item>
                                    <a-form-item label="运行时驱动" field="gatewayPluginDriver"
                                        style="margin-bottom:18px;">
                                        <a-select v-model="form.gatewayPluginDriver" style="width:240px;"
                                            @change="changeForm">
                                            <a-option value="higress-wasm/v1">Higress Wasm</a-option>
                                        </a-select>
                                    </a-form-item>
                                    <a-form-item label="镜像地址" field="gatewayPluginUrl"
                                        style="margin-bottom:18px;">
                                        <a-input v-model="form.gatewayPluginUrl" size="large" style="width:500px;"
                                            placeholder="oci://... 或 http(s)://..." @change="changeForm" />
                                    </a-form-item>
                                    <a-form-item label="执行阶段" field="gatewayPluginPhase"
                                        style="margin-bottom:18px;">
                                        <a-select v-model="form.gatewayPluginPhase" style="width:240px;"
                                            @change="changeForm">
                                            <a-option value="UNSPECIFIED_PHASE">默认阶段</a-option>
                                            <a-option value="AUTHN">认证阶段</a-option>
                                            <a-option value="AUTHZ">鉴权阶段</a-option>
                                            <a-option value="STATS">统计阶段</a-option>
                                        </a-select>
                                    </a-form-item>
                                    <a-form-item label="优先级" field="gatewayPluginPriority"
                                        style="margin-bottom:18px;">
                                        <div class="df df-c">
                                            <a-input-number v-model="form.gatewayPluginPriority" :min="0" :max="1000"
                                                style="width:240px;" @change="changeForm" />
                                            <span class="c-99 mt-6">同一执行阶段按优先级降序执行，数值越大越先执行；默认值为 0。</span>
                                        </div>
                                    </a-form-item>
                                    <a-form-item label="支持范围" style="margin-bottom:18px;">
                                        <div class="df df-c">
                                            <div>
                                                <a-checkbox v-model="form.gatewayPluginSupportGlobal"
                                                    @change="changeForm">支持全局配置</a-checkbox>
                                                <a-checkbox v-model="form.gatewayPluginSupportRule" class="ml-20"
                                                    @change="changeForm">支持规则配置</a-checkbox>
                                            </div>
                                            <span class="c-99 mt-6">全局配置默认开启；规则配置会显示在应用域名管理的“更多”中。</span>
                                        </div>
                                    </a-form-item>
                                    <a-form-item label="全局默认启用" style="margin-bottom:18px;">
                                        <div class="df df-c">
                                            <div>
                                                <a-switch v-model="form.gatewayPluginDefaultEnabled"
                                                    :disabled="!form.gatewayPluginSupportGlobal" @change="changeForm" />
                                            </div>
                                            <span class="c-99 mt-6">仅作用于全局配置；关闭后可先完善配置，再到网关插件列表手动启用。</span>
                                        </div>
                                    </a-form-item>
                                    <a-form-item label="默认配置" style="margin-bottom:0;">
                                        <div class="df df-c">
                                            <a-textarea v-model="form.gatewayPluginDefaultConfig" :rows="8"
                                                :spellcheck="false" placeholder="请输入 JSON 配置，默认为 {}"
                                                style="width:500px;" @change="changeForm" />
                                            <span class="c-99 mt-6">请提供不含真实密钥的初始 JSON；安装后用户仍可在网关插件列表中修改。</span>
                                        </div>
                                    </a-form-item>
                                </div>
                            </a-form-item>

                            <a-alert v-if="form.type == 'system-image'" type="info" show-icon
                                class="zpk-primary-alert mt-16 mb-20" title="系统镜像说明"
                                :closable="false">
                                <div class="registry-alert-item">1. 系统镜像用于创建具备完整系统环境的轻量虚拟机，可以像普通主机一样运行系统服务。</div>
                                <div class="registry-alert-item mt-6">2. 底层系统数据会持久保存，实例重启后已安装的软件、系统配置和用户数据不会丢失。</div>
                            </a-alert>

                            <a-form-item v-if="form.type == 'system-image'" label="系统镜像配置">
                                <div class="greybox" style="margin-bottom:0;">
                                    <a-form-item label="分类" field="systemImageCategory" required
                                        style="margin-bottom:18px;">
                                        <a-select v-model="form.systemImageCategory" size="large"
                                            style="width:500px;" @change="changeForm">
                                            <a-option value="operating-system">操作系统</a-option>
                                            <a-option value="site-management">建站管理</a-option>
                                            <a-option value="enterprise-app">企业应用</a-option>
                                        </a-select>
                                    </a-form-item>
                                    <a-form-item label="镜像地址" field="systemImageTemplate" required
                                        style="margin-bottom:18px;">
                                        <div class="df df-c">
                                            <a-input v-model="form.systemImageTemplate" size="large"
                                                style="width:500px;" placeholder="例如 ubuntu:{version}"
                                                @change="syncSystemImageConfig" />
                                            <span class="c-99 mt-6">使用 {version} 作为安装时所选系统版本的占位符。</span>
                                        </div>
                                    </a-form-item>
                                    <a-form-item label="系统版本" field="systemImageVersions" required
                                        style="margin-bottom:0;">
                                        <div class="df df-c">
                                            <a-input-tag v-model="form.systemImageVersions" size="large"
                                                style="width:500px;" placeholder="输入版本后按回车，例如 22.04"
                                                allow-clear unique-value @change="syncSystemImageConfig" />
                                            <span class="c-99 mt-6">安装时用户从这里配置的版本中选择，所选版本会替换镜像地址中的 {version}。</span>
                                        </div>
                                    </a-form-item>
                                </div>
                            </a-form-item>

                            <a-alert v-if="form.type == 'tradition'" type="info" show-icon
                                class="zpk-primary-alert mt-16 mb-20" title="说明" :closable="false">
                                <div class="registry-alert-item">1. 传统应用会作为独立应用安装，运行方式固定为 Deployment。</div>
                                <div class="registry-alert-item mt-6">2. 安装时通过“传统应用版本”启动参数替换运行容器镜像中的 {version}。</div>
                                <div class="registry-alert-item mt-6">3. 传统应用容器的启动命令可在页面下方“应用配置”中配置。</div>
                                <div class="registry-alert-item mt-6">4. 页面下方“脚本配置”中的安装、升级脚本只在安装或升级此制品时执行。</div>
                            </a-alert>

                            <a-form-item v-if="form.type == 'tradition'" label="传统应用配置"
                                class="tradition-config-item">
                                <a-spin :loading="formulaSettingLoading" class="tradition-config-spin">
                                    <div class="greybox" style="margin-bottom:0;">
                                        <a-form-item label="应用语言" field="traditionImageLanguage" required
                                            style="margin-bottom:18px;">
                                            <a-select v-model="form.traditionImageLanguage" size="large"
                                                style="width:500px;" placeholder="请选择应用语言" allow-search
                                                @change="changeTraditionLanguage">
                                                <a-option v-for="option in traditionLanguageOptions"
                                                    :key="option.value" :value="option.value" :label="option.label">
                                                    {{ option.label }}
                                                </a-option>
                                            </a-select>
                                        </a-form-item>
                                        <a-form-item label="镜像地址" field="traditionImageTemplate" required
                                            style="margin-bottom:18px;">
                                            <div class="df df-c">
                                                <a-input v-model="form.traditionImageTemplate" size="large"
                                                    style="width:500px;" placeholder="例如 php:{version}-fpm-alpine"
                                                    @change="syncTraditionRuntimeConfig" />
                                                <span class="c-99 mt-6">使用 {version} 作为传统应用版本占位符。</span>
                                            </div>
                                        </a-form-item>
                                        <a-form-item label="传统应用版本" field="traditionImageVersion" required
                                            style="margin-bottom:18px;">
                                            <div class="df df-c">
                                                <a-input-tag v-model="form.traditionImageVersion" size="large"
                                                    style="width:500px;" placeholder="输入版本后按回车，例如 8.1"
                                                    allow-clear unique-value @change="syncTraditionVersionConfig" />
                                                <span class="c-99 mt-6">每次输入一个语言版本并按回车，可添加多个版本，例如 7.4、8.1。</span>
                                            </div>
                                        </a-form-item>
                                        <a-form-item label="系统重启还原" style="margin-bottom:18px;">
                                            <div class="df df-c" style="align-items:flex-start;">
                                                <a-switch v-model="form.traditionSystemRebootRestore"
                                                    @change="syncTraditionRuntimeConfig" />
                                                <span class="c-99 mt-6">关闭后使用持久存储保留容器系统层。</span>
                                            </div>
                                        </a-form-item>
                                        <a-form-item v-if="option?.edit" label="NGINX 网关"
                                            style="margin-bottom:18px;">
                                            <div class="df df-c" style="align-items:flex-start;">
                                                <a-switch v-model="form.traditionNginxGateway"
                                                    :disabled="traditionNginxGatewayChanging"
                                                    @change="toggleTraditionNginxGateway" />
                                                <span class="c-99 mt-6">开启后安装时会将 NGINX 服务作为对外网关，用于转发请求至后端服务。</span>
                                                <span class="c-99 mt-6">常用于 PHP-FPM FastCGI 等无法直接提供 HTTP 服务的场景；若后端可直接提供 HTTP 服务，则无需启用。</span>
                                            </div>
                                        </a-form-item>
                                        <a-form-item v-if="form.traditionNginxGateway" label="NGINX 模板"
                                            field="traditionNginxVhostTemplate" style="margin-bottom:0;">
                                            <div class="nginx-template-field">
                                                <div class="nginx-template-input">
                                                    <a-textarea v-model="form.traditionNginxVhostTemplate"
                                                        class="nginx-template-textarea"
                                                        :auto-size="{minRows:10, maxRows:20}"
                                                        :spellcheck="false" placeholder="请输入 NGINX vhost 模板"
                                                        style="width:500px;" />
                                                    <a-button size="mini" type="text"
                                                        class="nginx-template-example-entry"
                                                        @click="nginxTemplateExampleVisible = true">
                                                        查看完整示例
                                                    </a-button>
                                                </div>
                                                <span class="c-99 mt-6">用于生成此传统应用的 NGINX 访问配置。</span>
                                            </div>
                                        </a-form-item>
                                    </div>
                                </a-spin>
                            </a-form-item>

                            <a-form-item v-if="form.type == 'app-plugin'" label="依赖传统应用">
                                <div class="df df-ww env-sel">
                                    <div v-for="item in traditionList" :key="item.identifie"
                                        :class="{ 'active': form.traditionName == item.identifie }"
                                        @click="changeEnv(item)" class="env-item df df-c ai-c cursor">
                                        <img :src="traditionIcon(item.icon)" class="img" alt="" />
                                        <div class="lh-1">{{ item.name }}</div>
                                        <input type="radio" v-model="form.traditionName" :value="item.identifie"
                                            style="display:none;" />
                                    </div>
                                </div>
                            </a-form-item>
                            <a-form-item v-if="form.type == 'app-plugin'" label="依赖应用版本">
                                <a-select v-model="form.traditionVersion" placeholder="请选择" @change="changeForm"
                                    size="large" style="width:500px;">
                                    <a-option
                                        v-for="(item, index) in traditionList?.find?.(i => i.identifie == form.traditionName)?.versions || []"
                                        :key="index" :label="item" :value="item"></a-option>
                                </a-select>
                            </a-form-item>
                            <a-form-item
                                v-if="form.type != 'system-image' && form.type != 'gateway-plugin' && (!option || !option.pureManifest && form.type != 'docker' && form.type != 'light' && form.type != 'helm')"
                                label="代码包">
                                <div class="df ai-e">
                                    <files-upload @success="uploadSuccess" @testDockerfile="v => zip.hasDockerfile = v">
                                        <div v-if="zip.name" class="upfilebox df df-c ai-c jc-c">
                                            <img src="@/assets/img/zip.png" alt=""
                                                style="width:60px;height:60px;display:block;" />
                                            <div class="df ai-c mt-20">
                                                <icon-check-circle-fill class="c-green file-status-icon" />
                                                <div class="fs-14 c-33"
                                                    style="vertical-align:middle;max-width:200px;overflow:hidden;text-overflow:ellipsis;">
                                                    {{
                                                        zip.name }}</div>
                                            </div>
                                            <div class="mask df df-c ai-c jc-c">
                                                <a-button type="primary">重新上传</a-button>
                                            </div>
                                        </div>
                                        <div v-else class="upfilebox df df-c ai-c jc-c">
                                            <div class="df df-c ai-c">
                                                <icon-upload class="uploadicon c-99" />
                                                <span class="uploadbtn df ai-c">
                                                    <icon-upload class="uploadicon c-33" />
                                                    <span class="lh-1 c-33">上传代码包</span>
                                                </span>
                                            </div>
                                        </div>
                                    </files-upload>
                                    <div class="c-blue cursor ml-20" @click="deleteUpload">删除</div>
                                    <a-tooltip v-if="form.type == 'app-plugin'"
                                        content="请在应用插件的外层目录打包，并保留插件安装所需的完整目录结构。例如微擎插件需要将 addons 目录一起打包，使压缩包根目录直接包含 addons 目录。安装时会解压到 /www/wwwroot/&lt;站点域名&gt;，并随当前版本发布。"
                                        position="top">
                                        <icon-exclamation-circle-fill class="fs-16 c-99 ml-4" />
                                    </a-tooltip>
                                    <a-tooltip v-else-if="form.type == 'tradition'"
                                        content="代码包为可选配置。请进入项目根目录后打包，压缩包根目录应直接包含安装内容，不要包含外层目录。例如：cd 项目目录 && zip -r app.zip .。安装时会解压到 /www/wwwroot/&lt;站点域名&gt;，并随当前版本发布。"
                                        position="top">
                                        <icon-exclamation-circle-fill class="fs-16 c-99 ml-4" />
                                    </a-tooltip>
                                </div>
                                <div v-if="zip.hasDockerfile === false && !['tradition', 'app-plugin'].includes(form.type)" class="c-red mt-10">没有检测到Dockerfile文件，请重新上传
                                </div>
                            </a-form-item>

                            <slot></slot>

                            <a-form-item v-if="form.type == 'system-image'" label="CMD">
                                <div class="df df-c">
                                    <div v-for="(item, index) in form.cmd" :key="index" class="df ai-e"
                                        :style="{ marginTop: index == 0 ? 0 : '10px' }">
                                        <a-textarea v-model="form.cmd[index]" @change="changeForm"
                                            :spellcheck="false" :rows="2" style="width:500px;"
                                            placeholder="请输入"></a-textarea>
                                        <div class="df ai-c">
                                            <span class="ml-10 cursor c-blue"
                                                @click="form.cmd.length > 1 ? form.cmd.splice(index, 1) : form.cmd = ['']; changeForm();">删除</span>
                                            <span class="ml-10 cursor c-blue" v-if="index + 1 == form.cmd.length"
                                                @click="form.cmd.push('')">添加</span>
                                            <a-tooltip v-if="index + 1 == form.cmd.length"
                                                content="运行容器的启动命令。该配置可选，留空时使用镜像自身的 ENTRYPOINT/CMD。"
                                                position="top">
                                                <icon-exclamation-circle-fill class="fs-16 c-99 ml-4" />
                                            </a-tooltip>
                                        </div>
                                    </div>
                                </div>
                            </a-form-item>

                            <a-form-item v-if="form.type != 'helm' && form.type != 'light' && form.type != 'gateway-plugin'"
                                label="脚本配置" field="shell" class="mt-16">
                                <div style="flex:1;">
                                    <manifest-config-table :rows="form.shell" add-text="添加脚本"
                                        table-class="shell-config-table"
                                        @add="addShellTask">
                                        <template #columns>
                                            <manifest-config-table-column data-index="type" title="类型" width="180px">
                                                <template #cell="{ record }">
                                                    <a-select v-model="record.type" placeholder="请选择类型"
                                                        style="width:160px;" @change="changeForm">
                                                        <a-option v-for="item in shellTypeOptions" :key="item.value"
                                                            :label="item.label" :value="item.value"></a-option>
                                                    </a-select>
                                                </template>
                                            </manifest-config-table-column>
                                            <manifest-config-table-column data-index="title" title="名称" width="160px">
                                                <template #cell="{ record }">
                                                    <a-input v-model="record.title" placeholder="请输入任务名称"
                                                        @input="changeForm" @change="changeForm"></a-input>
                                                </template>
                                            </manifest-config-table-column>
                                            <manifest-config-table-column title="任务" width="160px">
                                                <template #cell="{ record, index }">
                                                    <div class="shell-task-cell">
                                                        <span class="c-99">{{ getShellTaskStatus(record) }}</span>
                                                        <span class="c-blue cursor handle"
                                                            @click="openShellConfig(record, index)">编辑</span>
                                                    </div>
                                                </template>
                                            </manifest-config-table-column>
                                            <manifest-config-table-column title="操作" width="80px">
                                                <template #cell="{ index }">
                                                    <span class="c-blue cursor handle"
                                                        @click="form.shell.splice(index, 1); changeForm();">删除</span>
                                                </template>
                                            </manifest-config-table-column>
                                        </template>
                                    </manifest-config-table>
                                </div>
                            </a-form-item>


                        </div>
                    </div>

                    <div v-if="form.type != 'helm' && form.type != 'app-plugin' && form.type != 'gateway-plugin' && form.type != 'system-image'" class="bg-white com-line mt-20">
                        <div class="mt-16">
                            <a-form-item label="应用配置">
                                <div v-if="form.containers && form.containers.length">
                                    {{form.containers.map(i => i.name).join(', ')}}</div>
                                <div v-else>无</div>
                                <span class="cursor c-blue ml-10" @click="openAppset">编辑</span>
                            </a-form-item>


                            <a-form-item v-if="form.type != 'helm' && form.type != 'system-image'" label="域名设置" class="mt-16">
                                <div v-if="form.ingress && form.ingress.length">
                                    {{ form.ingress.map(item => item.name || '未命名配置').join(', ') }}
                                </div>
                                <div v-else>无</div>
                                <span class="cursor c-blue ml-10" @click="openDomainConfig">编辑</span>
                            </a-form-item>
                        </div>
                    </div>

                    <div class="bg-white com-line mt-20 manifest-start-params-section">
                        <a-form-item v-if="form.type != 'gateway-plugin'" class="mt-16 manifest-start-param-editor" label="启动参数" field="startParams">
                            <div class="manifest-field-stack">
                                <div v-if="form.type != 'system-image'" class="start-param-head">
                                    <div class="start-param-services">
                                        <a-checkbox v-model="form.mysql8">mysql8.0</a-checkbox>
                                        <a-checkbox v-model="form.mysql5">mysql5.6</a-checkbox>
                                        <a-checkbox v-model="form.redis">redis</a-checkbox>
                                        <a-checkbox v-model="form.mongodb6">mongodb</a-checkbox>
                                    </div>
                                    <a-button @click="openSpEdit">批量修改</a-button>
                                </div>
                                <manifest-config-table class="mt-10" :rows="form.startParams"
                                    add-text="添加启动参数"
                                    @add="form.startParams.push({ name: '', title: '', required: true, values_text: '', module_name: '', description: '' })">
                                    <template #columns>
                                        <manifest-config-table-column data-index="name" title="标识">
                                            <template #cell="{ record }">
                                                <a-input v-model="record.name" @change="getStart"
                                                    :disabled="computedSpDisabled(record)" :spellcheck="false"
                                                    placeholder="配置标识"></a-input>
                                            </template>
                                        </manifest-config-table-column>
                                        <manifest-config-table-column data-index="title" title="名称">
                                            <template #cell="{ record }">
                                                <a-input v-model="record.title" @change="getStart"
                                                    :disabled="computedSpDisabled(record)" :spellcheck="false"
                                                    placeholder="配置名称"></a-input>
                                            </template>
                                        </manifest-config-table-column>
                                        <manifest-config-table-column data-index="required" title="必填">
                                            <template #cell="{ record }">
                                                <a-switch v-model="record.required" @change="getStart"
                                                    :disabled="computedSpDisabled(record)" />
                                            </template>
                                        </manifest-config-table-column>
                                        <manifest-config-table-column data-index="values_text" title="默认值">
                                            <template #cell="{ record }">
                                                <a-input v-model="record.values_text" @change="getStart"
                                                    :disabled="computedSpDisabled(record)" :spellcheck="false"
                                                    placeholder="配置默认值"></a-input>
                                            </template>
                                        </manifest-config-table-column>
                                        <manifest-config-table-column data-index="module_name" title="依赖系统组件标识">
                                            <template #cell="{ record }">
                                                <a-input v-model="record.module_name" @change="getStart"
                                                    :disabled="computedSpDisabled(record)" :spellcheck="false"
                                                    placeholder="依赖的系统组件标识名"></a-input>
                                            </template>
                                        </manifest-config-table-column>
                                        <manifest-config-table-column title="操作" width="100px">
                                            <template #cell="{ record, index }">
                                                <span v-if="!isFixedStartParam(record)" class="c-blue cursor handle" @click="openSpDesc(record)">编辑描述</span>
                                                <span v-if="!isFixedStartParam(record)" class="c-blue cursor handle"
                                                    @click="form.startParams.splice(index, 1); getStart();">删除</span>
                                            </template>
                                        </manifest-config-table-column>
                                    </template>
                                </manifest-config-table>
                            </div>
                        </a-form-item>

                        <a-form-item class="mt-20" label="安装依赖" :inert="Boolean(option?.startParamsOnly)">
                            <manifest-config-table :rows="form.depends" table-class="install-depend-table"
                                :add-text="form.type == 'tradition' ? '' : '添加安装依赖'"
                                @add="openDependPicker()">
                                <template #columns>
                                    <manifest-config-table-column data-index="identifie" title="名称" width="36%">
                                        <template #cell="{ record, index }">
                                            <div class="install-depend-app-cell">
                                                <div class="install-depend-app-main">
                                                    <div class="install-depend-app-name">
                                                        {{ getDependName(record) || '未选择依赖' }}
                                                    </div>
                                                    <div v-if="record.identifie" class="install-depend-app-identifie">
                                                        {{ record.identifie }}
                                                    </div>
                                                </div>
                                                <a-button v-if="!isTraditionFixedDependency(record)" type="text" size="mini"
                                                    @click="openDependPicker(index)">
                                                    {{ record.identifie ? '更换' : '选择' }}
                                                </a-button>
                                            </div>
                                        </template>
                                    </manifest-config-table-column>
                                    <manifest-config-table-column data-index="subidentifie" title="子应用" width="34%">
                                        <template #cell="{ record, index }">
                                            <a-select v-model="record.subidentifie" size="large"
                                                @change="record.subname = getSubDependName(index, record.subidentifie); changeForm()"
                                                placeholder="请选择"
                                                :disabled="!record.identifie || isTraditionFixedDependency(record)">
                                                <a-option v-for="(name, identifie) in getSubDependsOptions(index)"
                                                    :key="identifie" :label="name" :value="identifie"></a-option>
                                            </a-select>
                                        </template>
                                    </manifest-config-table-column>
                                    <manifest-config-table-column data-index="required" title="必须安装" width="10%">
                                        <template #cell="{ record }">
                                            <a-switch v-model="record.required" :disabled="isTraditionFixedDependency(record)"
                                                @change="changeForm" />
                                        </template>
                                    </manifest-config-table-column>
                                    <manifest-config-table-column data-index="name" title="标识" width="12%">
                                        <template #cell="{ record }">{{ record.identifie }}</template>
                                    </manifest-config-table-column>
                                    <manifest-config-table-column title="操作" width="8%">
                                        <template #cell="{ index }">
                                            <span v-if="!isTraditionFixedDependency(form.depends[index])" class="c-blue cursor"
                                                @click="form.depends.splice(index, 1); changeForm();">删除</span>
                                            <span v-else class="c-99">固定</span>
                                        </template>
                                    </manifest-config-table-column>
                                </template>
                            </manifest-config-table>
                        </a-form-item>
                    </div>

                    <div class="bg-white pb-24 mt-20 df ai-c manifest-submit-section">
                        <a-button v-if="option.pureManifest" :loading="submiting" type="primary" @click="submit(otherData)">确定提交</a-button>
                        <a-button v-else :loading="submiting" :disabled="traditionNginxGatewayChanging" type="primary" @click="submit()">确定提交</a-button>
                    </div>
                </a-form>
            </div>
        </div>
        <a-drawer v-model:visible="showYaml" :width="640" title="预览 YAML" :footer="false" unmount-on-close>
            <div class="yaml-preview-panel">
                <div class="yaml-preview-drawer" v-html="yamlDom"></div>
                <div class="yaml-preview-actions">
                    <button class="copybtn" @click="onekeyCopy(yaml)">一键复制</button>
                    <a :href="downloadUrl" download="manifest.yaml" class="copybtn">下载</a>
                </div>
            </div>
        </a-drawer>

        <a-drawer v-model:visible="nginxTemplateExampleVisible" :width="720" title="NGINX 模板示例"
            unmount-on-close>
            <div class="nginx-template-example-panel">
                <a-alert type="info" show-icon :closable="false">
                    示例包含 PHP-FPM、静态资源缓存和前端控制器路由配置。使用前请根据实际应用检查端口和规则。
                </a-alert>
                <div class="nginx-template-example-placeholders">
                    <span>系统将自动替换：</span>
                    <a-button v-for="placeholder in nginxTemplatePlaceholders" :key="placeholder"
                        size="mini" type="outline" class="nginx-template-example-placeholder"
                        title="点击复制" @click="onekeyCopy(placeholder)">
                        {{ placeholder }}
                    </a-button>
                </div>
                <pre class="nginx-template-example-code"><code>{{ nginxTemplateExample }}</code></pre>
            </div>
            <template #footer>
                <div class="nginx-template-example-footer">
                    <a-button @click="nginxTemplateExampleVisible = false">关闭</a-button>
                    <a-button @click="onekeyCopy(nginxTemplateExample)">复制示例</a-button>
                    <a-button type="primary" @click="useNginxTemplateExample">使用此示例</a-button>
                </div>
            </template>
        </a-drawer>

        <a-modal v-model:visible="dependForm.show" :title="dependForm.editIndex >= 0 ? '修改子应用' : '添加子应用'" :width="640"
            :footer="false"
            @close="dependForm = { show: false, editIndex: -1, identifie_before: '', identifie_last: '', identifie: '', name: '', required: false, from: '' };">
            <a-form ref="depend" :rules="addRules" :model="dependForm"
                :label-col-props="{ span: 5, flex: '0 0 80px' }"
                :wrapper-col-props="{ span: 19, flex: '1' }" class="manifest-dialog-form">
                <a-form-item label="标识" field="identifie">
                    <w7-identifie v-model:author="dependForm.identifie_before"
                        v-model:identifie="dependForm.identifie_last" :author-disabled="true" />
                </a-form-item>
                <a-form-item label="名称" field="name">
                    <a-input placeholder="请输入名称" v-model="dependForm.name" size="large"
                        style="width:500px;"></a-input>
                </a-form-item>
                <a-form-item label="可选安装" field="from">
                    <a-switch v-model="dependForm.required" :checked-value="false" :unchecked-value="true" />
                </a-form-item>
            </a-form>
            <div class="dialog-footer">
                <a-button size="large" @click="dependForm.show = false">取消</a-button>
                <a-button type="primary" size="large" @click="addDepend">确定</a-button>
            </div>
        </a-modal>

        <a-modal v-model:visible="spEdit.show" title="启动参数" :width="700" modal-class="envdialog" :footer="false">
            <span class="c-66">格式：键=值 #标题 : 描述 : 必填(1或0) : 依赖组件标识</span>
            <a-textarea v-model="spEdit.values" class="mt-10" :spellcheck="false"
                placeholder="格式：键=值 #标题 : 描述 : 必填(1或0) : 依赖组件标识" :rows="12"
                :auto-size="false"></a-textarea>
            <div class="dialog-footer">
                <a-button @click="spEdit.show = false;">取消</a-button>
                <a-button @click="submitSpEdit" type="primary">确定</a-button>
            </div>
        </a-modal>

        <a-modal v-model:visible="spDesc.show" title="描述" :width="520" modal-class="envdialog">
            <a-input v-model="spDesc.value" placeholder="请编辑描述" />
            <template #footer>
                <a-button @click="spDesc.show = false">取消</a-button>
                <a-button type="primary" @click="submitSpDesc">确定</a-button>
            </template>
        </a-modal>

        <a-modal v-model:visible="shellConfig.show" title="配置脚本任务" :width="720" :footer="false">
            <a-alert type="info" show-icon class="zpk-primary-alert mb-20" title="提示" :closable="false">
                <div class="registry-alert-item">脚本会以独立任务运行，执行容器决定任务使用的镜像、容器 env 和目录挂载。</div>
                <div class="registry-alert-item mt-6">启动参数会同时注入任务的 env 环境，可在脚本中通过环境变量读取。</div>
                <div class="registry-alert-item mt-6">应用插件固定使用所选传统应用的应用容器。</div>
            </a-alert>
            <a-form v-if="shellConfig.item" :model="shellConfig.item" label-align="left"
                class="manifest-dialog-form shell-config-form"
                :label-col-props="{ flex: '0 0 90px' }" :wrapper-col-props="{ flex: '1' }">
                <a-form-item label="执行容器">
                    <span v-if="form.type == 'app-plugin'" class="c-99">使用所选传统应用容器</span>
                    <a-select v-else v-model="shellConfig.item.container" class="shell-config-control"
                        :placeholder="hasShellContainerOptions ? '请选择执行容器' : '请先在应用配置中添加容器'"
                        :disabled="!hasShellContainerOptions" allow-clear style="width:100%;" @change="changeForm">
                        <a-option v-for="item in shellContainerOptions" :key="item.value" :label="item.label"
                            :value="item.value"></a-option>
                    </a-select>
                </a-form-item>
                <a-form-item label="脚本命令">
                    <a-textarea v-model="shellConfig.item.shell" class="shell-config-control shell-command-input"
                        :auto-size="{ minRows: 8, maxRows: 18 }" :spellcheck="false"
                        placeholder="请输入脚本命令"></a-textarea>
                </a-form-item>
            </a-form>
            <div class="dialog-footer">
                <a-button size="large" @click="resetShellConfig">取消</a-button>
                <a-button type="primary" size="large" @click="saveShellConfig">确定</a-button>
            </div>
        </a-modal>

        <a-modal v-model:visible="domainConfig.show" title="配置域名" :width="1000"
            :body-style="{ padding: 0 }" unmount-on-close @ok="saveDomainConfig"
            @cancel="resetDomainConfig" @close="resetDomainConfig">
            <div class="domain-config-modal-content">
                <form-ingress v-if="domainConfig.show" v-model="domainConfig.ingress"
                    :app-names="app_names" :app-ports="app_ports"
                    :mainapp="option && option.mainapp" :identifie="identifie" />
            </div>
        </a-modal>

        <depend-picker v-model:visible="dependPicker.show" title="选择安装依赖" action-text="选择"
            @select="selectDependFromPicker" @close="closeDependPicker" />

    </div>
</template>

<script>
import jsyaml from "js-yaml";
import filesUpload from './files-upload.vue';
import formIngress from '@/components/form-ingress.vue';
import w7Identifie from "@/components/w7-identifie.vue";
import ManifestConfigTable from '@/components/manifest-config-table.vue';
import ManifestConfigTableColumn from '@/components/manifest-config-table-column.vue';
import dependPicker from '@/components/depend-picker.vue';
import {
    IconCheckCircleFill,
    IconClose,
    IconExclamationCircleFill,
    IconLoading,
    IconPlus,
    IconUpload,
} from '@arco-design/web-vue/es/icon';
import { confirm, messageError, messageWarning } from '@/utils/ui-feedback';
import {
    traditionFormDefaults,
    traditionManifestComputed,
    traditionManifestMethods,
    traditionManifestState,
    traditionValidationRules,
} from '@/components/manifest/tradition-manifest';
import {
    systemImageFormDefaults,
    systemImageManifestMethods,
    systemImageValidationRules,
} from '@/components/manifest/system-image-manifest';
import {
    appPluginFormDefaults,
    appPluginManifestMethods,
    appPluginManifestState,
} from '@/components/manifest/plugin-app-manifest';
import {
    gatewayPluginFormDefaults,
    gatewayPluginManifestMethods,
    gatewayPluginManifestState,
    gatewayPluginValidationRules,
} from '@/components/manifest/gateway-plugin-manifest';
import {
    dockerManifestMethods,
    normalizeDockerApplicationType,
} from '@/components/manifest/docker-manifest';
import {
    helmFormDefaults,
    helmManifestComputed,
    helmManifestMethods,
    helmManifestState,
} from '@/components/manifest/helm-manifest';
import {
    startParamsFormDefaults,
    startParamsManifestMethods,
    startParamsManifestState,
    startParamsManifestWatch,
} from '@/components/manifest/start-params-manifest';
import {
    shellFormDefaults,
    shellManifestComputed,
    shellManifestMethods,
    shellManifestState,
    shellManifestWatch,
} from '@/components/manifest/shell-manifest';
import {
    dependencyFormDefaults,
    dependencyManifestMethods,
    dependencyManifestState,
    dependencyManifestWatch,
} from '@/components/manifest/dependency-manifest';
import {
    applicationConfigFormDefaults,
    applicationConfigManifestMethods,
    applicationConfigManifestState,
    applicationConfigManifestWatch,
} from '@/components/manifest/application-config-manifest';
import {
    manifestMetadataFormDefaults,
    manifestMetadataMethods,
    manifestMetadataState,
    manifestMetadataValidationRules,
} from '@/components/manifest/manifest-metadata';
import {
    manifestSourceMethods,
    manifestSourceState,
} from '@/components/manifest/manifest-source';
import {
    manifestYamlMethods,
    manifestYamlState,
} from '@/components/manifest/manifest-yaml';

export default {
    emits: ['writefile', 'tradition-nginx-gateway-change'],
    props: [
        'data',
        'submiting',
        'option',
        'identifie',
        'version_id',
    ],
    components: {
        filesUpload,
        formIngress,
        w7Identifie,
        ManifestConfigTable,
        ManifestConfigTableColumn,
        dependPicker,
        IconCheckCircleFill,
        IconClose,
        IconExclamationCircleFill,
        IconLoading,
        IconPlus,
        IconUpload,
    },
    data() {
        return {
            json: {},
            yaml: '',
            form: {
                type: 'docker',
                ...manifestMetadataFormDefaults(),
                ...gatewayPluginFormDefaults(),
                ...traditionFormDefaults(),
                ...systemImageFormDefaults(),
                ...appPluginFormDefaults(),
                ...startParamsFormDefaults(),
                ...dependencyFormDefaults(),
                ...shellFormDefaults(),
                ...applicationConfigFormDefaults(),
                helm: helmFormDefaults(),

            },

            rules: {
                ...manifestMetadataValidationRules(this),
                ...gatewayPluginValidationRules(),
                ...traditionValidationRules(this),
                ...systemImageValidationRules(this),
            },
            otherData: {},

            ...manifestMetadataState(),
            ...manifestSourceState(),
            ...manifestYamlState(),
            ...traditionManifestState(),
            ...appPluginManifestState(),
            ...gatewayPluginManifestState(),
            ...startParamsManifestState(),
            ...dependencyManifestState(),
            ...shellManifestState(),
            ...applicationConfigManifestState(),
            ...helmManifestState(),

            initialApplicationType: '',
            currentApplicationType: '',
            savedEditorState: '',
            savedManifestJSON: null,
        }
    },
    created() {
        this.getDependsList();
        this.getTraditionList();
        this.initManifestYaml();
        this.init(this.data);
        const initializationRequests = [];
        if (!this.option?.pureManifest) {
            if (['tradition', 'gateway-plugin', 'system-image'].includes(this.form.type)) {
                initializationRequests.push(this.loadFormulaSetting());
            }
        } else {
            this.otherData.required = Boolean(this.option?.required);
        }
        this.markManifestSavedAfter(initializationRequests);

    },

    computed: {
        ...traditionManifestComputed,
        ...helmManifestComputed,
        ...shellManifestComputed,
    },
    watch: {
        ...startParamsManifestWatch,
        ...dependencyManifestWatch,
        ...shellManifestWatch,
        ...applicationConfigManifestWatch,
        data() {
            this.init(this.data);
            const initializationRequests = [];
            if (!this.option?.pureManifest && ['tradition', 'gateway-plugin', 'system-image'].includes(this.form.type)) {
                initializationRequests.push(this.loadFormulaSetting());
            }
            this.markManifestSavedAfter(initializationRequests);
        },
    },
    beforeUnmount() {
        this.cleanupManifestYaml();
    },
    methods: {
        ...traditionManifestMethods,
        ...systemImageManifestMethods,
        ...appPluginManifestMethods,
        ...gatewayPluginManifestMethods,
        ...dockerManifestMethods,
        ...helmManifestMethods,
        ...startParamsManifestMethods,
        ...dependencyManifestMethods,
        ...shellManifestMethods,
        ...applicationConfigManifestMethods,
        ...manifestMetadataMethods,
        ...manifestSourceMethods,
        ...manifestYamlMethods,
        getEditorStateSnapshot() {
            return JSON.stringify({
                form: this.form,
                json: this.json,
                zip: this.zip,
                web: this.web,
            });
        },
        markManifestSaved() {
            this.savedEditorState = this.getEditorStateSnapshot();
            this.savedManifestJSON = JSON.parse(JSON.stringify(this.json || {}));
        },
        markManifestSavedAfter(requests = []) {
            const token = (this._savedEditorStateToken || 0) + 1;
            this._savedEditorStateToken = token;
            Promise.allSettled(requests.filter(Boolean)).then(() => {
                this.$nextTick(() => {
                    if (this._savedEditorStateToken == token) this.markManifestSaved();
                });
            });
        },
        hasUnsavedChanges() {
            return !this.savedEditorState
                || this.savedEditorState !== this.getEditorStateSnapshot();
        },
        getSavedManifest() {
            const manifest = this.savedManifestJSON || this.json || {};
            return JSON.parse(JSON.stringify(manifest));
        },
        filterAnnotationsForType(annotation = {}, type = this.form.type) {
            let filtered = this.filterTraditionAnnotations(annotation, type);
            filtered = this.filterSystemImageAnnotations(filtered, type);
            return this.filterGatewayPluginAnnotations(filtered, type);
        },
        cleanupTypeSpecificManifest(previousType, nextType) {
            this.json.application = this.json.application || {};
            this.json.platform = this.json.platform || {};
            this.json.application.annotation = this.filterAnnotationsForType(
                this.json.application.annotation || {},
                nextType,
            );

            this.cleanupSystemImageType(previousType, nextType);
            this.cleanupTraditionType(previousType, nextType);
            this.cleanupAppPluginType(previousType, nextType);
            this.cleanupHelmType(previousType, nextType);
            this.cleanupGatewayPluginType(previousType, nextType);
        },
        init(data) {
            if (!data) { return }
            this.json = jsyaml.load(data);
            this.beginStartParamsInitialization(this.json);

            this.json.platform = this.json?.platform || {}

            this.ensureManifestBaseInfo(this.json);
            this.initJSON();
            if (!this.finishStartParamsInitialization()) {
                this.changeForm();
            }
            this.computedAppPort();
        },

        initJSON() {
            this._initializing = true;
            let j = this.json;
            if (j.application) {
                const applicationType = normalizeDockerApplicationType(j.application.type);
                if (applicationType === 'environment') {
                    this.form.type = 'tradition';
                } else {
                    this.form.type = applicationType;
                }
            }
            this.initialApplicationType = this.form.type;
            this.currentApplicationType = this.form.type;
            for (let i in j.bindings) {
                let o = j.bindings[i];
                if (!o?.menu?.length) {
                    o.menu = [{ displayorder: 0, do: 'home', title: '首页', icon: 'a-shouye', is_default: 1 }];
                }
            }

            this.initManifestMetadata(j);
            this.initGatewayPluginManifest(j?.platform || {});
            this.applyTraditionAnnotationForm(j?.application?.annotation || {});
            this.applySystemImageAnnotationForm(j?.application?.annotation || {});
            this.initSystemImageManifest(j?.platform || {});

            this.initManifestSource(j);

            if (j.platform) {
                this.initAppPluginManifest(j.platform);
                this.initApplicationConfig(j.platform);
                this.initShellManifest(j.platform);
                this.initStartParams(j.platform);
                this.initDependencies(j.platform);
                this.applyHelmManifestForm(j.platform);
            }
            this.syncTraditionDependency();
            this.syncSysboxDependency();
            this._initializing = false;
        },
        submit(otherData, callback, errorCallback) {
            if (this.submitStartParamsOnly(otherData, callback)) return;

            const fail = () => {
                if (typeof errorCallback == 'function') errorCallback();
            };

            this.$refs.formref.validate(async (errors) => {
                if (errors) {
                    messageWarning(this.form.type == 'tradition'
                        ? '请检查传统应用配置中的错误项'
                        : '必填项不能为空');
                    fail();
                    return;
                }

                const sysboxContainerNameError = this.validateSysboxContainerName();
                if (sysboxContainerNameError) {
                    messageWarning(sysboxContainerNameError);
                    fail();
                    return;
                }

                if (this.form.type == 'gateway-plugin') {
                    if (!this.form.gatewayPluginSupportGlobal && !this.form.gatewayPluginSupportRule) {
                        messageWarning('请至少选择一种支持范围');
                        fail();
                        return;
                    }
                    const defaultConfig = this.parseGatewayPluginDefaultConfig(true);
                    if (defaultConfig === null) {
                        fail();
                        return;
                    }
                    this.json.platform.gatewayPlugin.defaultConfig = defaultConfig;
                }

                try {
                    this.syncTraditionDependency();
                    this.syncSysboxDependency();
                    await this.saveFormulaTypeSetting();
                    await this.ensureDefaultTypeTags();
                    this.changeForm();
                    if (this.json.application) {
                        this.json.application.annotation = this.filterAnnotationsForType(
                            this.json.application.annotation || {},
                        );
                    }
                } catch (error) {
                    messageError(error?.response?.data?.error || error?.message || '制品配置保存失败');
                    fail();
                    return;
                }

                if (this.form.type == 'helm') {
                    this.cleanupHelmPlatformForSubmit();
                } else if (this.form.type == 'app-plugin') {
                    delete this.json.platform.volumeClaimTemplates;
                    delete this.json.platform.ingress;
                    delete this.json.platform.runtimeClassName;
                    this.applyPlatformShells();
                } else if (this.form.type == 'tradition') {
                    this.form.startParams = this.traditionStartParams();
                    this.ensureTraditionContainerDefaults();
                    this.applyPlatformShells();
                    this.syncTraditionIngress();
                } else if (this.form.type == 'docker') {
                    this.applyDockerPlatform();
                }

                if (this.form.type != 'gateway-plugin') {
                    if (this.form.type == 'system-image') {
                        this.ensureSystemImageContainer();
                    }
                    this.json.platform.startParams = this.serializeStartParams();
                }
                this.yaml = jsyaml.dump(this.json);

                this.$emit('complete', this.json, this.yaml, otherData, callback, errorCallback);
            });
        },
        changeFormtype(nextType) {
            const previousType = this.currentApplicationType || this.initialApplicationType;
            if (!nextType || nextType == previousType) { return }
            const typeNames = {
                docker: '原生应用',
                'app-plugin': '应用插件',
                helm: 'K8sYaml',
                tradition: '传统应用',
                'system-image': '系统镜像',
                'gateway-plugin': '网关插件',
            };
            this.form.type = previousType;
            confirm({
                title: '切换应用类型',
                content: `切换为“${typeNames[nextType] || nextType}”后，当前类型的专属配置将被清理，且无法通过再次切换恢复。是否继续？`,
                confirmButtonText: '继续切换',
                cancelButtonText: '取消',
                onOk: () => this.applyFormTypeChange(previousType, nextType),
            });
        },
        applyFormTypeChange(previousType, nextType) {
            this.form.type = nextType;
            this.cleanupTypeSpecificManifest(previousType, nextType);
            this.syncApplicationBuiltInStartParams();
            this.json.application = this.json.application || {};
            this.json.application.annotation = this.filterAnnotationsForType(
                this.json.application.annotation || {},
                nextType,
            );
            this.currentApplicationType = nextType;
            if (this.form.type == 'gateway-plugin') {
                this.form.once = true;
            }
            if (['tradition', 'gateway-plugin', 'system-image'].includes(this.form.type)) {
                this.loadFormulaSetting().catch(() => { });
            }
            this.syncManifestSourceForType();
            if (this.form.type == 'app-plugin') {
                this.getStart();
            }
            if (this.form.type == 'system-image') {
                this.syncSystemImageConfig();
            }
            this.syncTraditionDependency();
            this.syncSysboxDependency();

            this.changeForm();
        },

        replaceZpk(json) {
            json.application = this.json.application;
            json.source = this.json.source || {};
            json.platform = json.platform || {};
            this.json = json;
            this.initJSON();
            this.changeForm();
        },
        changeForm() {


            if (this._initializing) return;

            let j = this.json;
            this.applyApplicationMetadata(j.application);
            this.applyGatewayPluginManifest();
            j.platform = j.platform || {};

            if (this.form.type == 'system-image') {
                this.applySystemImageManifest();
            } else if (this.form.type == 'tradition') {
                this.applyTraditionManifest();
            } else {
                this.cleanupInactiveSysboxRuntime(j.platform);
            }
            this.syncSysboxDependency();

            j.platform = this.applyAppPluginManifest(j.platform);

            if (j.platform && this.form.type != 'gateway-plugin') {
                this.applyPlatformBaseInfo(j.platform);
                this.applyManifestDependencies(j.platform);
                j.platform.startParams = this.serializeStartParams();

                this.applyHelmPlatform();
                if (this.form.type != 'helm') {
                    this.applyPlatformShells();
                }
                if (this.form.type == 'docker') {
                    this.applyDockerPlatform();
                }
            }
            this.setYaml();
        },
    },
}
</script>
<style scoped>
.manifest-start-params-only > .bg-white:not(.manifest-base-section):not(.manifest-start-params-section):not(.manifest-submit-section) {
    pointer-events: none;
    opacity: 0.6;
}

.manifest-start-params-only .manifest-start-params-readonly {
    pointer-events: none;
    opacity: 0.6;
}

.manifest-start-params-only .manifest-start-params-section > :deep(.arco-form-item):not(.manifest-start-param-editor) {
    pointer-events: none;
    opacity: 0.6;
}

.tradition-config-spin {
    display: block;
    width: 100%;
}


.nginx-template-field {
    width: 500px;
}

.nginx-template-input {
    position: relative;
    width: 500px;
}

.nginx-template-example-entry {
    position: absolute;
    z-index: 2;
    top: 0;
    right: 0;
    padding: 6px 6px;
    height: 30px;
    background: var(--color-fill-2);
}

.nginx-template-example-entry:hover {
    background: var(--color-fill-3);
}

.nginx-template-example-panel {
    display: flex;
    flex-direction: column;
    gap: 16px;
}

.nginx-template-example-placeholders {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px;
    color: #86909c;
    font-size: 13px;
}

.nginx-template-example-placeholder {
    font-family: SFMono-Regular, Consolas, "Liberation Mono", monospace;
}

.nginx-template-example-code {
    box-sizing: border-box;
    min-height: 480px;
    margin: 0;
    padding: 16px;
    overflow: auto;
    border: 1px solid #e5e6eb;
    border-radius: 6px;
    color: #1d2129;
    background: #f7f8fa;
    font-family: SFMono-Regular, Consolas, "Liberation Mono", monospace;
    font-size: 13px;
    line-height: 1.65;
    tab-size: 4;
    white-space: pre;
}

.nginx-template-example-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
}

.env-sel .env-item {
    box-sizing: border-box;
    min-width: 80px;
    height: 80px;
    padding: 10px 10px 0;
    margin: 0 10px 10px 0;
    background-color: #f5f7fa;
}

.env-sel .env-item img {
    width: 40px;
    height: 40px;
    margin-bottom: 8px;
}

.env-sel .env-item.active {
    box-shadow: 0 0 3px 1px #badefd;
    background-color: #f0f2fb;
}

.com-line {
    padding-bottom: 20px;
    border-bottom: 1px solid #E7E7E7;
}

.manifest-field-stack {
    width: 100%;
    min-width: 0;
}

.domain-config-modal-content {
    min-width: 0;
    padding: 24px;
}

.start-param-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    width: 100%;
}

.start-param-services {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px 20px;
    min-width: 0;
}

.start-param-services :deep(.arco-checkbox) {
    margin-right: 0;
}

.install-depend-table {
    width: 100%;
    table-layout: fixed;
}

.install-depend-name-col {
    width: 36%;
}

.install-depend-sub-col {
    width: 34%;
}

.install-depend-required-col {
    width: 10%;
}

.install-depend-identifie-col {
    width: 12%;
}

.install-depend-action-col {
    width: 8%;
}

.install-depend-table td {
    box-sizing: border-box;
}

.install-depend-app-cell {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    min-width: 0;
}

.install-depend-app-main {
    min-width: 0;
}

.install-depend-app-name,
.install-depend-app-identifie {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.install-depend-app-name {
    color: #333;
}

.install-depend-app-identifie {
    margin-top: 2px;
    color: #999;
    font-size: 12px;
}

.install-depend-table :deep(.arco-select),
.install-depend-table :deep(.arco-input-wrapper) {
    width: 100%;
}

.install-depend-table td:nth-child(3),
.install-depend-table td:last-child {
    white-space: nowrap;
}

.shell-config-table {
    width: 100%;
    table-layout: fixed;
}

.shell-config-table :deep(.arco-input-wrapper),
.shell-config-table :deep(.arco-select) {
    width: 100%;
}

.shell-task-cell {
    display: flex;
    align-items: center;
    gap: 12px;
    white-space: nowrap;
    min-width: 0;
}

.shell-task-cell .c-99 {
    overflow: hidden;
    text-overflow: ellipsis;
}

.shell-config-form {
    width: 100%;
}

.shell-config-form :deep(.arco-form-item) {
    width: 100%;
    margin-bottom: 20px;
}

.shell-config-form :deep(.arco-form-item-layout-horizontal) {
    display: flex;
    align-items: flex-start;
    width: 100%;
}

.shell-config-form :deep(.arco-form-item-label-col) {
    flex: 0 0 90px;
    width: 90px;
    max-width: 90px;
}

.shell-config-form :deep(.arco-form-item-wrapper-col) {
    flex: 1 1 auto;
    width: calc(100% - 90px);
    min-width: 0;
}

.shell-config-form :deep(.arco-form-item-content-wrapper),
.shell-config-form :deep(.arco-form-item-content) {
    width: 100%;
    min-width: 0;
}

.shell-config-control,
.shell-config-form :deep(.arco-select),
.shell-config-form :deep(.arco-textarea-wrapper) {
    width: 100%;
}

.shell-command-input {
    min-height: 180px;
}

.shell-command-input :deep(.arco-textarea) {
    min-height: 180px;
}

.upfilebox {
    width: 280px;
    height: 160px;
    border: 1px solid #dcdcdc;
    border-radius: 3px;
    position: relative;
}

.upfilebox .uploadicon {
    font-size: 40px;
}

.upfilebox .uploadbtn {
    margin-top: 10px;
    height: 32px;
    padding: 0 18px;
    background: #f3f3f3;
    border-radius: 3px;
}

.upfilebox .uploadbtn .uploadicon {
    font-size: 14px;
    margin-right: 4px;
}

.upfilebox .mask {
    display: none;
}

.upfilebox:hover .mask {
    display: flex;
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
}

.app-icon {
    width: 64px;
    height: 64px;
    position: relative;
    border-radius: 8px;
}

.app-icon .img {
    width: 64px;
    height: 64px;
    display: block;
    border-radius: 8px;
}

.app-icon input[type='file'] {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 2;
    min-width: 0;
    opacity: 0;
    cursor: pointer;
}

.app-icon input[type='file']::-webkit-file-upload-button {
    display: none;
}


.tag-cpn {
    width: 500px;
    box-sizing: border-box;
    border-radius: 4px;
    padding: 7px 7px 1px;
    border: 1px solid #dcdfe6;
}

.tag-cpn.active {
    border-color: #409eff;
}

.tag-cpn .tag {
    height: 24px;
    margin-right: 6px;
    margin-bottom: 6px;
}

.tag-cpn .input {
    width: -webkit-min-content;
    width: min-content;
    height: 24px;
    min-width: 60px;
    margin-bottom: 6px;
}

.tag-cpn .input input {
    width: 100%;
    height: 100%;
    display: block;
    border: 0;
    outline: 0;
    padding: 0 6px;
}

.tag-cpn .input input::placeholder {
    color: #999;
}

.logobox {
    width: 64px;
    height: 64px;
    margin-right: 10px;
    border-radius: 4px;
    overflow: hidden;
    box-sizing: border-box;
    position: relative;
    border: 1px solid #f1f1f1;
}

.logobox .icon {
    width: 100%;
    height: 100%;
    position: absolute;
    top: 0;
    left: 0;
    z-index: 1;
}

.logobox input[type='file'] {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 2;
    min-width: 0;
    opacity: 0;
    cursor: pointer;
}

.content {
    padding: 20px;
    height: 100%;
    box-sizing: border-box;
}

.ml-4 {
    margin-left: 4px;
}

.ml-20 {
    margin-left: 20px;
}

.title {
    font-size: 14px;
    font-weight: bold;
}

.title::before {
    display: block;
    content: " ";
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #0052D9;
    margin-right: 12px;
}

.box {
    padding: 0 20px;
    overflow: auto;
    position: relative;
}

.box+.box {
    border-left: 1px solid #ccc;
}

.form {
    padding: 0;
}

.box :deep(.pre) {
    margin: 0;
    height: 100%;
    font-size: 16px;
    max-width: 100%;
    overflow: auto;
    background: #282c34;
}

.box :deep(input::-webkit-outer-spin-button),
.box :deep(input::-webkit-inner-spin-button) {
    -webkit-appearance: none;
}

.box :deep(input[type="number"]) {
    appearance: textfield;
}



.branch {
    margin-left: 30px;
    box-sizing: border-box;
    width: 30px;
    height: 28px;
    border: 1px dashed #ccc;
    border-right: 0;
    border-top: 0;
    position: relative;
    top: -10px;
}

.branch::after {
    content: " ";
    position: absolute;
    border-left: 1px dashed #ccc;
    height: 18px;
    top: 100%;
    left: -1px;
}

.branch.last::after {
    display: none;
}

.icon {
    border: 1px solid #f0f0f0;
    box-sizing: border-box;
    width: 64px;
    height: 64px;
}

.icon:hover i {
    color: #2d5fff;
}

.icon:hover {
    border-color: #2d5fff;
}

.selicon {
    box-sizing: border-box;
    text-align: center;
    line-height: 36px;
    width: 36px;
    height: 36px;
    border: 1px solid #f0f0f0;
    vertical-align: middle;
}

.iconbox {
    width: 790px;
    height: 500px;
    overflow: auto;
}

.iconbox::-webkit-scrollbar {
    width: 10px;
}

.iconbox::-webkit-scrollbar-track {
    background: transparent;
}

.iconbox::-webkit-scrollbar-thumb {
    background: #eee;
    border-radius: 6px;
}

.copybtn {
    display: block;
    padding: 8px 15px;
    border-radius: 4px;
    background: #ffffff;
    border: 0;
    outline: 0;
    font-size: 14px;
    line-height: 20x;
    cursor: pointer;
    margin-left: 10px;
}

.elseoption {
    padding: 6px 12px;
    cursor: pointer;
}

:deep(pre .hljs) {
    height: 100%;
    box-sizing: border-box;
}

.menulocation {
    padding: 8px 0;
}

.menulocation img {
    width: 48px;
    height: 40px;
}

.menu-single-location {
    line-height: 1.2;
    border-left: 4px solid #0052D9;
    padding-left: 4px;
}

.command-upfile {
    position: relative;
}

.command-upfile input[type='file'] {
    min-width: 30px;
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 1;
    opacity: 0;
}

.show-on-hover-container .show-on-hover {
    display: none;
}

.show-on-hover-container:hover .show-on-hover {
    display: block;
}
</style>
<style>
.menulocation .arco-radio {
    height: 20px;
}

.menulocation .arco-radio-label {
    padding-left: 4px;
    font-size: 12px;
}

.support-group .arco-checkbox {
    height: 18px;
    width: 120px;
    margin-right: 20px;
    margin-bottom: 10px;
}

.manifest-form .arco-form-item-label {
    color: rgba(0, 0, 0, 0.9);
}

.manifest-form .arco-form-item-layout-horizontal>.arco-form-item-label-col {
    flex: 0 0 120px !important;
    max-width: 120px !important;
}

.manifest-form .arco-form-item-layout-horizontal>.arco-form-item-wrapper-col {
    flex: 1 1 0 !important;
    max-width: calc(100% - 120px) !important;
    min-width: 0;
}

.manifest-form .arco-form-item-content-wrapper,
.manifest-form .arco-form-item-content {
    width: 100%;
    min-width: 0;
}

</style>
