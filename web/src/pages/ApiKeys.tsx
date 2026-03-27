import { useState, useEffect } from 'react'
import { Card, Table, Button, Modal, Form, Input, Tag, Space, message, Popconfirm } from 'antd'
import { PlusOutlined, DeleteOutlined, CopyOutlined, EyeOutlined, EyeInvisibleOutlined } from '@ant-design/icons'
import { getTokens, createToken, deleteToken } from '../services/api'

interface Token {
  id: number
  name: string
  key: string
  status: string
  created_at: number
  accessed_at: number
  used_quota: number
  remain_quota: number
}

const ApiKeys: React.FC = () => {
  const [loading, setLoading] = useState(false)
  const [tokens, setTokens] = useState<Token[]>([])
  const [modalVisible, setModalVisible] = useState(false)
  const [createLoading, setCreateLoading] = useState(false)
  const [showKey, setShowKey] = useState<number | null>(null)
  const [form] = Form.useForm()

  useEffect(() => {
    loadTokens()
  }, [])

  const loadTokens = async () => {
    try {
      setLoading(true)
      const res = await getTokens({ limit: 50 })
      // Transform data to match Token interface
      const tokenData = (res.data.data || []).map((t: any) => ({
        id: t.id,
        name: t.name || '未命名',
        key: t.key ? `sk-${t.key.substring(0, 8)}...` : '',
        status: t.status === 1 ? 'active' : 'disabled',
        created_at: t.created_time,
        accessed_at: t.accessed_time,
        used_quota: t.used_quota || 0,
        remain_quota: t.remain_quota || 0
      }))
      setTokens(tokenData)
    } catch (error) {
      console.error('Failed to load tokens:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: { name: string }) => {
    try {
      setCreateLoading(true)
      await createToken({ name: values.name })
      message.success('API Key 创建成功')
      setModalVisible(false)
      form.resetFields()
      loadTokens()
    } catch (error) {
      message.error('创建失败')
    } finally {
      setCreateLoading(false)
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteToken(id)
      message.success('已删除')
      loadTokens()
    } catch (error) {
      message.error('删除失败')
    }
  }

  const copyKey = (key: string) => {
    navigator.clipboard.writeText(key)
    message.success('已复制到剪贴板')
  }

  const formatQuota = (quota: number) => {
    if (quota >= 1000000) {
      return (quota / 1000000).toFixed(2) + 'M'
    }
    if (quota >= 1000) {
      return (quota / 1000).toFixed(2) + 'K'
    }
    return quota.toString()
  }

  const columns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    {
      title: 'API Key',
      dataIndex: 'key',
      key: 'key',
      render: (key: string, record: Token) => (
        <Space>
          <code style={{ background: '#f5f5f5', padding: '4px 8px', borderRadius: 4 }}>
            {showKey === record.id ? record.key : key}
          </code>
          <Button
            type="text"
            size="small"
            icon={showKey === record.id ? <EyeInvisibleOutlined /> : <EyeOutlined />}
            onClick={() => setShowKey(showKey === record.id ? null : record.id)}
          />
          <Button type="text" size="small" icon={<CopyOutlined />} onClick={() => copyKey(record.key)} />
        </Space>
      )
    },
    {
      title: '已用额度',
      dataIndex: 'used_quota',
      key: 'used_quota',
      render: (v: number) => formatQuota(v)
    },
    {
      title: '剩余额度',
      dataIndex: 'remain_quota',
      key: 'remain_quota',
      render: (v: number) => (
        <span style={{ color: v < 10000 ? '#ff4d4f' : '#52c41a' }}>
          {formatQuota(v)}
        </span>
      )
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'active' ? 'green' : 'red'}>
          {status === 'active' ? '启用' : '禁用'}
        </Tag>
      )
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (v: number) => new Date(v * 1000).toLocaleString()
    },
    {
      title: '最近使用',
      dataIndex: 'accessed_at',
      key: 'accessed_at',
      render: (v: number) => v > 0 ? new Date(v * 1000).toLocaleString() : '从未使用'
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: Token) => (
        <Popconfirm
          title="确定要删除这个API Key吗？"
          onConfirm={() => handleDelete(record.id)}
          okText="确定"
          cancelText="取消"
        >
          <Button type="text" danger size="small" icon={<DeleteOutlined />}>
            删除
          </Button>
        </Popconfirm>
      )
    }
  ]

  return (
    <div>
      <Card
        title="API Keys"
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalVisible(true)}>
            创建新的 Key
          </Button>
        }
      >
        <Table
          dataSource={tokens}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 10 }}
        />
      </Card>

      <Card title="使用说明" style={{ marginTop: 16 }}>
        <div style={{ color: '#666' }}>
          <h4>API Key 使用方法</h4>
          <pre style={{ background: '#f5f5f5', padding: 16, borderRadius: 8 }}>
{`import openai
openai.api_key = "your-api-key"
openai.api_base = "https://your-domain.com/v1"

response = openai.ChatCompletion.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello!"}]
)`}
          </pre>

          <h4 style={{ marginTop: 16 }}>注意事项</h4>
          <ul>
            <li>API Key 只显示一次，请妥善保管</li>
            <li>每个Key都有独立的额度限制</li>
            <li>可以设置Key的使用模型限制</li>
            <li>建议定期轮换Key以保障安全</li>
          </ul>
        </div>
      </Card>

      <Modal
        title="创建新的 API Key"
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
      >
        <Form form={form} onFinish={handleCreate} layout="vertical">
          <Form.Item
            name="name"
            label="Key 名称"
            rules={[{ required: true, message: '请输入Key名称' }]}
          >
            <Input placeholder="如: 开发环境 Key" />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={createLoading}>
                创建
              </Button>
              <Button onClick={() => setModalVisible(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default ApiKeys