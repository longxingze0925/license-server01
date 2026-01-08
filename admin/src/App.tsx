import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, App as AntdApp } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import MainLayout from './layouts/MainLayout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Apps from './pages/Apps';
import AppDetail from './pages/AppDetail';
import Licenses from './pages/Licenses';
import Subscriptions from './pages/Subscriptions';
import Devices from './pages/Devices';
import TeamMembers from './pages/TeamMembers';
import Customers from './pages/Customers';
import AuditLogs from './pages/AuditLogs';
import Profile from './pages/Profile';
import Blacklist from './pages/Blacklist';
import DataExport from './pages/DataExport';
import DataBackups from './pages/DataBackups';
import { useAuthStore } from './store';

// 路由守卫组件
const PrivateRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuthStore();
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />;
};

const App: React.FC = () => {
  return (
    <ConfigProvider locale={zhCN}>
      <AntdApp>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route
              path="/"
              element={
                <PrivateRoute>
                  <MainLayout />
                </PrivateRoute>
              }
            >
              <Route index element={<Dashboard />} />
              <Route path="apps" element={<Apps />} />
              <Route path="apps/:id" element={<AppDetail />} />
              <Route path="team" element={<TeamMembers />} />
              <Route path="customers" element={<Customers />} />
              <Route path="licenses" element={<Licenses />} />
              <Route path="subscriptions" element={<Subscriptions />} />
              <Route path="devices" element={<Devices />} />
              <Route path="blacklist" element={<Blacklist />} />
              <Route path="audit" element={<AuditLogs />} />
              <Route path="export" element={<DataExport />} />
              <Route path="backups" element={<DataBackups />} />
              <Route path="profile" element={<Profile />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </AntdApp>
    </ConfigProvider>
  );
};

export default App;
