import { useState, useEffect } from 'react'
import { Row, Col, Card, Input, Select, Tag, Button, Drawer, Descriptions, Statistic, message, Tabs } from 'antd'
import { SearchOutlined, CheckCircleOutlined, RobotOutlined, SyncOutlined, ExperimentOutlined, CrownOutlined } from '@ant-design/icons'
import { getMarketModels, getMarketStats, getMarketGroups, getModelTrial, startModelTrial, calculatePrice, Model, ModelGroup, MarketStats } from '../services/api'

const { Search } = Input
const { TabPane } = Tabs

const ModelMarket: React.FC = () => {
  const [models, setModels] = useState<Model[]>([])
  const [groups, setGroups] = useState<ModelGroup[]>([])
  const [stats, setStats] = useState<MarketStats | null>(null)
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<string>('')
  const [groupFilter, setGroupFilter] = useState<number | null>(null)
  const [selectedModel, setSelectedModel] = useState<Model | null>(null)
  const [drawerVisible, setDrawerVisible] = useState(false)
  const [priceCalc, setPriceCalc] = useState<{ prompt_tokens: number; completion_tokens: number; quota_cost: number } | null>(null)
  const [trialInfo, setTrialInfo] = useState<any>(null)
  const [trialLoading, setTrialLoading] = useState(false)

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

      // Apply group filter
      if (groupFilter) {
        allModels = allModels.filter((m: Model) => m.group_id === groupFilter)
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

    // Load trial info
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
      case 'embedding': return <CheckCircleOutlined />
      default: return <RobotOutlined />
    }
  }

  const getCapabilityTags = (capabilities: string) => {
    try {
      const caps = JSON.parse(capabilities) as string[]
      return caps.map(cap => <Tag key={cap} color="blue">{cap}</Tag>)
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
      case 'enterprise': return 'red'
      case 'premium': return 'gold'
      default: return 'green'
    }
  }

  const getSLAIcon = (sla: string) => {
    if (sla === 'enterprise') return <CrownOutlined />
    return null
  }

  return (
    <div>
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic title="可用模型" value={stats?.total_models || 0} suffix="个" />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic title="模型分组" value={stats?.total_groups || 0} />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title="支持试用模型"
              value={stats?.trial_models || 0}
              prefix={<ExperimentOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card style={{ marginBottom: 16 }}>
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={12} md={8}>
            <Search
              placeholder="搜索模型名称或ID"
              allowClear
              onSearch={handleSearch}
              prefix={<SearchOutlined />}
            />
          </Col>
          <Col xs={12} sm={6} md={4}>
            <Select
              placeholder="模型类型"
              allowClear
              style={{ width: '100%' }}
              value={typeFilter || undefined}
              onChange={setTypeFilter}
            >
              <Select.Option value="chat">Chat</Select.Option>
              <Select.Option value="embedding">Embedding</Select.Option>
              <Select.Option value="image">Image</Select.Option>
              <Select.Option value="audio">Audio</Select.Option>
            </Select>
          </Col>
          <Col xs={12} sm={6} md={4}>
            <Select
              placeholder="选择分组"
              allowClear
              style={{ width: '100%' }}
              value={groupFilter || undefined}
              onChange={(v) => setGroupFilter(v || null)}
            >
              {groups.map(g => (
                <Select.Option key={g.id} value={g.id}>{g.name}</Select.Option>
              ))}
            </Select>
          </Col>
          <Col xs={24} md={8} style={{ textAlign: 'right' }}>
            <Button icon={<SyncOutlined />} onClick={loadData}>刷新</Button>
          </Col>
        </Row>
      </Card>

      <Tabs defaultActiveKey="1">
        <TabPane tab={`全部模型 (${models.length})`} key="1">
          <Row gutter={[16, 16]}>
            {models.map(model => (
              <Col xs={24} sm={12} lg={8} xl={6} key={model.id}>
                <Card
                  hoverable
                  className="model-card"
                  onClick={() => showModelDetail(model)}
                  cover={
                    <div style={{
                      height: 100,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                      color: 'white',
                      fontSize: 36
                    }}>
                      {getModelIcon(model.model_type)}
                    </div>
                  }
                >
                  <Card.Meta
                    title={
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <span>{model.name}</span>
                        {model.is_trial && <Tag color="green" icon={<ExperimentOutlined />}>试用</Tag>}
                        {model.sla === 'enterprise' && <Tag color="red" icon={<CrownOutlined />}>企业</Tag>}
                      </div>
                    }
                    description={
                      <div style={{ fontSize: 12, color: '#999', marginBottom: 8 }}>
                        {model.provider} · {model.id}
                      </div>
                    }
                  />
                  <div style={{ marginTop: 8, fontSize: 12, color: '#666', height: 36, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                    {model.description}
                  </div>
                  <div style={{ display: 'flex', gap: 4, marginTop: 8, flexWrap: 'wrap' }}>
                    {getCapabilityTags(model.capabilities)?.slice(0, 2)}
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 12, paddingTop: 8, borderTop: '1px solid #f0f0f0' }}>
                    <span style={{ fontSize: 11, color: '#999' }}>输入</span>
                    <span style={{ fontSize: 13, color: '#1890ff', fontWeight: 500 }}>
                      {formatPrice(model.input_price)}
                    </span>
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        </TabPane>

        <TabPane tab="分组浏览" key="2">
          <Row gutter={[16, 16]}>
            {groups.map(group => (
              <Col xs={24} sm={12} lg={8} key={group.id}>
                <Card
                  hoverable
                  onClick={() => { setGroupFilter(group.id); window.scrollTo({ top: 0, behavior: 'smooth' }) }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                    <div style={{
                      width: 48,
                      height: 48,
                      borderRadius: 8,
                      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      color: 'white',
                      fontSize: 24
                    }}>
                      <RobotOutlined />
                    </div>
                    <div style={{ flex: 1 }}>
                      <div style={{ fontWeight: 600 }}>{group.name}</div>
                      <div style={{ fontSize: 12, color: '#999' }}>{group.description}</div>
                      <div style={{ fontSize: 12, color: '#1890ff' }}>{group.model_count} 个模型</div>
                    </div>
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        </TabPane>
      </Tabs>

      <Drawer
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span>{selectedModel?.name}</span>
            {selectedModel?.is_trial && <Tag color="green">支持试用</Tag>}
            {selectedModel?.sla && (
              <Tag color={getSLAColor(selectedModel.sla)} icon={getSLAIcon(selectedModel.sla)}>
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
            <Descriptions column={1} bordered size="small">
              <Descriptions.Item label="模型ID">{selectedModel.id}</Descriptions.Item>
              <Descriptions.Item label="提供商">{selectedModel.provider}</Descriptions.Item>
              <Descriptions.Item label="类型">{selectedModel.model_type}</Descriptions.Item>
              <Descriptions.Item label="上下文长度">{selectedModel.context_len.toLocaleString()} tokens</Descriptions.Item>
              <Descriptions.Item label="输入价格">{formatPrice(selectedModel.input_price)}</Descriptions.Item>
              <Descriptions.Item label="输出价格">{formatPrice(selectedModel.output_price)}</Descriptions.Item>
              {selectedModel.is_trial && selectedModel.trial_quota && (
                <Descriptions.Item label="试用额度">{formatQuota(selectedModel.trial_quota)} tokens</Descriptions.Item>
              )}
            </Descriptions>

            {selectedModel.is_trial && (
              <Card title="试用功能" style={{ marginTop: 16 }}>
                {trialInfo ? (
                  trialInfo.available ? (
                    <div>
                      <Statistic
                        title="剩余试用额度"
                        value={formatQuota(trialInfo.quota_limit - trialInfo.quota_used)}
                        suffix="tokens"
                      />
                      <div style={{ marginTop: 16 }}>
                        <Button type="primary" disabled>已开启试用</Button>
                      </div>
                    </div>
                  ) : (
                    <div>
                      <p style={{ color: '#666' }}>试用状态: {trialInfo.reason}</p>
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
                    <p style={{ color: '#666', marginBottom: 16 }}>
                      该模型支持试用，您将获得 {formatQuota(selectedModel.trial_quota || 0)} tokens 的免费试用额度。
                    </p>
                    <Button type="primary" loading={trialLoading} onClick={handleStartTrial}>
                      开启试用
                    </Button>
                  </div>
                )}
              </Card>
            )}

            <Card title="能力标签" style={{ marginTop: 16 }}>
              {getCapabilityTags(selectedModel.capabilities) || '无'}
            </Card>

            <Card title="价格计算器" style={{ marginTop: 16 }}>
              <div style={{ display: 'flex', gap: 16 }}>
                <div style={{ flex: 1 }}>
                  <div style={{ marginBottom: 8, color: '#666' }}>输入Token数</div>
                  <Input
                    type="number"
                    placeholder="如: 1000"
                    onChange={(e) => calcPrice(Number(e.target.value), priceCalc?.completion_tokens || 0)}
                  />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ marginBottom: 8, color: '#666' }}>输出Token数</div>
                  <Input
                    type="number"
                    placeholder="如: 500"
                    onChange={(e) => calcPrice(priceCalc?.prompt_tokens || 0, Number(e.target.value))}
                  />
                </div>
              </div>
              {priceCalc && (
                <div style={{ marginTop: 16, textAlign: 'center' }}>
                  <div style={{ fontSize: 14, color: '#666' }}>预计消耗额度</div>
                  <div style={{ fontSize: 32, fontWeight: 600, color: '#1890ff' }}>
                    {priceCalc.quota_cost.toLocaleString()}
                  </div>
                </div>
              )}
            </Card>

            <Card title="使用说明" style={{ marginTop: 16 }}>
              <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4, fontSize: 12 }}>
{`import openai
openai.api_key = "your-api-key"
openai.api_base = "https://your-domain.com/v1"

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