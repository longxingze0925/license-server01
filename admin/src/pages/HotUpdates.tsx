import React, { useEffect, useState } from 'react';
import { Table, Button, Space, Modal, Form, Input, InputNumber, message, Tag, Select, Upload, Progress, Descriptions, Tabs, Spin, App } from 'antd';
import { PlusOutlined, UploadOutlined, RollbackOutlined } from '@ant-design/icons';
import { hotUpdateApi, appApi } from '../api';

const HotUpdates: React.FC = () => {
  const { modal } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [pageLoading, setPageLoading] = useState(true);
  const [data, setData] = useState<any[]>([]);
  const [apps, setApps] = useState<any[]>([]);
  const [selectedApp, setSelectedApp] = useState<string>('');
  const [modalVisible, setModalVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [currentUpdate, setCurrentUpdate] = useState<any>(null);
  const [logs, setLogs] = useState<any[]>([]);
  const [form] = Form.useForm();
  const [fileList, setFileList] = useState<any[]>([]);

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
      console.error(error);
      setApps([]);
    } finally {
      setPageLoading(false);
    }
  };

  const fetchData = async () => {
    if (!selectedApp) return;
    setLoading(true);
    try {
      const result: any = await hotUpdateApi.list(selectedApp);
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
    setCurrentUpdate(null);
    form.resetFields();
    setFileList([]);
    setModalVisible(true);
  };

  const handleView = async (record: any) => {
    try {
      const detail = await hotUpdateApi.get(record.id);
      setCurrentUpdate(detail);
      const logsResult: any = await hotUpdateApi.getLogs(record.id);
      const list = logsResult?.list ?? (Array.isArray(logsResult) ? logsResult : []);
      setLogs(list);
      setDetailVisible(true);
    } catch (error) {
      console.error(error);
    }
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (fileList.length === 0) {
        message.error('请选择更新包文件');
        return;
      }
      const formData = new FormData();
      formData.append('file', fileList[0].originFileObj);
      formData.append('version', values.version);
      formData.append('version_code', values.version_code.toString());
      formData.append('update_type', values.update_type);
      formData.append('changelog', values.changelog || '');
      formData.append('rollout_percentage', (values.rollout_percentage || 100).toString());
      formData.append('force_update', values.force_update ? 'true' : 'false');

      await hotUpdateApi.create(selectedApp, formData);
      message.success('创建成功');
      setModalVisible(false);
      fetchData();
    } catch (error) {
      console.error(error);
    }
  };

  const handlePublish = async (id: string) => {
    modal.confirm({
      title: '确认发布',
      content: '确定要发布此热更新吗？',
      onOk: async () => {
        try {
          await hotUpdateApi.publish(id);
          message.success('发布成功');
          fetchData();
        } catch (error) {
          console.error(error);
        }
      },
    });
  };

  const handleRollback = async (id: string) => {
    modal.confirm({
      title: '确认回滚',
      content: '确定要回滚此热更新吗？这将使该版本失效。',
      onOk: async () => {
        try {
          await hotUpdateApi.rollback(id);
          message.success('回滚成功');
          fetchData();
        } catch (error) {
          console.error(error);
        }
      },
    });
  };

  const handleDelete = (record: any) => {
    modal.confirm({
      title: '确认删除',
      content: `确定要删除版本 "${record.version}" 吗？`,
      onOk: async () => {
        try {
          await hotUpdateApi.delete(record.id);
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
      published: { color: 'green', text: '已发布' },
      deprecated: { color: 'orange', text: '已废弃' },
      rolled_back: { color: 'red', text: '已回滚' },
    };
    const s = statusMap[status] || { color: 'default', text: status };
    return <Tag color={s.color}>{s.text}</Tag>;
  };

  const columns = [
    { title: '版本号', dataIndex: 'version', key: 'version' },
    { title: '版本码', dataIndex: 'version_code', key: 'version_code' },
    {
      title: '更新类型', dataIndex: 'update_type', key: 'update_type',
      render: (type: string) => type === 'full' ? '完整更新' : '增量更新'
    },
    { title: '文件大小', dataIndex: 'file_size', key: 'file_size', render: (v: number) => v ? `${(v / 1024).toFixed(2)} KB` : '-' },
    {
      title: '灰度比例', dataIndex: 'rollout_percentage', key: 'rollout_percentage',
      render: (v: number) => <Progress percent={v} size="small" style={{ width: 80 }} />
    },
    { title: '状态', dataIndex: 'status', key: 'status', render: getStatusTag },
    { title: '强制更新', dataIndex: 'force_update', key: 'force_update', render: (v: boolean) => v ? '是' : '否' },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', render: (v: string) => v?.slice(0, 10) },
    {
      title: '操作', key: 'action',
      render: (_: any, record: any) => (
        <Space>
          <Button type="link" size="small" onClick={() => handleView(record)}>详情</Button>
          {record.status === 'draft' && (
            <Button type="link" size="small" onClick={() => handlePublish(record.id)}>发布</Button>
          )}
          {record.status === 'published' && (
            <Button type="link" size="small" icon={<RollbackOutlined />} onClick={() => handleRollback(record.id)}>回滚</Button>
          )}
          {record.status === 'draft' && (
            <Button type="link" size="small" danger onClick={() => handleDelete(record)}>删除</Button>
          )}
        </Space>
      ),
    },
  ];

  const logColumns = [
    { title: '设备ID', dataIndex: 'device_id', key: 'device_id', ellipsis: true },
    { title: '机器码', dataIndex: 'machine_id', key: 'machine_id', ellipsis: true },
    {
      title: '状态', dataIndex: 'status', key: 'status',
      render: (status: string) => {
        const map: Record<string, { color: string; text: string }> = {
          success: { color: 'green', text: '成功' },
          failed: { color: 'red', text: '失败' },
          pending: { color: 'blue', text: '进行中' },
        };
        const s = map[status] || { color: 'default', text: status };
        return <Tag color={s.color}>{s.text}</Tag>;
      }
    },
    { title: '错误信息', dataIndex: 'error_message', key: 'error_message', ellipsis: true },
    { title: '更新时间', dataIndex: 'updated_at', key: 'updated_at', render: (v: string) => v?.slice(0, 19).replace('T', ' ') },
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
        <h2 style={{ margin: 0 }}>热更新管理</h2>
        <Space>
          <Select
            style={{ width: 200 }}
            placeholder="选择应用"
            value={selectedApp || undefined}
            onChange={setSelectedApp}
            options={apps.map(app => ({ label: app.name, value: app.id }))}
          />
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate} disabled={!selectedApp}>
            创建热更新
          </Button>
        </Space>
      </div>

      <Table columns={columns} dataSource={data} rowKey="id" loading={loading} />

      {/* 创建弹窗 */}
      <Modal
        title="创建热更新"
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => setModalVisible(false)}
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="version" label="版本号" rules={[{ required: true, message: '请输入版本号' }]}>
            <Input placeholder="如：1.0.1" />
          </Form.Item>
          <Form.Item name="version_code" label="版本码" rules={[{ required: true, message: '请输入版本码' }]}>
            <InputNumber min={1} style={{ width: '100%' }} placeholder="递增数字" />
          </Form.Item>
          <Form.Item name="update_type" label="更新类型" initialValue="full" rules={[{ required: true }]}>
            <Select options={[
              { label: '完整更新', value: 'full' },
              { label: '增量更新', value: 'patch' },
            ]} />
          </Form.Item>
          <Form.Item label="更新包文件" required>
            <Upload
              beforeUpload={() => false}
              fileList={fileList}
              onChange={({ fileList }) => setFileList(fileList.slice(-1))}
              maxCount={1}
            >
              <Button icon={<UploadOutlined />}>选择文件</Button>
            </Upload>
          </Form.Item>
          <Form.Item name="rollout_percentage" label="灰度比例(%)" initialValue={100}>
            <InputNumber min={1} max={100} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="force_update" label="强制更新" valuePropName="checked">
            <Select options={[
              { label: '否', value: false },
              { label: '是', value: true },
            ]} defaultValue={false} />
          </Form.Item>
          <Form.Item name="changelog" label="更新日志">
            <Input.TextArea rows={3} placeholder="本次更新内容..." />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情弹窗 */}
      <Modal
        title="热更新详情"
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={null}
        width={900}
      >
        {currentUpdate && (
          <Tabs items={[
            {
              key: 'info',
              label: '基本信息',
              children: (
                <Descriptions column={2} bordered size="small">
                  <Descriptions.Item label="版本号">{currentUpdate.version}</Descriptions.Item>
                  <Descriptions.Item label="版本码">{currentUpdate.version_code}</Descriptions.Item>
                  <Descriptions.Item label="更新类型">{currentUpdate.update_type === 'full' ? '完整更新' : '增量更新'}</Descriptions.Item>
                  <Descriptions.Item label="状态">{getStatusTag(currentUpdate.status)}</Descriptions.Item>
                  <Descriptions.Item label="文件大小">{currentUpdate.file_size ? `${(currentUpdate.file_size / 1024).toFixed(2)} KB` : '-'}</Descriptions.Item>
                  <Descriptions.Item label="灰度比例">{currentUpdate.rollout_percentage}%</Descriptions.Item>
                  <Descriptions.Item label="强制更新">{currentUpdate.force_update ? '是' : '否'}</Descriptions.Item>
                  <Descriptions.Item label="下载次数">{currentUpdate.download_count || 0}</Descriptions.Item>
                  <Descriptions.Item label="更新日志" span={2}>{currentUpdate.changelog || '-'}</Descriptions.Item>
                  <Descriptions.Item label="创建时间">{currentUpdate.created_at?.slice(0, 19).replace('T', ' ')}</Descriptions.Item>
                  <Descriptions.Item label="发布时间">{currentUpdate.published_at?.slice(0, 19).replace('T', ' ') || '-'}</Descriptions.Item>
                </Descriptions>
              ),
            },
            {
              key: 'logs',
              label: '更新日志',
              children: <Table columns={logColumns} dataSource={logs} rowKey="id" size="small" pagination={{ pageSize: 10 }} />,
            },
          ]} />
        )}
      </Modal>
    </div>
  );
};

export default HotUpdates;
