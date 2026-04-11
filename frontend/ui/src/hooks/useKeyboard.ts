import { useEffect, useRef } from 'react'

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
  const actionsRef = useRef(actions)
  actionsRef.current = actions

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      const target = e.target as HTMLElement
      const isInput =
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.tagName === 'SELECT' ||
        target.isContentEditable

      // Cmd+K — command palette (always active)
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        actionsRef.current.onCommandPalette()
        return
      }

      // Esc — collapse / blur
      if (e.key === 'Escape') {
        if (isInput) {
          ;(target as HTMLInputElement).blur()
          return
        }
        actionsRef.current.onCollapse()
        return
      }

      // Skip other shortcuts if user is typing in an input
      if (isInput) return

      switch (e.key) {
        case 'j':
          e.preventDefault()
          actionsRef.current.onNavigateDown()
          break
        case 'k':
          e.preventDefault()
          actionsRef.current.onNavigateUp()
          break
        case 'Enter':
          e.preventDefault()
          actionsRef.current.onExpand()
          break
        case '/':
          e.preventDefault()
          actionsRef.current.onFocusSearch()
          break
        case 'g':
          e.preventDefault()
          actionsRef.current.onToggleGroup()
          break
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])
}
