<template>
    <div ref="files_editor_cont" class="content">
        <div id="editor_textarea" style="width:100%;"></div>
        <a-button type="primary" style="margin-top:20px; width:100px;" @click="save">确定</a-button>
    </div>
</template>

<script>
import { basicSetup } from "codemirror"
import { indentWithTab } from "@codemirror/commands"
import { EditorView, keymap } from "@codemirror/view"

export default {
    props: ['filecont', 'filename'],
    data() {
        return {
            editor: null,
            files: [],
            fileid: '',
            disabled: false,
            form: {
                title: '',
                content: '',
            }
        }
    },
    created() {
        if (this.filename) {
            this.form.title = this.filename || '';
            this.disabled = true;
        }
    },
    watch: {
        filecont() {
            this.inputContent(this.filecont || '');
        }
    },
    mounted() {
        this.init(() => {
            this.inputContent(this.filecont || '');
        });
    },
    methods: {
        init(callback) {
            const ivory = "#abb2bf",
                stone = "#7d8799",
                darkBackground = "#21252b",
                highlightBackground = "#2c313a",
                background = "#282c34",
                tooltipBackground = "#353a42",
                selection = "#3E4451",
                cursor = "#528bff";

            let myTheme = EditorView.theme({
                "&": {
                    color: ivory,
                    backgroundColor: background,
                    height: "480px"
                },
                ".cm-content": { caretColor: cursor },
                ".cm-cursor, .cm-dropCursor": { borderLeftColor: cursor },
                "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection": { backgroundColor: selection },

                ".cm-panels": { backgroundColor: darkBackground, color: ivory },
                ".cm-panels.cm-panels-top": { borderBottom: "2px solid black" },
                ".cm-panels.cm-panels-bottom": { borderTop: "2px solid black" },

                ".cm-searchMatch": { backgroundColor: "#72a1ff59", outline: "1px solid #457dff" },
                ".cm-searchMatch.cm-searchMatch-selected": { backgroundColor: "#6199ff2f" },

                ".cm-activeLine": { backgroundColor: highlightBackground },
                ".cm-selectionMatch": { backgroundColor: "#aafe661a" },

                "&.cm-focused .cm-matchingBracket, &.cm-focused .cm-nonmatchingBracket": { backgroundColor: "#bad0f847", outline: "1px solid #515a6b" },

                ".cm-gutters": { backgroundColor: background, color: stone, border: "none" },

                ".cm-activeLineGutter": { backgroundColor: highlightBackground },

                ".cm-foldPlaceholder": { backgroundColor: "transparent", border: "none", color: "#ddd" },

                ".cm-tooltip": { border: "none", backgroundColor: tooltipBackground },
                ".cm-tooltip .cm-tooltip-arrow:before": { borderTopColor: "transparent", borderBottomColor: "transparent" },
                ".cm-tooltip .cm-tooltip-arrow:after": { borderTopColor: tooltipBackground, borderBottomColor: tooltipBackground },
                ".cm-tooltip-autocomplete": {
                    "& > ul > li[aria-selected]": {
                        backgroundColor: highlightBackground,
                        color: ivory
                    }
                },
                ".cm-scroller": { overflow: "auto" },
                ".cm-content, .cm-gutter": { minHeight: "300px" },
            }, { dark: true });

            this.$nextTick(() => {
                if (!this.editor) {
                    document.getElementById("editor_textarea").innerHTML = "";
                    this.editor = new EditorView({
                        doc: "",
                        extensions: [
                            basicSetup,
                            myTheme,
                            keymap.of([indentWithTab]),
                        ],
                        parent: document.getElementById("editor_textarea"),
                    });
                    callback && callback();
                }
            })
        },
        inputContent(file) {
            if (!this.editor) { return; }
            let txt = this.editor.state.doc.toString();
            this.editor.dispatch({ changes: { from: 0, to: txt.length, insert: file } });
        },
        save() {
            this.form.content = this.editor.state.doc.toString();
            this.$emit('complete', this.form);
        },
    }
}
</script>

<style scoped>
.content {
    padding: 0;
    height: 100%;
    box-sizing: border-box;
}
</style>
<style>
.cm-editor.cm-focused {
    outline: none !important;
}
</style>
