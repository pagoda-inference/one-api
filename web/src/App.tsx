import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Layout, Menu } from 'antd'
import { DashboardOutlined, ShopOutlined, KeyOutlined, PlusSquareOutlined, HistoryOutlined, FileTextOutlined, SettingOutlined, TeamOutlined } from '@ant-design/icons'
import React, { useState, useEffect } from 'react'

import Dashboard from './pages/Dashboard'
import ModelMarket from './pages/ModelMarket'
import ApiKeys from './pages/ApiKeys'
import Topup from './pages/Topup'
import Usage from './pages/Usage'
import Invoices from './pages/Invoices'
import OpsDashboard from './pages/OpsDashboard'
import Teams from './pages/Teams'
import { getUserInfo, User } from './services/api'

const { Header, Content } = Layout

const App: React.FC = () => {
  const [user, setUser] = useState<User | null>(null)
  const [menuItems, setMenuItems] = useState([
    { key: '/dashboard', icon: <DashboardOutlined />, label: '控制台' },
    { key: '/market', icon: <ShopOutlined />, label: '模型市场' },
    { key: '/topup', icon: <PlusSquareOutlined />, label: '充值' },
    { key: '/keys', icon: <KeyOutlined />, label: 'API Keys' },
    { key: '/usage', icon: <HistoryOutlined />, label: '用量明细' },
    { key: '/invoices', icon: <FileTextOutlined />, label: '发票管理' },
  ])

  useEffect(() => {
    const fetchUser = async () => {
      try {
        const res = await getUserInfo()
        setUser(res.data.data)
      } catch (error) {
        console.error('Failed to fetch user info:', error)
      }
    }
    fetchUser()
  }, [])

  useEffect(() => {
    if (user && user.role >= 10) {
      setMenuItems([
        { key: '/dashboard', icon: <DashboardOutlined />, label: '控制台' },
        { key: '/market', icon: <ShopOutlined />, label: '模型市场' },
        { key: '/topup', icon: <PlusSquareOutlined />, label: '充值' },
        { key: '/keys', icon: <KeyOutlined />, label: 'API Keys' },
        { key: '/usage', icon: <HistoryOutlined />, label: '用量明细' },
        { key: '/invoices', icon: <FileTextOutlined />, label: '发票管理' },
        { key: '/ops', icon: <SettingOutlined />, label: '运营管理' },
        { key: '/teams', icon: <TeamOutlined />, label: '团队管理' },
      ])
    }
  }, [user])

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
            <Route path="/ops" element={<OpsDashboard />} />
            <Route path="/teams" element={<Teams />} />
          </Routes>
        </Content>
      </Layout>
    </BrowserRouter>
  )
}

export default App