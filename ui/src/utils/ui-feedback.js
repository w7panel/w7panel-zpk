import { h } from 'vue';
import { Message, Modal } from '@arco-design/web-vue';

const messageTypes = ['info', 'success', 'warning', 'error', 'loading', 'normal'];

function normalizeMessageType(type) {
    if (type === 'warn') {
        return 'warning';
    }

    return messageTypes.includes(type) ? type : 'normal';
}

function normalizeMessageConfig(config) {
    if (typeof config === 'string') {
        return { content: config };
    }

    const content = config?.content || config?.message || '';
    return {
        ...config,
        content
    };
}

function renderModalContent(content, dangerouslyUseHTMLString) {
    if (dangerouslyUseHTMLString) {
        return () => h('div', { innerHTML: content });
    }

    return content;
}

export function message(config) {
    const normalizedConfig = normalizeMessageConfig(config);
    const type = normalizeMessageType(normalizedConfig.type);
    const { content, duration, id, icon, position, showIcon, closable, onClose, resetOnHover } = normalizedConfig;

    return Message[type]({
        content,
        duration,
        id,
        icon,
        position,
        showIcon,
        closable,
        onClose,
        resetOnHover
    });
}

export function messageSuccess(content, config = {}) {
    return message({ ...config, content, type: 'success' });
}

export function messageError(content, config = {}) {
    return message({ ...config, content, type: 'error' });
}

export function messageWarning(content, config = {}) {
    return message({ ...config, content, type: 'warning' });
}

export function alert(options = {}) {
    const {
        title = '提示',
        content = '',
        confirmButtonText,
        okText = confirmButtonText || '确定',
        dangerouslyUseHTMLString = false,
        ...modalOptions
    } = options;

    return Modal.info({
        title,
        content: renderModalContent(content, dangerouslyUseHTMLString),
        okText,
        hideCancel: true,
        ...modalOptions
    });
}

export function confirm(options = {}) {
    const {
        title = '提示',
        content = '',
        confirmButtonText,
        cancelButtonText,
        okText = confirmButtonText || '确定',
        cancelText = cancelButtonText || '取消',
        dangerouslyUseHTMLString = false,
        onOk,
        onCancel,
        ...modalOptions
    } = options;

    return Modal.confirm({
        title,
        content: renderModalContent(content, dangerouslyUseHTMLString),
        okText,
        cancelText,
        onOk,
        onCancel,
        ...modalOptions
    });
}
