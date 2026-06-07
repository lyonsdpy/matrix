export function parseRedirectParam(search: string = window.location.search): string | null {
  try {
    const value = new URLSearchParams(search).get('redirect')
    if (!value) return null
    return value
  } catch {
    return null
  }
}

export function goRedirect(url: string): void {
  window.location.href = url
}
