import { BrowserRouter, Routes, Route, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { Layout, Menu, Avatar, Dropdown, Badge, Button, Modal, message, Select, List, Popover } from 'antd'
import {
  DashboardOutlined, ShopOutlined, KeyOutlined, PlusSquareOutlined,
  HistoryOutlined, FileTextOutlined, SettingOutlined, TeamOutlined,
  BellOutlined, LogoutOutlined, UserOutlined,
  MenuFoldOutlined, MenuUnfoldOutlined, ApiOutlined, DatabaseOutlined,
  CloudServerOutlined, MoonOutlined, SunOutlined, ExperimentOutlined,
  ThunderboltOutlined
} from '@ant-design/icons'
import React, { useState, useEffect } from 'react'
import ReactMarkdown from 'react-markdown'
import Logo from './components/Logo'
import { useTheme } from './contexts/ThemeContext'
import { useLanguage } from './contexts/LanguageContext'
import { useTranslation } from 'react-i18next'

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
import BatchInference from './pages/BatchInference'
import ChatPlayground from './pages/ChatPlayground'
import { logout, User, getUnreadNotificationCount, getNotifications, markNotificationAsRead } from './services/api'

const { Content } = Layout

// Main Layout with Sidebar wrapping children
const AppLayout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const location = useLocation()
  const navigate = useNavigate()
  const { themeMode, toggleTheme } = useTheme()
  const { language, setLanguage } = useLanguage()
  const { t } = useTranslation()
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
        <div style={{ padding: 20, textAlign: 'center' }}>{t('common.loading')}</div>
      ) : notifications.length === 0 ? (
        <div style={{ padding: 20, textAlign: 'center', color: '#999' }}>{t('common.no_notifications')}</div>
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
                description={<span style={{ fontSize: 12 }}><ReactMarkdown>{item.content}</ReactMarkdown></span>}
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
    message.success(t('common.logout_success'))
    navigate('/login')
  }

  const userMenuItems = [
    { key: 'profile', icon: <UserOutlined />, label: t('menu.personal_settings') },
    { type: 'divider' as const },
    { key: 'logout', icon: <LogoutOutlined />, label: t('common.logout'), danger: true }
  ]

  const onUserMenuClick = ({ key }: { key: string }) => {
    if (key === 'logout') {
      Modal.confirm({
        title: t('common.confirm_logout'),
        content: t('common.logout_confirm_message'),
        onOk: handleLogout
      })
    } else if (key === 'profile') {
      navigate('/profile')
    }
  }

  // User null check must come first
  if (!user) {
    return (
      <Layout style={{ minHeight: '100vh', background: 'var(--bg-secondary)' }}>
        <Content style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Navigate to="/login" replace />
        </Content>
      </Layout>
    )
  }

  const isAdmin = (user.role ?? 0) >= 10

  // Experience Center
  const experienceCenterGroup = {
    type: 'group' as const,
    label: t('menu.experience_center'),
    key: 'experience',
    children: [
      { key: '/experience', icon: <ExperimentOutlined />, label: t('menu.text_chat') }
    ]
  }

  // Model group - batch only for non-admin users
  const modelGroupChildren = [
    { key: '/market', icon: <ShopOutlined />, label: t('menu.model_market') },
    ...(isAdmin ? [] : [{ key: '/batch', icon: <ThunderboltOutlined />, label: t('menu.batch_inference') }])
  ]
  const modelGroup = {
    type: 'group' as const,
    label: t('menu.model'),
    key: 'model',
    children: modelGroupChildren
  }

  // Console group
  const consoleGroup = {
    type: 'group' as const,
    label: t('menu.console'),
    key: 'console',
    children: [
      { key: '/dashboard', icon: <DashboardOutlined />, label: t('menu.dashboard') },
      { key: '/keys', icon: <KeyOutlined />, label: t('menu.tokens') },
      { key: '/docs', icon: <ApiOutlined />, label: t('menu.quick_start') },
      { key: '/usage', icon: <HistoryOutlined />, label: t('menu.usage_detail') },
    ]
  }

  // Personal Center group
  const personalCenterGroup = {
    type: 'group' as const,
    label: t('menu.personal_center'),
    key: 'personal',
    children: [
      { key: '/profile', icon: <UserOutlined />, label: t('menu.personal_settings') },
      { key: '/topup', icon: <PlusSquareOutlined />, label: t('menu.topup') },
      { key: '/invoices', icon: <FileTextOutlined />, label: t('menu.invoice_management') },
    ]
  }

  // Admin group
  const adminGroup = {
    type: 'group' as const,
    label: t('menu.ops_dashboard'),
    key: 'admin',
    children: [
      { key: '/ops', icon: <SettingOutlined />, label: t('menu.ops_dashboard') },
      { key: '/ops/models', icon: <DatabaseOutlined />, label: t('menu.model_management_admin') },
      { key: '/ops/providers', icon: <CloudServerOutlined />, label: t('menu.provider_management') },
      { key: '/ops/notifications', icon: <BellOutlined />, label: t('menu.system_notifications') },
      { key: '/teams', icon: <TeamOutlined />, label: t('menu.teams') },
    ]
  }

  const menuItems = isAdmin
    ? [experienceCenterGroup, modelGroup, consoleGroup, personalCenterGroup, adminGroup]
    : [experienceCenterGroup, modelGroup, consoleGroup, personalCenterGroup]

  // Get greeting based on time
  const getGreeting = () => {
    const hour = new Date().getHours()
    if (hour < 6) return t('greeting.dawn')
    if (hour < 9) return t('greeting.morning')
    if (hour < 12) return t('greeting.forenoon')
    if (hour < 14) return t('greeting.noon')
    if (hour < 18) return t('greeting.afternoon')
    if (hour < 22) return t('greeting.evening')
    return t('greeting.night')
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
            <span style={{ fontWeight: 700, fontSize: 16, color: 'var(--text-primary)' }}>{t('common.brand')}</span>
          ) : (
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8
            }}>
              <Logo width={90} height={26} />
              <span style={{ fontWeight: 700, fontSize: 15, color: 'var(--text-primary)', whiteSpace: 'nowrap' }}>{t('common.brand_full')}</span>
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
              <div style={{ color: 'rgba(255,255,255,0.8)', fontSize: 12, marginBottom: 4 }}>{t('common.account_balance')}</div>
              <div style={{ color: '#fff', fontSize: 20, fontWeight: 700 }}>
                ¥{(user.quota / 7200).toFixed(2)}
              </div>
            </div>
          </div>
        )}

        <Menu
          mode="inline"
          inlineCollapsed={collapsed}
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
              title={t('common.notifications')}
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
                { value: 'zh', label: t('common.chinese') },
                { value: 'en', label: t('common.english') },
              ]}
            />
            <Button
              type="text"
              icon={themeMode === 'dark' ? <SunOutlined /> : <MoonOutlined />}
              onClick={toggleTheme}
              style={{ fontSize: 16, color: 'var(--text-primary)' }}
            />
            <Button
              type="link"
              onClick={() => window.open('https://baotaai.bedicloud.net/guide', '_blank')}
              style={{ color: 'var(--text-primary)', fontSize: 14 }}
            >
              {t('common.docs')}
            </Button>

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
        <Route path="/batch" element={<ProtectedPage><BatchInference /></ProtectedPage>} />
        <Route path="/experience" element={<ProtectedPage><ChatPlayground /></ProtectedPage>} />
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
