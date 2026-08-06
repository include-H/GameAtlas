/**
 * Markdown 渲染异常时的安全兜底：把原始内容转义为纯文本 HTML。
 *
 * 绝不能把未转义的 Markdown（可能内嵌原生 HTML）直接交给 v-html，
 * 否则 MarkdownIt `html: false` 的转义保证会被兜底路径绕过，形成 XSS。
 * 换行转为 <br> 以保留基本的阅读排版，其余字符全部转义。
 */
export function escapePlainTextToHtml(content: string): string {
  return content
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
    .replace(/\n/g, '<br>')
}
