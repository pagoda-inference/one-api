import { BrowserRouter, Routes, Route, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { Layout, Menu, Avatar, Dropdown, Badge, Button, Modal, message, Select, List, Popover } from 'antd'
import {
  DashboardOutlined, ShopOutlined, KeyOutlined, PlusSquareOutlined,
  HistoryOutlined, FileTextOutlined, SettingOutlined, TeamOutlined,
  BellOutlined, LogoutOutlined, UserOutlined,
  MenuFoldOutlined, MenuUnfoldOutlined, ApiOutlined, DatabaseOutlined,
  CloudServerOutlined, MoonOutlined, SunOutlined
} from '@ant-design/icons'
import React, { useState, useEffect } from 'react'
import Logo from './components/Logo'
import { useTheme } from './contexts/ThemeContext'
import { useLanguage } from './contexts/LanguageContext'

import Dashboard from './pages/Dashboard'
import ModelMarket from './pages/ModelMarket'
import ApiKeys from './pages/ApiKeys'
import Topup from './pages/Topup'
import Usage from './pages/Usage'
import Invoices from './pages/Invoices'
import OpsDashboard from './pages/OpsDashboard'
import ModelManagement from './pages/ModelManagement'
import ProviderManagement from './pages/ProviderManagement'
import Teams from './pages/Teams'
import Login from './pages/Login'
import LarkOAuth from './pages/LarkOAuth'
import ApiDocs from './pages/ApiDocs'
import Profile from './pages/Profile'
import NotificationManage from './pages/NotificationManage'
import { logout, User, getUnreadNotificationCount, getNotifications, markNotificationAsRead } from './services/api'

const { Content } = Layout

// Main Layout with Sidebar wrapping children
const AppLayout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const location = useLocation()
  const navigate = useNavigate()
  const { themeMode, toggleTheme } = useTheme()
  const { language, setLanguage } = useLanguage()
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
  const [unreadCount, setUnreadCount] = useState(0)
  const [notifications, setNotifications] = useState<any[]>([])
  const [notificationLoading, setNotificationLoading] = useState(false)

  // Fetch unread notification count
  useEffect(() => {
    if (user) {
      fetchUnreadCount()
      // Poll every 60 seconds
      const interval = setInterval(fetchUnreadCount, 60000)
      return () => clearInterval(interval)
    }
  }, [user])

  const fetchUnreadCount = async () => {
    try {
      const res = await getUnreadNotificationCount()
      if (res.data?.success) {
        setUnreadCount(res.data.data || 0)
      }
    } catch (error) {
      console.error('Failed to fetch unread count:', error)
    }
  }

  const fetchNotifications = async () => {
    setNotificationLoading(true)
    try {
      const res = await getNotifications(10, 0)
      if (res.data?.success) {
        setNotifications(res.data.data || [])
      }
    } catch (error) {
      console.error('Failed to fetch notifications:', error)
    } finally {
      setNotificationLoading(false)
    }
  }

  const handleNotificationClick = async (notification: any) => {
    if (!notification.is_read) {
      try {
        await markNotificationAsRead(notification.id)
        fetchUnreadCount()
        fetchNotifications()
      } catch (error) {
        console.error('Failed to mark as read:', error)
      }
    }
  }

  const notificationContent = (
    <div style={{ width: 320, maxHeight: 400, overflow: 'auto' }}>
      {notificationLoading ? (
        <div style={{ padding: 20, textAlign: 'center' }}>加载中...</div>
      ) : notifications.length === 0 ? (
        <div style={{ padding: 20, textAlign: 'center', color: '#999' }}>暂无通知</div>
      ) : (
        <List
          dataSource={notifications}
          renderItem={(item: any) => (
            <List.Item
              style={{ cursor: 'pointer', background: item.is_read ? 'transparent' : '#f0f0f0' }}
              onClick={() => handleNotificationClick(item)}
            >
              <List.Item.Meta
                title={<span>{item.is_read ? '' : <span style={{ color: '#1890ff' }}>•</span>} {item.title}</span>}
                description={<span style={{ fontSize: 12 }}>{item.content}</span>}
              />
            </List.Item>
          )}
        />
      )}
    </div>
  )

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
    } else if (key === 'profile') {
      navigate('/profile')
    }
  }

  // Common menu items for all users
  const commonMenuItems = [
    { key: '/dashboard', icon: <DashboardOutlined />, label: '数据看板' },
    { key: '/market', icon: <ShopOutlined />, label: '模型广场' },
    { key: '/keys', icon: <KeyOutlined />, label: 'API Keys' },
    { key: '/docs', icon: <ApiOutlined />, label: 'API 文档' },
    { key: '/usage', icon: <HistoryOutlined />, label: '用量明细' },
    { key: '/topup', icon: <PlusSquareOutlined />, label: '充值中心' },
    { key: '/invoices', icon: <FileTextOutlined />, label: '发票管理' },
  ]

  // Admin menu items
  const adminMenuItems = [
    { key: '/ops', icon: <SettingOutlined />, label: '运营管理' },
    { key: '/ops/models', icon: <DatabaseOutlined />, label: '模型管理' },
    { key: '/ops/providers', icon: <CloudServerOutlined />, label: 'Provider 管理' },
    { key: '/ops/notifications', icon: <BellOutlined />, label: '系统通知' },
    { key: '/teams', icon: <TeamOutlined />, label: '团队管理' },
  ]

  if (!user) {
    return (
      <Layout style={{ minHeight: '100vh', background: 'var(--bg-secondary)' }}>
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
          background: 'var(--bg-primary)',
          boxShadow: '2px 0 8px var(--shadow)',
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
          borderBottom: '1px solid var(--border-color)'
        }}>
          {collapsed ? (
            <span style={{ fontWeight: 700, fontSize: 16, color: 'var(--text-primary)' }}>宝塔</span>
          ) : (
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8
            }}>
              <Logo width={90} height={26} />
              <span style={{ fontWeight: 700, fontSize: 15, color: 'var(--text-primary)', whiteSpace: 'nowrap' }}>BEDI 宝塔</span>
            </div>
          )}
        </div>

        {!collapsed && (
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
        )}

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

      <Layout style={{ marginLeft: collapsed ? 80 : 220, transition: 'margin-left 0.2s', background: 'var(--bg-secondary)' }}>
        <Layout.Header style={{
          background: 'var(--bg-primary)',
          padding: '0 24px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          boxShadow: '0 1px 4px var(--shadow)',
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
              style={{ fontSize: 16, color: 'var(--text-primary)' }}
            />
            <span style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
              {getGreeting()}，{user.display_name || user.username}
            </span>
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <Popover
              content={notificationContent}
              title="通知"
              trigger="click"
              placement="bottomRight"
              arrow={false}
              onOpenChange={(open) => {
                if (open) {
                  fetchNotifications()
                }
              }}
            >
              <Badge count={unreadCount} size="small" offset={[-2, 2]}>
                <Button type="text" icon={<BellOutlined />} style={{ fontSize: 16, color: 'var(--text-primary)' }} />
              </Badge>
            </Popover>
            <Select
              value={language}
              onChange={(value) => setLanguage(value)}
              style={{ width: 100 }}
              options={[
                { value: 'zh', label: '中文' },
                { value: 'en', label: 'English' },
              ]}
            />
            <Button
              type="text"
              icon={themeMode === 'dark' ? <SunOutlined /> : <MoonOutlined />}
              onClick={toggleTheme}
              style={{ fontSize: 16, color: 'var(--text-primary)' }}
            />

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
          background: 'var(--bg-secondary)'
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
        <Route path="/ops/providers" element={<ProtectedPage><ProviderManagement /></ProtectedPage>} />
        <Route path="/teams" element={<ProtectedPage><Teams /></ProtectedPage>} />
        <Route path="/docs" element={<ProtectedPage><ApiDocs /></ProtectedPage>} />
        <Route path="/profile" element={<ProtectedPage><Profile /></ProtectedPage>} />
        <Route path="/ops/notifications" element={<ProtectedPage><NotificationManage /></ProtectedPage>} />
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
