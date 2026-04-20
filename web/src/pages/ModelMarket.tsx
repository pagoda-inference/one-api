import { useState, useEffect } from 'react'
import { Row, Col, Card, Input, Tag, Button, Drawer, Descriptions, Statistic, message, Tabs, Badge, Segmented } from 'antd'
import {
  SearchOutlined, RobotOutlined, SyncOutlined,
  ExperimentOutlined, AppstoreOutlined, BarsOutlined,
  ThunderboltOutlined, PictureOutlined, AudioOutlined, FileTextOutlined,
  LockOutlined, CopyOutlined
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useTheme } from '../contexts/ThemeContext'
import { getMarketModels, getMarketStats, getMarketGroups, getModelTrial, startModelTrial, calculatePrice, Model, ModelGroup, MarketStats } from '../services/api'

const { Search } = Input

// 根据模型类型生成示例代码
const getModelExample = (modelId: string, modelType: string): string => {
  const apiBase = 'https://baotaAI.bedicloud.net/v1'

  // MinerU special case
  if (modelId.toLowerCase().includes('mineru')) {
    return `curl -X POST ${apiBase}/file_parse \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -F "files=@/path/to/document.pdf" \\
  -F "model=${modelId}"`
  }

  switch (modelType) {
    case 'embedding':
      return `import openai
openai.api_key = "your-api-key"
openai.api_base = "${apiBase}"

response = openai.Embedding.create(
    model="${modelId}",
    input="Hello world"
)
print(response['data'][0]['embedding'])`

    case 'reranker':
      return `curl -X POST ${apiBase}/rerank \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${modelId}",
    "query": "Hello world",
    "documents": ["Hello world", "Goodbye world"]
  }'`

    case 'image':
      return `import openai
openai.api_key = "your-api-key"
openai.api_base = "${apiBase}"

response = openai.Image.create(
    model="${modelId}",
    prompt="A cute puppy"
)
print(response['data'][0]['url'])`

    case 'audio':
      return `import openai
openai.api_key = "your-api-key"
openai.api_base = "${apiBase}"

with open("audio.mp3", "rb") as f:
    response = openai.Audio.transcribe(
        model="${modelId}",
        file=f
    )
print(response['text'])`

    default: // chat, vlm, ocr, video, other
      return `import openai
openai.api_key = "your-api-key"
openai.api_base = "${apiBase}"

response = openai.ChatCompletion.create(
    model="${modelId}",
    messages=[{"role": "user", "content": "Hello!"}]
)
print(response['choices'][0]['message']['content'])`
  }
}

