/**
 * OMEGA3-IOT Debugger Application
 */

// --- 全局状态 ---
const state = {
    token: localStorage.getItem('iot_v2_token') || '',
    uuid: localStorage.getItem('iot_v2_uuid') || '',   // 当前选中的设备instance_uuid
    baseUrl: ''
};

// --- 初始化 ---
document.addEventListener('DOMContentLoaded', () => {
    if (state.token) {
        document.getElementById('jwt_token').value = state.token;
        updateAuthUI(true);
    }
    document.getElementById('target_uuid').value = state.uuid;

    // 默认时间范围: 最近24小时
    const now = Math.floor(Date.now() / 1000);
    document.getElementById('hist_start').value = now - 86400;
    document.getElementById('hist_end').value = now;

    // 如果已有token，自动刷新设备列表
    if (state.token) refreshDeviceList();
});

// --- 核心API调用 (兼容两种响应格式) ---
async function apiCall(endpoint, method, data = null, isForm = false) {
    const baseUrl = document.getElementById('base_url').value.replace(/\/$/, '');
    const url = `${baseUrl}/api/v1${endpoint}`;

    const headers = {};
    if (state.token) headers['Authorization'] = `Bearer ${state.token}`;

    let body = null;
    if (data) {
        if (isForm) {
            headers['Content-Type'] = 'application/x-www-form-urlencoded';
            body = new URLSearchParams(data).toString();
        } else {
            headers['Content-Type'] = 'application/json';
            body = JSON.stringify(data);
        }
    }

    log(`→ ${method} ${endpoint}`, 'req', { data, isForm });

    try {
        const response = await fetch(url, { method, headers, body });
        const result = await response.json();

        // 处理成功响应 (HTTP 2xx)
        if (response.ok) {
            // 如果有标准包装 {code, message, data}，检查code是否为2xx
            if (result && typeof result === 'object' && 'code' in result) {
                if (result.code === 200 || result.code === 201) {
                    log(`← ${result.code} ${result.message || 'OK'}`, 'res', result.data);
                    return result;
                } else {
                    // 业务错误，但HTTP状态是2xx? 罕见，但按规范不应该发生
                    log(`✖ ${result.code} ${result.message}`, 'err', result);
                    alert(`业务错误: ${result.message}`);
                    return null;
                }
            } else {
                // 无标准包装，直接返回数据 (例如 /users/getUserAllDevices 返回直接数据)
                log(`← 200 (直接数据)`, 'res', result);
                // 构造一个类似标准响应的对象，方便统一处理
                return { code: 200, message: 'success', data: result };
            }
        } else {
            // HTTP错误 (4xx, 5xx)
            const msg = result?.message || `HTTP ${response.status}`;
            log(`✖ ${response.status} ${msg}`, 'err', result);
            alert(`请求失败: ${msg}`);
            return null;
        }
    } catch (err) {
        log(`✖ 网络错误`, 'err', err.message);
        alert(`网络请求失败: ${err.message}`);
        return null;
    }
}

// --- 用户认证 ---
async function doAuth(path) {
    const username = document.getElementById('user_name').value;
    const password = document.getElementById('user_pass').value;
    const res = await apiCall(path, 'POST', { username, password }, true);
    if (res && res.data && res.data.access_token) {
        state.token = res.data.access_token;
        localStorage.setItem('iot_v2_token', state.token);
        document.getElementById('jwt_token').value = state.token;
        updateAuthUI(true);
        refreshDeviceList(); // 登录后自动拉取设备
    }
}

// --- 匿名注册 (设备端) ---
async function registerAnon() {
    const typeId = document.getElementById('reg_type_id').value;
    const res = await apiCall('/device/deviceRegisterAnon', 'POST', { device_type_id: typeId }, true);
    if (res && res.data && res.data.device) {
        const d = res.data.device;
        const box = document.getElementById('reg_result');
        box.style.display = 'block';
        box.textContent = `reg_code: ${d.reg_code}\nuuid: ${d.uuid}\nverify_code: ${d.verify_code}\nexpires_at: ${d.expires_at}`;
        document.getElementById('bind_code').value = d.reg_code;
    }
}

