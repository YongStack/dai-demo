// 全局状态
let currentBucket = null;

// ==================== 工具函数 ====================

function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function formatRate(bytesPerSec) {
    if (bytesPerSec === 0) return '0 B/s';
    return formatBytes(bytesPerSec) + '/s';
}

function formatIOPS(iops) {
    if (iops === 0) return '0 次/s';
    return iops.toLocaleString() + ' 次/s';
}

function formatQuotaDisplay(quota, used) {
    if (quota === 0) return '不限';
    return formatBytes(quota);
}

function showToast(msg, type = 'success') {
    const toast = document.getElementById('toast');
    toast.textContent = msg;
    toast.className = 'toast ' + type + ' show';
    setTimeout(() => toast.classList.remove('show'), 3000);
}

async function apiCall(url, options = {}) {
    const resp = await fetch(url, options);
    const data = resp.status !== 204 ? await resp.json().catch(() => null) : null;
    if (!resp.ok) {
        throw new Error(data?.error || `HTTP ${resp.status}`);
    }
    return data;
}

// ==================== 桶列表管理 ====================

async function loadBuckets() {
    try {
        const buckets = await apiCall('/api/buckets');
        renderBucketList(buckets);
    } catch (e) {
        showToast('加载桶列表失败: ' + e.message, 'error');
    }
}

function renderBucketList(buckets) {
    const list = document.getElementById('bucket-list');
    if (!buckets || buckets.length === 0) {
        list.innerHTML = '<p class="empty-hint">暂无存储桶，点击「创建桶」开始</p>';
        return;
    }

    list.innerHTML = buckets.map(b => `
        <div class="bucket-item ${currentBucket === b.bucket ? 'active' : ''}" onclick="selectBucket('${b.bucket}')">
            <div class="bucket-item-name">${b.bucket}</div>
            <div class="bucket-item-stats">
                <span>容量: ${formatBytes(b.used_capacity)} / ${formatBytes(b.quota_capacity)}</span>
                <span>对象: ${b.object_count}</span>
            </div>
        </div>
    `).join('');
}

// ==================== 创建桶 ====================

document.getElementById('btn-create-toggle').addEventListener('click', () => {
    const form = document.getElementById('create-form');
    form.style.display = form.style.display === 'none' ? 'block' : 'none';
});

document.getElementById('btn-cancel-create').addEventListener('click', () => {
    document.getElementById('create-form').style.display = 'none';
});

document.getElementById('btn-create-bucket').addEventListener('click', async () => {
    const name = document.getElementById('new-bucket-name').value.trim();
    const capacityGB = parseFloat(document.getElementById('new-quota-capacity').value);
    const readBPS = parseInt(document.getElementById('new-quota-readbps').value) || 0;
    const writeBPS = parseInt(document.getElementById('new-quota-writebps').value) || 0;
    const readIOPS = parseInt(document.getElementById('new-quota-readiops').value) || 0;
    const writeIOPS = parseInt(document.getElementById('new-quota-writeiops').value) || 0;

    if (!name) {
        showToast('请输入桶名', 'error');
        return;
    }
    if (!capacityGB || capacityGB <= 0) {
        showToast('请输入有效的容量配额', 'error');
        return;
    }

    const req = {
        bucket: name,
        quota_capacity: Math.round(capacityGB * 1024 * 1024 * 1024),
        quota_read_bps: readBPS * 1024 * 1024,
        quota_write_bps: writeBPS * 1024 * 1024,
        quota_read_iops: readIOPS,
        quota_write_iops: writeIOPS,
    };

    try {
        await apiCall('/api/buckets', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(req),
        });
        showToast('桶创建成功', 'success');
        document.getElementById('create-form').style.display = 'none';
        document.getElementById('new-bucket-name').value = '';
        document.getElementById('new-quota-capacity').value = '';
        document.getElementById('new-quota-readbps').value = '0';
        document.getElementById('new-quota-writebps').value = '0';
        document.getElementById('new-quota-readiops').value = '0';
        document.getElementById('new-quota-writeiops').value = '0';
        await loadBuckets();
    } catch (e) {
        showToast('创建失败: ' + e.message, 'error');
    }
});

// ==================== 桶详情 ====================

async function selectBucket(bucketName) {
    currentBucket = bucketName;
    await loadBuckets();
    await loadQuotaDetail(bucketName);
    await loadObjects(bucketName);
}

async function loadQuotaDetail(bucketName) {
    try {
        const q = await apiCall(`/api/buckets/${bucketName}/quota`);
        renderQuotaDetail(q);
    } catch (e) {
        showToast('加载配额详情失败: ' + e.message, 'error');
    }
}

