import DefaultTheme from 'vitepress/theme'
import mermaid from 'mermaid'
import './style.css'

// Disable the global load handler before Vue mounts its diagram components.
if (typeof window !== 'undefined') {
  mermaid.initialize({ startOnLoad: false })
}

export default DefaultTheme
