import { useState, useEffect } from 'react'
import { Table, Card, Input, InputNumber, Switch, Button, message, Space, Tag } from 'antd'
import { useTheme } from '../contexts/ThemeContext'
import { getPlaygroundModels } from '../services/api'
import api from '../services/api'
import { useTranslation } from 'react-i18next'

interface PlaygroundModel {
  id: string
  name: string
  model_type: string
  is_vl: boolean
  is_reasoning: boolean
  max_tokens: number
  temperature: number
  min_p: number
  top_p: number
  top_k: number
  frequency_penalty: number
  presence_penalty: number
  repetition_penalty: number
  system_prompt: string
  enable_thinking: boolean
  thinking_budget: number
  enable_temperature: boolean
  enable_min_p: boolean
  enable_top_p: boolean
  enable_top_k: boolean
  enable_frequency_penalty: boolean
  enable_presence_penalty: boolean
  enable_repetition_penalty: boolean
  enable_system_prompt: boolean
  enable_thinking_budget: boolean
}

const PlaygroundConfig: React.FC = () => {
  const { appTheme } = useTheme()
  const { t } = useTranslation()
  const [models, setModels] = useState<PlaygroundModel[]>([])
  const [loading, setLoading] = useState(true)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editForm, setEditForm] = useState<Partial<PlaygroundModel>>({})

  useEffect(() => {
    loadModels()
  }, [])

  const loadModels = async () => {
    try {
      const res = await getPlaygroundModels()
      setModels(res.data.data || [])
    } catch (error) {
      console.error('Failed to load models:', error)
      message.error(t('playgroundConfig.load_failed'))
    } finally {
      setLoading(false)
    }
  }

  const handleEdit = (model: PlaygroundModel) => {
    setEditingId(model.id)
    setEditForm({
      max_tokens: model.max_tokens,
      temperature: model.temperature,
      min_p: model.min_p,
      top_p: model.top_p,
      top_k: model.top_k,
      frequency_penalty: model.frequency_penalty,
      presence_penalty: model.presence_penalty,
      repetition_penalty: model.repetition_penalty,
      system_prompt: model.system_prompt,
      enable_thinking: model.enable_thinking,
      thinking_budget: model.thinking_budget,
      enable_temperature: model.enable_temperature,
      enable_min_p: model.enable_min_p,
      enable_top_p: model.enable_top_p,
      enable_top_k: model.enable_top_k,
      enable_frequency_penalty: model.enable_frequency_penalty,
      enable_presence_penalty: model.enable_presence_penalty,
      enable_repetition_penalty: model.enable_repetition_penalty,
      enable_system_prompt: model.enable_system_prompt,
      enable_thinking_budget: model.enable_thinking_budget,
      is_vl: model.is_vl,
      is_reasoning: model.is_reasoning,
    })
  }

  const handleCancel = () => {
    setEditingId(null)
    setEditForm({})
  }

  const handleSave = async (modelId: string) => {
    try {
      await api.put(`/admin/market/playground/models/${encodeURIComponent(modelId)}`, editForm)
      message.success(t('playgroundConfig.save_success'))
      setEditingId(null)
      setEditForm({})
      loadModels()
    } catch (error) {
      console.error('Failed to save:', error)
      message.error(t('playgroundConfig.save_failed'))
    }
  }

  const columns = [
    {
      title: t('playgroundConfig.model_id'),
      dataIndex: 'id',
      key: 'id',
      width: 200,
      render: (id: string) => <code style={{ fontSize: 12 }}>{id}</code>
    },
    {
      title: t('playgroundConfig.name'),
      dataIndex: 'name',
      key: 'name',
      width: 120,
    },
    {
      title: t('playgroundConfig.type'),
      dataIndex: 'model_type',
      key: 'model_type',
      width: 80,
      render: (type: string) => <Tag>{type}</Tag>
    },
    {
      title: t('playgroundConfig.capabilities'),
      key: 'capabilities',
      width: 220,
      render: (_: any, record: PlaygroundModel) => {
        if (editingId === record.id) {
          return (
            <Space direction="vertical" size={4}>
              <Space size={6}>
                <Tag color="blue">{t('playgroundConfig.vl')}</Tag>
                <Switch
                  size="small"
                  checked={!!editForm.is_vl}
                  onChange={(checked) => setEditForm({ ...editForm, is_vl: checked })}
                />
              </Space>
              <Space size={6}>
                <Tag color="purple">{t('playgroundConfig.reasoning')}</Tag>
                <Switch
                  size="small"
                  checked={!!editForm.is_reasoning}
                  onChange={(checked) => setEditForm({ ...editForm, is_reasoning: checked })}
                />
              </Space>
            </Space>
          )
        }
        return (
          <Space>
            {record.is_vl && <Tag color="blue">{t('playgroundConfig.vl')}</Tag>}
            {record.is_reasoning && <Tag color="purple">{t('playgroundConfig.reasoning')}</Tag>}
          </Space>
        )
      }
    },
    {
      title: t('playgroundConfig.max_tokens'),
      dataIndex: 'max_tokens',
      key: 'max_tokens',
      width: 120,
      render: (value: number, record: PlaygroundModel) => {
        if (editingId === record.id) {
          return (
            <InputNumber
              value={editForm.max_tokens}
              min={256}
              max={32768}
              step={256}
              onChange={(v) => setEditForm({ ...editForm, max_tokens: v ?? undefined })}
              style={{ width: 100 }}
            />
          )
        }
        return value
      }
    },
    {
      title: t('playgroundConfig.temperature'),
      dataIndex: 'temperature',
      key: 'temperature',
      width: 130,
      render: (value: number, record: PlaygroundModel) => {
        if (editingId === record.id) {
          return (
            <Space direction="vertical" size={4}>
              <Switch
                size="small"
                checked={!!editForm.enable_temperature}
                onChange={(checked) => setEditForm({ ...editForm, enable_temperature: checked })}
              />
              <InputNumber
                value={editForm.temperature}
                min={0}
                max={2}
                step={0.1}
                precision={1}
                disabled={!editForm.enable_temperature}
                onChange={(v) => setEditForm({ ...editForm, temperature: v ?? undefined })}
                style={{ width: 90 }}
              />
            </Space>
          )
        }
        return value.toFixed(1)
      }
    },
    {
      title: t('playgroundConfig.min_p'),
      dataIndex: 'min_p',
      key: 'min_p',
      width: 120,
      render: (value: number, record: PlaygroundModel) => {
        if (editingId === record.id) {
          return (
            <Space direction="vertical" size={4}>
              <Switch
                size="small"
                checked={!!editForm.enable_min_p}
                onChange={(checked) => setEditForm({ ...editForm, enable_min_p: checked })}
              />
              <InputNumber
                value={editForm.min_p}
                min={0}
                max={1}
                step={0.01}
                precision={2}
                disabled={!editForm.enable_min_p}
                formatter={(v) => {
                  if (v === undefined || v === null) return ''
                  const n = Number(v)
                  return Number.isNaN(n) ? '' : n.toFixed(2)
                }}
                parser={(v) => {
                  if (!v) return 0
                  const n = Number(String(v).replace(/[^\d.-]/g, ''))
                  return Number.isNaN(n) ? 0 : n
                }}
                onChange={(v) => setEditForm({ ...editForm, min_p: v ?? undefined })}
                style={{ width: 90 }}
              />
            </Space>
          )
        }
        return Number(value || 0).toFixed(2)
      }
    },
    {
      title: t('playgroundConfig.top_p'),
      dataIndex: 'top_p',
      key: 'top_p',
      width: 120,
      render: (value: number, record: PlaygroundModel) => {
        if (editingId === record.id) {
          return (
            <Space direction="vertical" size={4}>
              <Switch
                size="small"
                checked={!!editForm.enable_top_p}
                onChange={(checked) => setEditForm({ ...editForm, enable_top_p: checked })}
              />
              <InputNumber
                value={editForm.top_p}
                min={0}
                max={1}
                step={0.05}
                precision={2}
                disabled={!editForm.enable_top_p}
                onChange={(v) => setEditForm({ ...editForm, top_p: v ?? undefined })}
                style={{ width: 90 }}
              />
            </Space>
          )
        }
        return value.toFixed(2)
      }
    },
    {
      title: t('playgroundConfig.top_k'),
      dataIndex: 'top_k',
      key: 'top_k',
      width: 120,
      render: (value: number, record: PlaygroundModel) => {
        if (editingId === record.id) {
          return (
            <Space direction="vertical" size={4}>
              <Switch
                size="small"
                checked={!!editForm.enable_top_k}
                onChange={(checked) => setEditForm({ ...editForm, enable_top_k: checked })}
              />
              <InputNumber
                value={editForm.top_k}
                min={1}
                max={100}
                disabled={!editForm.enable_top_k}
                onChange={(v) => setEditForm({ ...editForm, top_k: v ?? undefined })}
                style={{ width: 90 }}
              />
            </Space>
          )
        }
        return value
      }
    },
    {
      title: t('playgroundConfig.frequency_penalty'),
      dataIndex: 'frequency_penalty',
      key: 'frequency_penalty',
      width: 130,
      render: (value: number, record: PlaygroundModel) => {
        if (editingId === record.id) {
          return (
            <Space direction="vertical" size={4}>
              <Switch
                size="small"
                checked={!!editForm.enable_frequency_penalty}
                onChange={(checked) => setEditForm({ ...editForm, enable_frequency_penalty: checked })}
              />
              <InputNumber
                value={editForm.frequency_penalty}
                min={-2}
                max={2}
                step={0.1}
                precision={1}
                disabled={!editForm.enable_frequency_penalty}
                onChange={(v) => setEditForm({ ...editForm, frequency_penalty: v ?? undefined })}
                style={{ width: 90 }}
              />
            </Space>
          )
        }
        return Number(value || 0).toFixed(1)
      }
    },
    {
      title: t('playgroundConfig.presence_penalty'),
      dataIndex: 'presence_penalty',
      key: 'presence_penalty',
      width: 130,
      render: (value: number, record: PlaygroundModel) => {
        if (editingId === record.id) {
          return (
            <Space direction="vertical" size={4}>
              <Switch
                size="small"
                checked={!!editForm.enable_presence_penalty}
                onChange={(checked) => setEditForm({ ...editForm, enable_presence_penalty: checked })}
              />
              <InputNumber
                value={editForm.presence_penalty}
                min={-2}
                max={2}
                step={0.1}
                precision={1}
                disabled={!editForm.enable_presence_penalty}
                onChange={(v) => setEditForm({ ...editForm, presence_penalty: v ?? undefined })}
                style={{ width: 90 }}
              />
            </Space>
          )
        }
        return Number(value || 0).toFixed(1)
      }
    },
    {
      title: t('playgroundConfig.repetition_penalty'),
      dataIndex: 'repetition_penalty',
      key: 'repetition_penalty',
      width: 130,
      render: (value: number, record: PlaygroundModel) => {
        if (editingId === record.id) {
          return (
            <Space direction="vertical" size={4}>
              <Switch
                size="small"
                checked={!!editForm.enable_repetition_penalty}
                onChange={(checked) => setEditForm({ ...editForm, enable_repetition_penalty: checked })}
              />
              <InputNumber
                value={editForm.repetition_penalty}
                min={0}
                max={2}
                step={0.1}
                precision={1}
                disabled={!editForm.enable_repetition_penalty}
                onChange={(v) => setEditForm({ ...editForm, repetition_penalty: v ?? undefined })}
                style={{ width: 90 }}
              />
            </Space>
          )
        }
        return Number(value || 1).toFixed(1)
      }
    },
    {
      title: t('playgroundConfig.enable_thinking'),
      dataIndex: 'enable_thinking',
      key: 'enable_thinking',
      width: 100,
      render: (value: boolean, record: PlaygroundModel) => {
        if (editingId === record.id) {
          return (
            <Switch
              checked={editForm.enable_thinking}
              onChange={(checked) => setEditForm({ ...editForm, enable_thinking: checked })}
            />
          )
        }
        return <Switch checked={value} disabled />
      }
    },
    {
      title: t('playgroundConfig.thinking_budget'),
      dataIndex: 'thinking_budget',
      key: 'thinking_budget',
      width: 120,
      render: (value: number, record: PlaygroundModel) => {
        if (editingId === record.id) {
          return (
            <Space direction="vertical" size={4}>
              <Switch
                size="small"
                checked={!!editForm.enable_thinking_budget}
                onChange={(checked) => setEditForm({ ...editForm, enable_thinking_budget: checked })}
              />
              <InputNumber
                value={editForm.thinking_budget}
                min={0}
                max={16000}
                disabled={!editForm.enable_thinking_budget}
                onChange={(v) => setEditForm({ ...editForm, thinking_budget: v ?? undefined })}
                style={{ width: 90 }}
              />
            </Space>
          )
        }
        return value
      }
    },
    {
      title: t('playgroundConfig.system_prompt'),
      dataIndex: 'system_prompt',
      key: 'system_prompt',
      render: (value: string, record: PlaygroundModel) => {
        if (editingId === record.id) {
          return (
            <Space direction="vertical" size={4}>
              <Switch
                size="small"
                checked={!!editForm.enable_system_prompt}
                onChange={(checked) => setEditForm({ ...editForm, enable_system_prompt: checked })}
              />
              <Input.TextArea
                value={editForm.system_prompt}
                onChange={(e) => setEditForm({ ...editForm, system_prompt: e.target.value })}
                rows={2}
                disabled={!editForm.enable_system_prompt}
                style={{ width: 200 }}
                placeholder={t('playgroundConfig.system_prompt_placeholder')}
              />
            </Space>
          )
        }
        return (
          <span style={{
            fontSize: 11,
            color: value ? appTheme.textSecondary : '#999',
            maxWidth: 150,
            display: 'inline-block',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap'
          }}>
            {value || t('playgroundConfig.none')}
          </span>
        )
      }
    },
    {
      title: t('common.action'),
      key: 'action',
      width: 150,
      render: (_: any, record: PlaygroundModel) => {
        if (editingId === record.id) {
          return (
            <Space>
              <Button type="primary" size="small" onClick={() => handleSave(record.id)}>
                {t('common.save')}
              </Button>
              <Button size="small" onClick={handleCancel}>
                {t('common.cancel')}
              </Button>
            </Space>
          )
        }
        return (
          <Button type="link" size="small" onClick={() => handleEdit(record)}>
            {t('common.edit')}
          </Button>
        )
      }
    }
  ]

  return (
    <div>
      <Card
        title={t('playgroundConfig.title')}
        style={{ borderRadius: 12 }}
        styles={{ body: { padding: 0 } }}
      >
        <Table
          dataSource={models}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={false}
          scroll={{ x: 1400 }}
          style={{
            borderRadius: 12,
            overflow: 'hidden'
          }}
        />
      </Card>

      <Card style={{ marginTop: 16, borderRadius: 12 }}>
        <div style={{ color: appTheme.textSecondary }}>
          <h4>{t('playgroundConfig.notes')}</h4>
          <ul style={{ paddingLeft: 20 }}>
            <li>{t('playgroundConfig.note_max_tokens')}</li>
            <li>{t('playgroundConfig.note_temperature')}</li>
            <li>{t('playgroundConfig.note_min_p')}</li>
            <li>{t('playgroundConfig.note_top_p')}</li>
            <li>{t('playgroundConfig.note_top_k')}</li>
            <li>{t('playgroundConfig.note_frequency_penalty')}</li>
            <li>{t('playgroundConfig.note_presence_penalty')}</li>
            <li>{t('playgroundConfig.note_repetition_penalty')}</li>
            <li>{t('playgroundConfig.note_enable_thinking')}</li>
            <li>{t('playgroundConfig.note_thinking_budget')}</li>
            <li>{t('playgroundConfig.note_system_prompt')}</li>
          </ul>
          <p>{t('playgroundConfig.note_apply')}</p>
        </div>
      </Card>
    </div>
  )
}

export default PlaygroundConfig
