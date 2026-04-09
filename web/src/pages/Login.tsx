import { useState, useEffect } from 'react'
import { Button, message, Form, Input, Select } from 'antd'
import { GlobalOutlined, MoonOutlined, SunOutlined } from '@ant-design/icons'
import Logo from '../components/Logo'
import { useTranslation } from 'react-i18next'
import { useLanguage } from '../contexts/LanguageContext'
import { useTheme } from '../contexts/ThemeContext'

interface LarkApp {
  id: number
  name: string
  client_id: string
  enabled: boolean
  sort: number
}

const Login: React.FC = () => {
  const { t } = useTranslation()
  const { language, toggleLanguage } = useLanguage()
  const { themeMode, toggleTheme } = useTheme()
  const [loginType, setLoginType] = useState<'feishu' | 'password'>('feishu')
  const [loading, setLoading] = useState(false)
  const [larkApps, setLarkApps] = useState<LarkApp[]>([])
  const [selectedAppId, setSelectedAppId] = useState<number | null>(null)

  useEffect(() => {
    fetchLarkApps()
  }, [])

  const fetchLarkApps = async () => {
    try {
      const res = await fetch('/api/lark-apps')
      const data = await res.json()
      if (data.success && data.data && data.data.length > 0) {
        setLarkApps(data.data)
        // If only one app, select it by default
        if (data.data.length === 1) {
          setSelectedAppId(data.data[0].id)
        }
      }
    } catch (err) {
      console.error('Failed to fetch Lark apps:', err)
    }
  }

  const handlePasswordLogin = async (values: { username: string; password: string }) => {
    setLoading(true)
    try {
      const res = await fetch('/api/user/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(values),
        credentials: 'include',
      })
      const data = await res.json()
      if (data.success) {
        localStorage.setItem('user_info', JSON.stringify(data.data))
        localStorage.setItem('access_token', `sk-${data.data.id}`)
        window.location.href = '/dashboard'
      } else {
        message.error(data.message || t('login.login_failed'))
      }
    } catch {
      message.error(t('login.login_retry'))
    } finally {
      setLoading(false)
    }
  }

  const handleFeishuLogin = async () => {
    try {
      const stateRes = await fetch('/api/oauth/state')
      const stateData = await stateRes.json()
      if (!stateData.success || !stateData.data) {
        message.error(stateData.message || t('login.get_state_failed'))
        return
      }
      const state = stateData.data

      // Use selected app's client_id or fall back to legacy behavior
      let clientId = ''
      let redirectUri = ''

      if (selectedAppId) {
        const app = larkApps.find(a => a.id === selectedAppId)
        if (app) {
          clientId = app.client_id
          redirectUri = `${window.location.origin}/oauth/lark?app_id=${selectedAppId}`
        }
      } else {
        // Legacy single-app mode
        clientId = 'cli_a94c9bd14ef95bd2' // Fallback client_id
        redirectUri = `${window.location.origin}/oauth/lark`
      }

      const larkAuthUrl = `https://accounts.feishu.cn/open-apis/authen/v1/authorize?redirect_uri=${encodeURIComponent(redirectUri)}&client_id=${clientId}&state=${state}`
      window.location.href = larkAuthUrl
    } catch (err) {
      message.error(t('login.login_retry'))
    }
  }

  const isDark = themeMode === 'dark'

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      background: isDark ? '#000' : 'linear-gradient(135deg, #f8fbff 0%, #e8f4ff 100%)'
    }}>
      {/* 顶部右侧语言和主题切换 */}
      <div style={{
        position: 'fixed',
        top: 16,
        right: 16,
        zIndex: 100,
        display: 'flex',
        gap: 8
      }}>
        <Button
          type="text"
          icon={<GlobalOutlined />}
          onClick={toggleLanguage}
          style={{ color: isDark ? '#fff' : '#667eea' }}
        />
        <Button
          type="text"
          icon={isDark ? <SunOutlined /> : <MoonOutlined />}
          onClick={toggleTheme}
          style={{ color: isDark ? '#fff' : '#667eea' }}
        />
      </div>

      {/* 左侧品牌区域 */}
      <div style={{
        flex: 1,
        background: isDark ? '#001529' : 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 60,
        position: 'relative',
        overflow: 'hidden'
      }}>
        <div style={{
          position: 'absolute',
          top: '10%',
          left: '10%',
          width: 200,
          height: 200,
          borderRadius: '50%',
          border: '1px solid rgba(255,255,255,0.1)'
        }} />
        <div style={{
          position: 'absolute',
          bottom: '20%',
          right: '5%',
          width: 150,
          height: 150,
          borderRadius: '50%',
          background: 'rgba(255,255,255,0.05)'
        }} />
        <div style={{
          position: 'absolute',
          top: '40%',
          right: '15%',
          width: 80,
          height: 80,
          borderRadius: 20,
          background: 'rgba(255,255,255,0.08)',
          transform: 'rotate(15deg)'
        }} />

        <div style={{
          background: '#fff',
          borderRadius: 16,
          padding: '16px 32px',
          boxShadow: '0 8px 32px rgba(0,0,0,0.15)',
          marginBottom: 40,
          position: 'relative',
          zIndex: 1
        }}>
          <Logo width={160} height={40} />
        </div>

        <h1 style={{
          color: '#fff',
          fontSize: 28,
          fontWeight: 700,
          margin: 0,
          textAlign: 'center',
          position: 'relative',
          zIndex: 1
        }}>{t('login.title')}</h1>

        <p style={{
          color: 'rgba(255,255,255,0.9)',
          fontSize: 16,
          marginTop: 16,
          textAlign: 'center',
          maxWidth: 360,
          lineHeight: 1.6,
          position: 'relative',
          zIndex: 1
        }}>
          {t('login.subtitle')}<br />
          {language === 'zh' ? '统一接入 · 灵活路由 · 便捷管理' : 'Unified Access · Flexible Routing · Easy Management'}
        </p>

        <div style={{
          display: 'flex',
          gap: 24,
          marginTop: 48,
          position: 'relative',
          zIndex: 1
        }}>
          {[
            { icon: '🤖', label: language === 'zh' ? 'AI 模型' : 'AI Models' },
            { icon: '🔐', label: language === 'zh' ? '安全认证' : 'Secure Auth' },
            { icon: '📊', label: language === 'zh' ? '用量分析' : 'Usage Analytics' }
          ].map((item, i) => (
            <div key={i} style={{
              textAlign: 'center',
              color: '#fff'
            }}>
              <div style={{
                width: 56,
                height: 56,
                borderRadius: 16,
                background: 'rgba(255,255,255,0.15)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 24,
                marginBottom: 8
              }}>
                {item.icon}
              </div>
              <span style={{ fontSize: 12, opacity: 0.8 }}>{item.label}</span>
            </div>
          ))}
        </div>
      </div>

      {/* 右侧登录区域 */}
      <div style={{
        flex: 1,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 40,
        background: isDark ? '#141414' : '#fff'
      }}>
        <div style={{ width: '100%', maxWidth: 360 }}>
          <h2 style={{
            fontSize: 28,
            fontWeight: 700,
            color: isDark ? '#fff' : '#1a1a1a',
            margin: '0 0 8px'
          }}>{t('login.welcome_back')}</h2>
          <p style={{
            color: isDark ? '#a1a1a1' : '#666',
            margin: '0 0 24px',
            fontSize: 14
          }}>
            {loginType === 'feishu' ? t('login.feishu_login_desc') : t('login.password_login_desc')}
          </p>

          {loginType === 'feishu' ? (
            <>
              {/* 多飞书组织选择下拉 */}
              {larkApps.length > 1 && (
                <Form.Item style={{ marginBottom: 16 }}>
                  <Select
                    placeholder={language === 'zh' ? '选择组织' : 'Select Organization'}
                    value={selectedAppId}
                    onChange={(value) => setSelectedAppId(value)}
                    style={{ width: '100%', height: 48 }}
                    size="large"
                  >
                    {larkApps.map(app => (
                      <Select.Option key={app.id} value={app.id}>
                        {app.name}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>
              )}

              <Button
                type="primary"
                size="large"
                block
                onClick={handleFeishuLogin}
                disabled={larkApps.length > 1 && !selectedAppId}
                style={{
                  height: 56,
                  borderRadius: 12,
                  fontSize: 16,
                  fontWeight: 600,
                  background: 'linear-gradient(135deg, #0075ff 0%, #0052cc 100%)',
                  border: 'none',
                  boxShadow: '0 4px 16px rgba(0, 117, 255, 0.3)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: 12
                }}
              >
                <svg width="24" height="24" viewBox="0 0 36 36" fill="none">
                  <path d="M18 4C10.268 4 4 10.268 4 18s6.268 14 14 14 14-6.268 14-14S25.732 4 18 4z" fill="#fff"/>
                  <path d="M18 8c-5.523 0-10 4.477-10 10s4.477 10 10 10 10-4.477 10-10S23.523 8 18 8zm4.5 14.5h-9v-9h9v9z" fill="#12B7F5"/>
                </svg>
                {t('login.feishu_login')}
              </Button>

              <div style={{ textAlign: 'center', margin: '24px 0' }}>
                <a onClick={() => setLoginType('password')} style={{ color: '#667eea', cursor: 'pointer' }}>
                  {t('login.use_password')}
                </a>
              </div>
            </>
          ) : (
            <>
              <Form
                layout="vertical"
                onFinish={handlePasswordLogin}
                size="large"
              >
                <Form.Item name="username" rules={[{ required: true, message: language === 'zh' ? '请输入用户名' : 'Please enter username' }]}>
                  <Input placeholder={t('common.username')} />
                </Form.Item>
                <Form.Item name="password" rules={[{ required: true, message: language === 'zh' ? '请输入密码' : 'Please enter password' }]}>
                  <Input.Password placeholder={t('common.password')} />
                </Form.Item>
                <Form.Item>
                  <Button type="primary" htmlType="submit" block loading={loading} style={{
                    height: 48,
                    borderRadius: 10,
                    fontSize: 16,
                    fontWeight: 600,
                    background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                    border: 'none'
                  }}>
                    {t('common.login')}
                  </Button>
                </Form.Item>
              </Form>

              <div style={{ textAlign: 'center', margin: '16px 0' }}>
                <a onClick={() => setLoginType('feishu')} style={{ color: '#667eea', cursor: 'pointer' }}>
                  {t('login.use_feishu')}
                </a>
              </div>
            </>
          )}

          <p style={{
            color: isDark ? '#666' : '#999',
            fontSize: 12,
            marginTop: 24,
            textAlign: 'center'
          }}>
            {language === 'zh' ? '登录即表示同意我们的服务条款' : 'By logging in, you agree to our Terms of Service'}
          </p>

          <p style={{
            color: isDark ? '#444' : '#ccc',
            fontSize: 12,
            marginTop: 48,
            textAlign: 'center'
          }}>
            {t('login.copyright')}
          </p>
        </div>
      </div>
    </div>
  )
}

export default Login