"""
数据同步模块
支持将本地数据同步到云端服务器

功能特性：
- 增量同步支持
- 冲突检测和解决
- 批量操作
- 自动同步管理器

使用示例：
    from license_client import LicenseClient
    from data_sync import DataSyncClient

    # 初始化
    client = LicenseClient(server_url, app_key, skip_verify=True)
    sync_client = DataSyncClient(client)

    # 获取表列表
    tables = sync_client.get_table_list()

    # 拉取数据
    records, server_time = sync_client.pull_table("my_table")

    # 推送数据
    result = sync_client.push_record("my_table", "record_id", {"name": "test"})
"""

import json
import time
import threading
from typing import Optional, Dict, List, Any, Callable, Tuple
from dataclasses import dataclass, field
from enum import Enum


class ConflictResolution(Enum):
    """冲突解决策略"""
    USE_LOCAL = "use_local"
    USE_SERVER = "use_server"
    MERGE = "merge"


@dataclass
class SyncRecord:
    """同步记录"""
    id: str
    data: Dict[str, Any]
    version: int = 0
    is_deleted: bool = False
    updated_at: int = 0


@dataclass
class SyncResult:
    """同步结果"""
    record_id: str
    status: str  # success, conflict, error
    version: int = 0
    server_version: int = 0


@dataclass
class TableInfo:
    """表信息"""
    table_name: str
    record_count: int = 0
    last_updated: str = ""


@dataclass
class SyncChange:
    """同步变更记录"""
    id: str
    table: str
    record_id: str
    operation: str  # insert, update, delete
    data: Dict[str, Any]
    version: int = 0
    change_time: int = 0


@dataclass
class SyncStatus:
    """同步状态"""
    last_sync_time: int = 0
    pending_changes: int = 0
    table_status: Dict[str, int] = field(default_factory=dict)
    server_time: int = 0


@dataclass
class ConfigData:
    """配置数据"""
    key: str
    value: Any
    updated_at: int = 0


@dataclass
class WorkflowData:
    """工作流数据"""
    id: str
    name: str
    config: Dict[str, Any] = field(default_factory=dict)
    enabled: bool = True
    updated_at: int = 0


@dataclass
class MaterialData:
    """素材数据"""
    id: str
    type: str
    content: str
    tags: str = ""
    updated_at: int = 0


# 数据类型常量
DATA_TYPE_SCRIPTS = "scripts"  # 话术管理
DATA_TYPE_DANMAKU_GROUPS = "danmaku_groups"  # 互动规则
DATA_TYPE_AI_CONFIG = "ai_config"  # AI配置
DATA_TYPE_RANDOM_WORD_AI_CONFIG = "random_word_ai_config"  # 随机词AI配置


@dataclass
class BackupData:
    """备份数据"""
    id: str
    data_type: str
    data_json: str
    version: int = 0
    device_name: str = ""
    machine_id: str = ""
    is_current: bool = False
    data_size: int = 0
    item_count: int = 0
    checksum: str = ""
    created_at: str = ""
    updated_at: str = ""


@dataclass
class PostData:
    """帖子数据"""
    id: str
    content: str
    status: str = "draft"
    group_id: str = ""
    updated_at: int = 0


@dataclass
class CommentScriptData:
    """评论话术数据"""
    id: str
    content: str
    category: str = ""
    updated_at: int = 0


@dataclass
class PostGroup:
    """帖子分组"""
    id: str
    name: str
    count: int = 0


