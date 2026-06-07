# Android 端 API 变更指南

> 本文档描述后端 API 的最新变更，Android 端需据此更新。

---

## 一、设备文件夹重命名（DeviceGroup → DeviceFolder）

### 1.1 变更原因

原「设备组（Device Group）」与「用户组（User Group）」语义混淆。现将设备维度的分组重命名为「设备文件夹（Device Folder）」，语义更清晰：

- **DeviceFolder** = 设备文件夹（单人设备组织工具，类似文件夹/标签）
- **UserGroup** = 用户团队（多人协作，有角色、邀请、策略）

### 1.2 接口变更对照表

| 操作 | 旧接口 | 新接口 |
|------|--------|--------|
| 创建文件夹 | `POST /devices/groups/create_group` | `POST /devices/folders` |
| 设备加入文件夹 | `POST /devices/{uuid}/join_group` | `POST /devices/{uuid}/folders` |
| 设备移出文件夹 | `POST /devices/{uuid}/quit_group` | `DELETE /devices/{uuid}/folders/{folder_uuid}` |
| 获取文件夹设备 | `GET /devices/groups/{uuid}/members` | `GET /devices/folders/{uuid}/devices` |
| 删除文件夹 | `POST /devices/groups/{uuid}/dismiss_group` | `DELETE /devices/folders/{uuid}` |
| 我的文件夹列表 | `GET /users/me/device_groups` | `GET /users/me/device_folders` |

### 1.3 请求/响应变更

#### 创建文件夹

```
旧: POST /devices/groups/create_group  Body: {"name":"...", "description":"..."}
新: POST /devices/folders              Body: {"name":"...", "description":"..."}
```

响应字段变更：`group_uuid` → `folder_uuid`

```json
// 旧响应
{"group_uuid": "...", "name": "客厅设备", ...}
// 新响应
{"folder_uuid": "...", "name": "客厅设备", ...}
```

#### 设备加入文件夹

```
旧: POST /devices/{uuid}/join_group    Body: {"group_uuid": "..."}
新: POST /devices/{uuid}/folders       Body: {"folder_uuid": "..."}
```

响应字段变更：`group_uuid` → `folder_uuid`

#### 设备移出文件夹

```
旧: POST /devices/{uuid}/quit_group    Body: {"group_uuid": "..."}
新: DELETE /devices/{uuid}/folders/{folder_uuid}
```

**变更**：从 POST + Body 改为 DELETE + URL 参数，RESTful 更规范。

#### 获取文件夹设备

```
旧: GET /devices/groups/{uuid}/members
新: GET /devices/folders/{uuid}/devices
```

响应字段变更：`members` → `devices`，`message` 改为 "Folder devices retrieved successfully"

#### 删除文件夹

```
旧: POST /devices/groups/{uuid}/dismiss_group
新: DELETE /devices/folders/{uuid}
```

**变更**：从 POST 改为 DELETE。

#### 我的文件夹列表

```
旧: GET /users/me/device_groups
新: GET /users/me/device_folders
```

响应字段变更：`groups` → `folders`，`message` 改为 "Folders retrieved successfully"

### 1.4 Android 端修改清单

