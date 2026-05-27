<template>
    <div class="df content">
        <div class="fc" style="padding-left:0; overflow:auto;">
            <div>
                <el-form ref="formref" :model="form" label-width="120px" :rules="rules" label-position="left"
                    class="form manifest-form">
                    <div class="df jc-e">
                        <el-button v-if="!showYaml" type="primary" @click="showYaml = true;">预览yaml</el-button>
                    </div>
                    <div class="bg-white com-line df">
                        <div class="fc">
                            <div class="c-00-6 df ai-c">基础配置</div>
                            <el-form-item class="mt-16" label="名称" :prop="option?.pureManifest ? 'name' : ''">
                                <el-input v-model="form.name" size="large" style="width:500px;" @change="changeForm"
                                    placeholder="请输入"></el-input>
                            </el-form-item>
                            <el-form-item label="标识" :prop="option?.pureManifest ? 'identifie' : ''">
                                <div class="df jc-b" style="width:500px;">
                                    <w7-identifie v-model:author="form.author" v-model:identifie="form.identifie"
                                        @change="changeForm" disabled />
                                </div>
                            </el-form-item>
                            <el-form-item label="描述" prop="description">
                                <div class="df df-c">
                                    <el-input v-model="form.description" size="large" style="width:500px;"
                                        placeholder="请输入应用描述" @change="changeForm"></el-input>
                                </div>
                            </el-form-item>

                            <el-form-item v-if="option.pureManifest" label="可选安装">
                                <el-switch v-model="otherData.required" :active-value="false"
                                    :inactive-value="true"></el-switch>
                            </el-form-item>
                        </div>
                    </div>
                    <div class="bg-white mt-20 pb-24">
                        <div class="c-00-6 df ai-c">
                            <span class="">代码配置</span>
                        </div>
                        <div class="mt-16">
                            <el-form-item label="类型">
                                <div class="df df-c fc">

                                    <el-radio-group v-model="form.type" @change="changeFormtype">
                                        <el-radio label="docker">原生应用</el-radio>
                                        <el-radio label="tradition">传统应用</el-radio>
                                        <el-radio label="helm">K8sYaml</el-radio>
                                    </el-radio-group>

                                    <el-form-item v-if="form.type == 'helm'" class="mt-20" style="margin-bottom:10px;"
                                        label="启用helm配置">
                                        <el-switch v-model="form.helm.useHelm" @change="changeForm"></el-switch>
                                    </el-form-item>

                                    <div v-if="form.type == 'helm' && form.helm.useHelm" class="greybox"
                                        style="margin-bottom:20px;">
                                        <div class="greybox-title">Helm配置</div>

                                        <el-form-item label="Chart包来源" style="margin-bottom:18px;">
                                            <el-radio-group v-model="form.helm.helmtype" @change="changeForm">
                                                <el-radio label="1">helm仓库</el-radio>
                                                <el-radio label="2">helm下载包</el-radio>
                                            </el-radio-group>
                                        </el-form-item>

                                        <el-form-item v-if="form.type == 'helm' && form.helm.helmtype == '1'"
                                            label="helm仓库地址" prop="repository" style="margin-bottom:18px;">
                                            <el-input v-model="form.helm.repository" size="large"
                                                @change="getChartInfo(); changeForm();" style="width:500px;"
                                                placeholder="请输入" />
                                            <el-icon v-if="getChartInfoLoading" class="is-loading fs-24 ml-10">
                                                <Loading />
                                            </el-icon>
                                        </el-form-item>

                                        <el-form-item v-if="form.type == 'helm' && form.helm.helmtype == '1'"
                                            label="chart名称" prop="chartName" style="margin-bottom:18px;">
                                            <el-autocomplete ref="formhelmchartname" v-model="form.helm.chartName"
                                                :fetch-suggestions="async (queryString, callback) => {
                                                    const results = queryString
                                                        ? helmCharts.filter(item => item.toLowerCase().includes(queryString.toLowerCase()))
                                                        : helmCharts;
                                                    callback(results.map(item => ({ value: item, label: item })));
                                                }" placeholder="请输入内容" @change="getChartVersion(); changeForm();"
                                                @select="getChartVersion(); changeForm(); $refs.formhelmchartname.close()"
                                                style="width:240px;"></el-autocomplete>

                                            <el-autocomplete ref="formhelmversion" v-model="form.helm.version"
                                                :fetch-suggestions="async (queryString, callback) => {
                                                    const results = queryString
                                                        ? helmChartVersions.filter(item => item.toLowerCase().includes(queryString.toLowerCase()))
                                                        : helmChartVersions;
                                                    callback(results.map(item => ({ value: item, label: item })));
                                                }" placeholder="请输入内容" @change="changeForm"
                                                @select="changeForm(); $refs.formhelmversion.close()"
                                                style="width:240px;margin-left:20px;"></el-autocomplete>
                                        </el-form-item>

                                        <el-form-item v-if="form.type == 'helm' && form.helm.helmtype == '2'"
                                            label="Chart包地址" prop="chartName2" style="margin-bottom:18px;">
                                            <el-input v-model="form.helm.chartName2" size="large" style="width:500px;"
                                                @change="changeForm" placeholder="请输入" />
                                            <files-upload accept=".tgz" @success="helmUploadSuccess">
                                                <el-button type="primary" size="large"
                                                    style="margin-left:10px;">上传</el-button>
                                            </files-upload>
                                        </el-form-item>

                                        <el-form-item v-if="form.type == 'helm'" label="安装配置">
                                            <table class="table">
                                                <thead>
                                                    <tr class="thead">
                                                        <td>键</td>
                                                        <td>值</td>
                                                        <td>操作</td>
                                                    </tr>

                                                </thead>
                                                <tbody>
                                                    <tr v-for="(item, index) in form.helm.kv" :key="index">
                                                        <td>
                                                            <el-input v-model="item.name" size="large"
                                                                style="width:200px;" @change="changeForm"
                                                                placeholder="请输入" />
                                                        </td>
                                                        <td>
                                                            <el-input v-model="item.value" size="large"
                                                                style="width:200px;" @change="changeForm"
                                                                placeholder="请输入" />
                                                        </td>
                                                        <td>
                                                            <span v-if="!item.disabled" class="c-blue cursor"
                                                                @click="form.helm.kv.splice(index, 1); changeForm();">删除</span>
                                                        </td>
                                                    </tr>
                                                    <tr>
                                                        <td colspan="3" style="box-sizing:border-box; cursor:pointer;"
                                                            @click="form.helm.kv.push({ name: '', value: '' }); changeForm();">
                                                            <div class="df ai-c jc-c">
                                                                <span class="addmenu"><el-icon :size="14">
                                                                        <Plus />
                                                                    </el-icon>添加配置</span>
                                                            </div>
                                                        </td>
                                                    </tr>
                                                </tbody>
                                            </table>
                                        </el-form-item>

                                    </div>
                                    <div v-if="form.type == 'helm'" class="greybox">
                                        <div class="greybox-title">YAML配置</div>
                                        <div>
                                            <div v-for="(item, index) in form.helm.depend_yamls" :key="index"
                                                style="margin-bottom:30px;" class="show-on-hover-container">
                                                <div class="df yaml-header">
                                                      <el-upload
                                                        :before-upload="(e) => {helmyamlsUpload(e, index);return false}"
                                                        :show-file-list="false"
                                                    >
                                                    <el-button type="primary">上传文件</el-button>
                                                    </el-upload>
                                                    <div class="show-on-hover yaml-delete">
                                                        <el-icon :size="20" class="cursor"
                                                            @click="form.helm.depend_yamls.splice(index, 1); changeForm();">
                                                            <close />
                                                        </el-icon>
                                                    </div>

                                                </div>
                                                <el-form-item label="标题" style="margin-bottom: 20px">
                                                    <el-input v-model="item.nameInput" placeholder="标题"
                                                        @change="changeForm" style="width:500px;"><template
                                                            #append>.yaml</template></el-input>
                                                </el-form-item>
                                                <el-form-item label="YAML">
                                                    <el-input v-model="item.yaml" rows="5" type="textarea"
                                                        placeholder="请输入YAML" class="fc" @change="changeForm" />
                                                    <div class="command-upfile ml-20">
                                                        <input type="file" @change="e => helmyamlsUpload(e, index)" />
                                                    </div>
                                                </el-form-item>
                                            </div>
                                            <div class="df ai-c jc-c cursor"
                                                @click="form.helm.depend_yamls.push({ nameInput: '', name: '', yaml: '' })">
                                                <span class="addmenu"><el-icon :size="14">
                                                        <Plus />
                                                    </el-icon>添加YAML</span>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </el-form-item>

                            <el-form-item v-if="form.type == 'tradition'" label="环境类型">
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
                            </el-form-item>
                            <el-form-item v-if="form.type == 'tradition'" label="环境版本">
                                <el-select v-model="form.environmentVersion" placeholder="请选择" @change="changeForm"
                                    size="large" style="width:500px;">
                                    <el-option
                                        v-for="(item, index) in environmentList?.find?.(i => i.identifie == form.environmentName)?.versions || []"
                                        :key="index" :label="item" :value="item"></el-option>
                                </el-select>
                            </el-form-item>

                            <el-form-item
                                v-if="!option || !option.pureManifest && form.type != 'docker' && form.type != 'light' && form.type != 'helm'"
                                label="代码包">
                                <div class="df ai-e">
                                    <files-upload @success="uploadSuccess" @testDockerfile="v => zip.hasDockerfile = v">
                                        <div v-if="zip.name" class="upfilebox df df-c ai-c jc-c">
                                            <img src="@/assets/img/zip.png" alt=""
                                                style="width:60px;height:60px;display:block;" />
                                            <div class="df ai-c mt-20">
                                                <el-icon :size="18" class="c-green"
                                                    style="margin-right:6px;vertical-align:middle;">
                                                    <CircleCheckFilled />
                                                </el-icon>
                                                <div class="fs-14 c-33"
                                                    style="vertical-align:middle;max-width:200px;overflow:hidden;text-overflow:ellipsis;">
                                                    {{
                                                        zip.name }}</div>
                                            </div>
                                            <div class="mask df df-c ai-c jc-c">
                                                <el-button type="primary">重新上传</el-button>
                                            </div>
                                        </div>
                                        <div v-else class="upfilebox df df-c ai-c jc-c">
                                            <div class="df df-c ai-c">
                                                <el-icon class="uploadicon c-99">
                                                    <UploadFilled />
                                                </el-icon>
                                                <span class="uploadbtn df ai-c">
                                                    <el-icon class="uploadicon c-33">
                                                        <Upload />
                                                    </el-icon>
                                                    <span class="lh-1 c-33">上传代码包</span>
                                                </span>
                                            </div>
                                        </div>
                                    </files-upload>
                                    <div class="c-blue cursor ml-20"
                                        @click="$router.push('/zpk-filetree?id=' + identifie + '&versionid=' + version_id + '&vtitle=' + vtitle)">
                                        编辑</div>
                                    <div class="c-blue cursor ml-20" @click="deleteUpload">删除</div>
                                </div>
                                <div v-if="zip.hasDockerfile === false" class="c-red mt-10">没有检测到Dockerfile文件，请重新上传
                                </div>
                            </el-form-item>

                            <slot></slot>

                            <el-form-item v-if="form.type == 'tradition'" label="CMD">
                                <div class="df df-c">
                                    <div v-for="(item, index) in form.cmd" :key="index" class="df ai-e"
                                        :style="{ marginTop: index == 0 ? 0 : '10px' }">
                                        <el-input v-model="form.cmd[index]" @change="changeForm" type="textarea"
                                            :spellcheck="false" :rows="2" style="width:500px;"
                                            placeholder="请输入"></el-input>
                                        <div class="df ai-c">
                                            <span class="ml-10 cursor c-blue"
                                                @click="form.cmd.length > 1 ? form.cmd.splice(index, 1) : form.cmd = ['']; changeForm();">删除</span>
                                            <span class="ml-10 cursor c-blue" v-if="index + 1 == form.cmd.length"
                                                @click="form.cmd.push('')">添加</span>
                                            <el-popover placement="top-start" v-if="index + 1 == form.cmd.length"
                                                :width="300" trigger="hover"
                                                content="容器的启动参数。该参数为可选参数，如果不填写，则默认使用 Dockerfile 中的 CMD。输入规范，以“空格”作为参数的分割标识，例如 -u app.py">
                                                <template #reference>
                                                    <el-icon class="fs-16 c-99 ml-4">
                                                        <WarningFilled />
                                                    </el-icon>
                                                </template>
                                            </el-popover>
                                        </div>
                                    </div>
                                </div>
                            </el-form-item>

                            <el-form-item v-if="form.type != 'tradition' && form.type != 'helm' && form.type != 'light'"
                                label="脚本配置" prop="shell" class="mt-16">
                                <div style="flex:1;">
                                    <table class="table">
                                        <thead>
                                            <tr>
                                                <td>类型</td>
                                                <td>脚本</td>
                                                <td>镜像</td>
                                                <td>备注</td>
                                                <td>操作</td>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            <tr v-for="(item, index) in form.shell" :key="index">
                                                <td>
                                                    <el-select v-model="item.type" placeholder="请选择类型"
                                                        style="width:160px;">
                                                        <el-option label="应用被安装时触发" value="requireinstall"></el-option>
                                                        <el-option label="应用安装时触发" value="install"></el-option>
                                                        <el-option label="应用安装前触发" value="pre-install"></el-option>
                                                        <el-option label="应用更新时触发" value="upgrade"></el-option>
                                                        <el-option label="应用卸载时触发" value="uninstall"></el-option>
                                                        <el-option label="手动触发" value="custom"></el-option>
                                                    </el-select>
                                                </td>
                                                <td>
                                                    <el-input v-model="item.shell" :rows="2" type="textarea"
                                                        :spellcheck="false" placeholder="请输入"></el-input>
                                                </td>
                                                <td>
                                                    <el-input v-model="item.image" :spellcheck="false"
                                                        placeholder="不填默认当前应用镜像"></el-input>
                                                </td>
                                                <td>
                                                    <el-input v-model="item.title" placeholder="请输入"></el-input>
                                                </td>
                                                <td><span class="c-blue cursor handle"
                                                        @click="form.shell.splice(index, 1);">删除</span></td>
                                            </tr>
                                            <tr>
                                                <td colspan="5" class="cursor txt-c"
                                                    @click="form.shell.push({ title: '', type: '', shell: '' })">
                                                    <span class="addmenu"><el-icon :size="14">
                                                            <Plus />
                                                        </el-icon>添加脚本</span>
                                                </td>
                                            </tr>
                                        </tbody>
                                    </table>
                                </div>
                            </el-form-item>


                        </div>
                    </div>

                    <div v-if="form.type != 'helm' && form.type != 'tradition'" class="bg-white com-line mt-20">
                        <div class="mt-16">

                            <el-form-item label="应用配置">
                                <div v-if="form.containers && form.containers.length">
                                    {{form.containers.map(i => i.name).join(', ')}}</div>
                                <div v-else>无</div>
                                <span class="cursor c-blue ml-10" @click="openAppset">编辑</span>
                            </el-form-item>


                            <el-form-item v-if="form.type != 'helm'" label="域名设置" class="mt-16">
                                <div style="flex:1;">

                                    <form-ingress v-model="form.ingress" :app-names="app_names" :app-ports="app_ports"
                                        :mainapp="option && option.mainapp" :identifie="identifie"
                                        @checkDomainStartParams="checkDomainStartParams" />
                                </div>
                            </el-form-item>
                        </div>
                    </div>

                    <div class="bg-white com-line mt-20">
                        <el-form-item class="mt-16" label="启动参数" prop="startParams">
                            <div>
                                <el-checkbox v-model="form.mysql8" label="mysql8.0" />
                                <el-checkbox v-model="form.mysql5" label="mysql5.6" />
                                <el-checkbox v-model="form.redis" label="redis" />
                                <el-checkbox v-model="form.mongodb6" label="mongodb" />
                            </div>
                            <div style="width:100%;">
                                <el-button type="primary" @click="openSpEdit">批量修改</el-button>
                            </div>
                            <table class="table mt-10">
                                <thead>
                                    <tr>
                                        <td>标识</td>
                                        <td>名称</td>
                                        <td>必填</td>
                                        <td>默认值</td>
                                        <td>依赖系统组件标识</td>
                                        <td style="width:100px;">操作</td>
                                    </tr>
                                </thead>
                                <tbody>
                                    <tr v-for="(item, index) in form.startParams" :key="index">
                                        <td><el-input v-model="item.name" @change="getStart"
                                                :disabled="computedSpDisabled(item)" :spellcheck="false"
                                                placeholder="配置标识"></el-input></td>
                                        <td><el-input v-model="item.title" @change="getStart"
                                                :disabled="computedSpDisabled(item)" :spellcheck="false"
                                                placeholder="配置名称"></el-input></td>
                                        <td><el-switch v-model="item.required" @change="getStart"
                                                :disabled="computedSpDisabled(item)" />
                                        </td>
                                        <td><el-input v-model="item.values_text" @change="getStart"
                                                :disabled="computedSpDisabled(item)" :spellcheck="false"
                                                placeholder="配置默认值"></el-input></td>
                                        <td><el-input v-model="item.module_name" @change="getStart" :spellcheck="false"
                                                placeholder="依赖的系统组件标识名"></el-input></td>
                                        <td>
                                            <span class="c-blue cursor handle" @click="openSpDesc(item)">编辑描述</span>
                                            <span class="c-blue cursor handle"
                                                @click="form.startParams.splice(index, 1); getStart();">删除</span>
                                        </td>
                                    </tr>
                                    <tr>
                                        <td colspan="6" class="cursor txt-c"
                                            @click="form.startParams.push({ name: '', title: '', required: true, values_text: '', module_name: '', description: '' })">
                                            <span class="addmenu"><el-icon :size="14">
                                                    <Plus />
                                                </el-icon>添加启动参数</span>
                                        </td>
                                    </tr>
                                </tbody>

                            </table>
                        </el-form-item>

                        <el-form-item class="mt-20" label="安装依赖">
                            <table class="table">
                                <thead>
                                    <tr>
                                        <td>名称</td>
                                        <td>子应用</td>
                                        <td>必须安装</td>
                                        <td>标识</td>
                                        <td>操作</td>
                                    </tr>
                                </thead>
                                <tbody>
                                    <tr v-for="(item, index) in form.depends" :key="index">
                                        <td>
                                            <el-select v-model="item.identifie" size="large"
                                                @change="getSubDepends(index); item.subidentifie = ''; item.subname = ''; item.name = dependsList[item.identifie]; changeForm()"
                                                placeholder="请选择">
                                                <el-option v-for="(name, identifie) in dependsList" :key="identifie"
                                                    :label="name" :value="identifie"></el-option>
                                            </el-select>
                                        </td>
                                        <td>
                                            <el-select v-model="item.subidentifie" size="large"
                                                @change="item.subname = subDependsList[index][item.subidentifie]; changeForm()"
                                                placeholder="请选择">
                                                <el-option v-for="(name, identifie) in subDependsList[index]"
                                                    :key="identifie" :label="name" :value="identifie"></el-option>
                                            </el-select>
                                        </td>
                                        <td>
                                            <el-switch v-model="item.required" @change="changeForm" />
                                        </td>
                                        <td>{{ dependsList[item.identifie] }}</td>
                                        <td>
                                            <span class="c-blue cursor"
                                                @click="form.depends.splice(index, 1); changeForm();">删除</span>
                                        </td>
                                    </tr>
                                    <tr>
                                        <td colspan="5" class="cursor txt-c"
                                            @click="form.depends.push({ identifie: '', name: '', required: false, type: 'out' })">
                                            <span class="addmenu"><el-icon :size="14">
                                                    <Plus />
                                                </el-icon>添加安装依赖</span>
                                        </td>
                                    </tr>
                                </tbody>
                            </table>
                        </el-form-item>
                    </div>

                    <div class="bg-white pb-24 mt-20 df ai-c">
                        <el-button v-if="option.pureManifest" :loading="submiting" type="primary"
                            @click="submit(otherData)" style="width:90px;">确定提交</el-button>
                        <el-button v-else :loading="submiting" type="primary" @click="submit()"
                            style="width:90px;">确定提交</el-button>
                    </div>
                </el-form>
            </div>
        </div>
        <div v-show="showYaml" class="box" style="width:600px; position:relative; padding-right:0;">
            <div style="height:100%;" v-html="yamlDom"></div>
            <div class="df" style="position:absolute; right:20px; top:10px;">
                <button class="copybtn" @click="showYaml = false;">收起预览</button>
                <button class="copybtn" @click="onekeyCopy(yaml)">一键复制</button>
                <a :href="downloadUrl" download="manifest.yaml" class="copybtn" style="right:110px;">下载</a>
            </div>
        </div>

        <el-dialog v-model="dependForm.show" :title="dependForm.editIndex >= 0 ? '修改子应用' : '添加子应用'" width="640px"
            @close="dependForm = { show: false, editIndex: -1, identifie_before: '', identifie_last: '', identifie: '', name: '', required: false, from: '' };">
            <el-form ref="depend" :rules="addRules" :model="dependForm" label-width="80px">
                <el-form-item label="标识" prop="identifie">
                    <w7-identifie v-model:author="dependForm.identifie_before"
                        v-model:identifie="dependForm.identifie_last" @change="onChange" :author-disabled="true" />
                </el-form-item>
                <el-form-item label="名称" prop="name">
                    <el-input placeholder="请输入名称" v-model="dependForm.name" size="large"
                        style="width:500px;"></el-input>
                </el-form-item>
                <el-form-item label="可选安装" prop="from">
                    <el-switch v-model="dependForm.required" :active-value="false" :inactive-value="true" />
                </el-form-item>
                <el-form-item label="" class="mt-20">
                    <el-button type="primary" size="large" @click="addDepend">确定</el-button>
                </el-form-item>
            </el-form>
        </el-dialog>

        <el-dialog v-model="spEdit.show" title="启动参数" width="700px" custom-class="envdialog">
            <span class="c-66">格式：键=值 #标题 : 描述 : 必填(1或0) : 依赖组件标识</span>
            <el-input v-model="spEdit.values" class="mt-10" type="textarea" :spellcheck="false"
                placeholder="格式：键=值 #标题 : 描述 : 必填(1或0) : 依赖组件标识" :rows="12"
                :input-style="{ lineHeight: '24px' }"></el-input>
            <div class="df ai-c jc-c mt-20">
                <el-button @click="spEdit.show = false;">取消</el-button>
                <el-button @click="submitSpEdit" type="primary">确定</el-button>
            </div>
        </el-dialog>

    </div>
