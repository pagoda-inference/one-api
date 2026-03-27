import { useState, useEffect } from 'react'
import { Row, Col, Card, Table, DatePicker, Select, Statistic, Spin } from 'antd'
import { ApiOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import dayjs from 'dayjs'
import { getUsageSummary, getUsageByModel, getUsageDetail, getMarketModels, ModelUsage, Model, UsageSummary } from '../services/api'

const { RangePicker } = DatePicker
const { Option } = Select

const Usage: React.FC = () => {
  const [loading, setLoading] = useState(true)
  const [summary, setSummary] = useState<UsageSummary | null>(null)
  const [modelUsage, setModelUsage] = useState<ModelUsage[]>([])
  const [dailyUsage, setDailyUsage] = useState<any[]>([])
  const [models, setModels] = useState<Model[]>([])
  const [selectedModel, setSelectedModel] = useState<string>('')
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      setLoading(true)
      const [summaryRes, modelRes, modelsRes] = await Promise.all([
        getUsageSummary(),
        getUsageByModel(),
        getMarketModels({ limit: 100 })
      ])

      setSummary(summaryRes.data.data)
      setModelUsage(modelRes.data.data || [])
      setModels(modelsRes.data.data.models || [])
      setDailyUsage([])
    } catch (error) {
      console.error('Failed to load usage:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = async () => {
    try {
      setLoading(true)
      const params: any = {}

      if (dateRange) {
        params.start = dateRange[0].unix().toString()
        params.end = dateRange[1].unix().toString()
      }

      const [detailRes] = await Promise.all([
        getUsageDetail({ ...params, model: selectedModel || undefined })
      ])

      const data = detailRes.data.data
      setModelUsage(data.by_model || [])
      setDailyUsage(data.by_day || [])
    } catch (error) {
      console.error('Failed to search:', error)
    } finally {
      setLoading(false)
    }
  }

  const getUsageChartOption = () => {
    const dates = dailyUsage.map(d => d.day)
    const promptTokens = dailyUsage.map(d => d.prompt_tokens)
    const completionTokens = dailyUsage.map(d => d.completion_tokens)

    return {
      title: { text: '每日用量趋势', left: 'center' },
      tooltip: { trigger: 'axis' },
      legend: { data: ['输入Token', '输出Token'], bottom: 0 },
      xAxis: { type: 'category', data: dates },
      yAxis: { type: 'value', name: 'Token数' },
      series: [
        {
          name: '输入Token',
          data: promptTokens,
          type: 'bar',
          stack: 'total',
          itemStyle: { color: '#1890ff' }
        },
        {
          name: '输出Token',
          data: completionTokens,
          type: 'bar',
          stack: 'total',
          itemStyle: { color: '#52c41a' }
        }
      ]
    }
  }

  const formatQuota = (quota: number) => {
    if (quota >= 1000000) {
      return (quota / 1000000).toFixed(2) + 'M'
    }
    if (quota >= 1000) {
      return (quota / 1000).toFixed(2) + 'K'
    }
    return quota.toString()
  }

  const modelColumns = [
    { title: '模型', dataIndex: 'model_name', key: 'model_name' },
    {
      title: '请求数',
      dataIndex: 'request_count',
      key: 'request_count',
      sorter: (a: ModelUsage, b: ModelUsage) => a.request_count - b.request_count
    },
    {
      title: '消耗额度',
      dataIndex: 'quota',
      key: 'quota',
      render: (v: number) => formatQuota(v),
      sorter: (a: ModelUsage, b: ModelUsage) => a.quota - b.quota
    },
    {
      title: '输入Token',
      dataIndex: 'prompt_tokens',
      key: 'prompt_tokens',
      render: (v: number) => formatQuota(v)
    },
    {
      title: '输出Token',
      dataIndex: 'completion_tokens',
      key: 'completion_tokens',
      render: (v: number) => formatQuota(v)
    }
  ]

  const dailyColumns = [
    { title: '日期', dataIndex: 'day', key: 'day' },
    {
      title: '模型',
      dataIndex: 'model_name',
      key: 'model_name'
    },
    {
      title: '请求数',
      dataIndex: 'request_count',
      key: 'request_count'
    },
    {
      title: '消耗额度',
      dataIndex: 'quota',
      key: 'quota',
      render: (v: number) => formatQuota(v)
    },
    {
      title: '输入Token',
      dataIndex: 'prompt_tokens',
      key: 'prompt_tokens',
      render: (v: number) => formatQuota(v)
    },
    {
      title: '输出Token',
      dataIndex: 'completion_tokens',
      key: 'completion_tokens',
      render: (v: number) => formatQuota(v)
    }
  ]

  if (loading && !summary) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />
  }

  return (
    <div>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总请求数"
              value={summary?.total_requests || 0}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总消耗额度"
              value={summary?.total_quota || 0}
              formatter={(v) => formatQuota(Number(v))}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总输入Token"
              value={summary?.total_prompt_tokens || 0}
              formatter={(v) => formatQuota(Number(v))}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总输出Token"
              value={summary?.total_completion_tokens || 0}
              formatter={(v) => formatQuota(Number(v))}
            />
          </Card>
        </Col>
      </Row>

      <Card style={{ marginTop: 16 }}>
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={12} md={8}>
            <RangePicker
              style={{ width: '100%' }}
              onChange={(dates) => setDateRange(dates as any)}
            />
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Select
              placeholder="选择模型"
              allowClear
              style={{ width: '100%' }}
              value={selectedModel || undefined}
              onChange={(v) => setSelectedModel(v || '')}
            >
              {models.map(m => (
                <Option key={m.id} value={m.id}>{m.name}</Option>
              ))}
            </Select>
          </Col>
          <Col xs={24} md={4}>
            <button className="ant-btn ant-btn-primary" onClick={handleSearch}>
              查询
            </button>
          </Col>
        </Row>
      </Card>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <Card title="按模型统计">
            <Table
              dataSource={modelUsage}
              columns={modelColumns}
              rowKey="model_name"
              size="small"
              pagination={{ pageSize: 10 }}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="每日明细">
            <Table
              dataSource={dailyUsage}
              columns={dailyColumns}
              rowKey={(record) => `${record.day}-${record.model_name}`}
              size="small"
              pagination={{ pageSize: 10 }}
            />
          </Card>
        </Col>
      </Row>

      {dailyUsage.length > 0 && (
        <Card title="用量趋势图" style={{ marginTop: 16 }}>
          <ReactECharts option={getUsageChartOption()} style={{ height: 300 }} />
        </Card>
      )}
    </div>
  )
}

export default Usage