import { bus as wujieBus } from "wujie";

const getHandles = () => {
    return window.$wujie?.props?.handles || {};
};

export const emitWujieEvent = (eventName, ...args) => {
    const handles = getHandles();
    const handle = handles?.[eventName];
    if (typeof handle == 'function') {
        return handle.apply(handles, args);
    }

    const bus = window.$wujie?.bus || wujieBus;
    return bus?.$emit?.(eventName, ...args);
};

export default emitWujieEvent;
