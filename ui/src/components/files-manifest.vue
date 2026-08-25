<template>
    <div class="df content">
        <div class="fc" style="padding-left:0; overflow:auto;">
            <div>
                <a-form ref="formref" :model="form" :rules="rules" label-align="left"
                    :label-col-props="{ span: 5, flex: '0 0 120px' }"
                    :wrapper-col-props="{ span: 19, flex: '1' }" class="form manifest-form">
                    <div class="bg-white com-line df">
                        <div class="fc">
                            <div class="c-00-6 df ai-c">基础配置</div>
                            <a-form-item class="mt-16" label="名称" :field="option?.pureManifest ? 'name' : ''">
                                <a-input v-model="form.name" size="large" style="width:500px;" @change="changeForm"
                                    placeholder="请输入"></a-input>
                            </a-form-item>
                            <a-form-item label="标识" :field="option?.pureManifest ? 'identifie' : ''">
                                <div class="df jc-b" style="width:500px;">
                                    <w7-identifie v-model:author="form.author" v-model:identifie="form.identifie"
                                        @change="changeForm" disabled />
                                </div>
                            </a-form-item>
                            <a-form-item label="描述" field="description">
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
                    <div class="bg-white mt-20 pb-24">
                        <div class="c-00-6 df ai-c">
                            <span class="">代码配置</span>
                        </div>
                        <div class="mt-16">
                            <a-form-item label="类型">
                                <div class="df df-c fc">

                                    <a-radio-group v-model="form.type" @change="changeFormtype">
                                        <a-radio value="docker">原生应用</a-radio>
                                        <a-radio value="tradition">传统应用</a-radio>
                                        <a-radio value="helm">K8sYaml</a-radio>
                                        <a-radio value="environment">运行环境</a-radio>
                                        <a-radio value="system-image">系统镜像</a-radio>
                                        <a-radio value="gateway-plugin">网关插件</a-radio>
                                    </a-radio-group>

                                    <div v-if="form.type == 'gateway-plugin'" class="greybox mt-20"
                                        style="margin-bottom:0;">
                                        <div class="greybox-title">网关插件配置</div>
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

                                    <a-spin v-if="form.type == 'environment'" :loading="formulaSettingLoading"
                                        class="environment-config-spin">
                                        <a-alert type="info" show-icon class="zpk-primary-alert mt-20 mb-20"
                                            title="说明" :closable="false">
                                            <div class="registry-alert-item">1. 运行环境会作为独立应用安装，同时保存为站点可选的运行环境模板；运行方式固定为 Deployment。</div>
                                            <div class="registry-alert-item mt-6">2. 新建或升级站点并选择此运行环境时，系统会根据模板准备站点需要的运行环境。这里的修改只会用于之后新建的环境，已创建的环境不会自动更新。</div>
                                            <div class="registry-alert-item mt-6">3. 独立安装时通过“环境版本”启动参数替换运行容器镜像中的 {version}；创建站点环境时也会使用同一模板。</div>
                                            <div class="registry-alert-item mt-6">4. 系统会自动配置站点代码目录 /www/wwwroot 和服务目录 /www/server 的共享存储。</div>
                                            <div class="registry-alert-item mt-6">5. 环境容器的启动命令可在页面下方“启动命令”中配置。</div>
                                            <div class="registry-alert-item mt-6">6. 页面下方“脚本配置”中的安装、升级脚本只在安装或升级此制品时执行，站点管理新建环境时不会再次执行。</div>
                                            <div class="registry-alert-item mt-6">7. 环境准备完成后，系统会使用 Nginx 模板配置站点并完成绑定。</div>
                                        </a-alert>
                                        <div class="greybox" style="margin-bottom:0;">
                                            <div class="greybox-title">运行环境配置</div>
                                            <a-form-item label="环境语言" field="environmentImageLanguage" required
                                                style="margin-bottom:18px;">
                                                <a-select v-model="form.environmentImageLanguage" size="large"
                                                    style="width:500px;" placeholder="请选择环境语言" allow-search
                                                    @change="changeEnvironmentLanguage">
                                                    <a-option v-for="option in environmentLanguageOptions"
                                                        :key="option.value" :value="option.value"
                                                        :label="option.label">
                                                        {{ option.label }}
                                                    </a-option>
                                                </a-select>
                                            </a-form-item>
                                            <a-form-item label="镜像地址" field="environmentImageTemplate" required
                                                style="margin-bottom:18px;">
                                                <div class="df df-c">
                                                    <a-input v-model="form.environmentImageTemplate" size="large"
                                                        style="width:500px;" placeholder="例如 php:{version}-fpm-alpine"
                                                        @change="syncEnvironmentRuntimeConfig" />
                                                    <span class="c-99 mt-6">使用 {version} 作为运行环境版本占位符。</span>
                                                </div>
                                            </a-form-item>
                                            <a-form-item label="环境版本" field="environmentImageVersion" required
                                                style="margin-bottom:18px;">
                                                <div class="df df-c">
                                                    <a-input-tag v-model="form.environmentImageVersion" size="large"
                                                        style="width:500px;" placeholder="输入版本后按回车，例如 8.1"
                                                        allow-clear unique-value @change="syncEnvironmentVersionConfig" />
                                                    <span class="c-99 mt-6">每次输入一个语言版本并按回车，可添加多个版本，例如 7.4、8.1。</span>
                                                </div>
                                            </a-form-item>
                                            <a-form-item label="系统重启还原" style="margin-bottom:18px;">
                                                <div class="df df-c" style="align-items:flex-start;">
                                                    <a-switch v-model="form.environmentSystemRebootRestore"
                                                        @change="syncEnvironmentRuntimeConfig" />
                                                    <span class="c-99 mt-6">关闭后使用持久存储保留容器系统层。</span>
                                                </div>
                                            </a-form-item>
                                            <a-form-item label="Nginx 模板" field="environmentNginxVhostTemplate" required
                                                style="margin-bottom:0;">
                                                <div class="nginx-template-field">
                                                    <div class="nginx-template-input">
                                                        <a-textarea v-model="form.environmentNginxVhostTemplate"
                                                            class="nginx-template-textarea"
                                                            :auto-size="{minRows:10, maxRows:20}"
                                                            :spellcheck="false" placeholder="请输入 Nginx vhost 模板"
                                                            style="width:500px;" />
                                                        <a-button size="mini" type="text"
                                                            class="nginx-template-example-entry"
                                                            @click="nginxTemplateExampleVisible = true">
                                                            查看完整示例
                                                        </a-button>
                                                    </div>
                                                    <span class="c-99 mt-6">用于为使用此运行环境的站点生成访问配置。</span>
                                                </div>
                                            </a-form-item>
                                        </div>
                                    </a-spin>

                                    <a-alert v-if="form.type == 'system-image'" type="info" show-icon
                                        class="zpk-primary-alert mt-20 mb-20" title="系统镜像说明"
                                        :closable="false">
                                        <div class="registry-alert-item">1. 系统镜像用于创建具备完整系统环境的轻量虚拟机，可以像普通主机一样运行系统服务。</div>
                                        <div class="registry-alert-item mt-6">2. 底层系统数据会持久保存，实例重启后已安装的软件、系统配置和用户数据不会丢失。</div>
                                    </a-alert>
                                    <div v-if="form.type == 'system-image'" class="greybox"
                                        style="margin-bottom:0;">
                                        <div class="greybox-title">系统镜像配置</div>
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

                            <a-form-item v-if="form.type == 'tradition'" label="环境类型">
                                <div class="df df-ww env-sel">
                                    <div v-for="item in environmentList" :key="item.identifie"
                                        :class="{ 'active': form.environmentName == item.identifie }"
                                        @click="changeEnv(item)" class="env-item df df-c ai-c cursor">
                                        <img :src="environmentIcon(item.icon)" class="img" alt="" />
                                        <div class="lh-1">{{ item.name }}</div>
                                        <input type="radio" v-model="form.environmentName" :value="item.identifie"
                                            style="display:none;" />
                                    </div>
                                </div>
                            </a-form-item>
                            <a-form-item v-if="form.type == 'tradition'" label="环境版本">
                                <a-select v-model="form.environmentVersion" placeholder="请选择" @change="changeForm"
                                    size="large" style="width:500px;">
                                    <a-option
                                        v-for="(item, index) in environmentList?.find?.(i => i.identifie == form.environmentName)?.versions || []"
                                        :key="index" :label="item" :value="item"></a-option>
                                </a-select>
                            </a-form-item>
                            <a-form-item v-if="form.type == 'tradition'" label="应用类型">
                                <a-radio-group v-model="form.installType" @change="changeForm">
                                    <a-radio :value="traditionInstallTypes.site">整站应用</a-radio>
                                    <a-radio :value="traditionInstallTypes.extension">扩展</a-radio>
                                </a-radio-group>
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
                                    <a-tooltip v-if="form.type == 'tradition'"
                                        content="整站应用请从项目根目录打包，扩展应用请从扩展所在目录打包；压缩包根目录就是安装内容，请不要包含外层项目目录。安装后解压到 /www/wwwroot/&lt;站点域名&gt;。"
                                        position="top">
                                        <icon-exclamation-circle-fill class="fs-16 c-99 ml-4" />
                                    </a-tooltip>
                                    <a-tooltip v-else-if="form.type == 'environment'"
                                        content="代码包为可选配置。请将项目根目录内容直接压缩，不要包含外层项目目录，例如：cd 项目目录 && zip -r app.zip .。安装环境时会在预安装阶段解压到 /www/wwwroot/&lt;站点域名&gt;，并随当前版本发布。"
                                        position="top">
                                        <icon-exclamation-circle-fill class="fs-16 c-99 ml-4" />
                                    </a-tooltip>
                                </div>
                                <div v-if="zip.hasDockerfile === false && !['environment', 'tradition'].includes(form.type)" class="c-red mt-10">没有检测到Dockerfile文件，请重新上传
                                </div>
                            </a-form-item>

                            <slot></slot>

                            <a-form-item v-if="['environment', 'system-image'].includes(form.type)" label="CMD">
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

                    <div v-if="form.type != 'helm' && form.type != 'tradition' && form.type != 'environment' && form.type != 'gateway-plugin' && form.type != 'system-image'" class="bg-white com-line mt-20">
                        <div class="mt-16">
                            <a-form-item label="应用配置">
                                <div v-if="form.containers && form.containers.length">
                                    {{form.containers.map(i => i.name).join(', ')}}</div>
                                <div v-else>无</div>
                                <span class="cursor c-blue ml-10" @click="openAppset">编辑</span>
                            </a-form-item>


                            <a-form-item v-if="form.type != 'helm' && form.type != 'environment' && form.type != 'system-image'" label="域名设置" class="mt-16">
                                <div v-if="form.ingress && form.ingress.length">
                                    {{ form.ingress.map(item => item.name || '未命名配置').join(', ') }}
                                </div>
                                <div v-else>无</div>
                                <span class="cursor c-blue ml-10" @click="openDomainConfig">编辑</span>
                            </a-form-item>
                        </div>
                    </div>

                    <div class="bg-white com-line mt-20">
                        <a-form-item v-if="form.type != 'gateway-plugin'" class="mt-16" label="启动参数" field="startParams">
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
                                                <span v-if="!isSystemImageFixedStartParam(record) && !isEnvironmentFixedStartParam(record)" class="c-blue cursor handle" @click="openSpDesc(record)">编辑描述</span>
                                                <span v-if="!isSystemImageFixedStartParam(record) && !isEnvironmentFixedStartParam(record)" class="c-blue cursor handle"
                                                    @click="form.startParams.splice(index, 1); getStart();">删除</span>
                                            </template>
                                        </manifest-config-table-column>
                                    </template>
                                </manifest-config-table>
                            </div>
                        </a-form-item>

                        <a-form-item class="mt-20" label="安装依赖">
                            <manifest-config-table :rows="form.depends" table-class="install-depend-table"
                                :add-text="form.type == 'environment' ? '' : '添加安装依赖'"
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
                                                <a-button v-if="!isEnvironmentFixedDependency(record)" type="text" size="mini"
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
                                                :disabled="!record.identifie || isEnvironmentFixedDependency(record)">
                                                <a-option v-for="(name, identifie) in getSubDependsOptions(index)"
                                                    :key="identifie" :label="name" :value="identifie"></a-option>
                                            </a-select>
                                        </template>
                                    </manifest-config-table-column>
                                    <manifest-config-table-column data-index="required" title="必须安装" width="10%">
                                        <template #cell="{ record }">
                                            <a-switch v-model="record.required" :disabled="isEnvironmentFixedDependency(record)"
                                                @change="changeForm" />
                                        </template>
                                    </manifest-config-table-column>
                                    <manifest-config-table-column data-index="name" title="标识" width="12%">
                                        <template #cell="{ record }">{{ record.identifie }}</template>
                                    </manifest-config-table-column>
                                    <manifest-config-table-column title="操作" width="8%">
                                        <template #cell="{ index }">
                                            <span v-if="!isEnvironmentFixedDependency(form.depends[index])" class="c-blue cursor"
                                                @click="form.depends.splice(index, 1); changeForm();">删除</span>
                                            <span v-else class="c-99">固定</span>
                                        </template>
                                    </manifest-config-table-column>
                                </template>
                            </manifest-config-table>
                        </a-form-item>
                    </div>

                    <div class="bg-white pb-24 mt-20 df ai-c">
                        <a-button v-if="option.pureManifest" :loading="submiting" type="primary" @click="submit(otherData)">确定提交</a-button>
                        <a-button v-else :loading="submiting" type="primary" @click="submit()">确定提交</a-button>
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

        <a-drawer v-model:visible="nginxTemplateExampleVisible" :width="720" title="Nginx 模板示例"
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
                        v-model:identifie="dependForm.identifie_last" @change="onChange" :author-disabled="true" />
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
                <div class="registry-alert-item">脚本会以独立任务运行，运行环境决定任务使用的镜像、容器 env 和目录挂载。</div>
                <div class="registry-alert-item mt-6">启动参数会同时注入任务的 env 环境，可在脚本中通过环境变量读取。</div>
                <div class="registry-alert-item mt-6">传统应用固定使用所选运行环境的应用容器。</div>
            </a-alert>
            <a-form v-if="shellConfig.item" :model="shellConfig.item" label-align="left"
                class="manifest-dialog-form shell-config-form"
                :label-col-props="{ flex: '0 0 90px' }" :wrapper-col-props="{ flex: '1' }">
                <a-form-item label="运行环境">
                    <span v-if="form.type == 'tradition'" class="c-99">使用所选环境应用容器</span>
                    <a-select v-else v-model="shellConfig.item.container" class="shell-config-control"
                        :placeholder="hasShellContainerOptions ? '请选择运行环境' : '请先在应用配置中添加容器'"
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

        <a-modal v-model:visible="dependPicker.show" title="选择安装依赖" :width="840" :footer="false"
                    modal-class="depend-picker-modal" @close="closeDependPicker">
                    <div class="depend-picker-toolbar">
                        <a-input v-model="dependPicker.keyword" placeholder="搜索制品名称或标识" allow-clear
                            @keydown.enter="searchDependPicker">
                            <template #suffix>
                                <span class="depend-picker-search-action" @click="searchDependPicker">
                                    <icon-search :size="16" />
                                </span>
                            </template>
                        </a-input>
                    </div>
                    <a-tabs v-model:active-key="dependPicker.activeTab" @change="changeDependPickerTab">
                        <a-tab-pane key="local" title="本地仓库"></a-tab-pane>
                        <a-tab-pane key="official" title="官方仓库"></a-tab-pane>
                    </a-tabs>
                    <a-table :loading="dependPicker.loading" :data="dependPickerRows" :pagination="false"
                        row-key="identifie" class="depend-picker-table" @row-click="selectDependFromPicker">
                        <template #columns>
                            <a-table-column title="制品">
                                <template #cell="{ record }">
                                    <div class="depend-picker-product">
                                        <div class="depend-picker-product-name">{{ record.name || record.identifie }}</div>
                                        <div class="depend-picker-product-identifie">{{ record.identifie }}</div>
                                    </div>
                                </template>
                            </a-table-column>
                            <a-table-column title="版本" :width="120">
                                <template #cell="{ record }">
                                    {{ record.version?.name || '-' }}
                                </template>
                            </a-table-column>
                            <a-table-column title="操作" :width="100" align="center">
                                <template #cell="{ record }">
                                    <a-button type="text" size="small" @click.stop="selectDependFromPicker(record)">
                                        选择
                                    </a-button>
                                </template>
                            </a-table-column>
                        </template>
                    </a-table>
                    <div class="depend-picker-footer">
                        <a-pagination v-model:current="dependPicker.page" :page-size="dependPicker.pageSize"
                            :total="dependPickerTotal" @change="changeDependPickerPage" />
                    </div>
                </a-modal>

    </div>
