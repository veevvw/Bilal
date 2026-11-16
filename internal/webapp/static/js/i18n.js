import { arabic } from './messages.js';
const normalize = value => value === 'en' ? 'en' : 'ar';
let language = 'ar';
try { language = normalize(localStorage.getItem('synctune:language')); } catch { /* Storage is optional. */ }
export const getLanguage = () => language;
export function t(source, ...values) {
    const text = language === 'en' ? source : (arabic[source] ?? source);
    return String(text).replace(/\{\{(\d+)\}\}/g, (_, index) => String(values[Number(index)] ?? ''));
}
function apply() {
    document.documentElement.lang = language;
    document.documentElement.dir = language === 'ar' ? 'rtl' : 'ltr';
    for (const node of document.querySelectorAll('[data-i18n]')) node.textContent = t(node.dataset.i18n);
    for (const attr of ['aria-label','placeholder','title','alt','data-choice-question','data-choice-description','data-display']) {
        for (const node of document.querySelectorAll(`[data-i18n-${attr}]`)) node.setAttribute(attr, t(node.getAttribute(`data-i18n-${attr}`)));
    }
    for (const select of document.querySelectorAll('#ui-language, #ui-language-settings')) select.value = language;
}
export function errorText(source) {
    source = String(source || 'The action failed.');
    if (language === 'en') return source;
    if (arabic[source]) return arabic[source];
    for (const [key, translated] of Object.entries(arabic)) {
        if (!/%[dsfv]/.test(key) || key.length > 500) continue;
        const fragments = key.split(/%[dsfv]/);
        const escaped = fragments.map(part => part.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'));
        const match = source.match(new RegExp('^' + escaped.join('(.*?)') + '$', 's'));
        if (match) { let index = 0; return translated.replace(/%[dsfv]/g, () => match[++index]); }
    }
    return t('That action could not finish. Please try again.');
}
export function setLanguage(value) {
    const next = normalize(value);
    if (next === language) return;
    language = next;
    try { localStorage.setItem('synctune:language', next); } catch { /* Keep the current choice in memory. */ }
    // Reload recreates dynamic controls and translated choice menus consistently.
    location.reload();
}
if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', apply, { once: true });
else apply();
document.addEventListener('change', event => { if (['ui-language', 'ui-language-settings'].includes(event.target.id)) setLanguage(event.target.value); });
apply();
