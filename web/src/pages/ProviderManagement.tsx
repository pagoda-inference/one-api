import { useState, useEffect } from 'react'
import { Table, Button, Space, Tag, Modal, Form, Input, Select, InputNumber, message, Popconfirm, Row, Image, Upload } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, CloudServerOutlined, UploadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { getProviders, createProvider, updateProvider, deleteProvider, getProviderStatuses, uploadProviderLogo, Provider } from '../services/api'

const { TextArea } = Input

const ProviderManagement: React.FC = () => {
  const [providers, setProviders] = useState<Provider[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingProvider, setEditingProvider] = useState<Provider | null>(null)
  const [form] = Form.useForm()
  const [statuses, setStatuses] = useState<{ value: string; label: string }[]>([])
  const [uploadingLogo, setUploadingLogo] = useState(false)
  const [logoPreview, setLogoPreview] = useState<string>('')

  useEffect(() => {
    loadProviders()
    loadOptions()
  }, [])

  const loadProviders = async () => {
    try {
      setLoading(true)
      const res = await getProviders()
      if (res.data.success) {
        setProviders(res.data.data)
      }
    } catch (error) {
      message.error('加载供应商列表失败')
    } finally {
      setLoading(false)
    }
  }

  const loadOptions = async () => {
    try {
      const res = await getProviderStatuses()
      if (res.data.success) setStatuses(res.data.data)
    } catch (error) {
      console.error('Failed to load options:', error)
    }
  }

  const handleLogoUpload = async (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    try {
      setUploadingLogo(true)
      const res = await uploadProviderLogo(formData)
      if (res.data.success) {
        form.setFieldsValue({ logo_url: res.data.data.url })
        setLogoPreview(res.data.data.url)
        message.success('Logo上传成功')
      }
    } catch (error) {
      message.error('Logo上传失败')
    } finally {
      setUploadingLogo(false)
    }
    return false
  }

  const handleCreate = () => {
    setEditingProvider(null)
    form.resetFields()
    setLogoPreview('')
    setModalVisible(true)
  }

  const handleEdit = (record: Provider) => {
    setEditingProvider(record)
    form.setFieldsValue(record)
    setLogoPreview(record.logo_url || '')
    setModalVisible(true)
  }

  const handleDelete = async (id: number) => {
    try {
      const res = await deleteProvider(id)
      if (res.data.success) {
        message.success('删除成功')
        loadProviders()
      } else {
        message.error(res.data.message || '删除失败')
      }
    } catch (error) {
      message.error('删除失败')
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      if (editingProvider) {
        const res = await updateProvider(editingProvider.id, values)
        if (res.data.success) {
          message.success('更新成功')
          setModalVisible(false)
          loadProviders()
        } else {
          message.error(res.data.message || '更新失败')
        }
      } else {
        const res = await createProvider(values)
        if (res.data.success) {
          message.success('创建成功')
          setModalVisible(false)
          loadProviders()
        } else {
          message.error(res.data.message || '创建失败')
        }
      }
    } catch (error: any) {
      message.error(editingProvider ? '更新失败' : '创建失败')
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'green'
      case 'maintenance': return 'orange'
      case 'disabled': return 'red'
      default: return 'default'
    }
  }

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'active': return '正常'
      case 'maintenance': return '维护中'
      case 'disabled': return '禁用'
      default: return status
    }
  }

  const columns: ColumnsType<Provider> = [
    {
      title: 'Logo',
      dataIndex: 'logo_url',
      key: 'logo_url',
      width: 80,
      render: (v: string) => v ? (
        <Image
          src={v}
          alt="logo"
          width={40}
          height={40}
          style={{ objectFit: 'contain', background: '#f5f5f5', borderRadius: 4 }}
          fallback="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="
        />
      ) : <div style={{ width: 40, height: 40, background: '#f5f5f5', borderRadius: 4 }} />
    },
    {
      title: 'Code',
      dataIndex: 'code',
      key: 'code',
      width: 120,
      render: (v: string) => <Tag>{v}</Tag>
    },
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 150,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      width: 200,
      ellipsis: true,
    },
    {
      title: '官网',
      dataIndex: 'website',
      key: 'website',
      width: 180,
      ellipsis: true,
      render: (v: string) => v ? (
        <a href={v} target="_blank" rel="noopener noreferrer">{v}</a>
      ) : '-'
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (v: string) => <Tag color={getStatusColor(v)}>{getStatusLabel(v)}</Tag>
    },
    {
      title: '排序',
      dataIndex: 'sort_order',
      key: 'sort_order',
      width: 80,
    },
    {
      title: '操作',
      key: 'action',
      width: 130,
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Popconfirm title="确定删除？" onConfirm={() => handleDelete(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2 style={{ margin: 0 }}><CloudServerOutlined /> Provider 管理</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
          添加 Provider
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={providers}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (total) => `共 ${total} 条` }}
      />

      <Modal
        title={editingProvider ? '编辑 Provider' : '添加 Provider'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        width={600}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" initialValues={{ status: 'active', sort_order: 0 }}>
          <Row gutter={16}>
            <Form.Item name="code" label="Code" rules={[{ required: true, message: '请输入 Code' }]} style={{ flex: 1 }}>
              <Input placeholder="如: bedi, openai" disabled={!!editingProvider} />
            </Form.Item>
            <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }]} style={{ flex: 1 }}>
              <Input placeholder="如: BEDI" />
            </Form.Item>
          </Row>

          <Form.Item name="logo_url" label="Logo URL">
            <Space direction="vertical" style={{ width: '100%' }}>
              <Input
                placeholder="如: /logos/bedi.png 或 https://..."
                value={form.getFieldValue('logo_url')}
                onChange={(e) => {
                  form.setFieldsValue({ logo_url: e.target.value })
                  setLogoPreview(e.target.value)
                }}
              />
              <Upload beforeUpload={handleLogoUpload} showUploadList={false} accept="image/*">
                <Button icon={<UploadOutlined />} loading={uploadingLogo}>
                  上传Logo
                </Button>
              </Upload>
              {logoPreview && (
                <div style={{ marginTop: 8 }}>
                  <img
                    src={logoPreview}
                    alt="logo preview"
                    style={{ width: 48, height: 48, objectFit: 'contain', background: '#f5f5f5', borderRadius: 8 }}
                    onError={() => setLogoPreview('')}
                  />
                </div>
              )}
            </Space>
          </Form.Item>

          <Form.Item name="description" label="描述">
            <TextArea rows={2} placeholder="供应商描述信息" />
          </Form.Item>

          <Form.Item name="website" label="官网">
            <Input placeholder="如: https://bedicloud.net" />
          </Form.Item>

          <Row gutter={16}>
            <Form.Item name="status" label="状态" style={{ flex: 1 }}>
              <Select options={statuses} placeholder="选择状态" />
            </Form.Item>
            <Form.Item name="sort_order" label="排序" style={{ flex: 1 }}>
              <InputNumber style={{ width: '100%' }} placeholder="0" min={0} />
            </Form.Item>
          </Row>
        </Form>
      </Modal>
    </div>
  )
}

export default ProviderManagement