class DataSyncClient:
    """数据同步客户端"""

    def __init__(self, license_client):
        """
        初始化数据同步客户端

        Args:
            license_client: LicenseClient 实例
        """
        self.client = license_client
        self.last_sync_time: Dict[str, int] = {}

    def _request(self, method: str, endpoint: str, data: Optional[Dict] = None, params: Optional[Dict] = None) -> Dict:
        """发送 HTTP 请求"""
        url = f"{self.client.server_url}/api/client{endpoint}"

        # 添加基础参数
        base_params = {
            "app_key": self.client.app_key,
            "machine_id": self.client.machine_id
        }
        if params:
            base_params.update(params)

        try:
            if method == 'GET':
                resp = self.client._session.get(url, params=base_params, timeout=self.client.timeout)
            elif method == 'POST':
                if data:
                    data.update(base_params)
                else:
                    data = base_params
                resp = self.client._session.post(url, json=data, timeout=self.client.timeout)
            elif method == 'DELETE':
                resp = self.client._session.delete(url, params=base_params, json=data, timeout=self.client.timeout)
            elif method == 'PUT':
                if data:
                    data.update(base_params)
                else:
                    data = base_params
                resp = self.client._session.put(url, json=data, timeout=self.client.timeout)
            else:
                raise ValueError(f"不支持的请求方法: {method}")

            result = resp.json()
            if result.get('code') != 0:
                raise Exception(result.get('message', '请求失败'))
            return result.get('data', {})
        except Exception as e:
            raise Exception(f"请求失败: {e}")

    # ==================== 基础同步功能 ====================

    def get_table_list(self) -> List[TableInfo]:
        """获取服务器上的所有表名"""
        data = self._request('GET', '/sync/tables')
        return [TableInfo(
            table_name=t.get('table_name', ''),
            record_count=t.get('record_count', 0),
            last_updated=t.get('last_updated', '')
        ) for t in data] if isinstance(data, list) else []

    def pull_table(self, table_name: str, since: int = 0) -> Tuple[List[SyncRecord], int]:
        """
        从服务器拉取指定表的数据

        Args:
            table_name: 表名
            since: 增量同步时间戳（0表示全量）

        Returns:
            (记录列表, 服务器时间)
        """
        params = {"table": table_name}
        if since > 0:
            params["since"] = str(since)

        data = self._request('GET', '/sync/table', params=params)
        records = [SyncRecord(
            id=r.get('id', ''),
            data=r.get('data', {}),
            version=r.get('version', 0),
            is_deleted=r.get('is_deleted', False),
            updated_at=r.get('updated_at', 0)
        ) for r in data.get('records', [])]

        server_time = data.get('server_time', 0)
        self.last_sync_time[table_name] = server_time

        return records, server_time

    def pull_all_tables(self, since: int = 0) -> Tuple[Dict[str, List[SyncRecord]], int]:
        """
        从服务器拉取所有表的数据

        Args:
            since: 增量同步时间戳（0表示全量）

        Returns:
            (表名到记录列表的映射, 服务器时间)
        """
        params = {}
        if since > 0:
            params["since"] = str(since)

        data = self._request('GET', '/sync/tables/all', params=params)
        tables = {}
        for table_name, records in data.get('tables', {}).items():
            tables[table_name] = [SyncRecord(
                id=r.get('id', ''),
                data=r.get('data', {}),
                version=r.get('version', 0),
                is_deleted=r.get('is_deleted', False),
                updated_at=r.get('updated_at', 0)
            ) for r in records]

        return tables, data.get('server_time', 0)

    def push_record(self, table_name: str, record_id: str, data: Dict[str, Any], version: int = 0) -> SyncResult:
        """
        推送单条记录到服务器

        Args:
            table_name: 表名
            record_id: 记录ID
            data: 记录数据
            version: 版本号（用于冲突检测）

        Returns:
            同步结果
        """
        req_data = {
            "table": table_name,
            "record_id": record_id,
            "data": data,
            "version": version
        }
        result = self._request('POST', '/sync/table', req_data)
        return SyncResult(
            record_id=record_id,
            status=result.get('status', 'error'),
            version=result.get('version', 0),
            server_version=result.get('server_version', 0)
        )

    def push_record_batch(self, table_name: str, records: List[Dict[str, Any]]) -> List[SyncResult]:
        """
        批量推送记录到服务器

        Args:
            table_name: 表名
            records: 记录列表，每条记录包含 record_id, data, version, deleted

        Returns:
            同步结果列表
        """
        req_data = {
            "table": table_name,
            "records": records
        }
        result = self._request('POST', '/sync/table/batch', req_data)
        return [SyncResult(
            record_id=r.get('record_id', ''),
            status=r.get('status', 'error'),
            version=r.get('version', 0),
            server_version=r.get('server_version', 0)
        ) for r in result.get('results', [])]

    def delete_record(self, table_name: str, record_id: str) -> bool:
        """删除服务器上的记录"""
        req_data = {
            "table": table_name,
            "record_id": record_id
        }
        self._request('DELETE', '/sync/table', req_data)
        return True

    # ==================== 高级同步功能 ====================

    def push_changes(self, changes: List[SyncChange]) -> List[SyncResult]:
        """
        推送客户端变更到服务端（Push）

        Args:
            changes: 变更列表

        Returns:
            同步结果列表
        """
        req_data = {
            "changes": [{
                "id": c.id,
                "table": c.table,
                "record_id": c.record_id,
                "operation": c.operation,
                "data": c.data,
                "version": c.version,
                "change_time": c.change_time
            } for c in changes]
        }
        result = self._request('POST', '/sync/push', req_data)
        return [SyncResult(
            record_id=r.get('record_id', ''),
            status=r.get('status', 'error'),
            version=r.get('version', 0),
            server_version=r.get('server_version', 0)
        ) for r in result.get('results', [])]

    def get_changes(self, since: int = 0, tables: Optional[List[str]] = None) -> Tuple[List[SyncChange], int]:
        """
        获取服务端变更（Pull）

        Args:
            since: 从指定时间戳开始获取变更
            tables: 指定要获取的表（为空则获取所有表）

        Returns:
            (变更列表, 服务器时间)
        """
        params = {}
        if since > 0:
            params["since"] = str(since)
        if tables:
            params["tables"] = ",".join(tables)

        data = self._request('GET', '/sync/changes', params=params)
        changes = [SyncChange(
            id=c.get('id', ''),
            table=c.get('table', ''),
            record_id=c.get('record_id', ''),
            operation=c.get('operation', ''),
            data=c.get('data', {}),
            version=c.get('version', 0),
            change_time=c.get('change_time', 0)
        ) for c in data.get('changes', [])]

        return changes, data.get('server_time', 0)

    def get_sync_status(self) -> SyncStatus:
        """获取同步状态"""
        data = self._request('GET', '/sync/status')
        return SyncStatus(
            last_sync_time=data.get('last_sync_time', 0),
            pending_changes=data.get('pending_changes', 0),
            table_status=data.get('table_status', {}),
            server_time=data.get('server_time', 0)
        )

    def resolve_conflict(self, table_name: str, record_id: str, resolution: ConflictResolution,
                         merged_data: Optional[Dict[str, Any]] = None) -> SyncResult:
        """
        解决数据冲突

        Args:
            table_name: 表名
            record_id: 记录ID
            resolution: 解决策略
            merged_data: 当策略为 merge 时，提供合并后的数据

        Returns:
            同步结果
        """
        req_data = {
            "table": table_name,
            "record_id": record_id,
            "resolution": resolution.value
        }
        if merged_data:
            req_data["merged_data"] = merged_data

        result = self._request('POST', '/sync/conflict/resolve', req_data)
        return SyncResult(
            record_id=record_id,
            status=result.get('status', 'error'),
            version=result.get('version', 0)
        )

    # ==================== 分类数据同步功能 ====================

    def get_configs(self, since: int = 0) -> Tuple[List[ConfigData], int]:
        """获取配置数据"""
        params = {}
        if since > 0:
            params["since"] = str(since)

        data = self._request('GET', '/sync/configs', params=params)
        configs = [ConfigData(
            key=c.get('key', ''),
            value=c.get('value'),
            updated_at=c.get('updated_at', 0)
        ) for c in data.get('configs', [])]

        return configs, data.get('server_time', 0)

    def save_configs(self, configs: List[ConfigData]) -> bool:
        """保存配置数据"""
        req_data = {
            "configs": [{"key": c.key, "value": c.value, "updated_at": c.updated_at} for c in configs]
        }
        self._request('POST', '/sync/configs', req_data)
        return True

    def get_workflows(self, since: int = 0) -> Tuple[List[WorkflowData], int]:
        """获取工作流数据"""
        params = {}
        if since > 0:
            params["since"] = str(since)

        data = self._request('GET', '/sync/workflows', params=params)
        workflows = [WorkflowData(
            id=w.get('id', ''),
            name=w.get('name', ''),
            config=w.get('config', {}),
            enabled=w.get('enabled', True),
            updated_at=w.get('updated_at', 0)
        ) for w in data.get('workflows', [])]

        return workflows, data.get('server_time', 0)

    def save_workflows(self, workflows: List[WorkflowData]) -> bool:
        """保存工作流数据"""
        req_data = {
            "workflows": [{
                "id": w.id,
                "name": w.name,
                "config": w.config,
                "enabled": w.enabled,
                "updated_at": w.updated_at
            } for w in workflows]
        }
        self._request('POST', '/sync/workflows', req_data)
        return True

    def delete_workflow(self, workflow_id: str) -> bool:
        """删除工作流"""
        self._request('DELETE', f'/sync/workflows/{workflow_id}')
        return True

    def get_materials(self, since: int = 0) -> Tuple[List[MaterialData], int]:
        """获取素材数据"""
        params = {}
        if since > 0:
            params["since"] = str(since)

        data = self._request('GET', '/sync/materials', params=params)
        materials = [MaterialData(
            id=m.get('id', ''),
            type=m.get('type', ''),
            content=m.get('content', ''),
            tags=m.get('tags', ''),
            updated_at=m.get('updated_at', 0)
        ) for m in data.get('materials', [])]

        return materials, data.get('server_time', 0)

    def save_materials(self, materials: List[MaterialData]) -> bool:
        """保存素材数据"""
        req_data = {
            "materials": [{
                "id": m.id,
                "type": m.type,
                "content": m.content,
                "tags": m.tags,
                "updated_at": m.updated_at
            } for m in materials]
        }
        self._request('POST', '/sync/materials/batch', req_data)
        return True

    def get_posts(self, since: int = 0, group_id: str = "") -> Tuple[List[PostData], int]:
        """获取帖子数据"""
        params = {}
        if since > 0:
            params["since"] = str(since)
        if group_id:
            params["group_id"] = group_id

        data = self._request('GET', '/sync/posts', params=params)
        posts = [PostData(
            id=p.get('id', ''),
            content=p.get('content', ''),
            status=p.get('status', 'draft'),
            group_id=p.get('group_id', ''),
            updated_at=p.get('updated_at', 0)
        ) for p in data.get('posts', [])]

        return posts, data.get('server_time', 0)

    def save_posts(self, posts: List[PostData]) -> bool:
        """批量保存帖子数据"""
        req_data = {
            "posts": [{
                "id": p.id,
                "content": p.content,
                "status": p.status,
                "group_id": p.group_id,
                "updated_at": p.updated_at
            } for p in posts]
        }
        self._request('POST', '/sync/posts/batch', req_data)
        return True

    def update_post_status(self, post_id: str, status: str) -> bool:
        """更新帖子状态"""
        req_data = {"status": status}
        self._request('PUT', f'/sync/posts/{post_id}/status', req_data)
        return True

    def get_post_groups(self) -> List[PostGroup]:
        """获取帖子分组"""
        data = self._request('GET', '/sync/posts/groups')
        return [PostGroup(
            id=g.get('id', ''),
            name=g.get('name', ''),
            count=g.get('count', 0)
        ) for g in data.get('groups', [])]

    def get_comment_scripts(self, since: int = 0, category: str = "") -> Tuple[List[CommentScriptData], int]:
        """获取评论话术"""
        params = {}
        if since > 0:
            params["since"] = str(since)
        if category:
            params["category"] = category

        data = self._request('GET', '/sync/comment-scripts', params=params)
        scripts = [CommentScriptData(
            id=s.get('id', ''),
            content=s.get('content', ''),
            category=s.get('category', ''),
            updated_at=s.get('updated_at', 0)
        ) for s in data.get('scripts', [])]

        return scripts, data.get('server_time', 0)

    def save_comment_scripts(self, scripts: List[CommentScriptData]) -> bool:
        """批量保存评论话术"""
        req_data = {
            "scripts": [{
                "id": s.id,
                "content": s.content,
                "category": s.category,
                "updated_at": s.updated_at
            } for s in scripts]
        }
        self._request('POST', '/sync/comment-scripts/batch', req_data)
        return True

    # ==================== 便捷方法 ====================

    def sync_table_to_server(self, table_name: str, records: List[Dict[str, Any]], id_field: str = "id") -> List[SyncResult]:
        """
        将本地表数据同步到服务器

        Args:
            table_name: 表名
            records: 记录列表
            id_field: ID字段名

        Returns:
            同步结果列表
        """
        items = []
        for record in records:
            record_id = str(record.get(id_field, ''))
            if not record_id:
                continue
            items.append({
                "record_id": record_id,
                "data": record,
                "version": 0,
                "deleted": False
            })

        if not items:
            return []

        return self.push_record_batch(table_name, items)

    def sync_table_from_server(self, table_name: str, since: int = 0) -> Tuple[List[SyncRecord], List[str], int]:
        """
        从服务器同步表数据到本地

        Args:
            table_name: 表名
            since: 增量同步时间戳

        Returns:
            (需要更新的记录, 需要删除的记录ID, 服务器时间)
        """
        records, server_time = self.pull_table(table_name, since)

        updates = []
        deletes = []

        for r in records:
            if r.is_deleted:
                deletes.append(r.id)
            else:
                updates.append(r)

        return updates, deletes, server_time

    def get_last_sync_time(self, table_name: str) -> int:
        """获取指定表的最后同步时间"""
        return self.last_sync_time.get(table_name, 0)

    def set_last_sync_time(self, table_name: str, t: int):
        """设置指定表的最后同步时间"""
        self.last_sync_time[table_name] = t


