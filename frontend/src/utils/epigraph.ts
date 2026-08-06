type EpigraphLineType = 'cn' | 'en'

/**
 * 题记块（:::epigraph）内每一行的排版分类：
 * - 'en'：整行为纯英文（含数字/标点），渲染为小字
 * - 'cn'：包含中文（或既无中文也无英文），渲染为装饰性大字
 *
 * 历史 bug：曾用「ASCII 字母数 > 中文字符数」判定，导致
 * “Sam Fisher 依然更激进了，”这类中英混合行被误判为纯英文小字。
 */
export function classifyEpigraphLine(line: string): EpigraphLineType {
  const asciiLetters = (line.match(/[A-Za-z]/g) || []).length
  const cjkChars = (line.match(/[\u3400-\u9fff]/g) || []).length

  if (asciiLetters > 0 && cjkChars === 0) {
    return 'en'
  }

  return 'cn'
}
