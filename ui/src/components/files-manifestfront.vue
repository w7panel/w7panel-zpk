<template>
    <div class="df content">
        <div class="fc" style="padding-left:0; overflow:auto;">
            <div v-if="noRole && !isCreate">
                <el-empty :image-size="200" description="">
                    <template #description>
                        <span class="c-99">暂无数据，点击</span>
                        <span class="cursor c-blue" @click="isCreate = true;">创建应用配置</span>
                    </template>
                </el-empty>
            </div>
            <div v-else>
                <el-form ref="formref" :model="form" label-width="100px" :rules="rules" label-position="left"
                    class="form manifest-form">
                    <div class="df jc-e">
                        <el-button v-if="!showYaml" type="primary" @click="showYaml = true;">预览yaml</el-button>
                    </div>
                    <div class="bg-white com-line df" style="margin-bottom:20px;">
                        <div class="fc">
                            <div class="c-00-6 df ai-c">基础配置</div>
                            <el-form-item class="mt-16" label="名称">
                                <el-input v-model="form.name" size="large" :disabled="!noPlatform" style="width:500px;"
                                    @change="changeForm" placeholder="请输入"></el-input>
                            </el-form-item>
                            <el-form-item label="标识">
                                <div class="df jc-b" style="width:500px;">
                                    <w7-identifie v-model:author="form.author" v-model:identifie="form.identifie"
                                        @change="onChange" disabled />
                                </div>
                            </el-form-item>
                            <el-form-item label="描述">
                                <div class="df df-c">
                                    <el-input v-model="form.description" :disabled="!noPlatform" size="large"
                                        style="width:500px;" placeholder="请输入应用描述" @change="changeForm"></el-input>
                                </div>
                            </el-form-item>
                        </div>
                    </div>

                    <div class="bg-white">
                        <div>
                            <el-form-item v-if="form.type != 'tradition'" label="前端配置">
                                <div class="f1">

                                    <files-upload @success="webUploadSuccess">
                                        <div v-if="web.name" class="upfilebox df df-c ai-c jc-c">
                                            <img src="@/assets/img/zip.png" alt=""
                                                style="width:60px;height:60px;display:block;" />
                                            <div class="df ai-c mt-20">
                                                <el-icon :size="18" class="c-green"
                                                    style="margin-right:6px;vertical-align:middle;">
                                                    <CircleCheckFilled />
                                                </el-icon>
                                                <div class="fs-14 c-33"
                                                    style="vertical-align:middle;max-width:200px;overflow:hidden;text-overflow:ellipsis;">
                                                    {{ web.name }}</div>
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

                                    <div>
                                        <div class="df ai-c mt-16">
                                            <el-tabs v-model="form.menu_type" class="fc"
                                                @tab-click="() => { roleEdit.index = -1 }">
                                                <el-tab-pane label="微擎云端控制台" name="console"></el-tab-pane>
                                                <el-tab-pane label="微擎面板控制台" name="thirdparty_cd"></el-tab-pane>
                                            </el-tabs>
                                        </div>

                                        <div v-show="form.menu_type == 'console'" class="df ai-c mt-10">
                                            <div style="">
                                                <el-checkbox v-model="form.role_founder" label="创始人端" />
                                                <el-checkbox v-model="form.role_super" label="超级管理端" />
                                            </div>
                                        </div>
                                        <div v-show="form.menu_type == 'thirdparty_cd'" class="df ai-c mt-10">
                                            <div style="">
                                                <el-checkbox v-model="cdrole.founder" label="创始人" />
                                                <el-checkbox v-model="cdrole.super" label="管理员" />
                                                <el-checkbox v-model="cdrole.tech" label="技术人员" />
                                                <el-checkbox v-model="cdrole.normal" label="普通用户" />
                                            </div>
                                        </div>
                                        <div v-for="(r, rindex) in form.role.filter(i => i.support == form.menu_type)"
                                            :key="rindex" class="role" style="margin:16px 0 30px 0;">
                                            <div class="df ai-c jc-b">
                                                <div class="df ai-c">
                                                    <div v-if="roleEdit.index == rindex" class="df">
                                                        <span>名称：</span>
                                                        <el-input v-model="roleEdit.title" style="width:140px;"
                                                            placeholder="请输入名称"></el-input>
                                                        <span class="ml-20">标识：</span>
                                                        <el-input v-model="roleEdit.name" style="width:140px;"
                                                            :disabled="r.name == 'founder' || r.name == 'super'"
                                                            placeholder="请输入标识"></el-input>
                                                    </div>
                                                    <div v-else-if="r.support === 'thirdparty_cd'"
                                                        class="df ai-c">
                                                        <span class="lh-1 mr-20">{{ r.title }}</span>
                                                        <el-checkbox v-model="r.load_mode" true-label="iframe"
                                                        false-label="static_hosting" @change="changeLoadMode(r)">
                                                        <span class="c-66">支持iframe</span></el-checkbox>
                                                    </div>
                                                    <div v-else
                                                        class="df ai-c cursor">
                                                        <div @click="roleEdit = { index: rindex, title: r.title, name: r.name }" class="mr-20">
                                                            <span class="lh-1">{{ r.title }}</span>
                                                            <el-icon v-if="r.support != 'thirdparty_cd'" color="#333333"
                                                                :size="14" style="margin-left:4px;">
                                                                <Edit />
                                                            </el-icon>
                                                        </div>
                                                        <el-checkbox v-model="r.load_mode" true-label="iframe"
                                                        false-label="static_hosting" @change="changeLoadMode(r)"><span
                                                            class="c-66">支持iframe</span></el-checkbox>
                                                    </div>

                                                    <div v-if="roleEdit.index == rindex"
                                                        class="ml-40 c-blue cursor lh-1" style="text-wrap: nowrap;"
                                                        @click="submitRoleEdit">确定</div>
                                                    <div v-if="roleEdit.index == rindex"
                                                        class="ml-20 c-blue cursor lh-1" style="text-wrap:nowrap;"
                                                        @click="deleteRoleEdit(rindex)">删除
                                                    </div>
                                                </div>
                                                <div class="df ai-c">
                                                    <el-popover placement="top" :width="240" trigger="hover">
                                                        <template #reference>
                                                            <div class="df ai-c cursor" style="margin-right:30px;">
                                                                <img src="@/assets/img/micon.png" alt=""
                                                                    style="width:20px;margin-right:5px;" />
                                                                <span class="c-66 lh-1">菜单布局</span>
                                                            </div>
                                                        </template>
                                                        <el-radio-group v-model="r.location" class="df"
                                                            @change="getMenu">
                                                            <div class="fc df df-c ai-c menulocation cursor"
                                                                @click="r.location = 'top'; getMenu()">
                                                                <img src="@/assets/img/menu-t.png" alt="" />
                                                                <el-radio label="top" class="mt-10">顶部菜单布局</el-radio>
                                                            </div>
                                                            <div class="fc df df-c ai-c menulocation cursor"
                                                                @click="r.location = 'left'; getMenu()">
                                                                <img src="@/assets/img/menu-l.png" alt="" />
                                                                <el-radio label="left" class="mt-10">左侧菜单布局</el-radio>
                                                            </div>
                                                        </el-radio-group>
                                                    </el-popover>
                                                    <el-checkbox v-if="r.name != 'founder' && r.name != 'super'"
                                                        v-model="r.is_default_register" :true-label="2" :false-label="1"
                                                        @change="chengeRegister(r, r.is_default_register)">默认邀请端</el-checkbox>
                                                </div>
                                            </div>

                                            <div class="mt-10 greybox">
                                                <template v-if="r.load_mode === 'iframe'">
                                                    <div class="greybox-title">iframe配置<el-tooltip><template #content>iframe使用场景受到了严格的限制，如果需要对接授权登录，可将 {access_token} 传递给iframe，然后由后端服务请求授权接口地址（http://xxxx）获取用户信息。由于iframe受到了浏览器安全限制，生成cookies时必须设置 SameSite: None, Secure: true，并且header设置允许 * 跨域，才能正常传递。</template><ArcoIcon name="icon-41" :size="16"/></el-tooltip></div>
                                                    <el-form-item label="地址类型" style="margin-bottom:20px;">
                                                        <el-radio-group v-model="r.type" @change="changeBackendType(r)">
                                                            <el-radio label="internal" value="internal">应用地址</el-radio>
                                                            <el-radio label="external" value="external">远程地址</el-radio>
                                                        </el-radio-group>
                                                    </el-form-item>
                                                    <el-form-item label="页面地址" style="margin-bottom:20px;">
                                                        <div v-if="r.type == 'internal'" class="backend-url-config df ai-c">
                                                            <span class="backend-url-fixed">https://</span>
                                                            <span class="backend-url-fixed backend-url-placeholder">{{ getIframeDomainPlaceholder() }}</span>
                                                            <span class="backend-url-fixed">/</span>
                                                            <el-input v-model="r.backend_path" @input="getMenu" @change="getMenu" placeholder="请输入目录"
                                                                class="backend-url-control backend-url-input" />
                                                        </div>
                                                        <div v-else class="backend-url-config backend-url-config-external df ai-c">
                                                            <el-select v-model="r.root_protocol"
                                                                class="backend-url-control backend-url-protocol"
                                                                @change="getMenu">
                                                                <el-option label="http://" value="http://"></el-option>
                                                                <el-option label="https://" value="https://"></el-option>
                                                            </el-select>
                                                            <el-input v-model="r.root_url" @change="getMenu"
                                                                placeholder="请输入地址"
                                                                class="backend-url-control backend-url-input" />
                                                        </div>
                                                    </el-form-item>
                                                    <el-form-item label="变量传递" style="margin-bottom:20px;">
                                                        只支持query方式，get方式拼接到页面地址后面
                                                    </el-form-item>
                                                </template>
                                                <template v-else>

                                                <div class="greybox-title">变量传递配置<el-tooltip><template #content>将开发者设置的变量值传递给后端接口和前端JS变量中</template><ArcoIcon name="icon-41" :size="16"/></el-tooltip></div>
                                                <el-form-item label="接口类型" style="margin-bottom:20px;">
                                                    <el-radio-group v-model="r.type" @change="changeBackendType(r)">
                                                        <el-radio label="internal" value="internal">应用内网地址（internal）</el-radio>
                                                        <el-radio label="external" value="external">应用外网地址（external）</el-radio>
                                                    </el-radio-group>
                                                </el-form-item>
                                                <el-form-item label="接口地址" style="margin-bottom:20px;">
                                                    <div v-if="r.type == 'internal'" class="backend-url-config df ai-c">
                                                        <span class="backend-url-fixed">http://</span>
                                                        <el-select v-model="r.backend_identifie" filterable
                                                            default-first-option placeholder="选择应用标识"
                                                            class="backend-url-control backend-url-identifie"
                                                            @change="changeBackendIdentifie(r)">
                                                            <el-option v-for="app in backendAppOptions" :key="app.id"
                                                                :label="app.title && app.title != app.id ? `${app.id}（${app.title}）` : app.id"
                                                                :value="app.id"></el-option>
                                                        </el-select>
                                                        <span
                                                            class="backend-url-fixed">.default.svc.cluster.local:</span>
                                                        <el-autocomplete v-model="r.backend_port"
                                                            :fetch-suggestions="(query, cb) => queryBackendPortSuggestions(r.backend_identifie, query, cb)"
                                                            placeholder="端口"
                                                            class="backend-url-control backend-url-port" @input="getMenu"
                                                            @change="getMenu" @select="getMenu"></el-autocomplete>
                                                    </div>
                                                    <div v-else class="backend-url-config backend-url-config-external df ai-c">
                                                        <el-select v-model="r.root_protocol"
                                                            class="backend-url-control backend-url-protocol"
                                                            @change="getMenu">
                                                            <el-option label="http://" value="http://"></el-option>
                                                            <el-option label="https://" value="https://"></el-option>
                                                        </el-select>
                                                        <el-input v-model="r.root_url" @change="getMenu"
                                                            placeholder="请输入地址"
                                                            class="backend-url-control backend-url-input" />
                                                    </div>
                                                </el-form-item>

                                                <div class="df ai-c">代理配置<el-tooltip><template #content>面板提供转发服务到接口地址，接口后端可通过HTTP变量获取传递值</template><ArcoIcon name="icon-41" :size="16"/></el-tooltip></div>
                                                <div class="mb-20">
                                                    <div style="margin-bottom:20px;" class="df">
                                                        <div style="width:100px;">代理地址</div>
                                                        <div>/panel-api/v1/microapp/{{identifie}}/proxy</div>
                                                    </div>
                                                    <div class="mb-20">
                                                        <div>请求头(Header)</div>
                                                        <table class="table">
                                                            <thead>
                                                                <tr>
                                                                    <td>key</td>
                                                                    <td>value</td>
                                                                    <td>操作</td>
                                                                </tr>
                                                            </thead>
                                                            <tbody>
                                                                <tr v-for="(item, index) in r.proxy_request_header" :key="index">
                                                                    <td>
                                                                        <el-input v-model="item.key" placeholder="key"
                                                                            @change="getMenu"
                                                                            style="width:200px;margin-right:10px;"></el-input>
                                                                    </td>
                                                                    <td>
                                                                        <el-autocomplete v-model="item.value"
                                                                            :fetch-suggestions="(query) => systemVar.filter(i=>i.includes(query)).map(i=>({value:i}))"
                                                                            placeholder="value"
                                                                            class="backend-url-control backend-url-port" @input="getMenu"
                                                                            @change="getMenu" @select="getMenu"></el-autocomplete>
                                                                    </td>
                                                                    <td><span class="c-blue cursor handle"
                                                                            @click="r.proxy_request_header.length <= 1 ? r.proxy_request_header = [{ key: '', value: '', isSelect: false }] : r.proxy_request_header.splice(index, 1)">删除</span></td>
                                                                </tr>
                                                                <tr>
                                                                    <td colspan="5" class="cursor txt-c"
                                                                        @click="r.proxy_request_header.push({ key: '', value: '' })">
                                                                        <span class="addmenu"><el-icon :size="14">
                                                                                <Plus />
                                                                            </el-icon>添加请求头</span>
                                                                    </td>
                                                                </tr>
                                                            </tbody>
                                                        </table>
                                                    </div>
                                                    <div v-if="!(form.menu_type == 'thirdparty_cd' && r.name == 'normal')">
                                                        <div >请求参数(Query)</div>
                                                        <table class="table">
                                                            <thead>
                                                                <tr>
                                                                    <td>key</td>
                                                                    <td>value</td>
                                                                    <td>操作</td>
                                                                </tr>
                                                            </thead>
                                                            <tbody>
                                                                <tr v-for="(item, index) in r.proxy_request_query" :key="index">
                                                                    <td>
                                                                        <el-input v-model="item.key" placeholder="key"
                                                                            @change="getMenu"
                                                                            style="width:200px;margin-right:10px;"></el-input>
                                                                    </td>
                                                                    <td>
                                                                        <el-autocomplete v-model="item.value"
                                                                            :fetch-suggestions="(query) => systemVar.filter(i=>i.includes(query)).map(i=>({value:i}))"
                                                                            placeholder="value"
                                                                            class="backend-url-control backend-url-port" @input="getMenu"
                                                                            @change="getMenu" @select="getMenu"></el-autocomplete>
                                                                    </td>
                                                                    <td><span class="c-blue cursor handle"
                                                                            @click="r.proxy_request_query.length <= 1 ? r.proxy_request_query = [{ key: '', value: '', isSelect: false }] : r.proxy_request_query.splice(index, 1)">删除</span></td>
                                                                </tr>
                                                                <tr>
                                                                    <td colspan="5" class="cursor txt-c"
                                                                        @click="r.proxy_request_query.push({ key: '', value: '' })">
                                                                        <span class="addmenu"><el-icon :size="14">
                                                                                <Plus />
                                                                            </el-icon>添加请求参数</span>
                                                                    </td>
                                                                </tr>
                                                            </tbody>
                                                        </table>
                                                    </div>
                                                </div>
                                                <div class="df ai-c">前端配置<el-tooltip><template #content>面板提供microapp机制渲染前端包，可通过window.$wujie.props.frontend_props
