import { useState, useEffect } from 'react'
import { Card, Table, Button, Modal, Form, Input, message, Popconfirm, Tag, Space } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import { getAllNotifications, createNotification, deleteNotification } from '../services/api'
import { useTranslation } from 'react-i18next'

const { TextArea } = Input

interface Notification {
  id: number
  user_id: number
  title: string
  content: string
  type: string
  is_read: boolean
  read_at?: number
  created_at: number
}

const NotificationManage: React.FC = () => {
  const { t } = useTranslation()
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [loading, setLoading] = useState(true)
  const [modalVisible, setModalVisible] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm()

  useEffect(() => {
    loadNotifications()
  }, [])

  const loadNotifications = async () => {
    setLoading(true)
    try {
      const res = await getAllNotifications(100, 0)
      if (res.data?.success) {
        setNotifications(res.data.data || [])
      }
    } catch (error) {
      console.error('Failed to load notifications:', error)
      message.error(t('notification.load_notification_failed'))
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: { title: string; content: string }) => {
    setSubmitting(true)
    try {
      const res = await createNotification(values)
      if (res.data?.success) {
        message.success(t('notification.notification_created'))
        setModalVisible(false)
        form.resetFields()
        loadNotifications()
      } else {
        message.error(res.data?.message || t('notification.create_failed'))
      }
    } catch (error) {
      console.error('Failed to create notification:', error)
      message.error(t('notification.create_failed'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async (id: number) => {
    try {
      const res = await deleteNotification(id)
      if (res.data?.success) {
        message.success(t('notification.delete_success'))
        loadNotifications()
      } else {
        message.error(res.data?.message || t('notification.delete_failed'))
      }
    } catch (error) {
      console.error('Failed to delete notification:', error)
      message.error(t('notification.delete_failed'))
    }
  }

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 60,
    },
    {
      title: t('notification.title'),
      dataIndex: 'title',
      key: 'title',
    },
    {
      title: t('notification.content'),
      dataIndex: 'content',
      key: 'content',
      ellipsis: true,
    },
    {
      title: t('notification.type'),
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => (
        <Tag color={type === 'alert' ? 'red' : 'blue'}>
          {type === 'alert' ? t('notification.alert') : t('notification.system')}
        </Tag>
      ),
    },
    {
      title: t('notification.create_time'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (timestamp: number) => new Date(timestamp * 1000).toLocaleString(),
    },
    {
      title: t('notification.action'),
      key: 'action',
      width: 100,
      render: (_: any, record: Notification) => (
        <Space>
          <Popconfirm
            title={t('notification.confirm_delete_notification')}
            onConfirm={() => handleDelete(record.id)}
            okText={t('common.confirm')}
            cancelText={t('common.cancel')}
          >
            <Button type="link" danger icon={<DeleteOutlined />} size="small" />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Card
        title={t('notification.system_notification_management')}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalVisible(true)}>
            {t('notification.send_notification')}
          </Button>
        }
      >
        <Table
          dataSource={notifications}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 20 }}
        />
      </Card>

      <Modal
        title={t('notification.send_notification')}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false)
          form.resetFields()
        }}
        footer={null}
      >
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Form.Item
            name="title"
            label={t('notification.title')}
            rules={[{ required: true, message: t('notification.notification_title_placeholder') }]}
          >
            <Input placeholder={t('notification.notification_title_placeholder')} />
          </Form.Item>
          <Form.Item
            name="content"
            label={t('notification.content')}
            rules={[{ required: true, message: t('notification.notification_content_placeholder') }]}
          >
            <TextArea rows={4} placeholder={t('notification.notification_content_placeholder')} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={submitting}>
                {t('notification.send')}
              </Button>
              <Button onClick={() => {
                setModalVisible(false)
                form.resetFields()
              }}>
                {t('common.cancel')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default NotificationManage