import React, { useEffect, useState } from 'react';
import { Row, Col, Card, Statistic, Spin } from 'antd';
import {
  UserOutlined,
  AppstoreOutlined,
  KeyOutlined,
  DesktopOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloseCircleOutlined,
  CrownOutlined,
} from '@ant-design/icons';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import { statsApi } from '../api';
import { useAuthStore } from '../store';

const COLORS = ['#1890ff', '#52c41a', '#faad14', '#f5222d', '#722ed1'];

const Dashboard: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<any>(null);
  const [licenseTrend, setLicenseTrend] = useState<any[]>([]);
  const [licenseType, setLicenseType] = useState<any[]>([]);
  const { user } = useAuthStore();

  const isViewer = user?.role === 'viewer';

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      const [dashboard, trend, typeData] = await Promise.all([
        statsApi.dashboard(),
        statsApi.licenseTrend(),
        statsApi.licenseType(),
      ]);
      setData(dashboard);
      setLicenseTrend((trend as any) || []);
      setLicenseType((typeData as any) || []);
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;
  }

  return (
    <div>
      <h2 style={{ marginBottom: 24 }}>仪表盘</h2>

      {/* 第一行：核心统计 */}
      <Row gutter={[16, 16]}>
        {!isViewer && (
          <>
            <Col xs={24} sm={12} md={6}>
              <Card style={{ height: 120 }}>
                <Statistic
                  title="客户数"
                  value={data?.customers?.total || 0}
                  prefix={<UserOutlined />}
                  valueStyle={{ color: '#1890ff' }}
                />
                <div style={{ marginTop: 8, color: '#999', fontSize: 12 }}>
                  今日新增: {data?.customers?.today_new || 0}
                </div>
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card style={{ height: 120 }}>
                <Statistic
                  title="应用数量"
                  value={data?.applications?.total || 0}
                  prefix={<AppstoreOutlined />}
                  valueStyle={{ color: '#722ed1' }}
                />
              </Card>
            </Col>
          </>
        )}
        <Col xs={24} sm={12} md={isViewer ? 8 : 6}>
          <Card style={{ height: 120 }}>
            <Statistic
              title="授权总数"
              value={data?.licenses?.total || 0}
              prefix={<KeyOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
            <div style={{ marginTop: 8, color: '#999', fontSize: 12 }}>
              今日新增: {data?.licenses?.today_new || 0}
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={isViewer ? 8 : 6}>
          <Card style={{ height: 120 }}>
            <Statistic
              title="订阅总数"
              value={data?.subscriptions?.total || 0}
              prefix={<CrownOutlined />}
              valueStyle={{ color: '#faad14' }}
            />
            <div style={{ marginTop: 8, color: '#999', fontSize: 12 }}>
              有效订阅: {data?.subscriptions?.active || 0}
            </div>
          </Card>
        </Col>
        {isViewer && (
          <Col xs={24} sm={12} md={8}>
            <Card style={{ height: 120 }}>
              <Statistic
                title="设备总数"
                value={data?.devices?.total || 0}
                prefix={<DesktopOutlined />}
                valueStyle={{ color: '#13c2c2' }}
              />
              <div style={{ marginTop: 8, color: '#999', fontSize: 12 }}>
                活跃设备: {data?.devices?.active || 0}
              </div>
            </Card>
          </Col>
        )}
      </Row>

      {/* 第二行：授权状态 + 设备（管理员） */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} sm={12} md={6}>
          <Card style={{ height: 120 }}>
            <Statistic
              title="激活授权"
              value={data?.licenses?.active || 0}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card style={{ height: 120 }}>
            <Statistic
              title="待激活"
              value={data?.licenses?.pending || 0}
              prefix={<ClockCircleOutlined />}
              valueStyle={{ color: '#faad14' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card style={{ height: 120 }}>
            <Statistic
              title="已过期"
              value={data?.licenses?.expired || 0}
              prefix={<CloseCircleOutlined />}
              valueStyle={{ color: '#f5222d' }}
            />
          </Card>
        </Col>
        {!isViewer && (
          <Col xs={24} sm={12} md={6}>
            <Card style={{ height: 120 }}>
              <Statistic
                title="设备总数"
                value={data?.devices?.total || 0}
                prefix={<DesktopOutlined />}
                valueStyle={{ color: '#13c2c2' }}
              />
              <div style={{ marginTop: 8, color: '#999', fontSize: 12 }}>
                活跃设备: {data?.devices?.active || 0}
              </div>
            </Card>
          </Col>
        )}
      </Row>

      {/* 第三行：图表 */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={16}>
          <Card title="授权趋势（近30天）">
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={licenseTrend}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="date" />
                <YAxis />
                <Tooltip />
                <Line type="monotone" dataKey="count" stroke="#1890ff" strokeWidth={2} />
              </LineChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="授权类型分布">
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={licenseType}
                  dataKey="count"
                  nameKey="type"
                  cx="50%"
                  cy="50%"
                  outerRadius={80}
                  label={({ payload }: any) => `${payload?.type}: ${payload?.count}`}
                >
                  {licenseType.map((_, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default Dashboard;
