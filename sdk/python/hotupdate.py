"""
热更新模块
提供完整的热更新功能，包括检查更新、下载、安装、回滚等

安全特性：
- 支持 HTTPS 和证书固定（继承自 LicenseClient）
- 文件哈希校验
- 自动备份和回滚

使用示例：
    from license_client import LicenseClient
    from hotupdate import HotUpdateManager

    client = LicenseClient(
        server_url="https://192.168.1.100:8080",
        app_key="your_app_key",
        cert_fingerprint="SHA256:AB:CD:EF:..."  # 证书固定
    )

    # 创建热更新管理器
    updater = HotUpdateManager(client, current_version="1.0.0")

    # 检查更新
    update_info = updater.check_update()
    if update_info and update_info.get('has_update'):
        print(f"发现新版本: {update_info['to_version']}")

        # 下载更新
        update_file = updater.download_update(update_info)

        # 应用更新
        updater.apply_update(update_info, update_file, target_dir="./app")
"""

import os
import json
import hashlib
import shutil
import zipfile
import threading
import time
from pathlib import Path
from typing import Optional, Dict, Callable, List
from enum import Enum

try:
    import requests
except ImportError:
    raise ImportError("请安装 requests 库: pip install requests")


class HotUpdateStatus(Enum):
    """更新状态"""
    PENDING = "pending"
    DOWNLOADING = "downloading"
    INSTALLING = "installing"
    SUCCESS = "success"
    FAILED = "failed"
    ROLLBACK = "rollback"


class HotUpdateError(Exception):
    """热更新错误"""
    pass


