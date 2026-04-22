import { useState, useEffect } from 'react'
import { useTheme } from '../contexts/ThemeContext'
import { Table, Button, Space, Tag, Modal, Form, Input, Select, InputNumber, message, Popconfirm, Row, Image, Upload } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, CloudServerOutlined, UploadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { getProviders, createProvider, updateProvider, deleteProvider, getProviderStatuses, uploadProviderLogo, Provider } from '../services/api'
import { useTranslation } from 'react-i18next'

const { TextArea } = Input

const ProviderManagement: React.FC = () => {
  const { t } = useTranslation()
  const { appTheme } = useTheme()
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
      message.error(t('provider.load_provider_list_failed'))
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
        message.success(t('provider.logo_upload_success'))
      }
    } catch (error) {
      message.error(t('provider.logo_upload_failed'))
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
        message.success(t('provider.delete_success'))
        loadProviders()
      } else {
        message.error(res.data.message || t('provider.delete_failed'))
      }
    } catch (error) {
      message.error(t('provider.delete_failed'))
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      if (editingProvider) {
        const res = await updateProvider(editingProvider.id, values)
        if (res.data.success) {
          message.success(t('provider.update_success'))
          setModalVisible(false)
          loadProviders()
        } else {
          message.error(res.data.message || t('provider.update_failed'))
        }
      } else {
        const res = await createProvider(values)
        if (res.data.success) {
          message.success(t('provider.create_success'))
          setModalVisible(false)
          loadProviders()
        } else {
          message.error(res.data.message || t('provider.create_failed'))
        }
      }
    } catch (error: any) {
      message.error(editingProvider ? t('provider.update_failed') : t('provider.create_failed'))
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
      case 'active': return t('provider.status_active')
      case 'maintenance': return t('provider.status_maintenance')
      case 'disabled': return t('provider.status_disabled')
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
          style={{ objectFit: 'contain', background: appTheme.bgElevated, borderRadius: 4 }}
          fallback="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="
        />
      ) : <div style={{ width: 40, height: 40, background: appTheme.bgElevated, borderRadius: 4 }} />
    },
    {
      title: t('provider.code'),
      dataIndex: 'code',
      key: 'code',
      width: 120,
      render: (v: string) => <Tag>{v}</Tag>
    },
    {
      title: t('provider.name'),
      dataIndex: 'name',
      key: 'name',
      width: 150,
    },
    {
      title: t('provider.description'),
      dataIndex: 'description',
      key: 'description',
      width: 200,
      ellipsis: true,
    },
    {
      title: t('provider.website'),
      dataIndex: 'website',
      key: 'website',
      width: 180,
      ellipsis: true,
      render: (v: string) => v ? (
        <a href={v} target="_blank" rel="noopener noreferrer">{v}</a>
      ) : '-'
    },
    {
      title: t('provider.status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (v: string) => <Tag color={getStatusColor(v)}>{getStatusLabel(v)}</Tag>
    },
    {
      title: t('provider.sort_order'),
      dataIndex: 'sort_order',
      key: 'sort_order',
      width: 80,
    },
    {
      title: t('provider.action'),
      key: 'action',
      width: 130,
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            {t('provider.edit')}
          </Button>
          <Popconfirm title={t('provider.confirm_delete')} onConfirm={() => handleDelete(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              {t('provider.delete')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2 style={{ margin: 0 }}><CloudServerOutlined /> {t('provider.provider_management')}</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
          {t('provider.add_provider')}
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={providers}
        rowKey="id"
        loading={loading}
        scroll={{ x: true }}
        pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (total) => t('provider.total_records').replace('{total}', String(total)) }}
      />

      <Modal
        title={editingProvider ? t('provider.edit_provider') : t('provider.add_provider_title')}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        width={600}
        okText={t('provider.save')}
        cancelText={t('provider.cancel')}
      >
        <Form form={form} layout="vertical" initialValues={{ status: 'active', sort_order: 0 }}>
          <Row gutter={16}>
            <Form.Item name="code" label={t('provider.code')} rules={[{ required: true, message: t('provider.enter_code') }]} style={{ flex: 1 }}>
              <Input placeholder={t('provider.like_bedi')} disabled={!!editingProvider} />
            </Form.Item>
            <Form.Item name="name" label={t('provider.name')} rules={[{ required: true, message: t('provider.enter_name') }]} style={{ flex: 1 }}>
              <Input placeholder={t('provider.like_bedi')} />
            </Form.Item>
          </Row>

          <Form.Item name="logo_url" label={t('provider.logo_url')}>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Input
                placeholder={t('provider.like_https_bedicloud')}
                value={form.getFieldValue('logo_url')}
                onChange={(e) => {
                  form.setFieldsValue({ logo_url: e.target.value })
                  setLogoPreview(e.target.value)
                }}
              />
              <Upload beforeUpload={handleLogoUpload} showUploadList={false} accept="image/*">
                <Button icon={<UploadOutlined />} loading={uploadingLogo}>
                  {t('provider.upload_logo')}
                </Button>
              </Upload>
              {logoPreview && (
                <div style={{ marginTop: 8 }}>
                  <img
                    src={logoPreview}
                    alt="logo preview"
                    style={{ width: 48, height: 48, objectFit: 'contain', background: appTheme.bgElevated, borderRadius: 8 }}
                    onError={() => setLogoPreview('')}
                  />
                </div>
              )}
            </Space>
          </Form.Item>

          <Form.Item name="description" label={t('provider.description')}>
            <TextArea rows={2} placeholder={t('provider.provider_description_placeholder')} />
          </Form.Item>

          <Form.Item name="website" label={t('provider.website')}>
            <Input placeholder={t('provider.like_https_bedicloud')} />
          </Form.Item>

          <Row gutter={16}>
            <Form.Item name="status" label={t('provider.status')} style={{ flex: 1 }}>
              <Select options={statuses} placeholder={t('provider.select_status')} />
            </Form.Item>
            <Form.Item name="sort_order" label={t('provider.sort_order')} style={{ flex: 1 }}>
              <InputNumber style={{ width: '100%' }} placeholder="0" min={0} />
            </Form.Item>
          </Row>
        </Form>
      </Modal>
    </div>
  )
}

export default ProviderManagement
