import { defineConfig } from 'vitepress'

export default defineConfig({
  lang: 'zh-CN',
  title: 'Go 学习笔记',
  description: '按章节由浅入深地学 Go，配套 go-learn 练习工程',
  cleanUrls: true,

  // 部署在 GitHub Pages 的项目子路径下，必须和仓库名一致；
  // 换成用户名.github.io 这类用户主页仓库时要改成 '/'
  base: '/go-learn/',

  themeConfig: {
    nav: [
      { text: '首页', link: '/' },
      { text: '学习路线', link: '/guide/' },
      {
        text: '章节',
        items: [
          { text: '第 1 章 · 环境搭建与第一个 Go 程序', link: '/guide/go01-hello' },
        ],
      },
    ],

    // 章节按篇分组，每写完一章就把对应条目补成可点击链接。
    sidebar: [
      {
        text: '开始',
        items: [{ text: '学习路线与章节规划', link: '/guide/' }],
      },
      {
        text: '基础篇',
        items: [
          { text: '第 1 章 · 环境搭建与第一个 Go 程序', link: '/guide/go01-hello' },
        ],
      },
    ],

    outline: { level: [2, 3], label: '本页目录' },
    docFooter: { prev: '上一章', next: '下一章' },
    returnToTopLabel: '回到顶部',
    sidebarMenuLabel: '目录',
    darkModeSwitchLabel: '主题',

    search: {
      provider: 'local',
      options: {
        translations: {
          button: { buttonText: '搜索文档', buttonAriaLabel: '搜索文档' },
          modal: {
            noResultsText: '没有找到相关内容',
            resetButtonTitle: '清除查询条件',
            footer: {
              selectText: '选择',
              navigateText: '切换',
              closeText: '关闭',
            },
          },
        },
      },
    },
  },
})
