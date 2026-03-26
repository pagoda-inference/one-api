import { useState, useEffect } from 'react'
import { Row, Col, Card, Statistic, Spin, Table, Tag, Progress } from 'antd'
import { DollarOutlined, ClockCircleOutlined, ApiOutlined, RiseOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import dayjs from 'dayjs'
import { getDashboard, getUsageByDay, getTopupOrders, User, UsageSummary, TopupOrder } from '../services/api'

const Dashboard: React.FC = () => {
  const [loading, setLoading] = useState(true)
  const [user, setUser] = useState<User | null>(null)
  const [usage, setUsage] = useState<UsageSummary | null>(null)
  const [marketStats, setMarketStats] = useState<{ total_models: number; total_providers: number } | null>(null)
  const [recentOrders, setRecentOrders] = useState<TopupOrder[]>([])
  const [usageChartData, setUsageChartData] = useState<{ date: string; tokens: number }[]>([])

  useEffect(() => {
    loadDashboard()
  }, [])

  const loadDashboard = async () => {
    try {
      setLoading(true)
      const [dashboardRes, usageRes] = await Promise.all([
        getDashboard(),
        getUsageByDay()
      ])

      const data = dashboardRes.data.data
      setUser(data.user)
      setUsage(data.usage)
      setMarketStats(data.market.total_models)
      setRecentOrders(data.recent_orders || [])

      // Process usage chart data
      if (usageRes.data.data) {
        const chartData = usageRes.data.data.map((item: any) => ({
          date: item.day,
          tokens: item.prompt_tokens + item.completion_tokens
        }))
        setUsageChartData(chartData)
      }
    } catch (error) {
      console.error('Failed to load dashboard:', error)
    } finally {
      setLoading(false)
    }
  }

  const getUsageChartOption = () => {
    const dates = usageChartData.map(d => d.date)
    const tokens = usageChartData.map(d => d.tokens)

    return {
      title: { text: '最近7天用量趋势', left: 'center' },
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'category', data: dates },
      yAxis: { type: 'value', name: 'Token数' },
      series: [{
        data: tokens,
        type: 'line',
        areaStyle: { color: '#1890ff33' },
        lineStyle: { color: '#1890ff' },
        itemStyle: { color: '#1890ff' }
      }]
    }
  }

  const formatQuota = (quota: number) => {
    if (quota >= 1000000000) {
      return (quota / 1000000000).toFixed(2) + 'B'
    }
    if (quota >= 1000000) {
      return (quota / 1000000).toFixed(2) + 'M'
    }
    if (quota >= 1000) {
      return (quota / 1000).toFixed(2) + 'K'
    }
    return quota.toString()
  }

  if (loading) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />
  }

  return (
    <div>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="账户余额"
              value={user?.quota || 0}
              formatter={(value) => formatQuota(Number(value))}
              prefix={<DollarOutlined />}
              suffix="quota"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="本月请求数"
              value={usage?.total_requests || 0}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="本月Token用量"
              value={usage?.total_tokens || 0}
              formatter={(value) => formatQuota(Number(value))}
              prefix={<RiseOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="可用模型"
              value={marketStats?.total_models || 0}
              suffix="个"
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={16}>
          <Card title="用量趋势">
            {usageChartData.length > 0 ? (
              <ReactECharts option={getUsageChartOption()} style={{ height: 300 }} />
            ) : (
              <div style={{ textAlign: 'center', color: '#999', padding: 60 }}>暂无用量数据</div>
            )}
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="最近充值">
            {recentOrders.length > 0 ? (
              <Table
                dataSource={recentOrders}
                size="small"
                pagination={false}
                rowKey="id"
                columns={[
                  { title: '金额', dataIndex: 'amount', render: (v) => `¥${v}` },
                  { title: '额度', dataIndex: 'quota', render: (v) => formatQuota(v) },
                  {
                    title: '状态',
                    dataIndex: 'status',
                    render: (status) => {
                      const map: Record<string, { color: string; text: string }> = {
                        paid: { color: 'green', text: '已支付' },
                        pending: { color: 'orange', text: '待支付' },
                        cancelled: { color: 'gray', text: '已取消' }
                      }
                      const s = map[status] || { color: 'gray', text: status }
                      return <Tag color={s.color}>{s.text}</Tag>
                    }
                  }
                ]}
              />
            ) : (
              <div style={{ textAlign: 'center', color: '#999', padding: 40 }}>暂无充值记录</div>
            )}
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Dashboard