<template>
    <div class="w7-identifie" :class="{ 'is-disabled': disabled }">
        <input
            class="w7-identifie__input w7-identifie__author"
            :value="currentAuthor"
            :disabled="isAuthorDisabled"
            :placeholder="authorPlaceholder"
            :spellcheck="false"
            inputmode="text"
            pattern="[A-Za-z0-9]+"
            @input="inputAuthor"
        />
        <span class="w7-identifie__separator">-</span>
        <input
            class="w7-identifie__input w7-identifie__code"
            :value="currentIdentifie"
            :disabled="isIdentifieDisabled"
            :placeholder="identifiePlaceholder"
            :spellcheck="false"
            inputmode="text"
            pattern="[A-Za-z0-9]+"
            @input="inputIdentifie"
        />
    </div>
</template>

<script>
export default {
    name: 'W7Identifie',
    props: {
        modelValue: {
            type: String,
            default: ''
        },
        author: {
            type: String,
            default: ''
        },
        identifie: {
            type: String,
            default: ''
        },
        disabled: {
            type: Boolean,
            default: false
        },
        authorDisabled: {
            type: Boolean,
            default: false
        },
        identifieDisabled: {
            type: Boolean,
            default: false
        },
        authorPlaceholder: {
            type: String,
            default: 'w7'
        },
        identifiePlaceholder: {
            type: String,
            default: 'xxxxx'
        },
        onChange: {
            type: Function,
            default: null
        }
    },
    emits: ['update:modelValue', 'update:author', 'update:identifie', 'change'],
    computed: {
        modelAuthor() {
            const index = this.modelValue.indexOf('-');
            return index > -1 ? this.modelValue.slice(0, index) : this.modelValue;
        },
        modelIdentifie() {
            const index = this.modelValue.indexOf('-');
            return index > -1 ? this.modelValue.slice(index + 1) : '';
        },
        currentAuthor() {
            return this.author || this.modelAuthor;
        },
        currentIdentifie() {
            return this.identifie || this.modelIdentifie;
        },
        isAuthorDisabled() {
            return this.disabled || this.authorDisabled;
        },
        isIdentifieDisabled() {
            return this.disabled || this.identifieDisabled;
        }
    },
    methods: {
        inputAuthor(e) {
            this.inputPart('author', e);
        },
        inputIdentifie(e) {
            this.inputPart('identifie', e);
        },
        inputPart(type, e) {
            const value = String(e.target.value || '');
            if (!/^[A-Za-z0-9]*$/.test(value)) {
                // Reject invalid input instead of rewriting it. This keeps
                // the component a source-level restriction, while legacy
                // values loaded from storage remain untouched until edited.
                e.target.value = type === 'author' ? this.currentAuthor : this.currentIdentifie;
                return;
            }
            this.updateValue(type, value, e.target);
        },
        updateValue(type, value, input) {
            if (input) {
                input.value = value;
            }
            const data = {
                author: type === 'author' ? value : this.currentAuthor,
                identifie: type === 'identifie' ? value : this.currentIdentifie,
                type
            };
            data.value = `${data.author}-${data.identifie}`;

            this.$emit(type === 'author' ? 'update:author' : 'update:identifie', value);
            this.$emit('update:modelValue', data.value);
            this.$emit('change', data.value, type, data);
            if (this.onChange) {
                this.onChange(data.value, type, data);
            }
        }
    }
}
</script>

<style scoped>
.w7-identifie {
    display: inline-flex;
    align-items: center;
    width: 254px;
    height: 40px;
    overflow: hidden;
    color: #1f2937;
    background: #fff;
    border-radius: 2px;
    box-shadow: 0 0 0 1px #dcdfe6 inset;
    position: relative;
}

.w7-identifie input:disabled {
    background-color: #f4f5f7;
}

.w7-identifie__input {
    min-width: 0;
    height: 100%;
    padding: 0;
    font-size: 14px;
    line-height: 40px;
    color: #1f2937;
    text-align: center;
    letter-spacing: 0;
    background: transparent;
    border: 0;
    outline: none;
}

.w7-identifie__input::placeholder {
    color: #a8abb2;
    opacity: 1;
}

.w7-identifie__author {
    width: 66px;
    border-right: 1px solid #dcdfe6;
}

.w7-identifie__separator {
    width: 40px;
    position: absolute;
    left: 66px;
    top: 0;
    font-size: 20px;
    line-height: 40px;
    color: #374151;
    text-align: center;
}

.w7-identifie__code {
    flex: 1;
    padding-left: 40px;
    padding-right: 14px;
    text-align: left;
}

.w7-identifie.is-disabled {
    cursor: not-allowed;
    opacity: .65;
}

.w7-identifie.is-disabled .w7-identifie__input {
    cursor: not-allowed;
}
</style>
