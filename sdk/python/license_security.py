"""
License Security Module
授权安全模块 - 提供反破解防护

功能：
1. 代码完整性校验
2. 反调试检测
3. 时间回拨检测
4. 多点分散验证
5. 环境检测
"""

import os
import sys
import time
import hashlib
import ctypes
import threading
import random
import string
from typing import Optional, Callable, List
from functools import wraps

# ==================== 代码完整性校验 ====================

class IntegrityChecker:
    """代码完整性校验器"""

    def __init__(self):
        self._file_hashes = {}
        self._init_hashes()

    def _init_hashes(self):
        """初始化文件哈希（首次运行时记录）"""
        try:
            # 获取当前模块文件
            current_file = os.path.abspath(__file__)
            client_file = os.path.join(os.path.dirname(current_file), 'license_client.py')

            for filepath in [current_file, client_file]:
                if os.path.exists(filepath):
                    self._file_hashes[filepath] = self._calculate_hash(filepath)
        except:
            pass

    def _calculate_hash(self, filepath: str) -> str:
        """计算文件哈希"""
        try:
            with open(filepath, 'rb') as f:
                return hashlib.sha256(f.read()).hexdigest()
        except:
            return ""

    def verify(self) -> bool:
        """验证文件完整性"""
        for filepath, expected_hash in self._file_hashes.items():
            if os.path.exists(filepath):
                current_hash = self._calculate_hash(filepath)
                if current_hash != expected_hash:
                    return False
        return True

    def get_checksum(self) -> str:
        """获取校验和（用于服务器验证）"""
        combined = ''.join(sorted(self._file_hashes.values()))
        return hashlib.md5(combined.encode()).hexdigest()[:16]


# ==================== 反调试检测 ====================

class AntiDebug:
    """反调试检测"""

    @staticmethod
    def is_debugger_present() -> bool:
        """检测是否有调试器"""
        checks = [
            AntiDebug._check_trace,
            AntiDebug._check_env,
            AntiDebug._check_parent_process,
            AntiDebug._check_timing,
        ]

        for check in checks:
            try:
                if check():
                    return True
            except:
                pass
        return False

    @staticmethod
    def _check_trace() -> bool:
        """检查 sys.gettrace"""
        return sys.gettrace() is not None

    @staticmethod
    def _check_env() -> bool:
        """检查调试相关环境变量"""
        debug_vars = ['PYTHONDEBUG', 'PYTHONINSPECT', 'PYCHARM_DEBUG', 'VSCODE_DEBUG']
        for var in debug_vars:
            if os.environ.get(var):
                return True
        return False

    @staticmethod
    def _check_parent_process() -> bool:
        """检查父进程是否为调试器"""
        try:
            import psutil
            parent = psutil.Process(os.getpid()).parent()
            if parent:
                parent_name = parent.name().lower()
                debuggers = ['pycharm', 'vscode', 'code', 'idea', 'debug', 'ida', 'olly', 'x64dbg']
                for debugger in debuggers:
                    if debugger in parent_name:
                        return True
        except:
            pass
        return False

    @staticmethod
    def _check_timing() -> bool:
        """时间检测（调试会导致执行变慢）"""
        start = time.perf_counter()
        # 执行一些简单操作
        _ = sum(range(10000))
        elapsed = time.perf_counter() - start
        # 正常情况下应该很快，调试时会变慢
        return elapsed > 0.1  # 100ms 阈值


# ==================== 时间回拨检测 ====================

class TimeChecker:
    """时间回拨检测"""

    def __init__(self, cache_file: str = None):
        self._last_check_time = time.time()
        self._cache_file = cache_file or os.path.join(
            os.path.expanduser('~'), '.license_cache', '.time_check'
        )
        self._load_last_time()

    def _load_last_time(self):
        """加载上次检查时间"""
        try:
            if os.path.exists(self._cache_file):
                with open(self._cache_file, 'r') as f:
                    saved_time = float(f.read().strip())
                    if saved_time > self._last_check_time:
                        self._last_check_time = saved_time
        except:
            pass

    def _save_current_time(self):
        """保存当前时间"""
        try:
            os.makedirs(os.path.dirname(self._cache_file), exist_ok=True)
            with open(self._cache_file, 'w') as f:
                f.write(str(time.time()))
        except:
            pass

    def check(self) -> bool:
        """
        检查时间是否被回拨

        Returns:
            True: 时间正常
            False: 检测到时间回拨
        """
        current_time = time.time()

        # 允许 5 分钟的误差（系统时间同步可能有小幅调整）
        if current_time < self._last_check_time - 300:
            return False

        self._last_check_time = current_time
        self._save_current_time()
        return True

    def get_server_time_diff(self, server_time: float) -> float:
        """获取与服务器时间的差异"""
        return abs(time.time() - server_time)


