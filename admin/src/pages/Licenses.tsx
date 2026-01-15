import React, { useEffect, useState } from 'react';
import { Table, Button, Space, Modal, Form, Input, Select, message, Tag, InputNumber, Descriptions, Checkbox, App } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, StopOutlined, ReloadOutlined, CopyOutlined, DownloadOutlined } from '@ant-design/icons';
import { licenseApi, appApi, teamApi, exportApi } from '../api';
import dayjs from 'dayjs';

const { Option } = Select;

const Licenses: React.FC = () => {
  const { modal } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<any[]>([]);
  const [apps, setApps] = useState<any[]>([]);
  const [members, setMembers] = useState<any[]>([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [currentLicense, setCurrentLicense] = useState<any>(null);
  const [form] = Form.useForm();
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 });
  const [filters, setFilters] = useState<any>({});
  const [selectedAppFeatures, setSelectedAppFeatures] = useState<string[]>([]);

  useEffect(() => {
    fetchData();
    fetchApps();
    fetchMembers();
  }, []);

  const fetchData = async (page = 1, pageSize = 10, filterParams = filters) => {
    setLoading(true);
    try {
      const result: any = await licenseApi.list({ page, page_size: pageSize, ...filterParams });
      setData(result.list || []);
      setPagination({ current: page, pageSize, total: result.total || 0 });
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  const fetchApps = async () => {
    try {
      const result: any = await appApi.list();
      setApps(result || []);
    } catch (error) {
      console.error(error);
    }
  };

  const fetchMembers = async () => {
    try {
      const result: any = await teamApi.list({ page_size: 100 });
      setMembers(result.list || []);
    } catch (error) {
      console.error(error);
    }
  };

  const handleCreate = () => {
    setCurrentLicense(null);
    form.resetFields();
    setSelectedAppFeatures([]);
    setModalVisible(true);
  };

  const handleEdit = (record: any) => {
    setCurrentLicense(record);
    // 获取应用的功能列表
    const app = apps.find(a => a.id === record.app_id);
    setSelectedAppFeatures(app?.features || []);
    // 解析已选功能
    let selectedFeatures: string[] = [];
    if (record.features) {
      try {
        const featuresObj = typeof record.features === 'string' ? JSON.parse(record.features) : record.features;
        selectedFeatures = Object.keys(featuresObj).filter(k => featuresObj[k]);
      } catch (e) {
        console.error(e);
      }
    }
    form.setFieldsValue({
      ...record,
      expires_at: record.expires_at ? dayjs(record.expires_at) : null,
      features: selectedFeatures,
    });
    setModalVisible(true);
  };

  const handleAppChange = (appId: string) => {
    const app = apps.find(a => a.id === appId);
    setSelectedAppFeatures(app?.features || []);
    // 默认全选所有功能
    form.setFieldsValue({ features: app?.features || [] });
    // 设置默认设备数
    if (app?.max_devices_default) {
      form.setFieldsValue({ max_devices: app.max_devices_default });
    }
  };

  const handleView = async (record: any) => {
    try {
      const detail = await licenseApi.get(record.id);
      setCurrentLicense(detail);
      setDetailVisible(true);
    } catch (error) {
      console.error(error);
    }
  };

  const handleDelete = (record: any) => {
    modal.confirm({
      title: '确认删除',
      content: `确定要删除此授权吗？`,
      onOk: async () => {
        try {
          await licenseApi.delete(record.id);
          message.success('删除成功');
          fetchData(pagination.current, pagination.pageSize);
        } catch (error) {
          console.error(error);
        }
      },
    });
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      // features 直接传数组，后端会处理
      const submitData = {
        app_id: values.app_id,
        customer_id: values.customer_id,
        type: 'subscription', // 默认使用订阅类型
        max_devices: values.max_devices,
        duration_days: values.duration_days,
        features: values.features || [],
        notes: values.remark,
      };
      if (currentLicense) {
        await licenseApi.update(currentLicense.id, submitData);
        message.success('更新成功');
      } else {
        await licenseApi.create(submitData);
        message.success('创建成功');
      }
      setModalVisible(false);
      fetchData(pagination.current, pagination.pageSize);
    } catch (error) {
      console.error(error);
    }
  };

  const handleRevoke = (record: any) => {
    modal.confirm({
      title: '吊销授权',
      content: '确定要吊销此授权吗？吊销后将无法使用。',
      onOk: async () => {
        try {
          await licenseApi.revoke(record.id);
          message.success('授权已吊销');
          fetchData(pagination.current, pagination.pageSize);
        } catch (error) {
          console.error(error);
        }
      },
    });
  };

  const handleReset = (record: any) => {
    modal.confirm({
      title: '重置设备',
      content: '确定要重置此授权的设备绑定吗？',
      onOk: async () => {
        try {
          await licenseApi.resetDevices(record.id);
          message.success('设备已重置');
          fetchData(pagination.current, pagination.pageSize);
        } catch (error) {
          console.error(error);
        }
      },
    });
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    message.success('已复制到剪贴板');
  };

  const handleTableChange = (pag: any) => {
    fetchData(pag.current, pag.pageSize);
  };

  const handleSearch = (values: any) => {
    setFilters(values);
    fetchData(1, pagination.pageSize, values);
  };

  const getStatusTag = (status: string) => {
    const statusMap: Record<string, { color: string; text: string }> = {
      pending: { color: 'default', text: '待激活' },
      active: { color: 'green', text: '已激活' },
      expired: { color: 'red', text: '已过期' },
      revoked: { color: 'orange', text: '已吊销' },
    };
    const s = statusMap[status] || { color: 'default', text: status };
    return <Tag color={s.color}>{s.text}</Tag>;
  };

  const columns = [
    {
      title: '授权码',
      dataIndex: 'license_key',
      key: 'license_key',
      render: (text: string) => (
        <Space>
          <code>{text?.slice(0, 16)}...</code>
          <Button type="link" size="small" icon={<CopyOutlined />} onClick={() => copyToClipboard(text)} />
        </Space>
      ),
    },
    {
      title: '应用',
      dataIndex: 'app_id',
      key: 'app_id',
      render: (id: string) => apps.find(a => a.id === id)?.name || id,
    },
    { title: '状态', dataIndex: 'status', key: 'status', render: (s: string) => getStatusTag(s) },
    { title: '设备数', dataIndex: 'max_devices', key: 'max_devices' },
    {
      title: '过期时间',
      dataIndex: 'expires_at',
      key: 'expires_at',
      render: (v: string) => v ? dayjs(v).format('YYYY-MM-DD') : '永久',
    },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', render: (v: string) => v?.slice(0, 10) },
    {
      title: '操作', key: 'action', width: 280,
      render: (_: any, record: any) => (
        <Space>
          <Button type="link" size="small" onClick={() => handleView(record)}>详情</Button>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>编辑</Button>
          <Button type="link" size="small" icon={<ReloadOutlined />} onClick={() => handleReset(record)}>重置</Button>
          {record.status === 'active' && (
            <Button type="link" size="small" danger icon={<StopOutlined />} onClick={() => handleRevoke(record)}>吊销</Button>
          )}
          <Button type="link" size="small" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record)}>删除</Button>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <h2 style={{ margin: 0 }}>授权管理</h2>
        <Space>
          <Button icon={<DownloadOutlined />} onClick={() => window.open(exportApi.licenses(filters), '_blank')}>导出</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>创建授权</Button>
        </Space>
      </div>

      {/* 搜索筛选 */}
      <Form layout="inline" onFinish={handleSearch} style={{ marginBottom: 16 }}>
        <Form.Item name="app_id">
          <Select placeholder="选择应用" allowClear style={{ width: 150 }}>
            {apps.map(app => <Option key={app.id} value={app.id}>{app.name}</Option>)}
          </Select>
        </Form.Item>
        <Form.Item name="status">
          <Select placeholder="状态" allowClear style={{ width: 120 }}>
            <Option value="pending">待激活</Option>
            <Option value="active">已激活</Option>
            <Option value="expired">已过期</Option>
            <Option value="revoked">已吊销</Option>
          </Select>
        </Form.Item>
        <Form.Item name="keyword">
          <Input placeholder="搜索授权码" allowClear />
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit">搜索</Button>
        </Form.Item>
      </Form>

      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        pagination={pagination}
        onChange={handleTableChange}
      />

      {/* 创建/编辑弹窗 */}
      <Modal
        title={currentLicense ? '编辑授权' : '创建授权'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="app_id" label="应用" rules={[{ required: true, message: '请选择应用' }]}>
            <Select placeholder="选择应用" disabled={!!currentLicense} onChange={handleAppChange}>
              {apps.map(app => <Option key={app.id} value={app.id}>{app.name}</Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="customer_id" label="团队成员">
            <Select placeholder="选择团队成员（可选）" allowClear showSearch optionFilterProp="children">
              {members.map(m => <Option key={m.id} value={m.id}>{m.name || m.email}</Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="max_devices" label="最大设备数" initialValue={1}>
            <InputNumber min={1} max={100} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="duration_days" label="有效天数" initialValue={365} rules={[{ required: true, message: '请输入有效天数' }]} extra="-1表示永久有效">
            <InputNumber min={-1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="features" label="功能权限">
            {selectedAppFeatures.length > 0 ? (
              <Checkbox.Group>
                <Space direction="vertical">
                  {selectedAppFeatures.map(feature => (
                    <Checkbox key={feature} value={feature}>{feature}</Checkbox>
                  ))}
                </Space>
              </Checkbox.Group>
            ) : (
              <span style={{ color: '#999' }}>请先选择应用，或该应用未配置功能列表</span>
            )}
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea placeholder="备注信息" rows={2} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情弹窗 */}
      <Modal
        title="授权详情"
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={null}
        width={700}
      >
        {currentLicense && (
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="授权码" span={2}>
              <code>{currentLicense.license_key}</code>
              <Button type="link" size="small" icon={<CopyOutlined />} onClick={() => copyToClipboard(currentLicense.license_key)} />
            </Descriptions.Item>
            <Descriptions.Item label="应用">
              {apps.find(a => a.id === currentLicense.app_id)?.name || currentLicense.app_id}
            </Descriptions.Item>
            <Descriptions.Item label="团队成员">
              {members.find(m => m.id === currentLicense.customer_id)?.name || currentLicense.customer_email || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="状态">{getStatusTag(currentLicense.status)}</Descriptions.Item>
            <Descriptions.Item label="最大设备数">{currentLicense.max_devices}</Descriptions.Item>
            <Descriptions.Item label="已用设备数">{currentLicense.used_devices || 0}</Descriptions.Item>
            <Descriptions.Item label="有效天数">
              {currentLicense.duration_days === -1 ? '永久' : `${currentLicense.duration_days} 天`}
            </Descriptions.Item>
            <Descriptions.Item label="过期时间">
              {currentLicense.expires_at ? dayjs(currentLicense.expires_at).format('YYYY-MM-DD HH:mm') : '激活后计算'}
            </Descriptions.Item>
            <Descriptions.Item label="激活时间">
              {currentLicense.activated_at ? dayjs(currentLicense.activated_at).format('YYYY-MM-DD HH:mm') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="最后心跳">
              {currentLicense.last_heartbeat ? dayjs(currentLicense.last_heartbeat).format('YYYY-MM-DD HH:mm:ss') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间" span={2}>
              {dayjs(currentLicense.created_at).format('YYYY-MM-DD HH:mm')}
            </Descriptions.Item>
            <Descriptions.Item label="功能权限" span={2}>
              {currentLicense.features && currentLicense.features !== '[]' ? currentLicense.features : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="备注" span={2}>{currentLicense.notes || '-'}</Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
};

export default Licenses;
