---
layout: home

hero:
  name: Go 学习笔记
  text: 由浅入深，一章一个主题
  tagline: 配套仓库里的 go-learn 练习工程，边读边跑 go run ./cmd/go-learn
  actions:
    - theme: brand
      text: 从第 1 章开始
      link: /guide/go01-hello
    - theme: alt
      text: 查看学习路线
      link: /guide/

features:
  - title: 每章都有可运行的代码
    details: 章节里的例子对应 internal/chapter/ 下的示例包，读完就能 go run 看到真实输出。
  - title: 由浅入深
    details: 从第一个 go run 讲起，再逐步走到接口、泛型、并发、运行时和工程化。
  - title: 只讲验证过的结论
    details: 涉及格式化细节、并发时序和性能数字的地方都标注了实测结果，避免记住想当然的规则。
---
