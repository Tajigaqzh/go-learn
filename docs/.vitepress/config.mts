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
          { text: '第 2 章 · 变量、常量与基本类型', link: '/guide/go02-variables' },
          { text: '第 3 章 · 运算符与格式化输出', link: '/guide/go03-operators-fmt' },
          { text: '第 4 章 · 控制流', link: '/guide/go04-control-flow' },
          { text: '第 5 章 · 函数与闭包', link: '/guide/go05-functions' },
          { text: '第 6 章 · 指针、值与内存入门', link: '/guide/go06-pointers' },
          { text: '第 7 章 · 数组、切片与映射', link: '/guide/go07-slices-maps' },
          { text: '第 8 章 · 字符串、字节与 Unicode', link: '/guide/go08-strings' },
          { text: '第 9 章 · 结构体与方法', link: '/guide/go09-structs-methods' },
          { text: '第 10 章 · 接口与类型系统', link: '/guide/go10-interfaces' },
          { text: '第 11 章 · 错误处理', link: '/guide/go11-errors' },
          { text: '第 12 章 · 包、模块与依赖管理', link: '/guide/go12-modules' },
          { text: '第 13 章 · 泛型', link: '/guide/go13-generics' },
          { text: '第 14 章 · 反射', link: '/guide/go14-reflection' },
          { text: '第 15 章 · 标准库精讲（一）：时间、数学与排序', link: '/guide/go15-stdlib-time-sort' },
          { text: '第 16 章 · 标准库精讲（二）：正则、文本与模板', link: '/guide/go16-stdlib-text' },
          { text: '第 17 章 · 文件、路径与 IO', link: '/guide/go17-files-io' },
          { text: '第 18 章 · 序列化与配置', link: '/guide/go18-serde-config' },
          { text: '第 19 章 · 测试、基准与代码质量', link: '/guide/go19-testing' },
          { text: '第 20 章 · 并发基础', link: '/guide/go20-concurrency' },
          { text: '第 21 章 · 并发模式与陷阱', link: '/guide/go21-concurrency-patterns' },
        ],
      },
      {
        text: '并发与运行时篇',
        items: [
          { text: '第 22 章 · context 与生命周期管理', link: '/guide/go22-context' },
          { text: '第 23 章 · 运行时、调度与内存模型', link: '/guide/go23-runtime' },
          { text: '第 24 章 · 性能分析与优化', link: '/guide/go24-performance' },
        ],
      },
      {
        text: '工程与生态篇',
        items: [
          { text: '第 25 章 · 网络编程基础与 TCP/UDP', link: '/guide/go25-net' },
          { text: '第 26 章 · HTTP 服务端与 REST API', link: '/guide/go26-http-server' },
          { text: '第 27 章 · HTTP 客户端与外部服务', link: '/guide/go27-http-client' },
          { text: '第 28 章 · 数据库编程', link: '/guide/go28-database' },
          { text: '第 29 章 · 命令行工具、日志与配置', link: '/guide/go29-cli-logging' },
          { text: '第 30 章 · 常用第三方生态与选型', link: '/guide/go30-ecosystem' },
          { text: '第 31 章 · unsafe、cgo 与代码生成', link: '/guide/go31-unsafe-cgo' },
        ],
      },
      {
        text: '附录',
        items: [
          { text: '附录 A · Go 版本特性对照', link: '/appendix/versions' },
          { text: '附录 B · 常见错误速查', link: '/appendix/errors' },
          { text: '附录 C · 标准库常用包速查', link: '/appendix/stdlib' },
        ],
      },
    ],

    sidebar: [
      {
        text: '开始',
        items: [{ text: '学习路线与章节规划', link: '/guide/' }],
      },
      {
        text: '基础篇',
        items: [
          { text: '第 1 章 · 环境搭建与第一个 Go 程序', link: '/guide/go01-hello' },
          { text: '第 2 章 · 变量、常量与基本类型', link: '/guide/go02-variables' },
          { text: '第 3 章 · 运算符与格式化输出', link: '/guide/go03-operators-fmt' },
          { text: '第 4 章 · 控制流', link: '/guide/go04-control-flow' },
          { text: '第 5 章 · 函数与闭包', link: '/guide/go05-functions' },
          { text: '第 6 章 · 指针、值与内存入门', link: '/guide/go06-pointers' },
          { text: '第 7 章 · 数组、切片与映射', link: '/guide/go07-slices-maps' },
          { text: '第 8 章 · 字符串、字节与 Unicode', link: '/guide/go08-strings' },
          { text: '第 9 章 · 结构体与方法', link: '/guide/go09-structs-methods' },
          { text: '第 10 章 · 接口与类型系统', link: '/guide/go10-interfaces' },
          { text: '第 11 章 · 错误处理', link: '/guide/go11-errors' },
          { text: '第 12 章 · 包、模块与依赖管理', link: '/guide/go12-modules' },
        ],
      },
      {
        text: '进阶篇',
        items: [
          { text: '第 13 章 · 泛型', link: '/guide/go13-generics' },
          { text: '第 14 章 · 反射', link: '/guide/go14-reflection' },
          { text: '第 15 章 · 标准库精讲（一）：时间、数学与排序', link: '/guide/go15-stdlib-time-sort' },
          { text: '第 16 章 · 标准库精讲（二）：正则、文本与模板', link: '/guide/go16-stdlib-text' },
          { text: '第 17 章 · 文件、路径与 IO', link: '/guide/go17-files-io' },
          { text: '第 18 章 · 序列化与配置', link: '/guide/go18-serde-config' },
          { text: '第 19 章 · 测试、基准与代码质量', link: '/guide/go19-testing' },
        ],
      },
      {
        text: '并发与运行时篇',
        items: [
          { text: '第 20 章 · 并发基础', link: '/guide/go20-concurrency' },
          { text: '第 21 章 · 并发模式与陷阱', link: '/guide/go21-concurrency-patterns' },
          { text: '第 22 章 · context 与生命周期管理', link: '/guide/go22-context' },
          { text: '第 23 章 · 运行时、调度与内存模型', link: '/guide/go23-runtime' },
          { text: '第 24 章 · 性能分析与优化', link: '/guide/go24-performance' },
        ],
      },
      {
        text: '工程与生态篇',
        items: [
          { text: '第 25 章 · 网络编程基础与 TCP/UDP', link: '/guide/go25-net' },
          { text: '第 26 章 · HTTP 服务端与 REST API', link: '/guide/go26-http-server' },
          { text: '第 27 章 · HTTP 客户端与外部服务', link: '/guide/go27-http-client' },
          { text: '第 28 章 · 数据库编程', link: '/guide/go28-database' },
          { text: '第 29 章 · 命令行工具、日志与配置', link: '/guide/go29-cli-logging' },
          { text: '第 30 章 · 常用第三方生态与选型', link: '/guide/go30-ecosystem' },
          { text: '第 31 章 · unsafe、cgo 与代码生成', link: '/guide/go31-unsafe-cgo' },
        ],
      },
      {
        text: '附录',
        items: [
          { text: '附录 A · Go 版本特性对照', link: '/appendix/versions' },
          { text: '附录 B · 常见错误速查', link: '/appendix/errors' },
          { text: '附录 C · 标准库常用包速查', link: '/appendix/stdlib' },
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