从JS变量获取传递值</template><ArcoIcon name="icon-41" :size="16"/></el-tooltip></div>
                                                <div
                                                    v-if="!(form.menu_type == 'thirdparty_cd' && r.name == 'normal')">
                                                    <div >前端配置</div>
                                                    <table class="table">
                                                        <thead>
                                                            <tr>
                                                                <td>key</td>
                                                                <td>value</td>
                                                                <td>操作</td>
                                                            </tr>
                                                        </thead>
                                                        <tbody>
                                                            <!-- ["${system.group}", "${system.userid}", "${system.openid}", "${system.nickname}", "${system.role}", "${system.access_token}", "${system.group}", "${system.url}"] -->
                                                            <tr v-for='item in ["${system.userid}", "${system.openid}", "${system.nickname}", "${system.role}", "${system.access_token}", "${system.group}", "${system.url}"]' :key="item">
                                                                <td>
                                                                    {{ item.match(/\${system.(.*)}/)[1] }}
                                                                </td>
                                                                <td>
                                                                    {{ item }}
                                                                </td>
                                                                <td></td>
                                                            </tr>
                                                            
                                                            <tr v-for="(item, index) in r.frontend_props" :key="index">
                                                                <td>
                                                                    <el-input v-model="item.key" placeholder="key"
                                                                        @change="getMenu"
                                                                        style="width:200px;margin-right:10px;"></el-input>
                                                                </td>
                                                                <td>
                                                                    <el-autocomplete v-model="item.value"
                                                                        :fetch-suggestions="(query) => systemVar.filter(i=>i.includes(query)).map(i=>({value:i}))"
                                                                        placeholder="value"
                                                                        class="backend-url-control backend-url-port" @input="getMenu"
                                                                        @change="getMenu" @select="getMenu"></el-autocomplete>
                                                                </td>
                                                                <td><span class="c-blue cursor handle"
                                                                        @click="r.frontend_props.length <= 1 ? r.frontend_props = [{ key: '', value: '', isSelect: false }] : r.frontend_props.splice(index, 1)">删除</span></td>
                                                            </tr>
                                                            <tr>
                                                                <td colspan="5" class="cursor txt-c"
                                                                    @click="r.frontend_props.push({ key: '', value: '' })">
                                                                    <span class="addmenu"><el-icon :size="14">
                                                                            <Plus />
                                                                        </el-icon>添加前端配置</span>
                                                                </td>
                                                            </tr>
                                                        </tbody>
                                                    </table>
                                                </div>
                                                </template>

                                            </div>

                                            <table v-if="r.load_mode !== 'iframe'" class="menutable table mt-10">
                                                <thead>
                                                    <tr>
                                                        <td>排序</td>
                                                        <td>路由</td>
                                                        <td>名称</td>
                                                        <td>
                                                            <div class="df ai-c jc-c">
                                                                <span>欢迎页</span>
                                                                <el-popover placement="top" :width="200" trigger="hover"
                                                                    content="用户进入系统首页访问的页面">
                                                                    <template #reference>
                                                                        <el-icon color="#999" :size="16"
                                                                            class="cursor ml-4">
                                                                            <QuestionFilled />
                                                                        </el-icon>
                                                                    </template>
                                                                </el-popover>
                                                            </div>
                                                        </td>
                                                        <td>图标</td>
                                                        <td>操作</td>
                                                    </tr>
                                                </thead>
                                                <tbody>
                                                    <tr v-for="(item, index) in r.menu" :key="index">
                                                        <td>
                                                            <div><el-input type="number" v-model="item.displayorder"
                                                                    @change="getMenu"
                                                                    style="width:60px; height:36px;"></el-input>
                                                            </div>
                                                            <div v-for="(sub, subid) in item.children" :key="subid"
                                                                class="df mt-10">
                                                                <div class="branch"
                                                                    :class="{ last: subid == item.children.length - 1 }">
                                                                </div>
                                                                <el-input type="number" v-model="sub.displayorder"
                                                                    @change="getMenu"
                                                                    style="width:60px; height:36px;"></el-input>
                                                            </div>
                                                        </td>
                                                        <td>
                                                            <div><el-input v-model="item.do" @change="onMenuChanged(r.menu)"
                                                                    style="width:150px; height:36px;"
                                                                    placeholder="路由"></el-input>
                                                            </div>
                                                            <div v-for="(sub, subid) in item.children" :key="subid"
                                                                class="mt-10">
                                                                <el-input v-model="sub.do" @change="onMenuChanged(r.menu)"
                                                                    style="width:150px; height:36px;"
                                                                    placeholder="路由"></el-input>
                                                            </div>
                                                        </td>
                                                        <td>
                                                            <div><el-input v-model="item.title" @change="onMenuChanged(r.menu)"
                                                                    style="width:150px; height:36px;"
                                                                    placeholder="名称"></el-input>
                                                            </div>
                                                            <div v-for="(sub, subid) in item.children" :key="subid"
                                                                class="mt-10">
                                                                <el-input v-model="sub.title" @change="onMenuChanged(r.menu)"
                                                                    style="width:150px; height:36px;"
                                                                    placeholder="名称"></el-input>
                                                            </div>
                                                        </td>
                                                        <td>
                                                            <div style="height:36px; text-align:center;" :style="{visibility: item.children?.length > 0 ? 'hidden' : 'visible'}">
                                                                <el-checkbox :model-value="Number(item.is_default) === 1" size="large"
                                                                    :name="r.name"
                                                                    @click.stop="setMenuDefault(r.menu, item)"></el-checkbox>
                                                            </div>
                                                            <div v-for="(sub, subid) in item.children" :key="subid"
                                                                class="mt-10 df ai-c jc-c" style="height:36px;">
                                                                <el-checkbox :model-value="Number(sub.is_default) === 1" size="large"
                                                                    :name="r.name"
                                                                    @click.stop="setMenuDefault(r.menu, sub)"></el-checkbox>
                                                            </div>
                                                        </td>
                                                        <td>
                                                            <div class="selicon cursor df ai-c jc-c"
                                                                v-if="item.icon_svg"
                                                                @click="dialogVisible = true; activeItem = item;"
                                                                v-html="elementsToSvg(item.icon_svg)"></div>
                                                            <div class="selicon cursor df ai-c jc-c"
                                                                v-else-if="item.icon"
                                                                @click="dialogVisible = true; activeItem = item;"><i
                                                                    class="fs-24 wi" :class="'wi-' + item.icon"></i>
                                                            </div>
                                                            <div class="selicon cursor df ai-c jc-c" v-else
                                                                @click="dialogVisible = true; activeItem = item;">
                                                                <el-icon :size="24">
                                                                    <Menu />
                                                                </el-icon>
                                                            </div>
                                                            <div v-for="(sub, subid) in item.children" :key="subid"
                                                                class="df ai-c jc-c mt-10"
                                                                style="width:36px; height:36px;">
                                                            </div>
                                                        </td>
                                                        <td>
                                                            <div class="df ai-c" style="height:36px;">
                                                                <span class="handle c-blue cursor"
                                                                    @click="addSub(r.menu, item)">添加子菜单</span>
                                                                <el-popover placement="top" :width="240"
                                                                    trigger="hover">
                                                                    <template #reference><span
                                                                            class="handle c-blue cursor">设置位置</span></template>
                                                                    <div>
                                                                        <div class="df ai-c jc-b">
                                                                            <div class="menu-single-location">单个菜单位置设置
                                                                            </div>
                                                                        </div>
                                                                        <el-radio-group v-model="item.location"
                                                                            class="df mt-10" @change="getMenu">
                                                                            <div class="fc df df-c ai-c menulocation cursor"
                                                                                @click="item.location = 'normal'; getMenu()">
                                                                                <img v-if="r.location == 'top'"
                                                                                    src="@/assets/img/menu-t.png"
                                                                                    alt="" />
                                                                                <img v-else
                                                                                    src="@/assets/img/menu-l.png"
                                                                                    alt="" />
                                                                                <el-radio label="normal"
                                                                                    class="mt-10">默认位置</el-radio>
                                                                            </div>
                                                                            <div v-if="r.location == 'top'"
                                                                                class="fc df df-c ai-c menulocation cursor"
                                                                                @click="item.location = 'back'; getMenu()">
                                                                                <img src="@/assets/img/menu-r.png"
                                                                                    alt="" />
                                                                                <el-radio label="back"
                                                                                    class="mt-10">顶部右侧</el-radio>
                                                                            </div>
                                                                            <div v-else
                                                                                class="fc df df-c ai-c menulocation cursor"
                                                                                @click="item.location = 'back'; getMenu()">
                                                                                <img src="@/assets/img/menu-b.png"
                                                                                    alt="" />
                                                                                <el-radio label="back"
                                                                                    class="mt-10">左侧底部</el-radio>
                                                                            </div>
                                                                        </el-radio-group>
                                                                    </div>
                                                                </el-popover>
                                                                <span class="handle c-blue cursor"
                                                                    @click="removeMenu(r.menu, index)">删除</span>
                                                            </div>
                                                            <div v-for="(sub, subid) in item.children" :key="subid"
                                                                class="mt-10 df ai-c" style="height:36px;">
                                                                <span class="handle c-blue cursor"
                                                                    @click="removeSubMenu(r.menu, item, subid)">删除</span>
                                                            </div>
                                                        </td>
                                                    </tr>
                                                    <tr>
                                                        <td colspan="9" class="cursor txt-c"
                                                            @click="addMenu(r.menu)">
                                                            <span class="addmenu"><el-icon :size="14">
                                                                    <Plus />
                                                                </el-icon>添加一级菜单</span>
                                                        </td>
                                                    </tr>
                                                </tbody>
                                            </table>
                                        </div>
                                        <div v-if="form.menu_type == 'console'"
                                            class="mt-10 addrole df ai-c jc-c cursor" @click="showAddRole = true">添加管理端
                                        </div>
                                        <div class="mt-16 ml-20 c-red fs-12">注：上传后端代码包的前提下，如果未添加管理端配置，默认类型为系统组件，否则为原生应用
                                        </div>
                                    </div>
                                </div>
                            </el-form-item>
                        </div>
                        <span v-if="form.type == 'tradition'">传统应用</span>
                    </div>
                    <div class="bg-white pb-24 mt-20 df ai-c">
                        <el-button :loading="submiting" type="primary" @click="submit()"
                            style="width:90px;margin-left:100px;">确定提交</el-button>
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
        <el-dialog v-model="dialogVisible" title="选择图标" width="820px" top="50px">
            <sel-svg @submit="selectIcon"></sel-svg>
        </el-dialog>
        <el-dialog v-model="showIngress" title="添加业务端" width="640px">
            <el-form ref="ingress" :model="newIngressEnd" label-width="80px">
                <el-form-item label="名称" prop="name" :rules="[{ required: true, message: '内容不能为空', trigger: 'blur' }]">
                    <el-input placeholder="请输入业务端名称" v-model="newIngressEnd.name" size="large"
                        style="width:500px;"></el-input>
                </el-form-item>
                <el-form-item label="" class="mt-20">
                    <el-button @click="addIngressEnd" type="primary" size="large">确定</el-button>
                </el-form-item>
            </el-form>
        </el-dialog>

        <el-dialog v-model="showAddRole" title="添加管理端" width="640px">
            <el-form ref="role" :model="newRole" label-width="80px">
                <el-form-item label="名称" prop="title" :rules="[{ required: true, message: '内容不能为空', trigger: 'blur' }]">
                    <el-input placeholder="请输入管理端名称" v-model="newRole.title" size="large"
                        style="width:500px;"></el-input>
                </el-form-item>
                <el-form-item label="标识" prop="name" :rules="[{ required: true, message: '内容不能为空', trigger: 'blur' }]">
                    <el-input placeholder="请输入管理端标识" v-model="newRole.name" size="large"
                        style="width:500px;"></el-input>
                </el-form-item>
                <el-form-item label="" class="mt-20">
                    <el-button @click="addRole" type="primary" size="large">确定</el-button>
                </el-form-item>
            </el-form>
        </el-dialog>
    </div>