class HotUpdateManager:
    """热更新管理器"""

    def __init__(
        self,
        client,  # LicenseClient 实例
        current_version: str,
        update_dir: Optional[str] = None,
        backup_dir: Optional[str] = None,
        auto_check: bool = False,
        check_interval: int = 3600,
        callback: Optional[Callable[[HotUpdateStatus, float, Optional[Exception]], None]] = None
    ):
        """
        初始化热更新管理器

        Args:
            client: LicenseClient 实例
            current_version: 当前版本号
            update_dir: 更新文件存放目录
            backup_dir: 备份目录
            auto_check: 是否自动检查更新
            check_interval: 自动检查间隔（秒）
            callback: 更新状态回调函数
        """
        self.client = client
        self.current_version = current_version
        self.update_dir = update_dir or os.path.join(Path.home(), '.app_updates')
        self.backup_dir = backup_dir or os.path.join(Path.home(), '.app_backups')
        self.auto_check = auto_check
        self.check_interval = check_interval
        self.callback = callback

        self._latest_update: Optional[Dict] = None
        self._is_updating = False
        self._stop_auto_check = False
        self._auto_check_thread: Optional[threading.Thread] = None
        self._lock = threading.Lock()

        # 确保目录存在
        os.makedirs(self.update_dir, exist_ok=True)
        os.makedirs(self.backup_dir, exist_ok=True)

    def check_update(self) -> Optional[Dict]:
        """
        检查更新

        Returns:
            更新信息字典，如果没有更新返回 None
        """
        try:
            url = f"{self.client.server_url}/api/client/hotupdate/check"
            params = {
                "app_key": self.client.app_key,
                "version": self.current_version,
                "machine_id": self.client.machine_id
            }

            # 使用 client 的 session（支持证书固定）
            session = getattr(self.client, '_session', None) or requests
            resp = session.get(url, params=params, timeout=30)
            result = resp.json()

            if result.get('code') != 0:
                raise HotUpdateError(result.get('message', '检查更新失败'))

            data = result.get('data', {})

            with self._lock:
                self._latest_update = data

            return data if data.get('has_update') else None

        except requests.exceptions.RequestException as e:
            raise HotUpdateError(f"网络请求失败: {e}")

    def get_latest_update(self) -> Optional[Dict]:
        """获取最新的更新信息（从缓存）"""
        with self._lock:
            return self._latest_update

    def download_update(
        self,
        update_info: Dict,
        progress_callback: Optional[Callable[[int, int], None]] = None
    ) -> str:
        """
        下载更新

        Args:
            update_info: 更新信息
            progress_callback: 下载进度回调 (downloaded_bytes, total_bytes)

        Returns:
            下载的文件路径
        """
        if not update_info or not update_info.get('has_update'):
            raise HotUpdateError("没有可用的更新")

        with self._lock:
            if self._is_updating:
                raise HotUpdateError("正在更新中")
            self._is_updating = True

        try:
            # 上报下载状态
            self._report_status(update_info.get('id'), HotUpdateStatus.DOWNLOADING)
            self._notify_callback(HotUpdateStatus.DOWNLOADING, 0)

            # 构建下载URL
            download_url = self.client.server_url + update_info['download_url']

            # 使用 client 的 session（支持证书固定）
            session = getattr(self.client, '_session', None) or requests

            # 下载文件
            resp = session.get(download_url, stream=True, timeout=300)
            resp.raise_for_status()

            total_size = int(resp.headers.get('content-length', 0))

            # 创建文件
            filename = f"update_{update_info.get('from_version', 'unknown')}_to_{update_info['to_version']}.zip"
            file_path = os.path.join(self.update_dir, filename)

            downloaded = 0
            hash_obj = hashlib.sha256()

            with open(file_path, 'wb') as f:
                for chunk in resp.iter_content(chunk_size=32 * 1024):
                    if chunk:
                        f.write(chunk)
                        hash_obj.update(chunk)
                        downloaded += len(chunk)

                        if progress_callback:
                            progress_callback(downloaded, total_size)

                        if total_size > 0:
                            progress = downloaded / total_size
                            self._notify_callback(HotUpdateStatus.DOWNLOADING, progress)

            # 验证哈希
            file_hash = hash_obj.hexdigest()
            expected_hash = update_info.get('file_hash', '')

            if expected_hash and file_hash != expected_hash:
                os.remove(file_path)
                error = HotUpdateError("文件校验失败")
                self._report_status(update_info.get('id'), HotUpdateStatus.FAILED, str(error))
                self._notify_callback(HotUpdateStatus.FAILED, 0, error)
                raise error

            self._notify_callback(HotUpdateStatus.DOWNLOADING, 1)
            return file_path

        except requests.exceptions.RequestException as e:
            error = HotUpdateError(f"下载失败: {e}")
            self._report_status(update_info.get('id'), HotUpdateStatus.FAILED, str(error))
            self._notify_callback(HotUpdateStatus.FAILED, 0, error)
            raise error

        finally:
            with self._lock:
                self._is_updating = False

    def apply_update(
        self,
        update_info: Dict,
        update_file: str,
        target_dir: str,
        pre_update_hook: Optional[Callable[[], bool]] = None,
        post_update_hook: Optional[Callable[[], bool]] = None
    ) -> bool:
        """
        应用更新

        Args:
            update_info: 更新信息
            update_file: 更新文件路径
            target_dir: 目标目录
            pre_update_hook: 更新前钩子，返回 False 取消更新
            post_update_hook: 更新后钩子，返回 False 触发回滚

        Returns:
            是否成功
        """
        if not update_info:
            raise HotUpdateError("更新信息为空")

        # 执行更新前钩子
        if pre_update_hook and not pre_update_hook():
            raise HotUpdateError("更新前检查失败")

        # 上报安装状态
        self._report_status(update_info.get('id'), HotUpdateStatus.INSTALLING)
        self._notify_callback(HotUpdateStatus.INSTALLING, 0)

        # 备份当前版本
        backup_path = os.path.join(
            self.backup_dir,
            f"backup_{self.current_version}_{int(time.time())}"
        )

        try:
            self._backup_current_version(target_dir, backup_path)
        except Exception as e:
            error = HotUpdateError(f"备份失败: {e}")
            self._report_status(update_info.get('id'), HotUpdateStatus.FAILED, str(error))
            self._notify_callback(HotUpdateStatus.FAILED, 0, error)
            raise error

        # 解压更新包
        try:
            self._extract_update(update_file, target_dir)
        except Exception as e:
            # 回滚
            self._rollback(backup_path, target_dir)
            error = HotUpdateError(f"解压失败: {e}")
            self._report_status(update_info.get('id'), HotUpdateStatus.FAILED, str(error))
            self._notify_callback(HotUpdateStatus.FAILED, 0, error)
            raise error

        # 执行更新后钩子
        if post_update_hook and not post_update_hook():
            # 回滚
            self._rollback(backup_path, target_dir)
            error = HotUpdateError("更新后检查失败，已回滚")
            self._report_status(update_info.get('id'), HotUpdateStatus.ROLLBACK, str(error))
            self._notify_callback(HotUpdateStatus.ROLLBACK, 0, error)
            raise error

        # 更新成功
        self.current_version = update_info['to_version']
        self._report_status(update_info.get('id'), HotUpdateStatus.SUCCESS)
        self._notify_callback(HotUpdateStatus.SUCCESS, 1)

        # 清理下载的更新包
        try:
            os.remove(update_file)
        except:
            pass

        # 清理旧备份
        self._clean_old_backups(keep=3)

        return True

    def rollback(self, target_dir: str) -> bool:
        """
        回滚到上一个版本

        Args:
            target_dir: 目标目录

        Returns:
            是否成功
        """
        # 查找最新的备份
        backups = []
        for entry in os.scandir(self.backup_dir):
            if entry.is_dir():
                backups.append((entry.path, entry.stat().st_mtime))

        if not backups:
            raise HotUpdateError("没有可用的备份")

        # 按时间排序，获取最新的
        backups.sort(key=lambda x: x[1], reverse=True)
        latest_backup = backups[0][0]

        return self._rollback(latest_backup, target_dir)

    def start_auto_check(self):
        """启动自动检查更新"""
        if not self.auto_check:
            return

        self._stop_auto_check = False

        def check_loop():
            # 立即检查一次
            try:
                self.check_update()
            except:
                pass

            while not self._stop_auto_check:
                time.sleep(self.check_interval)
                if not self._stop_auto_check:
                    try:
                        self.check_update()
                    except:
                        pass

        self._auto_check_thread = threading.Thread(target=check_loop, daemon=True)
        self._auto_check_thread.start()

    def stop_auto_check(self):
        """停止自动检查更新"""
        self._stop_auto_check = True

    def get_update_history(self) -> List[Dict]:
        """获取更新历史"""
        try:
            url = f"{self.client.server_url}/api/client/hotupdate/history"
            params = {
                "app_key": self.client.app_key,
                "machine_id": self.client.machine_id
            }

            # 使用 client 的 session（支持证书固定）
            session = getattr(self.client, '_session', None) or requests
            resp = session.get(url, params=params, timeout=30)
            result = resp.json()

            if result.get('code') != 0:
                return []

            return result.get('data', [])

        except:
            return []

    def is_updating(self) -> bool:
        """是否正在更新"""
        with self._lock:
            return self._is_updating

    def get_current_version(self) -> str:
        """获取当前版本"""
        return self.current_version

    def set_current_version(self, version: str):
        """设置当前版本"""
        self.current_version = version

    # 内部方法

    def _report_status(
        self,
        hot_update_id: Optional[str],
        status: HotUpdateStatus,
        error_msg: str = ""
    ):
        """上报更新状态"""
        if not hot_update_id:
            return

        try:
            url = f"{self.client.server_url}/api/client/hotupdate/report"
            data = {
                "app_key": self.client.app_key,
                "hot_update_id": hot_update_id,
                "machine_id": self.client.machine_id,
                "from_version": self.current_version,
                "status": status.value
            }
            if error_msg:
                data["error_message"] = error_msg

            # 使用 client 的 session（支持证书固定）
            session = getattr(self.client, '_session', None) or requests

            # 异步上报
            threading.Thread(
                target=lambda: session.post(url, json=data, timeout=10),
                daemon=True
            ).start()
        except:
            pass

    def _notify_callback(
        self,
        status: HotUpdateStatus,
        progress: float,
        error: Optional[Exception] = None
    ):
        """通知回调"""
        if self.callback:
            try:
                self.callback(status, progress, error)
            except:
                pass

    def _backup_current_version(self, source_dir: str, backup_path: str):
        """备份当前版本"""
        if os.path.exists(source_dir):
            shutil.copytree(source_dir, backup_path)

    def _extract_update(self, zip_file: str, target_dir: str):
        """解压更新包"""
        # 确保目标目录存在
        os.makedirs(target_dir, exist_ok=True)

        # 检查是否是 zip 文件
        if zipfile.is_zipfile(zip_file):
            with zipfile.ZipFile(zip_file, 'r') as zf:
                zf.extractall(target_dir)
        else:
            # 如果不是 zip，尝试直接复制
            if os.path.isdir(zip_file):
                shutil.copytree(zip_file, target_dir, dirs_exist_ok=True)
            else:
                shutil.copy2(zip_file, target_dir)

    def _rollback(self, backup_path: str, target_dir: str) -> bool:
        """回滚"""
        try:
            # 删除当前目录内容
            if os.path.exists(target_dir):
                shutil.rmtree(target_dir)

            # 从备份恢复
            shutil.copytree(backup_path, target_dir)
            return True
        except Exception as e:
            raise HotUpdateError(f"回滚失败: {e}")

    def _clean_old_backups(self, keep: int = 3):
        """清理旧备份"""
        try:
            backups = []
            for entry in os.scandir(self.backup_dir):
                if entry.is_dir():
                    backups.append((entry.path, entry.stat().st_mtime))

            if len(backups) <= keep:
                return

            # 按时间排序
            backups.sort(key=lambda x: x[1])

            # 删除旧的备份
            for backup_path, _ in backups[:-keep]:
                shutil.rmtree(backup_path)
        except:
            pass


# 便捷函数

def check_and_update(
    client,
    current_version: str,
    target_dir: str,
    auto_apply: bool = False,
    callback: Optional[Callable[[HotUpdateStatus, float, Optional[Exception]], None]] = None
) -> Optional[Dict]:
    """
    检查并更新（便捷函数）

    Args:
        client: LicenseClient 实例
        current_version: 当前版本
        target_dir: 目标目录
        auto_apply: 是否自动应用更新
        callback: 状态回调

    Returns:
        更新信息，如果没有更新返回 None
    """
    manager = HotUpdateManager(client, current_version, callback=callback)

    update_info = manager.check_update()

    if not update_info or not update_info.get('has_update'):
        return None

    if auto_apply or update_info.get('force_update'):
        update_file = manager.download_update(update_info)
        manager.apply_update(update_info, update_file, target_dir)

    return update_info
