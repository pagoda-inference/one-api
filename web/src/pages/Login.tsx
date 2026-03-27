import { useState } from 'react'
import { Form, Input, Button, Card, message, Tabs, Divider } from 'antd'
import { UserOutlined, LockOutlined, MailOutlined } from '@ant-design/icons'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { login, sendEmailCode, resetPassword } from '../services/api'

const { TabPane } = Tabs

const Login: React.FC = () => {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [loading, setLoading] = useState(false)
  const [activeTab, setActiveTab] = useState<'login' | 'reset'>('login')
  const [resetEmail, setResetEmail] = useState('')
  const [countdown, setCountdown] = useState(0)

  const redirectTo = searchParams.get('redirect') || '/dashboard'

  const onLogin = async (values: { username: string; password: string }) => {
    if (!values.username || !values.password) {
      message.error('请输入用户名和密码')
      return
    }
    try {
      setLoading(true)
      const res = await login(values)
      if (res.data.token) {
        localStorage.setItem('access_token', res.data.token)
        message.success('登录成功')
        navigate(redirectTo, { replace: true })
      } else {
        message.error('登录失败')
      }
    } catch (error: any) {
      message.error(error.response?.data?.message || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  const onSendCode = async () => {
    if (!resetEmail) {
      message.error('请输入邮箱地址')
      return
    }
    try {
      await sendEmailCode(resetEmail, 'reset')
      message.success('验证码已发送到邮箱')
      setCountdown(60)
      const timer = setInterval(() => {
        setCountdown((prev) => {
          if (prev <= 1) {
            clearInterval(timer)
            return 0
          }
          return prev - 1
        })
      }, 1000)
    } catch (error: any) {
      message.error(error.response?.data?.message || '发送失败')
    }
  }

  const onResetPassword = async (values: { email: string; code: string; password: string }) => {
    try {
      setLoading(true)
      await resetPassword(values)
      message.success('密码重置成功，请使用新密码登录')
      setActiveTab('login')
    } catch (error: any) {
      message.error(error.response?.data?.message || '重置失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      padding: '20px'
    }}>
      <Card
        style={{
          width: '100%',
          maxWidth: 420,
          borderRadius: 16,
          boxShadow: '0 20px 60px rgba(0,0,0,0.3)',
          overflow: 'hidden'
        }}
        styles={{ body: { padding: 0 } }}
      >
        <div style={{
          background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
          padding: '32px 24px',
          textAlign: 'center'
        }}>
          <h1 style={{ color: '#fff', margin: 0, fontSize: 28, fontWeight: 700 }}>One API</h1>
          <p style={{ color: 'rgba(255,255,255,0.8)', margin: '8px 0 0', fontSize: 14 }}>AI 模型网关平台</p>
        </div>

        <div style={{ padding: '24px 24px 32px' }}>
          <Tabs
            activeKey={activeTab}
            onChange={(key) => setActiveTab(key as 'login' | 'reset')}
            centered
            style={{ marginBottom: 24 }}
          >
            <TabPane tab="登录" key="login" />
            <TabPane tab="找回密码" key="reset" />
          </Tabs>

          {activeTab === 'login' ? (
            <Form
              name="login"
              onFinish={onLogin}
              layout="vertical"
              requiredMark={false}
              initialValues={{ username: '', password: '' }}
            >
              <Form.Item
                name="username"
                rules={[{ required: true, message: '请输入用户名或邮箱' }]}
              >
                <Input
                  prefix={<UserOutlined style={{ color: '#bfbfbf' }} />}
                  placeholder="用户名 / 邮箱"
                  size="large"
                  style={{ borderRadius: 8, height: 48 }}
                />
              </Form.Item>

              <Form.Item
                name="password"
                rules={[{ required: true, message: '请输入密码' }]}
              >
                <Input.Password
                  prefix={<LockOutlined style={{ color: '#bfbfbf' }} />}
                  placeholder="密码"
                  size="large"
                  style={{ borderRadius: 8, height: 48 }}
                />
              </Form.Item>

              <Form.Item style={{ marginBottom: 16 }}>
                <Button
                  type="primary"
                  htmlType="submit"
                  block
                  loading={loading}
                  style={{
                    height: 48,
                    borderRadius: 8,
                    fontSize: 16,
                    fontWeight: 600,
                    background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                    border: 'none'
                  }}
                >
                  登录
                </Button>
              </Form.Item>

              <div style={{ textAlign: 'center' }}>
                <a style={{ color: '#667eea' }} onClick={() => navigate('/register')}>
                  还没有账号？立即注册
                </a>
              </div>
            </Form>
          ) : (
            <Form
              name="reset"
              onFinish={onResetPassword}
              layout="vertical"
              requiredMark={false}
            >
              <Form.Item>
                <Input
                  prefix={<MailOutlined style={{ color: '#bfbfbf' }} />}
                  placeholder="邮箱地址"
                  size="large"
                  style={{ borderRadius: 8, height: 48 }}
                  value={resetEmail}
                  onChange={(e) => setResetEmail(e.target.value)}
                />
              </Form.Item>

              <Form.Item style={{ marginBottom: 16 }}>
                <Button
                  type="primary"
                  onClick={onSendCode}
                  disabled={countdown > 0}
                  style={{
                    height: 48,
                    borderRadius: 8,
                    width: '100%',
                    background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                    border: 'none'
                  }}
                >
                  {countdown > 0 ? `${countdown}秒后重发` : '发送验证码'}
                </Button>
              </Form.Item>

              <Divider plain style={{ color: '#bfbfbf', fontSize: 12 }}>请查收邮件中的验证码</Divider>

              <Form.Item
                name="code"
                rules={[{ required: true, message: '请输入验证码' }]}
              >
                <Input
                  placeholder="验证码"
                  size="large"
                  style={{ borderRadius: 8, height: 48 }}
                />
              </Form.Item>

              <Form.Item
                name="password"
                rules={[
                  { required: true, message: '请输入新密码' },
                  { min: 8, message: '密码至少8位' }
                ]}
              >
                <Input.Password
                  prefix={<LockOutlined style={{ color: '#bfbfbf' }} />}
                  placeholder="新密码"
                  size="large"
                  style={{ borderRadius: 8, height: 48 }}
                />
              </Form.Item>

              <Form.Item style={{ marginBottom: 16 }}>
                <Button
                  type="primary"
                  htmlType="submit"
                  block
                  loading={loading}
                  style={{
                    height: 48,
                    borderRadius: 8,
                    fontSize: 16,
                    fontWeight: 600,
                    background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                    border: 'none'
                  }}
                >
                  重置密码
                </Button>
              </Form.Item>
            </Form>
          )}
        </div>
      </Card>
    </div>
  )
}

export default Login
