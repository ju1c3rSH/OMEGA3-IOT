# OMEGA3-IOT 项目开发规范

## 1. JSON 字段命名规范
- 所有 JSON 字段名使用小写字母
- 不同单词间使用下划线 `_` 分割
- 保持一致性，避免混用驼峰命名

✅ 正确示例：
```go
    OwnerUUID    string `json:"owner_uuid"`
    InstanceUUID string `json:"instance_uuid"`
    AddTime      int    `json:"add_time"`
    LastSeen     int    `json:"last_seen"`
```
## 2. 设备的类型（Type）
``` text
    1 -> 基础测试用定位器
```

## 3. 每种类型的Properties
```text
    1 :{Location,Tempreature} //其中内容均使用Json表示（Altitude.....）
```
### 3.1.  PropertyMeta的规范 （非必要请勿缺省）
```text
    	Writable    bool
	Description string
	Unit        string
	Range       []int
	Format      string
	Enum        []string
	//TODO Required Type ?
```

## 4. 设备录入系统流程
```text
    设备注册遵循：设备开机通过网络/Lora的方式向主服务器进行报备注册。
    Lora需要通过网关。
    用户通过后续设备上显示的RegCode向服务器发送请求，将设备绑定至用户。
    
    流程：匿名注册——> 临时UUID和RegCode与VerifyCode，开辟当前UUID的Topic——>RegCode被使用，Topic下发指令，停止重置计时，并在服务端处转为正式UUID，存入数据库
    在存入instance表之后，服务端在broker发布一条广播：
    /data/device/{Device_UUID}/action 其含有GO_ON的信息，设备会在此前订阅这里，接收到之后开始工作。
    
```
### 4.1. RegCode规范
```textmate
    RegCode应为一串8位的含大小写字母、数字、符号的字符串。
    在设备注册后由主服务器生成，并下发给设备显示。
```

### 4.1.2 VerifyCode规范
```text
    VerifyCode应为一串16位的含大小写字母、数字、符号的字符串。
    VerifyHash为其取hash。
    Verify*是给设备上传数据时鉴权用的。
```

### 4.2. 设备注册失败超时处理
```text
    大概不会弄这种机制吧（）
    但是可以弄成创建临时注册记录，临时的uuid和ExpireTime。
    Expire的条件可以是：设备重启（临时的UUID存储在设备的易失性存储，如果Reg成功则转为持久UUID，写入设备的存储）或者到点没有接收到当前临时UUID的Topic的注册成功指令，则清理所有数据。
```


### 4.2.1 设备初步注册时的响应规范


## 响应结构
```json
{
  "code": 200,
  "device": {
    "expires_at": 1760858932,
    "id": 154,
    "reg_code": "A0WU@HG6",
    "type": 1,
    "uuid": "7b64cea8-ed24-4e73-b0a9-2af503bd4e69",
    "verify_code": "tOFX*mc8=V}?Cnh2"
  },
  "message": "Device Registered successfully"
}
```
## 字段说明

### 根级字段
- code (int): 状态码，200 表示成功
  - message (string): 人类可读的成功/错误信息
  - device (object): 设备注册详情对象

### device 对象字段
- id (int):
  设备在数据库中的唯一数字 ID
  - uuid (string):
    设备全局唯一标识符（UUID v4 格式）
  - reg_code (string):
    8 位注册码，由大小写字母、数字、符号组成，用于用户绑定设备
  - verify_code (string):
    16 位校验码，由大小写字母、数字、符号组成，用于设备上传数据时的身份验证
  - type (int):
    设备类型 ID（1 = 基础测试用定位器）
  - expires_at (int):
    凭证过期时间（Unix 时间戳，秒级），示例值 1760858932 对应 UTC 时间 2025-10-18 15:28:52

## 安全与使用提示
- verify_code 包含特殊字符（如 *, =, ?, }），必须原样存储和传输
  - reg_code 仅用于初始绑定，绑定成功后失效
  - expires_at 为临时凭证有效期，正式设备记录无过期时间
  - 所有字符串字段均区分大小写
### 5.服务器与设备的通信
```text
    移动网络设备使用MQ，Lora设备使用Lora双向联通。

```

## 5.1. 设备在MQTT中的规范
```text
    遵循 tcp:/data/device/{device_uuid}/的规范，其下属有：
    1.properties
    2.event
    3.action
```
## 5.1.1.  设备在通过MQTT传输Properties数据时的规范
```text
    使用json传输属于自己的props数据
    例：
```
```json
{
  "verify_code": "your_actual_16_char_verify_code_here",
  "timestamp": 1756882749,
  "data": {
    "properties": {
      "gps_location": {
        "meta": {
          "Writable": false,
          "Description": "GPS位置",
          "Format": "string"
        },
        "value": "39.9042,116.4074"
      },
      "battery_level": {
        "meta": {
          "Writable": true,
          "Description": "电量",
          "Unit": "%",
          "Range": [
            0,
            100
          ],
          "Format": "int"
        },
        "value": "85"
      }
    }
  }
}
```

## 5.1.2. 解析prop时的规范
```text

```

## 5.2.  HeartBeat检测online的方法



### 6.1.  CT01模块的使用
```text
    上电后执行命令顺序：
```

```text
    # 1. 配置 APN (使用默认)
AT+QICSGP=1,1,"","",""
# 模块应返回: OK

# 2. 设置客户端 ID (务必唯一)
AT+MQTTCLIENT="DXCT01_Test_Client_001"
# 模块应返回: OK

# 3. 配置服务器信息
AT+MIPSTART="yuyuko.food",1883,4
# 模块应返回: OK
# 然后可能返回: +MIPSTART:SUCCESS (表示配置成功)

# 4. 连接服务器
AT+MCONNECT=1,60
# 模块应返回: OK
# 然后可能返回: +MCONNECT:SUCCESS (表示连接成功)

# 5. 订阅主题
AT+MSUB="test/from_server",0
# 模块应返回: OK

# 6. 发布消息
#AT+MPUBEX="data/device/3eae4aed-5d6e-44f7-b59c-7ec6b6c43bc1/properties",1,0,268
#要发送的（其中268是字符数）
{
  "verify_code": "D8SGbdW}^:21.y12",
  "timestamp": 1756882749,
  "data": {
    "properties": {
      "gps_location": {
        
        "value": "39.9042,116.4074"
      },
      "battery_level": {
        
        "value": "66"
      }
    }
  }
}
# 7. (等待) 接收消息 (如果其他客户端向 test/from_server 发布了消息)
收←◆OK

+MPUBEX: SUCCESS

# 8. 断开连接
AT+MDISCONNECT
# 模块应返回: OK
```

## 6.2. 设备