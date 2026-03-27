import { useState, useEffect } from 'react'
import { Table, Button, Space, Tag, Modal, Form, Input, Select, InputNumber, Upload, message, Popconfirm } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, UploadOutlined, DatabaseOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { listModels, createModel, updateModel, deleteModel, uploadModelLogo, getModelTypes, getModelStatuses, ModelItem } from '../services/api'

const { TextArea } = Input

const ModelManagement: React.FC = () => {
  const [models, setModels] = useState<ModelItem[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingModel, setEditingModel] = useState<ModelItem | null>(null)
  const [form] = Form.useForm()
  const [modelTypes, setModelTypes] = useState<{ value: string; label: string }[]>([])
  const [modelStatuses, setModelStatuses] = useState<{ value: string; label: string }[]>([])
  const [uploadingLogo, setUploadingLogo] = useState(false)

  useEffect(() => {
    loadModels()
    loadOptions()
  }, [])

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
    setModalVisible(true)
  }

  const handleEdit = (record: ModelItem) => {
    setEditingModel(record)
    form.setFieldsValue(record)
    setModalVisible(true)
  }

  const handleDelete = async (id: string) => {
    try {
      const res = await deleteModel(id)
      if (res.data.success) {
        message.success('删除成功')
        loadModels()
      }
    } catch (error) {
      message.error('删除失败')
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      if (editingModel) {
        await updateModel(editingModel.id, values)
        message.success('更新成功')
      } else {
        await createModel(values)
        message.success('创建成功')
      }
      setModalVisible(false)
      loadModels()
    } catch (error) {
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
      width: 100,
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
      width: 150,
      fixed: 'right',
      render: (_, record) => (
        <Space>
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
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <h2 style={{ margin: 0 }}><DatabaseOutlined /> 模型管理</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
          添加模型
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={models}
        rowKey="id"
        loading={loading}
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
        <Form form={form} layout="vertical">
          <Form.Item name="id" label="模型ID" hidden={!editingModel}>
            <Input disabled />
          </Form.Item>

          <Form.Item name="name" label="模型名称" rules={[{ required: true, message: '请输入模型名称' }]}>
            <Input placeholder="如: Qwen2.5-72B-Instruct" />
          </Form.Item>

          <Form.Item name="provider" label="提供商">
            <Input placeholder="如: Qwen, DeepSeek, BEDI" />
          </Form.Item>

          <Form.Item name="model_type" label="模型类型">
            <Select options={modelTypes} placeholder="选择模型类型" />
          </Form.Item>

          <Form.Item name="description" label="描述">
            <TextArea rows={3} placeholder="模型描述信息" />
          </Form.Item>

          <Form.Item name="context_len" label="上下文长度">
            <InputNumber style={{ width: '100%' }} placeholder="如: 32768" min={0} />
          </Form.Item>

          <Space style={{ width: '100%' }} size="large">
            <Form.Item name="input_price" label="输入价格 (元/千token)" style={{ flex: 1 }}>
              <InputNumber style={{ width: '100%' }} placeholder="0.000000" min={0} precision={6} />
            </Form.Item>

            <Form.Item name="output_price" label="输出价格 (元/千token)" style={{ flex: 1 }}>
              <InputNumber style={{ width: '100%' }} placeholder="0.000000" min={0} precision={6} />
            </Form.Item>
          </Space>

          <Form.Item name="status" label="状态">
            <Select options={modelStatuses} placeholder="选择状态" />
          </Form.Item>

          <Form.Item name="sort_order" label="排序">
            <InputNumber style={{ width: '100%' }} placeholder="0" min={0} />
          </Form.Item>

          <Form.Item name="icon_url" label="Logo URL">
            <Space direction="vertical" style={{ width: '100%' }}>
              <Input placeholder="Logo URL，如: /logos/qwen.png" />
              <Upload beforeUpload={handleLogoUpload} showUploadList={false} accept="image/*">
                <Button icon={<UploadOutlined />} loading={uploadingLogo}>
                  上传Logo
                </Button>
              </Upload>
            </Space>
          </Form.Item>

          <Form.Item name="capabilities" label="能力 (JSON)">
            <TextArea rows={2} placeholder='["chat","vision"]' />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default ModelManagement