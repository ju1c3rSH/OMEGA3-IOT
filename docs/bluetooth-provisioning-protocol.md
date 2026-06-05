# 蓝牙配网通讯协议

> 版本: 1.0.0 | 状态: Draft

## 概述

本文档定义了 OMEGA3-IOT 平台中 **蓝牙配网 (Bluetooth Provisioning)** 的通讯协议。适用于初始无网络的设备，通过 BLE 与手机 App 建立连接后，接收 WiFi/蜂窝网络配置信息，完成网络接入和云端注册。

### 适用场景

- 设备首次开机，无任何网络连接
- 手机 App 通过 BLE 扫描发现设备并建立连接
- App 将网络凭证（WiFi SSID/密码等）通过蓝牙下发给设备
- 设备连接网络后自行注册到云端

### 参考实现

- [乐鑫 Blufi 协议](https://github.com/espressif/esp-idf/tree/master/components/blufi)
- [ESP-IDF WiFi Provisioning](https://docs.espressif.com/projects/esp-idf/en/stable/esp32/api-reference/provisioning/wifi_provisioning.html)

---

## BLE 基础约束

| 约束 | 值 | 说明 |
|------|------|------|
| 默认 MTU | 20 字节 | BLE 4.2 之前 |
| 协商后 MTU | 244~512 字节 | BLE 4.2+，连接后协商 |
| GATT 特征值上限 | 512 字节 | 单次读写上限 |
| 连接间隔 | 7.5ms ~ 4s | 影响传输速率与功耗 |

**角色分配：**
- **设备** → BLE Peripheral（被连接方），广播配网服务
- **手机 App** → BLE Central（主动连接方）

---

## GATT 服务定义

```
Service UUID: 0x1820 (临时自定义，正式版分配正式 UUID)
├── Characteristic: 0x2C01 (Write)     ← App 写入数据到设备
│   Properties: Write, WriteNoResponse
│   Security: Encryption recommended
├── Characteristic: 0x2C02 (Notify)    ← 设备向 App 推送数据
│   Properties: Notify
│   Security: Encryption recommended
└── Characteristic: 0x2C03 (Read)      ← App 读取设备基础信息
    Properties: Read
    Value: 设备 UUID (16B) + 设备类型ID (2B) + 协议版本 (1B)
```

---

## 协议帧格式

每个 BLE 数据包封装为一个协议帧。超过 MTU 限制的数据自动分包传输。

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|    Header     |     Type      |    SeqNum     |   TotalSeq    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           Length (LE16)        |                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               +
|                         Payload (变长)                         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### 字段说明

| 字段 | 偏移 | 大小 | 说明 |
|------|------|------|------|
| **Header** | 0 | 1B | 协议魔数 `0xA3`（ASCII: OMEGA3 标识） |
| **Type** | 1 | 1B | 消息类型，见 [消息类型表](#消息类型) |
| **SeqNum** | 2 | 1B | 当前分包序号，从 `0` 开始 |
| **TotalSeq** | 3 | 1B | 总分包数。单包消息为 `1` |
| **Length** | 4 | 2B | Payload 有效字节数，小端序 (LE) |
| **Payload** | 6 | 变长 | 有效载荷，最大 = MTU - 6 字节 |

### 帧头常量

```c
#define PROTO_HEADER        0xA3
#define PROTO_HEADER_SIZE   6
#define PROTO_MAX_PAYLOAD   (MTU - PROTO_HEADER_SIZE)  // 通常 238 字节 (MTU=244)
```

---

## 消息类型

### 握手阶段

| 值 | 名称 | 方向 | Payload 说明 |
|------|------|------|------|
| `0x01` | `HANDSHAKE_REQ` | App → 设备 | 协议版本 (1B) + App 标识 (可选) |
| `0x02` | `HANDSHAKE_ACK` | 设备 → App | 协议版本 (1B) + 设备 UUID (16B) + 设备类型ID (2B) + SN (变长) |

### 网络配置下发

| 值 | 名方 | 方向 | Payload 说明 |
|------|------|------|------|
| `0x10` | `WIFI_CONFIG` | App → 设备 | JSON: `{"ssid":"...","password":"..."}` 或二进制 TLV |
| `0x11` | `WIFI_CONFIG_ACK` | 设备 → App | 结果码 (1B): `0x00`=成功收到 |
| `0x12` | `CELLULAR_CONFIG` | App → 设备 | JSON: `{"apn":"...","user":"...","pass":"..."}` |
| `0x13` | `CELLULAR_CONFIG_ACK` | 设备 → App | 结果码 (1B) |

### 状态上报

| 值 | 名称 | 方向 | Payload 说明 |
|------|------|------|------|
| `0x20` | `WIFI_STATUS` | 设备 → App | 状态码 (1B) + 信号强度 (1B, 可选) |
| `0x21` | `CLOUD_STATUS` | 设备 → App | 状态码 (1B) + 设备注册信息 (JSON, 可选) |
| `0x22` | `PROVISIONING_DONE` | 设备 → App | 最终状态码 (1B) |

### 控制与维护

| 值 | 名称 | 方向 | Payload 说明 |
|------|------|------|------|
| `0xF0` | `PING` | App → 设备 | 时间戳 (4B, 可选) |
| `0xF1` | `PONG` | 设备 → App | 时间戳 (4B, 可选) |
| `0xFE` | `ERROR` | 双向 | 错误码 (1B) + 可选描述 |
| `0xFF` | `ABORT` | 双向 | 中止原因码 (1B) |

---

## 错误码

| 值 | 名称 | 说明 |
|------|------|------|
| `0x00` | `OK` | 成功 |
| `0x01` | `ERR_INVALID_FRAME` | 帧格式错误（魔数、长度等） |
| `0x02` | `ERR_SEQ_MISMATCH` | 分包序号不连续 |
| `0x03` | `ERR_CHECKSUM` | 校验失败 |
| `0x04` | `ERR_UNSUPPORTED_TYPE` | 不支持的消息类型 |
| `0x05` | `ERR_PROTO_VERSION` | 协议版本不兼容 |
| `0x10` | `ERR_WIFI_SSID_NOT_FOUND` | 扫描不到指定 SSID |
| `0x11` | `ERR_WIFI_AUTH_FAILED` | WiFi 密码错误 |
| `0x12` | `ERR_WIFI_TIMEOUT` | WiFi 连接超时 |
| `0x13` | `ERR_WIFI_DHCP_FAILED` | DHCP 获取 IP 失败 |
| `0x20` | `ERR_CLOUD_DNS` | 云端域名解析失败 |
| `0x21` | `ERR_CLOUD_CONNECT` | 云端连接失败 |
| `0x22` | `ERR_CLOUD_AUTH` | 云端注册/认证失败 |
| `0xFF` | `ERR_UNKNOWN` | 未知错误 |

---

## 完整配网流程

```
手机 App                                  设备 (BLE Peripheral)
   |                                         |
   |  ① BLE 扫描, 发现配网广播                 |  设备上电, 开始广播
   |                                         |
   |--- HANDSHAKE_REQ [0x01] -------------->|
   |    (proto_version=1)                    |
   |                                         |
   |<-- HANDSHAKE_ACK [0x02] ---------------|
   |    (proto_ver + UUID + typeID + SN)     |
   |                                         |
   |  ② App 校验设备合法性                     |
   |                                         |
   |--- WIFI_CONFIG [0x10] 分包1/N -------->|
   |--- WIFI_CONFIG [0x10] 分包2/N -------->|
   |     ...                                 |
   |--- WIFI_CONFIG [0x10] 分包N/N -------->|
   |                                         |
   |<-- WIFI_CONFIG_ACK [0x11] -------------|
   |    (0x00 = OK)                          |
   |                                         |
   |  ③ 设备断开 BLE, 尝试连接 WiFi            |
   |                                         |
   |<-- WIFI_STATUS [0x20] -----------------|  (设备重连 BLE 或通过广播)
   |    (0x00=成功, 其他=失败)                  |
   |                                         |
   |  ④ 设备连接云端, 完成注册                  |
   |                                         |
   |<-- CLOUD_STATUS [0x21] ----------------|
   |    (0x00=注册成功 + 设备信息JSON)          |
   |                                         |
   |<-- PROVISIONING_DONE [0x22] -----------|
   |    (0x00=全部完成)                        |
   |                                         |
   |  ⑤ App 调用 BindDeviceByRegCode         |
   |     (bind_by = 0 = Bluetooth)           |
```

### 各阶段超时

| 阶段 | 超时 | 说明 |
|------|------|------|
| 握手 | 10s | 连接后 10s 内未完成握手则断开 |
| 配置下发 | 30s | 单次配网操作的最大时长 |
| WiFi 连接 | 45s | 设备尝试连接 WiFi 的等待时间 |
| 云端注册 | 60s | 设备注册到云端的最大等待时间 |
| 整体流程 | 180s | 从连接到配网完成的总超时 |

---

## 分包与重组

当 Payload 超过单包容量时，发送端自动分包，接收端按 `SeqNum` 顺序重组。

### 分包规则

```
单包最大 Payload = MTU - 6 (帧头大小)

总包数 TotalSeq = ceil(总数据长度 / 单包最大Payload)

第 N 包的 Payload = 原始数据 [N*maxPayload : (N+1)*maxPayload]
```

### 发送端逻辑 (App 侧)

```
function SendPayload(msgType, data):
    maxPayload = MTU - 6
    totalSeq = ceil(len(data) / maxPayload)

    for seq = 0 to totalSeq-1:
        chunk = data[seq*maxPayload : (seq+1)*maxPayload]
        frame = Frame(0xA3, msgType, seq, totalSeq, len(chunk), chunk)
        bleWrite(frame)
        waitACK(seq, timeout=5s)  // 超时重传，最多 3 次
```

### 接收端逻辑 (设备侧)

```
expectedSeq = 0
buffer = []

function OnFrameReceived(frame):
    if frame.header != 0xA3:
        sendError(ERR_INVALID_FRAME)
        return

    if frame.seqNum != expectedSeq:
        sendError(ERR_SEQ_MISMATCH)
        return

    buffer.append(frame.payload)
    expectedSeq++

    if frame.seqNum == frame.totalSeq - 1:
        // 所有分包到齐
        completeData = buffer.concat()
        processMessage(frame.type, completeData)
        buffer = []
        expectedSeq = 0
```

### 流控与重传

- 每个分包发送后等待 ACK（`WIFI_CONFIG_ACK` 或单独的 ACK 帧）
- ACK 超时: 5 秒，最多重传 3 次
- 3 次重传失败后发送 `ABORT` 帧并断开连接

---

## 安全机制

### 1. 传输加密

WiFi 密码等敏感数据 **不得明文传输**。

**推荐方案：**
- 使用 BLE 配对绑定（LE Secure Connections）建立加密链路
- GATT 特征值设置为需要加密读写
- 额外的应用层 AES-128-CTR 加密（可选，密钥通过 ECDH 协商）

### 2. 设备认证

```
HANDSHAKE_ACK Payload 结构:
┌──────────┬──────────┬──────────┬──────────┬──────────┐
│ ProtoVer │ DeviceUUID (16B) │ TypeID   │ SN Length│  SN ...  │
│  (1B)    │                  │  (2B)    │  (1B)    │  (变长)  │
└──────────┴──────────┴──────────┴──────────┴──────────┘
```

App 端验证：
- `DeviceUUID` 是否在已知的合法设备列表中（首次配网可跳过）
- `TypeID` 是否在 `GlobalDeviceTypeManager` 中存在

### 3. 防重放

- 每次握手生成随机 Nonce（可选）
- 协议帧中的 `SeqNum` 本身提供了基本的防重放保护

### 4. 超时保护

每个阶段都有独立超时，超时后设备自动：
1. 清除已接收的不完整数据
2. 断开 BLE 连接
3. 清除内存中的 WiFi 凭证
4. 重新开始广播

---

## Payload 编码格式

### WiFi 配置 (JSON 格式)

```json
{
    "ssid": "MyWiFi",
    "password": "p@ssw0rd",
    "bssid": "AA:BB:CC:DD:EE:FF",
    "security": "wpa2"
}
```

| 字段 | 必需 | 说明 |
|------|------|------|
| `ssid` | ✅ | WiFi 名称 |
| `password` | ✅ | WiFi 密码 |
| `bssid` | ❌ | 指定 AP 的 MAC 地址（多 AP 同名时使用） |
| `security` | ❌ | 加密类型: `open`, `wep`, `wpa`, `wpa2`, `wpa3` |

### 蜂窝网络配置 (JSON 格式)

```json
{
    "apn": "cmnet",
    "user": "",
    "password": "",
    "auth_type": "none"
}
```

### 二进制 TLV 格式 (可选，嵌入式友好)

对于资源受限的设备，可使用 TLV 格式替代 JSON：

```
┌──────────┬──────────┬──────────┐
│   Tag    │  Length   │  Value   │
│  (1B)    │  (1B)    │  (NB)    │
└──────────┴──────────┴──────────┘
```

| Tag | 名称 | 值类型 | 说明 |
|------|------|--------|------|
| `0x01` | SSID | string | WiFi 名称 |
| `0x02` | Password | string | WiFi 密码 |
| `0x03` | BSSID | bytes[6] | AP MAC 地址 |
| `0x04` | Security | uint8 | 加密类型枚举 |
| `0x10` | APN | string | 蜂窝 APN |
| `0x11` | Cellular User | string | 蜂窝用户名 |
| `0x12` | Cellular Pass | string | 蜂窝密码 |

---

## 与 OMEGA3-IOT 后端集成

### 绑定方式标记

设备通过蓝牙配网成功后，App 调用绑定接口时需指定 `bind_by`:

```json
POST /api/v1/users/bindDeviceByRegCode
{
    "reg_code": "AB12CD34",
    "device_nick": "客厅传感器",
    "bind_by": 0
}
```

| 值 | 常量 | 含义 |
|------|------|------|
| `0` | `BindByBluetooth` | 蓝牙配网 |
| `1` | `BindByCellular` | 蜂窝网络 |
| `2` | `BindByWiFi` | WiFi 直连 |

### 配网后设备行为

1. 设备连接 WiFi 成功后，通过 HTTP 调用 `POST /api/v1/device/deviceRegisterAnon` 获取 `reg_code`
2. 设备将 `reg_code` 通过 BLE 返回给 App
3. App 调用 `BindDeviceByRegCode` 完成绑定
4. 设备收到云端的 `enable_properties_upload` 指令后开始上报数据

---

## 附录: 帧示例

### 握手请求 (App → 设备)

```
A3 01 00 01 01 00 01
│  │  │  │  │  │  └─ Payload: proto_version = 1
│  │  │  │  │  └──── Length = 1
│  │  │  │  └─────── TotalSeq = 1
│  │  │  └────────── SeqNum = 0
│  │  └───────────── Type = HANDSHAKE_REQ
│  └──────────────── Header = 0xA3
```

### WiFi 配置单包 (App → 设备, MTU=244)

```
A3 10 00 01 20 00 {"ssid":"Home","password":"12345678"}
│  │  │  │  │  └─ Length = 32 (0x0020)
│  │  │  │  └──── TotalSeq = 1
│  │  │  └─────── SeqNum = 0
│  │  └────────── Type = WIFI_CONFIG
│  └───────────── Header = 0xA3
```

### WiFi 配置多包 (App → 设备, MTU=20, 大数据)

```
第 1 包: A3 10 00 03 0E 00 {"ssid":"Home",
第 2 包: A3 10 01 03 0E 00 "password":"very_l
第 3 包: A3 10 02 03 0A 00 ong_pass"}
```

### 错误帧 (设备 → App)

```
A3 FE 00 01 01 00 11
│  │  │  │  │  └─ Payload: ERR_WIFI_AUTH_FAILED
│  │  │  │  └──── Length = 1
│  │  │  └─────── TotalSeq = 1
│  │  └────────── SeqNum = 0
│  └───────────── Type = ERROR
└──────────────── Header = 0xA3
```