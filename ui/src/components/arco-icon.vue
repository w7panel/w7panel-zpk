<template>
    <span
        class="arco-icon"
        :class="{ 'arco-icon--colored': normalizedColor }"
        :style="iconStyle"
        v-html="svgContent"
    ></span>
</template>

<script>
const svgContext = require.context('@/assets/arco-design-icon', false, /\.svg$/)
const iconUrlMap = svgContext.keys().reduce((map, filePath) => {
    const fileName = filePath.replace(/^\.\/(.*)\.svg$/, '$1')
    const iconModule = svgContext(filePath)
    map[fileName] = iconModule.default || iconModule
    return map
}, {})

const svgCache = {}

export default {
    name: 'ArcoIcon',
    props: {
        name: {
            type: String,
            default: ''
        },
        icon: {
            type: String,
            default: ''
        },
        size: {
            type: [Number, String],
            default: 24
        },
        color: {
            type: String,
            default: ''
        }
    },
    data() {
        return {
            svgContent: ''
        }
    },
    computed: {
        iconName() {
            return this.normalizeName(this.name || this.icon)
        },
        normalizedColor() {
            return this.color ? this.color.trim() : ''
        },
        iconStyle() {
            const size = this.normalizeSize(this.size)
            const style = {
                width: size,
                height: size,
                fontSize: size
            }

            if (this.normalizedColor) {
                style.color = this.normalizedColor
            }

            return style
        }
    },
    watch: {
        iconName: {
            immediate: true,
            handler() {
                this.loadIcon()
            }
        }
    },
    methods: {
        normalizeName(name) {
            return String(name || '')
                .trim()
                .replace(/^\.?\//, '')
                .replace(/\.svg$/i, '')
        },
        normalizeSize(size) {
            if (typeof size === 'number') {
                return `${size}px`
            }

            const value = String(size || '').trim()
            return /^\d+(\.\d+)?$/.test(value) ? `${value}px` : value
        },
        async loadIcon() {
            const iconName = this.iconName
            const iconUrl = iconUrlMap[iconName]

            if (!iconUrl) {
                this.svgContent = ''
                return
            }

            if (svgCache[iconName]) {
                this.svgContent = svgCache[iconName]
                return
            }

            try {
                const response = await fetch(iconUrl)
                if (!response.ok) {
                    this.svgContent = ''
                    return
                }

                const svgText = await response.text()
                svgCache[iconName] = svgText
                if (this.iconName === iconName) {
                    this.svgContent = svgText
                }
            } catch (error) {
                this.svgContent = ''
            }
        }
    }
}
</script>

<style scoped>
.arco-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
    color: inherit;
    vertical-align: top;
}

.arco-icon :deep(svg) {
    display: block;
    width: 100%;
    height: 100%;
    color: inherit;
}

.arco-icon--colored :deep(svg *[stroke]:not([stroke="none"])) {
    stroke: currentColor !important;
}
</style>
