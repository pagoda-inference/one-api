import { useState, useEffect } from 'react'
import { Table, Button, Space, Tag, Form, Input, Select, Upload, message, Popconfirm, Row, Col, Drawer } from 'antd'
import { PlusOutlined, DeleteOutlined, CloudServerOutlined, UploadOutlined, DownloadOutlined, StopOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { relayApi } from '../services/api'

const { Dragger } = Upload

interface BatchFile {
  id: string
  object: string
  bytes: number
  created_at: number
  filename: string
  purpose: string
  status: string
}

interface BatchJob {
  id: string
  object: string
  endpoint: string
  input_file_id: string
  output_file_id?: string
  error_file_id?: string
  completion_window: string
  status: string
  created_at: number
  in_progress_at?: number
  expires_at: number
  completed_at?: number
  cancelled_at?: number
  failed_at?: number
  metadata?: string
  request_counts?: {
    total: number
    completed: number
    failed: number
  }
}

const BatchInference: React.FC = () => {
  const { t } = useTranslation()
  const [batches, setBatches] = useState<BatchJob[]>([])
  const [files, setFiles] = useState<BatchFile[]>([])
  const [loading, setLoading] = useState(false)
  const [drawerVisible, setDrawerVisible] = useState(false)
  const [uploadingFile, setUploadingFile] = useState(false)
  const [form] = Form.useForm()

  useEffect(() => {
    loadBatches()
    loadFiles()
  }, [])

  const loadBatches = async () => {
    try {
      setLoading(true)
      const res = await relayApi.get('/batches?limit=100')
      if (res.data.data) {
        setBatches(res.data.data)
      }
    } catch (error) {
      console.error('Failed to load batches:', error)
      message.error(t('batch.load_failed'))
    } finally {
      setLoading(false)
    }
  }

  const loadFiles = async () => {
    try {
      const res = await relayApi.get('/files?purpose=batch')
      if (res.data.data) {
        setFiles(res.data.data)
      }
    } catch (error) {
      console.error('Failed to load files:', error)
    }
  }

  const handleFileUpload = async (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('purpose', 'batch')
    try {
      setUploadingFile(true)
      const res = await relayApi.post('/files', formData, {
        headers: { 'Content-Type': 'multipart/form-data' }
      })
      if (res.data.id) {
        message.success(t('batch.file_upload_success'))
        loadFiles()
        form.setFieldsValue({ input_file_id: res.data.id })
      }
    } catch (error) {
      message.error(t('batch.file_upload_failed'))
    } finally {
      setUploadingFile(false)
    }
    return false
  }

  const handleCreate = () => {
    form.resetFields()
    setDrawerVisible(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      const res = await relayApi.post('/batches', {
        input_file_id: values.input_file_id,
        api_key: values.api_key,
        model: values.model,
        completion_window: '24h',
        metadata: values.description || ''
      })
      if (res.data.id) {
        message.success(t('batch.create_success'))
        setDrawerVisible(false)
        loadBatches()
      }
    } catch (error: any) {
      const errMsg = error.response?.data?.error?.message || t('batch.create_failed')
      message.error(errMsg)
    }
  }

  const handleCancel = async (id: string) => {
    try {
      await relayApi.post(`/batches/${id}/cancel`)
      message.success(t('batch.cancel_success'))
      loadBatches()
    } catch (error) {
      message.error(t('batch.cancel_failed'))
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await relayApi.delete(`/files/${id}`)
      message.success(t('batch.delete_success'))
      loadFiles()
    } catch (error) {
      message.error(t('batch.delete_failed'))
    }
  }

  const handleDownloadResults = async (batch: BatchJob) => {
    if (!batch.request_counts?.completed) {
      message.warning(t('batch.no_results'))
      return
    }
    window.open(`/v1/files/${batch.output_file_id}/content`, '_blank')
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'green'
      case 'in_progress': return 'blue'
      case 'validating': return 'orange'
      case 'finalizing': return 'cyan'
      case 'failed': return 'red'
      case 'cancelled': return 'default'
      case 'expired': return 'default'
      default: return 'default'
    }
  }

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'validating': return t('batch.status_validating')
      case 'in_progress': return t('batch.status_in_progress')
      case 'finalizing': return t('batch.status_finalizing')
      case 'completed': return t('batch.status_completed')
      case 'failed': return t('batch.status_failed')
      case 'cancelled': return t('batch.status_cancelled')
      case 'expired': return t('batch.status_expired')
      default: return status
    }
  }

  const batchColumns: ColumnsType<BatchJob> = [
    {
      title: t('batch.id'),
      dataIndex: 'id',
      key: 'id',
      width: 200,
      ellipsis: true,
      render: (v: string) => <code style={{ fontSize: 11 }}>{v}</code>
    },
    {
      title: t('batch.endpoint'),
      dataIndex: 'endpoint',
      key: 'endpoint',
      width: 150,
    },
    {
      title: t('batch.status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (v: string) => <Tag color={getStatusColor(v)}>{getStatusLabel(v)}</Tag>
    },
    {
      title: t('batch.progress'),
      key: 'progress',
      width: 150,
      render: (_, record) => {
        if (!record.request_counts) return '-'
        const { total, completed, failed } = record.request_counts
        return (
          <span>
            {completed || 0} / {total || 0}
            {(failed || 0) > 0 && <span style={{ color: '#ff4d4f', marginLeft: 8 }}>({failed} failed)</span>}
          </span>
        )
      }
    },
    {
      title: t('batch.created_at'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
      render: (v: number) => new Date(v * 1000).toLocaleString()
    },
    {
      title: t('batch.completion_window'),
      dataIndex: 'completion_window',
      key: 'completion_window',
      width: 100,
    },
    {
      title: t('batch.action'),
      key: 'action',
      width: 180,
      render: (_, record) => (
        <Space size="small">
          {record.status === 'completed' && record.output_file_id && (
            <Button type="link" size="small" icon={<DownloadOutlined />} onClick={() => handleDownloadResults(record)}>
              {t('batch.download_results')}
            </Button>
          )}
          {['validating', 'in_progress', 'finalizing'].includes(record.status) && (
            <Popconfirm title={t('batch.confirm_cancel')} onConfirm={() => handleCancel(record.id)}>
              <Button type="link" size="small" danger icon={<StopOutlined />}>
                {t('batch.cancel')}
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  const fileColumns: ColumnsType<BatchFile> = [
    {
      title: t('batch.filename'),
      dataIndex: 'filename',
      key: 'filename',
      ellipsis: true,
    },
    {
      title: t('batch.size'),
      dataIndex: 'bytes',
      key: 'bytes',
      width: 100,
      render: (v: number) => v > 1024 * 1024 ? `${(v / 1024 / 1024).toFixed(1)} MB` : `${(v / 1024).toFixed(1)} KB`
    },
    {
      title: t('batch.created_at'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
      render: (v: number) => new Date(v * 1000).toLocaleString()
    },
    {
      title: t('batch.action'),
      key: 'action',
      width: 80,
      render: (_, record) => (
        <Popconfirm title={t('batch.confirm_delete_file')} onConfirm={() => handleDelete(record.id)}>
          <Button type="link" size="small" danger icon={<DeleteOutlined />}>
            {t('common.delete')}
          </Button>
        </Popconfirm>
      ),
    },
  ]

  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2 style={{ margin: 0 }}><CloudServerOutlined /> {t('batch.batch_inference')}</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
          {t('batch.new_batch')}
        </Button>
      </div>

      <div style={{ marginBottom: 24 }}>
        <h3 style={{ marginBottom: 12 }}>{t('batch.input_files')}</h3>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Dragger
              accept=".jsonl,.json"
              showUploadList={false}
              beforeUpload={handleFileUpload}
              disabled={uploadingFile}
            >
              <p className="ant-upload-drag-icon">
                <UploadOutlined />
              </p>
              <p className="ant-upload-text">{t('batch.upload_file')}</p>
              <p className="ant-upload-hint">{t('batch.upload_hint')}</p>
            </Dragger>
          </Col>
          <Col xs={24} md={12}>
            <Table
              size="small"
              columns={fileColumns}
              dataSource={files}
              rowKey="id"
              pagination={{ pageSize: 5, showSizeChanger: false }}
              loading={loading}
            />
          </Col>
        </Row>
      </div>

      <div>
        <h3 style={{ marginBottom: 12 }}>{t('batch.batch_jobs')}</h3>
        <Table
          columns={batchColumns}
          dataSource={batches}
          rowKey="id"
          loading={loading}
          scroll={{ x: true }}
          pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (total) => t('batch.total_records', { total }) }}
        />
      </div>

      <Drawer
        title={t('batch.new_batch')}
        placement="right"
        width={500}
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
        footer={
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={() => setDrawerVisible(false)}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={handleSubmit}>{t('batch.create')}</Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="input_file_id"
            label={t('batch.input_file')}
            rules={[{ required: true, message: t('batch.select_input_file') }]}
          >
            <Select
              placeholder={t('batch.select_file')}
              options={files.map(f => ({
                value: f.id,
                label: `${f.filename} (${f.bytes} bytes)`
              }))}
            />
          </Form.Item>

          <Form.Item
            name="api_key"
            label={t('batch.api_key')}
            rules={[{ required: true, message: t('batch.api_key_placeholder') }]}
          >
            <Input.Password
              placeholder={t('batch.api_key_placeholder')}
              autoComplete="off"
            />
          </Form.Item>

          <Form.Item
            name="model"
            label={t('batch.model')}
            rules={[{ required: true, message: t('batch.model_placeholder') }]}
          >
            <Select
              placeholder={t('batch.model_placeholder')}
              options={[
                { value: 'bedi/deepseek-r1', label: 'DeepSeek-V3.1' },
                { value: 'bedi/qwen3-235b-a22b', label: 'Qwen3-235B-A22B' },
                { value: 'bedi/qwq-32b', label: 'QwQ-32B' },
              ]}
            />
          </Form.Item>

          <Form.Item
            name="description"
            label={t('batch.description')}
          >
            <Input.TextArea rows={2} placeholder={t('batch.description_placeholder')} />
          </Form.Item>
        </Form>
      </Drawer>
    </div>
  )
}

export default BatchInference
