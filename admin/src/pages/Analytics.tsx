import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Select, DatePicker, Statistic, Spin } from 'antd';
import { Line, Pie, Column, Area } from '@ant-design/charts';
import {
  KeyOutlined,
  DesktopOutlined,
  UserOutlined,
  RiseOutlined,
  FallOutlined,
  AppstoreOutlined,
} from '@ant-design/icons';
import { statsApi, appApi } from '../api';
import dayjs from 'dayjs';

const { RangePicker } = DatePicker;

const Analytics: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [pageLoading, setPageLoading] = useState(true);
  const [apps, setApps] = useState<any[]>([]);
  const [selectedApp, setSelectedApp] = useState<string>('');
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs().subtract(30, 'day'),
    dayjs(),
  ]);
  const [dashboardData, setDashboardData] = useState<any>(null);
  const [licenseTrend, setLicenseTrend] = useState<any[]>([]);
  const [deviceTrend, setDeviceTrend] = useState<any[]>([]);
  const [licenseTypeData, setLicenseTypeData] = useState<any[]>([]);
  const [deviceOSData, setDeviceOSData] = useState<any[]>([]);

  useEffect(() => {
    fetchApps();
    fetchDashboard();
  }, []);

  useEffect(() => {
    fetchTrendData();
  }, [selectedApp, dateRange]);

  const fetchApps = async () => {
    try {
      const result: any = await appApi.list();
      const appList = Array.isArray(result) ? result : (result?.list || []);
      setApps(appList);
    } catch (error) {
      console.error(error);
      setApps([]);
    } finally {
      setPageLoading(false);
    }
  };

  const fetchDashboard = async () => {
    try {
      const result: any = await statsApi.dashboard();
      setDashboardData(result);
    } catch (error) {
      console.error(error);
    }
  };

  const fetchTrendData = async () => {
    setLoading(true);
    try {
      const params: any = {
        start_date: dateRange[0].format('YYYY-MM-DD'),
        end_date: dateRange[1].format('YYYY-MM-DD'),
      };
      if (selectedApp) {
        params.app_id = selectedApp;
      }

      const [licenseResult, deviceResult, typeResult, osResult]: any = await Promise.all([
        statsApi.licenseTrend(params).catch(() => null),
        statsApi.deviceTrend(params).catch(() => null),
        statsApi.licenseType(params).catch(() => null),
        statsApi.deviceOS(params).catch(() => null),
      ]);

      const getList = (r: any) => r?.list ?? (Array.isArray(r) ? r : []);
      setLicenseTrend(getList(licenseResult));
      setDeviceTrend(getList(deviceResult));
      setLicenseTypeData(getList(typeResult));
      setDeviceOSData(getList(osResult));
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  const licenseTrendConfig = {
    data: licenseTrend,
    xField: 'date',
    yField: 'count',
    seriesField: 'type',
    smooth: true,
    animation: { appear: { animation: 'path-in', duration: 1000 } },
  };

  const deviceTrendConfig = {
    data: deviceTrend,
    xField: 'date',
    yField: 'count',
    smooth: true,
    areaStyle: { fill: 'l(270) 0:#ffffff 0.5:#7ec2f3 1:#1890ff' },
  };

  const licenseTypeConfig = {
    data: licenseTypeData,
    angleField: 'count',
    colorField: 'type',
    radius: 0.8,
    label: {
      type: 'outer',
      content: '{name} {percentage}',
    },
    interactions: [{ type: 'element-active' }],
  };

  const deviceOSConfig = {
    data: deviceOSData,
    xField: 'os',
    yField: 'count',
    label: { position: 'middle' as const },
    color: '#1890ff',
  };

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
        <h2 style={{ margin: 0 }}>数据分析</h2>
        <div style={{ display: 'flex', gap: 16 }}>
          <Select
            style={{ width: 200 }}
            placeholder="全部应用"
            allowClear
            value={selectedApp || undefined}
            onChange={setSelectedApp}
            options={apps.map(app => ({ label: app.name, value: app.id }))}
          />
          <RangePicker
            value={dateRange}
            onChange={(dates) => dates && setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs])}
            presets={[
              { label: '最近7天', value: [dayjs().subtract(7, 'day'), dayjs()] },
              { label: '最近30天', value: [dayjs().subtract(30, 'day'), dayjs()] },
              { label: '最近90天', value: [dayjs().subtract(90, 'day'), dayjs()] },
            ]}
          />
        </div>
      </div>

      {/* 概览统计 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总授权数"
              value={dashboardData?.total_licenses || 0}
              prefix={<KeyOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
            <div style={{ marginTop: 8, fontSize: 12, color: '#666' }}>
              活跃: {dashboardData?.active_licenses || 0}
            </div>
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="总设备数"
              value={dashboardData?.total_devices || 0}
              prefix={<DesktopOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
            <div style={{ marginTop: 8, fontSize: 12, color: '#666' }}>
              在线: {dashboardData?.online_devices || 0}
            </div>
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="总用户数"
              value={dashboardData?.total_users || 0}
              prefix={<UserOutlined />}
              valueStyle={{ color: '#722ed1' }}
            />
            <div style={{ marginTop: 8, fontSize: 12, color: '#666' }}>
              本月新增: {dashboardData?.new_users_this_month || 0}
            </div>
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="应用数量"
              value={dashboardData?.total_apps || apps.length}
              prefix={<AppstoreOutlined />}
              valueStyle={{ color: '#fa8c16' }}
            />
            <div style={{ marginTop: 8, fontSize: 12, color: '#666' }}>
              活跃: {dashboardData?.active_apps || apps.filter(a => a.status === 'active').length}
            </div>
          </Card>
        </Col>
      </Row>

      {/* 增长指标 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="授权增长率"
              value={dashboardData?.license_growth_rate || 0}
              precision={1}
              valueStyle={{ color: (dashboardData?.license_growth_rate || 0) >= 0 ? '#3f8600' : '#cf1322' }}
              prefix={(dashboardData?.license_growth_rate || 0) >= 0 ? <RiseOutlined /> : <FallOutlined />}
              suffix="%"
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="设备增长率"
              value={dashboardData?.device_growth_rate || 0}
              precision={1}
              valueStyle={{ color: (dashboardData?.device_growth_rate || 0) >= 0 ? '#3f8600' : '#cf1322' }}
              prefix={(dashboardData?.device_growth_rate || 0) >= 0 ? <RiseOutlined /> : <FallOutlined />}
              suffix="%"
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="用户增长率"
              value={dashboardData?.user_growth_rate || 0}
              precision={1}
              valueStyle={{ color: (dashboardData?.user_growth_rate || 0) >= 0 ? '#3f8600' : '#cf1322' }}
              prefix={(dashboardData?.user_growth_rate || 0) >= 0 ? <RiseOutlined /> : <FallOutlined />}
              suffix="%"
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="续费率"
              value={dashboardData?.renewal_rate || 0}
              precision={1}
              valueStyle={{ color: '#1890ff' }}
              suffix="%"
            />
          </Card>
        </Col>
      </Row>

      <Spin spinning={loading}>
        {/* 趋势图表 */}
        <Row gutter={16} style={{ marginBottom: 24 }}>
          <Col span={12}>
            <Card title="授权趋势" size="small">
              {licenseTrend.length > 0 ? (
                <Line {...licenseTrendConfig} height={300} />
              ) : (
                <div style={{ height: 300, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#999' }}>暂无数据</div>
              )}
            </Card>
          </Col>
          <Col span={12}>
            <Card title="设备趋势" size="small">
              {deviceTrend.length > 0 ? (
                <Area {...deviceTrendConfig} height={300} />
              ) : (
                <div style={{ height: 300, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#999' }}>暂无数据</div>
              )}
            </Card>
          </Col>
        </Row>

        {/* 分布图表 */}
        <Row gutter={16}>
          <Col span={12}>
            <Card title="授权类型分布" size="small">
              {licenseTypeData.length > 0 ? (
                <Pie {...licenseTypeConfig} height={300} />
              ) : (
                <div style={{ height: 300, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#999' }}>暂无数据</div>
              )}
            </Card>
          </Col>
          <Col span={12}>
            <Card title="设备操作系统分布" size="small">
              {deviceOSData.length > 0 ? (
                <Column {...deviceOSConfig} height={300} />
              ) : (
                <div style={{ height: 300, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#999' }}>暂无数据</div>
              )}
            </Card>
          </Col>
        </Row>
      </Spin>
    </div>
  );
};

export default Analytics;
