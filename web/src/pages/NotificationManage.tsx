import { useState, useEffect } from 'react'
import { Card, Table, Button, Modal, Form, Input, message, Popconfirm, Tag, Space } from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import { getAllNotifications, createNotification, deleteNotification } from '../services/api'

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
      message.error('加载通知失败')
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: { title: string; content: string }) => {
    setSubmitting(true)
    try {
      const res = await createNotification(values)
      if (res.data?.success) {
        message.success('通知已创建')
        setModalVisible(false)
        form.resetFields()
        loadNotifications()
      } else {
        message.error(res.data?.message || '创建失败')
      }
    } catch (error) {
      console.error('Failed to create notification:', error)
      message.error('创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async (id: number) => {
    try {
      const res = await deleteNotification(id)
      if (res.data?.success) {
        message.success('删除成功')
        loadNotifications()
      } else {
        message.error(res.data?.message || '删除失败')
      }
    } catch (error) {
      console.error('Failed to delete notification:', error)
      message.error('删除失败')
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
      title: '标题',
      dataIndex: 'title',
      key: 'title',
    },
    {
      title: '内容',
      dataIndex: 'content',
      key: 'content',
      ellipsis: true,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => (
        <Tag color={type === 'alert' ? 'red' : 'blue'}>
          {type === 'alert' ? '告警' : '系统'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (timestamp: number) => new Date(timestamp * 1000).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: any, record: Notification) => (
        <Space>
          <Popconfirm
            title="确认删除此通知？"
            onConfirm={() => handleDelete(record.id)}
            okText="确认"
            cancelText="取消"
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
        title="系统通知管理"
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalVisible(true)}>
            发送通知
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
        title="发送系统通知"
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
            label="通知标题"
            rules={[{ required: true, message: '请输入通知标题' }]}
          >
            <Input placeholder="请输入通知标题" />
          </Form.Item>
          <Form.Item
            name="content"
            label="通知内容"
            rules={[{ required: true, message: '请输入通知内容' }]}
          >
            <TextArea rows={4} placeholder="请输入通知内容" />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={submitting}>
                发送
              </Button>
              <Button onClick={() => {
                setModalVisible(false)
                form.resetFields()
              }}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default NotificationManage