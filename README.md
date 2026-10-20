# mempool

main.go为主函数（原来就有，新加一些代码来调用dump）

gc-hook为回调函数（新加文件）

gc-snapshot为快照（新加文件）  
 
mpool为mempool的主体实现（原来就有，新加一些插桩点）

instr为插桩函数的具体实现

todolist：

1.下一步让学生先人工插 free list，看收益，再决定是否需要自动识别，下周肯定会有定量数据。

2.与 MPool 类似的还有一部分用 sync.pool 封装实现，分析放在 11 月；MPool 有成效后，11 月看 sync.pool 情况，希望拿到优化空间和实际成效。