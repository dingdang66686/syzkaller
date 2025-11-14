# Concurrent Execution Mechanism - Implementation Summary

## 问题描述 (Problem Statement)
需要给syzkaller增加并发执行机制，一个prog可以包含多个序列，executor拿到后使用多个线程分别执行这些序列，用于测试并发bug。

## 实现方案 (Implementation Solution)

### 1. 数据结构扩展 (Data Structure Extension)

在 `prog/prog.go` 中扩展了 `Prog` 结构：

```go
type Prog struct {
    Target   *Target
    Calls    []*Call      // 原有单序列字段（保持向后兼容）
    Sequences [][]*Call   // 新增多序列字段
    // ...
}
```

添加了辅助方法：
- `HasSequences()`: 检查是否为多序列程序
- `AllCalls()`: 获取所有调用（不论格式）
- `NumSequences()`: 获取序列数量

### 2. 生成机制 (Generation Mechanism)

在 `prog/generation.go` 中添加了 `GenerateConcurrent` 函数：

```go
func (target *Target) GenerateConcurrent(rs rand.Source, ncalls int, numSequences int, ct *ChoiceTable) *Prog
```

特点：
- 将总调用数平均分配到各个序列
- 每个序列独立生成，避免资源依赖
- 支持自定义序列数量

### 3. 序列化格式 (Serialization Format)

在 `prog/encodingexec.go` 中实现了新的序列化格式：

**多序列格式：**
```
<MULTISEQ_MARKER> <序列数量>
<序列1调用数> <调用...> <SEQSEP>
<序列2调用数> <调用...> <SEQSEP>
...
<序列N调用数> <调用...>
<EOF>
```

**单序列格式（向后兼容）：**
```
<调用数量> <调用...> <EOF>
```

添加的指令常量：
- `execInstrMultiSeq (-6)`: 多序列标记
- `execInstrSeqSep (-5)`: 序列分隔符

### 4. Executor执行机制 (Executor Execution)

在 `executor/executor.cc` 中实现了并发执行逻辑：

```cpp
// 新增辅助函数
execute_sequence()  // 执行单个序列

// 修改execute_one函数
void execute_one() {
    // 检测多序列标记
    if (is_multi_sequence && flag_threaded) {
        // 为每个序列调用execute_sequence
        // 利用现有线程池并发执行
    } else {
        // 原有单序列逻辑
    }
}
```

特点：
- 使用现有的线程池机制（kMaxThreads）
- 仅在threaded模式下启用并发
- 保持与现有代码的兼容性

### 5. 测试 (Testing)

在 `prog/concurrent_test.go` 中添加了完整测试套件：
- `TestGenerateConcurrent`: 测试程序生成
- `TestSerializeMultiSequence`: 测试序列化
- `TestAllCallsHelper`: 测试辅助方法
- `TestConcurrentMutation`: 测试变异操作

所有测试通过 ✅

### 6. 工具和文档 (Tools and Documentation)

创建了示例工具 `tools/syz-concurrentgen/concurrentgen.go`:
```bash
./bin/syz-concurrentgen -calls 20 -sequences 3
```

添加了完整文档 `docs/concurrent_execution.md`。

## 使用示例 (Usage Example)

### 生成并发程序
```go
// 生成20个调用分布在3个序列中
p := target.GenerateConcurrent(rs, 20, 3, ct)

// 检查是否为多序列
if p.HasSequences() {
    fmt.Printf("有 %d 个序列\n", len(p.Sequences))
    for i, seq := range p.Sequences {
        fmt.Printf("序列 %d: %d 个调用\n", i+1, len(seq))
    }
}

// 序列化执行
data, _ := p.SerializeForExec()
```

### 使用命令行工具
```bash
# 生成3个序列，总共15个调用
./bin/syz-concurrentgen -calls 15 -sequences 3 -count 1
```

## 技术特点 (Technical Features)

1. **向后兼容**: 完全兼容现有单序列程序
2. **高效执行**: 复用现有线程池，无额外开销
3. **灵活配置**: 支持自定义序列数量和调用数
4. **完整测试**: 包含单元测试和集成测试
5. **文档齐全**: 提供使用指南和示例

## 应用场景 (Use Cases)

- 测试内核竞态条件 (kernel race conditions)
- 发现并发同步问题 (concurrency synchronization issues)
- 检测锁顺序问题 (lock ordering problems)
- 测试资源竞争 (resource contention)

## 文件清单 (File List)

修改的文件：
- `prog/prog.go` - 添加Sequences字段和辅助方法
- `prog/encodingexec.go` - 实现多序列序列化
- `prog/generation.go` - 添加GenerateConcurrent函数
- `executor/executor.cc` - 实现并发执行逻辑

新增文件：
- `prog/concurrent_test.go` - 测试套件
- `tools/syz-concurrentgen/concurrentgen.go` - 示例工具
- `docs/concurrent_execution.md` - 功能文档
- `CONCURRENT_EXECUTION_SUMMARY.md` - 实现总结（本文件）

## 验证状态 (Verification Status)

✅ 所有代码编译通过
✅ 单元测试全部通过
✅ 示例工具正常运行
✅ 保持向后兼容性
