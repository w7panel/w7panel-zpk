<template>
    <div class="df df-ww iconbox">
        <div v-for="(item, name) in iconMap" :key="name" class="icon df df-c ai-c jc-c cursor" v-html="item.context" @click="selectIcon(item)">
        </div>
    </div>
</template>
<script>

export default{
    data(){
        return {
            iconMap: {}
        }
    },
    created(){
        this.init();
    },
    methods: {
        init(){
            this.loadAllSvgIcons();
        },
        selectIcon(item){
            let icon = this.parseSvgToElements(item.context);
            this.$emit('submit',{
                ...item,
                json: icon,
            });
        },
        loadAllSvgIcons() {
            try {
                const svgContext = require.context('@/assets/arco-design-icon', false, /\.svg$/)
                svgContext.keys().forEach(filePath => {
                    const fileName = filePath.replace(/^\.\/(.*)\.\w+$/, '$1')
                    const svgContent = svgContext(filePath)
                    this.iconMap[fileName] = {src:svgContent};
                })

                Object.keys(this.iconMap).map(async (name) => {
                    const svgUrl = this.iconMap[name].src;
                    try {
                        const response = await fetch(svgUrl)
                        if (!response.ok){return}
                        const svgContent = await response.text()
                        const svgTag = svgContent;
                        this.iconMap[name].context = svgTag;
                    } catch (err) {
                        return null
                    }
                })

            } catch (error) {

                this.iconMap = {}
            }
        },

        parseSvgToElements(svgText) {
            try {
                const tempContainer = document.createElement('div');
                tempContainer.innerHTML = svgText;
                const svgElement = tempContainer.querySelector('svg');
                if (!svgElement) throw new Error('未找到有效的SVG根元素');


                const resultElements = [];


                const svgRootObj = { type: 'svg' };
                Array.from(svgElement.attributes).forEach(attr => {

                    svgRootObj[attr.name] = isNaN(attr.value) ? attr.value : Number(attr.value);
                });
                delete svgRootObj.xmlns;
                resultElements.push(svgRootObj);


                const extractElements = (parent) => {
                    Array.from(parent.children).forEach(child => {
                        const elementObj = { type: child.tagName.toLowerCase() };

                        Array.from(child.attributes).forEach(attr => {
                            elementObj[attr.name] = isNaN(attr.value) ? attr.value : Number(attr.value);
                        });

                        if (child.textContent.trim()) {
                            elementObj.content = child.textContent.trim();
                        }

                        resultElements.push(elementObj);


                        if (child.children.length) {
                            extractElements(child);
                        }
                    });
                };


                extractElements(svgElement);


                return resultElements;

            } catch (error) {


                return [{
                    type: 'svg',
                    width: 36,
                    height: 36
                }];
            }
        },
    }
}
</script>
<style scoped>

.iconbox{width:790px; height:500px; overflow:auto;}
.iconbox::-webkit-scrollbar {width:10px;}
.iconbox::-webkit-scrollbar-track {background: transparent;}
.iconbox::-webkit-scrollbar-thumb {background: #eee; border-radius: 6px;}
.iconbox :deep(svg){width:24px;height:24px;}

.icon{border:1px solid #f0f0f0; box-sizing:border-box; width:64px; height:64px;}
.icon:hover i{color:#2d5fff;}
.icon:hover{border-color:#2d5fff;}
</style>
