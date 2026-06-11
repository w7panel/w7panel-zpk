<script>
import { Fragment, h } from 'vue';
import { IconPlus } from '@arco-design/web-vue/es/icon';

export default {
    name: 'ManifestConfigTable',
    inheritAttrs: false,
    emits: ['add'],
    props: {
        title: {
            type: String,
            default: '',
        },
        columns: {
            type: Array,
            default: () => [],
        },
        rows: {
            type: Array,
            default: () => [],
        },
        addText: {
            type: String,
            default: '',
        },
        tableClass: {
            type: [String, Array, Object],
            default: '',
        },
        alwaysShow: {
            type: Boolean,
            default: false,
        },
        rowKey: {
            type: [String, Function],
            default: '',
        },
    },
    computed: {
        safeRows() {
            return Array.isArray(this.rows) ? this.rows : [];
        },
        hasRows() {
            return this.safeRows.length > 0;
        },
        normalizedColumns() {
            let slotColumns = this.getSlotColumns();
            if (slotColumns.length) {
                return slotColumns;
            }
            return this.columns.map((column, index) => ({
                key: column.key || column.dataIndex || column.label || index,
                dataIndex: column.dataIndex || column.key,
                title: column.title || column.label,
                class: column.class,
                style: column.style,
                slots: {},
            }));
        },
        tableClassList() {
            return ['table', this.tableClass];
        },
    },
    methods: {
        flattenVNodes(nodes) {
            return (nodes || []).flatMap(node => {
                if (node?.type === Fragment) {
                    return this.flattenVNodes(node.children || []);
                }
                return node ? [node] : [];
            });
        },
        getSlotColumns() {
            let nodes = this.$slots.columns?.() || this.$slots.default?.() || [];
            return this.flattenVNodes(nodes)
                .filter(node => node?.type?.name === 'ManifestConfigTableColumn' || node?.type?.__name === 'ManifestConfigTableColumn')
                .map((node, index) => {
                    let props = node.props || {};
                    let dataIndex = props.dataIndex || props['data-index'] || props.field;
                    let style = props.style;
                    if (props.width !== undefined && props.width !== '') {
                        style = {
                            ...(typeof style == 'object' && !Array.isArray(style) ? style : {}),
                            width: typeof props.width == 'number' ? props.width + 'px' : props.width,
                        };
                    }
                    return {
                        key: dataIndex || props.title || index,
                        dataIndex,
                        title: props.title || '',
                        class: props.class,
                        style,
                        slots: node.children || {},
                    };
                });
        },
        renderColumnTitle(column) {
            if (column.slots?.title) {
                return column.slots.title({ column });
            }
            return column.title;
        },
        renderCell(column, record, index) {
            if (column.slots?.cell) {
                return column.slots.cell({ record, item: record, index, column });
            }
            if (column.slots?.default) {
                return column.slots.default({ record, item: record, index, column });
            }
            return record?.[column.dataIndex];
        },
        renderAddContent(slotName) {
            if (this.$slots[slotName]) {
                return this.$slots[slotName]();
            }
            return h('span', { class: 'addmenu' }, [
                h(IconPlus, { size: 14 }),
                this.addText,
            ]);
        },
        getRowKey(item, index) {
            if (typeof this.rowKey == 'function') {
                return this.rowKey(item, index);
            }
            if (this.rowKey && item?.[this.rowKey] !== undefined) {
                return item[this.rowKey];
            }
            return index;
        },
    },
    render() {
        let columns = this.normalizedColumns;
        let shouldRenderTable = this.alwaysShow || columns.length || this.hasRows || this.$slots.prepend || this.$slots.append;
        let children = [];
        if (this.title || this.$slots.title) {
            children.push(h('div', { class: 'manifest-front-table-title' }, this.$slots.title?.() || this.title));
        }
        if (shouldRenderTable) {
            children.push(h('table', { class: this.tableClassList }, [
                h('thead', [
                    h('tr', columns.map(column => h('td', {
                        key: column.key,
                        class: column.class,
                        style: column.style,
                    }, this.renderColumnTitle(column)))),
                ]),
                h('tbody', [
                    this.$slots.prepend?.(),
                    ...this.safeRows.map((record, index) => h('tr', { key: this.getRowKey(record, index) },
                        columns.map(column => h('td', {
                            key: column.key,
                            class: column.class,
                            style: column.style,
                        }, this.renderCell(column, record, index))))),
                    this.addText ? h('tr', [
                        h('td', {
                            colspan: Math.max(columns.length, 1),
                            class: 'cursor txt-c manifest-config-table-add',
                            onClick: () => this.$emit('add'),
                        }, this.renderAddContent('add')),
                    ]) : null,
                    this.$slots.append?.(),
                ]),
            ]));
        } else if (this.addText) {
            children.push(h('div', {
                class: 'empty-config-action cursor',
                onClick: () => this.$emit('add'),
            }, this.renderAddContent('empty-add')));
        }
        return h('div', {
            class: ['manifest-front-table-block', this.$attrs.class],
            style: this.$attrs.style,
        }, children);
    },
}
</script>

<style scoped>
.manifest-front-table-block {
    margin-top: 24px;
    width: 100%;
}

.manifest-front-table-block:first-child {
    margin-top: 0;
}

.manifest-front-table-title {
    margin-bottom: 10px;
    line-height: 22px;
}

.manifest-config-table-add {
    box-sizing: border-box;
}

.empty-config-action {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 40px;
    background: transparent;
}

.table {
    width: 100%;
}

.manifest-param-table {
    table-layout: fixed;
}

.manifest-param-table td:nth-child(1) {
    width: 30%;
}

.manifest-param-table td:nth-child(2) {
    width: 55%;
}

.manifest-param-table td:nth-child(3) {
    width: 15%;
}

.config-variable-table td:nth-child(1) {
    width: 20%;
}

.config-variable-table td:nth-child(2) {
    width: 34%;
}

.config-variable-table td:nth-child(3) {
    width: 31%;
}

.config-variable-table td:nth-child(4) {
    width: 15%;
}

.frontend-param-table td:nth-child(1) {
    width: 18%;
}

.frontend-param-table td:nth-child(2) {
    width: 28%;
}

.frontend-param-table td:nth-child(3) {
    width: 39%;
}

.frontend-param-table td:nth-child(4) {
    width: 15%;
}

.table thead tr:first-child td {
    background: var(--color-neutral-2);
    border-color: var(--color-border-1) !important;
}

.table td {
    background: #f7f8fa;
    border-color: var(--color-border-1) !important;
}

.table tbody tr:hover td,
.table tbody tr:hover td.cursor.txt-c {
    background: var(--color-neutral-2);
}

.table tbody tr:last-child td,
.table tbody tr td.cursor.txt-c {
    background: #f7f8fa;
}

.table :deep(tbody tr td) {
    background: #f7f8fa;
}

.table :deep(tbody tr:hover td),
.table :deep(tbody tr:hover td.cursor.txt-c) {
    background: var(--color-neutral-2);
}

.table :deep(tbody tr:last-child td),
.table :deep(tbody tr td.cursor.txt-c) {
    background: #f7f8fa;
}

.table :deep(.arco-select),
.table :deep(.arco-input-wrapper),
.table :deep(.arco-textarea-wrapper) {
    width: 100%;
    max-width: 100%;
}
</style>
