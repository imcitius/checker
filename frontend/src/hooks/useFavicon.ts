import { useEffect } from 'react'

function createFaviconSVG(isHealthy: boolean): string {
  const color = isHealthy ? '#22c55e' : '#ef4444'
  return `data:image/svg+xml,${encodeURIComponent(
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">
      <circle cx="16" cy="16" r="14" fill="${color}" />
      ${isHealthy
        ? '<path d="M10 16l4 4 8-8" stroke="white" stroke-width="3" fill="none" stroke-linecap="round" stroke-linejoin="round"/>'
        : '<path d="M11 11l10 10M21 11l-10 10" stroke="white" stroke-width="3" fill="none" stroke-linecap="round"/>'
      }
    </svg>`)}`
}

export function useFavicon(unhealthyCount: number, totalEnabled: number) {
  useEffect(() => {
    if (totalEnabled === 0) return

    const isHealthy = unhealthyCount === 0
    const href = createFaviconSVG(isHealthy)

    let link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
    if (!link) {
      link = document.createElement('link')
      link.rel = 'icon'
      document.head.appendChild(link)
    }
    link.type = 'image/svg+xml'
    link.href = href

    // Also update the page title with status
    const baseTitle = 'Checker'
    document.title = unhealthyCount > 0
      ? `(${unhealthyCount} failing) ${baseTitle}`
      : baseTitle
  }, [unhealthyCount, totalEnabled])
}