// --- 用户绑定设备 (JSON) ---
async function bindDevice() {
    const data = {
        reg_code: document.getElementById('bind_code').value,
        device_nick: document.getElementById('bind_nick').value,
        device_remark: ""
    };
    const res = await apiCall('/users/bindDeviceByRegCode', 'POST', data, false);
    if (res) refreshDeviceList();
}

// --- 获取我的设备列表 (兼容实际返回 {instance_count, instances}) ---
async function refreshDeviceList() {
    const res = await apiCall('/users/getUserAllDevices', 'GET');
    const container = document.getElementById('device_container');
    if (res && res.data) {
        // 实际返回结构: { instance_count: 1, instances: [...] }
        let devices = [];
        if (Array.isArray(res.data)) {
            devices = res.data; // 万一直接返回数组
        } else if (res.data.instances && Array.isArray(res.data.instances)) {
            devices = res.data.instances;
        } else if (res.data.devices && Array.isArray(res.data.devices)) {
            devices = res.data.devices; // 兼容旧版
        }

        if (devices.length === 0) {
            container.innerHTML = '<div style="text-align:center; padding:20px; color:#94a3b8;">暂无设备</div>';
            return;
        }

        container.innerHTML = devices.map(d => {
            const deviceUuid = d.instance_uuid || d.uuid;
            const isSelected = (state.uuid === deviceUuid);
            return `
                <div class="device-item ${isSelected ? 'selected' : ''}" onclick="selectDevice('${deviceUuid}')">
                    <span class="dot ${d.online ? 'online' : 'offline'}"></span>
                    <strong>${d.name || '无名称'}</strong>
                    <span style="opacity:0.6; font-size:0.7rem">(${(deviceUuid || '').substring(0,8)}...)</span>
                </div>
            `;
        }).join('');
    }
}

// --- 选择设备 ---
function selectDevice(uuid) {
    state.uuid = uuid;
    localStorage.setItem('iot_v2_uuid', uuid);
    document.getElementById('target_uuid').value = uuid;
    refreshDeviceList();  // 刷新高亮
}

// --- 发送指令 (actions) ---
async function sendAction() {
    const uuid = document.getElementById('target_uuid').value;
    if (!uuid) return alert("请先选择或输入设备UUID");
    let params = {};
    try {
        params = JSON.parse(document.getElementById('mqtt_params').value);
    } catch (e) {
        alert("Params JSON 格式错误");
        return;
    }
    const data = {
        command: document.getElementById('mqtt_cmd').value,
        params: params
    };
    await apiCall(`/devices/${uuid}/actions`, 'POST', data, false);
}

// --- 历史数据查询 ---
async function queryHistory() {
    const uuid = document.getElementById('target_uuid').value;
    if (!uuid) return alert("请选择设备");
    let properties = [];
    try {
        properties = JSON.parse(document.getElementById('hist_props').value);
    } catch (e) {
        alert("properties JSON 格式错误");
        return;
    }
    const data = {
        start_timestamp: parseInt(document.getElementById('hist_start').value),
        end_timestamp: parseInt(document.getElementById('hist_end').value),
        properties: properties,
        limit: parseInt(document.getElementById('hist_limit').value),
        offset: 0
    };
    await apiCall(`/devices/${uuid}/getHistoryData`, 'POST', data, false);
}

// --- 创建组 (JSON) ---
async function createGroup() {
    const name = document.getElementById('group_name').value;
    if (!name) return alert("请输入组名称");
    await apiCall('/devices/groups/create_group', 'POST', { name, description: "" }, false);
}

