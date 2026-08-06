import { describe, expect, it } from 'vitest'
import { escapePlainTextToHtml } from './markdown-safe-text'

describe('escapePlainTextToHtml', () => {
  it('escapes raw HTML tags', () => {
    expect(escapePlainTextToHtml('<script>alert(1)</script>')).toBe(
      '&lt;script&gt;alert(1)&lt;/script&gt;'
    )
  })

  it('escapes quotes and ampersands', () => {
    expect(escapePlainTextToHtml('a & b "c" \'d\'')).toBe('a &amp; b &quot;c&quot; &#39;d&#39;')
  })

  it('turns newlines into <br>', () => {
    expect(escapePlainTextToHtml('line1\nline2')).toBe('line1<br>line2')
  })

  it('keeps plain markdown text uninterpreted', () => {
    expect(escapePlainTextToHtml('plain **markdown** [link](http://x)')).toBe(
      'plain **markdown** [link](http://x)'
    )
  })

  it('escapes empty string safely', () => {
    expect(escapePlainTextToHtml('')).toBe('')
  })
})
