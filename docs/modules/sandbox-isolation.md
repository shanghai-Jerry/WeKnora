# 沙箱隔离机制

## 概述

沙箱隔离机制是一个**可插拔的安全执行层**，用于在隔离环境中运行不可信的脚本(如用户生成的代码、AI 生成的脚本等)，防止恶意代码对宿主机造成损害。

## 架构设计

```
Manager (编排层)
  ├── ScriptValidator (安全验证器)
  └── Sandbox (执行层)
        ├── DockerSandbox (强隔离 — 生产推荐)
        ├── LocalSandbox (基本隔离 — 开发/回退)
        └── disabledSandbox (禁用 — 总是拒绝)
```

### 核心接口

定义在 `internal/sandbox/sandbox.go`:

- **`Sandbox` 接口** — 执行原语: `Execute(ctx, config)`, `Cleanup(ctx)`, `Type()`, `IsAvailable(ctx)`
- **`Manager` 接口** — 统一编排: `Execute(ctx, config)`, `Cleanup(ctx)`, `GetSandbox()`, `GetType()`

## 执行流程

```
用户调用 manager.Execute()
  ↓
ScriptValidator 安全验证 (除非 SkipValidation=true)
  ├── ValidateScript() — 检查脚本内容
  ├── ValidateArgs() — 检查参数注入
  └── ValidateStdin() — 检查 stdin 注入
  ↓ (通过)
底层 Sandbox.Execute()
  ├── Docker: docker run --rm [安全标志] <镜像> <解释器> <脚本>
  └── Local:  exec.CommandContext(解释器, 脚本路径, 参数...)
  ↓
返回 ExecuteResult (Stdout, Stderr, ExitCode, Duration, Killed)
```

## 安全机制（多层防御）

| 层级 | 机制 | 适用范围 |
|------|------|----------|
| L1: 脚本静态分析 | 检测危险命令、base64 解码、反弹 shell、`os.system`、`eval`、pickle 反序列化等 ~40+ 规则 | 全部 |
| L2: 参数验证 | 检测 `$()`、反引号、`&&`、`;`、`|`、`../` 路径遍历等 | 全部 |
| L3: stdin 验证 | 检测 stdin 中嵌入的 shell 命令模式 | 全部 |
| L4: 解释器白名单 | 仅允许 python3/bash/node/ruby/perl/php | Local |
| L5: 命令白名单 | 仅允许 ls/echo/cat/grep 等基本命令 | Local |
| L6: 路径限制 | 脚本路径必须在 AllowedPaths 内 | Local |
| L7: 环境变量过滤 | 阻止 `LD_PRELOAD`、`PYTHONPATH`、`NODE_OPTIONS` 等 | Local |
| L8: 进程组隔离 | 超时时向整个进程组发 SIGKILL | Local |
| L9: 非 root 用户 | `--user 1000:1000` | Docker |
| L10: 无 Linux Capability | `--cap-drop ALL` | Docker |
| L11: 只读根文件系统 | `--read-only` + 临时 tmpfs `/tmp` | Docker |
| L12: 资源限制 | 内存限制(默认 256MB)、CPU 限制(默认 1 核)、PIDs 限制(100) | Docker |
| L13: 网络隔离 | `--network none` (除非 AllowNetwork=true) | Docker |
| L14: 禁止权限提升 | `--security-opt no-new-privileges` | Docker |
| L15: 超时控制 | 上下文超时 + 强制杀死 | 全部 |
| L16: 镜像预拉取 | 初始化时异步 `docker pull` | Docker |

## 如何使用

### 1. 配置

```go
// 使用默认配置 (Local 模式, 回退启用, 60s 超时)
config := sandbox.DefaultConfig()

// 自定义 Docker 配置
config := &sandbox.Config{
    Type:            sandbox.SandboxTypeDocker,
    FallbackEnabled: true,          // Docker 不可用时回退到 Local
    DefaultTimeout:  30 * time.Second,
    DockerImage:     "wechatopenai/weknora-sandbox:latest",
    MaxMemory:       512 * 1024 * 1024, // 512MB
    MaxCPU:          2.0,
}

sandbox.ValidateConfig(config)
```

### 2. 创建 Manager

```go
// 完整配置
manager, err := sandbox.NewManager(config)

// 便捷方式
manager, err := sandbox.NewManagerFromType("docker", true, "")

// 禁用沙箱（所有执行返回错误）
manager := sandbox.NewDisabledManager()
```

### 3. 执行脚本

```go
execCfg := &sandbox.ExecuteConfig{
    Script:          "/path/to/script.py",
    Args:            []string{"--input", "data.txt"},
    WorkDir:         "/path/to/work",
    Timeout:         10 * time.Second,
    Stdin:           "input data",
    AllowNetwork:    false,
    ReadOnlyRootfs:  true,
    MemoryLimit:     128 * 1024 * 1024,
    CPULimit:        0.5,
    AllowedCommands: sandbox.DefaultConfig().AllowedCommands,
    AllowedPaths:    sandbox.DefaultConfig().AllowedPaths,
    SkipValidation:  false,
    Env: map[string]string{"MY_VAR": "value"},
}

result, err := manager.Execute(context.Background(), execCfg)
if err != nil { /* 处理错误 */ }

fmt.Println(result.GetOutput())  // Stdout
fmt.Println(result.ExitCode)     // 退出码
fmt.Println(result.IsSuccess())  // ExitCode == 0
```

### 4. 三种沙箱模式对比

| 特性 | Docker | Local | Disabled |
|------|--------|-------|----------|
| 隔离强度 | 高（容器级） | 低（进程级） | N/A |
| 依赖 | Docker daemon | 无 | 无 |
| 可用性 | 需 docker version 可用 | 总是可用 | 总是"可用" |
| 推荐场景 | 生产环境 | 开发/CI 回退 | 功能关闭 |
| 自动清理 | `--rm` | N/A | N/A |
| 资源限制 | 内存/CPU/PIDs | 仅超时 | N/A |

### 5. 关键设计决策

- **分层架构**: `Sandbox` 接口(执行)与 `Manager`(编排+验证)分离，后端可替换
- **安全优先**: 验证器在任何执行之前运行，除非显式设置 `SkipValidation=true`
- **回退策略**: Docker 不可用时优雅回退到 Local，保证开发体验
- **深度防御**: 多层安全机制叠加，不依赖单一防线
- **禁用模式**: 使用专用类型而非 nil，避免空指针风险
- **最小特权**: Docker 容器使用所有可用的安全加固选项

### 6. 相关文件

| 文件 | 说明 |
|------|------|
| `internal/sandbox/sandbox.go` | 接口定义、类型、常量、默认配置 |
| `internal/sandbox/manager.go` | Manager 实现、初始化、验证编排 |
| `internal/sandbox/docker.go` | Docker 容器沙箱实现 |
| `internal/sandbox/local.go` | 本地进程沙箱实现 |
| `internal/sandbox/validator.go` | 安全验证器（脚本/参数/stdin） |
| `internal/sandbox/sandbox_test.go` | 集成测试 |
| `internal/sandbox/validator_test.go` | 验证器测试 |