// --- 加入/退出组 (使用 group_uuid) ---
async function groupOp(type) {
    const deviceUuid = document.getElementById('target_uuid').value;
    const groupUuid = document.getElementById('group_uuid_op').value;
    if (!deviceUuid || !groupUuid) return alert("需要设备UUID和组UUID");
    await apiCall(`/devices/${deviceUuid}/${type}_group`, 'POST', { group_uuid: groupUuid }, false);
}

// --- 查询系统日志 (GET) ---
async function querySystemLogs() {
    const type = document.getElementById('log_type').value;
    const targetUuid = document.getElementById('target_uuid').value;
    if (!targetUuid) return alert("请先选择设备（日志查询需要目标UUID）");

    const start = document.getElementById('hist_start').value;
    const end = document.getElementById('hist_end').value;

    const paramName = (type === 'device') ? 'device_uuid' : 'user_uuid';
    const url = `/logs/${type}?${paramName}=${targetUuid}&start_time=${start}&end_time=${end}&limit=50`;

    await apiCall(url, 'GET');
}

// --- 分享设备 ---
async function shareDevice() {
    const deviceUuid = document.getElementById('target_uuid').value;
    const sharedWithUuid = document.getElementById('share_user_uuid').value;
    if (!deviceUuid || !sharedWithUuid) return alert("需要设备UUID和目标用户UUID");

    const data = {
        shared_with_uuid: sharedWithUuid,
        permission: document.getElementById('share_perm').value,
        expires_at: null
    };
    await apiCall(`/devices/${deviceUuid}/share`, 'POST', data, false);
}

// --- 辅助函数 ---
function log(title, tag, detail) {
    const out = document.getElementById('log-output');
    const div = document.createElement('div');
    div.className = 'log-line';
    const time = new Date().toLocaleTimeString();
    let detailHtml = '';
    if (detail !== undefined && detail !== null) {
        let detailStr = (typeof detail === 'object') ? JSON.stringify(detail, null, 2) : String(detail);
        detailHtml = `<pre style="color:#9cdcfe; margin-top:4px; font-size:0.7rem;">${escapeHtml(detailStr)}</pre>`;
    }
    div.innerHTML = `<span style="color:#888">[${time}]</span> <span class="log-tag tag-${tag}">${tag.toUpperCase()}</span> <strong>${escapeHtml(title)}</strong>${detailHtml}`;
    out.prepend(div);
    document.getElementById('log-count').textContent = out.children.length + " entries";
}

function escapeHtml(unsafe) {
    if (!unsafe) return unsafe;
    return unsafe.replace(/[&<>"]/g, function(m) {
        if (m === '&') return '&amp;';
        if (m === '<') return '&lt;';
        if (m === '>') return '&gt;';
        if (m === '"') return '&quot;';
        return m;
    });
}

function switchRegTab(tab) {
    document.querySelectorAll('#reg_tabs .tab').forEach(t => t.classList.remove('active'));
    event.target.classList.add('active');
    document.getElementById('panel-anon').style.display = tab === 'anon' ? 'block' : 'none';
    document.getElementById('panel-bind').style.display = tab === 'bind' ? 'block' : 'none';
}

function updateAuthUI(ok) {
    const el = document.getElementById('auth_status');
    el.textContent = ok ? '● 已认证' : '● 未认证';
    el.className = 'status-pill ' + (ok ? 'active' : '');
}

function clearStorage() {
    if (confirm("确定清除Token和本地缓存？")) {
        localStorage.clear();
        location.reload();
    }
}

// 暴露全局函数
window.doAuth = doAuth;
window.registerAnon = registerAnon;
window.bindDevice = bindDevice;
window.refreshDeviceList = refreshDeviceList;
window.selectDevice = selectDevice;
window.sendAction = sendAction;
window.queryHistory = queryHistory;
window.createGroup = createGroup;
window.groupOp = groupOp;
window.querySystemLogs = querySystemLogs;
window.shareDevice = shareDevice;
window.switchRegTab = switchRegTab;
window.clearStorage = clearStorage;
