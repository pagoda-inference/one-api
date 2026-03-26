import { useState, useEffect } from 'react'
import { Row, Col, Card, Input, Select, Tag, Button, Drawer, Descriptions, Statistic, message } from 'antd'
import { SearchOutlined, CheckCircleOutlined, RobotOutlined, SyncOutlined } from '@ant-design/icons'
import { getMarketModels, getMarketProviders, getMarketStats, calculatePrice, Model, calculatePrice } from '../services/api'

const { Search } = Input

const ModelMarket: React.FC = () => {
  const [loading, setLoading] = useState(true)
  const [models, setModels] = useState<Model[]>([])
  const [providers, setProviders] = useState<string[]>([])
  const [stats, setStats] = useState<{ total_models: number; total_providers: number } | null>(null)
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<string>('')
  const [providerFilter, setProviderFilter] = useState<string>('')
  const [selectedModel, setSelectedModel] = useState<Model | null>(null)
  const [drawerVisible, setDrawerVisible] = useState(false)
  const [priceCalc, setPriceCalc] = useState<{ prompt_tokens: number; completion_tokens: number; quota_cost: number } | null>(null)

  useEffect(() => {
    loadData()
  }, [typeFilter, providerFilter])

  const loadData = async () => {
    try {
      setLoading(true)
      const [modelsRes, providersRes, statsRes] = await Promise.all([
        getMarketModels({ type: typeFilter, q: search, limit: 50 }),
        getMarketProviders(),
        getMarketStats()
      ])

      setModels(modelsRes.data.data.models || [])
      setProviders(providersRes.data.data || [])
      setStats(statsRes.data.data)
    } catch (error) {
      console.error('Failed to load models:', error)
    } finally {
      setLoading(false)
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
            <Statistic title="模型提供商" value={stats?.total_providers || 0} />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title="平均输入价格"
              value={stats?.avg_input_price || 0}
              formatter={(v) => `¥${Number(v).toFixed(4)}/1K`}
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
              placeholder="提供商"
              allowClear
              style={{ width: '100%' }}
              value={providerFilter || undefined}
              onChange={(v) => { setProviderFilter(v || ''); loadData() }}
            >
              {providers.map(p => (
                <Select.Option key={p.name} value={p.name}>{p.name}</Select.Option>
              ))}
            </Select>
          </Col>
          <Col xs={24} md={8} style={{ textAlign: 'right' }}>
            <Button icon={<SyncOutlined />} onClick={loadData}>刷新</Button>
          </Col>
        </Row>
      </Card>

      <Row gutter={[16, 16]}>
        {models.map(model => (
          <Col xs={24} sm={12} lg={8} xl={6} key={model.id}>
            <Card
              hoverable
              className="model-card"
              onClick={() => showModelDetail(model)}
              cover={
                <div style={{
                  height: 120,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                  color: 'white',
                  fontSize: 48
                }}>
                  {getModelIcon(model.model_type)}
                </div>
              }
            >
              <Card.Meta
                title={
                  <div>
                    <div>{model.name}</div>
                    <div style={{ fontSize: 12, color: '#999', fontWeight: 'normal' }}>
                      {model.provider} · {model.id}
                    </div>
                  </div>
                }
                description={
                  <div style={{ marginTop: 8 }}>
                    <div style={{ fontSize: 12, color: '#666', height: 40, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                      {model.description}
                    </div>
                    <div style={{ display: 'flex', gap: 8, marginTop: 8, flexWrap: 'wrap' }}>
                      {getCapabilityTags(model.capabilities)?.slice(0, 3)}
                    </div>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 12, paddingTop: 12, borderTop: '1px solid #f0f0f0' }}>
                      <span style={{ fontSize: 12, color: '#999' }}>输入</span>
                      <span style={{ fontSize: 14, color: '#1890ff', fontWeight: 500 }}>
                        {formatPrice(model.input_price)}
                      </span>
                    </div>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                      <span style={{ fontSize: 12, color: '#999' }}>输出</span>
                      <span style={{ fontSize: 14, color: '#1890ff', fontWeight: 500 }}>
                        {formatPrice(model.output_price)}
                      </span>
                    </div>
                  </div>
                }
              />
            </Card>
          </Col>
        ))}
      </Row>

      <Drawer
        title={selectedModel?.name}
        placement="right"
        width={500}
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
            </Descriptions>

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
          </div>
        )}
      </Drawer>
    </div>
  )
}

export default ModelMarket