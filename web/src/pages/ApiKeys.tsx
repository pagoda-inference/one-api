import { useState, useEffect } from 'react'
import { Card, Table, Button, Modal, Form, Input, InputNumber, Tag, Space, message, Popconfirm, Row, Col, Switch, Select } from 'antd'
import { PlusOutlined, DeleteOutlined, CopyOutlined, EyeOutlined, EyeInvisibleOutlined, EditOutlined } from '@ant-design/icons'
import { getTokens, createToken, deleteToken, updateToken, getMarketModels } from '../services/api'
import { useTranslation } from 'react-i18next'

interface Token {
  id: number
  name: string
  key: string
  fullKey: string
  status: string
  status_id: number
  created_at: number
  accessed_at: number
  used_quota: number
  remain_quota: number
  rate_limit_rpm: number
  rate_limit_tpm: number
  rate_limit_concurrent: number
  models: string
  modelNames: string[]
}

const ApiKeys: React.FC = () => {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [tokens, setTokens] = useState<Token[]>([])
  const [modalVisible, setModalVisible] = useState(false)
  const [editModalVisible, setEditModalVisible] = useState(false)
  const [editingToken, setEditingToken] = useState<Token | null>(null)
  const [createLoading, setCreateLoading] = useState(false)
  const [showKey, setShowKey] = useState<number | null>(null)
  const [form] = Form.useForm()
  const [editForm] = Form.useForm()
  const [availableModels, setAvailableModels] = useState<{ id: string; name: string }[]>([])

  useEffect(() => {
    loadTokens()
    loadModels()
  }, [])

  const loadModels = async () => {
    try {
      const res = await getMarketModels({ limit: 500 })
      if (res.data?.success) {
        const models = (res.data.data?.models || []).map((m: any) => ({
          id: m.id,
          name: m.name || m.id,
        }))
        setAvailableModels(models)
      }
    } catch (error) {
      console.error('Failed to load models:', error)
    }
  }

  const loadTokens = async () => {
    try {
      setLoading(true)
      const res = await getTokens({ limit: 50 })
      // Transform data to match Token interface
      const tokenData = (res.data.data || []).map((t: any) => ({
        id: t.id,
        name: t.name || t('common.no_data'),
        key: t.key ? `sk-${t.key.substring(0, 8)}...` : '',  // 用于显示
        fullKey: t.key ? `sk-${t.key}` : '',  // 用于复制
        status: t.status === 1 ? 'active' : 'disabled',
        status_id: t.status,
        created_at: t.created_time,
        accessed_at: t.accessed_time,
        used_quota: t.used_quota || 0,
        remain_quota: t.remain_quota || 0,
        rate_limit_rpm: t.rate_limit_rpm || 0,
        rate_limit_tpm: t.rate_limit_tpm || 0,
        rate_limit_concurrent: t.rate_limit_concurrent || 0,
        models: t.models || '',
        modelNames: t.models ? t.models.split(',').filter(Boolean) : []
      }))
      setTokens(tokenData)
    } catch (error) {
      console.error('Failed to load tokens:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: { name: string; models?: string[] }) => {
    try {
      setCreateLoading(true)
      await createToken({
        name: values.name,
        models: values.models?.join(','),
      })
      message.success(t('token.api_key_created'))
      setModalVisible(false)
      form.resetFields()
      loadTokens()
    } catch (error) {
      message.error(t('token.create_failed'))
    } finally {
      setCreateLoading(false)
    }
  }

  const handleEdit = (record: Token) => {
    setEditingToken(record)
    editForm.setFieldsValue({
      name: record.name,
      status: record.status_id === 1,
      remain_quota: record.remain_quota,
      rate_limit_rpm: record.rate_limit_rpm,
      rate_limit_tpm: record.rate_limit_tpm,
      rate_limit_concurrent: record.rate_limit_concurrent,
      models: record.modelNames,
    })
    setEditModalVisible(true)
  }

  const handleUpdate = async (values: { name: string; status?: boolean; remain_quota?: number; rate_limit_rpm?: number; rate_limit_tpm?: number; rate_limit_concurrent?: number; models?: string[] }) => {
    if (!editingToken) return
    try {
      await updateToken(editingToken.id, {
        name: values.name,
        status: values.status ? 1 : 2,
        remain_quota: values.remain_quota,
        rate_limit_rpm: values.rate_limit_rpm,
        rate_limit_tpm: values.rate_limit_tpm,
        rate_limit_concurrent: values.rate_limit_concurrent,
        models: values.models?.join(','),
      })
      message.success(t('token.update_success'))
      setEditModalVisible(false)
      loadTokens()
    } catch (error) {
      message.error(t('token.update_failed'))
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteToken(id)
      message.success(t('token.deleted'))
      loadTokens()
    } catch (error) {
      message.error(t('token.delete_failed'))
    }
  }

  const copyKey = (key: string) => {
    navigator.clipboard.writeText(key)
    message.success(t('token.copy_success'))
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
    { title: t('common.name'), dataIndex: 'name', key: 'name' },
    {
      title: t('token.api_key'),
      dataIndex: 'key',
      key: 'key',
      render: (key: string, record: Token) => (
        <Space>
          <code style={{ background: '#f5f5f5', padding: '4px 8px', borderRadius: 4 }}>
            {showKey === record.id ? record.fullKey : key}
          </code>
          <Button
            type="text"
            size="small"
            icon={showKey === record.id ? <EyeInvisibleOutlined /> : <EyeOutlined />}
            onClick={() => setShowKey(showKey === record.id ? null : record.id)}
          />
          <Button type="text" size="small" icon={<CopyOutlined />} onClick={() => copyKey(record.fullKey)} />
        </Space>
      )
    },
    {
      title: t('token.used_quota'),
      dataIndex: 'used_quota',
      key: 'used_quota',
      render: (v: number) => formatQuota(v)
    },
    {
      title: t('token.remaining_quota'),
      dataIndex: 'remain_quota',
      key: 'remain_quota',
      render: (v: number) => (
        <span style={{ color: v < 10000 ? '#ff4d4f' : '#52c41a' }}>
          {formatQuota(v)}
        </span>
      )
    },
    {
      title: t('common.status'),
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'active' ? 'green' : 'red'}>
          {status === 'active' ? t('channel.enabled') : t('channel.disabled')}
        </Tag>
      )
    },
    {
      title: t('token.created_time'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (v: number) => new Date(v * 1000).toLocaleString()
    },
    {
      title: t('token.last_used'),
      dataIndex: 'accessed_at',
      key: 'accessed_at',
      render: (v: number) => v > 0 ? new Date(v * 1000).toLocaleString() : t('token.never_used')
    },
    {
      title: t('token.rate_limit_rpm_tpm'),
      key: 'rate_limit',
      render: (_: any, record: Token) => {
        const rpm = record.rate_limit_rpm || 0
        const tpm = record.rate_limit_tpm || 0
        const concurrent = record.rate_limit_concurrent || 0
        const parts = []
        if (rpm > 0) parts.push(`RPM:${rpm}`)
        if (tpm > 0) parts.push(`TPM:${tpm}`)
        if (concurrent > 0) parts.push(`${t('token.concurrent')}:${concurrent}`)
        return parts.length > 0 ? <Tag>{parts.join(' / ')}</Tag> : <Tag color="green">{t('token.unlimited')}</Tag>
      }
    },
    {
      title: t('token.allowed_models'),
      key: 'models',
      render: (_: any, record: Token) => {
        if (!record.models) {
          return <Tag color="blue">{t('token.all_models')}</Tag>
        }
        const modelList = record.modelNames
        if (modelList.length === 0) {
          return <Tag color="blue">{t('token.all_models')}</Tag>
        }
        if (modelList.length > 3) {
          return <Tag>{modelList.slice(0, 3).join(', ')}...</Tag>
        }
        return <Tag>{modelList.join(', ')}</Tag>
      }
    },
    {
      title: t('common.action'),
      key: 'action',
      render: (_: any, record: Token) => (
        <Space>
          <Button type="text" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            {t('common.edit')}
          </Button>
          <Popconfirm
            title={t('token.confirm_delete_key')}
            onConfirm={() => handleDelete(record.id)}
            okText={t('common.confirm')}
            cancelText={t('common.cancel')}
          >
            <Button type="text" danger size="small" icon={<DeleteOutlined />}>
              {t('common.delete')}
            </Button>
          </Popconfirm>
        </Space>
      )
    }
  ]

  return (
    <div>
      <Card
        title={t('menu.tokens')}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalVisible(true)}>
            {t('token.create_new_api_key')}
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

      <Card title={t('token.usage_instructions')} style={{ marginTop: 16 }}>
        <div style={{ color: '#666' }}>
          <h4>{t('token.api_key_usage_method')}</h4>
          <pre style={{ background: '#f5f5f5', padding: 16, borderRadius: 8 }}>
{`import openai
openai.api_key = "your-api-key"
openai.api_base = "https://baotaai.bedicloud.net/v1"

response = openai.ChatCompletion.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello!"}]
)`}
          </pre>

          <h4 style={{ marginTop: 16 }}>{t('token.notes')}</h4>
          <ul>
            <li>{t('token.api_key_display_warning')}</li>
            <li>{t('token.each_key_independent_quota')}</li>
            <li>{t('token.can_set_model_restrictions')}</li>
            <li>{t('token.rotate_keys_regularly')}</li>
          </ul>
        </div>
      </Card>

      <Modal
        title={t('token.create_new_api_key')}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
      >
        <Form form={form} onFinish={handleCreate} layout="vertical">
          <Form.Item
            name="name"
            label={t('token.key_name')}
            rules={[{ required: true, message: t('token.enter_key_name') }]}
          >
            <Input placeholder={t('token.like_dev_env_key')} />
          </Form.Item>
          <Form.Item
            name="models"
            label={t('token.allowed_models')}
            tooltip={t('token.no_selection_allow_all')}
          >
            <Select
              mode="multiple"
              placeholder={t('token.no_selection_allow_all')}
              allowClear
              options={availableModels.map(m => ({ label: m.name, value: m.id }))}
              style={{ width: '100%' }}
              maxTagCount={3}
            />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={createLoading}>
                {t('common.add')}
              </Button>
              <Button onClick={() => setModalVisible(false)}>
                {t('common.cancel')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('channel.edit_channel')}
        open={editModalVisible}
        onCancel={() => setEditModalVisible(false)}
        footer={null}
      >
        <Form form={editForm} onFinish={handleUpdate} layout="vertical">
          <Form.Item name="name" label={t('token.key_name')} rules={[{ required: true, message: t('token.enter_key_name') }]}>
            <Input placeholder={t('token.like_dev_env_key')} />
          </Form.Item>
          <Form.Item name="models" label={t('token.allowed_models')} tooltip={t('token.no_selection_allow_all')}>
            <Select
              mode="multiple"
              placeholder={t('token.no_selection_allow_all')}
              allowClear
              options={availableModels.map(m => ({ label: m.name, value: m.id }))}
              style={{ width: '100%' }}
              maxTagCount={3}
            />
          </Form.Item>
          <Form.Item name="status" label={t('common.status')} valuePropName="checked">
            <Switch checkedChildren={t('channel.enabled')} unCheckedChildren={t('channel.disabled')} />
          </Form.Item>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="rate_limit_rpm" label={t('token.rpm_limit')}>
                <InputNumber style={{ width: '100%' }} placeholder={t('token.no_limit')} min={0} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="rate_limit_tpm" label={t('token.tpm_limit')}>
                <InputNumber style={{ width: '100%' }} placeholder={t('token.no_limit')} min={0} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="rate_limit_concurrent" label={t('token.concurrent')}>
                <InputNumber style={{ width: '100%' }} placeholder={t('token.no_limit')} min={0} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="remain_quota" label={t('token.remain_quota')}>
            <InputNumber style={{ width: '100%' }} placeholder={t('token.negative_unlimited')} min={-1} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {t('common.save')}
              </Button>
              <Button onClick={() => setEditModalVisible(false)}>
                {t('common.cancel')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default ApiKeys