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