# ==================== 多点分散验证 ====================

class DistributedValidator:
    """分散验证器 - 将验证逻辑分散到多个点"""

    def __init__(self):
        self._validators: List[Callable[[], bool]] = []
        self._check_results = {}
        self._lock = threading.Lock()

    def register(self, name: str, validator: Callable[[], bool]):
        """注册验证器"""
        self._validators.append((name, validator))

    def validate_all(self) -> bool:
        """执行所有验证"""
        with self._lock:
            self._check_results.clear()
            for name, validator in self._validators:
                try:
                    result = validator()
                    self._check_results[name] = result
                    if not result:
                        return False
                except:
                    self._check_results[name] = False
                    return False
            return True

    def get_validation_token(self) -> str:
        """生成验证令牌（用于服务器二次验证）"""
        # 将验证结果编码为令牌
        results = ''.join(['1' if v else '0' for v in self._check_results.values()])
        timestamp = str(int(time.time()))
        combined = f"{results}:{timestamp}"
        return hashlib.md5(combined.encode()).hexdigest()


# ==================== 环境检测 ====================

class EnvironmentChecker:
    """环境检测 - 检测虚拟机、沙箱等"""

    @staticmethod
    def is_virtual_machine() -> bool:
        """检测是否在虚拟机中运行"""
        vm_indicators = []

        # 检查 MAC 地址前缀（常见虚拟机）
        try:
            import uuid
            mac = ':'.join(['{:02x}'.format((uuid.getnode() >> i) & 0xff) for i in range(0, 48, 8)][::-1])
            vm_mac_prefixes = ['00:0c:29', '00:50:56', '08:00:27', '00:1c:42', '00:16:3e']
            for prefix in vm_mac_prefixes:
                if mac.lower().startswith(prefix):
                    vm_indicators.append('mac')
                    break
        except:
            pass

        # 检查系统信息
        try:
            import platform
            system_info = platform.platform().lower()
            vm_keywords = ['vmware', 'virtualbox', 'virtual', 'qemu', 'xen', 'hyperv']
            for keyword in vm_keywords:
                if keyword in system_info:
                    vm_indicators.append('platform')
                    break
        except:
            pass

        # 检查特定文件/目录
        vm_paths = [
            '/sys/class/dmi/id/product_name',  # Linux
            'C:\\Windows\\System32\\drivers\\vmmouse.sys',  # Windows VMware
            'C:\\Windows\\System32\\drivers\\VBoxMouse.sys',  # Windows VirtualBox
        ]
        for path in vm_paths:
            if os.path.exists(path):
                try:
                    if os.path.isfile(path):
                        with open(path, 'r') as f:
                            content = f.read().lower()
                            if any(kw in content for kw in ['vmware', 'virtualbox', 'qemu']):
                                vm_indicators.append('file')
                                break
                except:
                    pass

        return len(vm_indicators) >= 2  # 至少两个指标才判定为虚拟机

    @staticmethod
    def is_sandbox() -> bool:
        """检测是否在沙箱中运行"""
        sandbox_indicators = []

        # 检查用户名
        try:
            import getpass
            username = getpass.getuser().lower()
            sandbox_users = ['sandbox', 'virus', 'malware', 'test', 'sample', 'analysis']
            if any(su in username for su in sandbox_users):
                sandbox_indicators.append('username')
        except:
            pass

        # 检查进程数量（沙箱通常进程很少）
        try:
            import psutil
            if len(psutil.pids()) < 50:
                sandbox_indicators.append('process_count')
        except:
            pass

        # 检查磁盘大小（沙箱通常磁盘很小）
        try:
            import shutil
            total, used, free = shutil.disk_usage('/')
            if total < 60 * 1024 * 1024 * 1024:  # 小于 60GB
                sandbox_indicators.append('disk_size')
        except:
            pass

        return len(sandbox_indicators) >= 2


# ==================== 函数混淆装饰器 ====================

def _obfuscate_string(s: str) -> str:
    """字符串混淆"""
    return ''.join([chr(ord(c) ^ 0x5A) for c in s])

def _deobfuscate_string(s: str) -> str:
    """字符串反混淆"""
    return ''.join([chr(ord(c) ^ 0x5A) for c in s])