const ModelMarket: React.FC = () => {
  const { t } = useTranslation()
  const { themeMode } = useTheme()
  const [models, setModels] = useState<Model[]>([])
  const [groups, setGroups] = useState<ModelGroup[]>([])
  const [stats, setStats] = useState<MarketStats | null>(null)
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<string>('')
  const [groupFilter, setGroupFilter] = useState<string | null>(null)
  const [capabilityFilter, setCapabilityFilter] = useState<string[]>([])
  const [contextFilter, setContextFilter] = useState<string>('')
  const [trialOnly, setTrialOnly] = useState(false)
  const [filtersCollapsed, setFiltersCollapsed] = useState(false)
  const [selectedModel, setSelectedModel] = useState<Model | null>(null)
  const [drawerVisible, setDrawerVisible] = useState(false)
  const [priceCalc, setPriceCalc] = useState<{ prompt_tokens: number; completion_tokens: number; quota_cost: number } | null>(null)
  const [trialInfo, setTrialInfo] = useState<any>(null)
  const [trialLoading, setTrialLoading] = useState(false)
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [activeTab, setActiveTab] = useState<string>('all')
  useEffect(() => {
    loadData()
  }, [typeFilter, groupFilter, capabilityFilter, contextFilter, trialOnly])

  const loadData = async () => {
    try {
      const [modelsRes, groupsRes, statsRes] = await Promise.all([
        getMarketModels({ type: typeFilter, limit: 100 }),
        getMarketGroups(),
        getMarketStats()
      ])

      let allModels = modelsRes.data.data.models || []

      // Apply group filter (filter by provider code, case-insensitive)
      if (groupFilter) {
        allModels = allModels.filter((m: Model) => m.provider.toLowerCase() === groupFilter.toLowerCase())
      }

      // Apply capability filter (AND logic - must have all selected capabilities)
      if (capabilityFilter.length > 0) {
        allModels = allModels.filter((m: Model) => {
          try {
            const caps = JSON.parse(m.capabilities || '[]') as string[]
            return capabilityFilter.every(cap => caps.includes(cap))
          } catch {
            return false
          }
        })
      }

      // Apply context length filter
      if (contextFilter) {
        const minContext = parseInt(contextFilter) * 1024
        allModels = allModels.filter((m: Model) => m.context_len >= minContext)
      }

      // Apply trial only filter
      if (trialOnly) {
        allModels = allModels.filter((m: Model) => m.is_trial)
      }

      // Apply search filter
      if (search) {
        const keyword = search.toLowerCase()
        allModels = allModels.filter((m: Model) =>
          m.name.toLowerCase().includes(keyword) ||
          m.id.toLowerCase().includes(keyword) ||
          m.provider.toLowerCase().includes(keyword)
        )
      }

      // Filter out disabled models (only show active and maintenance)
      allModels = allModels.filter((m: Model) => m.status !== 'disabled')

      setModels(allModels)
      setGroups(groupsRes.data.data || [])
      setStats(statsRes.data.data)
    } catch (error) {
      console.error('Failed to load models:', error)
    }
  }

  const handleSearch = (value: string) => {
    setSearch(value)
    loadData()
  }

  const showModelDetail = async (model: Model) => {
    setSelectedModel(model)
    setDrawerVisible(true)
    setPriceCalc(null)
    setTrialInfo(null)

    try {
      const res = await getModelTrial(model.id)
      setTrialInfo(res.data.data)
    } catch (error) {
      console.error('Failed to load trial info:', error)
    }
  }

  const handleStartTrial = async () => {
    if (!selectedModel) return
    setTrialLoading(true)
    try {
      await startModelTrial(selectedModel.id)
      message.success(t('modelMarket.trial_enabled'))
      const res = await getModelTrial(selectedModel.id)
      setTrialInfo(res.data.data)
    } catch (error) {
      message.error(t('modelMarket.trial_failed'))
    } finally {
      setTrialLoading(false)
    }
  }

  const calcPrice = async (promptTokens: number, completionTokens: number) => {
    if (!selectedModel) return
    try {
      const res = await calculatePrice({
        model_id: selectedModel.id,
        prompt_tokens: promptTokens,
        completion_tokens: completionTokens
      })
      setPriceCalc(res.data.data)
    } catch (error) {
      console.error('Failed to calculate price:', error)
    }
  }

  const getModelIcon = (modelType: string) => {
    switch (modelType) {
      case 'chat': return <RobotOutlined />
      case 'embedding': return <FileTextOutlined />
      case 'image': return <PictureOutlined />
      case 'audio': return <AudioOutlined />
      case 'vlm': return <ExperimentOutlined />
      default: return <RobotOutlined />
    }
  }

  const getModelLogo = (model: Model) => {
    // 使用模型配置的 icon_url（用户上传的Logo）
    if (model.icon_url) {
      return model.icon_url.startsWith('http') ? model.icon_url : model.icon_url
    }
    return null
  }

  // Low saturation candy colors for capability tags (n1n style)
  const getCapabilityTagStyle = (cap: string): { bg: string; color: string } => {
    const styles: Record<string, { bg: string; color: string }> = {
      'chat': { bg: '#e8f4ff', color: '#1890ff' },
      'vision': { bg: '#fff7e6', color: '#fa8c16' },
      'function_call': { bg: '#f6ffed', color: '#52c41a' },
      'embedding': { bg: '#f9f0ff', color: '#722ed1' },
      'reranker': { bg: '#fff1f0', color: '#ff4d4f' },
      'ocr': { bg: '#f0f5ff', color: '#597ef7' },
      'reasoning': { bg: '#fff0f5', color: '#eb2f96' },
      'audio': { bg: '#e6fffb', color: '#13c2c2' }
    }
    return styles[cap] || { bg: '#f5f5f5', color: '#8c8c8c' }
  }

  const getCapabilityTags = (capabilities: string) => {
    try {
      const caps = JSON.parse(capabilities) as string[]
      return caps.map(cap => {
        const style = getCapabilityTagStyle(cap)
        return (
          <Tag
            key={cap}
            style={{
              background: style.bg,
              color: style.color,
              border: 'none',
              borderRadius: 4,
              fontSize: 11
            }}
          >
            {cap}
          </Tag>
        )
      })
    } catch {
      return null
    }
  }

  const formatPrice = (price: number) => {
    if (price === 0) return t('modelMarket.free')
    return `¥${price.toFixed(4)}/1K`
  }

  const formatQuota = (quota: number) => {
    if (quota >= 1000000) return (quota / 1000000).toFixed(1) + 'M'
    if (quota >= 1000) return (quota / 1000).toFixed(0) + 'K'
    return quota.toString()
  }

  const getSLAColor = (sla: string) => {
    switch (sla) {
      case 'enterprise': return '#ff4d4f'
      case 'premium': return '#faad14'
      default: return '#52c41a'
    }
  }

  // Model type options with icons (ordered by category)
  const typeOptions = [
    { value: '', label: t('modelMarket.all_types') },
    { value: 'chat', label: t('modelMarket.chat'), icon: <RobotOutlined /> },
    { value: 'vlm', label: t('modelMarket.vlm'), icon: <ExperimentOutlined /> },
    { value: 'reranker', label: t('modelMarket.reranker'), icon: <SyncOutlined /> },
    { value: 'embedding', label: t('modelMarket.embedding'), icon: <FileTextOutlined /> },
    { value: 'ocr', label: 'OCR', icon: <FileTextOutlined /> },
    { value: 'image', label: t('modelMarket.image'), icon: <PictureOutlined /> },
    { value: 'audio', label: t('modelMarket.audio'), icon: <AudioOutlined /> }
  ]

  // Stat card helper
  const StatCard = ({ title, value, suffix, icon, color }: any) => (
    <Card
      style={{
        borderRadius: 12,
        border: 'none',
        boxShadow: '0 2px 8px var(--shadow)',
        background: 'var(--bg-card)'
      }}
      styles={{ body: { padding: '16px 20px', background: 'var(--bg-card)' } }}
    >
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div>
          <div style={{ color: 'var(--text-secondary)', fontSize: 13, marginBottom: 4 }}>{title}</div>
          <div style={{ fontSize: 22, fontWeight: 700, color: 'var(--text-primary)' }}>
            {value}
            <span style={{ fontSize: 13, color: 'var(--text-secondary)', marginLeft: 4 }}>{suffix}</span>
          </div>
        </div>
        <div style={{
          width: 40,
          height: 40,
          borderRadius: 10,
          background: `${color}15`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center'
        }}>
          <div style={{
            width: 28,
            height: 28,
            borderRadius: 8,
            background: color,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: '#fff',
            fontSize: 14
          }}>
            {icon}
          </div>
        </div>
      </div>
    </Card>
  )

  // Model Card Component
  const ModelCard = ({ model }: { model: Model }) => {
    const isMaintenance = model.status === 'maintenance'
    return (
    <Card
      hoverable={!isMaintenance}
      onClick={() => !isMaintenance && showModelDetail(model)}
      className="model-card"
      style={{
        borderRadius: 12,
        border: isMaintenance ? '2px solid #faad14' : 'none',
        boxShadow: '0 2px 8px var(--shadow)',
        overflow: 'hidden',
        transition: 'all 0.3s',
        background: isMaintenance ? 'var(--bg-card)' : 'var(--bg-card)',
        opacity: isMaintenance ? 0.75 : 1,
        cursor: isMaintenance ? 'not-allowed' : 'pointer'
      }}
      styles={{ body: { padding: 16, background: 'var(--bg-card)' } }}
    >
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
        {getModelLogo(model) ? (
          <img
            src={getModelLogo(model)!}
            alt={model.name}
            style={{
              width: 44,
              height: 44,
              borderRadius: 10,
              objectFit: 'contain',
              flexShrink: 0,
              background: themeMode === 'dark' ? 'var(--bg-secondary)' : '#f5f5f5'
            }}
            onError={(e) => {
              // 如果图片加载失败，显示图标
              const target = e.target as HTMLImageElement
              target.style.display = 'none'
              target.nextElementSibling?.classList.remove('hidden')
            }}
          />
        ) : null}
        <div style={{
          width: 44,
          height: 44,
          borderRadius: 10,
          background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: '#fff',
          fontSize: 18,
          flexShrink: 0,
          ...(getModelLogo(model) ? { display: 'none' } : {})
        }} className={getModelLogo(model) ? 'hidden' : ''}>
          {getModelIcon(model.model_type)}
        </div>

        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
            <span style={{ fontWeight: 600, fontSize: 14, color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: model.sla === 'enterprise' ? 140 : '100%' }}>{model.name}</span>
            {model.is_trial && (
              <Tag
                style={{
                  background: '#f6ffed',
                  color: '#52c41a',
                  border: 'none',
                  borderRadius: 4,
                  fontSize: 10,
                  padding: '0 4px'
                }}
              >
                {t('modelMarket.trial')}
              </Tag>
            )}
            {model.sla === 'enterprise' && (
              <Tag
                style={{
                  background: '#ff4d4f15',
                  color: '#ff4d4f',
                  border: 'none',
                  borderRadius: 4,
                  fontSize: 10,
                  padding: '0 4px'
                }}
              >
                {t('modelMarket.enterprise_exclusive')}
              </Tag>
            )}
            {model.visible_to_teams && model.visible_to_teams !== '' && (
              <Tag
                style={{
                  background: '#722ed115',
                  color: '#722ed1',
                  border: 'none',
                  borderRadius: 4,
                  fontSize: 10,
                  padding: '0 4px'
                }}
                icon={<LockOutlined />}
              >
                {t('modelMarket.team_private')}
              </Tag>
            )}
            {isMaintenance && (
              <Tag
                style={{
                  background: '#faad1405',
                  color: '#faad14',
                  border: '1px solid #faad14',
                  borderRadius: 4,
                  fontSize: 10,
                  padding: '0 4px'
                }}
              >
                {t('modelMarket.maintenance')}
              </Tag>
            )}
          </div>

          <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 8 }}>
            {model.provider}
          </div>

          <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 8, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {model.description || t('modelMarket.no_description')}
          </div>

          <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', marginBottom: 8 }}>
            {getCapabilityTags(model.capabilities)?.slice(0, 3)}
          </div>

          <div style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            paddingTop: 8,
            borderTop: '1px solid var(--border-color)'
          }}>
            <div style={{ display: 'flex', gap: 8 }}>
              <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{t('modelMarket.input')}</span>
              <span style={{ fontSize: 12, fontWeight: 500, color: '#667eea' }}>
                {formatPrice(model.input_price)}
              </span>
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{t('modelMarket.output')}</span>
              <span style={{ fontSize: 12, fontWeight: 500, color: '#764ba2' }}>
                {formatPrice(model.output_price)}
              </span>
            </div>
          </div>
        </div>
      </div>
    </Card>
  )
  }

  // List View
  const ModelListItem = ({ model }: { model: Model }) => {
    const isMaintenance = model.status === 'maintenance'
    return (
    <Card
      hoverable={!isMaintenance}
      onClick={() => !isMaintenance && showModelDetail(model)}
      style={{
        borderRadius: 12,
        border: isMaintenance ? '2px solid #faad14' : 'none',
        boxShadow: '0 2px 8px var(--shadow)',
        marginBottom: 8,
        background: 'var(--bg-card)',
        opacity: isMaintenance ? 0.75 : 1,
        cursor: isMaintenance ? 'not-allowed' : 'pointer'
      }}
      styles={{ body: { padding: '12px 16px', background: 'var(--bg-card)' } }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        {getModelLogo(model) ? (
          <img
            src={getModelLogo(model)!}
            alt={model.name}
            style={{
              width: 36,
              height: 36,
              borderRadius: 8,
              objectFit: 'contain',
              background: themeMode === 'dark' ? 'var(--bg-secondary)' : '#f5f5f5'
            }}
            onError={(e) => {
              const target = e.target as HTMLImageElement
              target.style.display = 'none'
              target.nextElementSibling?.classList.remove('hidden')
            }}
          />
        ) : null}
        <div style={{
          width: 36,
          height: 36,
          borderRadius: 8,
          background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: '#fff',
          fontSize: 16,
          ...(getModelLogo(model) ? { display: 'none' } : {})
        }} className={getModelLogo(model) ? 'hidden' : ''}>
          {getModelIcon(model.model_type)}
        </div>

        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ fontWeight: 600, fontSize: 14, color: 'var(--text-primary)' }}>{model.name}</span>
            {model.is_trial && (
              <Tag style={{ background: '#f6ffed', color: '#52c41a', border: 'none', borderRadius: 4, fontSize: 10 }}>{t('modelMarket.trial')}</Tag>
            )}
            {isMaintenance && (
              <Tag style={{ background: '#faad1405', color: '#faad14', border: '1px solid #faad14', borderRadius: 4, fontSize: 10 }}>
                {t('modelMarket.maintenance')}
              </Tag>
            )}
            <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{model.provider}</span>
          </div>
        </div>

        <div style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{t('modelMarket.input')}</div>
            <div style={{ fontSize: 13, fontWeight: 500, color: '#667eea' }}>{formatPrice(model.input_price)}</div>
          </div>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{t('modelMarket.output')}</div>
            <div style={{ fontSize: 13, fontWeight: 500, color: '#764ba2' }}>{formatPrice(model.output_price)}</div>
          </div>
        </div>
      </div>
    </Card>
  )
  }

  return (
    <div>
      {/* Stats Row */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={8}>
          <StatCard
            title={t('modelMarket.available_models')}
            value={stats?.total_models || 0}
            suffix=""
            icon={<RobotOutlined />}
            color="#667eea"
          />
        </Col>
        <Col xs={24} sm={8}>
          <StatCard
            title={t('modelMarket.provider_count')}
            value={stats?.total_groups || 0}
            suffix=""
            icon={<BarsOutlined />}
            color="#52c41a"
          />
        </Col>
        <Col xs={24} sm={8}>
          <StatCard
            title={t('modelMarket.support_trial')}
            value={stats?.trial_models || 0}
            suffix=""
            icon={<ThunderboltOutlined />}
            color="#faad14"
          />
        </Col>
      </Row>

      {/* Filters */}
      <Card
        style={{ marginBottom: 16, borderRadius: 12, border: 'none', background: 'var(--bg-card)' }}
        styles={{ body: { padding: '12px 20px', background: 'var(--bg-card)' } }}
      >
        {/* Search Row */}
        <Row gutter={[16, 12]} align="middle" style={{ marginBottom: filtersCollapsed ? 0 : 12 }}>
          <Col xs={24} sm={12} md={8}>
            <Search
              placeholder={t('modelMarket.search_placeholder')}
              allowClear
              onSearch={handleSearch}
              prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
              style={{ width: '100%' }}
            />
          </Col>
          <Col xs={24} md={16} style={{ textAlign: 'right', display: 'flex', justifyContent: 'flex-end', gap: 12 }}>
            <Segmented
              value={viewMode}
              onChange={(v) => setViewMode(v as 'grid' | 'list')}
              options={[
                { value: 'grid', icon: <AppstoreOutlined /> },
                { value: 'list', icon: <BarsOutlined /> }
              ]}
            />
            <Button icon={<SyncOutlined />} onClick={loadData}>{t('common.refresh')}</Button>
            <Button
              icon={filtersCollapsed ? <ThunderboltOutlined /> : <BarsOutlined />}
              onClick={() => setFiltersCollapsed(!filtersCollapsed)}
              title={filtersCollapsed ? t('modelMarket.expand_filters') : t('modelMarket.collapse_filters')}
            />
          </Col>
        </Row>

        {!filtersCollapsed && (
          <>
            {/* Type Filter Tags */}
        <div style={{ marginBottom: 10 }}>
          <span style={{ color: 'var(--text-secondary)', fontSize: 12, marginRight: 8 }}>{t('modelMarket.filter_by_type')}:</span>
          {typeOptions.map(opt => (
            <Tag
              key={opt.value}
              className="filter-tag"
              style={{
                cursor: 'pointer',
                borderRadius: 16,
                padding: '2px 12px',
                border: typeFilter === opt.value ? '1px solid #667eea' : '1px solid var(--border-color)',
                background: typeFilter === opt.value ? '#667eea15' : 'transparent',
                color: typeFilter === opt.value ? '#667eea' : 'var(--text-secondary)',
                fontSize: 12
              }}
              onClick={() => setTypeFilter(opt.value)}
            >
              {opt.label}
            </Tag>
          ))}
        </div>

        {/* Provider Filter Tags */}
        <div style={{ marginBottom: 10 }}>
          <span style={{ color: 'var(--text-secondary)', fontSize: 12, marginRight: 8 }}>{t('modelMarket.filter_by_provider')}:</span>
          <Tag
            className="filter-tag"
            style={{
              cursor: 'pointer',
              borderRadius: 16,
              padding: '2px 12px',
              border: !groupFilter ? '1px solid #52c41a' : '1px solid var(--border-color)',
              background: !groupFilter ? '#52c41a15' : 'transparent',
              color: !groupFilter ? '#52c41a' : 'var(--text-secondary)',
              fontSize: 12
            }}
            onClick={() => setGroupFilter(null)}
          >
            {t('modelMarket.all_types')}
          </Tag>
          {groups.map(g => (
            <Tag
              key={g.code}
              className="filter-tag"
              style={{
                cursor: 'pointer',
                borderRadius: 16,
                padding: '2px 12px',
                border: groupFilter === g.code ? '1px solid #52c41a' : '1px solid var(--border-color)',
                background: groupFilter === g.code ? '#52c41a15' : 'transparent',
                color: groupFilter === g.code ? '#52c41a' : 'var(--text-secondary)',
                fontSize: 12
              }}
              onClick={() => setGroupFilter(g.code)}
            >
              {g.name}
            </Tag>
          ))}
        </div>

        {/* Capability Filter Tags */}
        <div style={{ marginBottom: 10 }}>
          <span style={{ color: 'var(--text-secondary)', fontSize: 12, marginRight: 8 }}>{t('modelMarket.filter_by_capability')}:</span>
          {[
            { value: 'vision', label: t('modelMarket.cap_vision') },
            { value: 'moe', label: t('modelMarket.cap_moe') },
            { value: 'reasoning', label: t('modelMarket.cap_reasoning') },
            { value: 'function_call', label: t('modelMarket.cap_tools') },
            { value: 'embedding', label: t('modelMarket.embedding') },
            { value: 'reranker', label: t('modelMarket.reranker') },
          ].map(opt => (
            <Tag
              key={opt.value}
              className="filter-tag"
              style={{
                cursor: 'pointer',
                borderRadius: 16,
                padding: '2px 12px',
                border: capabilityFilter.includes(opt.value) ? '1px solid #722ed1' : '1px solid var(--border-color)',
                background: capabilityFilter.includes(opt.value) ? '#722ed115' : 'transparent',
                color: capabilityFilter.includes(opt.value) ? '#722ed1' : 'var(--text-secondary)',
                fontSize: 12
              }}
              onClick={() => {
                if (capabilityFilter.includes(opt.value)) {
                  setCapabilityFilter(capabilityFilter.filter(c => c !== opt.value))
                } else {
                  setCapabilityFilter([...capabilityFilter, opt.value])
                }
              }}
            >
              {opt.label}
            </Tag>
          ))}
        </div>

        {/* Context Length Filter Tags */}
        <div style={{ marginBottom: 10 }}>
          <span style={{ color: 'var(--text-secondary)', fontSize: 12, marginRight: 8 }}>{t('modelMarket.filter_by_context')}:</span>
          {[
            { value: '8', label: t('modelMarket.context_8k') },
            { value: '16', label: t('modelMarket.context_16k') },
            { value: '32', label: t('modelMarket.context_32k') },
            { value: '128', label: t('modelMarket.context_128k') },
          ].map(opt => (
            <Tag
              key={opt.value}
              className="filter-tag"
              style={{
                cursor: 'pointer',
                borderRadius: 16,
                padding: '2px 12px',
                border: contextFilter === opt.value ? '1px solid #fa8c16' : '1px solid var(--border-color)',
                background: contextFilter === opt.value ? '#fa8c1615' : 'transparent',
                color: contextFilter === opt.value ? '#fa8c16' : 'var(--text-secondary)',
                fontSize: 12
              }}
              onClick={() => setContextFilter(contextFilter === opt.value ? '' : opt.value)}
            >
              {opt.label}
            </Tag>
          ))}
        </div>

        {/* Trial Only Filter */}
        <div>
          <Tag
            className="filter-tag"
            style={{
              cursor: 'pointer',
              borderRadius: 16,
              padding: '2px 12px',
              border: trialOnly ? '1px solid #eb2f96' : '1px solid var(--border-color)',
              background: trialOnly ? '#eb2f9615' : 'transparent',
              color: trialOnly ? '#eb2f96' : 'var(--text-secondary)',
              fontSize: 12
            }}
            onClick={() => setTrialOnly(!trialOnly)}
          >
            🔥 {t('modelMarket.trial_only')}
          </Tag>
        </div>
          </>
        )}
      </Card>

      {/* Model List */}
      <Tabs
        activeKey={activeTab}
        onChange={(key) => {
          setActiveTab(key)
          if (key === 'groups') {
            setGroupFilter(null) // Clear group filter when entering groups tab
          }
        }}
        items={[
          {
            key: 'all',
            label: `${t('modelMarket.all_models')} (${models.length})`,
            children: viewMode === 'grid' ? (
              <Row gutter={[16, 16]}>
                {models.map(model => (
                  <Col xs={24} sm={12} lg={8} xl={6} key={model.id}>
                    <ModelCard model={model} />
                  </Col>
                ))}
              </Row>
            ) : (
              <div>
                {models.map(model => (
                  <ModelListItem key={model.id} model={model} />
                ))}
              </div>
            )
          },
          {
            key: 'groups',
            label: t('modelMarket.browse_by_group'),
            children: (
              <Row gutter={[16, 16]}>
                {groups.map(group => (
                  <Col xs={24} sm={12} lg={8} key={group.id}>
                    <Card
                      hoverable
                      onClick={() => {
                        setGroupFilter(group.code)
                        setActiveTab('all')
                      }}
                      style={{
                        borderRadius: 12,
                        border: 'none',
                        boxShadow: '0 2px 8px var(--shadow)',
                        background: 'var(--bg-card)'
                      }}
                      styles={{ body: { padding: 16, background: 'var(--bg-card)' } }}
                    >
                      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                        {group.logo_url ? (
                          <img
                            src={group.logo_url}
                            alt={group.name}
                            style={{ width: 48, height: 48, objectFit: 'contain', borderRadius: 12 }}
                            onError={(e) => {
                              (e.target as HTMLImageElement).style.display = 'none'
                            }}
                          />
                        ) : (
                          <div style={{
                            width: 48,
                            height: 48,
                            borderRadius: 12,
                            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            color: '#fff',
                            fontSize: 20
                          }}>
                            <RobotOutlined />
                          </div>
                        )}
                        <div style={{ flex: 1 }}>
                          <div style={{ fontWeight: 600, marginBottom: 4 }}>{group.name}</div>
                          <Badge
                            count={group.model_count}
                            style={{ backgroundColor: '#667eea' }}
                            showZero
                            overflowCount={999}
                          />
                        </div>
                      </div>
                    </Card>
                  </Col>
                ))}
              </Row>
            )
          }
        ]}
      />

      {/* Detail Drawer */}
      <Drawer
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ fontWeight: 600 }}>{selectedModel?.name}</span>
            {selectedModel?.is_trial && (
              <Tag style={{ background: '#f6ffed', color: '#52c41a', border: 'none', borderRadius: 4 }}>{t('modelMarket.support_trial')}</Tag>
            )}
            {selectedModel?.sla && (
              <Tag
                style={{
                  background: `${getSLAColor(selectedModel.sla)}15`,
                  color: getSLAColor(selectedModel.sla),
                  border: 'none',
                  borderRadius: 4
                }}
              >
                {selectedModel.sla === 'enterprise' ? '企业版' : selectedModel.sla === 'premium' ? '高级版' : '标准版'}
              </Tag>
            )}
          </div>
        }
        placement="right"
        width={520}
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
      >
        {selectedModel && (
          <div>
            <Descriptions column={1} size="small" style={{ marginBottom: 16 }}>
              <Descriptions.Item label={t('modelMarket.model_id')}>
                <code style={{ background: '#f5f5f5', padding: '2px 6px', borderRadius: 4, fontSize: 12 }}>{selectedModel.id}</code>
              </Descriptions.Item>
              <Descriptions.Item label={t('modelMarket.provider')}>{selectedModel.provider}</Descriptions.Item>
              <Descriptions.Item label={t('modelMarket.type')}>
                <Tag style={{ borderRadius: 4 }}>{selectedModel.model_type}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label={selectedModel.model_type === 'embedding' ? t('modelMarket.dimensions') : t('modelMarket.context_length')}>
                {selectedModel.context_len.toLocaleString()}{selectedModel.model_type === 'embedding' ? '' : ' tokens'}
              </Descriptions.Item>
              <Descriptions.Item label={t('modelMarket.input_price')}>
                <span style={{ color: '#667eea', fontWeight: 500 }}>{formatPrice(selectedModel.input_price)}</span>
              </Descriptions.Item>
              <Descriptions.Item label={t('modelMarket.output_price')}>
                <span style={{ color: '#764ba2', fontWeight: 500 }}>{formatPrice(selectedModel.output_price)}</span>
              </Descriptions.Item>
              {selectedModel.description && (
                <Descriptions.Item label={t('common.description')}>
                  <span style={{ color: '#595959' }}>{selectedModel.description}</span>
                </Descriptions.Item>
              )}
            </Descriptions>

            <Card title={t('modelMarket.trial_function')} style={{ marginBottom: 16, borderRadius: 12 }} styles={{ body: { padding: 12 } }}>
              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                {getCapabilityTags(selectedModel.capabilities)}
              </div>
            </Card>

            {selectedModel.is_trial && (
              <Card
                title={t('modelMarket.trial_function')}
                style={{ marginBottom: 16, borderRadius: 12 }}
                styles={{ header: { borderRadius: '12px 12px 0 0' } }}
              >
                {trialInfo ? (
                  trialInfo.available ? (
                    <div>
                      <Statistic
                        title={t('modelMarket.remaining_trial_quota')}
                        value={formatQuota(trialInfo.quota_limit - trialInfo.quota_used)}
                        suffix="tokens"
                        valueStyle={{ color: '#52c41a' }}
                      />
                      <Button type="primary" disabled style={{ marginTop: 16, borderRadius: 8 }}>
                        {t('modelMarket.trial_enabled')}
                      </Button>
                    </div>
                  ) : (
                    <div>
                      <p style={{ color: '#8c8c8c', marginBottom: 12 }}>试用状态: {trialInfo.reason}</p>
                      {trialInfo.quota_used > 0 && (
                        <Statistic
                          title={t('token.used_quota')}
                          value={formatQuota(trialInfo.quota_used)}
                          suffix={`/ ${formatQuota(trialInfo.quota_limit || 0)}`}
                        />
                      )}
                    </div>
                  )
                ) : (
                  <div>
                    <p style={{ color: '#595959', marginBottom: 16 }}>
                      {t('modelMarket.trial_quota_tokens', { quota: formatQuota(selectedModel.trial_quota || 0) })}
                    </p>
                    <Button
                      type="primary"
                      loading={trialLoading}
                      onClick={handleStartTrial}
                      style={{ borderRadius: 8, background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', border: 'none' }}
                    >
                      {t('modelMarket.start_trial')}
                    </Button>
                  </div>
                )}
              </Card>
            )}

            <Card
              title={t('modelMarket.price_calculator')}
              style={{ marginBottom: 16, borderRadius: 12 }}
              styles={{ header: { borderRadius: '12px 12px 0 0' } }}
            >
              <div style={{ display: 'flex', gap: 12, marginBottom: 12 }}>
                <div style={{ flex: 1 }}>
                  <div style={{ marginBottom: 6, color: '#8c8c8c', fontSize: 12 }}>{t('modelMarket.input_token_count')}</div>
                  <Input
                    type="number"
                    placeholder="如: 1000"
                    style={{ borderRadius: 8 }}
                    onChange={(e) => calcPrice(Number(e.target.value), priceCalc?.completion_tokens || 0)}
                  />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ marginBottom: 6, color: '#8c8c8c', fontSize: 12 }}>{t('modelMarket.output_token_count')}</div>
                  <Input
                    type="number"
                    placeholder="如: 500"
                    style={{ borderRadius: 8 }}
                    onChange={(e) => calcPrice(priceCalc?.prompt_tokens || 0, Number(e.target.value))}
                  />
                </div>
              </div>
              {priceCalc && (
                <div style={{ textAlign: 'center', padding: '16px 0', background: '#f9f0ff', borderRadius: 8 }}>
                  <div style={{ fontSize: 12, color: '#8c8c8c', marginBottom: 4 }}>{t('modelMarket.estimated_quota_consumption')}</div>
                  <div style={{ fontSize: 28, fontWeight: 700, color: '#722ed1' }}>
                    {priceCalc.quota_cost.toLocaleString()}
                  </div>
                </div>
              )}
            </Card>

            <Card
              title={t('modelMarket.usage_example')}
              extra={
                <Button
                  size="small"
                  icon={<CopyOutlined />}
                  onClick={() => {
                    navigator.clipboard.writeText(getModelExample(selectedModel.id, selectedModel.model_type))
                    message.success(t('common.copied_to_clipboard'))
                  }}
                >
                  {t('common.copy')}
                </Button>
              }
              style={{ borderRadius: 12 }}
              styles={{ header: { borderRadius: '12px 12px 0 0' } }}
            >
              <pre style={{
                background: '#1a1a2e',
                color: '#e0e0e0',
                padding: 16,
                borderRadius: 8,
                fontSize: 12,
                fontFamily: 'Monaco, Consolas, monospace',
                overflow: 'auto'
              }}>
{getModelExample(selectedModel.id, selectedModel.model_type)}
              </pre>
            </Card>
          </div>
        )}
      </Drawer>
    </div>
  )
}

export default ModelMarket