</template>

<script>
import jsyaml from "js-yaml";
import hljs from 'highlight.js';
import filesUpload from './files-upload.vue';
import formIngress from '@/components/form-ingress.vue';
import w7Identifie from "@/components/w7-identifie.vue";
import myAxios from '../utils/index';

import { bus } from "wujie";

export default {
    emits: ['writefile'],
    props: [
        'data',
        'submiting',
        'option',
        'identifie',
        'version_id',
    ],
    components: { filesUpload, formIngress, w7Identifie },
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
                        required: true, trigger: 'blur', validator: (rule, value, callback) => {
                            if (this.form.author) { callback() }
                            else { callback(new Error("请输入完整")) }
                        }
                    },
                    {
                        required: true, trigger: 'blur', validator: (rule, value, callback) => {
                            if (/^[a-zA-Z0-9]+$/.test(value)) { callback() }
                            else { callback(new Error("标识格式有误")) }
                        }
                    },
                    {
                        required: true, trigger: 'blur', validator: (rule, value, callback) => {
                            if (/^[a-zA-Z0-9]+$/.test(this.form.author)) { callback() }
                            else { callback(new Error("标识格式有误")) }
                        }
                    },
                ],
                port: [
                    { required: true, message: '内容不能为空', trigger: 'blur' },
                    {
                        required: true, trigger: 'blur', validator: (rule, value, callback) => {
                            if (value?.filter?.(i => i.name && i.protocol && i.port)?.length) {
                                callback()
                            } else {
                                callback(new Error("必填项不能为空"))
                            }
                        }
                    },
                ],
            },
            addRules: {
                identifie: [
                    { required: true, message: '内容不能为空', trigger: 'blur' },

                    {
                        required: true, trigger: 'blur', validator: (rule, value, callback) => {
                            if (/^[a-zA-Z0-9]+-[a-zA-Z0-9]+$/.test(value)) { callback() }
                            else { callback(new Error("标识格式有误")) }
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

            dependsList: {},
            subDependsList: {},

            buildImageLogData: null,
            buildImageInterval: null,


            helmCharts: [],
            helmChartVersions: [],

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
        'option.app_ports'(v) {
            this.computedAppPort(this.json.platform?.['container-v2']);
        }
    },
    beforeUnmount() {
        try {
            clearInterval(this.buildImageInterval)
        } catch { }
    },
    methods: {
        computedSpDisabled(item) {
            return ((this.disabledDomainStartParams || this.form.type == 'tradition') && item.mark === 'domain')
                || (this.json?.platform?.['volumeClaimTemplates']?.length && item.mark === 'storage');
        },
        openAppset() {
            let volumes = this.json?.platform?.volumes;
            let volumeClaimTemplates = this.json?.platform?.volumeClaimTemplates;
            let containers = this.json?.platform?.['container-v2']?.filter(i => !i.isInitContainer);
            let initContainers = this.json?.platform?.['container-v2']?.filter(i => i.isInitContainer);

            window.$wujie?.bus.$emit("containerPlugin", {
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

                this.form.storage = Boolean(data?.volumeClaimTemplates?.length);


                this.json.platform.runtimeClassName = data?.pluginData?.gpu ? 'nvidia' : '';

                this.json.platform.workload = this.json?.platform?.workload || {};
                this.json.platform.workload.type = data?.pluginData?.kind || 'deployments';

                this.json.platform['container-v2'][0].shells = this.form.shell.filter(i => i.type && i.shell);

                this.json.platform.ingress = (this.form.ingress || []).map(i => ({
                    name: i.name,
                    routes: i.routes.filter(r => {
                        if (r.backend.port) r.backend.port = Number(r.backend.port);
                        return r.path && r.backend?.port;
                    }),
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
            } else {
                this.form.depends.push({
                    identifie: item.identifie,
                    name: item.name,
                    subidentifie: '',
                    required: true,
                    type: 'out',
                    temporary: true,
                })
            }

            if (!this.form.startParams?.find?.(i => i.values_text == '%DOMAIN_SSL_URL%' || i.values_text == '%DOMAIN_URL%')) {
                this.form.startParams.push({
                    name: 'DOMAIN_URL',
                    values_text: '%DOMAIN_SSL_URL%',
                    title: '域名',
                    required: true,
                    module_name: '',
                })
                this.getStart();
            }

            this.form.environmentName = item.identifie;
            this.form.environmentVersion = '';

            if (item.versions?.length) {
                this.form.environmentVersion = item.versions[0];
            }

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
                    i.versions = versions;
                    return i
                })

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
                this.$message.error('请上传yaml文件');
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
        getSubDepends(index) {
            this.subDependsList[index] = { "": "无" };
            let identifie = this.form.depends[index].identifie;
            return myAxios.get('/respo/v2/info/' + identifie + '/1.0.0', {
                headers: { cancelerror: true }
            }).then(res => {
                let json = jsyaml.load(res?.data?.data?.manifest);
                if (json?.platform?.depends?.length) {
                    let d = json?.platform?.depends;
                    d.map(i => {
                        this.subDependsList[index][i.identifie] = i.name;
                    })
                }
            })
        },
        getDependsList() {
            myAxios.get('/respo/list?limit=999').then(res => {
                let obj = {};
                let list = res.data?.data?.list || []
                list?.filter(i => i.install_only_once).map(i => {
                    obj[i.identifie] = i.name;
                })
                this.dependsList = obj;
            })
        },

        async computedAppPort(containers) {
            let ports = {};
            let arr = [];
            if (containers) {
                containers?.map?.(i => {
                    arr = arr.concat(i?.ports?.map?.(j => j.containerPort) || [])
                })
            }
            arr = [...new Set(arr)];
            ports[this.identifie] = arr;

            if (this.option?.app_ports?.length) {
                this.option.app_ports.map(i => {
                    ports[i.name] = i.port;
                })
            }
            this.app_ports = ports;

            let names = [{
                id: this.identifie,
                name: this.identifie,
                title: this.form.name,
            }];

            if (this.option?.app_ports?.length) {
                names = names.concat(this.option.app_ports.map(i => {
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
            this.$prompt('请编辑描述', '描述', {
                inputValue: item.description,
            }).then((value) => {
                item.description = value.value || '';
                this.getStart();
            }).catch(() => { });
        },
        openAddDepend() {
            this.dependForm.show = true;
            let identifie = this.json?.platform?.baseInfo?.identifie || this.json?.application?.identifie;
            this.dependForm.identifie_before = identifie.match(/^([^-]+)-(.+)$/)?.[1] || '';
        },
        addDepend() {
            this.$refs.depend.validate((valid) => {
                if (!valid) { return }
                if (!this.dependForm.identifie_before || !this.dependForm.identifie_last) {
                    this.$message.warning('标识请填写完整');
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
                this.$refs.formref.validate(async (valid) => {
                    if (!valid) { this.$message.warning('必填项不能为空'); return }

                    if (this.form.type == 'helm' || this.form.type == 'tradition') {
                        try {
                            delete this.json.platform['container-v2']
                            delete this.json.platform['volumes']
                            delete this.json.platform['volumeClaimTemplates']
                            delete this.json.platform.ingress
                            delete this.json.platform.runtimeClassName
                        } catch { }
                    } else if (this.form.type == 'docker') {

                        if (this.json.platform?.['container-v2']?.[0]) {
                            this.json.platform['container-v2'][0].shells = this.form.shell.filter(i => i.type && i.shell);
                        }

                        this.json.platform.ingress = (this.form.ingress || []).map(i => ({
                            name: i.name,
                            routes: i.routes.filter(r => {
                                if (r.backend.port) r.backend.port = Number(r.backend.port);
                                return r.path && r.backend?.port;
                            }),
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
                this.$message.success('添加成功');
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
                this.$message.success('删除成功');
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
                this.$message.success('添加成功');
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
            this.$confirm('确定要删除吗', "提示", {
                confirmButtonText: "确定",
                cancelButtonText: "取消",
            }).then(() => {
                this.zip.name = '';
                this.zip.url = '';
                delete this.json.source;
                this.setYaml();
            }).catch(() => { })
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
            this.computedAppPort(this.json.platform?.['container-v2']);
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
                this.form.type = j.application.type || 'docker';
            }
            for (let i in j.bindings) {
                let o = j.bindings[i];
                if (!o?.menu?.length) {
                    o.menu = [{ displayorder: 0, do: 'home', title: '首页', icon: 'a-shouye', is_default: 1 }];
                }
                o.framework = o.framework || 'vue2';
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
                this.form.shell = JSON.parse(JSON.stringify(j.platform?.['container-v2']?.[0]?.shells || []));
                this.form.build_context = j.platform?.['container-v2']?.[0]?.build?.context || '';
                this.form.containers = j.platform?.['container-v2'] || [];

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
                bus.$emit('submit' + this.wujieId, (data) => {
                    resolve(data)
                })
            })
        },
        submit(otherData, callback) {

            this.$refs.formref.validate(async (valid) => {
                if (!valid) { this.$message.warning('必填项不能为空'); return }

                if (this.form.type == 'helm' || this.form.type == 'tradition') {
                    try {
                        delete this.json.platform['container-v2']
                        delete this.json.platform['volumes']
                        delete this.json.platform['volumeClaimTemplates']
                        delete this.json.platform.ingress
                        delete this.json.platform.runtimeClassName
                    } catch { }
                } else if (this.form.type == 'docker') {

                    if (this.json.platform?.['container-v2']?.[0]) {
                        this.json.platform['container-v2'][0].shells = this.form.shell.filter(i => i.type && i.shell);
                    }


                    this.json.platform.ingress = (this.form.ingress || []).map(i => ({
                        name: i.name,
                        routes: i.routes.filter(r => {
                            if (r.backend.port) r.backend.port = Number(r.backend.port);
                            return r.path && r.backend?.port;
                        }),
                    }));
                }

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
                    this.form.startParams.push({
                        name: 'DOMAIN_URL',
                        values_text: '%DOMAIN_SSL_URL%',
                        title: '域名',
                        required: true,
                        module_name: '',
                    })
                }

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
                j.platform.tradition = {
                    environmentName: this.form.environmentName,
                    environmentVersion: this.form.environmentVersion,
                    environmentLanguage: environmentLanguage,
                    cmd: this.form.cmd,
                }
            } else {
                delete j.platform.tradition;
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
                if (this.form.type == 'docker') {

                    this.json.platform.ingress = (this.form.ingress || []).map(i => ({
                        name: i.name,
                        routes: i.routes.filter(r => {
                            if (r.backend.port) { r.backend.port = Number(r.backend.port); }
                            return r.path && r.backend?.port;
                        }),
                    }));
                }
            }
            this.setYaml();
        },
        setYaml() {
            this.yaml = jsyaml.dump(this.json, {
                indent: 4,
                sortKeys: (a, b) => {
                    if (b == 'menu') { return -1; }
                    return a > b ? 1 : -1;
                },
            });
            this.yamlDom = `<pre class='pre'><code class='language-yaml'>${this.yaml}</code></pre>`;
            this.$nextTick(() => {
                window.hljs.highlightAll();
                this.download();
            });
        },
        getStart() {
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
            this.json.platform.startParams = start;
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
            this.$message.success("复制成功")
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

.box>>>.pre {
    margin: 0;
    height: 100%;
    font-size: 16px;
    max-width: 100%;
    overflow: auto;
    background: #282c34;
}

.box>>>input::-webkit-outer-spin-button,
.box>>>input::-webkit-inner-spin-button {
    -webkit-appearance: none;
}

.box>>>input[type="number"] {
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

div>>>pre .hljs {
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
.menulocation .el-radio {
    height: 20px;
}

.menulocation .el-radio__label {
    padding-left: 4px;
    font-size: 12px;
}

.support-group .el-checkbox {
    height: 18px;
    width: 120px;
    margin-right: 20px;
    margin-bottom: 10px;
}

.manifest-form .el-form-item__label {
    color: rgba(0, 0, 0, 0.9);
}

.envdialog .el-dialog__body {
    padding-top: 0;
}

.container-drawer .el-drawer {
    width: 1000px !important;
}

.container-drawer .el-drawer__footer {
    padding: 16px;
    border-top: 1px solid #E7E7E7;
}

.container-drawer .el-drawer__header {
    padding: 16px;
    margin-bottom: 0;
    border-bottom: 1px solid #E7E7E7;
}
</style>
