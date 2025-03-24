# Watchman

Watchman 是一个用 Go 语言编写的文件备份监控工具，支持增量备份和定时任务。

## 功能特点

- 支持增量备份，只同步发生变化的文件
- 支持定时备份任务
- 支持按分钟间隔备份
- 实时显示备份进度
- 支持多个备份任务管理
- 使用 SHA256 校验确保文件完整性
- 保持文件修改时间

## 项目结构

```
.
├── cmd/            # 主程序入口
│   └── watchman/   # watchman 主程序
├── internal/       # 内部包
│   ├── backup/     # 备份相关实现
│   └── config/     # 配置相关实现
├── pkg/           # 可以被外部使用的包
└── Makefile       # 项目构建文件
```

## 安装

### 方法一：使用 Makefile（推荐）

1. 确保已安装 Go 1.16 或更高版本
2. 克隆项目
3. 安装依赖：
   ```bash
   make deps
   ```
4. 构建项目：
   ```bash
   make build
   ```
5. （可选）安装到系统：
   ```bash
   sudo make install
   ```

### 方法二：直接构建

1. 确保已安装 Go 1.16 或更高版本
2. 克隆项目
3. 编译项目：
   ```bash
   go build -o watchman cmd/watchman/main.go
   ```

## 开发

### 使用 Makefile 进行开发

1. 运行测试：
   ```bash
   make test
   ```

2. 开发模式（自动重新构建和运行）：
   ```bash
   make dev
   ```

3. 清理构建文件：
   ```bash
   make clean
   ```

4. 查看所有可用命令：
   ```bash
   make help
   ```

## 使用方法

### 启动守护进程

```bash
./watchman
```

### 添加备份任务

有两种方式添加备份任务：

1. 使用 cron 表达式（推荐用于复杂的定时需求）：
```bash
./watchman add <name> <source_path> <target_path> <schedule>
```
示例：
```bash
./watchman add mybackup /source/dir /target/dir "0 */1 * * * *"
```

2. 使用分钟间隔（推荐用于简单的定时需求）：
```bash
./watchman -n <minutes> add <name> <source_path> <target_path>
```
示例：
```bash
./watchman -n 30 add mybackup /source/dir /target/dir
```

注意：使用 `-n` 参数时，必须将其放在 `add` 命令之前。

cron 表达式格式：
```
秒 分 时 日 月 星期
```

### 列出所有备份任务

```bash
./watchman list
```

### 停止备份任务

```bash
./watchman stop <task_id>
```

### 删除备份任务

```bash
./watchman delete <task_id>
```

## 配置

配置文件默认保存在 `~/.watchman/config.json`，可以通过 `-config` 参数指定其他位置：

```bash
./watchman -config /path/to/config.json
```

## 注意事项

1. 确保有足够的权限访问源目录和目标目录
2. 建议定期检查备份日志
3. 对于大量文件的备份，建议使用相对宽松的备份间隔
4. 备份过程中请勿修改源文件
5. 使用 `-n` 参数时，备份任务会在每 N 分钟的第 0 秒执行
6. 所有的全局参数（如 `-n` 和 `-config`）必须放在命令之前