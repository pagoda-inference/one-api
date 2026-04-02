import { BrowserRouter, Routes, Route, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { Layout, Menu, Avatar, Dropdown, Badge, Button, Modal, message } from 'antd'
import {
  DashboardOutlined, ShopOutlined, KeyOutlined, PlusSquareOutlined,
  HistoryOutlined, FileTextOutlined, SettingOutlined, TeamOutlined,
  BellOutlined, GlobalOutlined, LogoutOutlined, UserOutlined,
  MenuFoldOutlined, MenuUnfoldOutlined, ApiOutlined, DatabaseOutlined
} from '@ant-design/icons'
import React, { useState } from 'react'
import Logo from './components/Logo'

import Dashboard from './pages/Dashboard'
import ModelMarket from './pages/ModelMarket'
import ApiKeys from './pages/ApiKeys'
import Topup from './pages/Topup'
import Usage from './pages/Usage'
import Invoices from './pages/Invoices'
import OpsDashboard from './pages/OpsDashboard'
import ModelManagement from './pages/ModelManagement'
import Teams from './pages/Teams'
import Login from './pages/Login'
import LarkOAuth from './pages/LarkOAuth'
import ApiDocs from './pages/ApiDocs'
import { logout, User } from './services/api'

const { Content } = Layout

// Main Layout with Sidebar wrapping children
const AppLayout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const location = useLocation()
  const navigate = useNavigate()
  // 同步从 localStorage 初始化，避免首次渲染时 user 为 null
  const [user, setUser] = useState<User | null>(() => {
    const userInfoStr = localStorage.getItem('user_info')
    if (userInfoStr) {
      try {
        return JSON.parse(userInfoStr)
      } catch (error) {
        localStorage.removeItem('user_info')
      }
    }
    return null
  })
  const [collapsed, setCollapsed] = useState(false)

  const handleLogout = async () => {
    try {
      await logout()
    } catch (e) {}
    localStorage.removeItem('access_token')
    setUser(null)
    message.success('已退出登录')
    navigate('/login')
  }

  const userMenuItems = [
    { key: 'profile', icon: <UserOutlined />, label: '个人中心' },
    { type: 'divider' as const },
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', danger: true }
  ]

  const onUserMenuClick = ({ key }: { key: string }) => {
    if (key === 'logout') {
      Modal.confirm({
        title: '确认退出',
        content: '确定要退出登录吗？',
        onOk: handleLogout
      })
    }
  }

  // Common menu items for all users
  const commonMenuItems = [
    { key: '/dashboard', icon: <DashboardOutlined />, label: '数据看板' },
    { key: '/market', icon: <ShopOutlined />, label: '模型广场' },
    { key: '/docs', icon: <ApiOutlined />, label: 'API 文档' },
    { key: '/keys', icon: <KeyOutlined />, label: 'API Keys' },
    { key: '/usage', icon: <HistoryOutlined />, label: '用量明细' },
    { key: '/topup', icon: <PlusSquareOutlined />, label: '充值中心' },
    { key: '/invoices', icon: <FileTextOutlined />, label: '发票管理' },
  ]

  // Admin menu items
  const adminMenuItems = [
    { key: '/ops', icon: <SettingOutlined />, label: '运营管理' },
    { key: '/ops/models', icon: <DatabaseOutlined />, label: '模型管理' },
    { key: '/teams', icon: <TeamOutlined />, label: '团队管理' },
  ]

  if (!user) {
    return (
      <Layout style={{ minHeight: '100vh', background: '#f5f7fa' }}>
        <Content style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Navigate to="/login" replace />
        </Content>
      </Layout>
    )
  }

  const menuItems = (user.role ?? 0) >= 10
    ? [...commonMenuItems, ...adminMenuItems]
    : commonMenuItems

  // Get greeting based on time
  const getGreeting = () => {
    const hour = new Date().getHours()
    if (hour < 6) return '凌晨好'
    if (hour < 9) return '早上好'
    if (hour < 12) return '上午好'
    if (hour < 14) return '中午好'
    if (hour < 18) return '下午好'
    if (hour < 22) return '晚上好'
    return '夜里好'
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Layout.Sider
        trigger={null}
        collapsible
        collapsed={collapsed}
        width={220}
        style={{
          background: '#fff',
          boxShadow: '2px 0 8px rgba(0,0,0,0.06)',
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
          zIndex: 100,
          overflow: 'auto'
        }}
      >
        <div style={{
          height: 64,
          display: 'flex',
          alignItems: 'center',
          justifyContent: collapsed ? 'center' : 'flex-start',
          padding: collapsed ? 0 : '0 20px',
          borderBottom: '1px solid #f0f0f0'
        }}>
          {collapsed ? (
            <div style={{ width: 32, height: 32, overflow: 'hidden', borderRadius: 6 }}>
              <Logo width={90} height={32} />
            </div>
          ) : (
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 10
            }}>
              <Logo width={100} height={28} />
              <span style={{ fontWeight: 700, fontSize: 16, color: '#333' }}>BEDI 宝塔</span>
            </div>
          )}
        </div>

        <div style={{ padding: '16px 12px' }}>
          <div style={{
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            borderRadius: 10,
            padding: '12px 16px',
            marginBottom: 16
          }}>
            <div style={{ color: 'rgba(255,255,255,0.8)', fontSize: 12, marginBottom: 4 }}>账户余额</div>
            <div style={{ color: '#fff', fontSize: 20, fontWeight: 700 }}>
              ¥{(user.quota / 7200).toFixed(2)}
            </div>
          </div>
        </div>

        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          style={{
            border: 'none',
            padding: '0 8px'
          }}
        />
      </Layout.Sider>

      <Layout style={{ marginLeft: collapsed ? 80 : 220, transition: 'margin-left 0.2s' }}>
        <Layout.Header style={{
          background: '#fff',
          padding: '0 24px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          boxShadow: '0 1px 4px rgba(0,0,0,0.08)',
          position: 'sticky',
          top: 0,
          zIndex: 99,
          height: 64,
          lineHeight: '64px'
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <Button
              type="text"
              icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              onClick={() => setCollapsed(!collapsed)}
              style={{ fontSize: 16 }}
            />
            <span style={{ color: '#333', fontWeight: 500 }}>
              {getGreeting()}，{user.display_name || user.username}
            </span>
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            {/* TODO: 接通通知接口 */}
            <Badge count={0} size="small">
              <Button type="text" icon={<BellOutlined />} style={{ fontSize: 16 }} />
            </Badge>
            {/* TODO: 后续支持多语言 */}
            <Button type="text" icon={<GlobalOutlined />} style={{ fontSize: 16 }} disabled />

            <Dropdown
              menu={{ items: userMenuItems, onClick: onUserMenuClick }}
              placement="bottomRight"
              trigger={['click']}
            >
              <Avatar
                style={{ background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', cursor: 'pointer' }}
              >
                {user.display_name?.[0] || user.username?.[0]?.toUpperCase()}
              </Avatar>
            </Dropdown>
          </div>
        </Layout.Header>

        <Content style={{
          margin: 24,
          minHeight: 'calc(100vh - 64px - 48px)',
          background: '#f5f7fa'
        }}>
          {children}
        </Content>
      </Layout>
    </Layout>
  )
}

// Simple page wrapper with auth check
const ProtectedPage: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const token = localStorage.getItem('access_token')
  if (!token) {
    return <Navigate to="/login" replace />
  }
  return <AppLayout>{children}</AppLayout>
}

const App: React.FC = () => {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/oauth/lark" element={<LarkOAuth />} />
        <Route path="/dashboard" element={<ProtectedPage><Dashboard /></ProtectedPage>} />
        <Route path="/market" element={<ProtectedPage><ModelMarket /></ProtectedPage>} />
        <Route path="/keys" element={<ProtectedPage><ApiKeys /></ProtectedPage>} />
        <Route path="/topup" element={<ProtectedPage><Topup /></ProtectedPage>} />
        <Route path="/usage" element={<ProtectedPage><Usage /></ProtectedPage>} />
        <Route path="/invoices" element={<ProtectedPage><Invoices /></ProtectedPage>} />
        <Route path="/ops" element={<ProtectedPage><OpsDashboard /></ProtectedPage>} />
        <Route path="/ops/models" element={<ProtectedPage><ModelManagement /></ProtectedPage>} />
        <Route path="/teams" element={<ProtectedPage><Teams /></ProtectedPage>} />
        <Route path="/docs" element={<ProtectedPage><ApiDocs /></ProtectedPage>} />
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
