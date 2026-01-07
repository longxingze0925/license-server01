import React, { useEffect, useState } from 'react';
import { Table, Button, Space, Modal, Form, Input, message, Tag, Select, Descriptions, Tabs, Spin } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, CodeOutlined } from '@ant-design/icons';
import { secureScriptApi, appApi } from '../api';

const SecureScripts: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [pageLoading, setPageLoading] = useState(true);
  const [data, setData] = useState<any[]>([]);
  const [apps, setApps] = useState<any[]>([]);
  const [selectedApp, setSelectedApp] = useState<string>('');
  const [modalVisible, setModalVisible] = useState(false);
  const [contentModalVisible, setContentModalVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [currentScript, setCurrentScript] = useState<any>(null);
  const [deliveries, setDeliveries] = useState<any[]>([]);
  const [form] = Form.useForm();
  const [contentForm] = Form.useForm();
  const [editMode, setEditMode] = useState(false);

  useEffect(() => {
    fetchApps();
  }, []);

  useEffect(() => {
    if (selectedApp) {
      fetchData();
    }
  }, [selectedApp]);

  const fetchApps = async () => {
    try {
      const result: any = await appApi.list();
      const appList = Array.isArray(result) ? result : (result?.list || []);
      setApps(appList);
      if (appList.length > 0) {
        setSelectedApp(appList[0].id);
      }
    } catch (error) {
      console.error('fetchApps error:', error);
      setApps([]);
    } finally {
      setPageLoading(false);
    }
  };

  const fetchData = async () => {
    if (!selectedApp) return;
    setLoading(true);
    try {
      const result: any = await secureScriptApi.list(selectedApp);
      const list = result?.list ?? (Array.isArray(result) ? result : []);
      setData(list);
    } catch (error) {
      console.error(error);
      setData([]);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setCurrentScript(null);
    setEditMode(false);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: any) => {
    setCurrentScript(record);
    setEditMode(true);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleView = async (record: any) => {
    try {
      const detail = await secureScriptApi.get(record.id);
      setCurrentScript(detail);
      const deliveriesResult: any = await secureScriptApi.getDeliveries(record.id);
      const list = deliveriesResult?.list ?? (Array.isArray(deliveriesResult) ? deliveriesResult : []);
      setDeliveries(list);
      setDetailVisible(true);
    } catch (error) {
      console.error(error);
    }
  };

  const handleEditContent = (record: any) => {
    setCurrentScript(record);
    contentForm.setFieldsValue({ content: record.content || '' });
    setContentModalVisible(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editMode && currentScript) {
        await secureScriptApi.update(currentScript.id, values);
        message.success('更新成功');
      } else {
        await secureScriptApi.create(selectedApp, values);
        message.success('创建成功');
      }
      setModalVisible(false);
      fetchData();
    } catch (error) {
      console.error(error);
    }
  };

  const handleContentSubmit = async () => {
    try {
      const values = await contentForm.validateFields();
      await secureScriptApi.updateContent(currentScript.id, values);
      message.success('脚本内容已更新');
      setContentModalVisible(false);
      fetchData();
    } catch (error) {
      console.error(error);
    }
  };

  const handlePublish = async (id: string) => {
    Modal.confirm({
      title: '确认发布',
      content: '确定要发布此脚本吗？发布后客户端将可以获取此脚本。',
      onOk: async () => {
        try {
          await secureScriptApi.publish(id);
          message.success('发布成功');
          fetchData();
        } catch (error) {
          console.error(error);
        }
      },
    });
  };

  const handleDeprecate = async (id: string) => {
    Modal.confirm({
      title: '确认废弃',
      content: '确定要废弃此脚本吗？废弃后客户端将无法获取此脚本。',
      onOk: async () => {
        try {
          await secureScriptApi.deprecate(id);
          message.success('已废弃');
          fetchData();
        } catch (error) {
          console.error(error);
        }
      },
    });
  };

  const handleDelete = (record: any) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除脚本 "${record.name}" 吗？`,
      onOk: async () => {
        try {
          await secureScriptApi.delete(record.id);
          message.success('删除成功');
          fetchData();
        } catch (error) {
          console.error(error);
        }
      },
    });
  };

  const getStatusTag = (status: string) => {
    const statusMap: Record<string, { color: string; text: string }> = {
      draft: { color: 'default', text: '草稿' },
      active: { color: 'green', text: '已发布' },
      deprecated: { color: 'orange', text: '已废弃' },
    };
    const s = statusMap[status] || { color: 'default', text: status };
    return <Tag color={s.color}>{s.text}</Tag>;
  };

  const getTypeTag = (type: string) => {
    const typeMap: Record<string, { color: string; text: string }> = {
      python: { color: 'blue', text: 'Python' },
      lua: { color: 'purple', text: 'Lua' },
      instruction: { color: 'cyan', text: '指令脚本' },
    };
    const t = typeMap[type] || { color: 'default', text: type };
    return <Tag color={t.color}>{t.text}</Tag>;
  };

  const columns = [
    { title: '脚本名称', dataIndex: 'name', key: 'name' },
    { title: '脚本类型', dataIndex: 'script_type', key: 'script_type', render: getTypeTag },
    { title: '版本', dataIndex: 'version', key: 'version' },
    { title: '所需功能', dataIndex: 'required_feature', key: 'required_feature', render: (v: string) => v || '-' },
    { title: '状态', dataIndex: 'status', key: 'status', render: getStatusTag },
    { title: '下发次数', dataIndex: 'delivery_count', key: 'delivery_count', render: (v: number) => v || 0 },
    { title: '更新时间', dataIndex: 'updated_at', key: 'updated_at', render: (v: string) => v?.slice(0, 10) },
    {
      title: '操作', key: 'action',
      render: (_: any, record: any) => (
        <Space>
          <Button type="link" size="small" onClick={() => handleView(record)}>详情</Button>
          <Button type="link" size="small" icon={<CodeOutlined />} onClick={() => handleEditContent(record)}>编辑内容</Button>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>编辑</Button>
          {record.status === 'draft' && (
            <Button type="link" size="small" onClick={() => handlePublish(record.id)}>发布</Button>
          )}
          {record.status === 'active' && (
            <Button type="link" size="small" onClick={() => handleDeprecate(record.id)}>废弃</Button>
          )}
          <Button type="link" size="small" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record)}>删除</Button>
        </Space>
      ),
    },
  ];

  const deliveryColumns = [
    { title: '设备ID', dataIndex: 'device_id', key: 'device_id', ellipsis: true },
    { title: '机器码', dataIndex: 'machine_id', key: 'machine_id', ellipsis: true },
    { title: '脚本版本', dataIndex: 'script_version', key: 'script_version' },
    {
      title: '状态', dataIndex: 'status', key: 'status',
      render: (status: string) => {
        const map: Record<string, { color: string; text: string }> = {
          delivered: { color: 'green', text: '已下发' },
          failed: { color: 'red', text: '失败' },
          pending: { color: 'blue', text: '待下发' },
        };
        const s = map[status] || { color: 'default', text: status };
        return <Tag color={s.color}>{s.text}</Tag>;
      }
    },
    { title: '下发时间', dataIndex: 'delivered_at', key: 'delivered_at', render: (v: string) => v?.slice(0, 19).replace('T', ' ') || '-' },
  ];

  if (pageLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%', minHeight: 300 }}>
        <Spin size="large" tip="加载中..." />
      </div>
    );
  }

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2 style={{ margin: 0 }}>安全脚本管理</h2>
        <Space>
          <Select
            style={{ width: 200 }}
            placeholder="选择应用"
            value={selectedApp || undefined}
            onChange={setSelectedApp}
            options={apps.map(app => ({ label: app.name, value: app.id }))}
          />
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate} disabled={!selectedApp}>
            创建脚本
          </Button>
        </Space>
      </div>

      <Table columns={columns} dataSource={data} rowKey="id" loading={loading} />

      {/* 创建/编辑弹窗 */}
      <Modal
        title={editMode ? '编辑脚本' : '创建脚本'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="脚本名称" rules={[{ required: true, message: '请输入脚本名称' }]}>
            <Input placeholder="请输入脚本名称" />
          </Form.Item>
          <Form.Item name="script_type" label="脚本类型" initialValue="python" rules={[{ required: true }]}>
            <Select options={[
              { label: 'Python', value: 'python' },
              { label: 'Lua', value: 'lua' },
              { label: '指令脚本', value: 'instruction' },
            ]} />
          </Form.Item>
          <Form.Item name="version" label="版本号" initialValue="1.0.0" rules={[{ required: true, message: '请输入版本号' }]}>
            <Input placeholder="如：1.0.0" />
          </Form.Item>
          <Form.Item name="required_feature" label="所需功能">
            <Input placeholder="需要此功能权限才能获取脚本，留空表示无限制" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="脚本描述..." />
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑内容弹窗 */}
      <Modal
        title="编辑脚本内容"
        open={contentModalVisible}
        onOk={handleContentSubmit}
        onCancel={() => setContentModalVisible(false)}
        width={800}
      >
        <Form form={contentForm} layout="vertical">
          <Form.Item name="content" label="脚本内容" rules={[{ required: true, message: '请输入脚本内容' }]}>
            <Input.TextArea rows={20} placeholder="请输入脚本内容..." style={{ fontFamily: 'monospace' }} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情弹窗 */}
      <Modal
        title="脚本详情"
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={null}
        width={900}
      >
        {currentScript && (
          <Tabs items={[
            {
              key: 'info',
              label: '基本信息',
              children: (
                <Descriptions column={2} bordered size="small">
                  <Descriptions.Item label="脚本名称">{currentScript.name}</Descriptions.Item>
                  <Descriptions.Item label="脚本类型">{getTypeTag(currentScript.script_type)}</Descriptions.Item>
                  <Descriptions.Item label="版本">{currentScript.version}</Descriptions.Item>
                  <Descriptions.Item label="状态">{getStatusTag(currentScript.status)}</Descriptions.Item>
                  <Descriptions.Item label="所需功能">{currentScript.required_feature || '-'}</Descriptions.Item>
                  <Descriptions.Item label="下发次数">{currentScript.delivery_count || 0}</Descriptions.Item>
                  <Descriptions.Item label="描述" span={2}>{currentScript.description || '-'}</Descriptions.Item>
                  <Descriptions.Item label="创建时间">{currentScript.created_at?.slice(0, 19).replace('T', ' ')}</Descriptions.Item>
                  <Descriptions.Item label="更新时间">{currentScript.updated_at?.slice(0, 19).replace('T', ' ')}</Descriptions.Item>
                </Descriptions>
              ),
            },
            {
              key: 'content',
              label: '脚本内容',
              children: (
                <Input.TextArea
                  value={currentScript.content || '暂无内容'}
                  rows={15}
                  readOnly
                  style={{ fontFamily: 'monospace' }}
                />
              ),
            },
            {
              key: 'deliveries',
              label: '下发记录',
              children: <Table columns={deliveryColumns} dataSource={deliveries} rowKey="id" size="small" pagination={{ pageSize: 10 }} />,
            },
          ]} />
        )}
      </Modal>
    </div>
  );
};

export default SecureScripts;