</template>

<script>
import jsyaml from "js-yaml";
import hljs from 'highlight.js';
import filesUpload from './files-upload.vue';
import formIngress from '@/components/form-ingress.vue';
import w7Identifie from "@/components/w7-identifie.vue";
import ManifestConfigTable from '@/components/manifest-config-table.vue';
import ManifestConfigTableColumn from '@/components/manifest-config-table-column.vue';
import myAxios from '../utils/index';
import {
    createEnvironmentAppDependency,
    getEnvironmentAppRootfsAnnotation,
    isEnvironmentAppDependency,
    removeEnvironmentAppCodeStorage,
} from '@/utils/environment-app';
import {
    applyTraditionEnvironmentDependencyStartParams,
    createTraditionEnvironmentDependency,
    normalizeTraditionInstall,
    traditionInstallTypes,
} from '@/utils/tradition-app';
import {
    IconCheckCircleFill,
    IconClose,
    IconExclamationCircleFill,
    IconLoading,
    IconPlus,
    IconSearch,
    IconUpload,
} from '@arco-design/web-vue/es/icon';
import { confirm, messageError, messageSuccess, messageWarning } from '@/utils/ui-feedback';
import emitWujieEvent from '@/utils/wujie-event';

const environmentAnnotationKeys = {
    parent: 'w7.cc/parent',
    imageLanguage: 'w7.cc/image_language',
    imageTemplate: 'w7.cc/image_template',
    imageVersion: 'w7.cc/image_version',
    nginxVhostTemplate: 'w7.cc/nginx_vhost_template',
    systemRebootRestore: 'w7.cc/system-reboot-restore',
};

const systemImageAnnotationKeys = {
    category: 'w7.cc/system_image_category',
    versions: environmentAnnotationKeys.imageVersion,
    rootfs: 'sysbox/rootfs-rw-layer',
};

const isDerivedDependencyReleaseStartParam = item => Boolean(
    item?.hidden && /_RELEASE_NAME$/.test(item?.name || '')
);

const isLegacyTraditionInstallDirectoryStartParam = (item, applicationType) => (
    applicationType === 'tradition' && item?.name === 'CODE_INSTALL_DIRECTORY'
);

const gatewayPluginAnnotationPrefix = 'w7.cc/plugin-';
const gatewayPluginAnnotationKeys = ['w7.cc/official-app'];

const gatewayPluginCategoryOptions = [
    { label: '路由', value: 'route' },
    { label: 'AI', value: 'ai' },
    { label: '认证', value: 'auth' },
    { label: '安全', value: 'security' },
    { label: '流量', value: 'traffic' },
    { label: '转换', value: 'transform' },
    { label: '可观测性', value: 'o11y' },
    { label: '自定义', value: 'custom' },
];

const environmentLanguagePresets = [
    { label: 'PHP', value: 'php' },
    { label: 'Java', value: 'java' },
    { label: 'Node.js', value: 'nodejs' },
    { label: 'Python', value: 'python' },
    { label: 'Go', value: 'go' },
    { label: '.NET', value: 'dotnet' },
    { label: 'Ruby', value: 'ruby' },
    { label: 'Rust', value: 'rust' },
];

const nginxTemplatePlaceholders = [
    '{SERVER_NAME}',
    '{ROOT_DIR}',
    '{K8S_DOMAIN}',
];

const nginxTemplateExample = String.raw`server {
    listen 80;
    server_name {SERVER_NAME};

    root {ROOT_DIR};
    index index.php index.html index.htm;

    access_log /dev/stdout;
    error_log /dev/stderr;

    # 静态文件处理
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
        try_files $uri =404;
    }

    # 隐藏文件保护
    location ~ /\. {
        deny all;
        access_log off;
        log_not_found off;
    }

    # PHP 代理配置
    location ~ ^/(.+\.php)(/.*)?$ {
        # 使用 FastCGI 连接到 PHP-FPM
        fastcgi_pass {K8S_DOMAIN}:9000;

        set $real_scheme $http_x_forwarded_proto;
        if ($real_scheme = "") {
            set $real_scheme $scheme;
        }

        # 关键 FastCGI 参数
        include fastcgi_params;
        fastcgi_split_path_info ^(.+\.php)(/.+)$;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        fastcgi_param SCRIPT_NAME $fastcgi_script_name;
        fastcgi_param PATH_INFO $fastcgi_path_info;
        fastcgi_param PATH_TRANSLATED $document_root$fastcgi_path_info;
        fastcgi_index index.php;

        # 必要的请求头
        fastcgi_param HTTP_X_REAL_IP $remote_addr;
        fastcgi_param HTTP_X_FORWARDED_FOR $proxy_add_x_forwarded_for;
        fastcgi_param HTTP_X_FORWARDED_PROTO $real_scheme;
        fastcgi_param REQUEST_SCHEME $real_scheme;
        fastcgi_param HTTPS $real_scheme;

        # FastCGI 超时设置
        fastcgi_connect_timeout 30s;
        fastcgi_send_timeout 60s;
        fastcgi_read_timeout 60s;
    }

    # 主路由配置（支持前端控制器）
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }
}`;

