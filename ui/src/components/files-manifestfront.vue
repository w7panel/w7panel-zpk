<template>
    <div class="df content">
        <div class="fc" style="padding-left:0; overflow:auto;">
            <div v-if="noRole && !isCreate">
                <a-empty :image-size="200" description="">
                    <span class="c-99">暂无数据，点击</span>
                    <span class="cursor c-blue" @click="isCreate = true;">创建应用配置</span>
                </a-empty>
            </div>
            <div v-else>
                <a-form ref="formref" :model="form" :rules="rules" label-align="left" :label-col-props="{ span: 3 }"
                    :wrapper-col-props="{ span: 21 }" class="form manifest-form">
                    <div class="bg-white com-line df" style="margin-bottom:20px;">
                        <div class="fc">
                            <div class="c-00-6 df ai-c">基础配置</div>
                            <a-form-item class="mt-16" label="名称" field="name">
                                <a-input v-model="form.name" size="large" :disabled="!noPlatform" style="width:500px;"
                                    @change="changeForm" placeholder="请输入"></a-input>
                            </a-form-item>
                            <a-form-item label="标识" field="identifie">
                                <div class="df jc-b" style="width:500px;">
                                    <w7-identifie v-model:author="form.author" v-model:identifie="form.identifie"
                                        @change="onChange" disabled />
                                </div>
                            </a-form-item>
                            <a-form-item label="描述">
                                <div class="df df-c">
                                    <a-input v-model="form.description" :disabled="!noPlatform" size="large"
                                        style="width:500px;" placeholder="请输入应用描述" @change="changeForm"></a-input>
                                </div>
                            </a-form-item>
                        </div>
                    </div>

                    <div class="bg-white manifest-front-panel">
                        <div class="c-00-6 df ai-c">前端配置</div>
                        <div class="manifest-front-panel-body">
                            <div class="roles-box">
                                <a-form-item label="前端包上传">
                                    <template #label>
                                        <div class="df ai-c">
                                            前端包上传
                                            <a-tooltip position="tl"
                                                :content="form.type == 'environment'
                                                    ? '压缩包根目录就是前端构建产物目录，请进入构建产物目录后压缩，不要把外层目录一起压入。例如：cd dist && zip -r frontend.zip .。环境应用的菜单配置可以不设置。'
                                                    : '压缩包根目录就是前端构建产物目录，请进入构建产物目录后压缩，不要把外层目录一起压入。例如：cd dist && zip -r frontend.zip .'">
                                                <ArcoIcon name="icon-41" :size="16" />
                                            </a-tooltip>
                                        </div>
                                    </template>
                                    <div class="manifest-front-upload-section">
                                        <files-upload @success="webUploadSuccess">
                                            <div v-if="web.name" class="upfilebox df df-c ai-c jc-c">
                                                <img src="@/assets/img/zip.png" alt=""
                                                    style="width:60px;height:60px;display:block;" />
                                                <div class="df ai-c mt-20">
                                                    <icon-check-circle-fill class="c-green file-status-icon" />
                                                    <div class="fs-14 c-33"
                                                        style="vertical-align:middle;max-width:200px;overflow:hidden;text-overflow:ellipsis;">
                                                        {{ web.name }}</div>
                                                </div>
                                                <div class="mask df df-c ai-c jc-c">
                                                    <a-button type="primary">重新上传</a-button>
                                                </div>
                                            </div>
                                            <div v-else class="upfilebox df df-c ai-c jc-c">
                                                <div class="df df-c ai-c">
                                                    <svg class="uploadicon upload-cloud-icon c-99"
                                                        xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024"
                                                        aria-hidden="true">
                                                        <path fill="currentColor"
                                                            d="M544 864V672h128L512 480 352 672h128v192H320v-1.6c-5.376.32-10.496 1.6-16 1.6A240 240 0 0 1 64 624c0-123.136 93.12-223.488 212.608-237.248A239.81 239.81 0 0 1 512 192a239.87 239.87 0 0 1 235.456 194.752c119.488 13.76 212.48 114.112 212.48 237.248a240 240 0 0 1-240 240c-5.376 0-10.56-1.28-16-1.6v1.6z">
                                                        </path>
                                                    </svg>
                                                    <span class="uploadbtn df ai-c">
                                                        <icon-upload class="uploadicon c-33" />
                                                        <span class="lh-1 c-33">上传代码包</span>
                                                    </span>
                                                </div>
                                            </div>
                                        </files-upload>
                                    </div>
                                </a-form-item>
                                <a-form-item label="开启管理端" style="margin:0;">
                                    <a-checkbox v-for="role in panelRoleOptions" class="mr-8" :key="role.name"
                                        :model-value="Boolean(cdrole[role.name])"
                                        @change="checked => togglePanelRole(checked, role)">
                                        {{ role.title }}
                                    </a-checkbox>
                                </a-form-item>
                            </div>

                            <div v-for="(r, rindex) in form.role.filter(i => i.support == form.menu_type)" :key="rindex"
                                class="role mt-16">
                                <div class="df ai-c jc-b greybox-header">
                                    <div class="df ai-c">
                                        <div v-if="roleEdit.index == rindex" class="df">
                                            <span>名称：</span>
                                            <a-input v-model="roleEdit.title" style="width:140px;"
                                                placeholder="请输入名称"></a-input>
                                            <span class="ml-20">标识：</span>
                                            <a-input v-model="roleEdit.name" style="width:140px;"
                                                :disabled="r.name == 'founder' || r.name == 'super'"
                                                placeholder="请输入标识"></a-input>
                                        </div>
                                        <div v-else-if="r.support === 'thirdparty_cd'" class="df ai-c">
                                            <span class="lh-1 mr-20">{{ r.title }}</span>
                                            <a-checkbox v-if="form.type !== 'gateway-plugin'" :model-value="r.load_mode === 'iframe'"
                                                @change="v => { r.load_mode = v ? 'iframe' : 'static_hosting'; changeLoadMode(r); }">
                                                <span class="c-66">支持iframe</span></a-checkbox>
                                        </div>
                                        <div v-else class="df ai-c cursor">
                                            <div @click="roleEdit = { index: rindex, title: r.title, name: r.name }"
                                                class="mr-20">
                                                <span class="lh-1">{{ r.title }}</span>
                                                <icon-edit v-if="r.support != 'thirdparty_cd'" class="role-edit-icon" />
                                            </div>
                                            <a-checkbox v-if="form.type !== 'gateway-plugin'" :model-value="r.load_mode === 'iframe'"
                                                @change="v => { r.load_mode = v ? 'iframe' : 'static_hosting'; changeLoadMode(r); }"><span
                                                    class="c-66">支持iframe</span></a-checkbox>
                                        </div>

                                        <div v-if="roleEdit.index == rindex" class="ml-40 c-blue cursor lh-1"
                                            style="text-wrap: nowrap;" @click="submitRoleEdit">确定</div>
                                        <div v-if="roleEdit.index == rindex" class="ml-20 c-blue cursor lh-1"
                                            style="text-wrap:nowrap;" @click="deleteRoleEdit(rindex)">删除
                                        </div>
                                    </div>
                                    <div class="df ai-c">
                                        <a-checkbox v-if="r.name != 'founder' && r.name != 'super'"
                                            :model-value="r.is_default_register == 2"
                                            @change="v => { r.is_default_register = v ? 2 : 1; chengeRegister(r, r.is_default_register); }">默认邀请端</a-checkbox>
                                    </div>
                                </div>

                                <div class="greybox manifest-front-config">
                                    <template v-if="form.type === 'gateway-plugin'">
                                        <div class="greybox-title">插件配置页面</div>
                                        <a-alert type="info" show-icon class="zpk-primary-alert" title="MicroApp 接入说明"
                                            :closable="false">
                                            <div class="registry-alert-item">
                                                <strong>运行方式：</strong>网关插件前端作为 MicroApp 嵌入面板，用于编辑插件配置，不需要填写接口地址或自行请求 WasmPlugin。
                                            </div>
                                            <div class="registry-alert-item mt-6">
                                                <strong>配置数据：</strong>通过 <code>window.$wujie.props.pluginConfig</code> 获取当前配置，通过 <code>pluginEnabled</code> 获取当前配置的启用状态。
                                            </div>
                                            <div class="registry-alert-item mt-6">
                                                <strong>配置范围：</strong><code>configScope</code> 为 <code>global</code> 时表示全局配置；为 <code>rule</code> 时表示域名规则配置，并同时注入当前 <code>domain</code>（域名）、<code>ingressName</code>（Ingress 名称）、<code>namespace</code> 和 <code>path</code>。
                                            </div>
                                            <div class="registry-alert-item mt-6">
                                                <strong>菜单配置：</strong>全局配置在创始人端添加菜单，规则配置在普通用户端添加菜单。
                                            </div>
                                            <div class="registry-alert-item mt-6">
                                                <strong>保存方式：</strong>调用 <code>window.$wujie.props.savePluginConfig(config, enabled)</code> 保存当前配置及启用状态，具体资源和规则关联由面板处理。
                                            </div>
                                        </a-alert>
                                        <manifest-config-table :rows="r.frontend_props"
                                            table-class="manifest-param-table frontend-param-table mt-20"
                                            add-text="添加前端配置" always-show @add="addParamRow(r.frontend_props)">
                                            <template #title>
                                                <div class="df ai-c">
                                                    前端配置
                                                    <a-tooltip position="tl"
                                                        content="配置会通过 window.$wujie.props.frontend_props 和同名顶层属性传递给插件前端">
                                                        <ArcoIcon name="icon-41" :size="16" />
                                                    </a-tooltip>
                                                </div>
                                            </template>
                                            <template #columns>
                                                <manifest-config-table-column data-index="key" title="key">
                                                    <template #cell="{ record }">
                                                        <a-input v-model="record.key" placeholder="key"
                                                            @change="getMenu"
                                                            style="width:200px;margin-right:10px;"></a-input>
                                                    </template>
                                                </manifest-config-table-column>
                                                <manifest-config-table-column data-index="value" title="value">
                                                    <template #cell="{ record }">
                                                        <div class="param-value-field">
                                                            <a-input v-model="record.value" placeholder="value"
                                                                @change="changeConfigValue(record)">
                                                                <template #suffix>
                                                                    <a-popover trigger="click" position="bottom"
                                                                        :content-style="{ width: '360px' }">
                                                                        <span class="config-value-suffix">选择系统配置</span>
                                                                        <template #content>
                                                                            <div class="var-picker">
                                                                                <template v-for="group in variableGroups"
                                                                                    :key="group.title">
                                                                                    <div class="var-picker-title">{{ group.title }}</div>
                                                                                    <div v-if="group.options.length">
                                                                                        <div v-for="param in group.options"
                                                                                            :key="param.value"
                                                                                            class="var-picker-item"
                                                                                            @click="selectConfigVariable(record, param)">
                                                                                            <div class="var-picker-name">
                                                                                                {{ param.key || param.value }}
                                                                                                <span>{{ param.displayValue }}</span>
                                                                                            </div>
                                                                                        </div>
                                                                                    </div>
                                                                                    <div v-else class="var-picker-empty">暂无可选配置</div>
                                                                                </template>
                                                                            </div>
                                                                        </template>
                                                                    </a-popover>
                                                                </template>
                                                            </a-input>
                                                        </div>
                                                    </template>
                                                </manifest-config-table-column>
                                                <manifest-config-table-column title="描述">
                                                    <template #cell="{ record }">{{ getConfigVariableLabel(record) }}</template>
                                                </manifest-config-table-column>
                                                <manifest-config-table-column title="操作">
                                                    <template #cell="{ index }">
                                                        <span class="c-blue cursor handle"
                                                            @click="removeParamRow(r.frontend_props, index)">删除</span>
                                                    </template>
                                                </manifest-config-table-column>
                                            </template>
                                            <template #prepend>
                                                <tr v-for="item in frontendDefaultProps" :key="item.value"
                                                    class="frontend-default-prop-row">
                                                    <td>{{ item.key }}</td>
                                                    <td>{{ item.value }}</td>
                                                    <td>{{ item.description }}</td>
                                                    <td></td>
                                                </tr>
                                            </template>
                                        </manifest-config-table>
                                    </template>
                                    <template v-else-if="r.load_mode === 'iframe'">
                                        <div class="greybox-title">iframe配置</div>
                                        <a-alert type="info" show-icon class="zpk-primary-alert mb-20" title="提示"
                                            :closable="false">
                                            <div class="registry-alert-item">由于iframe受到了浏览器安全限制，后端地址必须支持https协议访问。</div>
                                            <div class="registry-alert-item mt-6">由于iframe受到了浏览器安全限制，生成cookies时必须设置 SameSite: None, Secure: true，并且header设置允许跨域，才能正常传递。</div>
                                            <div class="registry-alert-item mt-6">Access-Control-Allow-Origin 需要设置为具体域名，可从请求头 Origin 或 Referer 获取，不可设置为 *。</div>
                                            <div class="registry-alert-item mt-6">变量传递的请求参数只支持query方式，会将GET参数固定拼接到地址后。</div>
                                        </a-alert>
                                        <a-form-item label="地址类型" style="margin-bottom:20px;">
                                            <a-radio-group v-model="r.type" @change="changeBackendType(r)" :disabled="form.type === 'tradition'">
                                                <a-radio value="internal">应用地址</a-radio>
                                                <a-radio value="external">远程地址</a-radio>
                                            </a-radio-group>
                                        </a-form-item>
                                        <a-form-item label="页面地址" style="margin-bottom:20px;">
                                            <div class="backend-url-form-field">
                                                <div v-if="r.type == 'internal'" class="backend-url-config df ai-c">
                                                    <span class="backend-url-fixed">https://</span>
                                                    <span class="backend-url-fixed backend-url-placeholder">{{
                                                        getIframeDomainDisplayPlaceholder() }}</span>
                                                    <span class="backend-url-fixed">/</span>
                                                    <a-input v-model="r.backend_path" @input="getMenu" @change="getMenu"
                                                        placeholder="请输入目录"
                                                        class="backend-url-control backend-url-input" />
                                                </div>
                                                <div v-else
                                                    class="backend-url-config backend-url-config-external df ai-c">
                                                    <a-input v-model="r.root_url" @change="getMenu" placeholder="请输入地址"
                                                        class="backend-url-control backend-url-input" prepend='https://'/>
                                                </div>
                                                <div v-if="r.type == 'internal' && !hasBackendDomainConfig()"
                                                    class="domain-warning">
                                                    <icon-exclamation-circle-fill class="domain-warning-icon"
                                                        :size="14" />
                                                    <span>当前应用后端配置尚未启用域名设置，请前往后端包管理界面配置。</span>
                                                </div>
                                            </div>
                                        </a-form-item>
                                        <div class="df ai-c manifest-front-section-title">变量传递配置</div>
                                        <manifest-config-table title="请求参数(Query)" :rows="r.proxy_request_query"
                                            table-class="manifest-param-table config-variable-table"
                                            add-text="添加请求参数" @add="addParamRow(r.proxy_request_query)">
                                            <template #columns>
                                                <manifest-config-table-column data-index="key" title="key">
                                                    <template #cell="{ record }">
                                                        <a-input v-model="record.key" placeholder="key"
                                                            @change="getMenu"
                                                            style="width:200px;margin-right:10px;"></a-input>
                                                    </template>
                                                </manifest-config-table-column>
                                                <manifest-config-table-column data-index="value" title="value">
                                                    <template #cell="{ record }">
                                                        <div class="param-value-field">
                                                            <a-input v-model="record.value" placeholder="value"
                                                                @change="changeConfigValue(record)">
                                                                <template #suffix>
                                                                    <a-popover trigger="click" position="bottom"
                                                                        :content-style="{ width: '360px' }">
                                                                        <span
                                                                            class="config-value-suffix">选择系统配置</span>
                                                                        <template #content>
                                                                            <div class="var-picker">
                                                                                <template
                                                                                    v-for="group in variableGroups"
                                                                                    :key="group.title">
                                                                                    <div class="var-picker-title">{{
                                                                                        group.title }}</div>
                                                                                    <div
                                                                                        v-if="group.options.length">
                                                                                        <div v-for="param in group.options"
                                                                                            :key="param.value"
                                                                                            class="var-picker-item"
                                                                                            @click="selectConfigVariable(record, param)">
                                                                                            <div
                                                                                                class="var-picker-name">
                                                                                                {{ param.key ||
                                                                                                param.value }}
                                                                                                <span>{{
                                                                                                    param.displayValue
                                                                                                    }}</span></div>
                                                                                        </div>
                                                                                    </div>
                                                                                    <div v-else
                                                                                        class="var-picker-empty">
                                                                                        暂无可选配置</div>
                                                                                </template>
                                                                            </div>
                                                                        </template>
                                                                    </a-popover>
                                                                </template>
                                                            </a-input>
                                                        </div>
                                                    </template>
                                                </manifest-config-table-column>
                                                <manifest-config-table-column title="描述">
                                                    <template #cell="{ record }">{{ getConfigVariableLabel(record)
                                                        }}</template>
                                                </manifest-config-table-column>
                                                <manifest-config-table-column title="操作">
                                                    <template #cell="{ index }">
                                                        <span class="c-blue cursor handle"
                                                            @click="removeParamRow(r.proxy_request_query, index)">删除</span>
                                                    </template>
                                                </manifest-config-table-column>
                                            </template>
                                        </manifest-config-table>
                                        <manifest-config-table :rows="r.frontend_props"
                                            table-class="manifest-param-table frontend-param-table"
                                            add-text="添加前端配置" always-show @add="addParamRow(r.frontend_props)">
                                            <template #title>
                                                <div class="df ai-c">
                                                    前端配置
                                                    <a-tooltip
                                                        position="tl"
                                                        content="面板提供microapp机制渲染前端包，可通过window.$wujie.props.frontend_props 从JS变量获取传递值">
                                                        <ArcoIcon name="icon-41" :size="16" />
                                                    </a-tooltip>
                                                </div>
                                            </template>
                                            <template #columns>
                                                <manifest-config-table-column data-index="key" title="key">
                                                    <template #cell="{ record }">
                                                        <a-input v-model="record.key" placeholder="key"
                                                            @change="getMenu"
                                                            style="width:200px;margin-right:10px;"></a-input>
                                                    </template>
                                                </manifest-config-table-column>
                                                <manifest-config-table-column data-index="value" title="value">
                                                    <template #cell="{ record }">
                                                        <div class="param-value-field">
                                                            <a-input v-model="record.value" placeholder="value"
                                                                @change="changeConfigValue(record)">
                                                                <template #suffix>
                                                                    <a-popover trigger="click" position="bottom"
                                                                        :content-style="{ width: '360px' }">
                                                                        <span
                                                                            class="config-value-suffix">选择系统配置</span>
                                                                        <template #content>
                                                                            <div class="var-picker">
                                                                                <template
                                                                                    v-for="group in variableGroups"
                                                                                    :key="group.title">
                                                                                    <div class="var-picker-title">{{
                                                                                        group.title }}</div>
                                                                                    <div
                                                                                        v-if="group.options.length">
                                                                                        <div v-for="param in group.options"
                                                                                            :key="param.value"
                                                                                            class="var-picker-item"
                                                                                            @click="selectConfigVariable(record, param)">
                                                                                            <div
                                                                                                class="var-picker-name">
                                                                                                {{ param.key ||
                                                                                                param.value }}
                                                                                                <span>{{
                                                                                                    param.displayValue
                                                                                                    }}</span></div>
                                                                                        </div>
                                                                                    </div>
                                                                                    <div v-else
                                                                                        class="var-picker-empty">
                                                                                        暂无可选配置</div>
                                                                                </template>
                                                                            </div>
                                                                        </template>
                                                                    </a-popover>
                                                                </template>
                                                            </a-input>
                                                        </div>
                                                    </template>
                                                </manifest-config-table-column>
                                                <manifest-config-table-column title="描述">
                                                    <template #cell="{ record }">{{ getConfigVariableLabel(record)
                                                        }}</template>
                                                </manifest-config-table-column>
                                                <manifest-config-table-column title="操作">
                                                    <template #cell="{ index }">
                                                        <span class="c-blue cursor handle"
                                                            @click="removeParamRow(r.frontend_props, index)">删除</span>
                                                    </template>
                                                </manifest-config-table-column>
                                            </template>
                                            <template #prepend>
                                                <tr v-for="item in frontendDefaultProps" :key="item.value"
                                                    class="frontend-default-prop-row">
                                                    <td>{{ item.key }}</td>
                                                    <td>{{ item.value }}</td>
                                                    <td>{{ item.description }}</td>
                                                    <td></td>
                                                </tr>
                                            </template>
                                        </manifest-config-table>
                                    </template>
                                    <template v-else>
                                        <div class="greybox-title">代理配置<a-tooltip position="tl" content="面板提供转发服务到接口地址，接口后端可通过HTTP变量获取传递值">
                                                <ArcoIcon name="icon-41" :size="16" />
                                            </a-tooltip></div>
                                        <a-form-item label="地址类型" style="margin-bottom:20px;">
                                            <a-radio-group :disabled="form.type === 'tradition'" v-model="r.type" @change="changeBackendType(r)">
                                                <a-radio value="internal">应用地址</a-radio>
                                                <a-radio value="external">远程地址</a-radio>
                                            </a-radio-group>
                                        </a-form-item>
                                        <a-form-item label="接口地址" style="margin-bottom:20px;">
                                            <div class="backend-url-form-field" v-if="r.type == 'internal' && form.type === 'tradition'">
                                                <div class="backend-url-config df ai-c">
                                                    <span class="backend-url-fixed">https://</span>
                                                    <span class="backend-url-fixed backend-url-placeholder">{{
                                                        getIframeDomainDisplayPlaceholder() }}</span>
                                                    <span class="backend-url-fixed">/</span>
                                                    <a-input v-model="r.backend_path" @input="getMenu" @change="getMenu"
                                                        placeholder="请输入目录"
                                                        class="backend-url-control backend-url-input" />
                                                </div>
                                            </div>
                                            <div v-else-if="r.type == 'internal'" class="backend-url-config df ai-c">
                                                <span class="backend-url-fixed">http://</span>
                                                <a-select v-model="r.backend_url" allow-search
                                                    placeholder="选择应用标识"
                                                    class="backend-url-control backend-url-identifie"
                                                    @change="changeBackendUrl(r)">
                                                    <a-option v-for="app in backendAppOptions" :key="app.id"
                                                        :label="app.id" :value="app.id">
                                                        <div class="backend-app-option">
                                                            <span>{{ app.id }}</span>
                                                            <span v-if="app.title && app.title != app.id">{{ app.title
                                                                }}</span>
                                                        </div>
                                                    </a-option>
                                                </a-select>
                                                <span class="backend-url-fixed">.default.svc.cluster.local:</span>
                                                <a-auto-complete v-model="r.backend_port"
                                                    :data="getBackendPortOptions(r.backend_url, r.backend_port)"
                                                    :filter-option="false" placeholder="端口"
                                                    class="backend-url-control backend-url-port" @input="getMenu"
                                                    @change="getMenu" @select="getMenu"></a-auto-complete>
                                            </div>
                                            <div v-else class="backend-url-config backend-url-config-external df ai-c">
                                                <a-select v-model="r.root_protocol"
                                                    class="backend-url-control backend-url-protocol" @change="getMenu">
                                                    <a-option label="http://" value="http://"></a-option>
                                                    <a-option label="https://" value="https://"></a-option>
                                                </a-select>
                                                <a-input v-model="r.root_url" @change="getMenu" placeholder="请输入地址"
                                                    :disabled="form.type === 'tradition'"
                                                    class="backend-url-control backend-url-input" />
                                            </div>
                                        </a-form-item>
                                        <div class="df ai-c manifest-front-section-title">变量传递配置<a-tooltip position="tl"
                                                content="将开发者设置的变量值传递给后端接口和前端JS变量中">
                                                <ArcoIcon name="icon-41" :size="16" />
                                            </a-tooltip></div>
                                        <div class="manifest-front-block">
                                            <a-alert type="info" show-icon class="zpk-primary-alert mb-20" title="提示"
                                                :closable="false">
                                                <div class="registry-alert-item">面板提供统一的反向代理服务，会将设置的接口地址转为同域地址：{{`/panel-api/v1/microapp/${identifie}/proxy`}}</div>
                                                <div class="registry-alert-item mt-6">代理配置中的请求参数仅在反向代理服务到后端接口地址层传递，不会对外暴露，可用于处理token等敏感数据</div>
                                            </a-alert>
                                            <manifest-config-table title="请求头(Header)" :rows="r.proxy_request_header"
                                                table-class="manifest-param-table config-variable-table"
                                                add-text="添加请求头" @add="addParamRow(r.proxy_request_header)">
                                                <template #columns>
                                                    <manifest-config-table-column data-index="key" title="key">
                                                        <template #cell="{ record }">
                                                            <a-input v-model="record.key" placeholder="key"
                                                                @change="getMenu"
                                                                style="width:200px;margin-right:10px;"></a-input>
                                                        </template>
                                                    </manifest-config-table-column>
                                                    <manifest-config-table-column data-index="value" title="value">
                                                        <template #cell="{ record }">
                                                            <div class="param-value-field">
                                                                <a-input v-model="record.value" placeholder="value"
                                                                    @change="changeConfigValue(record)">
                                                                    <template #suffix>
                                                                        <a-popover trigger="click" position="bottom"
                                                                            :content-style="{ width: '360px' }">
                                                                            <span
                                                                                class="config-value-suffix">选择系统配置</span>
                                                                            <template #content>
                                                                                <div class="var-picker">
                                                                                    <template
                                                                                        v-for="group in variableGroups"
                                                                                        :key="group.title">
                                                                                        <div class="var-picker-title">{{
                                                                                            group.title }}</div>
                                                                                        <div
                                                                                            v-if="group.options.length">
                                                                                            <div v-for="param in group.options"
                                                                                                :key="param.value"
                                                                                                class="var-picker-item"
                                                                                                @click="selectConfigVariable(record, param)">
                                                                                                <div
                                                                                                    class="var-picker-name">
                                                                                                    {{ param.key ||
                                                                                                    param.value }}
                                                                                                    <span>{{
                                                                                                        param.displayValue
                                                                                                        }}</span></div>
                                                                                            </div>
                                                                                        </div>
                                                                                        <div v-else
                                                                                            class="var-picker-empty">
                                                                                            暂无可选配置</div>
                                                                                    </template>
                                                                                </div>
                                                                            </template>
                                                                        </a-popover>
                                                                    </template>
                                                                </a-input>
                                                            </div>
                                                        </template>
                                                    </manifest-config-table-column>
                                                    <manifest-config-table-column title="描述">
                                                        <template #cell="{ record }">{{ getConfigVariableLabel(record)
                                                            }}</template>
                                                    </manifest-config-table-column>
                                                    <manifest-config-table-column title="操作">
                                                        <template #cell="{ index }">
                                                            <span class="c-blue cursor handle"
                                                                @click="removeParamRow(r.proxy_request_header, index)">删除</span>
                                                        </template>
                                                    </manifest-config-table-column>
                                                </template>
                                            </manifest-config-table>
                                            <manifest-config-table
                                                title="请求参数(Query)" :rows="r.proxy_request_query"
                                                table-class="manifest-param-table config-variable-table"
                                                add-text="添加请求参数" @add="addParamRow(r.proxy_request_query)">
                                                <template #columns>
                                                    <manifest-config-table-column data-index="key" title="key">
                                                        <template #cell="{ record }">
                                                            <a-input v-model="record.key" placeholder="key"
                                                                @change="getMenu"
                                                                style="width:200px;margin-right:10px;"></a-input>
                                                        </template>
                                                    </manifest-config-table-column>
                                                    <manifest-config-table-column data-index="value" title="value">
                                                        <template #cell="{ record }">
                                                            <div class="param-value-field">
                                                                <a-input v-model="record.value" placeholder="value"
                                                                    @change="changeConfigValue(record)">
                                                                    <template #suffix>
                                                                        <a-popover trigger="click" position="bottom"
                                                                            :content-style="{ width: '360px' }">
                                                                            <span
                                                                                class="config-value-suffix">选择系统配置</span>
                                                                            <template #content>
                                                                                <div class="var-picker">
                                                                                    <template
                                                                                        v-for="group in variableGroups"
                                                                                        :key="group.title">
                                                                                        <div class="var-picker-title">{{
                                                                                            group.title }}</div>
                                                                                        <div
                                                                                            v-if="group.options.length">
                                                                                            <div v-for="param in group.options"
                                                                                                :key="param.value"
                                                                                                class="var-picker-item"
                                                                                                @click="selectConfigVariable(record, param)">
                                                                                                <div
                                                                                                    class="var-picker-name">
                                                                                                    {{ param.key ||
                                                                                                    param.value }}
                                                                                                    <span>{{
                                                                                                        param.displayValue
                                                                                                        }}</span></div>
                                                                                            </div>
                                                                                        </div>
                                                                                        <div v-else
                                                                                            class="var-picker-empty">
                                                                                            暂无可选配置</div>
                                                                                    </template>
                                                                                </div>
                                                                            </template>
                                                                        </a-popover>
                                                                    </template>
                                                                </a-input>
                                                            </div>
                                                        </template>
                                                    </manifest-config-table-column>
                                                    <manifest-config-table-column title="描述">
                                                        <template #cell="{ record }">{{ getConfigVariableLabel(record)
                                                            }}</template>
                                                    </manifest-config-table-column>
                                                    <manifest-config-table-column title="操作">
                                                        <template #cell="{ index }">
                                                            <span class="c-blue cursor handle"
                                                                @click="removeParamRow(r.proxy_request_query, index)">删除</span>
                                                        </template>
                                                    </manifest-config-table-column>
                                                </template>
                                            </manifest-config-table>
                                                <manifest-config-table :rows="r.frontend_props"
                                            table-class="manifest-param-table frontend-param-table"
                                            add-text="添加前端配置" always-show @add="addParamRow(r.frontend_props)">
                                            <template #title>
                                                <div class="df ai-c">
                                                    前端配置<a-tooltip position="tl"
                                                        content="面板提供microapp机制渲染前端包，可通过window.$wujie.props.frontend_props 从JS变量获取传递值">
                                                        <ArcoIcon name="icon-41" :size="16" />
                                                    </a-tooltip>
                                                </div>
                                            </template>
                                            <template #columns>
                                                <manifest-config-table-column data-index="key" title="key">
                                                    <template #cell="{ record }">
                                                        <a-input v-model="record.key" placeholder="key"
                                                            @change="getMenu"
                                                            style="width:200px;margin-right:10px;"></a-input>
                                                    </template>
                                                </manifest-config-table-column>
                                                <manifest-config-table-column data-index="value" title="value">
                                                    <template #cell="{ record }">
                                                        <div class="param-value-field">
                                                            <a-input v-model="record.value" placeholder="value"
                                                                @change="changeConfigValue(record)">
                                                                <template #suffix>
                                                                    <a-popover trigger="click" position="bottom"
                                                                        :content-style="{ width: '360px' }">
                                                                        <span
                                                                            class="config-value-suffix">选择系统配置</span>
                                                                        <template #content>
                                                                            <div class="var-picker">
                                                                                <template
                                                                                    v-for="group in variableGroups"
                                                                                    :key="group.title">
                                                                                    <div class="var-picker-title">{{
                                                                                        group.title }}</div>
                                                                                    <div
                                                                                        v-if="group.options.length">
                                                                                        <div v-for="param in group.options"
                                                                                            :key="param.value"
                                                                                            class="var-picker-item"
                                                                                            @click="selectConfigVariable(record, param)">
                                                                                            <div
                                                                                                class="var-picker-name">
                                                                                                {{ param.key ||
                                                                                                param.value }}
                                                                                                <span>{{
                                                                                                    param.displayValue
                                                                                                    }}</span></div>
                                                                                        </div>
                                                                                    </div>
                                                                                    <div v-else
                                                                                        class="var-picker-empty">
                                                                                        暂无可选配置</div>
                                                                                </template>
                                                                            </div>
                                                                        </template>
                                                                    </a-popover>
                                                                </template>
                                                            </a-input>
                                                        </div>
                                                    </template>
                                                </manifest-config-table-column>
                                                <manifest-config-table-column title="描述">
                                                    <template #cell="{ record }">{{ getConfigVariableLabel(record)
                                                        }}</template>
                                                </manifest-config-table-column>
                                                <manifest-config-table-column title="操作">
                                                    <template #cell="{ index }">
                                                        <span class="c-blue cursor handle"
                                                            @click="removeParamRow(r.frontend_props, index)">删除</span>
                                                    </template>
                                                </manifest-config-table-column>
                                            </template>
                                            <template #prepend>
                                                <tr v-for="item in frontendDefaultProps" :key="item.value"
                                                    class="frontend-default-prop-row">
                                                    <td>{{ item.key }}</td>
                                                    <td>{{ item.value }}</td>
                                                    <td>{{ item.description }}</td>
                                                    <td></td>
                                                </tr>
                                            </template>
                                        </manifest-config-table>
                                        </div>


                                    </template>

                                </div>

                                <div class="mt-10 greybox manifest-front-config">
                                    <div class="greybox-title">菜单配置</div>
                                    <manifest-config-table :rows="r.menu" table-class="menutable mt-10"
                                        add-text="添加一级菜单" always-show @add="addMenu(r.menu)">
                                        <template #columns>
                                            <manifest-config-table-column data-index="displayorder" title="排序">
                                                <template #cell="{ record }">
                                                    <div><a-input type="number"
                                                            :model-value="String(record.displayorder ?? '')"
                                                            @update:model-value="v => record.displayorder = v"
                                                            @change="getMenu"
                                                            style="width:60px; height:36px;"></a-input>
                                                    </div>
                                                    <div v-for="(sub, subid) in record.children" :key="subid"
                                                        class="df mt-10">
                                                        <div class="branch"
                                                            :class="{ last: subid == record.children.length - 1 }">
                                                        </div>
                                                        <a-input type="number"
                                                            :model-value="String(sub.displayorder ?? '')"
                                                            @update:model-value="v => sub.displayorder = v"
                                                            @change="getMenu"
                                                            style="width:60px; height:36px;"></a-input>
                                                    </div>
                                                </template>
                                            </manifest-config-table-column>
                                            <manifest-config-table-column data-index="do" title="路由">
                                                <template #cell="{ record }">
                                                    <div><a-input v-model="record.do" @change="onMenuChanged(r.menu)"
                                                            style="width:150px; height:36px;"
                                                            placeholder="路由"></a-input>
                                                    </div>
                                                    <div v-for="(sub, subid) in record.children" :key="subid"
                                                        class="mt-10">
                                                        <a-input v-model="sub.do" @change="onMenuChanged(r.menu)"
                                                            style="width:150px; height:36px;"
                                                            placeholder="路由"></a-input>
                                                    </div>
                                                </template>
                                            </manifest-config-table-column>
                                            <manifest-config-table-column data-index="title" title="名称">
                                                <template #cell="{ record }">
                                                    <div><a-input v-model="record.title" @change="onMenuChanged(r.menu)"
                                                            style="width:150px; height:36px;"
                                                            placeholder="名称"></a-input>
                                                    </div>
                                                    <div v-for="(sub, subid) in record.children" :key="subid"
                                                        class="mt-10">
                                                        <a-input v-model="sub.title" @change="onMenuChanged(r.menu)"
                                                            style="width:150px; height:36px;"
                                                            placeholder="名称"></a-input>
                                                    </div>
                                                </template>
                                            </manifest-config-table-column>
                                            <manifest-config-table-column data-index="default">
                                                <template #title>
                                                    <div class="df ai-c jc-c">
                                                        <span>欢迎页</span>
                                                        <a-tooltip position="tl" content="用户进入系统首页访问的页面">
                                                            <icon-question-circle-fill class="cursor ml-4 c-99"
                                                                :size="16" />
                                                        </a-tooltip>
                                                    </div>
                                                </template>
                                                <template #cell="{ record }">
                                                    <div class="menu-default-cell">
                                                        <div class="menu-default-check"
                                                            :style="{ visibility: record.children?.length > 0 ? 'hidden' : 'visible' }">
                                                            <a-checkbox :model-value="Number(record.is_default) === 1"
                                                                :name="r.name"
                                                                @click.stop="setMenuDefault(r.menu, record)"></a-checkbox>
                                                        </div>
                                                        <div v-for="(sub, subid) in record.children" :key="subid"
                                                            class="mt-10 menu-default-check">
                                                            <a-checkbox :model-value="Number(sub.is_default) === 1"
                                                                :name="r.name"
                                                                @click.stop="setMenuDefault(r.menu, sub)"></a-checkbox>
                                                        </div>
                                                    </div>
                                                </template>
                                            </manifest-config-table-column>
                                            <manifest-config-table-column data-index="icon" title="图标">
                                                <template #cell="{ record }">
                                                    <div class="selicon cursor df ai-c jc-c" v-if="record.icon_svg"
                                                        @click="dialogVisible = true; activeItem = record;"
                                                        v-html="elementsToSvg(record.icon_svg)"></div>
                                                    <div class="selicon cursor df ai-c jc-c" v-else-if="record.icon"
                                                        @click="dialogVisible = true; activeItem = record;"><i
                                                            class="fs-24 wi" :class="'wi-' + record.icon"></i>
                                                    </div>
                                                    <div class="selicon cursor df ai-c jc-c" v-else
                                                        @click="dialogVisible = true; activeItem = record;">
                                                        <svg class="default-menu-icon"
                                                            xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024"
                                                            aria-hidden="true">
                                                            <path fill="currentColor"
                                                                d="M160 448a32 32 0 0 1-32-32V160.064a32 32 0 0 1 32-32h256a32 32 0 0 1 32 32V416a32 32 0 0 1-32 32zm448 0a32 32 0 0 1-32-32V160.064a32 32 0 0 1 32-32h255.936a32 32 0 0 1 32 32V416a32 32 0 0 1-32 32zM160 896a32 32 0 0 1-32-32V608a32 32 0 0 1 32-32h256a32 32 0 0 1 32 32v256a32 32 0 0 1-32 32zm448 0a32 32 0 0 1-32-32V608a32 32 0 0 1 32-32h255.936a32 32 0 0 1 32 32v256a32 32 0 0 1-32 32z">
                                                            </path>
                                                        </svg>
                                                    </div>
                                                    <div v-for="(sub, subid) in record.children" :key="subid"
                                                        class="df ai-c jc-c mt-10" style="width:36px; height:36px;">
                                                    </div>
                                                </template>
                                            </manifest-config-table-column>
                                            <manifest-config-table-column title="操作">
                                                <template #cell="{ record, index }">
                                                    <div class="df ai-c" style="height:36px;">
                                                        <span class="handle c-blue cursor"
                                                            @click="addSub(r.menu, record)">添加子菜单</span>
                                                        <a-popover position="top" :content-style="{ width: '240px' }">
                                                            <span class="handle c-blue cursor">设置位置</span>
                                                            <template #content>
                                                                <div>
                                                                    <div class="df ai-c jc-b">
                                                                        <div class="menu-single-location">单个菜单位置设置
                                                                        </div>
                                                                    </div>
                                                                    <a-radio-group v-model="record.location"
                                                                        class="df mt-10" @change="getMenu">
                                                                        <div class="fc df df-c ai-c menulocation cursor"
                                                                            @click="record.location = 'normal'; getMenu()">
                                                                            <img v-if="r.location == 'top'"
                                                                                src="@/assets/img/menu-t.png" alt="" />
                                                                            <img v-else src="@/assets/img/menu-l.png"
                                                                                alt="" />
                                                                            <a-radio value="normal"
                                                                                class="mt-10">默认位置</a-radio>
                                                                        </div>
                                                                        <div v-if="r.location == 'top'"
                                                                            class="fc df df-c ai-c menulocation cursor"
                                                                            @click="record.location = 'back'; getMenu()">
                                                                            <img src="@/assets/img/menu-r.png" alt="" />
                                                                            <a-radio value="back"
                                                                                class="mt-10">顶部右侧</a-radio>
                                                                        </div>
                                                                        <div v-else
                                                                            class="fc df df-c ai-c menulocation cursor"
                                                                            @click="record.location = 'back'; getMenu()">
                                                                            <img src="@/assets/img/menu-b.png" alt="" />
                                                                            <a-radio value="back"
                                                                                class="mt-10">左侧底部</a-radio>
                                                                        </div>
                                                                    </a-radio-group>
                                                                </div>
                                                            </template>
                                                        </a-popover>
                                                        <span class="handle c-blue cursor"
                                                            @click="removeMenu(r.menu, index)">删除</span>
                                                    </div>
                                                    <div v-for="(sub, subid) in record.children" :key="subid"
                                                        class="mt-10 df ai-c" style="height:36px;">
                                                        <span class="handle c-blue cursor"
                                                            @click="removeSubMenu(r.menu, record, subid)">删除</span>
                                                    </div>
                                                </template>
                                            </manifest-config-table-column>
                                        </template>
                                    </manifest-config-table>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="bg-white pb-24 manifest-submit-bar">
                            <a-button :loading="submiting" type="primary" @click="submit()"
                                style="width:90px;">确定提交</a-button>
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
        <a-modal v-model:visible="dialogVisible" title="选择图标" :width="820" :footer="false">
            <sel-svg @submit="selectIcon"></sel-svg>
        </a-modal>
        <a-modal v-model:visible="showIngress" title="添加业务端" :width="640" :footer="false">
            <a-form ref="ingress" :model="newIngressEnd" label-align="left" class="manifest-dialog-form"
                :label-col-props="{ flex: '0 0 80px' }" :wrapper-col-props="{ flex: '1' }">
                <a-form-item label="名称" field="name" :rules="[{ required: true, message: '内容不能为空', trigger: 'blur' }]">
                    <a-input placeholder="请输入业务端名称" v-model="newIngressEnd.name" size="large"
                        style="width:100%;"></a-input>
                </a-form-item>
            </a-form>
            <div class="dialog-footer">
                <a-button size="large" @click="showIngress = false">取消</a-button>
                <a-button @click="addIngressEnd" type="primary" size="large">确定</a-button>
            </div>
        </a-modal>

        <a-modal v-model:visible="showAddRole" title="添加管理端" :width="640" :footer="false">
            <a-form ref="role" :model="newRole" label-align="left" class="manifest-dialog-form"
                :label-col-props="{ flex: '0 0 80px' }" :wrapper-col-props="{ flex: '1' }">
                <a-form-item label="名称" field="title" :rules="[{ required: true, message: '内容不能为空', trigger: 'blur' }]">
                    <a-input placeholder="请输入管理端名称" v-model="newRole.title" size="large" style="width:100%;"></a-input>
                </a-form-item>
                <a-form-item label="标识" field="name" :rules="[{ required: true, message: '内容不能为空', trigger: 'blur' }]">
                    <a-input placeholder="请输入管理端标识" v-model="newRole.name" size="large" style="width:100%;"></a-input>
                </a-form-item>
            </a-form>
            <div class="dialog-footer">
                <a-button size="large" @click="showAddRole = false">取消</a-button>
                <a-button @click="addRole" type="primary" size="large">确定</a-button>
            </div>
        </a-modal>
    </div>
