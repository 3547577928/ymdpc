export type MarkdownHeading = { level: number; text: string; id: string }

function normalizeHeadingText(value: string) {
  return value
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/[*_`~]/g, '')
    .trim()
}

export function headingBaseId(value: string) {
  const normalized = normalizeHeadingText(value)
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, '-')
    .replace(/^-+|-+$/g, '')
  return normalized || 'section'
}

export function createHeadingSlugger() {
  const counts = new Map<string, number>()
  return (text: string) => {
    const base = headingBaseId(text)
    const count = (counts.get(base) ?? 0) + 1
    counts.set(base, count)
    return count === 1 ? base : `${base}-${count}`
  }
}

export function extractMarkdownHeadings(content: string): MarkdownHeading[] {
  const slug = createHeadingSlugger()
  const headings: MarkdownHeading[] = []
  let inFence = false
  for (const line of content.split('\n')) {
    if (/^\s*```/.test(line)) {
      inFence = !inFence
      continue
    }
    if (inFence) continue
    const match = line.match(/^\s{0,3}(#{1,3})\s+(.+?)\s*#*\s*$/)
    if (!match) continue
    const text = normalizeHeadingText(match[2])
    if (text) headings.push({ level: match[1].length, text, id: slug(text) })
  }
  return headings
}