</template>

<script>
import jsyaml from "js-yaml";
import hljs from 'highlight.js';
import filesUpload from './files-upload.vue';
import selSvg from '@/components/sel-svg.vue';
import W7Identifie from '@/components/w7-identifie.vue';
import ArcoIcon from '@/components/arco-icon.vue';


export default {
    emits: ['writefile'],
    props: [
        'data',
        'submiting',
        'option',
        'identifie',
        'version_id',
        'app_ports'
    ],
    components: {
        filesUpload,
        selSvg,
        W7Identifie,
        ArcoIcon
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

                front_type: ['thirdparty_cd'],
                menu_type: 'console',
                type: 'docker',
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

            cdrole: {
                founder: false,
                super: false,
                tech: false,
                normal: false,
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
                port: [{ required: true, message: '内容不能为空', trigger: 'blur' }],
                cpu: [{ required: true, message: '内容不能为空', trigger: 'blur' }],
                mem: [{ required: true, message: '内容不能为空', trigger: 'blur' }],
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
        'cdrole.founder'(v) {
            this.switchRole(v, 'founder', '创始人', 'thirdparty_cd');
        },
        'cdrole.super'(v) {
            this.switchRole(v, 'super', '管理员', 'thirdparty_cd');
        },
        'cdrole.tech'(v) {
            this.switchRole(v, 'tech', '技术人员', 'thirdparty_cd');
        },
        'cdrole.normal'(v) {
            this.switchRole(v, 'normal', '普通用户', 'thirdparty_cd');
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
        systemVar() {
            return (this.startParams || []).map(item => `{{.Values.${item.name}}}`)
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
                    ports: ports.length ? ports : (old.ports || []),
                });
            };

            addApp({
                id: this.currentBackendIdentifie,
                title: this.form.name || this.currentBackendIdentifie,
                ports: this.form.port?.map?.(i => i.port) || [],
            });
            (this.app_ports || []).forEach(addApp);
            (this.option?.app_ports || []).forEach(addApp);

            return [...apps.values()];
        },
    },
    methods: {
        normalizeBackendPorts(ports) {
            if (!Array.isArray(ports)) { ports = ports ? [ports] : [] }
            return [...new Set(ports.map(i => {
                if (i && typeof i == 'object') {
                    return i.port ?? i.containerPort ?? '';
                }
                return i;
            }).filter(i => i !== '' && i !== undefined && i !== null).map(i => String(i)))];
        },
        getBackendPorts(identifie) {
            return this.backendAppOptions.find(i => i.id == identifie)?.ports || [];
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
            return this.getBackendPorts(identifie)[0] || '';
        },
        changeBackendIdentifie(role) {
            role.backend_port = this.getDefaultBackendPort(role.backend_identifie);
            this.getMenu();
        },
        changeBackendType(role) {
            if (role.load_mode == 'iframe') {
                this.syncIframeBackendDefaults(role);
                this.getMenu();
                return;
            }
            if (role.type == 'internal') {
                if (!role.backend_identifie) {
                    role.backend_identifie = this.getDefaultBackendIdentifie();
                }
                if (!role.backend_port) {
                    role.backend_port = this.getDefaultBackendPort(role.backend_identifie);
                }
            } else {
                role.root_protocol = role.root_protocol || 'http://';
            }
            this.getMenu();
        },
        changeLoadMode(role) {
            if (role.load_mode == 'iframe') {
                role.type = role.type || 'internal';
                this.syncIframeBackendDefaults(role);
            } else {
                if (role.type == 'internal' && role.backend_identifie == this.getIframeDomainPlaceholder()) {
                    role.backend_identifie = this.getDefaultBackendIdentifie();
                    role.backend_port = this.getDefaultBackendPort(role.backend_identifie);
                }
            }
            this.getMenu();
        },
        syncRoleBackendDefaults() {
            let changed = false;
            this.form.role.forEach(role => {
                if (role.load_mode == 'iframe') {
                    changed = this.syncIframeBackendDefaults(role) || changed;
                    return;
                }
                if (role.type != 'internal') { return }
                if (!role.backend_identifie) {
                    role.backend_identifie = this.getDefaultBackendIdentifie();
                    changed = true;
                }
                if (!role.backend_port) {
                    role.backend_port = this.getDefaultBackendPort(role.backend_identifie);
                    changed = true;
                }
            });
            return changed;
        },
        getIframeDomainPlaceholder() {
            return '{{.Values.DOMAIN_URL}}';
        },
        syncIframeBackendDefaults(role) {
            let changed = false;
            let placeholder = this.getIframeDomainPlaceholder();
            if (role.type == 'internal') {
                if (role.backend_identifie != placeholder) {
                    role.backend_identifie = placeholder;
                    changed = true;
                }
                if (role.backend_path === undefined || role.backend_path === null) {
                    role.backend_path = '';
                    changed = true;
                }
            } else {
                if (!role.root_protocol) {
                    role.root_protocol = 'http://';
                    changed = true;
                }
                if (role.root_url === undefined || role.root_url === null) {
                    role.root_url = '';
                    changed = true;
                }
            }
            return changed;
        },
        parseIframeBackendUrl(url) {
            let value = String(url || '');
            let placeholder = this.getIframeDomainPlaceholder();
            if (value.includes(placeholder)) {
                let path = value.slice(value.indexOf(placeholder) + placeholder.length).replace(/^\/+/, '');
                return {
                    type: 'internal',
                    backend_identifie: placeholder,
                    backend_path: path,
                    root_protocol: 'http://',
                };
            }
            let externalBackend = this.parseExternalBackendUrl(value);
            return {
                type: 'external',
                backend_identifie: placeholder,
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
            return this.getExternalBackendUrl(role);
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
            if (port === '' || port === undefined || port === null) { return '' }
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
            let sub = { title: '', displayorder: 0, is_default: 0 };
            item.children.push(sub);
            this.menuDefaultTarget = sub;
            this.normalizeMenuDefault(menu, sub);
            this.getMenu();
        },
        selectIcon(item) {
            this.activeItem.icon_svg = item.json;
            this.dialogVisible = false;
            this.getMenu();
        },
        addRole() {
            this.$refs.role.validate((valid) => {
                if (!valid) { return }
                let backend_identifie = this.getDefaultBackendIdentifie();
                this.form.role.push({
                    title: this.newRole.title,
                    name: this.newRole.name,
                    support: this.form.menu_type,
                    status: 1,
                    load_mode: 'static_hosting',
                    is_default_register: 1,
                    location: 'left',
                    menu: [],

                    type: 'internal',
                    backend_identifie: backend_identifie,
                    backend_port: this.getDefaultBackendPort(backend_identifie),
                    backend_path: '',
                    root_protocol: 'http://',
                    root_url: '',

                    proxy_request_header: [{ key: '', value: '' }],
                    proxy_request_query: [{ key: '', value: '' }],

                    frontend_props: [{ key: '', value: '' }],
                })
                this.getMenu();
                this.showAddRole = false;
                this.newRole = { title: '', name: '' };
            });
        },
        getMenu() {
            let role = [];
            let front_type = new Set();
            let consoleRole = {
                founder: false,
                super: false,
            }
            let cdRole = {
                founder: false,
                super: false,
                tech: false,
                normal: false,
            }
            this.form.role.forEach(r => {

                if (r.type == 'external') {
                    try {
                        r.frontend_props?.map(i => i.isSelect = false)
                    } catch { }
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

                if (['super', 'founder', 'tech', 'normal'].includes(r.name)) {
                    if (r.support == 'console' && ['super', 'founder'].includes(r.name)) {
                        consoleRole[r.name] = true;
                    }
                    if (r.support == 'thirdparty_cd') {
                        cdRole[r.name] = true;
                    }
                }

                let proxy_request_header = Object.fromEntries(r.proxy_request_header.filter(i => i.key && i.value).map(({ key, value, isSelect }) => {
                    value = isSelect ? `"{{.Values.${value}}}"` : value;
                    return [key, value]
                }))
                let proxy_request_query = Object.fromEntries(r.proxy_request_query.filter(i => i.key && i.value).map(({ key, value, isSelect }) => {
                    value = isSelect ? `"{{.Values.${value}}}"` : value;
                    return [key, value]
                }))
                let frontend_props = Object.fromEntries(r.frontend_props.filter(i => i.key && i.value).map(({ key, value, isSelect }) => {
                    value = isSelect ? `"{{.Values.${value}}}"` : value;
                    return [key, value]
                }))

                if (r.load_mode == 'iframe') {
                    this.syncIframeBackendDefaults(r);
                    itemObj.backend_config = {
                        type: r.type,
                        backend_identifie: this.getIframeBackendUrl(r),
                        ...((r.support != 'thirdparty_cd' || r.name != 'normal') ? {
                            frontend_props: frontend_props
                        } : {}),
                    };
                } else {
                    itemObj.backend_config = {
                        type: r.type,
                        ...(r.type == 'internal' ? {
                            backend_identifie: r.backend_identifie,
                            backend_port: this.formatBackendPort(r.backend_port),
                            proxy_request: {
                                headers: proxy_request_header,
                                query: proxy_request_query,
                            },
                        } : {

                            backend_identifie: this.getExternalBackendUrl(r),
                        }),
                        ...((r.support != 'thirdparty_cd' || r.name != 'normal') ? {
                            frontend_props: frontend_props
                        } : {}),
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

                if (itemObj.menu.length > 0 || r.load_mode == 'iframe') {
                    if (r.support) {
                        front_type.add(r.support);
                    }
                    role.push(itemObj);
                }
            });

            this.form.role_founder = consoleRole.founder;
            this.form.role_super = consoleRole.super;
            this.cdrole = {
                ...this.cdrole,
                ...cdRole,
            }

            this.json.application.front_type = [...front_type];
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
                let backend_identifie = this.getDefaultBackendIdentifie();
                this.form.role.push({
                    title: title,
                    name: name,
                    status: 1,
                    support: type,
                    load_mode: 'static_hosting',
                    is_default_register: 1,
                    location: 'left',
                    menu: [],

                    type: 'internal',
                    backend_identifie: backend_identifie,
                    backend_port: this.getDefaultBackendPort(backend_identifie),
                    backend_path: '',
                    root_protocol: 'http://',
                    root_url: '',

                    proxy_request_header: [{ key: '', value: '' }],
                    proxy_request_query: [{ key: '', value: '' }],

                    frontend_props: [{ key: '', value: '' }],
                });
            } else if (hasrole) {
                this.form.role.splice(roleindex, 1);
            }
            this.getMenu();
        },


        getCreateImg(v) { this.form.image = v; },

        webUploadSuccess(data, filename) {
            if (data?.url || data?.data?.url) {
                let url = data?.url || data?.data?.url;
                this.web.name = url.match(/[^\/]+$/)[0];
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
            if (j.application) {
                if (/^[^-]+-.+$/.test(j.application?.identifie)) {
                    let i = j.application.identifie;
                    j.application.author = i.match(/^([^-]+)-(.+)$/)[1];
                } else if (j.application.identifie && j.application.author) {
                    j.application.identifie = j.application.author + '-' + j.application.identifie;
                }
                this.form.type = j.application.type || 'docker';

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
            this.form.role.map((item, index) => {
                if (!item.support) {
                    item.support = 'console';
                    let thirdparty_cd = JSON.parse(JSON.stringify(item));
                    thirdparty_cd.support = 'thirdparty_cd';
                    this.form.role.push(thirdparty_cd);
                }

                if (item.load_mode == 'iframe') {
                    let iframeBackend = this.parseIframeBackendUrl(item?.backend_config?.backend_identifie || '');
                    item.type = iframeBackend.type;
                    item.backend_identifie = iframeBackend.backend_identifie;
                    item.backend_path = iframeBackend.backend_path;
                    item.root_protocol = iframeBackend.root_protocol;
                    item.root_url = iframeBackend.root_url;
                    item.backend_port = '';
                } else {
                    item.type = item?.backend_config?.type || 'internal';
                    item.backend_path = '';
                    if (item.type != 'internal') {
                        let externalBackend = this.parseExternalBackendUrl(item?.backend_config?.backend_identifie || '');
                        item.root_protocol = externalBackend.protocol;
                        item.root_url = externalBackend.url;
                        item.backend_identifie = this.getDefaultBackendIdentifie();
                        item.backend_port = this.getDefaultBackendPort(item.backend_identifie);
                    } else {
                        item.root_protocol = 'http://';
                        item.root_url = '';
                        item.backend_identifie = item?.backend_config?.backend_identifie;
                        item.backend_identifie = item.backend_identifie || this.getDefaultBackendIdentifie();
                        item.backend_port = item?.backend_config?.backend_port ?? '';
                    }
                }


                item.proxy_request_header = Object.entries(item?.backend_config?.proxy_request?.headers || {}).map(([k, v]) => {
                    let match = v.match(/^\"\{\{\s*\.Values\.([^\.]+)\s*\}\}\"$/)
                    return { key: k, value: match ? match[1] : v, isSelect: Boolean(match) }
                });
                item.proxy_request_query = Object.entries(item?.backend_config?.proxy_request?.query || {}).map(([k, v]) => {
                    let match = v.match(/^\"\{\{\s*\.Values\.([^\.]+)\s*\}\}\"$/)
                    return { key: k, value: match ? match[1] : v, isSelect: Boolean(match) }
                });
                item.proxy_request_header = item.proxy_request_header.length ? item.proxy_request_header : [{ key: '', value: '' }]
                item.proxy_request_query = item.proxy_request_query.length ? item.proxy_request_query : [{ key: '', value: '' }]

                item.frontend_props = Object.entries(item?.backend_config?.frontend_props || {}).map(([k, v]) => {
                    let match = v.match(/^\"\{\{\s*\.Values\.([^\.]+)\s*\}\}\"$/)
                    return { key: k, value: match ? match[1] : v, isSelect: Boolean(match) }
                });
                item.frontend_props = item.frontend_props.length ? item.frontend_props : [{ key: '', value: '' }];

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
                if (j.platform.container) {

                    delete j.platform.container.hook;

                    if (!j.platform.container.ports?.length) {
                        if (j.platform.container.containerPort) {
                            this.form.port = [{ name: "默认", port: j.platform.container.containerPort, protocol: 'TCP', lbPort: '' }];
                            j.platform.container.ports = this.form.port;
                        } else {
                            this.form.port = [];
                            j.platform.container.ports = [];
                        }
                    } else if (j.platform.container.ports?.length) {
                        this.form.port = JSON.parse(JSON.stringify(j.platform.container.ports));
                        this.form.port.forEach(i => {
                            if (!i.protocol) { i.protocol = 'TCP' }
                        })
                    }
                    j.platform.container.privileged = j.platform.container?.privileged == 'false' ? false : (!!j.platform.container.privileged);

                    this.form.cpu = j.platform.container.cpu || 1;
                    this.form.mem = j.platform.container.mem || 2;
                    this.form.image = j.platform.container.image || '';
                    this.form.privileged = j.platform.container.privileged || false;
                    this.form.build_context = j.platform.container.build?.context || '';

                    if (j.platform.container.language) {
                        delete j.platform.container.language;
                    }
                    let shell = j.platform.container?.shells;
                    if (typeof shell == 'object') {
                        this.form.shell = shell;
                    } else {
                        this.form.shell = [];
                    }

                    this.form.cmd = j.platform.container.cmd || [''];
                    if (!this.form.cmd.length) { this.form.cmd = ['']; }

                    if (j.platform.container.securityContext) {
                        let sc = j.platform.container.securityContext;
                        this.form.securityContext.runAsNonRoot = sc.runAsNonRoot || false;

                        this.form.securityContext.runAsUser = sc.runAsUser === undefined ? '' : sc.runAsUser;
                        this.form.securityContext.runAsGroup = sc.runAsGroup === undefined ? '' : sc.runAsGroup;

                        this.form.securityContext.fsGroup = sc.fsGroup === undefined ? '' : sc.fsGroup;
                    }

                    let env = j?.platform?.container?.env;
                    this.form.env = env?.length ? env : [];

                    if (this?.option?.lightApp) {
                        if (j?.platform?.container?.startParams) {
                            j.platform.container.startParams = [];
                            this.form.startParams = [];
                        }
                    } else {
                        let startParams = j?.platform?.startParams;
                        this.form.startParams = startParams?.length ? startParams : [];
                        this.form.mysql8 = false;
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
                                if (i.module_name == 'w7_redis') { i.mark = 'redis'; this.form.redis = true; }
                                if (i.module_name == 'w7_mongodb') { i.mark = 'mongodb6'; this.form.mongodb6 = true; }
                            })
                        }
                    }

                    let volumes = j?.platform?.container?.volumes;
                    this.form.volumes = volumes?.length ? volumes : [];
                }

                this.form.ingress = j.platform.ingress || [];
                this.form.depend = j.platform.depends || [];
                this.form.helm = j.platform.helm || {
                    repository: '',
                    chartName: 'default',
                };
            }
            this.syncRoleBackendDefaults();
        },
        submit(otherData, callback) {
            this.$nextTick(() => {
                this.getMenu();
                this.$refs.formref.validate((valid) => {
                    if (!valid) { this.$message.warning('必填项不能为空'); return }
                    this.$emit('complete', this.json, this.yaml, otherData, callback);
                });
            })
        },

        replaceZpk(json) {
            json.application = this.json.application;
            json.source = this.json.source || {};
            json.platform = json.platform || {};
            json.platform.container = json.platform.container || {};
            this.json = json;
            this.initJSON();
            this.changeForm();
        },
        changeForm() {
            let j = this.json;
            if (j.application) {
                j.application.name = this.form.name;
                j.application.identifie = this.form.author + '-' + this.form.identifie;
                j.application.author = this.form.author;
                j.application.theme = this.form.theme;
                j.application.description = this.form.description;
                j.application.type = this.form.type;
                if (this.form.type != 'tradition') {
                    this.form.language = '';
                }
            }
            if (j.platform) {
                j.platform.container = j.platform.container || {};

                j.platform.container.cpu = Number(this.form.cpu);
                j.platform.container.mem = Number(this.form.mem);
                j.platform.container.image = this.form.image;
                j.platform.container.privileged = this.form.privileged;
                if (j.platform.container.build) {
                    j.platform.container.build.build_context = this.form.build_context;
                } else {
                    j.platform.container.build = { build_context: this.form.build_context }
                }

                j.platform.container.cmd = this.form.cmd.filter(i => i);

                let sc = this.form.securityContext;
                let securityContext = {};
                if (sc.runAsNonRoot) {
                    securityContext = { runAsNonRoot: sc.runAsNonRoot };
                }
                if (sc.runAsUser !== '') {
                    securityContext.runAsUser = Number(sc.runAsUser) || 0;
                }
                if (sc.runAsGroup !== '') {
                    securityContext.runAsGroup = Number(sc.runAsGroup) || 0;
                }
                if (sc.fsGroup !== '') {
                    securityContext.fsGroup = Number(sc.fsGroup) || 0;
                }
                j.platform.container.securityContext = securityContext;

                j.platform.depends = this.form.depend;

                if (this.form.type == 'helm') {
                    j.platform.helm = {
                        repository: this.form.helm.repository,
                        chartName: this.form.helm.chartName,
                    }
                } else {
                    delete j.platform.helm;
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
        addIngressEnd() {
            this.$refs.ingress.validate((valid) => {
                if (!valid) { return }
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
    width: 36px;
    height: 36px;
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
    width: 190px;
}

.backend-url-port {
    width: 120px;
}

.backend-url-protocol {
    width: 110px;
}

.backend-url-input {
    flex: 1;
}

.backend-url-config :deep(.el-input__wrapper) {
    height: 30px;
    box-shadow: none !important;
    border-radius: 0;
    background: transparent;
    padding: 0 10px;
}

.backend-url-config :deep(.el-input__inner) {
    height: 30px;
    line-height: 30px;
}

.backend-url-config :deep(.el-select .el-input.is-focus .el-input__wrapper) {
    box-shadow: none !important;
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
</style>