</template>

<script>
import jsyaml from "js-yaml";
import hljs from 'highlight.js';
import filesUpload from './files-upload.vue';
import selSvg from '@/components/sel-svg.vue';
import W7Identifie from '@/components/w7-identifie.vue';
import ArcoIcon from '@/components/arco-icon.vue';
import ManifestConfigTable from '@/components/manifest-config-table.vue';
import ManifestConfigTableColumn from '@/components/manifest-config-table-column.vue';
import emitWujieEvent from '@/utils/wujie-event';
import {
    IconCheckCircleFill,
    IconEdit,
    IconExclamationCircleFill,
    IconQuestionCircleFill,
    IconUpload,
} from '@arco-design/web-vue/es/icon';
import { messageSuccess, messageWarning } from '@/utils/ui-feedback';


export default {
    emits: ['writefile'],
    props: [
        'data',
        'submiting',
        'option',
        'identifie',
        'version_id',
        'app_ports',
        'manifestInfo'
    ],
    components: {
        filesUpload,
        selSvg,
        W7Identifie,
        ArcoIcon,
        ManifestConfigTable,
        ManifestConfigTableColumn,
        IconCheckCircleFill,
        IconEdit,
        IconExclamationCircleFill,
        IconQuestionCircleFill,
        IconUpload,
    },
    data() {
        return {
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
                menu_type: 'thirdparty_cd',
                type: 'front',
                name: "",
                author: "",
                description: "",
                identifie: "",
                port: [],
                cpu: 1,
                mem: 2,
                support: [],
                role: [],
                volumes: [],
                hostPath: [],
                env: [],
                startParams: [],

                depend: [],

                mysql8: false,
                redis: false,


                hasIframe: false,
                role_founder: false,
                role_super: false,



                image: "",

                taginput: "",
                tags: [],

                ingress: [],
                privileged: false,
                build_context: '',

                cmd: [''],
                shell: [],

                securityContext: {
                    runAsNonRoot: false,
                    runAsUser: '',
                    runAsGroup: '',
                    fsGroup: '',
                },

                language: '',
                helm: {
                    repository: '',
                    chartName: 'default',
                },
                entry: 'public',



                proxy_request_header: [{ key: '', value: '' }],
                proxy_request_query: [{ key: '', value: '' }],

                frontend_props: [{ key: '', value: '' }],
            },

            cdrole: {},
            panelRoles: {
                founder: '创始人',
                super: '管理员',
                tech: '技术人员',
                normal: '普通用户',
            },

            roleEdit: {
                index: -1,
                name: '',
                title: '',
            },
            showAddRole: false,
            newRole: { title: "", name: "" },
            app_names: [],

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
                port: [{ required: true, message: '内容不能为空', trigger: 'blur' }],
                cpu: [{ required: true, message: '内容不能为空', trigger: 'blur' }],
                mem: [{ required: true, message: '内容不能为空', trigger: 'blur' }],
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

            showIngress: false,
            newIngressEnd: { name: "" },
            ingressEditIndex: -1,
            ingressEdit: '',


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

            noRole: false,
            isCreate: false,

            startParams: [],
            menuDefaultTarget: null,
        }
    },
    created() {
        this.baseurl = window?.$wujie?.props?.url || '';
        hljs.configure({ ignoreUnescapedHTML: true });
        this.initPanelRoles();
        this.init(this.data);



        this.jsonp('https://console.w7.cc/zpk?path=/respo/list&page=1&limit=99&tag=%E8%BF%90%E8%A1%8C%E7%8E%AF%E5%A2%83', 'getlanguage' + (this.json?.application?.identifie || Math.random()), (data) => {
            this.languageList = data?.data?.list || [];
        });

        this.getIcon();
    },
    watch: {
        'dependForm.identifie_before'() {
            this.dependForm.identifie = this.dependForm.identifie_before + '-' + this.dependForm.identifie_last;
        },
        'dependForm.identifie_last'() {
            this.dependForm.identifie = this.dependForm.identifie_before + '-' + this.dependForm.identifie_last;
        },

        'form.role_founder'(v) {
            this.switchRole(v, 'founder', '创始人端', 'console');
        },
        'form.role_super'(v) {
            this.switchRole(v, 'super', '超级管理端', 'console');
        },
        data() { this.init(this.data) },
        app_ports() {
            if (this.syncRoleBackendDefaults()) { this.getMenu(); }
        },
        'option.app_ports'() {
            if (this.syncRoleBackendDefaults()) { this.getMenu(); }
        },
    },
    computed: {
        panelRoleOptions() {
            return Object.entries(this.panelRoles || {}).map(([name, title]) => ({
                name,
                title: title || name,
            }));
        },
        systemVarOptions() {
            return (this.startParams || []).filter(item => item?.name).map(item => ({
                label: item.title || item.name,
                key: item.title || item.name,
                value: item.name,
                displayValue: item.name,
                insertValue: this.wrapConfigVariable(item.name),
                description: item.values_text || item.description || item.desc || item.name,
                type: 'start',
            }))
        },
        systemBuiltinVarOptions() {
            return [
                { value: 'system.url' },
                { value: 'system.group' },
                { value: 'system.userid' },
                { value: 'system.role' },
                { value: 'system.access_token' },
                { value: 'system.openid' },
                { value: 'system.nickname' },
                { value: 'system.cloud_uid' },
                { value: 'system.cloud_accesstoken' },
            ].map(item => ({
                ...item,
                key: item.value.replace(/^system\./, ''),
                label: item.value.replace(/^system\./, ''),
                displayValue: this.wrapConfigVariable(item.value),
                insertValue: this.wrapConfigVariable(item.value),
                type: 'system',
            }));
        },
        frontendDefaultProps() {
            return [
                {
                    key: 'url',
                    value: this.wrapConfigVariable('system.url'),
                    description: '面板代理请求微应用后端服务的地址，可能会把相对 backendUrl 拼成当前 origin 下的绝对地址',
                },
                {
                    key: 'group',
                    value: this.wrapConfigVariable('system.group'),
                    description: '应用标识分组；如果应用下有多个子应用，该值为主应用标识',
                },
                {
                    key: 'userid',
                    value: this.wrapConfigVariable('system.userid'),
                    description: '面板登录用户 ID',
                },
                {
                    key: 'role',
                    value: this.wrapConfigVariable('system.role'),
                    description: '面板用户角色，取值包括 founder、super、normal、technician',
                },
                {
                    key: 'access_token',
                    value: this.wrapConfigVariable('system.access_token'),
                    description: '面板登录用户自身维护的 access token，只能用于获取用户信息，不能准确定位 appid',
                },
                {
                    key: 'openid',
                    value: this.wrapConfigVariable('system.openid'),
                    description: '微擎云端用户 openid',
                },
                {
                    key: 'nickname',
                    value: this.wrapConfigVariable('system.nickname'),
                    description: '微擎云端用户昵称',
                },
                {
                    key: 'cloud_uid',
                    value: this.wrapConfigVariable('system.cloud_uid'),
                    description: '微擎云端用户 uid',
                },
                {
                    key: 'cloud_accesstoken',
                    value: this.wrapConfigVariable('system.cloud_accesstoken'),
                    description: '微擎云端用户 access token',
                }
            ];
        },
        variableGroups() {
            return [
                { title: '启动参数变量', options: this.systemVarOptions },
                { title: '系统内置变量', options: this.systemBuiltinVarOptions },
            ];
        },
        currentBackendIdentifie() {
            if (this.form.author && this.form.identifie) {
                return this.form.author + '-' + this.form.identifie;
            }
            return this.identifie || '';
        },
        backendAppOptions() {
            let apps = new Map();
            let addApp = (item) => {
                if (!item) { return }
                let id = item.id || item.identifie || item.name;
                if (!id) { return }
                let ports = this.normalizeBackendPorts(item.ports || item.port || []);
                let old = apps.get(id) || {};
                apps.set(id, {
                    id,
                    title: item.title || old.title || id,
                    domainEnabled: Boolean(item.domainEnabled || old.domainEnabled),
                    ports: ports.length ? ports : (old.ports || []),
                });
            };

            addApp({
                id: this.currentBackendIdentifie,
                title: this.form.name || this.currentBackendIdentifie,
                domainEnabled: this.hasManifestDomainConfig(this.json),
                ports: this.form.port?.map?.(i => i.port) || [],
            });
            (this.app_ports || []).forEach(addApp);
            (this.option?.app_ports || []).forEach(addApp);
            return [...apps.values()];
        },
    },
    methods: {
        onChange() { },
        changeStartParamValue(item, value) {
            item.value = value;
            this.changeConfigValue(item);
        },
        changeConfigValue(item) {
            let name = this.unwrapConfigVariable(item.value);
            item.isSelect = this.systemVarOptions.some(i => i.value == name);
            item.variableType = item.isSelect ? 'start' : '';
            this.getMenu();
        },
        wrapConfigVariable(value) {
            return value ? '${' + value + '}' : '';
        },
        unwrapConfigVariable(value) {
            let match = String(value || '').trim().match(/^\$\{([^}]+)\}$/);
            return match ? match[1] : String(value || '').trim();
        },
        selectConfigVariable(item, variable) {
            item.value = variable.insertValue || variable.displayValue;
            item.isSelect = variable.type == 'start';
            item.variableType = variable.type;
            this.getMenu();
        },
        getConfigVariableLabel(item) {
            let name = this.unwrapConfigVariable(item?.value);
            let variable = this.systemVarOptions.find(i => i.value == name)
                || this.systemBuiltinVarOptions.find(i => i.value == name);
            return variable?.label || '';
        },
        addParamRow(rows) {
            if (!Array.isArray(rows)) { return }
            rows.push({ key: '', value: '', isSelect: false, variableType: '' });
            this.getMenu();
        },
        removeParamRow(rows, index) {
            if (!Array.isArray(rows)) { return }
            rows.splice(index, 1);
            this.getMenu();
        },
        serializeConfigValue(item) {
            let value = item?.value;
            let name = this.unwrapConfigVariable(value);
            if (item?.isSelect || this.systemVarOptions.some(i => i.value == name)) {
                return this.wrapConfigVariable(name);
            }
            return value;
        },
        serializeParamEntries(rows) {
            return Object.fromEntries((rows || [])
                .filter(i => i.key && i.value)
                .map(item => [item.key, this.serializeConfigValue(item)]));
        },
        parseConfigValue(value) {
            let raw = String(value || '');
            let match = raw.match(/^"\{\{\s*\.Values\.([^}]+?)\s*\}\}"$/);
            if (!match) {
                match = raw.match(/^\{\{\s*\.Values\.([^}]+?)\s*\}\}$/);
            }
            if (match) {
                let name = match[1].trim();
                return { value: this.wrapConfigVariable(name), isSelect: true, variableType: 'start' };
            }
            let variableName = this.unwrapConfigVariable(raw);
            if (this.systemVarOptions.some(i => i.value == variableName)) {
                return { value: this.wrapConfigVariable(variableName), isSelect: true, variableType: 'start' };
            }
            let systemMatch = raw.match(/^\$\{system\.[^}]+\}$/);
            return { value: raw, isSelect: false, variableType: systemMatch ? 'system' : '' };
        },
        normalizeBackendPorts(ports) {
            if (!Array.isArray(ports)) { ports = ports ? [ports] : [] }
            return [...new Set(ports.map(i => {
                if (i && typeof i == 'object') {
                    return i.port ?? i.containerPort ?? '';
                }
                return i;
            }).filter(i => i !== '' && i !== undefined && i !== null).map(i => String(i)))];
        },
        getFallbackApplicationInfo() {
            let info = this.manifestInfo || {};
            let fullIdentifie = info.identifie || this.identifie || '';
            let author = '';
            let identifie = fullIdentifie;
            let match = String(fullIdentifie).match(/^([^-]+)-(.+)$/);
            if (match?.length) {
                author = match[1];
                identifie = match[2];
            }
            return {
                name: info.name || info.title || fullIdentifie,
                description: info.description || info.desc || '',
                author,
                identifie,
                fullIdentifie,
            };
        },
        applyApplicationFallbacks(j) {
            if (!j.application) { j.application = {}; }
            let fallback = this.getFallbackApplicationInfo();
            if (!j.application.identifie && fallback.fullIdentifie) {
                j.application.identifie = fallback.fullIdentifie;
            }
            if (!j.application.author && fallback.author) {
                j.application.author = fallback.author;
            }
            if (!j.application.name && fallback.name) {
                j.application.name = fallback.name;
            }
            if (!j.application.description && fallback.description) {
                j.application.description = fallback.description;
            }
        },
        getBackendPorts(identifie) {
            return this.backendAppOptions.find(i => i.id == identifie)?.ports || [];
        },
        hasManifestDomainConfig(json) {
            let startParams = json?.platform?.startParams || [];
            return startParams.some(i => String(i?.name).toLocaleUpperCase() === 'DOMAIN_URL');
        },
        getBackendPortOptions(identifie, query) {
            let q = String(query || '');
            return this.getBackendPorts(identifie)
                .filter(i => !q || String(i).includes(q))
                .map(i => String(i));
        },
        queryBackendPortSuggestions(identifie, query, cb) {
            let q = String(query || '');
            let ports = this.getBackendPorts(identifie)
                .filter(i => !q || String(i).includes(q))
                .map(i => ({ value: String(i) }));
            cb(ports);
        },
        getDefaultBackendIdentifie() {
            return this.currentBackendIdentifie || this.backendAppOptions[0]?.id || '';
        },
        getDefaultBackendPort(identifie) {
            return this.normalizeBackendPortValue(this.getBackendPorts(identifie)[0]);
        },
        normalizeBackendPortValue(port) {
            let value = port === undefined || port === null ? '' : String(port).trim();
            return value === '0' ? '' : value;
        },
        usesDomainBackendAddress(role) {
            return role?.type == 'internal' && (role.load_mode == 'iframe' || this.form.type == 'tradition');
        },
        changeBackendUrl(role) {
            role.backend_port = this.getDefaultBackendPort(role.backend_url);
            this.getMenu();
        },
        changeBackendType(role) {
            if (role.load_mode == 'iframe' || this.usesDomainBackendAddress(role)) {
                this.syncIframeBackendDefaults(role);
                this.getMenu();
                return;
            }
            if (role.type == 'internal') {
                if (!role.backend_url) {
                    role.backend_url = this.getDefaultBackendIdentifie();
                }
                if (!role.backend_port) {
                    role.backend_port = this.getDefaultBackendPort(role.backend_url);
                }
            } else {
                role.root_protocol = role.root_protocol || 'http://';
            }
            this.getMenu();
        },
        changeLoadMode(role) {
            if (role.load_mode == 'iframe') {
                if (!role.type || (role.type == 'external' && !this.normalizeHttpsExternalUrl(role.root_url))) {
                    role.type = 'internal';
                }
                this.syncIframeBackendDefaults(role);
            } else {
                if (this.form.type != 'tradition' && role.type == 'internal' && role.backend_url == this.getIframeDomainPlaceholder()) {
                    role.backend_url = this.getDefaultBackendIdentifie();
                    role.backend_port = this.getDefaultBackendPort(role.backend_url);
                }
            }
            this.getMenu();
        },
        hasBackendDomainConfig() {
            let backend = this.backendAppOptions.find(item => item.id == this.currentBackendIdentifie);
            if (backend) { return Boolean(backend.domainEnabled) }
            return this.hasManifestDomainConfig(this.json) || this.hasManifestDomainConfig(this.manifestInfo);
        },
        syncRoleBackendDefaults() {
            let changed = false;
            this.form.role.forEach(role => {
                if (this.usesDomainBackendAddress(role)) {
                    changed = this.syncIframeBackendDefaults(role) || changed;
                    return;
                }
                if (role.type != 'internal') { return }
                if (!role.backend_url) {
                    role.backend_url = this.getDefaultBackendIdentifie();
                    changed = true;
                }
                if (!role.backend_port) {
                    role.backend_port = this.getDefaultBackendPort(role.backend_url);
                    changed = true;
                }
            });
            return changed;
        },
        getIframeDomainPlaceholder() {
            return '${DOMAIN_URL}';
        },
        getIframeDomainDisplayPlaceholder() {
            return '${DOMAIN_URL}';
        },
        syncIframeBackendDefaults(role) {
            let changed = false;
            let placeholder = this.getIframeDomainPlaceholder();
            if (role.type == 'internal') {
                if (role.backend_url != placeholder) {
                    role.backend_url = placeholder;
                    changed = true;
                }
                if (role.backend_path === undefined || role.backend_path === null) {
                    role.backend_path = '';
                    changed = true;
                }
            } else {
                if (role.root_protocol != 'https://') {
                    role.root_protocol = 'https://';
                    changed = true;
                }
                let rootUrl = this.normalizeHttpsExternalUrl(role.root_url);
                if (role.root_url !== rootUrl) {
                    role.root_url = rootUrl;
                    changed = true;
                }
            }
            return changed;
        },
        parseIframeBackendUrl(url) {
            let value = String(url || '').trim();
            let placeholder = this.getIframeDomainPlaceholder();
            if (!value) {
                return {
                    type: 'internal',
                    backend_url: placeholder,
                    backend_path: '',
                    root_protocol: 'https://',
                    root_url: '',
                };
            }
            if (value.includes(placeholder)) {
                let path = value.slice(value.indexOf(placeholder) + placeholder.length).replace(/^\/+/, '');
                return {
                    type: 'internal',
                    backend_url: placeholder,
                    backend_path: path,
                    root_protocol: 'https://',
                };
            }
            let externalBackend = this.parseExternalBackendUrl(value);
            return {
                type: 'external',
                backend_url: placeholder,
                backend_path: '',
                root_protocol: externalBackend.protocol,
                root_url: externalBackend.url,
            };
        },
        getIframeBackendUrl(role) {
            if (role.type == 'internal') {
                let path = String(role.backend_path || '').trim().replace(/^\/+/, '');
                return `https://${this.getIframeDomainPlaceholder()}${path ? '/' + path : ''}`;
            }
            let url = this.normalizeHttpsExternalUrl(role.root_url);
            return url ? `https://${url}` : '';
        },
        normalizeHttpsExternalUrl(url) {
            return String(url || '').trim().replace(/^[a-z][a-z\d+.-]*:\/\//i, '');
        },
        parseExternalBackendUrl(url) {
            let match = (url || '').match(/^([a-z][a-z\d+.-]*:\/\/)(.*)$/i);
            return {
                protocol: match?.[1] || 'http://',
                url: match ? match[2] : (url || ''),
            };
        },
        getExternalBackendUrl(role) {
            if (!role.root_url) { return '' }
            if (/^[a-z][a-z\d+.-]*:\/\//i.test(role.root_url)) {
                return role.root_url;
            }
            return (role.root_protocol || 'http://') + role.root_url;
        },
        formatBackendPort(port) {
            port = this.normalizeBackendPortValue(port);
            if (port === '') { return '' }
            return /^\d+$/.test(String(port)) ? Number(port) : port;
        },
        deleteRoleEdit() {
            let r = this.form.role.filter(i => i.support == this.form.menu_type)[this.roleEdit.index]
            let findIndex = this.form.role.findIndex(i => i.support == r.support && i.name == r.name);
            if (findIndex > -1) {
                this.form.role.splice(findIndex, 1);
            }
            this.roleEdit.index = -1;
            this.getMenu();
        },
        submitRoleEdit() {
            if (this.roleEdit.title && this.roleEdit.name) {
                let r = this.form.role.filter(i => i.support == this.form.menu_type)[this.roleEdit.index];
                r.title = this.roleEdit.title;
                r.name = this.roleEdit.name;
                this.getMenu();
            }
            this.roleEdit.index = -1;
        },
        isCompleteMenuRoute(item) {
            return !!(item?.title && item?.do);
        },
        getMenuDefaultCandidates(menu, { completeOnly = false } = {}) {
            let candidates = [];
            (menu || []).forEach(item => {
                let children = Array.isArray(item.children) ? item.children : [];
                if (children.length) {
                    children.forEach(sub => {
                        if (!completeOnly || this.isCompleteMenuRoute(sub)) {
                            candidates.push(sub);
                        }
                    });
                    return;
                }
                if (!completeOnly || this.isCompleteMenuRoute(item)) {
                    candidates.push(item);
                }
            });
            return candidates;
        },
        hasIncompleteDefaultMenu(menu) {
            return (menu || []).some(item => {
                let children = Array.isArray(item.children) ? item.children : [];
                if (children.length) {
                    return children.some(sub => Number(sub.is_default) === 1 && !this.isCompleteMenuRoute(sub));
                }
                return Number(item.is_default) === 1 && !this.isCompleteMenuRoute(item);
            });
        },
        getCurrentMenuDefault(menu) {
            let selected = null;
            (menu || []).some(item => {
                if (Number(item.is_default) === 1) {
                    selected = item;
                    return true;
                }
                let children = Array.isArray(item.children) ? item.children : [];
                selected = children.find(sub => Number(sub.is_default) === 1) || null;
                return !!selected;
            });
            return selected;
        },
        normalizeBuiltMenuDefault(menu, skipFallback = false) {
            let selected = menu.find(item => Number(item.is_default) === 1);
            menu.forEach(item => {
                item.is_default = item === selected ? 1 : 0;
            });
            if (!selected && menu.length && !skipFallback) {
                menu[0].is_default = 1;
            }
        },
        normalizeMenuDefault(menu, preferredItem) {
            let candidates = this.getMenuDefaultCandidates(menu);
            (menu || []).forEach(item => {
                let children = Array.isArray(item.children) ? item.children : [];
                if (!candidates.includes(item)) {
                    item.is_default = 0;
                }
                children.forEach(sub => {
                    if (!candidates.includes(sub)) {
                        sub.is_default = 0;
                    }
                });
            });

            let selected = preferredItem && candidates.includes(preferredItem) ? preferredItem : null;
            selected = selected || candidates.find(item => Number(item.is_default) === 1);

            candidates.forEach(item => {
                item.is_default = item === selected ? 1 : 0;
            });
            if (!selected && candidates.length) {
                candidates[0].is_default = 1;
            }
        },
        getMenuPreferredDefault(menu) {
            let candidates = this.getMenuDefaultCandidates(menu);
            return candidates.includes(this.menuDefaultTarget) ? this.menuDefaultTarget : null;
        },
        onMenuChanged(menu) {
            this.normalizeMenuDefault(menu);
            this.getMenu();
        },
        addMenu(menu) {
            menu.push({ title: '', icon: '', displayorder: 0, is_default: 0, location: 'normal' });
            this.normalizeMenuDefault(menu);
            this.getMenu();
        },
        removeMenu(menu, index) {
            menu.splice(index, 1);
            this.normalizeMenuDefault(menu);
            this.getMenu();
        },
        removeSubMenu(menu, item, subid) {
            item.children.splice(subid, 1);
            this.normalizeMenuDefault(menu);
            this.getMenu();
        },
        setMenuDefault(menu, item) {
            this.menuDefaultTarget = item;
            item.is_default = 1;
            this.normalizeMenuDefault(menu, item);
            this.getMenu();
        },
        getIcon() {
            const xhr = new XMLHttpRequest();
            xhr.open("GET", "https://cdn.w7.cc/ued/font/w7/iconfont.css");
            xhr.send();
            xhr.onreadystatechange = () => {
                if (xhr.readyState === 4 && xhr.status === 200) {
                    let css = xhr.response;
                    this.icons = css.match(/(?<=\.)wi-[^:]+/g);
                }
            }
        },
        addSub(menu, item) {
            item.children = item.children || [];
            let selected = this.getCurrentMenuDefault(menu);
            let sub = { title: '', displayorder: 0, is_default: 0 };
            item.children.push(sub);
            if (!selected) {
                this.normalizeMenuDefault(menu);
            }
            this.getMenu();
        },
        selectIcon(item) {
            this.activeItem.icon_svg = item.json;
            this.dialogVisible = false;
            this.getMenu();
        },
        getDefaultPanelRoles() {
            return {
                founder: '创始人',
                super: '管理员',
                tech: '技术人员',
                normal: '普通用户',
            };
        },
        normalizePanelRoles(roles) {
            let source = roles && typeof roles == 'object' ? roles : this.getDefaultPanelRoles();
            let normalized = {};
            Object.entries(source).forEach(([name, title]) => {
                if (!name) { return }
                normalized[name] = title || name;
            });
            if (!Object.keys(normalized).length) {
                normalized = this.getDefaultPanelRoles();
            }
            this.form.role.filter(item => item.support == 'thirdparty_cd').forEach(item => {
                if (!normalized[item.name]) {
                    normalized[item.name] = item.title || item.name;
                }
            });
            return normalized;
        },
        setPanelRoles(roles) {
            this.panelRoles = this.normalizePanelRoles(roles);
            let nextRoleState = {};
            Object.keys(this.panelRoles).forEach(name => {
                nextRoleState[name] = Boolean(this.form.role.find(item => item.support == 'thirdparty_cd' && item.name == name));
            });
            this.cdrole = nextRoleState;
        },
        initPanelRoles() {
            emitWujieEvent("getRole", (roles) => {
                const result = {}
                for (let role of roles) {
                    result[role.name] = role.title;
                }
                this.setPanelRoles(result);
            })
        },
        togglePanelRole(checked, role) {
            this.cdrole = {
                ...this.cdrole,
                [role.name]: checked,
            };
            this.switchRole(checked, role.name, role.title, 'thirdparty_cd');
        },
        addRole() {
            this.$refs.role.validate((errors) => {
                if (errors) { return }
                let backend_url = this.getDefaultBackendIdentifie();
                this.form.role.push({
                    title: this.newRole.title,
                    name: this.newRole.name,
                    support: this.form.menu_type,
                    status: 1,
                    load_mode: 'static_hosting',
                    is_default_register: 1,
                    location: 'left',
                    menu: [],

                    type: this.form.type === 'tradition' ? 'external' : 'internal',
                    backend_url: backend_url,
                    backend_port: this.getDefaultBackendPort(backend_url),
                    backend_path: '',
                    root_protocol: 'http://',
                    root_url: this.form.type === 'tradition' ? this.getIframeDomainPlaceholder() : '',

                    proxy_request_header: [],
                    proxy_request_query: [],

                    frontend_props: [],
                })
                this.getMenu();
                this.showAddRole = false;
                this.newRole = { title: '', name: '' };
            });
        },
        getMenu() {
            let role = [];
            let consoleRole = {
                founder: false,
                super: false,
            }
            let cdRole = {};
            this.panelRoleOptions.forEach(role => {
                cdRole[role.name] = false;
            });
            this.form.role.filter(r => r.support == 'thirdparty_cd').forEach(r => {
                if (this.form.type === 'gateway-plugin') {
                    r.load_mode = 'static_hosting';
                }

                if (r.type == 'external') {
                    try {
                        r.frontend_props?.forEach(i => i.isSelect = false)
                    } catch {
                        // Keep legacy malformed frontend_props from blocking YAML generation.
                    }
                }
                this.normalizeMenuDefault(r.menu, this.getMenuPreferredDefault(r.menu));

                let itemObj = {
                    name: r.name,
                    title: r.title,
                    status: r.status,
                    support: r.support,
                    is_default_register: r.is_default_register,
                    location: r.location,
                    menu_type: r.menu_type,
                    load_mode: r.load_mode,
                };

                if (r.support == 'console' && ['super', 'founder'].includes(r.name)) {
                    consoleRole[r.name] = true;
                }
                if (r.support == 'thirdparty_cd') {
                    cdRole[r.name] = true;
                }

                let proxy_request_header = this.serializeParamEntries(r.proxy_request_header);
                let proxy_request_query = this.serializeParamEntries(r.proxy_request_query);
                let frontend_props = this.serializeParamEntries(r.frontend_props);

                if (this.form.type === 'gateway-plugin') {
                    itemObj.backend_config = {
                        type: 'external',
                        frontend_props: frontend_props,
                    };
                } else if (r.load_mode == 'iframe') {
                    this.syncIframeBackendDefaults(r);
                    itemObj.backend_config = {
                        type: r.type,
                        backend_url: this.getIframeBackendUrl(r),
                        proxy_request: {
                            query: proxy_request_query,
                        },
                        frontend_props: frontend_props
                    };
                } else {
                    let usesDomainBackendAddress = this.usesDomainBackendAddress(r);
                    if (usesDomainBackendAddress) {
                        this.syncIframeBackendDefaults(r);
                    }
                    itemObj.backend_config = {
                        type: r.type,
                        ...(usesDomainBackendAddress ? {
                            backend_url: this.getIframeBackendUrl(r),
                            proxy_request: {
                                headers: proxy_request_header,
                                query: proxy_request_query,
                            },
                        } : r.type == 'internal' ? {
                            backend_url: r.backend_url,
                            backend_port: this.formatBackendPort(r.backend_port),
                            proxy_request: {
                                headers: proxy_request_header,
                                query: proxy_request_query,
                            },
                        } : {

                            backend_url: this.getExternalBackendUrl(r),
                        }),
                        frontend_props: frontend_props
                    };
                }

                let menu = [];
                for (let i in r.menu) {
                    let o = r.menu[i];
                    if (!o.title || !o.do) { continue; }
                    let item = {
                        displayorder: Number(o.displayorder),
                        do: o.do,
                        title: o.title,
                        icon: o.icon,
                        icon_svg: o.icon_svg,
                        location: o.location,
                        is_default: o.is_default || 0,
                    };
                    delete item.children;
                    menu.push(item);
                    if (o.children) {
                        for (let j in o.children) {
                            let c = o.children[j];
                            if (!c.title || !c.do) { continue; }
                            menu.push({
                                displayorder: Number(c.displayorder),
                                do: c.do,
                                title: c.title,
                                icon: c.icon,
                                icon_svg: o.icon_svg,
                                is_default: c.is_default || 0,
                                parent: o.do,
                            });
                        }
                    }
                }
                itemObj.menu = menu;
                this.normalizeBuiltMenuDefault(itemObj.menu, this.hasIncompleteDefaultMenu(r.menu));

                if (itemObj.menu.length > 0 || r.load_mode == 'iframe' || this.form.type == 'environment') {
                    role.push(itemObj);
                }
            });

            this.form.role_founder = consoleRole.founder;
            this.form.role_super = consoleRole.super;
            this.cdrole = {
                ...this.cdrole,
                ...cdRole,
            }

            this.json.bindings = role;
            this.form.hasIframe = Boolean(this.form.role.find(i => i.load_mode == 'iframe'));

            this.setYaml();
        },

        transformMenu(data) {
            const map = new Map();
            data.forEach(item => map.set(item.do, item));
            return data.filter(item => {
                const node = map.get(item.do);
                if (item.parent && map.has(item.parent)) {
                    const parent = map.get(item.parent);
                    parent.children = [...(parent.children || []), { ...node }];
                    return false;
                }
                return true;
            });
        },
        switchRole(v, name, title, type) {
            this.roleEdit.index = -1;
            let hasrole = false;
            let roleindex = 0;
            for (let i = 0; i < this.form.role.length; i++) {
                if (this.form.role[i].name == name && this.form.role[i].support == type) {
                    roleindex = i;
                    hasrole = true;
                    break;
                }
            }
            if (v) {
                if (hasrole) { return }
                let backend_url = this.getDefaultBackendIdentifie();
                this.form.role.push({
                    title: title,
                    name: name,
                    status: 1,
                    support: type,
                    load_mode: 'static_hosting',
                    is_default_register: 1,
                    location: 'left',
                    menu: [],
                    type: this.form.type === 'tradition' ? 'external' : 'internal',
                    backend_url: backend_url,
                    backend_port: this.getDefaultBackendPort(backend_url),
                    backend_path: '',
                    root_protocol: 'http://',
                    root_url: this.form.type === 'tradition' ? this.getIframeDomainPlaceholder() : '',
                    proxy_request_header: [],
                    proxy_request_query: [],
                    frontend_props: [],
                });
            } else if (hasrole) {
                this.form.role.splice(roleindex, 1);
            }
            this.getMenu();
        },


        getCreateImg(v) { this.form.image = v; },

        webUploadSuccess(data) {
            if (data?.url || data?.data?.url) {
                let url = data?.url || data?.data?.url;
                this.web.name = url.match(/[^/]+$/)[0];
                if (!this.json.web) { this.json.web = {}; }
                this.json.web.type = 'zip';
                this.json.web.url = url;
                this.web.url = url;
                this.setYaml();
            }
        },
        init(data) {
            if (!data) { return }
            this.json = jsyaml.load(data);
            this.vtitle = this.json?.application?.name;
            this.noRole = !this.json?.bindings?.length && !this.json?.web?.url;
            this.noPlatform = !this.json.platform;

            this.initJSON();
            this.getMenu();
            this.changeForm();
        },

        initJSON() {
            let j = this.json;
            j.v = 2;
            this.applyApplicationFallbacks(j);
            if (j.application) {
                if (/^[^-]+-.+$/.test(j.application?.identifie)) {
                    let i = j.application.identifie;
                    j.application.author = i.match(/^([^-]+)-(.+)$/)[1];
                } else if (j.application.identifie && j.application.author) {
                    j.application.identifie = j.application.author + '-' + j.application.identifie;
                }
                this.form.type = j.application.type || 'front';

                this.form.name = j?.application?.name;
                if (/^[^-]+-.+$/.test(j.application.identifie)) {
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
            }
            for (let i in j.bindings) {
                let o = j.bindings[i];
                o.load_mode = o.load_mode || 'static_hosting';
            }

            this.startParams = j?.platform?.startParams || [];

            this.form.name = j?.application?.name;
            if (/^[^-]+-.+$/.test(j.application.identifie)) {
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
            this.form.role = j?.bindings?.length ? j.bindings : (this.form.role || []);
            this.form.role.forEach((item) => {
                if (!item.support) {
                    item.support = 'thirdparty_cd';
                }

                if (item.load_mode == 'iframe') {
                    let iframeBackend = this.parseIframeBackendUrl(item?.backend_config?.backend_url || '');
                    item.type = iframeBackend.type;
                    if (this.form.type === 'tradition') {
                        item.type = 'external';
                    }
                    item.backend_url = iframeBackend.backend_url;
                    item.backend_path = iframeBackend.backend_path;
                    item.root_protocol = iframeBackend.root_protocol;
                    item.root_url = iframeBackend.root_url;
                    item.backend_port = '';
                } else {
                    item.type = item?.backend_config?.type || 'internal';
                    if (this.form.type === 'tradition') {
                        item.type = 'external';
                    }
                    item.backend_path = '';
                    if (item.type != 'internal') {
                        let externalBackend = this.parseExternalBackendUrl(item?.backend_config?.backend_url || '');
                        item.root_protocol = externalBackend.protocol;
                        item.root_url = externalBackend.url;
                        item.backend_url = this.getDefaultBackendIdentifie();
                        item.backend_port = this.getDefaultBackendPort(item.backend_url);
                    } else {
                        item.root_protocol = 'http://';
                        item.root_url = '';
                        item.backend_url = item?.backend_config?.backend_url;
                        item.backend_url = item.backend_url || this.getDefaultBackendIdentifie();
                        item.backend_port = this.normalizeBackendPortValue(item?.backend_config?.backend_port);
                        if (this.form.type == 'tradition') {
                            let iframeBackend = this.parseIframeBackendUrl(item?.backend_config?.backend_url || '');
                            item.backend_url = iframeBackend.backend_url;
                            item.backend_path = iframeBackend.type == 'internal' ? iframeBackend.backend_path : '';
                            item.backend_port = '';
                        }
                    }
                }


                item.proxy_request_header = Object.entries(item?.backend_config?.proxy_request?.headers || {}).map(([k, v]) => {
                    return { key: k, ...this.parseConfigValue(v) }
                });
                item.proxy_request_query = Object.entries(item?.backend_config?.proxy_request?.query || {}).map(([k, v]) => {
                    return { key: k, ...this.parseConfigValue(v) }
                });
                item.proxy_request_header = item.proxy_request_header.length ? item.proxy_request_header : []
                item.proxy_request_query = item.proxy_request_query.length ? item.proxy_request_query : []

                item.frontend_props = Object.entries(item?.backend_config?.frontend_props || {}).map(([k, v]) => {
                    return { key: k, ...this.parseConfigValue(v) }
                });
                item.frontend_props = item.frontend_props.length ? item.frontend_props : [];

                item.menu = this.transformMenu(item.menu);
            })

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
                this.form.ingress = j.platform.ingress || [];
                this.form.depend = j.platform.depends || [];
                this.form.helm = j.platform.helm || {
                    repository: '',
                    chartName: 'default',
                };
            }
            this.setPanelRoles(this.panelRoles);
            this.syncRoleBackendDefaults();
        },
        submit(otherData, callback) {
            this.$nextTick(() => {
                this.getMenu();
                this.$refs.formref.validate((errors) => {
                    if (errors) { messageWarning('必填项不能为空'); return }
                    this.$emit('complete', this.json, this.yaml, otherData, callback);
                });
            })
        },
        changeForm() {
            let j = this.json;
            if (j.application) {
                j.application.name = this.form.name;
                j.application.description = this.form.description;
                j.application.type = this.form.type;
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
        download() {
            let file = new File([this.yaml], 'manifest.yaml', { type: 'text/plain' });
            this.downloadUrl = URL.createObjectURL(file);
        },
        openYamlPreview() {
            this.showYaml = true;
            this.$nextTick(() => {
                this.setYaml();
            });
        },
        onekeyCopy(text) {
            let supportClipboard = Boolean(navigator.clipboard && window.isSecureContext);
            if (supportClipboard) {
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
        addIngressEnd() {
            this.$refs.ingress.validate((errors) => {
                if (errors) { return }
                this.form.ingress.push({
                    name: this.newIngressEnd.name,
                    routes: [
                        {
                            path: '/',
                            backend: { port: '', name: this.option?.mainapp ? '' : 'current' },
                        }
                    ],
                })
                this.getIngress();
                this.showIngress = false;
                this.newIngressEnd = { name: '' };
            });
        },
        getIngress() {
            let arr = this.form.ingress.map(i => {
                return {
                    name: i.name,
                    routes: i.routes.filter(r => {
                        if (r.backend.port) { r.backend.port = Number(r.backend.port) }
                        return r.path && r.backend?.port
                    }),
                }
            })
            if (this.json?.platform) {
                this.json.platform.ingress = arr;
            }
            this.setYaml();
        },

        jsonp(url, name, callback) {
            var win = window?.rawWindow || window;
            win[name] = (data) => {
                callback(data);
                win[name] = null;
            };
            let u = new URL(url);
            u.searchParams.append('callback', name);
            let script = document.createElement("script");
            script.type = "text/javascript";
            script.setAttribute('ignore', 'true')
            script.async = true;
            script.src = u.href;
            script.onload = function () { document.body.removeChild(this); };
            script.onerror = function () { document.body.removeChild(this); };
            document.body.append(script);
        },



        elementsToSvg(elementsArray, options = {}) {
            try {

                if (!Array.isArray(elementsArray)) {
                    throw new Error('输入必须是数组');
                }


                const svgElementIndex = elementsArray.findIndex(item => item?.type === 'svg');
                let svgRoot = null;


                if (svgElementIndex !== -1) {
                    svgRoot = { ...elementsArray[svgElementIndex] };

                    elementsArray = [...elementsArray.slice(0, svgElementIndex), ...elementsArray.slice(svgElementIndex + 1)];
                } else {

                    svgRoot = {
                        type: 'svg',
                        xmlns: 'http://www.w3.org/2000/svg',
                        viewBox: '0 0 48 48'
                    };
                }


                const defaultSize = 24;
                svgRoot.width = options.width ?? svgRoot.width ?? defaultSize;
                svgRoot.height = options.height ?? svgRoot.height ?? defaultSize;


                const svgAttrs = [];
                for (const [key, value] of Object.entries(svgRoot)) {

                    if (key === 'type') continue;

                    svgAttrs.push(`${key}="${String(value)}"`);
                }
                const svgStartTag = `<svg ${svgAttrs.join(' ')}>`;
                const svgEndTag = `</svg>`;


                const childElementsStr = [];
                for (const element of elementsArray) {

                    if (!element || typeof element !== 'object' || !element.type) {

                        continue;
                    }

                    const { type, content, ...attrs } = element;


                    const elementAttrs = [];
                    for (const [key, value] of Object.entries(attrs)) {
                        elementAttrs.push(`${key}="${String(value)}"`);
                    }


                    if (content) {

                        childElementsStr.push(`  <${type} ${elementAttrs.join(' ')}>${content}</${type}>`);
                    } else {

                        childElementsStr.push(`  <${type} ${elementAttrs.join(' ')} />`);
                    }
                }


                const svgContent = [
                    svgStartTag,
                    ...childElementsStr,
                    svgEndTag
                ].join('\n');

                return svgContent;

            } catch (error) {


                const defaultWidth = options.width ?? 24;
                const defaultHeight = options.height ?? 24;
                return `<svg width="${defaultWidth}" height="${defaultHeight}" xmlns="http://www.w3.org/2000/svg"></svg>`;
            }
        },
    },
}
</script>
<style scoped>
.com-line {
    padding-bottom: 20px;
    border-bottom: 1px solid #E7E7E7;
}

.manifest-front-panel {
    padding-bottom: 20px;
}

.manifest-front-panel-body {
    width: 100%;
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

.upfilebox .upload-cloud-icon {
    display: block;
    width: 42px;
    height: 42px;
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
    width: 36px;
    height: 36px;
}

.default-menu-icon {
    width: 24px;
    height: 24px;
    color: #1d2129;
}

.menu-default-cell {
    text-align: center;
}

.menu-default-check {
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
}

.menu-default-check :deep(.arco-checkbox) {
    align-items: center;
    line-height: 1;
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

.manifest-front-config {
    padding: 22px 16px 24px;
    background: var(--color-neutral-1);
}

.manifest-front-config .greybox-title {
    margin-bottom: 22px;
}

.manifest-front-config :deep(.arco-form-item) {
    margin-bottom: 22px;
}

.manifest-front-config .mb-20 {
    margin-bottom: 26px;
}

.manifest-front-section-title {
    margin-top: 30px;
    margin-bottom: 14px;
    line-height: 22px;
    font-weight: 600;
}

.manifest-front-section-title-first {
    margin-top: 0;
}

.manifest-front-upload-section+.greybox-title {
    margin-top: 4px;
}

.manifest-front-block {
    padding-bottom: 4px;
}

.frontend-default-prop-row td:nth-child(3) {
    color: #4e5969;
    line-height: 20px;
}

.proxy-address-row {
    margin-bottom: 22px;
    line-height: 22px;
}

.iframe-tip-alert {
    margin-bottom: 22px;
}

.iframe-tip-list {
    margin: 0;
    padding-left: 18px;
    line-height: 22px;
}

.backend-url-form-field {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
}

.domain-warning {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-top: 8px;
    line-height: 20px;
    color: #f53f3f;
    font-size: 12px;
}

.domain-warning-icon {
    flex: 0 0 auto;
    color: currentColor;
}

.param-value-field {
    width: 100%;
}

.config-value-suffix {
    display: inline-flex;
    align-items: center;
    height: 20px;
    padding-left: 10px;
    border-left: 1px solid #e5e6eb;
    color: #165dff;
    line-height: 20px;
    white-space: nowrap;
    cursor: pointer;
}

.backend-app-option {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    width: 100%;
}

.backend-app-option span:last-child {
    color: #86909c;
    font-size: 12px;
}

.var-picker {
    max-height: 360px;
    overflow: auto;
}

.var-picker-title {
    margin: 10px 0 6px;
    color: #1d2129;
    font-weight: 600;
    line-height: 22px;
}

.var-picker-title:first-child {
    margin-top: 0;
}

.var-picker-item {
    padding: 8px;
    border-radius: 4px;
    cursor: pointer;
}

.var-picker-item:hover {
    background: #f2f3f5;
}

.var-picker-name {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    color: #1d2129;
    line-height: 20px;
}

.var-picker-name span {
    flex-shrink: 0;
    color: #86909c;
}

.var-picker-empty {
    margin-top: 2px;
    color: #86909c;
    font-size: 12px;
    line-height: 18px;
}

.manifest-submit-item {
    margin-bottom: 0;
}

.backend-url-config {
    width: 620px;
    height: 32px;
    box-sizing: border-box;
    overflow: hidden;
    border: 1px solid #dcdfe6;
    border-radius: 4px;
    background: #fff;
    transition: border-color .2s;
}

.backend-url-config:focus-within {
    border-color: #409eff;
}

.backend-url-config-external {
    width: 500px;
}

.backend-url-fixed {
    color: #c0c4cc;
    line-height: 30px;
    white-space: nowrap;
    align-self: stretch;
    padding: 0 8px;
    margin: 0;
    background: #f5f7fa;
    border-right: 1px solid #dcdfe6;
}

.backend-url-placeholder {
    min-width: 130px;
    color: #606266;
    background: #fff;
}

.backend-url-fixed+.backend-url-control,
.backend-url-fixed+.backend-url-fixed,
.backend-url-control+.backend-url-fixed,
.backend-url-control+.backend-url-control {
    border-left: 1px solid #dcdfe6;
}

.backend-url-identifie {
    flex: 0 0 190px !important;
    width: 190px !important;
}

.backend-url-port {
    flex: 0 0 120px !important;
    width: 120px !important;
}

.start-param-select {
    width: 100%;
    min-width: 0;
    max-width: 100%;
}

.backend-url-protocol {
    flex: 0 0 110px !important;
    width: 110px !important;
}

:deep(.backend-url-protocol) {
    flex: 0 0 110px !important;
    width: 110px !important;
}

.backend-url-input {
    flex: 1 1 0 !important;
    width: auto !important;
    min-width: 0;
}

.backend-url-config :deep(.arco-input-wrapper),
.backend-url-config :deep(.arco-select-view) {
    height: 30px;
    box-shadow: none !important;
    border: 0;
    border-radius: 0;
    background: transparent;
    padding: 0 10px;
}

.backend-url-config :deep(.arco-input) {
    height: 30px;
    line-height: 30px;
}

.backend-url-config :deep(.arco-select-view-single) {
    padding: 0 8px;
}

.elseoption {
    padding: 6px 12px;
    cursor: pointer;
}

.addrole {
    border: 1px dashed #2d5fff;
    background: rgb(240, 243, 250);
    padding: 10px;
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

.manifest-param-table .start-param-select {
    width: 100% !important;
    min-width: 0;
    max-width: 100%;
}

.manifest-param-table .arco-input-wrapper {
    width: 100% !important;
    max-width: 100%;
}

.manifest-dialog-form .arco-form-item-label-col {
    flex: 0 0 80px !important;
    width: 80px;
    max-width: 80px;
}

.manifest-dialog-form .arco-form-item-wrapper-col {
    flex: 1 1 auto !important;
    max-width: calc(100% - 80px);
    min-width: 0;
}

.manifest-dialog-form .arco-form-item-content-wrapper,
.manifest-dialog-form .arco-form-item-content {
    width: 100%;
    min-width: 0;
}
</style>
