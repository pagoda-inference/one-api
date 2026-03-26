import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Layout, Menu } from 'antd'
import { DashboardOutlined, ShopOutlined, KeyOutlined, RechargeOutlined, HistoryOutlined, FileTextOutlined } from '@ant-design/icons'
import React from 'react'

import Dashboard from './pages/Dashboard'
import ModelMarket from './pages/ModelMarket'
import ApiKeys from './pages/ApiKeys'
import Topup from './pages/Topup'
import Usage from './pages/Usage'
import Invoices from './pages/Invoices'

const { Header, Content } = Layout

const App: React.FC = () => {
  const menuItems = [
    { key: '/dashboard', icon: <DashboardOutlined />, label: '控制台' },
    { key: '/market', icon: <ShopOutlined />, label: '模型市场' },
    { key: '/topup', icon: <RechargeOutlined />, label: '充值' },
    { key: '/keys', icon: <KeyOutlined />, label: 'API Keys' },
    { key: '/usage', icon: <HistoryOutlined />, label: '用量明细' },
    { key: '/invoices', icon: <FileTextOutlined />, label: '发票管理' },
  ]

  return (
    <BrowserRouter>
      <Layout style={{ minHeight: '100vh' }}>
        <Header style={{ display: 'flex', alignItems: 'center' }}>
          <div style={{ color: 'white', fontSize: '20px', fontWeight: 'bold', marginRight: '48px' }}>
            One API
          </div>
          <Menu
            theme="dark"
            mode="horizontal"
            defaultSelectedKeys={['/dashboard']}
            items={menuItems}
            style={{ flex: 1, minWidth: 0 }}
          />
        </Header>
        <Content style={{ padding: '24px' }}>
          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/market" element={<ModelMarket />} />
            <Route path="/topup" element={<Topup />} />
            <Route path="/keys" element={<ApiKeys />} />
            <Route path="/usage" element={<Usage />} />
            <Route path="/invoices" element={<Invoices />} />
          </Routes>
        </Content>
      </Layout>
    </BrowserRouter>
  )
}

export default App