```kotlin
// 1. API 接口定义 (ApiService.kt / Retrofit interface)

// 旧
@POST("devices/groups/create_group")
suspend fun createDeviceGroup(@Body body: CreateGroupRequest): Response<...>

// 新
@POST("devices/folders")
suspend fun createDeviceFolder(@Body body: CreateFolderRequest): Response<...>

// 旧
@POST("devices/{uuid}/join_group")
suspend fun joinGroup(@Path("uuid") uuid: String, @Body body: JoinGroupRequest): Response<...>

// 新
@POST("devices/{uuid}/folders")
suspend fun addDeviceToFolder(@Path("uuid") uuid: String, @Body body: AddToFolderRequest): Response<...>

// 旧
@POST("devices/{uuid}/quit_group")
suspend fun quitGroup(@Path("uuid") uuid: String, @Body body: QuitGroupRequest): Response<...>

// 新
@HTTP(method = "DELETE", path = "devices/{uuid}/folders/{folder_uuid}", hasBody = false)
suspend fun removeDeviceFromFolder(
    @Path("uuid") uuid: String,
    @Path("folder_uuid") folderUuid: String
): Response<...>

// 旧
@GET("devices/groups/{uuid}/members")
suspend fun getGroupMembers(@Path("uuid") uuid: String, ...): Response<...>

// 新
@GET("devices/folders/{uuid}/devices")
suspend fun getFolderDevices(@Path("uuid") uuid: String, ...): Response<...>

// 旧
@POST("devices/groups/{uuid}/dismiss_group")
suspend fun dismissGroup(@Path("uuid") uuid: String): Response<...>

// 新
@DELETE("devices/folders/{uuid}")
suspend fun deleteFolder(@Path("uuid") uuid: String): Response<...>

// 旧
@GET("users/me/device_groups")
suspend fun getMyDeviceGroups(...): Response<...>

// 新
@GET("users/me/device_folders")
suspend fun getMyDeviceFolders(...): Response<...>
```

```kotlin
// 2. 数据模型 (data class)

// 旧
data class DeviceGroup(val group_uuid: String, val name: String, ...)
data class JoinGroupRequest(val group_uuid: String)
data class QuitGroupRequest(val group_uuid: String)

// 新
data class DeviceFolder(val folder_uuid: String, val name: String, ...)
data class AddToFolderRequest(val folder_uuid: String)
// RemoveDeviceFromFolder 不需要 request body
```

```kotlin
// 3. ViewModel / Repository 中的变量名

// 旧
val deviceGroups = MutableLiveData<List<DeviceGroup>>()
fun createGroup(name: String, description: String) { ... }
fun joinGroup(groupUUID: String, deviceUUID: String) { ... }

// 新
val deviceFolders = MutableLiveData<List<DeviceFolder>>()
fun createFolder(name: String, description: String) { ... }
fun addDeviceToFolder(folderUUID: String, deviceUUID: String) { ... }
```

```kotlin
// 4. UI / 字符串资源

// 旧
<string name="device_group">设备组</string>
<string name="create_group">创建设备组</string>
<string name="join_group">加入组</string>
<string name="quit_group">退出组</string>
<string name="dismiss_group">解散组</string>
<string name="group_members">组成员</string>

// 新
<string name="device_folder">设备文件夹</string>
<string name="create_folder">创建文件夹</string>
<string name="add_to_folder">加入文件夹</string>
<string name="remove_from_folder">移出文件夹</string>
<string name="delete_folder">删除文件夹</string>
<string name="folder_devices">文件夹设备</string>
```


## 三、变更总结

### 3.1 接口变更清单

| 变更类型 | 内容 | 影响范围 |
|----------|------|----------|
| **重命名** | DeviceGroup → DeviceFolder | 6 个接口 |
| **新增** | Admin 管理后台 | 24 个接口 |
| **新增** | User Group 用户组 | 20 个接口 |

### 3.2 Android 端需要修改的文件

| 文件类型 | 修改内容 |
|----------|----------|
| `ApiService.kt` | 更新 6 个设备文件夹接口路径 |
| `DeviceGroup.kt` → `DeviceFolder.kt` | 重命名数据类，字段 `group_uuid` → `folder_uuid` |
| `DeviceGroupViewModel.kt` → `DeviceFolderViewModel.kt` | 重命名 ViewModel |
| `DeviceGroupRepository.kt` → `DeviceFolderRepository.kt` | 重命名 Repository |
| 布局文件 | 更新 UI 文本（组 → 文件夹） |
| 字符串资源 | 更新所有相关字符串 |

### 3.3 迁移优先级

1. **P0（必须）**: 设备文件夹接口重命名 — 否则旧接口将不可用
2. **P2（可选）**: Admin 后台管理接口 — 仅管理端使用
4. **P2（可选）**: User Group 用户组接口 — 新功能，按需接入