function renderQuotaDetail(q) {
    document.getElementById('detail-empty').style.display = 'none';
    document.getElementById('detail-content').style.display = 'block';
    document.getElementById('detail-actions').style.display = 'flex';
    document.getElementById('detail-title').textContent = `配额详情 — ${q.bucket}`;

    // 容量配额
    document.getElementById('cap-used-total').textContent =
        `${formatBytes(q.used_capacity)} / ${formatBytes(q.quota_capacity)}`;
    document.getElementById('cap-remaining').textContent = formatBytes(q.remaining_capacity);
    document.getElementById('cap-percent').textContent = q.usage_percent.toFixed(2) + '%';
    document.getElementById('cap-object-count').textContent = q.object_count;

    const progress = document.getElementById('cap-progress');
    progress.style.width = Math.min(q.usage_percent, 100) + '%';
    progress.className = 'progress-fill';
    if (q.usage_percent >= 90) progress.classList.add('high');
    else if (q.usage_percent >= 70) progress.classList.add('mid');
    else progress.classList.add('low');

    // 读写带宽配额
    document.getElementById('bps-read-quota').textContent = formatQuotaDisplay(q.quota_read_bps, q.used_read_bps);
    document.getElementById('bps-read-used').textContent = formatRate(q.used_read_bps);
    document.getElementById('bps-write-quota').textContent = formatQuotaDisplay(q.quota_write_bps, q.used_write_bps);
    document.getElementById('bps-write-used').textContent = formatRate(q.used_write_bps);

    // IOPS 配额
    document.getElementById('iops-read-quota').textContent = q.quota_read_iops === 0 ? '不限' : formatIOPS(q.quota_read_iops);
    document.getElementById('iops-read-used').textContent = formatIOPS(q.used_read_iops);
    document.getElementById('iops-write-quota').textContent = q.quota_write_iops === 0 ? '不限' : formatIOPS(q.quota_write_iops);
    document.getElementById('iops-write-used').textContent = formatIOPS(q.used_write_iops);
}

// ==================== 对象管理 ====================

async function loadObjects(bucketName) {
    try {
        const resp = await apiCall(`/api/buckets/${bucketName}/objects`);
        renderObjects(resp.objects || []);
    } catch (e) {
        console.error('load objects:', e);
    }
}

function renderObjects(objects) {
    const list = document.getElementById('object-list');
    if (!objects || objects.length === 0) {
        list.innerHTML = '<p class="empty-hint">暂无对象</p>';
        return;
    }
    list.innerHTML = objects.map(o => `
        <div class="object-item">
            <div class="object-item-info">
                <div class="object-item-name">${o.key}</div>
                <div class="object-item-meta">${formatBytes(o.size)} · ${new Date(o.created_at).toLocaleString('zh-CN')}</div>
            </div>
            <div class="object-item-actions">
                <button class="btn btn-danger btn-icon" onclick="deleteObject('${o.bucket}', '${o.key}')">删除</button>
            </div>
        </div>
    `).join('');
}

// 上传文件
document.getElementById('btn-upload').addEventListener('click', async () => {
    const fileInput = document.getElementById('upload-file-input');
    const file = fileInput.files[0];
    if (!file) {
        showToast('请选择文件', 'error');
        return;
    }
    if (!currentBucket) {
        showToast('请先选择桶', 'error');
        return;
    }

    const formData = new FormData();
    formData.append('file', file);

    const hint = document.getElementById('upload-hint');
    hint.textContent = '上传中...';

    try {
        const resp = await fetch(`/api/buckets/${currentBucket}/objects`, {
            method: 'POST',
            body: formData,
        });
        const data = await resp.json();
        if (!resp.ok) {
            throw new Error(data.error || `HTTP ${resp.status}`);
        }
        showToast(`上传成功: ${data.file} (${formatBytes(data.size)})`, 'success');
        fileInput.value = '';
        hint.textContent = '';
        await loadQuotaDetail(currentBucket);
        await loadObjects(currentBucket);
        await loadBuckets();
    } catch (e) {
        hint.textContent = '';
        showToast('上传失败: ' + e.message, 'error');
    }
});

// 删除对象
async function deleteObject(bucket, key) {
    try {
        await apiCall(`/api/buckets/${bucket}/objects/${key}`, { method: 'DELETE' });
        showToast('对象已删除', 'success');
        await loadQuotaDetail(bucket);
        await loadObjects(bucket);
        await loadBuckets();
    } catch (e) {
        showToast('删除失败: ' + e.message, 'error');
    }
}

// 删除桶
document.getElementById('btn-delete-bucket').addEventListener('click', async () => {
    if (!currentBucket) return;
    if (!confirm(`确认删除桶 "${currentBucket}"？此操作不可恢复，桶内所有对象将被删除。`)) return;
    try {
        await apiCall(`/api/buckets/${currentBucket}`, { method: 'DELETE' });
        showToast('桶已删除', 'success');
        currentBucket = null;
        document.getElementById('detail-empty').style.display = 'block';
        document.getElementById('detail-content').style.display = 'none';
        document.getElementById('detail-actions').style.display = 'none';
        document.getElementById('detail-title').textContent = '配额详情';
        await loadBuckets();
    } catch (e) {
        showToast('删除失败: ' + e.message, 'error');
    }
});

// 初始化
loadBuckets();
