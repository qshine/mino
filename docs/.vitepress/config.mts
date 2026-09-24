import { withMermaid } from 'vitepress-plugin-mermaid'

const repository = 'https://github.com/qshine/mino'

export default withMermaid({
  title: 'Mino',
  description: 'Build an agent from scratch in Go, one illustrated chapter at a time.',
  base: '/mino/',
  srcDir: 'books',
  // Keep English at the site root while both languages have explicit source directories.
  rewrites: { 'en/:path*': ':path*' },
  lastUpdated: true,
  cleanUrls: false,
  head: [['meta', { name: 'theme-color', content: '#176b58' }]],
  vite: {
    // The plugin injects Mermaid imports; pre-bundle its CommonJS dependencies for local preview.
    optimizeDeps: { include: ['mermaid'] }
  },
  mermaid: {
    // The VitePress component renders diagrams; avoid a second automatic pass.
    startOnLoad: false,
    securityLevel: 'strict', theme: 'base',
    flowchart: { useMaxWidth: false },
    sequence: { useMaxWidth: false },
    themeVariables: {
      primaryColor: '#e8f3ee', primaryTextColor: '#17382d',
      primaryBorderColor: '#4d8d79', lineColor: '#526c62',
      secondaryColor: '#f4f1e7', tertiaryColor: '#eef3f5',
      fontFamily: '-apple-system, BlinkMacSystemFont, Segoe UI, sans-serif'
    }
  },
  locales: {
    root: {
      label: 'English', lang: 'en', titleTemplate: ':title · Build an Agent',
      themeConfig: {
        siteTitle: 'Mino · Build an Agent',
        nav: [
          { text: 'Read the book', link: '/chapters/01-terminal-chat' },
          { text: 'Roadmap', link: '/plan-todo-chapters' },
          { text: 'GitHub', link: repository }
        ],
        sidebar: [
          { text: 'Start here', items: [
            { text: 'About this book', link: '/' },
            { text: 'Setup and installation', link: '/getting-started' },
            { text: 'Chapter roadmap', link: '/plan-todo-chapters' }
          ] },
          { text: 'Part I · Connect to a model', items: [
            { text: '01 A terminal conversation', link: '/chapters/01-terminal-chat' }
          ] },
          { text: 'Maintain the book', collapsed: true, items: [
            { text: 'Writing and updates', link: '/maintaining-the-book' },
            { text: 'Application releases', link: '/releases' }
          ] }
        ],
        editLink: { pattern: `${repository}/edit/main/docs/books/:path`, text: 'Improve this page on GitHub' },
        footer: { message: 'One problem per chapter. Working code behind every explanation.' }
      }
    },
    zh: {
      label: '简体中文', lang: 'zh-CN', titleTemplate: ':title · 从零构建 Agent',
      description: '用 Go 从一个终端问答程序开始，逐章理解并实现 Agent。',
      themeConfig: {
        siteTitle: 'Mino · 从零构建 Agent',
        nav: [
          { text: '阅读教程', link: '/zh/chapters/01-terminal-chat' },
          { text: '章节路线', link: '/zh/plan-todo-chapters' },
          { text: 'GitHub', link: repository }
        ],
        sidebar: [
          { text: '开始阅读', items: [
            { text: '关于这本书', link: '/zh/' },
            { text: '准备与安装', link: '/zh/getting-started' },
            { text: '章节路线与进度', link: '/zh/plan-todo-chapters' }
          ] },
          { text: '第一部分 · 搭通对话', items: [
            { text: '01 与模型对话', link: '/zh/chapters/01-terminal-chat' }
          ] },
          { text: '维护这本书', collapsed: true, items: [
            { text: '写作与更新流程', link: '/zh/maintaining-the-book' },
            { text: '应用版本发布', link: '/zh/releases' }
          ] }
        ],
        outline: { level: [2, 3], label: '本页目录' },
        docFooter: { prev: '上一页', next: '下一页' },
        darkModeSwitchLabel: '外观',
        lightModeSwitchTitle: '切换到浅色模式',
        darkModeSwitchTitle: '切换到深色模式',
        sidebarMenuLabel: '章节目录', returnToTopLabel: '回到顶部',
        lastUpdated: { text: '最近更新' },
        editLink: { pattern: `${repository}/edit/main/docs/books/:path`, text: '在 GitHub 上改进本页' },
        footer: { message: '每章解决一个问题，让原理变成可运行的代码。' }
      }
    }
  },
  themeConfig: {
    outline: [2, 3],
    search: {
      provider: 'local',
      options: { locales: { zh: { translations: {
        button: { buttonText: '搜索', buttonAriaLabel: '搜索教程' },
        modal: {
          displayDetails: '显示详细列表', resetButtonTitle: '清空搜索',
          backButtonTitle: '关闭搜索', noResultsText: '没有找到相关内容',
          footer: { selectText: '选择', navigateText: '切换', closeText: '关闭' }
        }
      } } } }
    }
  }
})
