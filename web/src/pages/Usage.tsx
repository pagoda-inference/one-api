import { useState, useEffect } from 'react'
import { Row, Col, Card, Table, DatePicker, Select, Statistic, Spin } from 'antd'
import { ApiOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import dayjs from 'dayjs'
import {
  getUsageSummary, getUsageByModel, getUsageDetail, getMarketModels,
  getAdminUsageSummary, getAdminUsageByUsers, getAdminUsageByModels,
  ModelUsage, Model, UsageSummary, User
} from '../services/api'
import { useTranslation } from 'react-i18next'

const { RangePicker } = DatePicker
const { Option } = Select

interface UserUsageItem {
  user_id: number
  username: string
  display_name: string
  quota: number
  request_count: number
  prompt_tokens: number
  completion_tokens: number
}

const Usage: React.FC = () => {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(true)
  const [summary, setSummary] = useState<UsageSummary | null>(null)
  const [modelUsage, setModelUsage] = useState<ModelUsage[]>([])
  const [userUsage, setUserUsage] = useState<UserUsageItem[]>([])
  const [dailyUsage, setDailyUsage] = useState<any[]>([])
  const [models, setModels] = useState<Model[]>([])
  const [selectedModel, setSelectedModel] = useState<string>('')
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null)
  const [isAdmin, setIsAdmin] = useState(false)

  useEffect(() => {
    checkUserRole()
    loadData()
  }, [])

  const checkUserRole = () => {
    const userInfoStr = localStorage.getItem('user_info')
    if (userInfoStr) {
      try {
        const userInfo: User = JSON.parse(userInfoStr)
        setIsAdmin((userInfo.role ?? 0) >= 10)
      } catch (error) {
        console.error('Failed to parse user info:', error)
      }
    }
  }

  const loadData = async () => {
    try {
      setLoading(true)
      const end = dayjs()
      const start = dayjs().subtract(7, 'day')

      // Check role from localStorage directly
      const userInfoStr = localStorage.getItem('user_info')
      let isAdminUser = false
      if (userInfoStr) {
        try {
          const userInfo: User = JSON.parse(userInfoStr)
          isAdminUser = (userInfo.role ?? 0) >= 10
          setIsAdmin(isAdminUser)
        } catch (error) {
          console.error('Failed to parse user info:', error)
        }
      }

      if (isAdminUser) {
        // Admin: load all users usage
        const [summaryRes, usersRes, modelsRes] = await Promise.all([
          getAdminUsageSummary({ start: start.unix().toString(), end: end.unix().toString() }),
          getAdminUsageByUsers({ start: start.unix().toString(), end: end.unix().toString() }),
          getAdminUsageByModels({ start: start.unix().toString(), end: end.unix().toString() })
        ])

        setSummary(summaryRes.data?.data || null)
        setUserUsage(usersRes.data?.data?.users || [])
        setModelUsage(modelsRes.data?.data?.models || [])
        setModels(modelsRes.data?.data?.models || [])
      } else {
        // User: load own usage
        const [summaryRes, modelRes, modelsRes, detailRes] = await Promise.all([
          getUsageSummary(),
          getUsageByModel(),
          getMarketModels({ limit: 100 }),
          getUsageDetail({ start: start.unix().toString(), end: end.unix().toString() })
        ])

        setSummary(summaryRes.data?.data || null)
        setModelUsage(modelRes.data?.data || [])
        setModels(modelsRes.data?.data?.models || [])
        const data = detailRes.data?.data || {}
        setDailyUsage(data.by_day || [])
      }
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

      // Check role from localStorage directly
      const userInfoStr = localStorage.getItem('user_info')
      let isAdminUser = false
      if (userInfoStr) {
        try {
          const userInfo: User = JSON.parse(userInfoStr)
          isAdminUser = (userInfo.role ?? 0) >= 10
        } catch (error) {
          console.error('Failed to parse user info:', error)
        }
      }

      if (isAdminUser) {
        const [usersRes, modelsRes] = await Promise.all([
          getAdminUsageByUsers(params),
          getAdminUsageByModels({ ...params, model: selectedModel || undefined })
        ])
        setUserUsage(usersRes.data?.data?.users || [])
        setModelUsage(modelsRes.data?.data?.models || [])
      } else {
        const detailRes = await getUsageDetail({ ...params, model: selectedModel || undefined })
        const data = detailRes.data?.data || {}
        setModelUsage(data.by_model || [])
        setDailyUsage(data.by_day || [])
      }
    } catch (error) {
      console.error('Failed to search:', error)
    } finally {
      setLoading(false)
    }
  }

  const getUsageChartOption = () => {
    try {
      const safeDailyUsage = dailyUsage || []
      if (safeDailyUsage.length === 0) {
        return {}
      }
      const dates = safeDailyUsage.map((d: any) => d.day || '')
      const promptTokens = safeDailyUsage.map((d: any) => d.prompt_tokens || 0)
      const completionTokens = safeDailyUsage.map((d: any) => d.completion_tokens || 0)

      return {
        title: { text: t('usage.daily_usage_trend'), left: 'center' },
        tooltip: { trigger: 'axis' },
        legend: { data: [t('usage.input_token'), t('usage.output_token')], bottom: 0 },
        grid: { left: 100, right: 20, top: 40, bottom: 40 },
        xAxis: { type: 'category', data: dates },
        yAxis: {
          type: 'value',
          name: t('usage.token_count'),
          nameLocation: 'middle',
          nameGap: 70,
          nameTextStyle: { color: '#8c8c8c', fontSize: 11 },
          axisLabel: { color: '#8c8c8c' }
        },
        series: [
          {
            name: t('usage.input_token'),
            data: promptTokens,
            type: 'bar',
            stack: 'total',
            itemStyle: { color: '#1890ff' }
          },
          {
            name: t('usage.output_token'),
            data: completionTokens,
            type: 'bar',
            stack: 'total',
            itemStyle: { color: '#52c41a' }
          }
        ]
      }
    } catch (e) {
      console.error('Chart error:', e)
      return {}
    }
  }

  const formatQuota = (quota: number) => {
    if (quota === null || quota === undefined || Number.isNaN(quota)) {
      return '0'
    }
    if (quota >= 1000000) {
      return (quota / 1000000).toFixed(2) + 'M'
    }
    if (quota >= 1000) {
      return (quota / 1000).toFixed(2) + 'K'
    }
    return quota.toString()
  }

  const modelColumns = [
    { title: t('dashboard.model'), dataIndex: 'model_name', key: 'model_name' },
    {
      title: t('usage.request_count'),
      dataIndex: 'request_count',
      key: 'request_count',
      sorter: (a: ModelUsage, b: ModelUsage) => a.request_count - b.request_count
    },
    {
      title: t('usage.consumed_quota'),
      dataIndex: 'quota',
      key: 'quota',
      render: (v: number) => formatQuota(v),
      sorter: (a: ModelUsage, b: ModelUsage) => a.quota - b.quota
    },
    {
      title: t('usage.input_token'),
      dataIndex: 'prompt_tokens',
      key: 'prompt_tokens',
      render: (v: number) => formatQuota(v)
    },
    {
      title: t('usage.output_token'),
      dataIndex: 'completion_tokens',
      key: 'completion_tokens',
      render: (v: number) => formatQuota(v)
    }
  ]

  const userColumns = [
    {
      title: t('usage.user'),
      dataIndex: 'display_name',
      key: 'display_name',
      render: (v: string, record: UserUsageItem) => v || record.username || `User ${record.user_id}`
    },
    {
      title: t('usage.request_count'),
      dataIndex: 'request_count',
      key: 'request_count',
      sorter: (a: UserUsageItem, b: UserUsageItem) => a.request_count - b.request_count
    },
    {
      title: t('usage.consumed_quota'),
      dataIndex: 'quota',
      key: 'quota',
      render: (v: number) => formatQuota(v),
      sorter: (a: UserUsageItem, b: UserUsageItem) => a.quota - b.quota
    },
    {
      title: t('usage.input_token'),
      dataIndex: 'prompt_tokens',
      key: 'prompt_tokens',
      render: (v: number) => formatQuota(v)
    },
    {
      title: t('usage.output_token'),
      dataIndex: 'completion_tokens',
      key: 'completion_tokens',
      render: (v: number) => formatQuota(v)
    }
  ]

  const dailyColumns = [
    { title: t('usage.date'), dataIndex: 'day', key: 'day' },
    {
      title: t('dashboard.model'),
      dataIndex: 'model_name',
      key: 'model_name'
    },
    {
      title: t('usage.request_count'),
      dataIndex: 'request_count',
      key: 'request_count'
    },
    {
      title: t('usage.consumed_quota'),
      dataIndex: 'quota',
      key: 'quota',
      render: (v: number) => formatQuota(v)
    },
    {
      title: t('usage.input_token'),
      dataIndex: 'prompt_tokens',
      key: 'prompt_tokens',
      render: (v: number) => formatQuota(v)
    },
    {
      title: t('usage.output_token'),
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
        <Col xs={24} sm={12} lg={isAdmin ? 6 : 6}>
          <Card>
            <Statistic
              title={isAdmin ? t('usage.total_consumed_quota') : t('usage.total_requests')}
              value={isAdmin ? (summary as any)?.total_quota ?? 0 : summary?.total_requests ?? 0}
              prefix={<ApiOutlined />}
              formatter={(v) => formatQuota(Number(v) || 0)}
            />
          </Card>
        </Col>
        {isAdmin ? (
          <>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title={t('usage.total_token')}
                  value={(summary as any)?.total_tokens ?? 0}
                  formatter={(v) => formatQuota(Number(v) || 0)}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title={t('usage.total_input_token')}
                  value={(summary as any)?.total_prompt_tokens ?? 0}
                  formatter={(v) => formatQuota(Number(v) || 0)}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title={t('usage.total_output_token')}
                  value={(summary as any)?.total_completion_tokens ?? 0}
                  formatter={(v) => formatQuota(Number(v) || 0)}
                />
              </Card>
            </Col>
          </>
        ) : (
          <>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title={t('usage.consumed_quota')}
                  value={summary?.total_quota ?? 0}
                  formatter={(v) => formatQuota(Number(v) || 0)}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title={t('usage.total_input_token')}
                  value={summary?.total_prompt_tokens ?? 0}
                  formatter={(v) => formatQuota(Number(v) || 0)}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <Card>
                <Statistic
                  title={t('usage.total_output_token')}
                  value={summary?.total_completion_tokens ?? 0}
                  formatter={(v) => formatQuota(Number(v) || 0)}
                />
              </Card>
            </Col>
          </>
        )}
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
              placeholder={t('usage.select_model')}
              allowClear
              style={{ width: '100%' }}
              value={selectedModel || undefined}
              onChange={(v) => setSelectedModel(v || '')}
            >
              {models.map(m => (
                <Option key={(m as any).model_name || m.id} value={(m as any).model_name || m.id}>{m.name || (m as any).model_name}</Option>
              ))}
            </Select>
          </Col>
          <Col xs={24} md={4}>
            <button className="ant-btn ant-btn-primary" onClick={handleSearch}>
              {t('common.query')}
            </button>
          </Col>
        </Row>
      </Card>

      {isAdmin ? (
        // Admin view: show user and model tables
        <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
          <Col xs={24} lg={12}>
            <Card title={t('usage.usage_by_user')}>
              <Table
                dataSource={userUsage}
                columns={userColumns}
                rowKey="user_id"
                size="small"
                pagination={{ pageSize: 10 }}
              />
            </Card>
          </Col>
          <Col xs={24} lg={12}>
            <Card title={t('usage.usage_by_model')}>
              <Table
                dataSource={modelUsage}
                columns={modelColumns}
                rowKey="model_name"
                size="small"
                pagination={{ pageSize: 10 }}
              />
            </Card>
          </Col>
        </Row>
      ) : (
        // User view: show model and daily tables
        <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
          <Col xs={24} lg={12}>
            <Card title={t('usage.usage_by_model')}>
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
            <Card title={t('usage.daily_detail')}>
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
      )}

      {!isAdmin && Array.isArray(dailyUsage) && dailyUsage.length > 0 ? (
        <Card title={t('usage.usage_trend_chart')} style={{ marginTop: 16 }}>
          <ReactECharts option={getUsageChartOption()} style={{ height: 300 }} />
        </Card>
      ) : null}
    </div>
  )
}

export default Usage