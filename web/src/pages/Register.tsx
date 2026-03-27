import { useState } from 'react'
import { Form, Input, Button, Card, message, Checkbox, Divider } from 'antd'
import { UserOutlined, LockOutlined, MailOutlined, GiftOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { register, sendEmailCode } from '../services/api'

const Register: React.FC = () => {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [codeSent, setCodeSent] = useState(false)
  const [countdown, setCountdown] = useState(0)

  const onSendCode = async (email: string) => {
    if (!email) {
      message.error('请输入邮箱地址')
      return
    }
    try {
      await sendEmailCode(email, 'register')
      message.success('验证码已发送到邮箱')
      setCodeSent(true)
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

  const onFinish = async (values: {
    username: string
    email: string
    password: string
    email_code: string
    invitation_code?: string
    agreement: boolean
  }) => {
    if (!values.agreement) {
      message.error('请同意用户协议')
      return
    }
    try {
      setLoading(true)
      await register({
        username: values.username,
        email: values.email,
        password: values.password,
        email_code: values.email_code,
        invitation_code: values.invitation_code
      })
      message.success('注册成功！请登录')
      navigate('/login')
    } catch (error: any) {
      message.error(error.response?.data?.message || '注册失败')
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
          <p style={{ color: 'rgba(255,255,255,0.8)', margin: '8px 0 0', fontSize: 14 }}>加入 AI 模型网关平台</p>
        </div>

        <div style={{ padding: '24px 24px 32px' }}>
          <Form
            name="register"
            onFinish={onFinish}
            layout="vertical"
            requiredMark={false}
            initialValues={{ agreement: false }}
          >
            <Form.Item
              name="username"
              rules={[
                { required: true, message: '请输入用户名' },
                { min: 3, max: 12, message: '用户名3-12位' }
              ]}
            >
              <Input
                prefix={<UserOutlined style={{ color: '#bfbfbf' }} />}
                placeholder="用户名 (3-12位)"
                size="large"
                style={{ borderRadius: 8, height: 48 }}
              />
            </Form.Item>

            <Form.Item
              name="email"
              rules={[
                { required: true, message: '请输入邮箱' },
                { type: 'email', message: '请输入有效邮箱' }
              ]}
            >
              <Input
                prefix={<MailOutlined style={{ color: '#bfbfbf' }} />}
                placeholder="邮箱地址"
                size="large"
                style={{ borderRadius: 8, height: 48 }}
              />
            </Form.Item>

            <Form.Item
              name="password"
              rules={[
                { required: true, message: '请输入密码' },
                { min: 8, max: 20, message: '密码8-20位' }
              ]}
            >
              <Input.Password
                prefix={<LockOutlined style={{ color: '#bfbfbf' }} />}
                placeholder="密码 (8-20位)"
                size="large"
                style={{ borderRadius: 8, height: 48 }}
              />
            </Form.Item>

            <div style={{ display: 'flex', gap: 8 }}>
              <Form.Item
                name="email_code"
                rules={[{ required: true, message: '请输入验证码' }]}
                style={{ flex: 1, marginBottom: 16 }}
              >
                <Input
                  placeholder="验证码"
                  size="large"
                  style={{ borderRadius: 8, height: 48 }}
                />
              </Form.Item>
              <Button
                onClick={() => {
                  const emailValue = document.querySelector('input[placeholder="邮箱地址"]') as HTMLInputElement
                  if (emailValue) onSendCode(emailValue.value)
                }}
                disabled={countdown > 0}
                style={{
                  height: 48,
                  borderRadius: 8,
                  background: codeSent ? '#f0f0f0' : 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                  border: 'none',
                  color: codeSent ? '#666' : '#fff',
                  fontWeight: 500
                }}
              >
                {countdown > 0 ? `${countdown}s` : '获取验证码'}
              </Button>
            </div>

            <Form.Item name="invitation_code">
              <Input
                prefix={<GiftOutlined style={{ color: '#bfbfbf' }} />}
                placeholder="邀请码 (选填)"
                size="large"
                style={{ borderRadius: 8, height: 48 }}
              />
            </Form.Item>

            <Form.Item name="agreement" valuePropName="checked">
              <Checkbox style={{ borderRadius: 4 }}>
                我已阅读并同意 <a style={{ color: '#667eea' }}>《用户协议》</a> 和 <a style={{ color: '#667eea' }}>《隐私政策》</a>
              </Checkbox>
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
                注册
              </Button>
            </Form.Item>

            <Divider plain style={{ color: '#bfbfbf', fontSize: 12 }}>已有账号？</Divider>

            <div style={{ textAlign: 'center' }}>
              <a style={{ color: '#667eea' }} onClick={() => navigate('/login')}>
                立即登录
              </a>
            </div>
          </Form>
        </div>
      </Card>
    </div>
  )
}

export default Register
