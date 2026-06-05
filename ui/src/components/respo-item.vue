<template>
    <div class="list-item" @click="goDetail">
        <div class="remote-tag" v-if="data.remote_formula_info_url">
            第三方
        </div>
        <div class="df">
            <div class="left">
                <div class="app-icon">
                    <img v-if="logoimg" :src="logoimg" class="img df-s0" alt="" />
                </div>
            </div>
            <div class="fc">
                <div class="df" style="height:64px;align-items: center;">
                    <div style="flex: 1;padding-right: 10px;">
                        <span class="one-hide app-name" style="vertical-align:middle;">{{ data.name }}</span>
                        <div class="mt-8 one-hide">
                            <span class="c-00-6 fs-14">标识：{{ data.identifie }}</span>
                            <span class="c-00-6 fs-14" style="margin-left:10px;">最新版本：{{ data.version?.name || ''
                                }}</span>
                        </div>
                    </div>
                    <span class="buy-users" v-if="data.install_total > 0">
                        <span class="user-avatar" v-for="item in data.install_users_avatar" :key="item.id">
                            <img :src="item" alt="">
                        </span>
                        <span class="user-total">
                            +{{ data.install_total }}
                        </span>
                    </span>
                </div>
            </div>
        </div>
        <div class="description" style="line-height:20px;">{{ data.description }}</div>
        <div class="tags">
            <span v-for="tag in data.tag" :key="tag.id" class="tag fs-12 cursor" @click.stop="tagClick(tag)">{{ tag.name
            }}</span>
        </div>
        <div class="icon-launch">
            <svg fill="none" stroke="#666" stroke-width="4" viewBox="0 0 48 48" aria-hidden="true" focusable="false"
                stroke-linecap="butt" stroke-linejoin="miter" class="arco-icon arco-icon-launch"
                style="font-size: 32px;">
                <path
                    d="M41 26v14a1 1 0 0 1-1 1H8a1 1 0 0 1-1-1V8a1 1 0 0 1 1-1h14M19.822 28.178 39.899 8.1M41 20V7H28">
                </path>
            </svg>
        </div>


    </div>
</template>

<script>
import { getPanelToken } from '@/utils/panel-token';

export default {
    props: ['data', 'webUrl'],
    data() {
        return {
            logoimg: '',
        }
    },
    created() {
        let base = this.webUrl || '';
        this.logoimg = base + this.data.icon;
    },
    methods: {
        tagClick(item) {
            this.$emit('tagClick', item);
        },
        goDetail() {
            let path = this.webUrl + '/respo/info/' + this.data.identifie;
            const panelToken = getPanelToken();
            if (!this.data.goods_id) {
                if (window.$wujie) {
                    window.open(`/app/store-install?path=${encodeURIComponent(path)}&thirdpartyCDToken=${encodeURIComponent(panelToken)}`)
                } else {
                    this.$emit('goInstall', this.data.identifie);
                }
            } else {
                if (window.$wujie) {
                    this.$router.push(`/site-detail/${this.data.identifie}?token=${encodeURIComponent(panelToken)}&path=${encodeURIComponent(path)}&fromHost=${location.origin}`)
                } else {
                    window.open(`#/site-detail/${this.data.identifie}?token=${encodeURIComponent(panelToken)}&path=${encodeURIComponent(path)}&fromHost=${location.origin}`)
                }
            }
        }
    }
}
</script>

<style scoped lang="scss">
.df {
    display: flex;
}

.list-item {
    width: 100%;
    height: 196px;
    box-sizing: border-box;
    border: 1px solid #e7e7e7;
    border-radius: 8px;
    padding: 20px;
    position: relative;
    cursor: pointer;

    .remote-tag {
        position: absolute;
        top: 0;
        right: 0;
        font-size: 12px;
        color: #fff;
        background-color: #2d5fff;
        padding: 0 10px;
        border-radius: 0 8px 0 8px;
    }

    .icon-launch {
        position: absolute;
        right: 20px;
        bottom: 20px;
        font-size: 0;

        svg {
            height: 20px;
            width: 20px;
        }
    }
}

.list-item:hover {
    box-shadow: 0px 3px 14px 2px rgba(0, 0, 0, 0.05), 0px 8px 10px 1px rgba(0, 0, 0, 0.06), 0px 5px 5px -3px rgba(0, 0, 0, 0.1);
}

.list-item .left .app-icon {
    width: 64px;
    height: 64px;
    margin-right: 23px;
    box-sizing: border-box;
    border-radius: 8px;
    position: relative;
    overflow: hidden;
}

.list-item .left .app-icon .img {
    width: 100%;
    height: 100%;
    display: block;
    border-radius: 8px;
}

.app-name {
    flex: 1;
    font-size: 16px;
    font-weight: 700;
}

.buy-users {
    display: flex;
    align-items: center;
    flex: 0 0 auto;
}

.user-avatar {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    overflow: hidden;
    margin-right: 4px;
    border: 1px solid #fff;
    z-index: 3;

    &+.user-avatar {
        margin-left: -18px;
        z-index: 2;
    }

    &+.user-avatar+.user-avatar {
        z-index: 1;
    }
}

.user-avatar img {
    width: 100%;
    height: 100%;
    display: block;
}

.user-total {
    font-size: 14px;
}

.description {
    color: #666;
    font-size: 14px;
    line-height: 20px;
    height: 40px;
    overflow: hidden;
    word-break: break-all;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    margin-top: 10px;
}

.tags {
    margin-top: 20px;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;

    .tag {
        display: inline-block;
        padding: 0 10px;
        height: 22px;
        font-size: 12px;
        line-height: 22px;
        background-color: #f2f4f5;
        border-radius: 4px;
        margin-right: 10px;
    }
}
</style>
