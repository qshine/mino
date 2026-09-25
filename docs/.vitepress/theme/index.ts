import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'
import mermaid from 'mermaid'
import './style.css'

// Disable the global load handler before Vue mounts its diagram components.
if (typeof window !== 'undefined') {
  mermaid.initialize({ startOnLoad: false })
}

export default {
  extends: DefaultTheme,
  enhanceApp({ router }) {
    if (typeof window === 'undefined' || !document.getElementById('mino-umami')) return

    let historyNavigation = false
    window.addEventListener('popstate', () => { historyNavigation = true })
    router.onAfterRouteChange = () => {
      if (!historyNavigation) return
      historyNavigation = false
      // Umami observes pushState/replaceState, but not popstate. Refresh its URL
      // after back/forward navigation, preserving VitePress's history state.
      // Leave ordinary navigation alone to preserve the tracker's referrer.
      history.replaceState(history.state, '', location.href)
    }
  }
} satisfies Theme
