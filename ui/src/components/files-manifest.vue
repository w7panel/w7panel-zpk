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

                            <a-form-item v-if="form.type == 'tradition'" label="环境类型">
                                <div class="df df-ww env-sel">
                                    <div v-for="(item, index) in environmentList" :key="index"
                                        :class="{ 'active': form.environmentName == item.identifie }"
                                        @click="changeEnv(item)" class="env-item df df-c ai-c cursor">
                                        <img :src="'http://zpk.w7.cc' + item.icon" class="img" alt="" />
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

                            <a-form-item
                                v-if="!option || !option.pureManifest && form.type != 'docker' && form.type != 'light' && form.type != 'helm'"
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
                                </div>
                                <div v-if="zip.hasDockerfile === false" class="c-red mt-10">没有检测到Dockerfile文件，请重新上传
                                </div>
                            </a-form-item>

                            <slot></slot>

                            <a-form-item v-if="form.type == 'tradition' && !isTraditionCommandDisabled" label="CMD">
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
                                                content="容器的启动参数。该参数为可选参数，如果不填写，则默认使用 Dockerfile 中的 CMD。输入规范，以“空格”作为参数的分割标识，例如 -u app.py"
                                                position="top">
                                                <icon-exclamation-circle-fill class="fs-16 c-99 ml-4" />
                                            </a-tooltip>
                                        </div>
                                    </div>
                                </div>
                            </a-form-item>

                            <a-form-item v-if="form.type != 'helm' && form.type != 'light'"
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

                    <div v-if="form.type != 'helm' && form.type != 'tradition'" class="bg-white com-line mt-20">
                        <div class="mt-16">

                            <a-form-item label="应用配置">
                                <div v-if="form.containers && form.containers.length">
                                    {{form.containers.map(i => i.name).join(', ')}}</div>
                                <div v-else>无</div>
                                <span class="cursor c-blue ml-10" @click="openAppset">编辑</span>
                            </a-form-item>


                            <a-form-item v-if="form.type != 'helm'" label="域名设置" class="mt-16">
                                <div style="flex:1;">

                                    <form-ingress v-model="form.ingress" :app-names="app_names" :app-ports="app_ports"
                                        :mainapp="option && option.mainapp" :identifie="identifie"
                                        @checkDomainStartParams="checkDomainStartParams" />
                                </div>
                            </a-form-item>
                        </div>
                    </div>

                    <div class="bg-white com-line mt-20">
                        <a-form-item class="mt-16" label="启动参数" field="startParams">
                            <div class="manifest-field-stack">
                                <div class="start-param-head">
                                    <div class="start-param-services">
                                        <a-checkbox v-model="form.mysql8">mysql8.0</a-checkbox>
                                        <a-checkbox v-model="form.mysql5">mysql5.6</a-checkbox>
                                        <a-checkbox v-model="form.redis">redis</a-checkbox>
                                        <a-checkbox v-model="form.mongodb6">mongodb</a-checkbox>
                                    </div>
                                    <a-button @click="openSpEdit">批量修改</a-button>
                                </div>
                                <manifest-config-table class="mt-10" :rows="form.startParams" add-text="添加启动参数"
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
                                                <span class="c-blue cursor handle" @click="openSpDesc(record)">编辑描述</span>
                                                <span class="c-blue cursor handle"
                                                    @click="form.startParams.splice(index, 1); getStart();">删除</span>
                                            </template>
                                        </manifest-config-table-column>
                                    </template>
                                </manifest-config-table>
                            </div>
                        </a-form-item>

                        <a-form-item class="mt-20" label="安装依赖">
                            <manifest-config-table :rows="form.depends" table-class="install-depend-table"
                                add-text="添加安装依赖"
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
                                                <a-button type="text" size="mini" @click="openDependPicker(index)">
                                                    {{ record.identifie ? '更换' : '选择' }}
                                                </a-button>
                                            </div>
                                        </template>
                                    </manifest-config-table-column>
                                    <manifest-config-table-column data-index="subidentifie" title="子应用" width="34%">
                                        <template #cell="{ record, index }">
                                            <a-select v-model="record.subidentifie" size="large"
                                                @change="record.subname = getSubDependName(index, record.subidentifie); changeForm()"
                                                placeholder="请选择" :disabled="!record.identifie">
                                                <a-option v-for="(name, identifie) in getSubDependsOptions(index)"
                                                    :key="identifie" :label="name" :value="identifie"></a-option>
                                            </a-select>
                                        </template>
                                    </manifest-config-table-column>
                                    <manifest-config-table-column data-index="required" title="必须安装" width="10%">
                                        <template #cell="{ record }">
                                            <a-switch v-model="record.required" @change="changeForm" />
                                        </template>
                                    </manifest-config-table-column>
                                    <manifest-config-table-column data-index="name" title="标识" width="12%">
                                        <template #cell="{ record }">{{ record.identifie }}</template>
                                    </manifest-config-table-column>
                                    <manifest-config-table-column title="操作" width="8%">
                                        <template #cell="{ index }">
                                            <span class="c-blue cursor"
                                                @click="form.depends.splice(index, 1); changeForm();">删除</span>
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
                environmentVersion: '',

            },

            shellConfig: {
                show: false,
                editIndex: -1,
                item: null,
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
        helmChartOptions() {
            return this.filterAutocompleteOptions(this.helmCharts, this.helmChartKeyword);
        },
        helmChartVersionOptions() {
            return this.filterAutocompleteOptions(this.helmChartVersions, this.helmChartVersionKeyword);
        },
        selectedEnvironment() {
            return this.environmentList?.find?.(i => i.identifie == this.form.environmentName) || null;
        },
        isTraditionCommandDisabled() {
            return this.form.type == 'tradition' && this.selectedEnvironment?.image_is_share === true;
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

        data() { this.init(this.data) },
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
        onChange() { },
        createDomainStartParam() {
            return {
                mark: 'domain',
                name: 'DOMAIN_URL',
                values_text: '%DOMAIN_SSL_URL%',
                title: '域名',
                required: true,
                module_name: '',
            };
        },
        isDomainStartParam(item) {
            return item?.mark === 'domain'
                || (item?.name === 'DOMAIN_URL' && ['%DOMAIN_URL%', '%DOMAIN_SSL_URL%'].includes(item?.values_text));
        },
        computedSpDisabled(item) {
            return ((this.disabledDomainStartParams || this.form.type == 'tradition') && this.isDomainStartParam(item))
                || (this.json?.platform?.['volumeClaimTemplates']?.length && item.mark === 'storage')
        },
        serializeStartParams() {
            let start = [];
            for (let i in this.form.startParams) {
                let o = this.form.startParams[i];
                if (o.name) {
                    start.push({
                        type: 'text',
                        name: o.name,
                        title: o.title,
                        required: o.required,
                        values_text: o.values_text,
                        module_name: o.module_name,
                        description: o.description || '',
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
        normalizeCommandByEnvironment() {
            if (this.isTraditionCommandDisabled) {
                this.form.cmd = [''];
            }
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


                this.json.platform.runtimeClassName = data?.pluginData?.gpu ? 'nvidia' : '';

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
            if (item.identifie == this.form.environmentName) { return }

            this.form.depends = this.form.depends?.filter?.(i => !i.temporary) || []
            let findIndex = this.form.depends?.findIndex(i => i.identifie == item.identifie && i.name == item.name)
            if (findIndex != -1) {
                this.form.depends[findIndex].temporary = true;
                this.form.depends[findIndex].from = 'https://zpk.w7.cc';
            } else {
                this.form.depends.push({
                    identifie: item.identifie,
                    name: item.name,
                    subidentifie: '',
                    required: true,
                    type: 'out',
                    temporary: true,
                    from: 'https://zpk.w7.cc',
                })
            }

            if (!this.form.startParams?.find?.(i => i.values_text == '%DOMAIN_SSL_URL%' || i.values_text == '%DOMAIN_URL%')) {
                this.form.startParams.push(this.createDomainStartParam())
                this.getStart();
            } else {
                this.form.startParams.forEach(i => {
                    if (this.isDomainStartParam(i)) { i.mark = 'domain'; }
                });
            }

            this.form.environmentName = item.identifie;
            this.form.environmentVersion = '';

            if (item.versions?.length) {
                this.form.environmentVersion = item.versions[0];
            }
            this.normalizeCommandByEnvironment();

            this.changeForm();
        },
        getEnvironmentList() {
            myAxios.get('https://zpk.w7.cc/zpk/respo/list?status=2&status=99&page=1&limit=9&tag=运行环境').then(res => {
                const environmentList = res.data?.data?.list
                let arr = environmentList || [];
                this.environmentList = arr.map(i => {
                    let versions = [];
                    try {
                        versions = i.annotation['w7.cc/image_version'].split(',')
                    } catch { }
                    let imageIsShare = false;
                    try {
                        imageIsShare = String(i.annotation['w7.cc/image_is_share']).toLowerCase() == 'true';
                    } catch {
                        imageIsShare = false;
                    }
                    i.versions = versions;
                    i.image_is_share = imageIsShare;
                    return i
                })
                this.normalizeCommandByEnvironment();

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
            })
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

            if (!this.option?.pureManifest) {
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
                this.form.environmentVersion = j.platform?.tradition?.environmentVersion || '';
                this.form.cmd = j.platform?.tradition?.cmd || [''];

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
                this.form.startParams = JSON.parse(JSON.stringify(startParams?.length ? startParams : []));

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
        submit(otherData, callback) {

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
                    if (this.json.platform?.tradition) {
                        this.json.platform.tradition.cmd = this.isTraditionCommandDisabled ? [] : this.form.cmd;
                    }
                    this.applyPlatformShells();
                } else if (this.form.type == 'docker') {

                    this.applyPlatformShells();

                    this.json.platform.ingress = (this.form.ingress || []).map(i => ({
                        name: i.name,
                        routes: this.formatIngressRoutes(i.routes),
                    }));
                }

                this.json.platform.startParams = this.serializeStartParams();
                this.yaml = jsyaml.dump(this.json);

                this.$emit('complete', this.json, this.yaml, otherData, callback);
            });
        },
        changeFormtype() {
            if (this.form.type !== 'light' && this.zip.url) {
                if (!this.json.source) { this.json.source = {}; }
                this.json.source.type = 'zip';
                this.json.source.url = this.zip.url;
            }
            if (this.form.type !== 'tradition' && this.web.url) {
                if (!this.json.web) { this.json.web = {}; }
                this.json.web.type = 'zip';
                this.json.web.url = this.web.url;
            }

            if (this.form.type == 'light') {

                if (this.json.source) { delete this.json.source; }
            }
            if (this.form.type == 'tradition') {

                if (!this.form.startParams?.find?.(i => i.values_text == '%DOMAIN_SSL_URL%' || i.values_text == '%DOMAIN_URL%')) {
                    this.form.startParams.push(this.createDomainStartParam())
                } else {
                    this.form.startParams.forEach(i => {
                        if (this.isDomainStartParam(i)) { i.mark = 'domain'; }
                    });
                }
                this.getStart();

                if (this.json.web) { delete this.json.web; }
            }

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
                j.application.once = Boolean(this.form.once);
                if (this.form.type != 'tradition') {
                    this.form.language = '';
                }
            }
            j.platform = j.platform || {};

            if (this.form.type == 'tradition') {
                let environmentLanguage = j.platform?.tradition?.environmentLanguage || '';
                try {
                    environmentLanguage = this.environmentList?.find?.(i => i.identifie == this.form.environmentName)?.annotation?.['w7.cc/image_language'] || environmentLanguage;
                } catch { }
                this.normalizeCommandByEnvironment();
                j.platform.tradition = {
                    environmentName: this.form.environmentName,
                    environmentVersion: this.form.environmentVersion,
                    environmentLanguage: environmentLanguage,
                    cmd: this.isTraditionCommandDisabled ? [] : this.form.cmd,
                }
            }

            if (j.platform) {
                if (this.form.type !== 'helm' || true) {

                    if (!this.option?.pureManifest) {
                        j.platform.baseInfo = j.platform.baseInfo || {};
                        j.platform.baseInfo.name = this.form.name;
                        j.platform.baseInfo.identifie = (this.form.author && this.form.identifie) ? (this.form.author + '-' + this.form.identifie) : '';
                        j.platform.baseInfo.description = this.form.description;
                    }
                }


                j.platform.depends = this.form.dependsIn.concat(this.form.depends);
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