class AutoSyncManager:
    """自动同步管理器"""

    def __init__(self, sync_client: DataSyncClient, tables: List[str], interval: float = 60.0):
        """
        初始化自动同步管理器

        Args:
            sync_client: DataSyncClient 实例
            tables: 要同步的表列表
            interval: 同步间隔（秒）
        """
        self.sync_client = sync_client
        self.tables = tables
        self.interval = interval
        self._stop_event = threading.Event()
        self._thread: Optional[threading.Thread] = None
        self.last_sync_time: Dict[str, int] = {}

        self.on_pull: Optional[Callable[[str, List[SyncRecord], List[str]], None]] = None
        self.on_conflict: Optional[Callable[[str, SyncResult], None]] = None
        self.on_error: Optional[Callable[[str, Exception], None]] = None

    def set_on_pull(self, callback: Callable[[str, List[SyncRecord], List[str]], None]):
        """设置拉取数据回调"""
        self.on_pull = callback

    def set_on_conflict(self, callback: Callable[[str, SyncResult], None]):
        """设置冲突处理回调"""
        self.on_conflict = callback

    def set_on_error(self, callback: Callable[[str, Exception], None]):
        """设置错误处理回调"""
        self.on_error = callback

    def start(self):
        """启动自动同步"""
        if self._thread and self._thread.is_alive():
            return

        self._stop_event.clear()
        self._thread = threading.Thread(target=self._sync_loop, daemon=True)
        self._thread.start()

    def stop(self):
        """停止自动同步"""
        self._stop_event.set()
        if self._thread:
            self._thread.join(timeout=5)

    def sync_now(self):
        """立即同步"""
        self._sync_all()

    def _sync_loop(self):
        """同步循环"""
        # 立即执行一次同步
        self._sync_all()

        while not self._stop_event.is_set():
            self._stop_event.wait(self.interval)
            if not self._stop_event.is_set():
                self._sync_all()

    def _sync_all(self):
        """同步所有表"""
        for table_name in self.tables:
            try:
                since = self.last_sync_time.get(table_name, 0)
                updates, deletes, server_time = self.sync_client.sync_table_from_server(table_name, since)

                if self.on_pull and (updates or deletes):
                    self.on_pull(table_name, updates, deletes)

                self.last_sync_time[table_name] = server_time
            except Exception as e:
                if self.on_error:
                    self.on_error(table_name, e)

    # ==================== 数据备份和同步功能 ====================

    def push_backup(self, data_type: str, data_json: str, device_name: str = "", item_count: int = 0) -> None:
        """
        推送备份数据到服务器

        Args:
            data_type: 数据类型（scripts/danmaku_groups/ai_config/random_word_ai_config）
            data_json: JSON格式的数据
            device_name: 设备名称（可选）
            item_count: 条目数量（可选）

        Raises:
            Exception: 推送失败时抛出异常
        """
        req_body = {
            "app_key": self.client.app_key,
            "machine_id": self.client.machine_id,
            "data_type": data_type,
            "data_json": data_json,
            "device_name": device_name,
            "item_count": item_count
        }

        resp = self.client.session.post(
            f"{self.client.server_url}/api/client/backup/push",
            json=req_body
        )
        result = resp.json()

        if result.get("code") != 0:
            raise Exception(f"API error: {result.get('message')}")

    def pull_backup(self, data_type: str) -> List[BackupData]:
        """
        从服务器拉取指定类型的备份数据

        Args:
            data_type: 数据类型（scripts/danmaku_groups/ai_config/random_word_ai_config）

        Returns:
            备份数据列表（按版本降序排列，第一个为当前版本）

        Raises:
            Exception: 拉取失败时抛出异常
        """
        params = {
            "app_key": self.client.app_key,
            "machine_id": self.client.machine_id,
            "data_type": data_type
        }

        resp = self.client.session.get(
            f"{self.client.server_url}/api/client/backup/pull",
            params=params
        )
        result = resp.json()

        if result.get("code") != 0:
            raise Exception(f"API error: {result.get('message')}")

        data_list = result.get("data", [])
        return [BackupData(**item) for item in data_list]

    def pull_all_backups(self) -> Dict[str, List[BackupData]]:
        """
        从服务器拉取所有类型的备份数据

        Returns:
            按数据类型分组的备份数据映射

        Raises:
            Exception: 拉取失败时抛出异常
        """
        params = {
            "app_key": self.client.app_key,
            "machine_id": self.client.machine_id
        }

        resp = self.client.session.get(
            f"{self.client.server_url}/api/client/backup/pull",
            params=params
        )
        result = resp.json()

        if result.get("code") != 0:
            raise Exception(f"API error: {result.get('message')}")

        data_list = result.get("data", [])

        # 按数据类型分组
        backup_map: Dict[str, List[BackupData]] = {}
        for item in data_list:
            backup = BackupData(**item)
            if backup.data_type not in backup_map:
                backup_map[backup.data_type] = []
            backup_map[backup.data_type].append(backup)

        return backup_map