export default {
    emits: ['writefile'],
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
        IconCheckCircleFill,
        IconClose,
        IconExclamationCircleFill,
        IconLoading,
        IconPlus,
        IconSearch,
        IconUpload,
    },
    data() {
        return {
            field_arr: [
                "metadata.name",
                "metadata.namespace",
                "spec.serviceAccountName",
                "status.hostIP",
                "status.podIP",
                "status.podIPs",
                "spec.nodeName",
            ],
            vtitle: '',
            showYaml: false,

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

            json: {},
            yaml: '',
            git: { url: '' },
            form: {
                type: 'docker',
                name: "",
                author: "",
                description: "",
                identifie: "",
                gatewayPluginCategory: 'custom',
                gatewayPluginDriver: 'higress-wasm/v1',
                gatewayPluginUrl: '',
                gatewayPluginPhase: 'UNSPECIFIED_PHASE',
                gatewayPluginPriority: 0,
                gatewayPluginSupportGlobal: true,
                gatewayPluginSupportRule: false,
                gatewayPluginDefaultEnabled: true,
                gatewayPluginDefaultConfig: '{}',
                environmentImageLanguage: '',
                environmentImageTemplate: '',
                environmentImageVersion: [],
                environmentSystemRebootRestore: true,
                environmentNginxVhostTemplate: '',
                systemImageCategory: 'operating-system',
                systemImageTemplate: '',
                systemImageVersions: [],
                startParams: [],
                dependsIn: [],
                depends: [],

                mysql8: false,
                redis: false,

                image: "",

                taginput: "",
                tags: [],




                build_context: '',


                shell: [],
                containers: [],
                language: '',
                helm: {
                    helmtype: '1',
                    repository: '',
                    chartName: 'default',
                    chartName2: '',
                    version: '',
                    kv: [],
                },
                entry: 'public',
                environmentName: '',
                environmentGoodsId: 0,
                environmentVersion: '',
                installType: traditionInstallTypes.site,
                cmd: [''],

            },

            shellConfig: {
                show: false,
                editIndex: -1,
                item: null,
            },

            domainConfig: {
                show: false,
                ingress: [],
            },

            app_ports: [],
            app_names: [],

            volumeRules: {
                mountPath: [{ required: true, message: '内容不能为空', trigger: 'blur' }],
                subPath: [{ required: true, message: '内容不能为空', trigger: 'blur' }],
                type: [{ required: true, message: '内容不能为空', trigger: 'blur' }],
                hostPath: [{ required: true, message: '内容不能为空', trigger: 'blur' }],
            },

            rules: {
                name: [
                    { required: true, message: '内容不能为空', trigger: 'blur' },
                ],
                identifie: [
                    { required: true, message: '内容不能为空', trigger: 'blur' },
                    {
                        required: true, trigger: 'blur', validator: (value, callback) => {
                            if (this.form.author) { callback() }
                            else { callback("请输入完整") }
                        }
                    },
                    {
                        required: true, trigger: 'blur', validator: (value, callback) => {
                            if (/^[a-zA-Z0-9]+$/.test(value)) { callback() }
                            else { callback("标识格式有误") }
                        }
                    },
                    {
                        required: true, trigger: 'blur', validator: (value, callback) => {
                            if (/^[a-zA-Z0-9]+$/.test(this.form.author)) { callback() }
                            else { callback("标识格式有误") }
                        }
                    },
                ],
                port: [
                    { required: true, message: '内容不能为空', trigger: 'blur' },
                    {
                        required: true, trigger: 'blur', validator: (value, callback) => {
                            if (value?.filter?.(i => i.name && i.protocol && i.port)?.length) {
                                callback()
                            } else {
                                callback("必填项不能为空")
                            }
                        }
                    },
                ],
                gatewayPluginUrl: [
                    { required: true, message: '请输入插件镜像地址', trigger: 'blur' },
                ],
                gatewayPluginCategory: [
                    { required: true, message: '请选择插件分类', trigger: 'change' },
                ],
                gatewayPluginDriver: [
                    { required: true, message: '请选择运行时驱动', trigger: 'change' },
                ],
                gatewayPluginPhase: [
                    { required: true, message: '请选择执行阶段', trigger: 'change' },
                ],
                gatewayPluginPriority: [
                    { required: true, message: '请输入优先级', trigger: 'change' },
                ],
                environmentImageLanguage: [
                    {
                        required: true,
                        message: '请选择环境语言',
                        trigger: 'change',
                    },
                ],
                environmentImageTemplate: [
                    {
                        required: true,
                        message: '请输入镜像地址',
                        trigger: 'blur',
                        validator: (value, callback) => this.validateEnvironmentImageTemplate(value, callback),
                    },
                ],
                environmentImageVersion: [
                    {
                        required: true,
                        message: '请输入环境版本',
                        trigger: 'change',
                        validator: (value, callback) => this.validateEnvironmentImageVersions(value, callback),
                    },
                ],
                environmentNginxVhostTemplate: [
                    {
                        required: true,
                        message: '请输入 Nginx 模板',
                        trigger: 'blur',
                    },
                ],
                systemImageCategory: [
                    { required: true, message: '请选择系统镜像分类', trigger: 'change' },
                ],
                systemImageTemplate: [
                    {
                        required: true,
                        trigger: 'blur',
                        validator: (value, callback) => this.validateSystemImageTemplate(value, callback),
                    },
                ],
                systemImageVersions: [
                    {
                        required: true,
                        trigger: 'change',
                        validator: (value, callback) => this.validateSystemImageVersions(value, callback),
                    },
                ],
            },
            addRules: {
                identifie: [
                    { required: true, message: '内容不能为空', trigger: 'blur' },

                    {
                        required: true, trigger: 'blur', validator: (value, callback) => {
                            if (/^[a-zA-Z0-9]+-[a-zA-Z0-9]+$/.test(value)) { callback() }
                            else { callback("标识格式有误") }
                        }
                    },
                ],
                name: [{ required: true, message: '内容不能为空', trigger: 'blur' }],
            },
            yamlDom: "",
            downloadUrl: "",

            dialogVisible: false,
            icons: [],
            activeItem: null,


            frameVisible: false,
            downloadFrame: null,
            beforeDownload: false,


            otherData: {},
            logoimg: '',
            logofile: null,

            baseurl: '',

            depend: {
                input: '',
                item: null,
            },

            dependForm: {
                show: false,
                editIndex: -1,
                identifie: '',
                identifie_before: '',
                identifie_last: '',
                name: '',
                required: true,
                from: '',
            },

            languageList: [],

            spEdit: {
                show: false,
                values: '',
            },
            spDesc: {
                show: false,
                item: null,
                value: '',
            },

            dependsList: {},
            subDependsList: {},

            dependPicker: {
                show: false,
                activeTab: 'local',
                keyword: '',
                page: 1,
                pageSize: 8,
                loading: false,
                editIndex: -1,
                lists: {
                    local: [],
                    official: [],
                },
                loaded: {
                    local: false,
                    official: false,
                },
            },

            buildImageLogData: null,
            buildImageInterval: null,


            helmCharts: [],
            helmChartVersions: [],
            helmChartKeyword: '',
            helmChartVersionKeyword: '',

            getChartInfoLoading: false,

            containerPluginData: {},

            environmentList: [],
            versionList: [],
            commandList: [],

            disabledDomainStartParams: false,
            formulaSettingLoading: false,
            formulaSettingLoaded: false,
            formulaBaseInfo: null,
            environmentPreviousDepends: null,
            systemImageDependencyManaged: false,
            initialApplicationType: '',
            currentApplicationType: '',
            gatewayPluginCategoryOptions,
            nginxTemplatePlaceholders,
            nginxTemplateExample,
            nginxTemplateExampleVisible: false,
        }
    },
    created() {
        this.getDependsList();
        this.getEnvironmentList();
        this.baseurl = window?.$wujie?.props?.url || '';
        hljs.configure({ ignoreUnescapedHTML: true });
        this.init(this.data);
        if (!this.option?.pureManifest) {
            this.getTag();
            if (['environment', 'gateway-plugin'].includes(this.form.type)) {
                this.loadFormulaSetting().catch(() => { });
            }
        } else {
            this.otherData.required = Boolean(this.option?.required);
        }

        this.languageList = [
            {
                "name": "php8.1",
                "identifie": "tradition_php81",
            },
            {
                "name": "php8.0",
                "identifie": "tradition_php80",
            },
            {
                "name": "php7.4",
                "identifie": "tradition_php74",
            },
            {
                "name": "php7.3",
                "identifie": "tradition_php73",
            },
            {
                "name": "PHP7.2",
                "identifie": "tradition_php72",
            }
        ]

    },

    computed: {
        traditionInstallTypes() {
            return traditionInstallTypes;
        },
        environmentLanguageOptions() {
            let currentLanguage = String(this.form.environmentImageLanguage || '').trim();
            if (currentLanguage && !environmentLanguagePresets.some(option => option.value == currentLanguage)) {
                return [
                    { label: `${currentLanguage}（已有值）`, value: currentLanguage },
                    ...environmentLanguagePresets,
                ];
            }
            return environmentLanguagePresets;
        },
        helmChartOptions() {
            return this.filterAutocompleteOptions(this.helmCharts, this.helmChartKeyword);
        },
        helmChartVersionOptions() {
            return this.filterAutocompleteOptions(this.helmChartVersions, this.helmChartVersionKeyword);
        },
        selectedEnvironment() {
            return this.environmentList?.find?.(i => i.identifie == this.form.environmentName) || null;
        },
        shellTypeOptions() {
            return [
                { label: '安装前执行', value: 'requireinstall' },
                { label: '安装后执行', value: 'install' },
                { label: '升级前执行', value: 'pre-upgrade' },
                { label: '升级后执行', value: 'upgrade' },
                { label: '卸载后执行', value: 'uninstall' },
                { label: '手动触发', value: 'custom' },
            ];
        },
        shellContainerOptions() {
            let containers = this.form.containers?.length
                ? this.form.containers
                : (this.json?.platform?.['container-v2'] || []);
            return (containers || [])
                .filter(item => item && !item.isInitContainer && item.name)
                .map(item => ({
                    label: item.name,
                    value: item.name,
                }));
        },
        hasShellContainerOptions() {
            return this.form.type != 'tradition' && this.shellContainerOptions.length > 0;
        },
        defaultShellContainer() {
            return this.hasShellContainerOptions ? this.shellContainerOptions[0].value : '';
        },
        environmentIcon() {
            return icon => {
                if (!icon) return '';
                return /^https?:\/\//i.test(icon) ? icon : `https://img.w7.cc${icon.startsWith('/') ? '' : '/'}${icon}`;
            };
        },
        dependPickerCurrentList() {
            return this.dependPicker.lists?.[this.dependPicker.activeTab] || [];
        },
        dependPickerRows() {
            let start = (this.dependPicker.page - 1) * this.dependPicker.pageSize;
            return this.dependPickerCurrentList.slice(start, start + this.dependPicker.pageSize);
        },
        dependPickerTotal() {
            return this.dependPickerCurrentList.length;
        },
    },
    watch: {
        'dependForm.identifie_before'() {
            this.dependForm.identifie = this.dependForm.identifie_before + '-' + this.dependForm.identifie_last;
        },
        'dependForm.identifie_last'() {
            this.dependForm.identifie = this.dependForm.identifie_before + '-' + this.dependForm.identifie_last;
        },

        data() {
            this.init(this.data);
            if (!this.option?.pureManifest && ['environment', 'gateway-plugin', 'system-image'].includes(this.form.type)) {
                this.loadFormulaSetting().catch(() => { });
            }
        },
        'form.storage'(v) {
            this.checkStartParams(v, 'storage', [
                { mark: 'storage', name: 'global.cluster.storageRWmode', title: '读写模式', required: true, values_text: '%STORAGE_RW_MODE%', module_name: '' },
                { mark: 'storage', name: 'global.cluster.storageSize', title: '存储大小', required: true, values_text: '%STORAGE_SIZE%', module_name: '' },
                { mark: 'storage', name: 'global.cluster.storageClassName', title: '存储类', required: true, values_text: '%STORAGE_CLASS_NAME%', module_name: '' },
            ])
        },
        'form.mysql8'(v) {
            if (v && this.form.mysql5) { this.form.mysql5 = false; }
            this.checkStartParams(v, 'mysql8', [
                { mark: 'mysql8', name: 'MYSQL_DATABASE', title: 'mysql数据库', required: true, values_text: 'dbname_%RANDOM%', module_name: 'w7_mysql.DB_NAME' },
                { mark: 'mysql8', name: 'MYSQL_PASSWORD', title: 'mysql密码', required: true, values_text: '%MYSQL_ROOT_PASSWORD%', module_name: 'w7_mysql' },
                { mark: 'mysql8', name: 'MYSQL_USERNAME', title: 'mysql用户名', required: true, values_text: '%MYSQL_ROOT_USERNAME%', module_name: 'w7_mysql' },
                { mark: 'mysql8', name: 'MYSQL_PORT', title: 'mysql端口', required: true, values_text: '%PORT%', module_name: 'w7_mysql' },
                { mark: 'mysql8', name: 'MYSQL_HOST', title: 'mysql地址', required: true, values_text: '%HOST%', module_name: 'w7_mysql' },
            ])
        },
        'form.mysql5'(v) {
            if (v && this.form.mysql8) { this.form.mysql8 = false; }
            this.checkStartParams(v, 'mysql5', [
                { mark: 'mysql5', name: 'MYSQL_DATABASE', title: 'mysql数据库', required: true, values_text: 'dbname_%RANDOM%', module_name: 'w7_mysql.DB_NAME' },
                { mark: 'mysql5', name: 'MYSQL_PASSWORD', title: 'mysql密码', required: true, values_text: '%MYSQL_ROOT_PASSWORD%', module_name: 'w7_mysql5' },
                { mark: 'mysql5', name: 'MYSQL_USERNAME', title: 'mysql用户名', required: true, values_text: '%MYSQL_ROOT_USERNAME%', module_name: 'w7_mysql5' },
                { mark: 'mysql5', name: 'MYSQL_PORT', title: 'mysql端口', required: true, values_text: '%PORT%', module_name: 'w7_mysql5' },
                { mark: 'mysql5', name: 'MYSQL_HOST', title: 'mysql地址', required: true, values_text: '%HOST%', module_name: 'w7_mysql5' },
            ])
        },
        'form.redis'(v) {
            this.checkStartParams(v, 'redis', [
                { mark: 'redis', name: 'REDIS_PASSWORD', title: 'redis密码', required: true, values_text: '%REDIS_PASSWORD%', module_name: 'w7_redis' },
                { mark: 'redis', name: 'REDIS_PORT', title: 'redis端口', required: true, values_text: '%PORT%', module_name: 'w7_redis' },
                { mark: 'redis', name: 'REDIS_HOST', title: 'redis地址', required: true, values_text: '%HOST%', module_name: 'w7_redis' },
            ]);
        },
        'form.mongodb6'(v) {
            this.checkStartParams(v, 'mongodb6', [
                { mark: 'mongodb6', name: 'MONGO_PORT', title: '端口', required: true, values_text: '%PORT%', module_name: 'w7_mongodb' },
                { mark: 'mongodb6', name: 'MONGO_HOST', title: '内网域名', required: true, values_text: '%HOST%', module_name: 'w7_mongodb' },
                { mark: 'mongodb6', name: 'MONGO_INITDB_ROOT_PASSWORD', title: '密码', required: true, values_text: '%MONGO_INITDB_ROOT_PASSWORD%', module_name: 'w7_mongodb' },
                { mark: 'mongodb6', name: 'MONGO_INITDB_ROOT_USERNAME', title: '用户名', required: true, values_text: '%MONGO_INITDB_ROOT_USERNAME%', module_name: 'w7_mongodb' },
            ]);
        },
        'form.domain'(v) {
            this.checkStartParams(v, 'domain', [
                { mark: 'domain', name: 'DOMAIN_URL', title: '域名', required: true, values_text: '%DOMAIN_URL%', module_name: '' },
            ]);
        },
        'form.ingress': {
            deep: true,
            handler() {
                this.changeForm();
            }
        },
        'form.shell': {
            deep: true,
            handler() {
                this.changeForm();
            }
        },
        'option.app_ports'(v) {
            this.computedAppPort();
        }
    },
    beforeUnmount() {
        try {
            clearInterval(this.buildImageInterval)
        } catch { }
    },
    methods: {
        useNginxTemplateExample() {
            const applyExample = () => {
                this.form.environmentNginxVhostTemplate = this.nginxTemplateExample;
                this.nginxTemplateExampleVisible = false;
                messageSuccess('已填入 Nginx 模板示例');
            };
            const currentTemplate = String(this.form.environmentNginxVhostTemplate || '').trim();

            if (!currentTemplate || currentTemplate == this.nginxTemplateExample.trim()) {
                applyExample();
                return;
            }

            confirm({
                title: '使用 Nginx 模板示例',
                content: '当前模板内容将被示例覆盖，是否继续？',
                confirmButtonText: '覆盖并使用',
                cancelButtonText: '取消',
                onOk: applyExample,
            });
        },
        validateEnvironmentImageTemplate(value, callback) {
            let template = String(value || '').trim();
            if (!template) {
                callback('请输入镜像地址');
                return;
            }
            if (/\s/.test(template)) {
                callback('镜像地址不能包含空格或换行');
                return;
            }
            if (!template.includes('{version}')) {
                callback('镜像地址必须包含 {version} 占位符');
                return;
            }
            let unsupportedPlaceholders = [...new Set(template.match(/\{[^{}]+\}/g) || [])]
                .filter(placeholder => placeholder != '{version}');
            if (unsupportedPlaceholders.length) {
                callback('镜像地址包含不支持的占位符：' + unsupportedPlaceholders.join('、'));
                return;
            }
            callback();
        },
        validateEnvironmentImageVersions(value, callback) {
            let versions = this.normalizeEnvironmentVersions(value);
            if (!versions.length) {
                callback('请输入至少一个环境版本');
                return;
            }
            let invalidVersions = [...new Set(versions.filter(version => !/^[A-Za-z0-9_][A-Za-z0-9._-]{0,127}$/.test(version)))];
            if (invalidVersions.length) {
                let invalidText = invalidVersions.slice(0, 3).join('、');
                if (invalidVersions.length > 3) { invalidText += ' 等' }
                callback('以下语言版本格式不正确：' + invalidText);
                return;
            }
            callback();
        },
        validateSystemImageTemplate(value, callback) {
            let template = String(value || '').trim();
            if (!template) { callback('请输入镜像地址'); return; }
            if (/\s/.test(template)) { callback('镜像地址不能包含空格或换行'); return; }
            if (!template.includes('{version}')) { callback('镜像地址必须包含 {version} 占位符'); return; }
            let unsupported = [...new Set(template.match(/\{[^{}]+\}/g) || [])]
                .filter(item => item != '{version}');
            if (unsupported.length) {
                callback('镜像地址包含不支持的占位符：' + unsupported.join('、'));
                return;
            }
            callback();
        },
        validateSystemImageVersions(value, callback) {
            let versions = this.normalizeEnvironmentVersions(value);
            if (!versions.length) { callback('请输入至少一个系统版本'); return; }
            let invalid = versions.filter(version => !/^[A-Za-z0-9_][A-Za-z0-9._-]{0,127}$/.test(version));
            if (invalid.length) { callback('系统版本格式不正确：' + invalid.slice(0, 3).join('、')); return; }
            callback();
        },
        systemImageStartParams() {
            let versions = this.normalizeEnvironmentVersions(this.form.systemImageVersions);
            let fixedNames = new Set([
                'IMAGE_VERSION',
                'global.cluster.storageRWmode',
                'global.cluster.storageSize',
                'global.cluster.storageClassName',
            ]);
            let customParams = (this.form.startParams || []).filter(item => !fixedNames.has(item?.name));
            return [
                { mark: 'system-image', name: 'IMAGE_VERSION', title: '镜像版本', required: true, values_text: versions.join('|'), module_name: '', description: '选择要安装的镜像版本', type: 'select' },
                { mark: 'storage', name: 'global.cluster.storageRWmode', title: '读写模式', required: true, values_text: '%STORAGE_RW_MODE%', module_name: '', description: '', type: 'text' },
                { mark: 'storage', name: 'global.cluster.storageSize', title: '存储大小', required: true, values_text: '%STORAGE_SIZE%', module_name: '', description: '', type: 'text' },
                { mark: 'storage', name: 'global.cluster.storageClassName', title: '存储类', required: true, values_text: '%STORAGE_CLASS_NAME%', module_name: '', description: '', type: 'text' },
                ...customParams,
            ];
        },
        environmentStartParams() {
            let versions = this.normalizeEnvironmentVersions(this.form.environmentImageVersion);
            let fixedNames = new Set([
                'IMAGE_VERSION',
                'DOMAIN_URL',
                'global.cluster.storageRWmode',
                'global.cluster.storageSize',
                'global.cluster.storageClassName',
            ]);
            let customParams = (this.form.startParams || []).filter(item => !fixedNames.has(item?.name));
            let params = [
                { mark: 'environment', name: 'IMAGE_VERSION', title: '环境版本', required: true, values_text: versions.join('|'), module_name: '', description: '选择要安装的运行环境版本', type: 'select' },
                { mark: 'environment-site', name: 'DOMAIN_URL', title: '站点域名', required: true, values_text: '%DOMAIN_URL%', module_name: '', description: '站点代码目录使用此域名隔离', type: 'text' },
            ];
            return [...params, ...customParams];
        },
        hasEnvironmentCodePackage() {
            return this.form.type == 'environment'
                && Boolean(String(this.zip?.url || this.json?.source?.url || '').trim());
        },
        validateEnvironmentStorage() {
            if (this.form.type != 'environment') { return true; }
            let volumes = this.json?.platform?.volumes;
            let storage = Array.isArray(volumes)
                && volumes.find(item => item?.persistentVolumeClaim);
            if (storage) { return true; }
            messageWarning('运行环境必须配置站点存储卷');
            return false;
        },
        syncEnvironmentVersionConfig() {
            if (this.form.type != 'environment') { return; }
            this.form.environmentImageVersion = this.normalizeEnvironmentVersions(this.form.environmentImageVersion);
            this.form.startParams = this.environmentStartParams();
            this.changeForm();
        },
        applyEnvironmentCommand() {
            if (this.form.type != 'environment') { return; }
            let container = this.getEnvironmentMainContainer();
            if (!container) { return; }
            let command = (this.form.cmd || [])
                .map(item => String(item || '').trim())
                .filter(Boolean);
            if (command.length) {
                container.command = command;
            } else {
                delete container.command;
            }
        },
        ensureEnvironmentContainerDefaults(forceImage = false) {
            if (this.form.type != 'environment') { return; }
            this.json.platform = this.json.platform || {};
            delete this.json.platform.container;
            let containers = Array.isArray(this.json.platform['container-v2'])
                ? this.json.platform['container-v2']
                : [];
            let mainContainer = containers.find(item => !item?.isInitContainer);
            if (!mainContainer) {
                mainContainer = {
                    name: (this.form.author && this.form.identifie)
                        ? `${this.form.author}-${this.form.identifie}`
                        : (this.json?.application?.identifie || 'environment'),
                    image: '',
                    imagePullPolicy: 'IfNotPresent',
                };
                containers.unshift(mainContainer);
            }
            let imageTemplate = String(this.form.environmentImageTemplate || '').trim();
            if (forceImage || !String(mainContainer.image || '').trim()) {
                mainContainer.image = imageTemplate;
            }
            if (!mainContainer.imagePullPolicy) {
                mainContainer.imagePullPolicy = 'IfNotPresent';
            }
            this.json.platform['container-v2'] = containers;
            this.form.containers = containers;
            this.json.platform.workload = this.json.platform.workload || {};
            if (!this.json.platform.workload.type) {
                this.json.platform.workload.type = 'Deployment';
            }
            if (!this.containerPluginData.kind) {
                this.containerPluginData.kind = 'Deployment';
            }
            this.fillDefaultShellContainer();
        },
        getEnvironmentMainContainer() {
            return this.json?.platform?.['container-v2']
                ?.find(item => !item?.isInitContainer);
        },
        syncEnvironmentRuntimeConfig() {
            if (this.form.type != 'environment') { return; }
            this.ensureEnvironmentContainerDefaults(true);
            this.applyEnvironmentRebootRestoreConfig();
            this.applyEnvironmentCommand();
            this.changeForm();
        },
        applyEnvironmentRebootRestoreConfig() {
            if (this.form.type != 'environment' || !this.json?.platform) { return; }
            if (!this.form.environmentSystemRebootRestore) {
                this.json.platform.runtimeClassName = 'sysbox-runc';
                this.json.platform.hostUsers = false;
                return;
            }
            delete this.json.platform.hostUsers;
            if (this.json.platform.runtimeClassName == 'sysbox-runc') {
                delete this.json.platform.runtimeClassName;
            }
        },
        syncSystemImageConfig() {
            if (this.form.type != 'system-image') { return; }
            this.form.systemImageVersions = this.normalizeEnvironmentVersions(this.form.systemImageVersions);
            this.form.startParams = this.systemImageStartParams();
            this.form.storage = true;
            this.json.platform = this.json.platform || {};
            this.ensureSystemImageContainer();
            this.changeForm();
        },
        getSystemImageRootfsAnnotation() {
            let containerName = this.json?.platform?.['container-v2']?.[0]?.name
                || ((this.form.author && this.form.identifie)
                    ? `${this.form.author}-${this.form.identifie}`
                    : 'system-image');
            containerName = String(containerName).replaceAll('_', '-');
            return JSON.stringify([{
                name: containerName,
                volumeName: 'system-rootfs',
                path: `${containerName}/system`,
                persistentSpecialMounts: true,
            }]);
        },
        ensureSystemImageContainer() {
            if (this.form.type != 'system-image') { return; }
            this.json.platform = this.json.platform || {};
            delete this.json.source;
            delete this.json.web;
            delete this.json.platform.container;
            let containers = this.json.platform['container-v2'] || [];
            if (!containers.length) {
                containers.push({
                    name: (this.form.author && this.form.identifie)
                        ? `${this.form.author}-${this.form.identifie}` : 'system-image',
                    image: String(this.form.systemImageTemplate || '').trim(),
                    imagePullPolicy: 'IfNotPresent',
                });
            }
            containers[0].image = String(this.form.systemImageTemplate || '').trim();
            let command = (this.form.cmd || [])
                .map(item => String(item || '').trim())
                .filter(Boolean);
            if (command.length) {
                containers[0].command = command;
            } else {
                delete containers[0].command;
            }
            this.json.platform['container-v2'] = containers;
            this.form.containers = containers;
            this.json.platform.workload = { ...(this.json.platform.workload || {}), type: 'Deployment' };
            this.containerPluginData.kind = 'Deployment';
            this.json.platform.runtimeClassName = 'sysbox-runc';
            this.json.platform.hostUsers = false;
            this.json.platform.volumes = [{
                name: 'system-rootfs',
                persistentVolumeClaim: { claimName: '' },
            }];
            containers[0].volumeMounts = [{
                name: 'system-rootfs',
                mountPath: '/system-rootfs',
            }];
            this.json.application = this.json.application || {};
            this.json.application.annotation = {
                ...(this.json.application.annotation || {}),
                [systemImageAnnotationKeys.rootfs]: this.getSystemImageRootfsAnnotation(),
            };
        },
        loadFormulaSetting() {
            if (this.option?.pureManifest || !this.identifie) {
                return Promise.resolve(null);
            }
            if (this.formulaSettingLoaded) {
                return Promise.resolve(this.formulaBaseInfo);
            }
            if (this._formulaSettingPromise) {
                return this._formulaSettingPromise;
            }

            this.formulaSettingLoading = true;
            this._formulaSettingPromise = myAxios.post('/respo/setting/get', {
                identifie: this.identifie,
            }).then(res => {
                let baseInfo = res?.data?.data?.base_info;
                if (!baseInfo) {
                    throw new Error('制品基础信息为空');
                }
                this.formulaBaseInfo = JSON.parse(JSON.stringify(baseInfo));
                this.applyEnvironmentAnnotationForm(baseInfo.annotation || {});
                this.applySystemImageAnnotationForm(baseInfo.annotation || {});
                if (this.form.type == 'gateway-plugin') {
                    this.form.once = true;
                }
                this.formulaSettingLoaded = true;
                return this.formulaBaseInfo;
            }).finally(() => {
                this.formulaSettingLoading = false;
                this._formulaSettingPromise = null;
            });
            return this._formulaSettingPromise;
        },
        applyEnvironmentAnnotationForm(annotation = {}) {
            this.form.environmentImageLanguage = String(annotation[environmentAnnotationKeys.imageLanguage] || '');
            this.form.environmentImageTemplate = String(annotation[environmentAnnotationKeys.imageTemplate] || '');
            this.form.environmentImageVersion = this.normalizeEnvironmentVersions(annotation[environmentAnnotationKeys.imageVersion]);
            this.form.environmentNginxVhostTemplate = String(annotation[environmentAnnotationKeys.nginxVhostTemplate] || '');
            const restoreValue = annotation[environmentAnnotationKeys.systemRebootRestore];
            this.form.environmentSystemRebootRestore = restoreValue === undefined || restoreValue === null || restoreValue === ''
                ? String(this.form.environmentImageLanguage || '').toLowerCase() != 'php'
                : String(restoreValue).toLowerCase() == 'true';
            if (this.form.type == 'environment') {
                this.ensureEnvironmentContainerDefaults();
                this.form.startParams = this.environmentStartParams();
                this.applyEnvironmentRebootRestoreConfig();
            }
        },
        applySystemImageAnnotationForm(annotation = {}) {
            this.form.systemImageCategory = String(annotation[systemImageAnnotationKeys.category] || 'operating-system');
            this.form.systemImageVersions = this.normalizeEnvironmentVersions(annotation[systemImageAnnotationKeys.versions]);
        },
        normalizeEnvironmentVersions(value) {
            let values = Array.isArray(value) ? value : String(value || '').split(/[,，\n]/);
            return [...new Set(values
                .map(item => String(item || ''))
                .flatMap(item => item.split(/[,，\n]/))
                .map(item => item.trim())
                .filter(Boolean))];
        },
        getEnvironmentParentIdentifie() {
            // AppGroup parent follows the complete application identifier
            // (author-identifie), rather than a fixed environment name.
            const currentIdentifie = this.form.author && this.form.identifie
                ? `${this.form.author}-${this.form.identifie}`
                : (this.json?.application?.identifie || this.form.identifie || this.identifie || '');
            return String(currentIdentifie)
                .trim()
                .replaceAll('_', '-')
                .toLowerCase();
        },
        changeEnvironmentLanguage(value) {
            this.form.environmentSystemRebootRestore = String(value || '').toLowerCase() != 'php';
        },
        getEnvironmentAnnotations() {
            let versions = this.normalizeEnvironmentVersions(this.form.environmentImageVersion);
            this.form.environmentImageVersion = versions;
            const annotations = {
                [environmentAnnotationKeys.parent]: this.getEnvironmentParentIdentifie(),
                [environmentAnnotationKeys.imageLanguage]: String(this.form.environmentImageLanguage || '').trim(),
                [environmentAnnotationKeys.imageTemplate]: String(this.form.environmentImageTemplate || '').trim(),
                [environmentAnnotationKeys.imageVersion]: versions.join(','),
                [environmentAnnotationKeys.nginxVhostTemplate]: String(this.form.environmentNginxVhostTemplate || ''),
                [environmentAnnotationKeys.systemRebootRestore]: String(Boolean(this.form.environmentSystemRebootRestore)),
            };
            if (this.form.environmentSystemRebootRestore) {
                if (this.json?.application?.annotation) {
                    delete this.json.application.annotation[systemImageAnnotationKeys.rootfs];
                }
            } else {
                const rootfs = getEnvironmentAppRootfsAnnotation(this.json);
                if (rootfs) annotations[systemImageAnnotationKeys.rootfs] = rootfs;
            }
            return annotations;
        },
        filterAnnotationsForType(annotation = {}, type = this.form.type) {
            let filtered = { ...(annotation || {}) };
            if (type != 'environment' && type != 'system-image') {
                Object.values(environmentAnnotationKeys).forEach(key => delete filtered[key]);
            } else if (type == 'system-image') {
                Object.values(environmentAnnotationKeys)
                    .filter(key => key != environmentAnnotationKeys.imageVersion)
                    .forEach(key => delete filtered[key]);
            }
            Object.keys(filtered)
                .filter(key => type != 'gateway-plugin'
                    && (key.startsWith(gatewayPluginAnnotationPrefix)
                        || gatewayPluginAnnotationKeys.includes(key)))
                .forEach(key => delete filtered[key]);
            if (type == 'environment') {
                Object.assign(filtered, this.getEnvironmentAnnotations());
            }
            if (type != 'system-image') {
                Object.values(systemImageAnnotationKeys)
                    .filter(key => type != 'environment' || (key != environmentAnnotationKeys.imageVersion && key != systemImageAnnotationKeys.rootfs))
                    .forEach(key => delete filtered[key]);
            } else {
                let versions = this.normalizeEnvironmentVersions(this.form.systemImageVersions);
                Object.assign(filtered, {
                    [systemImageAnnotationKeys.category]: this.form.systemImageCategory || 'operating-system',
                    [systemImageAnnotationKeys.versions]: versions.join(','),
                    [systemImageAnnotationKeys.rootfs]: this.getSystemImageRootfsAnnotation(),
                });
            }
            return filtered;
        },
        cleanupTypeSpecificManifest(previousType, nextType) {
            this.json.application = this.json.application || {};
            this.json.platform = this.json.platform || {};
            this.json.application.annotation = this.filterAnnotationsForType(
                this.json.application.annotation || {},
                nextType,
            );

            if (previousType == 'system-image' && nextType != 'system-image') {
                const fixedNames = new Set([
                    'IMAGE_VERSION',
                    'global.cluster.storageRWmode',
                    'global.cluster.storageSize',
                    'global.cluster.storageClassName',
                ]);
                this.form.startParams = (this.form.startParams || [])
                    .filter(item => !fixedNames.has(item?.name));
                this.json.platform.startParams = this.serializeStartParams();
                delete this.json.platform.hostUsers;
                if (this.json.platform.runtimeClassName == 'sysbox-runc') {
                    delete this.json.platform.runtimeClassName;
                }
                this.json.platform.volumes = (this.json.platform.volumes || [])
                    .filter(item => item?.name != 'system-rootfs');
                if (!this.json.platform.volumes.length) {
                    delete this.json.platform.volumes;
                }
                (this.json.platform['container-v2'] || []).forEach(container => {
                    container.volumeMounts = (container.volumeMounts || [])
                        .filter(item => item?.name != 'system-rootfs');
                    if (!container.volumeMounts.length) {
                        delete container.volumeMounts;
                    }
                });
                delete this.json.platform['container-v2'];
                this.form.containers = [];
                this.form.cmd = [''];
            }
            if (previousType == 'environment' && !['environment', 'system-image'].includes(nextType)) {
                this.form.startParams = (this.form.startParams || [])
                    .filter(item => !['IMAGE_VERSION', 'DOMAIN_URL'].includes(item?.name));
                this.json.platform.startParams = this.serializeStartParams();
                removeEnvironmentAppCodeStorage(this.json);
            }
            if (previousType != 'system-image' && nextType == 'system-image') {
                delete this.json.source;
                delete this.json.web;
                delete this.json.platform.ingress;
                delete this.json.platform.volumeClaimTemplates;
                delete this.json.platform.container;
                this.json.platform['container-v2'] = [];
                this.form.containers = [];
                this.form.cmd = [''];
                this.form.ingress = [];
                this.form.domain = false;
            }
            if (previousType == 'tradition' && nextType != 'tradition') {
                delete this.json.platform.tradition;
            }
            if (previousType == 'helm' && nextType != 'helm') {
                delete this.json.platform.helm;
            }
            if (previousType == 'gateway-plugin' && nextType != 'gateway-plugin') {
                delete this.json.platform.gatewayPlugin;
            }
        },
        shouldSaveFormulaTypeSetting() {
            if (['environment', 'gateway-plugin', 'system-image'].includes(this.form.type)) { return true }
            return ['environment', 'gateway-plugin', 'system-image'].includes(this.initialApplicationType)
                && this.form.type != this.initialApplicationType;
        },
        async saveFormulaTypeSetting() {
            if (this.option?.pureManifest || !this.shouldSaveFormulaTypeSetting()) { return }
            let baseInfo = await this.loadFormulaSetting();
            if (!baseInfo) {
                throw new Error('制品基础信息加载失败');
            }
            let nextBaseInfo = {
                ...baseInfo,
                annotation: this.filterAnnotationsForType(baseInfo.annotation || {}),
                once: this.form.type == 'gateway-plugin'
                    ? true
                    : Boolean(baseInfo.once),
            };
            await myAxios.post('/respo/setting/set', {
                identifie: this.identifie,
                base_info: nextBaseInfo,
            });
            if (this.form.type == 'gateway-plugin') {
                this.form.once = true;
            }
            this.formulaBaseInfo = JSON.parse(JSON.stringify(nextBaseInfo));
        },
        getDefaultTypeTagName(type = this.form.type) {
            if (type == 'environment') { return '运行环境' }
            if (type == 'gateway-plugin') { return '网关插件' }
            return '';
        },
        async ensureDefaultTypeTags() {
            if (this.option?.pureManifest || !this.identifie) { return }
            let names = [...new Set([
                this.getDefaultTypeTagName(this.initialApplicationType),
                this.getDefaultTypeTagName(this.form.type),
            ].filter(Boolean))];
            await Promise.all(names.map(name => myAxios.post('/respo/tag/add', {
                identifie: this.identifie,
                name,
            })));
        },
        isEnvironmentFixedDependency(record) {
            return (this.form.type == 'environment' && isEnvironmentAppDependency(record))
                || (this.form.type == 'system-image' && record?.identifie == 'w7panel-sysbox')
                || this.isTraditionEnvironmentDependency(record);
        },
        isTraditionEnvironmentDependency(record) {
            if (this.form.type != 'tradition') { return false }
            const dependencyIdentify = String(record?.identifie || '').replaceAll('_', '-');
            const environmentIdentify = String(this.form.environmentName || '').replaceAll('_', '-');
            return Boolean(environmentIdentify) && dependencyIdentify == environmentIdentify;
        },
        syncEnvironmentDependency() {
            if (this.form.type != 'environment') {
                if (this._initializing) {
                    this.environmentPreviousDepends = null;
                } else if (this.environmentPreviousDepends !== null) {
                    this.form.depends = this.environmentPreviousDepends;
                    this.environmentPreviousDepends = null;
                }
                return;
            }
            if (this.environmentPreviousDepends === null) {
                let depends = JSON.parse(JSON.stringify(this.form.depends || []));
                if (this.systemImageDependencyManaged) {
                    depends = depends.filter(item => item?.identifie != 'w7panel-sysbox');
                    this.systemImageDependencyManaged = false;
                }
                this.environmentPreviousDepends = this._initializing
                    ? depends.filter(item => !isEnvironmentAppDependency(item) && item?.identifie != 'w7panel-sysbox')
                    : depends;
            }
            this.form.depends = [createEnvironmentAppDependency()];
        },
        syncSystemImageDependency() {
            let depends = Array.isArray(this.form.depends) ? this.form.depends : [];
            if (this.form.type != 'system-image') {
                if (this.systemImageDependencyManaged) {
                    this.form.depends = depends.filter(item => item?.identifie != 'w7panel-sysbox');
                    this.systemImageDependencyManaged = false;
                }
                return;
            }

            this.systemImageDependencyManaged = true;
            let dependency = {
                identifie: 'w7panel-sysbox',
                name: '微擎sysbox',
                subidentifie: '',
                subname: '',
                required: true,
                type: 'out',
                from: 'https://zpk.w7.cc',
            };
            let inserted = false;
            this.form.depends = depends.reduce((result, item) => {
                if (item?.identifie != dependency.identifie) {
                    result.push(item);
                } else if (!inserted) {
                    result.push({ ...item, ...dependency });
                    inserted = true;
                }
                return result;
            }, []);
            if (!inserted) {
                this.form.depends.unshift(dependency);
            }
        },
        filterAutocompleteOptions(list = [], keyword = '') {
            let normalizedKeyword = String(keyword || '').toLowerCase();
            let source = normalizedKeyword
                ? list.filter(item => String(item || '').toLowerCase().includes(normalizedKeyword))
                : list;
            return source.map(item => ({ label: item, value: item }));
        },
        addShellTask() {
            this.form.shell.push({ title: '', type: '', container: this.defaultShellContainer, shell: '' });
            this.changeForm();
        },
        openShellConfig(record, index) {
            if (!record) { return }
            const item = { ...record };
            if (item.container === undefined) { item.container = ''; }
            if (this.form.type != 'tradition' && !item.container && this.defaultShellContainer) {
                item.container = this.defaultShellContainer;
            }
            if (item.shell === undefined) { item.shell = ''; }
            this.shellConfig = {
                show: true,
                editIndex: index,
                item,
            };
        },
        getShellTaskStatus(record) {
            if (!record?.shell) { return '未配置' }
            if (this.form.type == 'tradition') { return '所选环境应用容器' }
            return record.container || this.defaultShellContainer || '默认运行环境';
        },
        fillDefaultShellContainer() {
            if (this.form.type == 'tradition' || !this.defaultShellContainer) { return }
            (this.form.shell || []).forEach(item => {
                if (item && !item.container) {
                    item.container = this.defaultShellContainer;
                }
            });
        },
        resetShellConfig() {
            this.shellConfig = {
                show: false,
                editIndex: -1,
                item: null,
            };
        },
        saveShellConfig() {
            if (this.shellConfig.editIndex >= 0 && this.shellConfig.item) {
                this.form.shell[this.shellConfig.editIndex] = {
                    ...(this.form.shell[this.shellConfig.editIndex] || {}),
                    ...this.shellConfig.item,
                };
            }
            this.resetShellConfig();
            this.changeForm();
        },
        openDomainConfig() {
            let ingress = JSON.parse(JSON.stringify(this.form.ingress || []));
            this.domainConfig = {
                show: true,
                ingress,
            };
        },
        resetDomainConfig() {
            this.domainConfig = {
                show: false,
                ingress: [],
            };
        },
        saveDomainConfig() {
            let ingress = JSON.parse(JSON.stringify(this.domainConfig.ingress || []));
            this.form.ingress = ingress;
            this.checkDomainStartParams(Boolean(ingress.length));
            this.changeForm();
            this.domainConfig.show = false;
        },
        onChange() { },
        isDomainStartParam(item) {
            return item?.mark === 'domain'
                || (item?.name === 'DOMAIN_URL' && ['%DOMAIN_URL%', '%DOMAIN_SSL_URL%'].includes(item?.values_text));
        },
        isSystemImageFixedStartParam(item) {
            return this.form.type == 'system-image' && [
                'IMAGE_VERSION',
                'global.cluster.storageRWmode',
                'global.cluster.storageSize',
                'global.cluster.storageClassName',
            ].includes(item?.name);
        },
        isEnvironmentFixedStartParam(item) {
            return this.form.type == 'environment' && [
                'IMAGE_VERSION',
                'DOMAIN_URL',
                'global.cluster.storageRWmode',
                'global.cluster.storageSize',
                'global.cluster.storageClassName',
            ].includes(item?.name);
        },
        computedSpDisabled(item) {
            return this.isSystemImageFixedStartParam(item)
                || this.isEnvironmentFixedStartParam(item)
                || ((this.disabledDomainStartParams || this.form.type == 'tradition') && this.isDomainStartParam(item))
                || (this.json?.platform?.['volumeClaimTemplates']?.length && item.mark === 'storage')
        },
        serializeStartParams() {
            let start = [];
            let params = this.form.type == 'environment'
                ? this.environmentStartParams()
                : this.form.startParams;
            params = (params || []).filter(item => item?.mark !== 'environment-release'
                && !isDerivedDependencyReleaseStartParam(item)
                && !isLegacyTraditionInstallDirectoryStartParam(item, this.form.type));
            for (let i in params) {
                let o = params[i];
                if (o.name) {
                    start.push({
                        type: 'text',
                        name: o.name,
                        title: o.title,
                        required: o.required,
                        values_text: o.values_text,
                        module_name: o.module_name,
                        description: o.description || '',
                        hidden: Boolean(o.hidden),
                    })
                }
            }
            return start;
        },
        formatIngressRoutes(routes = []) {
            return routes.filter(r => r.path && r.backend?.port).map(r => ({
                ...r,
                backend: {
                    ...r.backend,
                    port: Number(r.backend.port),
                },
            }));
        },
        openAppset() {
            let volumes = this.json?.platform?.volumes;
            let volumeClaimTemplates = this.json?.platform?.volumeClaimTemplates;
            let containers = this.json?.platform?.['container-v2']?.filter(i => !i.isInitContainer);
            let initContainers = this.json?.platform?.['container-v2']?.filter(i => i.isInitContainer);

            emitWujieEvent("containerPlugin", {
                volumes,
                volumeClaimTemplates,
                containers,
                initContainers,
                isTemplate: true,
                pluginData: this.containerPluginData
            }, (data) => {
                let initContainers = data?.initContainers || [];
                initContainers.map(i => i.isInitContainer = true);
                let containers = data?.containers || [];
                let allConatiners = containers.concat(initContainers);
                this.form.containers = allConatiners;
                this.json.platform['container-v2'] = allConatiners;
                this.json.platform['volumes'] = data.volumes;
                this.json.platform['volumeClaimTemplates'] = data.volumeClaimTemplates;
                this.fillDefaultShellContainer();

                this.form.storage = Boolean(data?.volumeClaimTemplates?.length);


                this.json.platform.runtimeClassName = this.form.type == 'system-image'
                    ? 'sysbox-runc'
                    : (data?.pluginData?.gpu ? 'nvidia' : '');

                this.json.platform.workload = this.json?.platform?.workload || {};
                this.json.platform.workload.type = data?.pluginData?.kind || 'deployments';

                this.applyPlatformShells();

                this.json.platform.ingress = (this.form.ingress || []).map(i => ({
                    name: i.name,
                    routes: this.formatIngressRoutes(i.routes),
                }));

                this.yaml = jsyaml.dump(this.json);

                this.computedAppPort(this.json.platform?.['container-v2']);
            })
        },
        checkDomainStartParams(v) {
            this.form.domain = v;
            this.disabledDomainStartParams = !!v;
        },
        async changeEnv(item) {
            if (!item.identifie || item.identifie == this.form.environmentName) { return }

            this.form.depends = this.form.depends?.filter?.(i => !i.temporary) || []
            let findIndex = this.form.depends?.findIndex(i => i.identifie == item.identifie && i.name == item.name)
            if (findIndex != -1) {
                this.form.depends[findIndex] = {
                    ...this.form.depends[findIndex],
                    ...createTraditionEnvironmentDependency(item),
                };
            } else {
                this.form.depends.push(createTraditionEnvironmentDependency(item))
            }

            this.form.environmentName = item.identifie;
            this.form.environmentGoodsId = Number(item.goods_id || item.goodsId || item.id || 0);
            this.form.environmentImageTemplate = String(
                item.image_template || item.image || item.extra?.image || ''
            ).trim();
            this.form.environmentVersion = '';

            if (item.versions?.length) {
                this.form.environmentVersion = item.versions[0];
            }
            this.getStart();
            this.changeForm();
        },
        getEnvironmentList() {
            // 运行环境列表由制品市场提供，支持版本使用接口返回的 support_version 字段。
            // 该接口与市场前端（zm.w7.com）使用同一 API，避免继续依赖旧的 zpk.w7.cc 数据源。
            const listUrl = 'https://api.zm.w7.com/zpk-market/formula/list';
            myAxios.post(listUrl, {
                status: [2, 99],
                page: 1,
                limit: 99,
                tag: '运行环境',
            }, { dontalert: true }).then(res => {
                const environmentList = res.data?.data?.list || [];
                this.environmentList = environmentList.map(item => {
                    const identifie = item.identifie || item.identify || item.identifier
                        || item.formula_identifie || item.formula_identify || '';
                    const environmentLanguage = String(item.environment_language || '').trim();
                    const imageTemplate = String(
                        item.image_template || item.image || item.extra?.image || ''
                    ).trim();
                    const versions = String(item.support_version || '')
                        .split(',')
                        .map(version => version.trim())
                        .filter(Boolean);
                    return {
                        ...item,
                        identifie,
                        name: item.name || item.formula_name || item.title || identifie,
                        environment_language: environmentLanguage,
                        image_template: imageTemplate,
                        versions,
                    };
                }).filter(item => item.identifie);
                this.form.depends?.map?.((item, index) => {
                    if (this.form.type == 'tradition' && this.environmentList?.length) {
                        let now = this.environmentList.find(i => i.identifie == this.form.environmentName);
                        if (now) {
                            if (item.name == now.name && item.identifie == now.identifie) {
                                item.temporary = true;
                            }
                        }
                    }
                })
            }).catch(() => {
                this.environmentList = [];
            });
        },
        getChartInfo() {
            this.getChartInfoLoading = true;
            return myAxios.post('/helm/chart/list', {
                repository_url: this.form.helm.repository
            }, {
                dontalert: true,
            }).then(res => {
                let charts = res?.data?.data?.charts || [];
                this.helmCharts = charts;
                this.helmChartVersions = [];
                this.getChartInfoLoading = false;
            }).catch(() => {
                this.helmCharts = [];
                this.helmChartVersions = [];
                this.getChartInfoLoading = false;
            })
        },
        getChartVersion() {
            return myAxios.post('/helm/chart/version/list', {
                repository_url: this.form.helm.repository,
                chart: this.form.helm.chartName
            }, { dontalert: true }).then(res => {
                let versions = res?.data?.data?.versions || [];
                this.helmChartVersions = versions;
            }).catch(() => {
                this.helmChartVersions = [];
            })
        },
        helmyamlsUpload(e, index) {
            let file = e
            if (!file) { return }
            if (!(/\.yaml$/.test(file.name) || /\.yml$/.test(file.name))) {
                messageError('请上传yaml文件');
                return;
            }
            this.form.helm.depend_yamls[index].name = file.name;
            this.form.helm.depend_yamls[index].nameInput = file.name.replace(/\.yaml$/, '');
            const reader = new FileReader();
            reader.onload = (event) => {
                this.form.helm.depend_yamls[index].yaml = event.target.result;
                this.changeForm();
            };
            reader.readAsText(file, 'UTF-8');
        },
        getDependName(record) {
            if (!record) { return '' }
            return record.name || this.dependsList?.[record.identifie] || record.identifie || '';
        },
        getSubDependsOptions(index) {
            return this.subDependsList?.[index] || { "": "无" };
        },
        getSubDependName(index, identifie) {
            return this.getSubDependsOptions(index)?.[identifie] || '';
        },
        normalizeDependList(list = []) {
            return (list || []).filter(i => i?.install_only_once).map(i => ({
                ...i,
                name: i.name || i.identifie,
            }));
        },
        mergeDependsList(list = []) {
            let obj = { ...this.dependsList };
            list.forEach(i => {
                if (i?.identifie) {
                    obj[i.identifie] = i.name || i.identifie;
                }
            });
            this.dependsList = obj;
        },
        openDependPicker(index = -1) {
            if (this.form.type == 'environment') { return }
            this.dependPicker.show = true;
            this.dependPicker.editIndex = index;
            this.dependPicker.page = 1;
            if (!this.dependPicker.loaded?.[this.dependPicker.activeTab]) {
                this.getDependPickerList();
            }
        },
        closeDependPicker() {
            this.dependPicker.editIndex = -1;
        },
        searchDependPicker() {
            this.dependPicker.page = 1;
            this.getDependPickerList();
        },
        changeDependPickerTab() {
            this.dependPicker.page = 1;
            this.getDependPickerList();
        },
        changeDependPickerPage(page) {
            this.dependPicker.page = page;
        },
        getDependPickerList() {
            let tab = this.dependPicker.activeTab;
            this.dependPicker.loading = true;
            let params = {
                page: 1,
                limit: 999,
                keyword: this.dependPicker.keyword,
            };
            let request = tab == 'official'
                ? myAxios.get('https://zpk.w7.cc/zpk/respo/list?status=2&status=99', {
                    params,
                    dontalert: true,
                })
                : myAxios.get('/respo/list', {
                    params,
                    dontalert: true,
                });
            return request.then(res => {
                let list = this.normalizeDependList(res?.data?.data?.list || []);
                this.dependPicker.lists[tab] = list;
                this.dependPicker.loaded[tab] = true;
                this.mergeDependsList(list);
            }).catch(() => {
                this.dependPicker.lists[tab] = [];
                this.dependPicker.loaded[tab] = true;
            }).finally(() => {
                this.dependPicker.loading = false;
            });
        },
        selectDependFromPicker(record) {
            if (!record?.identifie) { return }
            let data = {
                identifie: record.identifie,
                name: record.name || record.identifie,
                subidentifie: '',
                subname: '',
                required: false,
                type: 'out',
                from: this.dependPicker.activeTab == 'official' ? 'https://zpk.w7.cc' : '',
            };
            let index = this.dependPicker.editIndex;
            if (index >= 0 && this.form.depends[index]) {
                data.required = Boolean(this.form.depends[index].required);
                this.form.depends.splice(index, 1, data);
            } else {
                this.form.depends.push(data);
                index = this.form.depends.length - 1;
            }
            this.dependPicker.show = false;
            this.getSubDepends(index, this.dependPicker.activeTab);
            this.changeForm();
        },
        getSubDepends(index, source = '') {
            this.subDependsList[index] = { "": "无" };
            let identifie = this.form.depends?.[index]?.identifie;
            if (!identifie) { return Promise.resolve() }
            let localUrl = '/respo/v2/info/' + identifie + '/1.0.0';
            let officialUrl = 'https://zpk.w7.cc/zpk/respo/v2/info/' + identifie + '/1.0.0';
            let urls = source == 'official' ? [officialUrl, localUrl] : [localUrl, officialUrl];
            let load = (urlIndex = 0) => {
                let url = urls[urlIndex];
                if (!url) { return Promise.resolve() }
                return myAxios.get(url, {
                    headers: { cancelerror: true },
                    dontalert: true,
                }).then(res => {
                    this.setSubDepends(index, res);
                }).catch(() => load(urlIndex + 1));
            };
            return load();
        },
        setSubDepends(index, res) {
            try {
                let json = jsyaml.load(res?.data?.data?.manifest);
                if (json?.platform?.depends?.length) {
                    let d = json?.platform?.depends;
                    d.map(i => {
                        this.subDependsList[index][i.identifie] = i.name;
                    })
                }
            } catch { }
        },
        getDependsList() {
            myAxios.get('/respo/list?limit=999').then(res => {
                let list = res.data?.data?.list || []
                this.mergeDependsList(this.normalizeDependList(list));
            })
        },

        async computedAppPort(containers = this.form.containers) {
            const normalizePorts = (ports) => {
                if (!Array.isArray(ports)) { ports = ports ? [ports] : [] }
                return [...new Set(ports.map(i => {
                    if (typeof i === 'object' && i !== null) {
                        return i.port ?? i.containerPort ?? '';
                    }
                    return i;
                }).filter(i => i !== '' && i !== undefined && i !== null).map(i => String(i)))]
            };
            let ports = {};
            let arr = [];
            if (containers) {
                containers?.map?.(i => {
                    arr = arr.concat(normalizePorts(i?.ports || []));
                    if (i?.port || i?.containerPort) {
                        arr = arr.concat(normalizePorts([i.port ?? i.containerPort]));
                    }
                })
            }
            let legacyContainer = this.json?.platform?.container;
            if (legacyContainer) {
                arr = arr.concat(normalizePorts(legacyContainer?.ports || []));
                if (legacyContainer?.port || legacyContainer?.containerPort) {
                    arr = arr.concat(normalizePorts([legacyContainer.port ?? legacyContainer.containerPort]));
                }
            }
            arr = arr.concat(normalizePorts(this.json?.platform?.port || []));
            arr = [...new Set(arr)];
            ports[this.identifie] = arr;

            if (this.option?.app_ports?.length) {
                this.option.app_ports.map(i => {
                    if (i.name == this.identifie) { return; }
                    ports[i.name] = normalizePorts(i.port || i.ports || []);
                })
            }
            this.app_ports = ports;

            let names = [{
                id: this.identifie,
                name: this.identifie,
                title: this.form.name || this.identifie,
            }];

            if (this.option?.app_ports?.length) {
                names = names.concat(this.option.app_ports.filter(i => i.name != this.identifie).map(i => {
                    return {
                        id: i.name,
                        name: i.name,
                        title: i.title || i.name,
                    }
                }));
            }
            this.app_names = names;

        },

        openSpEdit() {
            this.spEdit.show = true;
            let values = this.form.startParams.map(i => `${i.name || ''}=${i.values_text || ''} #${i.title || ' '}:${i.description || ' '}:${i.required ? 1 : 0}:${i.module_name || ''}`);
            this.spEdit.values = values.join('\n');
        },

        submitSpEdit() {
            let values = this.spEdit.values.split('\n');
            let arr = [];
            values.map(i => {
                let match = i.match(/^([^\s=#]+)\s*=\s*([^\s=#]+)\s*(#([^:：\s]*)\s*[:：]\s*([^:：\s]*)\s*[:：]\s*([^:：\s]*)\s*[:：]\s*([^:：\s]*))?$/);
                if (!match) { return }
                arr.push({
                    name: match[1],
                    values_text: match[2],
                    title: match[4] || '',
                    description: match[5] || '',
                    required: match[6] == '1',
                    module_name: match[7] || '',
                    type: 'text',
                });
            });
            this.form.startParams = arr;

            this.form.startParams.forEach((i, index) => {
                if (i.module_name == 'w7_mysql' || i.module_name == 'w7_mysql5') {
                    i.mark = i.module_name == 'w7_mysql' ? 'mysql8' : 'mysql5';
                    this.form[i.module_name == 'w7_mysql' ? 'mysql8' : 'mysql5'] = true;

                    let next = this.form.startParams[index + 1];
                    if (next && next.name == 'MYSQL_DATABASE' && !next.module_name) {
                        next.mark = i.mark;
                    }
                }
                if (i.module_name == 'w7_redis') { i.mark = 'redis'; this.form.redis = true; }
                if (i.module_name == 'w7_mongodb') { i.mark = 'mongodb6'; this.form.mongodb6 = true; }
            });
            this.spEdit.show = false;
            this.getStart();
        },
        delDepend(index) {
            if (this.form.dependsIn.length - 1 < index) {
                this.form.depends.splice(index - (this.form.dependsIn.length || 0), 1);
            } else {
                this.form.dependsIn.splice(index, 1);
            }
            this.changeForm();
        },

        openSpDesc(item) {
            this.spDesc.show = true;
            this.spDesc.item = item;
            this.spDesc.value = item.description || '';
        },
        submitSpDesc() {
            if (this.spDesc.item) {
                this.spDesc.item.description = this.spDesc.value || '';
                this.getStart();
            }
            this.spDesc.show = false;
        },
        openAddDepend() {
            this.dependForm.show = true;
            let identifie = this.json?.platform?.baseInfo?.identifie || this.json?.application?.identifie;
            this.dependForm.identifie_before = identifie.match(/^([^-]+)-(.+)$/)?.[1] || '';
        },
        addDepend() {
            this.$refs.depend.validate((errors) => {
                if (errors) { return }
                if (!this.dependForm.identifie_before || !this.dependForm.identifie_last) {
                    messageWarning('标识请填写完整');
                    return;
                }
                let o = {
                    identifie: this.dependForm.identifie,
                    required: this.dependForm.required,
                    name: this.dependForm.name,
                };

                if (this.dependForm.editIndex >= 0) {
                    this.form.dependsIn.splice(this.dependForm.editIndex, 1, o);
                } else {
                    this.form.dependsIn.push(o);
                }
                this.changeForm();

                this.dependForm.show = false;

                let file = o.identifie + '/manifest.yaml';
                let cont = `application:
    name: ${o.name}
    identifie: ${o.identifie}
    description: ''
    author: ''
platform:
    container:
        containerPort: 80
`
                this.$refs.formref.validate(async (errors) => {
                    if (errors) { messageWarning('必填项不能为空'); return }

                    if (this.form.type == 'helm') {
                        try {
                            delete this.json.platform['container-v2']
                            delete this.json.platform['volumes']
                            delete this.json.platform['volumeClaimTemplates']
                            delete this.json.platform.ingress
                            delete this.json.platform.runtimeClassName
                        } catch { }
                    } else if (this.form.type == 'tradition') {
                        try {
                            delete this.json.platform['volumes']
                            delete this.json.platform['volumeClaimTemplates']
                            delete this.json.platform.ingress
                            delete this.json.platform.runtimeClassName
                        } catch { }
                        this.applyPlatformShells();
                    } else if (this.form.type == 'docker') {

                        this.applyPlatformShells();

                        this.json.platform.ingress = (this.form.ingress || []).map(i => ({
                            name: i.name,
                            routes: this.formatIngressRoutes(i.routes),
                        }));
                    }

                    this.yaml = jsyaml.dump(this.json);

                    this.$emit('addfile', this.json, this.yaml, {
                        file: file,
                        cont: cont,
                    });
                });
            });
        },
        changeDepend(index, data) {
            if (!this.form.dependsIn?.[index]) { return }
            this.form.dependsIn[index] = data;
            this.changeForm();
            this.submit({ stop: true });
        },


        checkStartParams(check, mark, arr) {
            if (check) {
                let hasmark = false;
                this.form.startParams.forEach(i => { if (i.mark == mark) { hasmark = true } });
                if (!hasmark) {
                    arr.forEach(i => {
                        this.form.startParams.unshift(i);
                    });
                }
            } else {
                for (let i = this.form.startParams.length - 1; i >= 0; i--) {
                    if (this.form.startParams[i].mark == mark) {
                        this.form.startParams.splice(i, 1);
                    }
                }
            }
            this.getStart();
        },

        uplogo(event) {
            this.logofile = event.target.files[0];
            if (!this.logofile) { return }
            let formdata = new FormData();
            formdata.append('identifie', this.form.author + '-' + this.form.identifie);
            formdata.append('file', this.logofile);
            myAxios.post('/respo/icon', formdata).then(res => {
                messageSuccess('添加成功');
                this.logoimg = res.data?.data?.url;
                if (!/^https?:\/\//.test(this.logoimg)) {
                    this.logoimg = this.baseurl + this.logoimg;
                }
                this.logoimg = this.logoimg + '?time=' + Date.now();
            });
        },

        deleteTag(index) {
            myAxios.post('/respo/tag/delete', {
                tagId: this.form.tags[index].id,
            }).then(res => {
                messageSuccess('删除成功');
                this.form.tags.splice(index, 1);
            })
        },

        addTag() {
            if (!this.form.taginput) { return }
            let tag = this.form.taginput;
            myAxios.post('/respo/tag/add', {
                identifie: this.form.author + '-' + this.form.identifie,
                name: tag,
            }).then(res => {
                messageSuccess('添加成功');
                this.form.tags.push({ name: tag, id: res.data.id });
                this.form.taginput = '';
                this.getTag();
            });
        },

        getTag() {
            myAxios.get('/respo/list?status=1&status=2&status=99&limit=1000').then(res => {
                let list = res.data?.data?.list || [];
                let find = list.find(i => i.identifie == (this.form.author + '-' + this.form.identifie))
                if (find?.icon) {
                    this.logoimg = find.icon;
                    if (this.logoimg && !/^https?:\/\//.test(this.logoimg)) {
                        this.logoimg = this.baseurl + this.logoimg;
                    }
                }
                if (find?.tag) {
                    this.form.tags = find.tag;
                }
            })
        },

        uploadSuccess(data, filename) {
            if (data?.data?.url) {
                let url = data?.data?.url;
                this.zip.name = url.match(/[^\/]+$/)[0];
                if (!this.json.source) { this.json.source = {}; }
                this.json.source.type = 'zip';
                this.json.source.url = url;
                this.zip.url = url;
                if (this.form.type == 'environment') {
                    this.form.startParams = this.environmentStartParams();
                    this.ensureEnvironmentContainerDefaults();
                    this.json.platform.startParams = this.serializeStartParams();
                }
                this.setYaml();
            }
        },

        helmUploadSuccess(data, filename) {
            let url = data?.data?.url;
            this.form.helm.chartName2 = url;
            this.changeForm();
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
                    if (this.form.type == 'environment') {
                        this.form.startParams = this.environmentStartParams();
                        this.ensureEnvironmentContainerDefaults();
                        this.json.platform.startParams = this.serializeStartParams();
                    }
                    this.setYaml();
                },
            });
        },
        uploadHelmChart(data) {
            if (data?.url || data?.data?.url) {
                let url = data?.url || data?.data?.url;
                this.form.helm.chartName2 = url;
                this.changeForm();
            }
        },
        init(data) {
            if (!data) { return }
            this.json = jsyaml.load(data);

            this.json.platform = this.json?.platform || {}


            if (!this.option?.pureManifest) {
                this.json.platform.baseInfo = this.json.platform.baseInfo || {};
                this.json.platform.baseInfo.identifie = this.json.platform.baseInfo.identifie || this.json.application?.identifie || '';
                this.json.platform.baseInfo.name = this.json.platform.baseInfo.name || this.json?.application?.name || '';
                this.json.platform.baseInfo.description = this.json.platform.baseInfo.description || this.json?.application?.description || '';
            }
            this.vtitle = this.json?.application?.name;
            this.initJSON();
            this.changeForm();
            this.computedAppPort();
        },

        initJSON() {
            this._initializing = true;
            let j = this.json;
            if (j.application) {
                if (/^[^-]+-.+$/.test(j.application?.identifie)) {
                    let i = j.application.identifie;
                    j.application.author = i.match(/^([^-]+)-(.+)$/)[1];
                }
                this.form.once = j.application?.once || false;
                this.form.type = j.application.type === 'front' ? 'docker' : (j.application.type || 'docker');
            }
            this.initialApplicationType = this.form.type;
            this.currentApplicationType = this.form.type;
            for (let i in j.bindings) {
                let o = j.bindings[i];
                if (!o?.menu?.length) {
                    o.menu = [{ displayorder: 0, do: 'home', title: '首页', icon: 'a-shouye', is_default: 1 }];
                }
            }

            this.form.name = j?.application?.name;
            if (j.application?.identifie && /^[^-]+-.+$/.test(j.application.identifie)) {
                this.form.identifie = j.application.identifie.match(/^([^-]+)-(.+)$/)[2];
            } else {
                this.form.identifie = j?.application?.identifie;
            }
            this.form.author = j?.application?.author;
            if (!this.form.identifie && !this.form.author && this.identifie) {
                let arr = this.identifie.match(/^([^-]+)-(.+)$/);
                if (arr?.length) {
                    this.form.identifie = arr[2];
                    this.form.author = arr[1];
                }
            }

            this.form.description = j?.application?.description;
            const gatewayPlugin = j?.platform?.gatewayPlugin || {};
            const gatewayPluginRuntime = gatewayPlugin.runtime || {};
            const gatewayPluginRuntimeConfig = gatewayPluginRuntime.config || {};
            const gatewayPluginSupports = gatewayPlugin.supports || {};
            this.form.gatewayPluginCategory = gatewayPlugin.category || 'custom';
            this.form.gatewayPluginDriver = gatewayPluginRuntime.driver || 'higress-wasm/v1';
            this.form.gatewayPluginUrl = gatewayPluginRuntimeConfig.url || '';
            this.form.gatewayPluginPhase = gatewayPluginRuntimeConfig.phase || 'UNSPECIFIED_PHASE';
            this.form.gatewayPluginPriority = Number(gatewayPluginRuntimeConfig.priority || 0);
            this.form.gatewayPluginSupportGlobal = gatewayPluginSupports.global !== false;
            this.form.gatewayPluginSupportRule = gatewayPluginSupports.rule === true;
            this.form.gatewayPluginDefaultEnabled = gatewayPlugin.defaultEnabled !== false;
            this.form.gatewayPluginDefaultConfig = JSON.stringify(gatewayPlugin.defaultConfig || {}, null, 2);
            this.applyEnvironmentAnnotationForm(j?.application?.annotation || {});
            this.applySystemImageAnnotationForm(j?.application?.annotation || {});
            this.form.systemImageTemplate = j?.platform?.['container-v2']?.[0]?.image || '';

            if (!this.option?.pureManifest && this.form.type != 'gateway-plugin') {
                this.form.name = j?.platform?.baseInfo?.name || '';
                let identifie = j?.platform?.baseInfo?.identifie || '';
                this.form.identifie = identifie.match(/^([^-]+)-(.+)$/)?.[2] || '';
                this.form.author = identifie.match(/^([^-]+)-(.+)$/)?.[1] || '';
                this.form.description = j?.platform?.baseInfo?.description || '';
            }

            if (j?.source?.type == 'zip') {
                this.zip.codetype = 'zip';
                this.zip.url = j.source.url;
                this.zip.name = j.source.url.replace(/.*\//, '');
            }
            if (j?.web?.type == 'zip') {
                this.web.url = j.web.url;
                this.web.name = j.web.url.replace(/.*\//, '');
            }

            if (j.platform) {

                this.form.environmentName = j.platform?.tradition?.environmentName || '';
                this.form.environmentGoodsId = Number(j.platform?.tradition?.environmentGoodsId || 0);
                this.form.environmentVersion = j.platform?.tradition?.environmentVersion || '';
                this.form.environmentImageTemplate = j.platform?.tradition?.environmentImageTemplate || '';
                this.form.installType = j.platform?.tradition?.installType || traditionInstallTypes.site;
                if (['system-image', 'environment'].includes(this.form.type)) {
                    let command = this.form.type == 'environment'
                        ? this.getEnvironmentMainContainer()?.command
                        : j.platform?.['container-v2']?.[0]?.command;
                    this.form.cmd = Array.isArray(command) && command.length
                        ? command.map(item => String(item))
                        : [''];
                } else {
                    this.form.cmd = [''];
                }

                this.form.ingress = JSON.parse(JSON.stringify(j.platform?.ingress || []));
                this.form.shell = JSON.parse(JSON.stringify(j.platform?.shells || j.platform?.['container-v2']?.[0]?.shells || []));
                this.form.build_context = j.platform?.['container-v2']?.[0]?.build?.context || '';
                this.form.containers = j.platform?.['container-v2'] || [];
                this.fillDefaultShellContainer();

                this.containerPluginData = {
                    ...this.containerPluginData,
                    runtimeClassName: j.platform?.runtimeClassName || '',
                    kind: j?.platform?.workload?.type || '',
                };

                let startParams = j?.platform?.startParams;
                this.form.startParams = JSON.parse(JSON.stringify(startParams?.length ? startParams : []))
                    .filter(item => item?.mark !== 'environment-release'
                        && !isDerivedDependencyReleaseStartParam(item)
                        && !isLegacyTraditionInstallDirectoryStartParam(item, this.form.type));

                if (this.form.type == 'system-image') {
                    this.form.startParams = this.systemImageStartParams();
                    this.form.storage = true;
                    this.ensureSystemImageContainer();
                } else if (this.form.type == 'environment') {
                    this.ensureEnvironmentContainerDefaults();
                    this.form.startParams = this.environmentStartParams();
                }

                if (this.form.startParams?.length) {
                    this.form.startParams.forEach((i, index) => {
                        if (i.module_name == 'w7_mysql' || i.module_name == 'w7_mysql5') {
                            i.mark = i.module_name == 'w7_mysql' ? 'mysql8' : 'mysql5';
                            this.form[i.module_name == 'w7_mysql' ? 'mysql8' : 'mysql5'] = true;

                            let next = this.form.startParams[index + 1];
                            if (next && next.name == 'MYSQL_DATABASE' && !next.module_name) {
                                next.mark = i.mark;
                            }
                        }

                        if (i.name == 'DOMAIN_URL') { i.mark = 'domain'; this.form.domain = true; }
                        if (i.module_name == 'w7_redis') { i.mark = 'redis'; this.form.redis = true; }
                        if (i.module_name == 'w7_mongodb') { i.mark = 'mongodb6'; this.form.mongodb6 = true; }
                        if (['global.cluster.storageRWmode', 'global.cluster.storageSize', 'global.cluster.storageClassName'].includes(i.name)) {
                            i.mark = 'storage';
                            this.form.storage = true;
                        }
                    })
                }

                this.form.dependsIn = j.platform?.depends?.filter(i => i.type != 'out') || [];
                this.form.dependsIn.map(i => i.type = 'in');
                this.form.depends = j.platform?.depends?.filter(i => i.type == 'out') || [];

                this.form.depends.map((item, index) => {
                    if (this.isTraditionEnvironmentDependency(item)) {
                        item.temporary = true;
                        item.required = true;
                        item.multipleInstances = true;
                    }
                    if (this.form.type == 'tradition' && this.environmentList?.length) {
                        let now = this.environmentList.find(i => i.identifie == this.form.environmentName);
                        if (now) {
                            if (item.identifie == now.identifie && item.name == now.name) {
                                item.temporary = true;
                            }
                        }
                    }
                    this.getSubDepends(index);
                })

                let depend_yamls = j.platform?.helm?.depend_yamls || [];
                depend_yamls = depend_yamls.map(i => {
                    return {
                        name: i?.name,
                        nameInput: i?.name?.replace?.(/\.yaml$/, ''),
                        yaml: i.yaml,
                    }
                })

                this.form.helm = {
                    helmtype: j.platform?.helm?.repository ? '1' : '2',
                    repository: j.platform?.helm?.repository || '',
                    chartName: j.platform?.helm?.chartName || '',
                    chartName2: '',
                    version: j.platform?.helm?.version || '',
                    kv: j.platform?.helm?.kv || [],

                    depend_yamls: depend_yamls,
                    useHelm: Boolean(j.platform?.helm?.chartName),
                };
                if (!j.platform?.helm?.repository) {
                    this.form.helm.chartName2 = j?.platform?.helm?.chartName || '';
                    this.form.helm.chartName = '';
                } else {
                    this.getChartInfo();
                }
            }
            if (this.form.type != 'helm' && this.form.type != 'tradition' && this.form.ingress?.length) {
                this.form.domain = true;
                this.disabledDomainStartParams = true;
            }
            this.environmentPreviousDepends = null;
            this.syncEnvironmentDependency();
            this.syncSystemImageDependency();
            this._initializing = false;
        },
        getPanelData() {
            return new Promise((resolve, reject) => {
                emitWujieEvent('submit' + this.wujieId, (data) => {
                    resolve(data)
                })
            })
        },
        getValidShells() {
            return (this.form.shell || []).filter(i => i.type && i.shell);
        },
        applyPlatformShells() {
            this.json.platform = this.json.platform || {};
            const shells = this.getValidShells();
            if (!shells.length) {
                delete this.json.platform.shells;
                return;
            }
            this.json.platform.shells = shells;
            if (this.json.platform?.['container-v2']?.[0]) {
                delete this.json.platform['container-v2'][0].shells;
            }
        },
        parseGatewayPluginDefaultConfig(showMessage = false) {
            try {
                const config = JSON.parse(this.form.gatewayPluginDefaultConfig || '{}');
                if (config === null || typeof config !== 'object' || Array.isArray(config)) {
                    throw new Error('默认配置必须是 JSON 对象');
                }
                return config;
            } catch (error) {
                if (showMessage) {
                    messageWarning(error?.message || '默认配置 JSON 格式错误');
                }
                return null;
            }
        },
        submit(otherData, callback) {

            this.$refs.formref.validate(async (errors) => {
                if (errors) {
                    messageWarning(this.form.type == 'environment'
                        ? '请检查运行环境配置中的错误项'
                        : '必填项不能为空');
                    return;
                }

                if (!this.validateEnvironmentStorage()) { return }

                if (this.form.type == 'gateway-plugin') {
                    if (!this.form.gatewayPluginSupportGlobal && !this.form.gatewayPluginSupportRule) {
                        messageWarning('请至少选择一种支持范围');
                        return;
                    }
                    const defaultConfig = this.parseGatewayPluginDefaultConfig(true);
                    if (defaultConfig === null) { return }
                    this.json.platform.gatewayPlugin.defaultConfig = defaultConfig;
                }

                try {
                    this.syncEnvironmentDependency();
                    this.syncSystemImageDependency();
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
                    return;
                }

                if (this.form.type == 'helm') {
                    try {
                        delete this.json.platform['container-v2']
                        delete this.json.platform['volumes']
                        delete this.json.platform['volumeClaimTemplates']
                        delete this.json.platform.ingress
                        delete this.json.platform.runtimeClassName
                    } catch { }
                } else if (this.form.type == 'tradition') {
                    try {
                        delete this.json.platform['volumes']
                        delete this.json.platform['volumeClaimTemplates']
                        delete this.json.platform.ingress
                        delete this.json.platform.runtimeClassName
                    } catch { }
                    this.applyPlatformShells();
                } else if (this.form.type == 'environment') {
                    this.form.startParams = this.environmentStartParams();
                    this.ensureEnvironmentContainerDefaults();
                    this.applyEnvironmentCommand();
                    this.applyPlatformShells();
                } else if (this.form.type == 'docker') {

                    this.applyPlatformShells();

                    this.json.platform.ingress = (this.form.ingress || []).map(i => ({
                        name: i.name,
                        routes: this.formatIngressRoutes(i.routes),
                    }));
                }

                if (this.form.type != 'gateway-plugin') {
                    if (this.form.type == 'system-image') {
                        this.ensureSystemImageContainer();
                    }
                    this.json.platform.startParams = this.serializeStartParams();
                }
                this.yaml = jsyaml.dump(this.json);

                this.$emit('complete', this.json, this.yaml, otherData, callback);
            });
        },
        changeFormtype(nextType) {
            const previousType = this.currentApplicationType || this.initialApplicationType;
            if (!nextType || nextType == previousType) { return }
            const typeNames = {
                docker: '原生应用',
                tradition: '传统应用',
                helm: 'K8sYaml',
                environment: '运行环境',
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
            this.json.application = this.json.application || {};
            this.json.application.annotation = this.filterAnnotationsForType(
                this.json.application.annotation || {},
                nextType,
            );
            this.currentApplicationType = nextType;
            if (this.form.type == 'gateway-plugin') {
                this.form.once = true;
            }
            if (['environment', 'gateway-plugin', 'system-image'].includes(this.form.type)) {
                this.loadFormulaSetting().catch(() => { });
            }
            if (!['light', 'system-image'].includes(this.form.type) && this.zip.url) {
                if (!this.json.source) { this.json.source = {}; }
                this.json.source.type = 'zip';
                this.json.source.url = this.zip.url;
            }
            if (!['tradition', 'system-image'].includes(this.form.type) && this.web.url) {
                if (!this.json.web) { this.json.web = {}; }
                this.json.web.type = 'zip';
                this.json.web.url = this.web.url;
            }

            if (this.form.type == 'light') {

                if (this.json.source) { delete this.json.source; }
            }
            if (this.form.type == 'tradition') {
                this.getStart();

                if (this.json.web) { delete this.json.web; }
            }
            if (this.form.type == 'system-image') {
                this.syncSystemImageConfig();
            }
            this.syncEnvironmentDependency();
            this.syncSystemImageDependency();

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
            if (j.application) {
                if (this.option?.pureManifest) {
                    j.application.name = this.form.name;
                    j.application.identifie = this.form.author + '-' + this.form.identifie;
                    j.application.description = this.form.description;
                }

                j.application.author = this.form.author;
                j.application.theme = this.form.theme;
                j.application.type = this.form.type;
                if (this.form.type == 'gateway-plugin') {
                    this.form.once = true;
                }
                j.application.once = Boolean(this.form.once);
                if (this.form.type != 'tradition') {
                    this.form.language = '';
                }
            }
            if (this.form.type == 'gateway-plugin') {
                const defaultConfig = this.parseGatewayPluginDefaultConfig(false);
                const currentDefaultConfig = j.platform?.gatewayPlugin?.defaultConfig
                    || {};
                const currentConfigSchema = j.platform?.gatewayPlugin?.configSchema
                    || {};
                j.platform = {
                    gatewayPlugin: {
                        category: this.form.gatewayPluginCategory || 'custom',
                        defaultEnabled: Boolean(this.form.gatewayPluginDefaultEnabled),
                        supports: {
                            global: Boolean(this.form.gatewayPluginSupportGlobal),
                            rule: Boolean(this.form.gatewayPluginSupportRule),
                        },
                        defaultConfig: defaultConfig === null
                            ? currentDefaultConfig
                            : defaultConfig,
                        ...(Object.keys(currentConfigSchema).length ? { configSchema: currentConfigSchema } : {}),
                        runtime: {
                            driver: this.form.gatewayPluginDriver || 'higress-wasm/v1',
                            config: {
                                url: this.form.gatewayPluginUrl,
                                phase: this.form.gatewayPluginPhase || 'UNSPECIFIED_PHASE',
                                priority: Number(this.form.gatewayPluginPriority || 0),
                            },
                        },
                    },
                    depends: this.form.dependsIn.concat(this.form.depends),
                };
                delete j.source;
            } else {
                if (j.platform) { delete j.platform.gatewayPlugin; }
            }
            j.platform = j.platform || {};

            if (this.form.type == 'system-image') {
                this.form.systemImageVersions = this.normalizeEnvironmentVersions(this.form.systemImageVersions);
                this.form.startParams = this.systemImageStartParams();
                this.ensureSystemImageContainer();
            } else if (this.form.type == 'environment') {
                this.form.environmentImageVersion = this.normalizeEnvironmentVersions(this.form.environmentImageVersion);
                this.form.startParams = this.environmentStartParams();
                this.ensureEnvironmentContainerDefaults();
                this.applyEnvironmentRebootRestoreConfig();
                this.applyEnvironmentCommand();
            } else {
                delete j.platform.hostUsers;
                if (j.platform.runtimeClassName == 'sysbox-runc') {
                    delete j.platform.runtimeClassName;
                }
            }

            if (this.form.type == 'tradition') {
                let environmentLanguage = j.platform?.tradition?.environmentLanguage || '';
                try {
                    const selectedEnvironment = this.environmentList
                        ?.find?.(i => i.identifie == this.form.environmentName);
                    environmentLanguage = selectedEnvironment?.environment_language || environmentLanguage;
                    this.form.environmentGoodsId = Number(
                        selectedEnvironment?.goods_id
                        || selectedEnvironment?.goodsId
                        || selectedEnvironment?.id
                        || this.form.environmentGoodsId
                        || 0
                    );
                } catch { }
                const traditionInstall = normalizeTraditionInstall(this.form);
                this.form.installType = traditionInstall.installType;
                j.platform.tradition = {
                    environmentName: this.form.environmentName,
                    environmentGoodsId: this.form.environmentGoodsId,
                    environmentVersion: this.form.environmentVersion,
                    environmentLanguage: environmentLanguage,
                    environmentImageTemplate: this.form.environmentImageTemplate,
                    installType: traditionInstall.installType,
                }
            }

            if (j.platform && this.form.type != 'gateway-plugin') {
                if (this.form.type !== 'helm' || true) {

                    if (!this.option?.pureManifest) {
                        j.platform.baseInfo = j.platform.baseInfo || {};
                        j.platform.baseInfo.name = this.form.name;
                        j.platform.baseInfo.identifie = (this.form.author && this.form.identifie) ? (this.form.author + '-' + this.form.identifie) : '';
                        j.platform.baseInfo.description = this.form.description;
                    }
                }


                let dependencies = this.form.dependsIn.concat(this.form.depends);
                if (this.form.type == 'tradition') {
                    dependencies = applyTraditionEnvironmentDependencyStartParams(
                        dependencies,
                        this.form.environmentName,
                        this.form.environmentVersion,
                    );
                }
                j.platform.depends = dependencies;
                j.platform.startParams = this.serializeStartParams();

                if (this.form.type == 'helm') {

                    delete j.source
                    try {
                        delete j.platform?.['container-v2']?.image;
                    } catch { }

                    let depend_yamls = [];
                    if (this.form.helm?.depend_yamls?.length) {
                        depend_yamls = this.form.helm?.depend_yamls?.filter(i => i.nameInput && i.yaml).map(i => {
                            return {
                                name: i.nameInput + '.yaml',
                                yaml: i.yaml,
                            }
                        })
                    }

                    let kv = this.form?.helm?.kv?.filter(i => i.name && i.value).map(i => ({ name: i.name, value: i.value })) || [];
                    if (this.form.helm.helmtype == '1') {
                        j.platform.helm = {
                            repository: this.form.helm.repository,
                            chartName: this.form.helm.chartName,
                            version: this.form.helm.version,
                            depend_yamls: depend_yamls,
                            kv: kv,
                        }
                    } else {
                        j.platform.helm = {
                            chartName: this.form.helm.chartName2,
                            kv: kv,
                            depend_yamls: depend_yamls,
                        }
                    }
                } else {
                    delete j.platform.helm;
                }
                if (this.form.type == 'helm') {
                    delete j.platform.shells;
                } else {
                    this.applyPlatformShells();
                }
                if (this.form.type == 'docker') {

                    this.json.platform.ingress = (this.form.ingress || []).map(i => ({
                        name: i.name,
                        routes: this.formatIngressRoutes(i.routes),
                    }));
                }
            }
            this.setYaml();
        },
        setYaml() {
            this.yaml = jsyaml.dump(this.json, {
                indent: 2,
                sortKeys: (a, b) => {
                    if (b == 'menu') { return -1; }
                    return a > b ? 1 : -1;
                },
            });
            this.yamlDom = `<pre class='pre'><code class='language-yaml'>${this.escapeHtml(this.yaml)}</code></pre>`;
            this.$nextTick(() => {
                window.hljs.highlightAll();
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
            this.$nextTick(() => {
                this.setYaml();
            });
        },
        getStart() {
            this.json.platform.startParams = this.serializeStartParams();
            this.setYaml();
        },
        download() {
            let file = new File([this.yaml], 'manifest.yaml', { type: 'text/plain' });
            this.downloadUrl = URL.createObjectURL(file);
        },
        onekeyCopy(text) {
            if (0 && navigator.clipboard) {
                navigator.clipboard.writeText(text);
            } else {
                var textarea = document.createElement('textarea');
                document.body.appendChild(textarea);
                textarea.style.position = 'fixed';
                textarea.style.clip = 'rect(0 0 0 0)';
                textarea.style.top = '10px';
                textarea.value = text;
                textarea.select();
                document.execCommand('copy', true);
                document.body.removeChild(textarea);
            }
            messageSuccess("复制成功")
        },
    },
}
</script>
<style scoped>
.environment-config-spin {
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

.depend-picker-toolbar {
    width: 320px;
    margin-bottom: 12px;
}

.depend-picker-search-action {
    display: inline-flex;
    align-items: center;
    color: #666;
    cursor: pointer;
}

.depend-picker-table {
    margin-top: 8px;
}

.depend-picker-product {
    min-width: 0;
    cursor: pointer;
}

.depend-picker-product-name,
.depend-picker-product-identifie {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.depend-picker-product-name {
    color: #333;
}

.depend-picker-product-identifie {
    margin-top: 2px;
    color: #999;
    font-size: 12px;
}

.depend-picker-footer {
    display: flex;
    justify-content: flex-end;
    margin-top: 16px;
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
