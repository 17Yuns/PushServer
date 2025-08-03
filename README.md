# PushServer - 消息推送服务

一个高性能的消息推送服务，支持多平台消息转发，包括飞书、钉钉、企业微信等。

## 🚀 特性

- **多平台支持**: 支持飞书、钉钉、企业微信等主流办公平台
- **高并发处理**: 50个工作协程，10000缓冲区，支持高并发推送
- **多种推送策略**: 支持5种推送策略（all、failover、webhook_all、webhook_failover、mixed）
- **多消息类型**: 支持错误、警告、信息三种消息类型
- **多消息样式**: 支持文本和卡片两种消息样式
- **异步队列**: 基于Go协程的异步消息队列处理
- **任务状态管理**: 支持通过UUID查询推送任务状态
- **配置驱动**: 基于YAML配置文件的灵活配置
- **详细日志**: 分级日志记录，支持文件和控制台输出

## 📋 目录结构

```
PushServer/
├── config/                 # 配置文件目录
│   └── config.yaml         # 主配置文件
├── internal/               # 内部包目录
│   ├── config/            # 配置管理
│   ├── forwarder/         # 转发服务
│   ├── handler/           # HTTP处理器
│   ├── logger/            # 日志管理
│   ├── middleware/        # 中间件
│   ├── model/             # 数据模型
│   ├── platform/          # 平台实现
│   │   ├── feishu.go      # 飞书平台
│   │   ├── dingtalk.go    # 钉钉平台
│   │   ├── wechat.go      # 企业微信平台
│   │   └── platform.go   # 平台管理器
│   ├── pusher/            # 推送服务
│   ├── queue/             # 消息队列
│   ├── router/            # 路由管理
│   ├── server/            # 服务器
│   └── task/              # 任务管理
├── log/                   # 日志文件目录
├── scripts/               # 脚本目录
├── main.go               # 主程序入口
└── README.md             # 项目说明
```

## 🛠️ 安装和运行

### 环境要求

- Go 1.19+
- Windows/Linux/macOS

### 安装依赖

```bash
go mod tidy
```

### 配置文件

复制并修改配置文件：

```bash
cp config/config.yaml.example config/config.yaml
```

编辑 `config/config.yaml` 配置你的webhook地址和密钥。

### 运行服务

```bash
# 开发模式
go run main.go

# 编译运行
go build -o pushserver
./pushserver
```

服务默认运行在 `http://localhost:8080`

## 📖 配置说明

### 基本配置

```yaml
# 服务配置
server:
  port: 8080
  host: "0.0.0.0"
  mode: "debug"

# 日志配置
log:
  level: "info"
  format: "json"
  output: "file"
  file_path: "log/app.log"

# 队列配置
queue:
  worker_count: 50
  buffer_size: 10000
  timeout: 10
```

### 接收者配置

```yaml
recipients:
  ops_alert:
    name: "运维告警组"
    platforms:
      feishu:
        enabled: true
        webhooks:
          - url: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
            secret: "your-secret"
            name: "主要告警群"
          - url: "https://open.feishu.cn/open-apis/bot/v2/hook/yyy"
            secret: "your-secret"
            name: "备用告警群"
```

## 🔧 推送策略

| 策略 | 说明 |
|------|------|
| `all` | 所有渠道都发送 |
| `failover` | 渠道间故障转移 |
| `webhook_all` | 每个渠道内所有webhook都发送 |
| `webhook_failover` | 每个渠道内webhook故障转移 |
| `mixed` | 渠道间故障转移，渠道内webhook全发送 |

## 📝 消息类型和样式

### 消息类型

- `error`: 错误消息（🔴 红色）
- `warning`: 警告消息（🟡 黄色）
- `info`: 信息消息（🔵 蓝色）

### 消息样式

- `text`: 纯文本消息
- `card`: 卡片消息（富文本）

## 🔍 监控和日志

### 日志文件

- `log/app.log`: 所有日志
- `log/error.log`: 错误日志
- `log/info.log`: 信息日志
- `log/debug.log`: 调试日志

### 健康检查

```bash
curl http://localhost:8080/health
```

## 🤝 贡献

欢迎提交Issue和Pull Request！

## 📄 许可证

MIT License

## 📞 支持

如有问题，请提交Issue或联系维护者。