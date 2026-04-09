import { useState, useEffect } from 'react'
import { Row, Col, Card, Input, Select, Tag, Button, Drawer, Descriptions, Statistic, message, Tabs, Badge, Segmented } from 'antd'
import {
  SearchOutlined, RobotOutlined, SyncOutlined,
  ExperimentOutlined, AppstoreOutlined, BarsOutlined,
  ThunderboltOutlined, PictureOutlined, AudioOutlined, FileTextOutlined
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useTheme } from '../contexts/ThemeContext'
import { getMarketModels, getMarketStats, getMarketGroups, getModelTrial, startModelTrial, calculatePrice, Model, ModelGroup, MarketStats } from '../services/api'

const { Search } = Input

const ModelMarket: React.FC = () => {
  const { t } = useTranslation()
  const { themeMode } = useTheme()
  const [models, setModels] = useState<Model[]>([])
  const [groups, setGroups] = useState<ModelGroup[]>([])
  const [stats, setStats] = useState<MarketStats | null>(null)
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<string>('')
  const [groupFilter, setGroupFilter] = useState<string | null>(null)
  const [selectedModel, setSelectedModel] = useState<Model | null>(null)
  const [drawerVisible, setDrawerVisible] = useState(false)
  const [priceCalc, setPriceCalc] = useState<{ prompt_tokens: number; completion_tokens: number; quota_cost: number } | null>(null)
  const [trialInfo, setTrialInfo] = useState<any>(null)
  const [trialLoading, setTrialLoading] = useState(false)
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [activeTab, setActiveTab] = useState<string>('all')
  useEffect(() => {
    loadData()
  }, [typeFilter, groupFilter])

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

      // Apply search filter
      if (search) {
        const keyword = search.toLowerCase()
        allModels = allModels.filter((m: Model) =>
          m.name.toLowerCase().includes(keyword) ||
          m.id.toLowerCase().includes(keyword) ||
          m.provider.toLowerCase().includes(keyword)
        )
      }

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
      message.success('试用已开启')
      const res = await getModelTrial(selectedModel.id)
      setTrialInfo(res.data.data)
    } catch (error) {
      message.error('开启试用失败')
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
    if (price === 0) return '免费'
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
    { value: '', label: '全部类型' },
    { value: 'chat', label: '对话', icon: <RobotOutlined /> },
    { value: 'vlm', label: '多模态', icon: <ExperimentOutlined /> },
    { value: 'reranker', label: '重排序', icon: <SyncOutlined /> },
    { value: 'embedding', label: '嵌入', icon: <FileTextOutlined /> },
    { value: 'ocr', label: 'OCR', icon: <FileTextOutlined /> },
    { value: 'image', label: '图像', icon: <PictureOutlined /> },
    { value: 'audio', label: '音频', icon: <AudioOutlined /> }
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
  const ModelCard = ({ model }: { model: Model }) => (
    <Card
      hoverable
      onClick={() => showModelDetail(model)}
      className="model-card"
      style={{
        borderRadius: 12,
        border: 'none',
        boxShadow: '0 2px 8px var(--shadow)',
        overflow: 'hidden',
        transition: 'all 0.3s',
        background: 'var(--bg-card)'
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
                {t ? t('model.trial') : '试用'}
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
                企业专属
              </Tag>
            )}
          </div>

          <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 8 }}>
            {model.provider}
          </div>

          <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginBottom: 8, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {model.description || '暂无描述'}
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
              <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>输入</span>
              <span style={{ fontSize: 12, fontWeight: 500, color: '#667eea' }}>
                {formatPrice(model.input_price)}
              </span>
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <span style={{ fontSize: 11, color: 'var(--text-secondary)' }}>输出</span>
              <span style={{ fontSize: 12, fontWeight: 500, color: '#764ba2' }}>
                {formatPrice(model.output_price)}
              </span>
            </div>
          </div>
        </div>
      </div>
    </Card>
  )

  // List View
  const ModelListItem = ({ model }: { model: Model }) => (
    <Card
      hoverable
      onClick={() => showModelDetail(model)}
      style={{
        borderRadius: 12,
        border: 'none',
        boxShadow: '0 2px 8px var(--shadow)',
        marginBottom: 8,
        background: 'var(--bg-card)'
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
              <Tag style={{ background: '#f6ffed', color: '#52c41a', border: 'none', borderRadius: 4, fontSize: 10 }}>试用</Tag>
            )}
            <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{model.provider}</span>
          </div>
        </div>

        <div style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: 11, color: 'var(--text-secondary)' }}>输入</div>
            <div style={{ fontSize: 13, fontWeight: 500, color: '#667eea' }}>{formatPrice(model.input_price)}</div>
          </div>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: 11, color: 'var(--text-secondary)' }}>输出</div>
            <div style={{ fontSize: 13, fontWeight: 500, color: '#764ba2' }}>{formatPrice(model.output_price)}</div>
          </div>
        </div>
      </div>
    </Card>
  )

  return (
    <div>
      {/* Stats Row */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={8}>
          <StatCard
            title="可用模型"
            value={stats?.total_models || 0}
            suffix="个"
            icon={<RobotOutlined />}
            color="#667eea"
          />
        </Col>
        <Col xs={24} sm={8}>
          <StatCard
            title="供应商数量"
            value={stats?.total_groups || 0}
            suffix="个"
            icon={<BarsOutlined />}
            color="#52c41a"
          />
        </Col>
        <Col xs={24} sm={8}>
          <StatCard
            title="支持试用"
            value={stats?.trial_models || 0}
            suffix="个"
            icon={<ThunderboltOutlined />}
            color="#faad14"
          />
        </Col>
      </Row>

      {/* Filters */}
      <Card
        style={{ marginBottom: 16, borderRadius: 12, border: 'none', background: 'var(--bg-card)' }}
        styles={{ body: { padding: '16px 20px', background: 'var(--bg-card)' } }}
      >
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={12} md={8}>
            <Search
              placeholder="搜索模型名称、ID或提供商"
              allowClear
              onSearch={handleSearch}
              prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
              style={{ width: '100%' }}
            />
          </Col>
          <Col xs={12} sm={6} md={4}>
            <Select
              style={{ width: '100%', borderRadius: 8 }}
              value={typeFilter || undefined}
              onChange={setTypeFilter}
              options={typeOptions}
            />
          </Col>
          <Col xs={12} sm={6} md={4}>
            <Select
              placeholder="选择分组"
              allowClear
              style={{ width: '100%', borderRadius: 8 }}
              value={groupFilter || undefined}
              onChange={(v) => setGroupFilter(v || null)}
              options={groups.map(g => ({ value: g.code, label: g.name }))}
            />
          </Col>
          <Col xs={24} md={8} style={{ textAlign: 'right', display: 'flex', justifyContent: 'flex-end', gap: 12 }}>
            <Segmented
              value={viewMode}
              onChange={(v) => setViewMode(v as 'grid' | 'list')}
              options={[
                { value: 'grid', icon: <AppstoreOutlined /> },
                { value: 'list', icon: <BarsOutlined /> }
              ]}
            />
            <Button icon={<SyncOutlined />} onClick={loadData}>刷新</Button>
          </Col>
        </Row>
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
            label: `全部模型 (${models.length})`,
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
            label: '分组浏览',
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
                        {group.icon_url ? (
                          <img
                            src={group.icon_url}
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
              <Tag style={{ background: '#f6ffed', color: '#52c41a', border: 'none', borderRadius: 4 }}>支持试用</Tag>
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
              <Descriptions.Item label="模型ID">
                <code style={{ background: '#f5f5f5', padding: '2px 6px', borderRadius: 4, fontSize: 12 }}>{selectedModel.id}</code>
              </Descriptions.Item>
              <Descriptions.Item label="提供商">{selectedModel.provider}</Descriptions.Item>
              <Descriptions.Item label="类型">
                <Tag style={{ borderRadius: 4 }}>{selectedModel.model_type}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label={selectedModel.model_type === 'embedding' ? '维度' : '上下文长度'}>
                {selectedModel.context_len.toLocaleString()}{selectedModel.model_type === 'embedding' ? '' : ' tokens'}
              </Descriptions.Item>
              <Descriptions.Item label="输入价格">
                <span style={{ color: '#667eea', fontWeight: 500 }}>{formatPrice(selectedModel.input_price)}</span>
              </Descriptions.Item>
              <Descriptions.Item label="输出价格">
                <span style={{ color: '#764ba2', fontWeight: 500 }}>{formatPrice(selectedModel.output_price)}</span>
              </Descriptions.Item>
              {selectedModel.description && (
                <Descriptions.Item label="描述">
                  <span style={{ color: '#595959' }}>{selectedModel.description}</span>
                </Descriptions.Item>
              )}
            </Descriptions>

            <Card title="能力" style={{ marginBottom: 16, borderRadius: 12 }} styles={{ body: { padding: 12 } }}>
              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                {getCapabilityTags(selectedModel.capabilities)}
              </div>
            </Card>

            {selectedModel.is_trial && (
              <Card
                title="试用功能"
                style={{ marginBottom: 16, borderRadius: 12 }}
                styles={{ header: { borderRadius: '12px 12px 0 0' } }}
              >
                {trialInfo ? (
                  trialInfo.available ? (
                    <div>
                      <Statistic
                        title="剩余试用额度"
                        value={formatQuota(trialInfo.quota_limit - trialInfo.quota_used)}
                        suffix="tokens"
                        valueStyle={{ color: '#52c41a' }}
                      />
                      <Button type="primary" disabled style={{ marginTop: 16, borderRadius: 8 }}>
                        已开启试用
                      </Button>
                    </div>
                  ) : (
                    <div>
                      <p style={{ color: '#8c8c8c', marginBottom: 12 }}>试用状态: {trialInfo.reason}</p>
                      {trialInfo.quota_used > 0 && (
                        <Statistic
                          title="已使用"
                          value={formatQuota(trialInfo.quota_used)}
                          suffix={`/ ${formatQuota(trialInfo.quota_limit || 0)}`}
                        />
                      )}
                    </div>
                  )
                ) : (
                  <div>
                    <p style={{ color: '#595959', marginBottom: 16 }}>
                      该模型支持试用，您将获得 <strong>{formatQuota(selectedModel.trial_quota || 0)} tokens</strong> 的免费试用额度。
                    </p>
                    <Button
                      type="primary"
                      loading={trialLoading}
                      onClick={handleStartTrial}
                      style={{ borderRadius: 8, background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', border: 'none' }}
                    >
                      开启试用
                    </Button>
                  </div>
                )}
              </Card>
            )}

            <Card
              title="价格计算器"
              style={{ marginBottom: 16, borderRadius: 12 }}
              styles={{ header: { borderRadius: '12px 12px 0 0' } }}
            >
              <div style={{ display: 'flex', gap: 12, marginBottom: 12 }}>
                <div style={{ flex: 1 }}>
                  <div style={{ marginBottom: 6, color: '#8c8c8c', fontSize: 12 }}>输入Token数</div>
                  <Input
                    type="number"
                    placeholder="如: 1000"
                    style={{ borderRadius: 8 }}
                    onChange={(e) => calcPrice(Number(e.target.value), priceCalc?.completion_tokens || 0)}
                  />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ marginBottom: 6, color: '#8c8c8c', fontSize: 12 }}>输出Token数</div>
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
                  <div style={{ fontSize: 12, color: '#8c8c8c', marginBottom: 4 }}>预计消耗额度</div>
                  <div style={{ fontSize: 28, fontWeight: 700, color: '#722ed1' }}>
                    {priceCalc.quota_cost.toLocaleString()}
                  </div>
                </div>
              )}
            </Card>

            <Card
              title="使用示例"
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
{`import openai
openai.api_key = "your-api-key"
openai.api_base = "https://baotaai.bedicloud.net/v1"

response = openai.ChatCompletion.create(
    model="${selectedModel.id}",
    messages=[{"role": "user", "content": "Hello!"}]
)`}
              </pre>
            </Card>
          </div>
        )}
      </Drawer>
    </div>
  )
}

export default ModelMarket
