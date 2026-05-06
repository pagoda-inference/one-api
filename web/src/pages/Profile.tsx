import { useState, useEffect } from 'react'
import { useTheme } from '../contexts/ThemeContext'
import { Card, Form, Input, Button, message, Spin, Avatar, Switch, Select } from 'antd'
import {
  UserOutlined,
  MailOutlined,
  LockOutlined,
  SunOutlined,
  MoonOutlined,
  GlobalOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons'
import { getUserInfo, updateUserInfo } from '../services/api'
import { useTranslation } from 'react-i18next'

interface UserInfo {
  id: number
  username: string
  display_name: string
  email: string
  avatar_url: string
  role: number
  status: number
}

const Profile: React.FC = () => {
  const { t, i18n } = useTranslation()
  const { appTheme, toggleTheme, themeMode } = useTheme()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null)
  const [form] = Form.useForm()
  const [themeLoading, setThemeLoading] = useState(false)

  useEffect(() => {
    loadUserInfo()
  }, [])

  const loadUserInfo = async () => {
    try {
      setLoading(true)
      const res = await getUserInfo()
      if (res.data?.success) {
        const data = res.data.data
        setUserInfo(data)
        form.setFieldsValue({
          display_name: data.display_name,
          email: data.email,
        })
      }
    } catch (error) {
      console.error('Failed to load user info:', error)
      message.error(t('profile.load_user_info_failed'))
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (values: {
    display_name?: string
    email?: string
    password?: string
  }) => {
    try {
      setSaving(true)
      const res = await updateUserInfo(values)
      if (res.data?.success) {
        message.success(t('profile.save_success'))
        loadUserInfo()
      } else {
        message.error(res.data?.message || t('profile.save_failed'))
      }
    } catch (error) {
      console.error('Failed to update:', error)
      message.error(t('profile.save_failed'))
    } finally {
      setSaving(false)
    }
  }

  const handleThemeToggle = async (_checked: boolean) => {
    setThemeLoading(true)
    try {
      await toggleTheme()
    } finally {
      setThemeLoading(false)
    }
  }

  const handleLanguageChange = (lang: string) => {
    i18n.changeLanguage(lang)
    localStorage.setItem('i18nextLng', lang)
  }

  const getInitials = (name: string) => {
    if (!name) return 'U'
    return name
      .split(' ')
      .map((n) => n[0])
      .join('')
      .toUpperCase()
      .slice(0, 2)
  }

  const getRoleLabel = (role: number) => {
    if (role === 100) return t('profile.super_admin')
    if (role === 10) return t('profile.admin')
    return t('profile.normal_user')
  }

  const getStatusLabel = (status: number) => {
    return status === 1 ? t('profile.normal') : t('profile.disabled')
  }

  if (loading) {
    return (
      <div
        style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          minHeight: '400px',
        }}
      >
        <Spin size="large" />
      </div>
    )
  }

  return (
    <div style={{ maxWidth: 1200, margin: '0 auto', padding: '24px' }}>
      {/* Profile Header Card */}
      <Card
        style={{
          marginBottom: 16,
          background: themeMode === 'dark'
            ? 'rgba(255,255,255,0.03)'
            : 'rgba(255,255,255,0.80)',
          border: `1px solid ${
            themeMode === 'dark'
              ? 'rgba(255,255,255,0.08)'
              : 'rgba(0,0,0,0.08)'
          }`,
          borderRadius: 12,
        }}
        styles={{ body: { padding: 24 } }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 20,
          }}
        >
          {/* Avatar */}
          <div style={{ position: 'relative' }}>
            {userInfo?.avatar_url ? (
              <Avatar
                src={userInfo.avatar_url}
                size={72}
                style={{ border: '2px solid rgba(255,255,255,0.1)' }}
              />
            ) : (
              <Avatar
                size={72}
                style={{
                  background: 'linear-gradient(135deg, #5e6ad2 0%, #7170ff 100%)',
                  fontSize: 24,
                  fontWeight: 590,
                }}
              >
                {getInitials(userInfo?.display_name || userInfo?.username || 'U')}
              </Avatar>
            )}
            <div
              style={{
                position: 'absolute',
                bottom: -2,
                right: -2,
                width: 20,
                height: 20,
                borderRadius: '50%',
                background: '#10b981',
                border: '2px solid var(--bg-card)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <CheckCircleOutlined
                style={{ color: '#fff', fontSize: 12 }}
              />
            </div>
          </div>

          {/* User Info */}
          <div style={{ flex: 1 }}>
            <h2
              style={{
                fontSize: 20,
                fontWeight: 590,
                color: appTheme.textPrimary,
                marginBottom: 4,
                letterSpacing: '-0.24px',
              }}
            >
              {userInfo?.display_name || userInfo?.username}
            </h2>
            <p
              style={{
                fontSize: 14,
                color: appTheme.textSecondary,
                marginBottom: 8,
              }}
            >
              {userInfo?.email || t('profile.email_placeholder')}
            </p>
            <div style={{ display: 'flex', gap: 12 }}>
              <span
                style={{
                  fontSize: 12,
                  fontWeight: 510,
                  color: '#7170ff',
                  background:
                    themeMode === 'dark'
                      ? 'rgba(113, 112, 255, 0.15)'
                      : 'rgba(94, 106, 210, 0.10)',
                  padding: '2px 8px',
                  borderRadius: 4,
                }}
              >
                {getRoleLabel(userInfo?.role || 1)}
              </span>
              <span
                style={{
                  fontSize: 12,
                  fontWeight: 510,
                  color:
                    userInfo?.status === 1 ? '#10b981' : appTheme.textTertiary,
                  background: themeMode === 'dark'
                    ? 'rgba(16, 185, 129, 0.10)'
                    : 'rgba(16, 185, 129, 0.08)',
                  padding: '2px 8px',
                  borderRadius: 4,
                }}
              >
                {getStatusLabel(userInfo?.status || 1)}
              </span>
            </div>
          </div>
        </div>
      </Card>

      {/* Settings Cards */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))',
          gap: 16,
          alignItems: 'start',
        }}
      >
        <div style={{ minWidth: 0 }}>
        {/* Basic Info Card */}
        <Card
          title={
            <span
              style={{
                fontSize: 14,
                fontWeight: 590,
                color: appTheme.textPrimary,
                letterSpacing: '-0.12px',
              }}
            >
              {t('profile.basic_info')}
            </span>
          }
          style={{
            background: themeMode === 'dark'
              ? 'rgba(255,255,255,0.02)'
              : 'rgba(255,255,255,0.80)',
            border: `1px solid ${
              themeMode === 'dark'
                ? 'rgba(255,255,255,0.08)'
                : 'rgba(0,0,0,0.08)'
            }`,
            borderRadius: 12,
          }}
          styles={{ body: { padding: '16px 24px 24px' } }}
        >
          <Form form={form} layout="vertical" onFinish={handleSubmit}>
            <Form.Item
              label={
                <span style={{ fontSize: 13, fontWeight: 510, color: appTheme.textSecondary }}>
                  {t('profile.display_name')}
                </span>
              }
              name="display_name"
            >
              <Input
                prefix={<UserOutlined style={{ color: appTheme.textTertiary }} />}
                placeholder={t('profile.display_name_placeholder')}
                style={{
                  background: themeMode === 'dark'
                    ? 'rgba(255,255,255,0.02)'
                    : '#f9fafb',
                  border: `1px solid ${
                    themeMode === 'dark'
                      ? 'rgba(255,255,255,0.08)'
                      : 'rgba(0,0,0,0.10)'
                  }`,
                  borderRadius: 6,
                  height: 40,
                }}
              />
            </Form.Item>

            <Form.Item
              label={
                <span style={{ fontSize: 13, fontWeight: 510, color: appTheme.textSecondary }}>
                  {t('profile.email')}
                </span>
              }
              name="email"
              rules={[{ type: 'email', message: t('profile.valid_email_required') }]}
            >
              <Input
                prefix={<MailOutlined style={{ color: appTheme.textTertiary }} />}
                placeholder={t('profile.email_placeholder')}
                style={{
                  background: themeMode === 'dark'
                    ? 'rgba(255,255,255,0.02)'
                    : '#f9fafb',
                  border: `1px solid ${
                    themeMode === 'dark'
                      ? 'rgba(255,255,255,0.08)'
                      : 'rgba(0,0,0,0.10)'
                  }`,
                  borderRadius: 6,
                  height: 40,
                }}
              />
            </Form.Item>

            <Form.Item
              label={
                <span style={{ fontSize: 13, fontWeight: 510, color: appTheme.textSecondary }}>
                  {t('profile.new_password')}
                </span>
              }
              name="password"
              style={{ marginBottom: 16 }}
            >
              <Input.Password
                prefix={<LockOutlined style={{ color: appTheme.textTertiary }} />}
                placeholder={t('profile.leave_blank_no_change')}
                style={{
                  background: themeMode === 'dark'
                    ? 'rgba(255,255,255,0.02)'
                    : '#f9fafb',
                  border: `1px solid ${
                    themeMode === 'dark'
                      ? 'rgba(255,255,255,0.08)'
                      : 'rgba(0,0,0,0.10)'
                  }`,
                  borderRadius: 6,
                  height: 40,
                }}
              />
            </Form.Item>

            <Button
              type="primary"
              htmlType="submit"
              loading={saving}
              style={{
                height: 36,
                borderRadius: 6,
                fontWeight: 510,
                fontSize: 14,
                background: '#5e6ad2',
                borderColor: '#5e6ad2',
              }}
            >
              {t('profile.save_changes')}
            </Button>
          </Form>
        </Card>
        </div>

        <div style={{ minWidth: 0, display: 'grid', gap: 16 }}>
          {/* Preferences Card */}
          <Card
            title={
              <span
                style={{
                  fontSize: 14,
                  fontWeight: 590,
                  color: appTheme.textPrimary,
                  letterSpacing: '-0.12px',
                }}
              >
                {t('profile.preferences')}
              </span>
            }
            style={{
              background: themeMode === 'dark'
                ? 'rgba(255,255,255,0.02)'
                : 'rgba(255,255,255,0.80)',
              border: `1px solid ${
                themeMode === 'dark'
                  ? 'rgba(255,255,255,0.08)'
                  : 'rgba(0,0,0,0.08)'
              }`,
              borderRadius: 12,
            }}
            styles={{ body: { padding: '16px 24px' } }}
          >
            <div style={{ display: 'grid', gap: 0 }}>
            {/* Theme Setting */}
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '12px 0',
                borderBottom: `1px solid ${
                  themeMode === 'dark'
                    ? 'rgba(255,255,255,0.05)'
                    : 'rgba(0,0,0,0.06)'
                }`,
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                {themeMode === 'dark' ? (
                  <MoonOutlined
                    style={{ fontSize: 16, color: appTheme.textSecondary }}
                  />
                ) : (
                  <SunOutlined
                    style={{ fontSize: 16, color: appTheme.textSecondary }}
                  />
                )}
                <div>
                  <div
                    style={{
                      fontSize: 14,
                      fontWeight: 510,
                      color: appTheme.textPrimary,
                    }}
                  >
                    {t('profile.theme')}
                  </div>
                  <div
                    style={{
                      fontSize: 12,
                      color: appTheme.textTertiary,
                      marginTop: 2,
                    }}
                  >
                    {themeMode === 'dark'
                      ? t('profile.dark_mode')
                      : t('profile.light_mode')}
                  </div>
                </div>
              </div>
              <Switch
                checked={themeMode === 'dark'}
                onChange={handleThemeToggle}
                loading={themeLoading}
                checkedChildren={<MoonOutlined />}
                unCheckedChildren={<SunOutlined />}
              />
            </div>

            {/* Language Setting */}
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '12px 0',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <GlobalOutlined
                  style={{ fontSize: 16, color: appTheme.textSecondary }}
                />
                <div>
                  <div
                    style={{
                      fontSize: 14,
                      fontWeight: 510,
                      color: appTheme.textPrimary,
                    }}
                  >
                    {t('profile.language')}
                  </div>
                  <div
                    style={{
                      fontSize: 12,
                      color: appTheme.textTertiary,
                      marginTop: 2,
                    }}
                  >
                    {i18n.language === 'zh'
                      ? t('common.chinese')
                      : i18n.language === 'en'
                      ? t('common.english')
                      : t('common.chinese')}
                  </div>
                </div>
              </div>
              <Select
                value={i18n.language === 'zh' ? 'zh' : 'en'}
                onChange={handleLanguageChange}
                style={{
                  width: 120,
                }}
                options={[
                  { value: 'zh', label: t('common.chinese') },
                  { value: 'en', label: t('common.english') },
                ]}
              />
            </div>
            </div>
          </Card>

          {/* Account Info Card */}
          <Card
            title={
              <span
                style={{
                  fontSize: 14,
                  fontWeight: 590,
                  color: appTheme.textPrimary,
                  letterSpacing: '-0.12px',
                }}
              >
                {t('profile.account_info')}
              </span>
            }
            style={{
              background: themeMode === 'dark'
                ? 'rgba(255,255,255,0.02)'
                : 'rgba(255,255,255,0.80)',
              border: `1px solid ${
                themeMode === 'dark'
                  ? 'rgba(255,255,255,0.08)'
                  : 'rgba(0,0,0,0.08)'
              }`,
              borderRadius: 12,
            }}
            styles={{ body: { padding: '16px 24px' } }}
          >
            <div style={{ display: 'grid', gap: 12 }}>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                fontSize: 14,
              }}
            >
              <span style={{ color: appTheme.textTertiary }}>{t('profile.role')}</span>
              <span style={{ color: appTheme.textSecondary, fontWeight: 510 }}>
                {getRoleLabel(userInfo?.role || 1)}
              </span>
            </div>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                fontSize: 14,
              }}
            >
              <span style={{ color: appTheme.textTertiary }}>{t('profile.status')}</span>
              <span
                style={{
                  color:
                    userInfo?.status === 1 ? '#10b981' : appTheme.textTertiary,
                  fontWeight: 510,
                }}
              >
                {getStatusLabel(userInfo?.status || 1)}
              </span>
            </div>
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}

export default Profile
