import { useState, useEffect } from 'react'
import { useTheme } from '../contexts/ThemeContext'
import { Card, Form, Input, Button, message, Spin, Divider } from 'antd'
import { UserOutlined, MailOutlined, LockOutlined } from '@ant-design/icons'
import { getUserInfo, updateUserInfo } from '../services/api'
import { useTranslation } from 'react-i18next'

interface UserInfo {
  display_name: string
  email: string
  role: number
  status: number
}

const Profile: React.FC = () => {
  const { t } = useTranslation()
  const { appTheme } = useTheme()
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
      message.error(t('profile.load_user_info_failed'))
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (values: { display_name?: string; email?: string; password?: string }) => {
    try {
      setSaving(true)
      const res = await updateUserInfo(values)
      if (res.data?.success) {
        message.success(t('profile.save_success'))
        loadUserInfo()  // Refresh data
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
            <UserOutlined /> {t('profile.personal_center')}
          </span>
        }
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
        >
          <Divider>{t('profile.basic_info')}</Divider>

          <Form.Item
            label={t('profile.display_name')}
            name="display_name"
          >
            <Input prefix={<UserOutlined />} placeholder={t('profile.display_name_placeholder')} />
          </Form.Item>

          <Form.Item
            label={t('profile.email')}
            name="email"
            rules={[
              { type: 'email', message: t('profile.valid_email_required') }
            ]}
          >
            <Input prefix={<MailOutlined />} placeholder={t('profile.email_placeholder')} />
          </Form.Item>

          <Divider>{t('profile.change_password')}</Divider>

          <Form.Item
            label={t('profile.new_password')}
            name="password"
          >
            <Input.Password prefix={<LockOutlined />} placeholder={t('profile.leave_blank_no_change')} />
          </Form.Item>

          <Form.Item>
            <Button type="primary" htmlType="submit" loading={saving} block>
              {t('profile.save_changes')}
            </Button>
          </Form.Item>
        </Form>

        {userInfo && (
          <>
            <Divider>{t('profile.account_info')}</Divider>
            <div style={{ color: appTheme.textSecondary, fontSize: 13 }}>
              <p>{t('profile.role')}: {userInfo.role === 100 ? t('profile.super_admin') : userInfo.role === 10 ? t('profile.admin') : t('profile.normal_user')}</p>
              <p>{t('profile.status')}: {userInfo.status === 1 ? t('profile.normal') : t('profile.disabled')}</p>
            </div>
          </>
        )}
      </Card>
    </div>
  )
}

export default Profile