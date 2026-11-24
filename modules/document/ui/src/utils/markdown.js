import MarkdownIt from 'markdown-it';
import markdownItTaskLists from 'markdown-it-task-lists';

const GITHUB_ALERTS = {
    note: 'Note',
    tip: 'Tip',
    important: 'Important',
    warning: 'Warning',
    caution: 'Caution',
};
const markdownItGithubAlerts = (md) => {
    md.core.ruler.after('inline', 'github_alerts', (state) => {
        state.tokens.forEach((token, index) => {
            if (token.type !== 'blockquote_open') {
                return;
            }

            let inlineTokenIndex = state.tokens.findIndex((item, childIndex) => {
                return (
                    childIndex > index &&
                    item.type === 'inline' &&
                    /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*(?:\n|$)/i.test(item.content || '')
                );
            });
            if (inlineTokenIndex < 0) {
                return;
            }

            let inlineToken = state.tokens[inlineTokenIndex];
            let matched = inlineToken.content.match(/^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*(?:\n|$)/i);
            if (!matched) {
                return;
            }

            let type = matched[1].toLowerCase();
            token.attrJoin('class', 'markdown-alert markdown-alert-' + type);
            inlineToken.content = inlineToken.content.replace(
                /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*(?:\n|$)/i,
                ''
            );

            let children = inlineToken.children || [];
            let firstTextIndex = children.findIndex(
                (item) =>
                    item.type === 'text' && /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*$/i.test(item.content || '')
            );
            if (firstTextIndex >= 0) {
                children.splice(firstTextIndex, 1);
                if (['softbreak', 'hardbreak'].includes(children[firstTextIndex]?.type)) {
                    children.splice(firstTextIndex, 1);
                }
            }

            let titleToken = new state.Token('html_block', '', 0);
            titleToken.content = '<p class="markdown-alert-title">' + GITHUB_ALERTS[type] + '</p>\n';
            state.tokens.splice(index + 1, 0, titleToken);
        });
    });
};
const createInlineWrapRule = ({ marker, tag, name }) => {
    return (state, silent) => {
        let start = state.pos;
        if (!state.src.startsWith(marker, start)) {
            return false;
        }
        if (marker === '~' && state.src[start + 1] === '~') {
            return false;
        }

        let contentStart = start + marker.length;
        let close = contentStart;
        while ((close = state.src.indexOf(marker, close)) >= 0) {
            if (close === contentStart) {
                close += marker.length;
                continue;
            }
            if (marker === '~' && state.src[close + 1] === '~') {
                close += marker.length;
                continue;
            }

            if (!silent) {
                let oldPos = state.pos;
                let oldMax = state.posMax;
                state.push(name + '_open', tag, 1);
                state.pos = contentStart;
                state.posMax = close;
                state.md.inline.tokenize(state);
                state.pos = oldPos;
                state.posMax = oldMax;
                state.push(name + '_close', tag, -1);
            }

            state.pos = close + marker.length;
            return true;
        }

        return false;
    };
};
const markdownItMavonSyntax = (md) => {
    md.inline.ruler.before(
        'emphasis',
        'mavon_mark',
        createInlineWrapRule({
            marker: '==',
            tag: 'mark',
            name: 'mark',
        })
    );
    md.inline.ruler.before(
        'emphasis',
        'mavon_sup',
        createInlineWrapRule({
            marker: '^',
            tag: 'sup',
            name: 'sup',
        })
    );
    md.inline.ruler.before(
        'emphasis',
        'mavon_sub',
        createInlineWrapRule({
            marker: '~',
            tag: 'sub',
            name: 'sub',
        })
    );
};
const markdownParser = new MarkdownIt({
    html: true,
    linkify: true,
    breaks: false,
})
    .use(markdownItTaskLists, {
        enabled: true,
        label: true,
    })
    .use(markdownItMavonSyntax)
    .use(markdownItGithubAlerts);

const renderAlignedBlock = (align, content = '') => {
    let body = renderMarkdownWithGithubAlerts(content).trim();
    return '<div align="' + align + '">\n' + body + '\n</div>';
};

const renderGithubAlertBlock = (type, content = '') => {
    let body = markdownParser.render(content.trim());
    return (
        '<blockquote class="markdown-alert markdown-alert-' +
        type +
        '">\n' +
        '<p class="markdown-alert-title">' +
        GITHUB_ALERTS[type] +
        '</p>\n' +
        body +
        '</blockquote>'
    );
};
const normalizeEscapedBlockquoteMarkers = (content = '') => String(content || '').replace(/^(\s*)&gt;\s?/gm, '$1> ');
const renderMarkdownWithGithubAlerts = (content = '') => {
    let lines = normalizeEscapedBlockquoteMarkers(content).split('\n');
    let output = [];

    for (let index = 0; index < lines.length; index++) {
        let alignMatched = lines[index].replace(/\s+$/, '').match(/^:::\s*hljs-(left|center|right)\s*$/i);
        if (alignMatched) {
            let align = alignMatched[1].toLowerCase();
            let alignLines = [];
            index++;

            while (index < lines.length && !/^:::\s*$/.test(lines[index].trim())) {
                alignLines.push(lines[index]);
                index++;
            }

            output.push(renderAlignedBlock(align, alignLines.join('\n')));
            continue;
        }

        let matched = lines[index].replace(/\s+$/, '').match(/^>\s*\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*$/i);
        if (!matched) {
            output.push(lines[index]);
            continue;
        }

        let type = matched[1].toLowerCase();
        let alertLines = [];
        index++;

        while (index < lines.length && /^>\s?/.test(lines[index])) {
            alertLines.push(lines[index].replace(/^>\s?/, ''));
            index++;
        }
        index--;

        output.push(renderGithubAlertBlock(type, alertLines.join('\n')));
    }

    return markdownParser.render(output.join('\n'));
};

export function renderMarkdown(content = '') {
    return sanitizeMarkdown(renderMarkdownWithGithubAlerts(content));
}

// Sanitize rendered Markdown, including the legacy alignment extensions.
const tags = new Set(
    'a p div span br hr strong em s del mark sup sub blockquote pre code ul ol li input table thead tbody tr th td h1 h2 h3 h4 h5 h6'.split(
        ' '
    )
);
function sanitizeMarkdown(html) {
    const parsed = new DOMParser().parseFromString(String(html), 'text/html');
    const clean = (node) => {
        [...node.children].forEach((child) => {
            if (!tags.has(child.tagName.toLowerCase())) {
                child.remove();
                return;
            }
            [...child.attributes].forEach((attribute) => {
                const name = attribute.name.toLowerCase();
                if (name === 'class' && /^[\w -]*$/.test(attribute.value)) return;
                if (name === 'align' && /^(left|center|right)$/.test(attribute.value)) return;
                if (child.tagName === 'A' && name === 'href') {
                    try {
                        const value = attribute.value.trim();
                        const url = new URL(value, location.href);
                        if (['https:', 'http:'].includes(url.protocol) && !url.username && !url.password) return;
                    } catch (_) {
                        /* Invalid links are removed. */
                    }
                }
                if (child.tagName === 'INPUT' && ['type', 'checked', 'disabled'].includes(name)) return;
                child.removeAttribute(attribute.name);
            });
            if (child.tagName === 'INPUT') {
                child.type = 'checkbox';
                child.disabled = true;
            }
            if (child.tagName === 'A') {
                child.target = '_blank';
                child.rel = 'noopener noreferrer';
            }
            clean(child);
        });
    };
    clean(parsed.body);
    return parsed.body.innerHTML;
}
