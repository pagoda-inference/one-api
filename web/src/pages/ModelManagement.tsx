import { useState, useEffect } from 'react'
import { Table, Button, Space, Tag, Modal, Form, Input, Select, InputNumber, Upload, message, Popconfirm, Row, Checkbox } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, UploadOutlined, DatabaseOutlined, PictureOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { listModels, createModel, updateModel, deleteModel, batchDeleteModels, uploadModelLogo, getModelTypes, getModelStatuses, getProviders, listLogos, deleteLogo, getAllTenants, ModelItem, Provider, SimpleTenant } from '../services/api'

const { TextArea } = Input

const ModelManagement: React.FC = () => {
  const [models, setModels] = useState<ModelItem[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingModel, setEditingModel] = useState<ModelItem | null>(null)
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])
  const [form] = Form.useForm()
  const [modelTypes, setModelTypes] = useState<{ value: string; label: string }[]>([])
  const [modelStatuses, setModelStatuses] = useState<{ value: string; label: string }[]>([])
  const [uploadingLogo, setUploadingLogo] = useState(false)
  const [iconPreview, setIconPreview] = useState<string>('')
  const [providers, setProviders] = useState<Provider[]>([])
  const [logos, setLogos] = useState<{ name: string; url: string }[]>([])
  const [showLogoModal, setShowLogoModal] = useState(false)
  const [tenants, setTenants] = useState<SimpleTenant[]>([])

  useEffect(() => {
    loadModels()
    loadTenants()
  }, [])

  const loadTenants = async () => {
    try {
      const res = await getAllTenants()
      if (res.data.success) {
        setTenants(res.data.data || [])
      }
    } catch (error) {
      console.error('Failed to load tenants:', error)
    }
  }

  useEffect(() => {
    loadModels()
    loadOptions()
    loadProviders()
    loadLogos()
  }, [])

  const loadProviders = async () => {
    try {
      const res = await getProviders()
      if (res.data.success) {
        setProviders(res.data.data || [])
      }
    } catch (error) {
      console.error('Failed to load providers:', error)
    }
  }

  const loadLogos = async () => {
    try {
      const res = await listLogos()
      if (res.data.success) {
        setLogos(res.data.data || [])
      }
    } catch (error) {
      console.error('Failed to load logos:', error)
    }
  }

  const loadModels = async () => {
    try {
      setLoading(true)
      const res = await listModels()
      if (res.data.success) {
        setModels(res.data.data)
      }
    } catch (error) {
      message.error('加载模型列表失败')
    } finally {
      setLoading(false)
    }
  }

  const loadOptions = async () => {
    try {
      const [typesRes, statusesRes] = await Promise.all([
        getModelTypes(),
        getModelStatuses()
      ])
      if (typesRes.data.success) setModelTypes(typesRes.data.data)
      if (statusesRes.data.success) setModelStatuses(statusesRes.data.data)
    } catch (error) {
      console.error('Failed to load options:', error)
    }
  }

  const handleCreate = () => {
    setEditingModel(null)
    form.resetFields()
    setIconPreview('')
    setModalVisible(true)
  }

  const handleEdit = (record: ModelItem) => {
    setEditingModel(record)
    form.setFieldsValue(record)
    setIconPreview(record.icon_url || '')
    setModalVisible(true)
  }

  const handleDelete = async (id: string) => {
    try {
      const res = await deleteModel(id)
      if (res.data.success) {
        message.success('删除成功')
        loadModels()
      } else {
        message.error(res.data.message || '删除失败')
      }
    } catch (error) {
      message.error('删除失败')
    }
  }

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要删除的模型')
      return
    }
    try {
      const res = await batchDeleteModels(selectedRowKeys)
      if (res.data.success) {
        message.success(res.data.message)
        setSelectedRowKeys([])
        loadModels()
      } else {
        message.error(res.data.message || '批量删除失败')
      }
    } catch (error) {
      message.error('批量删除失败')
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      if (editingModel) {
        const res = await updateModel(editingModel.id, values)
        if (res.data.success) {
          message.success('更新成功')
          setModalVisible(false)
          loadModels()
        } else {
          message.error(res.data.message || '更新失败')
        }
      } else {
        const res = await createModel(values)
        if (res.data.success) {
          message.success('创建成功')
          setModalVisible(false)
          loadModels()
        } else {
          message.error(res.data.message || '创建失败')
        }
      }
    } catch (error: any) {
      message.error(editingModel ? '更新失败' : '创建失败')
    }
  }

  const handleLogoUpload = async (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    try {
      setUploadingLogo(true)
      const res = await uploadModelLogo(formData)
      if (res.data.success) {
        form.setFieldsValue({ icon_url: res.data.data.url })
        setIconPreview(res.data.data.url)
        message.success('Logo上传成功')
      }
    } catch (error) {
      message.error('Logo上传失败')
    } finally {
      setUploadingLogo(false)
    }
    return false
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
      case 'active': return '上线'
      case 'maintenance': return '维护中'
      case 'disabled': return '下架'
      default: return status
    }
  }

  const getTypeLabel = (type: string) => {
    const typeMap: Record<string, string> = {
      'chat': '对话模型',
      'vlm': '视觉模型',
      'embedding': 'Embedding',
      'reranker': 'Reranker',
      'ocr': 'OCR',
      'other': '其他'
    }
    return typeMap[type] || type
  }

  const columns: ColumnsType<ModelItem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 200,
      ellipsis: true,
    },
    {
      title: '模型名称',
      dataIndex: 'name',
      key: 'name',
      width: 150,
    },
    {
      title: '提供商',
      dataIndex: 'provider',
      key: 'provider',
      width: 100,
    },
    {
      title: '类型',
      dataIndex: 'model_type',
      key: 'model_type',
      width: 100,
      render: (v: string) => getTypeLabel(v)
    },
    {
      title: '上下文长度',
      dataIndex: 'context_len',
      key: 'context_len',
      width: 120,
      render: (v: number) => v ? v.toLocaleString() : '-'
    },
    {
      title: '输入价格',
      dataIndex: 'input_price',
      key: 'input_price',
      width: 100,
      render: (v: number) => v?.toFixed(6) || '0'
    },
    {
      title: '输出价格',
      dataIndex: 'output_price',
      key: 'output_price',
      width: 100,
      render: (v: number) => v?.toFixed(6) || '0'
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
        <h2 style={{ margin: 0 }}><DatabaseOutlined /> 模型管理</h2>
        <Space>
          {selectedRowKeys.length > 0 && (
            <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete}>
              批量删除 ({selectedRowKeys.length})
            </Button>
          )}
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            添加模型
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={models}
        rowKey="id"
        loading={loading}
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys as string[]),
        }}
        scroll={{ x: true }}
        pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (total) => `共 ${total} 条` }}
      />

      <Modal
        title={editingModel ? '编辑模型' : '添加模型'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
        width={600}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical" initialValues={{ status: 'active' }}>
          <Form.Item name="id" label="模型ID" rules={[{ required: true, message: '请输入模型ID' }]}>
            <Input placeholder="如: Qwen2.5-72B-Instruct 或 deepseek-ai/DeepSeek-R1" />
          </Form.Item>

          <Form.Item name="name" label="模型名称" rules={[{ required: true, message: '请输入模型名称' }]}>
            <Input placeholder="如: Qwen2.5-72B-Instruct" />
          </Form.Item>

          <Row gutter={16}>
            <Form.Item name="provider" label="提供商" style={{ flex: 1 }}>
              <Select
                showSearch
                placeholder="选择 Provider"
                options={providers.map(p => ({ value: p.code, label: p.name }))}
              />
            </Form.Item>

            <Form.Item name="model_type" label="模型类型" style={{ flex: 1 }}>
              <Select options={modelTypes} placeholder="选择模型类型" />
            </Form.Item>
          </Row>

          <Form.Item name="description" label="描述">
            <TextArea rows={2} placeholder="模型描述信息" />
          </Form.Item>

          <Row gutter={16}>
            <Form.Item name="context_len" label={form.getFieldValue('model_type') === 'embedding' ? '向量维度' : '上下文长度'} style={{ flex: 1 }}>
              <InputNumber
                style={{ width: '100%' }}
                placeholder={form.getFieldValue('model_type') === 'embedding' ? '如: 1536' : '如: 32768'}
                min={0}
              />
            </Form.Item>

            <Form.Item name="sort_order" label="排序" style={{ flex: 1 }}>
              <InputNumber style={{ width: '100%' }} placeholder="留空自动分配" min={0} />
            </Form.Item>
          </Row>

          <Row gutter={16}>
            <Form.Item name="input_price" label="输入价格 (元/千token)" style={{ flex: 1 }}>
              <InputNumber style={{ width: '100%' }} placeholder="0.000000" min={0} precision={6} />
            </Form.Item>

            <Form.Item name="output_price" label="输出价格 (元/千token)" style={{ flex: 1 }}>
              <InputNumber style={{ width: '100%' }} placeholder="0.000000" min={0} precision={6} />
            </Form.Item>
          </Row>

          <Form.Item name="status" label="状态">
            <Select options={modelStatuses} placeholder="选择状态" />
          </Form.Item>

          <Form.Item name="is_trial" valuePropName="checked">
            <Checkbox>支持试用</Checkbox>
          </Form.Item>

          <Row gutter={16}>
            <Form.Item name="rate_limit_rpm" label="RPM限流 (0=不限)" style={{ flex: 1 }}>
              <InputNumber style={{ width: '100%' }} placeholder="0" min={0} />
            </Form.Item>

            <Form.Item name="rate_limit_tpm" label="TPM限流 (0=不限)" style={{ flex: 1 }}>
              <InputNumber style={{ width: '100%' }} placeholder="0" min={0} />
            </Form.Item>
          </Row>

          <Form.Item
            name="visible_to_teams"
            label="可见团队"
            tooltip="不选择表示公共模型，选择团队则只有这些团队可见"
            valuePropName="value"
            getValueProps={(value) => ({ value: value ? value.split(',').filter(Boolean).map((v: string) => Number(v)) : [] })}
            getValueFromEvent={(values) => values && values.length > 0 ? ',' + values.join(',') + ',' : ''}
          >
            <Select
              mode="multiple"
              allowClear
              placeholder="不选择表示所有用户可见"
              options={tenants.map(t => ({ value: t.id, label: t.name }))}
            />
          </Form.Item>

          <Form.Item name="icon_url" label="Logo URL">
            <Space direction="vertical" style={{ width: '100%' }}>
              <Space>
                <Input
                  placeholder="Logo URL，如: /logos/qwen.png"
                  style={{ width: 300 }}
                  value={form.getFieldValue('icon_url')}
                  onChange={(e) => {
                    form.setFieldsValue({ icon_url: e.target.value })
                    setIconPreview(e.target.value)
                  }}
                />
                <Button icon={<PictureOutlined />} onClick={() => setShowLogoModal(true)}>
                  选择已有Logo
                </Button>
              </Space>
              <Upload beforeUpload={handleLogoUpload} showUploadList={false} accept="image/*">
                <Button icon={<UploadOutlined />} loading={uploadingLogo}>
                  上传新Logo
                </Button>
              </Upload>
              {iconPreview && (
                <div style={{ marginTop: 8 }}>
                  <img
                    src={iconPreview}
                    alt="logo preview"
                    style={{ width: 48, height: 48, objectFit: 'contain', background: '#f5f5f5', borderRadius: 8 }}
                    onError={() => setIconPreview('')}
                  />
                </div>
              )}
            </Space>
          </Form.Item>

          <Form.Item name="capabilities" label="能力 (JSON)">
            <TextArea rows={2} placeholder='["chat","vision"]' />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="选择Logo"
        open={showLogoModal}
        onCancel={() => setShowLogoModal(false)}
        footer={null}
        width={700}
      >
        <div style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: 8,
          maxHeight: 400,
          overflowY: 'auto',
        }}>
          {logos.map((logo) => (
            <div
              key={logo.name}
              style={{
                cursor: 'pointer',
                padding: 8,
                border: '1px solid #f0f0f0',
                borderRadius: 8,
                textAlign: 'center',
                width: 100,
                flexShrink: 0,
                position: 'relative',
              }}
            >
              <div
                onClick={() => {
                  form.setFieldsValue({ icon_url: logo.url })
                  setIconPreview(logo.url)
                  setShowLogoModal(false)
                }}
              >
                <img
                  src={logo.url}
                  alt={logo.name}
                  style={{ width: 64, height: 64, objectFit: 'contain' }}
                />
                <div style={{ fontSize: 12, marginTop: 4, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {logo.name}
                </div>
              </div>
              <Button
                type="text"
                danger
                size="small"
                icon={<DeleteOutlined />}
                style={{ position: 'absolute', top: 2, right: 2, padding: 0 }}
                onClick={(e) => {
                  e.stopPropagation()
                  Modal.confirm({
                    title: '确定删除此Logo？',
                    onOk: async () => {
                      try {
                        const res = await deleteLogo(logo.name)
                        if (res.data.success) {
                          message.success('删除成功')
                          loadLogos()
                        } else {
                          message.error(res.data.message || '删除失败')
                        }
                      } catch (error) {
                        message.error('删除失败')
                      }
                    },
                  })
                }}
              />
            </div>
          ))}
          {logos.length === 0 && (
            <div style={{ textAlign: 'center', color: '#999', padding: 40, width: '100%' }}>
              暂无可选的Logo，请先上传
            </div>
          )}
        </div>
      </Modal>
    </div>
  )
}

export default ModelManagement