# PushServer - 企业级消息推送服务

[![Go Version](https://img.shields.io/badge/Go-1.19+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](https://github.com)

PushServer 是一个高性能、高可用的企业级消息推送服务，支持多平台消息推送、多种推送策略、智能故障转移和系统通知等功能。

## 🚀 核心特性

### 📱 多平台支持
- **飞书 (Feishu)** - 支持文本和卡片消息
- **钉钉 (DingTalk)** - 支持文本和卡片消息  
- **企业微信 (WeChat Work)** - 支持文本和卡片消息
- **邮件 (Email)** - 支持SMTP直接发送
- **系统通知 (System)** - 控制台、文件、系统日志、HTTP通知

### 🎯 智能推送策略
- **all** - 向所有启用的渠道发送消息
- **failover** - 渠道间故障转移，一个成功即停止
- **webhook_failover** - 每个渠道内webhook故障转移
- **mixed** - 混合策略：渠道间故障转移，渠道内全发送
- **platform** - 指定平台推送，忽略策略配置

### 🛡️ 高可用保障
- **系统通知最后防线** - 当所有推送渠道都失败时自动触发
- **任务状态追踪** - 实时查询推送任务状态和结果
- **智能重试机制** - 支持多种故障转移策略
- **并发控制** - 可配置的工作协程和并发限制

### ⚡ 高性能架构
- **异步队列处理** - 基于Go协程的高并发处理
- **内存任务管理** - 高效的任务状态管理
- **连接池复用** - HTTP客户端连接复用
- **批量处理支持** - 可配置的批处理大小

## 📦 快速开始

### 环境要求
- Go 1.19+
- 支持的操作系统：Windows、Linux、macOS

### 安装部署

#### 1. 克隆项目
```bash
git clone https://github.com/your-org/PushServer.git
cd PushServer
```

#### 2. 安装依赖
```bash
go mod tidy
```

#### 3. 配置文件
复制配置模板并修改：
```bash
cp config/config.yaml.example config/config.yaml
```

#### 4. 启动服务
```bash
# 开发模式
go run main.go

# 生产模式
go build -o pushserver main.go
./pushserver
```

#### 5. Docker部署
```bash
# 构建镜像
docker build -t pushserver .

# 启动服务
docker-compose up -d
```

### 验证部署
```bash
# 健康检查
curl http://localhost:8080/health

# 测试推送
curl -X POST http://localhost:8080/api/v1/push \
  -H "Content-Type: application/json" \
  -d '{
    "recipient_alias": "ops_alert",
    "type": "info",
    "strategy": "failover",
    "style": "text",
    "content": {
      "title": "测试消息",
      "msg": "这是一条测试消息"
    }
  }'
```

## ⚙️ 配置说明

### 服务配置
```yaml
server:
  port: 8080              # 服务端口
  host: "0.0.0.0"         # 监听地址
  mode: "debug"           # 运行模式: debug/release/test
```

### 日志配置
```yaml
log:
  level: "info"           # 日志级别: debug/info/warn/error
  format: "json"          # 日志格式: json/text
  output: "file"          # 输出方式: stdout/file
  file_path: "log/app.log" # 日志文件路径
```

### 队列配置
```yaml
queue:
  worker_count: 50                    # 工作协程数量
  buffer_size: 10000                  # 队列缓冲区大小
  timeout: 10                         # 推送超时时间(秒)
  max_concurrent_per_platform: 20     # 每个平台最大并发数
  batch_size: 100                     # 批处理大小
```

### 推送平台配置
```yaml
recipients:
  ops_alert:                          # 接收者别名
    name: "运维告警组"                 # 接收者名称
    platforms:
      feishu:                         # 飞书平台
        enabled: true                 # 是否启用
        webhooks:
          - url: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
            secret: "your-secret"
            name: "告警机器人"
      dingtalk:                       # 钉钉平台
        enabled: true
        webhooks:
          - url: "https://oapi.dingtalk.com/robot/send?access_token=xxx"
            secret: "SEC..."
            name: "告警机器人"
      wechat:                         # 企业微信平台
        enabled: true
        webhooks:
          - url: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx"
            secret: ""
            name: "告警群"
      email:                          # 邮件平台
        enabled: true
        recipients:
          - email: "admin@example.com"
            name: "管理员邮箱"
      system:                         # 系统通知平台
        enabled: true
        notifications:
          - type: "console"           # 控制台通知
            name: "控制台通知"
          - type: "file"              # 文件通知
            name: "文件通知"
          - type: "http"              # HTTP通知
            name: "HTTP通知"
            url: "http://localhost:9090/webhook/system-notify"
```

### 邮件配置
```yaml
email:
  smtp_host: "smtp.gmail.com"         # SMTP服务器地址
  smtp_port: 587                      # SMTP端口
  username: "your-email@gmail.com"    # SMTP用户名
  password: "your-app-password"       # SMTP密码
  from: "noreply@example.com"         # 发件人邮箱
```

### 全局系统通知配置
```yaml
system:
  enabled: true                       # 启用系统通知作为最后防线
  notifications:
    - type: "console"                 # 控制台通知
      name: "控制台通知"
    - type: "file"                    # 文件通知
      name: "文件通知"
    - type: "http"                    # HTTP通知
      name: "HTTP通知"
      url: "http://your-webhook-server.com/api/notifications"
```

## 🔧 推送策略详解

### 1. all策略
向所有启用的渠道发送消息，不管成功失败都会发送到每个渠道。

**使用场景**：重要通知需要多渠道覆盖

### 2. failover策略
渠道间故障转移，按配置顺序尝试，一个成功即停止。

**使用场景**：优先级推送，有主备渠道

### 3. webhook_failover策略
每个渠道内的webhook故障转移，渠道间并行执行。

**使用场景**：每个平台有多个webhook地址

### 4. mixed策略
混合策略：渠道间故障转移 + 渠道内全发送。

**使用场景**：既要保证送达又要渠道内冗余

### 5. platform策略
指定平台推送，忽略strategy参数，只在指定平台内推送直到成功。

**使用场景**：明确知道要使用哪个平台

## 🛡️ 系统通知机制

### 最后防线机制
当所有配置的推送渠道都失败时，系统会自动触发系统通知作为最后防线，确保重要消息不丢失。

### 四种通知方式

#### 1. 控制台通知 (console)
直接输出到控制台，带有醒目的分隔线和图标，适合开发和调试。

#### 2. 文件通知 (file)
保存到 `notifications/` 目录，文件名格式：`system_notify_20250804_015102.txt`，包含完整的通知详情。

#### 3. 系统日志 (syslog)
写入到应用日志系统，根据消息类型选择日志级别，便于日志收集和监控。

#### 4. 内部存储通知 (http)
将通知存储到系统内部，用户可通过HTTP API接口查询和管理这些通知。

**内部存储通知特性**：
- 🗄️ **内存存储**：通知存储在内存中，重启后清空
- 🔍 **API查询**：提供完整的REST API进行通知管理
- 📊 **状态管理**：支持已读/未读状态管理
- 📈 **统计信息**：提供详细的通知统计数据
- 🗑️ **批量操作**：支持批量标记已读、删除等操作

**存储的通知数据格式**：
```json
{
  "id": "notify_20250804090748_666666",
  "title": "通知标题",
  "message": "通知内容",
  "type": "error|warning|info",
  "strategy": "all|failover|webhook_failover|mixed",
  "style": "text|card",
  "source": "PushServer-SystemNotification",
  "task_id": "关联的推送任务ID",
  "reason": "通知产生原因",
  "created_at": "2025-08-04T09:07:48Z",
  "status": "unread|read"
}
```

### 双重使用模式
- **自动触发**：作为最后防线，当所有渠道都失败时自动触发
- **主动指定**：用户可以通过 `platform: "system"` 主动使用系统通知

## 📊 监控和运维

### 健康检查
```bash
GET /health
```

### 任务状态查询
```bash
GET /api/v1/task/{task_id}
```

### 日志文件
- `log/app.log` - 应用日志
- `log/error.log` - 错误日志
- `log/info.log` - 信息日志
- `log/debug.log` - 调试日志

### 系统通知文件
- `notifications/` - 系统通知文件目录

## 🔌 API接口文档

### 基础信息
- **Base URL**: `http://localhost:8080`
- **Content-Type**: `application/json`
- **字符编码**: UTF-8

### 1. 健康检查

#### 接口描述
检查服务运行状态

#### 请求信息
- **URL**: `/health`
- **Method**: `GET`
- **参数**: 无

#### 响应示例
```json
{
  "status": "ok",
  "timestamp": "2025-01-04T01:30:00Z",
  "version": "1.0.0"
}
```

### 2. 消息推送

#### 接口描述
发送消息到指定的推送平台

#### 请求信息
- **URL**: `/api/v1/push`
- **Method**: `POST`
- **Content-Type**: `application/json`

#### 请求参数
| 参数名 | 类型 | 必填 | 描述 | 示例值 |
|--------|------|------|------|--------|
| recipient_alias | string | 是 | 接收者别名，对应配置文件中的recipients | "ops_alert" |
| type | string | 是 | 消息类型 | "info", "warning", "error" |
| strategy | string | 否 | 推送策略，platform参数存在时忽略 | "all", "failover", "webhook_failover", "mixed" |
| platform | string | 否 | 指定推送平台，存在时忽略strategy | "feishu", "dingtalk", "wechat", "email", "system" |
| style | string | 是 | 消息样式 | "text", "card" |
| content | object | 是 | 消息内容 | 见下方content对象 |

#### content对象
| 参数名 | 类型 | 必填 | 描述 | 示例值 |
|--------|------|------|------|--------|
| title | string | 是 | 消息标题 | "系统告警" |
| msg | string | 是 | 消息内容 | "服务器CPU使用率过高" |

#### 请求示例

**基础推送**：
```json
{
  "recipient_alias": "ops_alert",
  "type": "warning",
  "strategy": "failover",
  "style": "card",
  "content": {
    "title": "系统告警",
    "msg": "服务器CPU使用率达到85%，请及时处理"
  }
}
```

**指定平台推送**：
```json
{
  "recipient_alias": "ops_alert",
  "type": "info",
  "platform": "feishu",
  "style": "text",
  "content": {
    "title": "部署通知",
    "msg": "应用版本v1.2.0部署完成"
  }
}
```

**系统通知推送**：
```json
{
  "recipient_alias": "ops_alert",
  "type": "error",
  "platform": "system",
  "style": "card",
  "content": {
    "title": "紧急故障通知",
    "msg": "数据库连接异常，需要立即处理"
  }
}
```

#### 响应示例

**成功响应**：
```json
{
  "code": 200,
  "message": "消息推送任务已创建",
  "data": {
    "task_id": "f029b7c4-e0a2-4a35-8fd0-247542bab4b6",
    "recipient": "运维告警组",
    "strategy": "failover",
    "style": "card",
    "created_at": "2025-01-04T01:30:00Z"
  }
}
```

**错误响应**：
```json
{
  "code": 400,
  "message": "请求参数错误",
  "data": {
    "error": "recipient_alias不能为空"
  }
}
```

### 3. 任务状态查询

#### 接口描述
查询推送任务的执行状态和结果

#### 请求信息
- **URL**: `/api/v1/task/{task_id}`
- **Method**: `GET`
- **参数**: 
  - `task_id` (path): 任务ID

#### 响应示例

**任务执行中**：
```json
{
  "code": 200,
  "message": "任务查询成功",
  "data": {
    "task_id": "f029b7c4-e0a2-4a35-8fd0-247542bab4b6",
    "status": "processing",
    "recipient": "运维告警组",
    "strategy": "failover",
    "style": "card",
    "created_at": "2025-01-04T01:30:00Z",
    "updated_at": "2025-01-04T01:30:05Z",
    "results": [
      {
        "platform": "email",
        "webhook": "运维邮箱",
        "status": "failed",
        "message": "SMTP服务器连接失败",
        "timestamp": "2025-01-04T01:30:05Z"
      }
    ]
  }
}
```

**任务完成**：
```json
{
  "code": 200,
  "message": "任务查询成功",
  "data": {
    "task_id": "f029b7c4-e0a2-4a35-8fd0-247542bab4b6",
    "status": "completed",
    "recipient": "运维告警组",
    "strategy": "failover",
    "style": "card",
    "created_at": "2025-01-04T01:30:00Z",
    "updated_at": "2025-01-04T01:30:10Z",
    "results": [
      {
        "platform": "email",
        "webhook": "运维邮箱",
        "status": "failed",
        "message": "SMTP服务器连接失败",
        "timestamp": "2025-01-04T01:30:05Z"
      },
      {
        "platform": "feishu",
        "webhook": "告警机器人",
        "status": "success",
        "message": "消息发送成功",
        "timestamp": "2025-01-04T01:30:08Z"
      }
    ]
  }
}
```

**任务不存在**：
```json
{
  "code": 404,
  "message": "任务不存在",
  "data": {
    "task_id": "invalid-task-id"
  }
}
```

### 4. 错误码说明

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 200 | 请求成功 | - |
| 400 | 请求参数错误 | 检查请求参数格式和必填字段 |
| 404 | 资源不存在 | 检查任务ID或接收者别名是否正确 |
| 500 | 服务器内部错误 | 查看服务器日志，联系管理员 |

### 5. 推送结果状态

| 状态 | 说明 |
|------|------|
| success | 推送成功 |
| failed | 推送失败 |
| processing | 推送处理中 |
| timeout | 推送超时 |

### 6. 系统通知管理接口

#### 6.1 获取通知列表

**接口描述**：获取系统通知列表，支持分页和状态筛选

**请求信息**：
- **URL**: `/api/v1/notifications`
- **Method**: `GET`
- **参数**:
  - `status` (可选): 通知状态筛选 - `unread`, `read`, `all`
  - `limit` (可选): 每页数量，默认50，最大1000
  - `offset` (可选): 偏移量，默认0

**响应示例**：
```json
{
  "code": 200,
  "message": "获取系统通知成功",
  "data": {
    "notifications": [
      {
        "id": "notify_20250804090748_666666",
        "title": "系统通知最后防线测试",
        "message": "测试当所有配置的推送渠道都失败时...",
        "type": "warning",
        "strategy": "all",
        "style": "card",
        "source": "PushServer-SystemNotification",
        "task_id": "",
        "reason": "系统通知存储",
        "created_at": "2025-08-04T09:07:48Z",
        "status": "unread"
      }
    ],
    "pagination": {
      "total": 1,
      "limit": 50,
      "offset": 0,
      "count": 1
    }
  }
}
```

#### 6.2 获取单个通知

**请求信息**：
- **URL**: `/api/v1/notifications/{id}`
- **Method**: `GET`

#### 6.3 标记通知为已读

**请求信息**：
- **URL**: `/api/v1/notifications/{id}/read`
- **Method**: `PUT`

#### 6.4 标记所有通知为已读

**请求信息**：
- **URL**: `/api/v1/notifications/read-all`
- **Method**: `PUT`

#### 6.5 删除通知

**请求信息**：
- **URL**: `/api/v1/notifications/{id}`
- **Method**: `DELETE`

#### 6.6 清空所有通知

**请求信息**：
- **URL**: `/api/v1/notifications`
- **Method**: `DELETE`

#### 6.7 获取通知统计

**请求信息**：
- **URL**: `/api/v1/notifications/statistics`
- **Method**: `GET`

**响应示例**：
```json
{
  "code": 200,
  "message": "获取通知统计成功",
  "data": {
    "statistics": {
      "total": 5,
      "unread": 2,
      "read": 3,
      "by_type": {
        "error": 2,
        "warning": 2,
        "info": 1
      }
    }
  }
}
```

### 7. 使用示例

#### cURL示例
```bash
# 发送告警消息
curl -X POST http://localhost:8080/api/v1/push \
  -H "Content-Type: application/json" \
  -d '{
    "recipient_alias": "ops_alert",
    "type": "error",
    "strategy": "all",
    "style": "card",
    "content": {
      "title": "数据库告警",
      "msg": "数据库连接数超过阈值，当前连接数：150/100"
    }
  }'

# 查询任务状态
curl http://localhost:8080/api/v1/task/f029b7c4-e0a2-4a35-8fd0-247542bab4b6

# 获取系统通知列表
curl http://localhost:8080/api/v1/notifications

# 获取未读通知
curl "http://localhost:8080/api/v1/notifications?status=unread&limit=10"

# 标记通知为已读
curl -X PUT http://localhost:8080/api/v1/notifications/notify_20250804090748_666666/read

# 获取通知统计
curl http://localhost:8080/api/v1/notifications/statistics
```

#### Python示例
```python
import requests
import json

# 发送推送请求
def send_notification(title, message, recipient="ops_alert", msg_type="info"):
    url = "http://localhost:8080/api/v1/push"
    payload = {
        "recipient_alias": recipient,
        "type": msg_type,
        "strategy": "failover",
        "style": "card",
        "content": {
            "title": title,
            "msg": message
        }
    }
    
    response = requests.post(url, json=payload)
    return response.json()

# 查询任务状态
def get_task_status(task_id):
    url = f"http://localhost:8080/api/v1/task/{task_id}"
    response = requests.get(url)
    return response.json()

# 获取系统通知列表
def get_notifications(status="all", limit=50, offset=0):
    url = f"http://localhost:8080/api/v1/notifications"
    params = {"status": status, "limit": limit, "offset": offset}
    response = requests.get(url, params=params)
    return response.json()

# 标记通知为已读
def mark_notification_read(notification_id):
    url = f"http://localhost:8080/api/v1/notifications/{notification_id}/read"
    response = requests.put(url)
    return response.json()

# 获取通知统计
def get_notification_statistics():
    url = "http://localhost:8080/api/v1/notifications/statistics"
    response = requests.get(url)
    return response.json()

# 使用示例
result = send_notification("系统通知", "服务部署完成")
print(f"任务ID: {result['data']['task_id']}")

# 查询状态
status = get_task_status(result['data']['task_id'])
print(f"任务状态: {status['data']['status']}")

# 获取系统通知
notifications = get_notifications(status="unread", limit=10)
print(f"未读通知数量: {notifications['data']['pagination']['total']}")

# 获取统计信息
stats = get_notification_statistics()
print(f"通知统计: {stats['data']['statistics']}")
```

#### JavaScript示例
```javascript
// 发送推送请求
async function sendNotification(title, message, recipient = 'ops_alert', type = 'info') {
  const response = await fetch('http://localhost:8080/api/v1/push', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      recipient_alias: recipient,
      type: type,
      strategy: 'failover',
      style: 'card',
      content: {
        title: title,
        msg: message
      }
    })
  });
  
  return await response.json();
}

// 查询任务状态
async function getTaskStatus(taskId) {
  const response = await fetch(`http://localhost:8080/api/v1/task/${taskId}`);
  return await response.json();
}

// 获取系统通知列表
async function getNotifications(status = 'all', limit = 50, offset = 0) {
  const params = new URLSearchParams({ status, limit, offset });
  const response = await fetch(`http://localhost:8080/api/v1/notifications?${params}`);
  return await response.json();
}

// 标记通知为已读
async function markNotificationRead(notificationId) {
  const response = await fetch(`http://localhost:8080/api/v1/notifications/${notificationId}/read`, {
    method: 'PUT'
  });
  return await response.json();
}

// 获取通知统计
async function getNotificationStatistics() {
  const response = await fetch('http://localhost:8080/api/v1/notifications/statistics');
  return await response.json();
}

// 使用示例
sendNotification('部署通知', '应用版本v1.2.0部署成功')
  .then(result => {
    console.log('任务ID:', result.data.task_id);
    return getTaskStatus(result.data.task_id);
  })
  .then(status => {
    console.log('任务状态:', status.data.status);
    // 获取系统通知
    return getNotifications('unread', 10);
  })
  .then(notifications => {
    console.log('未读通知数量:', notifications.data.pagination.total);
    // 获取统计信息
    return getNotificationStatistics();
  })
  .then(stats => {
    console.log('通知统计:', stats.data.statistics);
  });
```

## 🤝 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 🆘 支持

如果您在使用过程中遇到问题，可以通过以下方式获取帮助：

- 📧 邮件：support@example.com
- 🐛 问题反馈：[GitHub Issues](https://github.com/your-org/PushServer/issues)
- 📖 文档：[在线文档](https://docs.example.com)

## 🎯 路线图

### 🚀 已完成功能
- [x] 5大推送平台支持（飞书、钉钉、企业微信、邮件、系统通知）
- [x] 4种智能推送策略
- [x] 系统通知内部存储功能
- [x] 完整的通知管理API
- [x] 高并发队列处理
- [x] 任务状态追踪
- [x] 多种系统通知方式（控制台、文件、日志、内部存储）

### 📋 计划中功能
- [ ] 支持更多推送平台（Slack、Teams等）
- [ ] Web管理界面
- [ ] 消息模板管理
- [ ] 推送统计和分析仪表板
- [ ] 消息去重机制
- [ ] 定时推送功能
- [ ] 消息优先级队列
- [ ] 集群部署支持
- [ ] 系统通知持久化存储（数据库）
- [ ] 通知订阅和过滤功能
- [ ] 推送失败重试策略优化
- [ ] 消息加密传输

---

**PushServer** - 让消息推送更简单、更可靠！ 🚀