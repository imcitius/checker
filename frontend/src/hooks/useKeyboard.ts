import { useEffect, useCallback } from 'react'

interface KeyboardActions {
  onNavigateDown: () => void
  onNavigateUp: () => void
  onExpand: () => void
  onCollapse: () => void
  onFocusSearch: () => void
  onToggleGroup: () => void
  onCommandPalette: () => void
}

export function useKeyboard(actions: KeyboardActions) {
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      const target = e.target as HTMLElement
      const isInput =
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.tagName === 'SELECT' ||
        target.isContentEditable

      // Cmd+K — command palette (always active)
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        actions.onCommandPalette()
        return
      }

      // Esc — collapse / blur
      if (e.key === 'Escape') {
        if (isInput) {
          ;(target as HTMLInputElement).blur()
          return
        }
        actions.onCollapse()
        return
      }

      // Skip other shortcuts if user is typing in an input
      if (isInput) return

      switch (e.key) {
        case 'j':
          e.preventDefault()
          actions.onNavigateDown()
          break
        case 'k':
          e.preventDefault()
          actions.onNavigateUp()
          break
        case 'Enter':
          e.preventDefault()
          actions.onExpand()
          break
        case '/':
          e.preventDefault()
          actions.onFocusSearch()
          break
        case 'g':
          e.preventDefault()
          actions.onToggleGroup()
          break
      }
    },
    [actions]
  )

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])
}