def secure_check(func):
    """安全检查装饰器"""
    @wraps(func)
    def wrapper(*args, **kwargs):
        # 执行前检查
        if AntiDebug.is_debugger_present():
            # 不直接报错，而是返回错误结果
            return False if func.__name__.startswith('is_') else None

        result = func(*args, **kwargs)

        # 随机延迟，增加时间分析难度
        time.sleep(random.uniform(0.001, 0.01))

        return result
    return wrapper


# ==================== 安全授权客户端包装器 ====================

class SecureLicenseClient:
    """安全授权客户端包装器"""

    def __init__(self, client, enable_integrity_check: bool = True):
        """
        包装原始客户端，添加安全防护

        Args:
            client: LicenseClient 实例
            enable_integrity_check: 是否启用完整性检查（生产环境建议启用，开发时可禁用）
        """
        self._client = client
        self._integrity_checker = IntegrityChecker()
        self._time_checker = TimeChecker()
        self._validator = DistributedValidator()
        self._env_checker = EnvironmentChecker()
        self._check_count = 0
        self._last_full_check = 0
        self._enable_integrity_check = enable_integrity_check

        # 注册验证器
        self._setup_validators()

    def _setup_validators(self):
        """设置分散验证器"""
        self._validator.register('integrity', self._integrity_checker.verify)
        self._validator.register('time', self._time_checker.check)
        self._validator.register('license', lambda: self._client._license_info is not None)
        self._validator.register('valid_flag', lambda: self._client._license_info.get('valid', False) if self._client._license_info else False)

    @secure_check
    def is_valid(self) -> bool:
        """安全的授权验证"""
        self._check_count += 1

        # 每 10 次检查执行一次完整验证
        if self._check_count % 10 == 0 or time.time() - self._last_full_check > 300:
            if not self._full_security_check():
                return False
            self._last_full_check = time.time()

        # 基本验证
        return self._client.is_valid()

    def _full_security_check(self) -> bool:
        """完整安全检查"""
        # 1. 反调试检测
        if AntiDebug.is_debugger_present():
            self._on_security_violation('debugger_detected')
            return False

        # 2. 时间检查
        if not self._time_checker.check():
            self._on_security_violation('time_rollback')
            return False

        # 3. 完整性检查（可通过构造函数参数控制）
        if self._enable_integrity_check:
            if not self._integrity_checker.verify():
                self._on_security_violation('integrity_failed')
                return False

        # 4. 分散验证
        if not self._validator.validate_all():
            self._on_security_violation('validation_failed')
            return False

        return True

    def _on_security_violation(self, reason: str):
        """安全违规处理"""
        # 可以选择：
        # 1. 静默失败（推荐，不给破解者提示）
        # 2. 记录日志
        # 3. 上报服务器
        # 4. 清除授权缓存

        # 静默清除缓存
        try:
            self._client._clear_cache()
        except:
            pass

    @secure_check
    def has_feature(self, feature: str) -> bool:
        """安全的功能检查"""
        if not self.is_valid():
            return False
        return self._client.has_feature(feature)

    def get_remaining_days(self) -> int:
        """获取剩余天数"""
        if not self.is_valid():
            return 0
        return self._client.get_remaining_days()

    def activate(self, license_key: str):
        """激活"""
        return self._client.activate(license_key)

    def login(self, email: str, password: str):
        """登录"""
        return self._client.login(email, password)

    def deactivate(self):
        """解绑"""
        return self._client.deactivate()

    def get_security_token(self) -> str:
        """获取安全令牌（用于服务器验证客户端完整性）"""
        parts = [
            self._integrity_checker.get_checksum(),
            self._validator.get_validation_token(),
            str(int(time.time())),
        ]
        combined = ':'.join(parts)
        return hashlib.sha256(combined.encode()).hexdigest()[:32]


# ==================== 便捷函数 ====================

def wrap_client(client, enable_integrity_check: bool = True) -> SecureLicenseClient:
    """
    包装客户端，添加安全防护

    Args:
        client: LicenseClient 实例
        enable_integrity_check: 是否启用完整性检查（生产环境建议启用，开发时可禁用）
    """
    return SecureLicenseClient(client, enable_integrity_check=enable_integrity_check)


def check_environment() -> dict:
    """检查运行环境"""
    return {
        'debugger': AntiDebug.is_debugger_present(),
        'virtual_machine': EnvironmentChecker.is_virtual_machine(),
        'sandbox': EnvironmentChecker.is_sandbox(),
    }
