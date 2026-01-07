import React, { useEffect, useState } from 'react';
import { Table, Button, Space, Modal, Form, Select, message, Tag, InputNumber, Descriptions, Input, Checkbox } from 'antd';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import { subscriptionApi, appApi, customerApi } from '../api';
import dayjs from 'dayjs';

const { Option } = Select;

const Subscriptions: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<any[]>([]);
  const [apps, setApps] = useState<any[]>([]);
  const [customers, setCustomers] = useState<any[]>([]);
  const [modalVisible, setModalVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [currentSubscription, setCurrentSubscription] = useState<any>(null);
  const [form] = Form.useForm();
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 });
  const [filters, setFilters] = useState<any>({});
  const [selectedAppFeatures, setSelectedAppFeatures] = useState<string[]>([]);

  useEffect(() => {
    fetchData();
    fetchApps();
    fetchCustomers();
  }, []);

  const fetchData = async (page = 1, pageSize = 10, filterParams = filters) => {
    setLoading(true);
    try {
      const result: any = await subscriptionApi.list({ page, page_size: pageSize, ...filterParams });
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

  const fetchCustomers = async () => {
    try {
      const result: any = await customerApi.list({ page_size: 100 });
      setCustomers(result.list || []);
    } catch (error) {
      console.error(error);
    }
  };

  const handleCreate = () => {
    setCurrentSubscription(null);
    form.resetFields();
    setSelectedAppFeatures([]);
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
      const detail = await subscriptionApi.get(record.id);
      setCurrentSubscription(detail);
      setDetailVisible(true);
    } catch (error) {
      console.error(error);
    }
  };

  const handleDelete = (record: any) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除此订阅吗？',
      onOk: async () => {
        try {
          await subscriptionApi.delete(record.id);
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
      if (currentSubscription) {
        await subscriptionApi.update(currentSubscription.id, values);
        message.success('更新成功');
      } else {
        await subscriptionApi.create(values);
        message.success('创建成功');
      }
      setModalVisible(false);
      fetchData(pagination.current, pagination.pageSize);
    } catch (error) {
      console.error(error);
    }
  };

  const handleRenew = (record: any) => {
    Modal.confirm({
      title: '续费订阅',
      content: (
        <Form id="renewForm">
          <Form.Item label="续费天数" name="days" initialValue={30}>
            <InputNumber min={1} max={3650} />
          </Form.Item>
        </Form>
      ),
      onOk: async () => {
        try {
          await subscriptionApi.renew(record.id, { days: 30 });
          message.success('续费成功');
          fetchData(pagination.current, pagination.pageSize);
        } catch (error) {
          console.error(error);
        }
      },
    });
  };

  const handleCancel = (record: any) => {
    Modal.confirm({
      title: '取消订阅',
      content: '确定要取消此订阅吗？',
      onOk: async () => {
        try {
          await subscriptionApi.cancel(record.id);
          message.success('订阅已取消');
          fetchData(pagination.current, pagination.pageSize);
        } catch (error) {
          console.error(error);
        }
      },
    });
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
      active: { color: 'green', text: '有效' },
      expired: { color: 'red', text: '已过期' },
      cancelled: { color: 'orange', text: '已取消' },
      suspended: { color: 'default', text: '已暂停' },
    };
    const s = statusMap[status] || { color: 'default', text: status };
    return <Tag color={s.color}>{s.text}</Tag>;
  };

  const columns = [
    {
      title: '用户',
      key: 'user',
      render: (_: any, record: any) => record.customer_email || record.customer_name || '-',
    },
    {
      title: '应用',
      key: 'app',
      render: (_: any, record: any) => record.app_name || '-',
    },
    { title: '状态', dataIndex: 'status', key: 'status', render: (s: string) => getStatusTag(s) },
    { title: '最大设备数', dataIndex: 'max_devices', key: 'max_devices' },
    {
      title: '剩余天数',
      dataIndex: 'remaining_days',
      key: 'remaining_days',
      render: (v: number) => v === -1 ? '永久' : `${v} 天`,
    },
    {
      title: '过期时间',
      dataIndex: 'expire_at',
      key: 'expire_at',
      render: (v: string) => v ? dayjs(v).format('YYYY-MM-DD') : '永久',
    },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', render: (v: string) => v?.slice(0, 10) },
    {
      title: '操作', key: 'action', width: 250,
      render: (_: any, record: any) => (
        <Space>
          <Button type="link" size="small" onClick={() => handleView(record)}>详情</Button>
          <Button type="link" size="small" onClick={() => handleRenew(record)}>续费</Button>
          {record.status === 'active' && (
            <Button type="link" size="small" danger onClick={() => handleCancel(record)}>取消</Button>
          )}
          <Button type="link" size="small" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record)}>删除</Button>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <h2 style={{ margin: 0 }}>订阅管理</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>创建订阅</Button>
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
            <Option value="active">有效</Option>
            <Option value="expired">已过期</Option>
            <Option value="cancelled">已取消</Option>
          </Select>
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
        title={currentSubscription ? '编辑订阅' : '创建订阅'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="app_id" label="应用" rules={[{ required: true, message: '请选择应用' }]}>
            <Select placeholder="选择应用" disabled={!!currentSubscription} onChange={handleAppChange}>
              {apps.map(app => <Option key={app.id} value={app.id}>{app.name}</Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="customer_id" label="客户">
            <Select placeholder="选择客户（可选）" showSearch optionFilterProp="children" allowClear disabled={!!currentSubscription}>
              {customers.map(c => <Option key={c.id} value={c.id}>{c.email} ({c.name || '未设置'})</Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="max_devices" label="最大设备数" initialValue={1}>
            <InputNumber min={1} max={100} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="days" label="有效天数" initialValue={365} rules={[{ required: true, message: '请输入有效天数' }]} extra="-1表示永久有效">
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
          <Form.Item name="notes" label="备注">
            <Input.TextArea placeholder="备注信息" rows={2} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情弹窗 */}
      <Modal
        title="订阅详情"
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={null}
        width={600}
      >
        {currentSubscription && (
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="应用">{currentSubscription.application?.name || '-'}</Descriptions.Item>
            <Descriptions.Item label="客户">
              {customers.find(c => c.id === currentSubscription.customer_id)?.email || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="状态">{getStatusTag(currentSubscription.status)}</Descriptions.Item>
            <Descriptions.Item label="最大设备数">{currentSubscription.max_devices}</Descriptions.Item>
            <Descriptions.Item label="剩余天数">
              {currentSubscription.remaining_days === -1 ? '永久' : `${currentSubscription.remaining_days} 天`}
            </Descriptions.Item>
            <Descriptions.Item label="开始时间">
              {currentSubscription.start_at ? dayjs(currentSubscription.start_at).format('YYYY-MM-DD HH:mm') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="过期时间">
              {currentSubscription.expire_at ? dayjs(currentSubscription.expire_at).format('YYYY-MM-DD HH:mm') : '永久'}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间" span={2}>
              {dayjs(currentSubscription.created_at).format('YYYY-MM-DD HH:mm')}
            </Descriptions.Item>
            <Descriptions.Item label="功能权限" span={2}>
              {currentSubscription.features && currentSubscription.features !== '[]' ? currentSubscription.features : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="备注" span={2}>{currentSubscription.notes || '-'}</Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
};

export default Subscriptions;
