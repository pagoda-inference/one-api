import { useState, useEffect } from 'react'
import { Card, Form, Input, Button, message, Spin, Divider } from 'antd'
import { UserOutlined, MailOutlined, LockOutlined } from '@ant-design/icons'
import { getUserInfo, updateUserInfo } from '../services/api'

interface UserInfo {
  display_name: string
  email: string
  role: number
  status: number
}

const Profile: React.FC = () => {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null)
  const [form] = Form.useForm()

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
      message.error('加载用户信息失败')
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (values: { display_name?: string; email?: string; password?: string }) => {
    try {
      setSaving(true)
      const res = await updateUserInfo(values)
      if (res.data?.success) {
        message.success('保存成功')
        loadUserInfo()  // Refresh data
      } else {
        message.error(res.data?.message || '保存失败')
      }
    } catch (error) {
      console.error('Failed to update:', error)
      message.error('保存失败')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '400px' }}>
        <Spin size="large" />
      </div>
    )
  }

  return (
    <div style={{ maxWidth: 600, margin: '0 auto', padding: '24px' }}>
      <Card
        title={
          <span>
            <UserOutlined /> 个人中心
          </span>
        }
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
        >
          <Divider>基本信息</Divider>

          <Form.Item
            label="显示名称"
            name="display_name"
          >
            <Input prefix={<UserOutlined />} placeholder="显示名称" />
          </Form.Item>

          <Form.Item
            label="邮箱"
            name="email"
            rules={[
              { type: 'email', message: '请输入有效的邮箱地址' }
            ]}
          >
            <Input prefix={<MailOutlined />} placeholder="邮箱地址（用于接收告警通知）" />
          </Form.Item>

          <Divider>修改密码</Divider>

          <Form.Item
            label="新密码"
            name="password"
          >
            <Input.Password prefix={<LockOutlined />} placeholder="留空则不修改密码" />
          </Form.Item>

          <Form.Item>
            <Button type="primary" htmlType="submit" loading={saving} block>
              保存修改
            </Button>
          </Form.Item>
        </Form>

        {userInfo && (
          <>
            <Divider>账户信息</Divider>
            <div style={{ color: '#666', fontSize: 13 }}>
              <p>角色: {userInfo.role === 100 ? '超级管理员' : userInfo.role === 10 ? '管理员' : '普通用户'}</p>
              <p>状态: {userInfo.status === 1 ? '正常' : '已禁用'}</p>
            </div>
          </>
        )}
      </Card>
    </div>
  )
}

export default Profile