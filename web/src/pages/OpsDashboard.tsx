import { useState, useEffect } from 'react'
import { Row, Col, Card, Table, Statistic, Spin, Progress, Tag, Tabs, Button, Space, Modal, Form, Input, message } from 'antd'
import { DollarOutlined, UserOutlined, ApiOutlined, RiseOutlined, SafetyCertificateOutlined, AlertOutlined, DashboardOutlined, LineChartOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import { getOpsStats, getChannelHealth, getSystemHealth, getAlertConfig, updateAlertConfig, exportReport, OpsStats, ChannelHealth, SystemHealth, AlertConfig } from '../services/api'

const { TabPane } = Tabs

const OpsDashboard: React.FC = () => {
  const [loading, setLoading] = useState(true)
  const [stats, setStats] = useState<OpsStats | null>(null)
  const [channelHealth, setChannelHealth] = useState<ChannelHealth[]>([])
  const [systemHealth, setSystemHealth] = useState<SystemHealth | null>(null)
  const [alertConfig, setAlertConfig] = useState<AlertConfig | null>(null)
  const [overallHealth, setOverallHealth] = useState(0)
  const [enabledCount, setEnabledCount] = useState(0)
  const [totalChannels, setTotalChannels] = useState(0)
  const [alertModalVisible, setAlertModalVisible] = useState(false)
  const [form] = Form.useForm()

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      setLoading(true)
      const [statsRes, healthRes, sysRes, alertRes] = await Promise.all([
        getOpsStats(),
        getChannelHealth(),
        getSystemHealth(),
        getAlertConfig()
      ])

      setStats(statsRes.data.data)
      setChannelHealth(healthRes.data.data.channels || [])
      setOverallHealth(healthRes.data.data.overall_health || 0)
      setEnabledCount(healthRes.data.data.enabled_count || 0)
      setTotalChannels(healthRes.data.data.total_count || 0)
      setSystemHealth(sysRes.data.data)
      setAlertConfig(alertRes.data.data)
    } catch (error) {
      console.error('Failed to load ops data:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleExport = async (type: string) => {
    try {
      const res = await exportReport({ type })
      message.success(`报表已生成: ${res.data.data.report_url}`)
    } catch (error) {
      message.error('导出失败')
    }
  }

  const handleUpdateAlert = async (values: any) => {
    try {
      await updateAlertConfig(values)
      message.success('告警配置已更新')
      setAlertModalVisible(false)
      loadData()
    } catch (error) {
      message.error('更新失败')
    }
  }

  const formatQuota = (quota: number) => {
    if (quota >= 1000000) return (quota / 1000000).toFixed(2) + 'M'
    if (quota >= 1000) return (quota / 1000).toFixed(2) + 'K'
    return quota.toString()
  }

  const getRevenueChartOption = () => {
    if (!stats) return {}
    const days = Object.keys(stats.revenue_by_day || {}).sort()
    const revenues = days.map(d => stats.revenue_by_day[d] || 0)
    const topups = days.map(d => stats.topup_by_day[d] || 0)

    return {
      title: { text: '日营收趋势 (近7天)', left: 'center' },
      tooltip: { trigger: 'axis' },
      legend: { data: ['营收', '充值'], bottom: 0 },
      xAxis: { type: 'category', data: days },
      yAxis: { type: 'value', name: '金额(元)' },
      series: [
        { name: '营收', data: revenues, type: 'bar', itemStyle: { color: '#52c41a' } },
        { name: '充值', data: topups, type: 'bar', itemStyle: { color: '#1890ff' } }
      ]
    }
  }

  const channelColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '类型', dataIndex: 'type', key: 'type', render: (v: number) => ['OpenAI', 'Azure', 'Anthropic', 'Google', '自定义'][v] || '未知' },
    { title: 'Base URL', dataIndex: 'base_url', key: 'base_url', ellipsis: true },
    {
      title: '成功率',
      dataIndex: 'success_rate',
      key: 'success_rate',
      render: (v: number) => <Progress percent={v} size="small" status={v < 80 ? 'exception' : undefined} />
    },
    { title: '平均延迟', dataIndex: 'avg_latency', key: 'avg_latency', render: (v: number) => `${v}ms` },
    { title: '优先级', dataIndex: 'priority', key: 'priority' },
    {
      title: '状态',
      dataIndex: 'is_enabled',
      key: 'is_enabled',
      render: (v: boolean) => <Tag color={v ? 'green' : 'red'}>{v ? '启用' : '禁用'}</Tag>
    }
  ]

  if (loading && !stats) {
    return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', marginTop: 100 }} />
  }

  return (
    <div>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="今日营收"
              value={stats?.today_revenue || 0}
              prefix={<DollarOutlined />}
              precision={2}
              suffix="元"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="今日用量"
              value={stats?.today_usage_tokens || 0}
              prefix={<ApiOutlined />}
              formatter={(v) => formatQuota(Number(v))}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="活跃用户"
              value={stats?.active_users || 0}
              prefix={<UserOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="渠道健康率"
              value={overallHealth}
              prefix={<SafetyCertificateOutlined />}
              suffix="%"
              precision={1}
            />
            <div style={{ marginTop: 8, color: '#666', fontSize: 12 }}>
              {enabledCount}/{totalChannels} 渠道在线
            </div>
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="总用户数" value={stats?.total_users || 0} prefix={<UserOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="总渠道数" value={stats?.total_channels || 0} prefix={<ApiOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总消耗Token"
              value={stats?.total_tokens || 0}
              formatter={(v) => formatQuota(Number(v))}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总额度"
              value={stats?.total_quota || 0}
              formatter={(v) => formatQuota(Number(v))}
            />
          </Card>
        </Col>
      </Row>

      <Tabs defaultActiveKey="1" style={{ marginTop: 16 }}>
        <TabPane tab={<span><LineChartOutlined /> 营收概览</span>} key="1">
          <Row gutter={[16, 16]}>
            <Col xs={24} lg={16}>
              <Card title="日营收趋势">
                <ReactECharts option={getRevenueChartOption()} style={{ height: 300 }} />
              </Card>
            </Col>
            <Col xs={24} lg={8}>
              <Card title="快捷操作">
                <Space direction="vertical" style={{ width: '100%' }}>
                  <Button block icon={<DollarOutlined />} onClick={() => handleExport('daily')}>
                    导出日报
                  </Button>
                  <Button block icon={<RiseOutlined />} onClick={() => handleExport('weekly')}>
                    导出周报
                  </Button>
                  <Button block icon={<DashboardOutlined />} onClick={() => handleExport('monthly')}>
                    导出月报
                  </Button>
                </Space>
              </Card>
            </Col>
          </Row>
        </TabPane>

        <TabPane tab={<span><ApiOutlined /> 渠道健康</span>} key="2">
          <Card title="渠道状态">
            <Table
              dataSource={channelHealth}
              columns={channelColumns}
              rowKey="id"
              size="small"
              pagination={{ pageSize: 10 }}
            />
          </Card>
        </TabPane>

        <TabPane tab={<span><AlertOutlined /> 告警配置</span>} key="3">
          <Card title="告警设置">
            <Row gutter={[16, 16]}>
              <Col xs={24} lg={12}>
                <Card size="small" title="当前配置">
                  <Space direction="vertical">
                    <div>渠道失败阈值: <Tag color="red">{alertConfig?.channel_failure_threshold}</Tag></div>
                    <div>队列利用率告警: <Tag color="orange">{alertConfig?.queue_utilization_alert}%</Tag></div>
                    <div>错误率告警: <Tag color="red">{alertConfig?.error_rate_alert}%</Tag></div>
                    <div>延迟阈值: <Tag color="blue">{alertConfig?.latency_threshold}ms</Tag></div>
                    <div>告警邮箱: <Tag>{alertConfig?.alert_email || '未配置'}</Tag></div>
                    <div>告警状态: <Tag color={alertConfig?.enabled ? 'green' : 'red'}>{alertConfig?.enabled ? '启用' : '禁用'}</Tag></div>
                  </Space>
                </Card>
              </Col>
              <Col xs={24} lg={12}>
                <Card size="small" title="系统信息">
                  {systemHealth && (
                    <Space direction="vertical">
                      <div>运行时间: <Tag>{Math.floor(systemHealth.uptime / 86400)}天{Math.floor((systemHealth.uptime % 86400) / 3600)}小时</Tag></div>
                      <div>最大并发: <Tag>{systemHealth.config?.max_concurrent}</Tag></div>
                      <div>健康检查间隔: <Tag>{systemHealth.config?.health_interval}s</Tag></div>
                      <div>熔断阈值: <Tag>{systemHealth.config?.cb_threshold}</Tag></div>
                      <div>队列利用率: <Progress percent={Math.round(systemHealth.queue?.utilization || 0)} size="small" /></div>
                    </Space>
                  )}
                </Card>
              </Col>
            </Row>
            <Button type="primary" style={{ marginTop: 16 }} onClick={() => {
              form.setFieldsValue(alertConfig)
              setAlertModalVisible(true)
            }}>
              修改配置
            </Button>
          </Card>
        </TabPane>
      </Tabs>

      <Modal
        title="修改告警配置"
        open={alertModalVisible}
        onCancel={() => setAlertModalVisible(false)}
        footer={null}
      >
        <Form form={form} onFinish={handleUpdateAlert} layout="vertical">
          <Form.Item name="channel_failure_threshold" label="渠道失败阈值">
            <Input type="number" />
          </Form.Item>
          <Form.Item name="queue_utilization_alert" label="队列利用率告警(%)">
            <Input type="number" />
          </Form.Item>
          <Form.Item name="error_rate_alert" label="错误率告警(%)">
            <Input type="number" />
          </Form.Item>
          <Form.Item name="latency_threshold" label="延迟阈值(ms)">
            <Input type="number" />
          </Form.Item>
          <Form.Item name="alert_email" label="告警邮箱">
            <Input />
          </Form.Item>
          <Form.Item name="alert_webhook" label="告警Webhook">
            <Input />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">保存</Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default OpsDashboard