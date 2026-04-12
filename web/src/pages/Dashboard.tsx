import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Row, Col, Card, Table, Tabs, Badge, Button, Spin, Empty } from 'antd'
import {
  DollarOutlined, ApiOutlined, RiseOutlined, TrophyOutlined,
  WalletOutlined, LineChartOutlined, ThunderboltOutlined, ExperimentOutlined,
  CopyOutlined, UserOutlined, TeamOutlined
} from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import {
  getDashboard, getUsageByDay, getUsageByModel, getTokens, getMarketStats,
  getSignInRecords, signIn, SignInRecord,
  getOpsStats, getAdminUsageByModels, User
} from '../services/api'
import { message } from 'antd'

const Dashboard: React.FC = () => {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(true)
  const [isAdmin, setIsAdmin] = useState(false)
  const [dashboardData, setDashboardData] = useState<any>(null)
  const [opsStats, setOpsStats] = useState<any>(null)
  const [usageData, setUsageData] = useState<any[]>([])
  const [modelUsage, setModelUsage] = useState<any[]>([])
  const [tokens, setTokens] = useState<any[]>([])
  const [marketStats, setMarketStats] = useState<any>(null)
  const [activeTab, setActiveTab] = useState('trend')
  const [signInRecords, setSignInRecords] = useState<SignInRecord[]>([])
  const [signingIn, setSigningIn] = useState(false)

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
        // Admin: load ops stats and daily usage
        const [opsRes, modelsRes, usageRes] = await Promise.all([
          getOpsStats(),
          getAdminUsageByModels({}),
          getUsageByDay({ start: new Date(Date.now() - 7 * 86400000).toISOString().split('T')[0] })
        ])
        setOpsStats(opsRes.data?.data || {})
        setModelUsage(modelsRes.data?.data?.models || [])
        setUsageData(usageRes.data?.data || [])
      } else {
        // User: load user dashboard
        const [dashRes, usageRes, modelRes, tokenRes, marketRes, signInRes] = await Promise.all([
          getDashboard(),
          getUsageByDay({ start: new Date(Date.now() - 7 * 86400000).toISOString().split('T')[0] }),
          getUsageByModel(),
          getTokens({ limit: 5 }),
          getMarketStats(),
          getSignInRecords().catch(() => ({ data: { data: [] } }))
        ])

        setDashboardData(dashRes.data.data)
        setUsageData(usageRes.data.data || [])
        setModelUsage(modelRes.data.data?.slice(0, 10) || [])
        setTokens(tokenRes.data.data || [])
        setMarketStats(marketRes.data.data)
        setSignInRecords(signInRes.data.data || [])
      }
    } catch (error) {
      console.error('Failed to load dashboard:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSignIn = async () => {
    try {
      setSigningIn(true)
      await signIn()
      message.success('签到成功！获得 100K 配额奖励')
      const res = await getSignInRecords()
      setSignInRecords(res.data.data || [])
    } catch (error: any) {
      message.error(error?.response?.data?.message || '签到失败')
    } finally {
      setSigningIn(false)
    }
  }

  const formatQuota = (quota: number) => {
    if (quota >= 1000000000) return (quota / 1000000000).toFixed(2) + 'B'
    if (quota >= 1000000) return (quota / 1000000).toFixed(2) + 'M'
    if (quota >= 1000) return (quota / 1000).toFixed(2) + 'K'
    return quota.toString()
  }

  const formatMoney = (quota: number) => {
    return (quota / 7200).toFixed(2)
  }

  // Stat card helper
  const StatCard = ({ title, value, suffix, icon, color, gradient }: any) => (
    <Card
      style={{
        borderRadius: 12,
        border: 'none',
        boxShadow: '0 2px 8px rgba(0,0,0,0.06)'
      }}
      styles={{ body: { padding: '20px 24px' } }}
    >
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div>
          <div style={{ color: '#8c8c8c', fontSize: 14, marginBottom: 8 }}>{title}</div>
          <div style={{ fontSize: 24, fontWeight: 700, color: '#262626' }}>
            {value}
            <span style={{ fontSize: 14, color: '#8c8c8c', marginLeft: 4 }}>{suffix}</span>
          </div>
        </div>
        <div style={{
          width: 48,
          height: 48,
          borderRadius: 12,
          background: gradient || `linear-gradient(135deg, ${color}15, ${color}30)`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center'
        }}>
          <div style={{
            width: 32,
            height: 32,
            borderRadius: 8,
            background: gradient || `linear-gradient(135deg, ${color}, ${color}cc)`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center'
          }}>
            {icon}
          </div>
        </div>
      </div>
    </Card>
  )

  // Usage chart
  const getUsageChartOption = () => ({
    tooltip: { trigger: 'axis' },
    grid: { left: 100, right: 30, top: 30, bottom: 40 },
    xAxis: {
      type: 'category',
      data: usageData.map(d => d.day?.slice(5) || ''),
      axisLine: { lineStyle: { color: '#e8e8e8' } },
      axisLabel: { color: '#8c8c8c' }
    },
    yAxis: {
      type: 'value',
      name: 'Token数',
      nameLocation: 'middle',
      nameGap: 70,
      nameTextStyle: { color: '#8c8c8c', fontSize: 12 },
      axisLine: { show: false },
      splitLine: { lineStyle: { color: '#f5f5f5' } },
      axisLabel: { color: '#8c8c8c' }
    },
    series: [{
      data: usageData.map(d => ({
        value: (d.prompt_tokens || 0) + (d.completion_tokens || 0),
        itemStyle: { color: '#667eea' }
      })),
      type: 'bar',
      barWidth: '60%',
      itemStyle: { borderRadius: [4, 4, 0, 0] },
      areaStyle: { color: 'rgba(102,126,234,0.1)' }
    }]
  })

  // Model usage chart
  const getModelChartOption = () => ({
    tooltip: { trigger: 'axis' },
    grid: { left: 100, right: 30, top: 30, bottom: 80 },
    xAxis: {
      type: 'category',
      data: modelUsage.map(d => d.model_name?.slice(0, 12) || d.model_id?.slice(0, 12) || ''),
      axisLine: { lineStyle: { color: '#e8e8e8' } },
      axisLabel: { color: '#8c8c8c', rotate: 45, fontSize: 10 },
    },
    yAxis: {
      type: 'value',
      name: 'Token数',
      nameLocation: 'middle',
      nameGap: 70,
      nameTextStyle: { color: '#8c8c8c', fontSize: 12 },
      axisLine: { show: false },
      splitLine: { lineStyle: { color: '#f5f5f5' } },
      axisLabel: { color: '#8c8c8c' }
    },
    series: [{
      data: modelUsage.map(d => ({
        value: (d.prompt_tokens || 0) + (d.completion_tokens || 0),
        itemStyle: { color: '#764ba2' }
      })),
      type: 'bar',
      barWidth: '60%',
      itemStyle: { borderRadius: [4, 4, 0, 0] }
    }]
  })

  // Admin tab items (no calendar)
  const adminTabItems = [
    { key: 'trend', label: '用量趋势' },
    { key: 'model', label: '模型排行' },
    { key: 'ranking', label: '调用排行' }
  ]

  // User tab items (with calendar)
  const userTabItems = [
    { key: 'calendar', label: '签到日历' },
    { key: 'trend', label: '用量趋势' },
    { key: 'model', label: '模型排行' },
    { key: 'ranking', label: '调用排行' }
  ]

  // Check if today is signed in
  const todayStr = new Date().toISOString().split('T')[0]
  const isTodaySignedIn = signInRecords.some(r => r.date === todayStr)

  const renderTabContent = () => {
    switch (activeTab) {
      case 'calendar':
        return (
          <Card style={{ borderRadius: 12, border: 'none' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
              <div>
                <span style={{ fontWeight: 600, fontSize: 16 }}>签到日历</span>
                <span style={{ marginLeft: 12, color: '#52c41a', fontSize: 14 }}>总额度: ¥{formatMoney(dashboardData?.user?.quota || 0)}</span>
              </div>
              <Button
                type="primary"
                style={{ borderRadius: 8, background: isTodaySignedIn ? '#d9d9d9' : 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', border: 'none' }}
                onClick={handleSignIn}
                disabled={isTodaySignedIn || signingIn}
                loading={signingIn}
              >
                {isTodaySignedIn ? '今日已签到' : '立即签到'}
              </Button>
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: 8 }}>
              {['日', '一', '二', '三', '四', '五', '六'].map(d => (
                <div key={d} style={{ textAlign: 'center', color: '#8c8c8c', fontSize: 12, padding: 8 }}>{d}</div>
              ))}
              {(() => {
                const now = new Date()
                const year = now.getFullYear()
                const month = now.getMonth()
                const firstDay = new Date(year, month, 1)
                const lastDay = new Date(year, month + 1, 0)
                const startDayOfWeek = firstDay.getDay()
                const daysInMonth = lastDay.getDate()
                const today = now.getDate()

                const cells = []
                for (let i = 0; i < startDayOfWeek; i++) {
                  cells.push(<div key={`empty-${i}`} style={{ aspectRatio: '1' }} />)
                }
                for (let day = 1; day <= daysInMonth; day++) {
                  const date = new Date(year, month, day)
                  const dateStr = date.toISOString().split('T')[0]
                  const isToday = day === today
                  const signData = signInRecords.find((r: SignInRecord) => r.date === dateStr)
                  const isSigned = signData?.status === 'completed'
                  cells.push(
                    <div
                      key={day}
                      style={{
                        aspectRatio: '1',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        borderRadius: 8,
                        background: isToday ? '#667eea15' : isSigned ? '#52c41a15' : '#f5f5f5',
                        border: isToday ? '2px solid #667eea' : 'none',
                        fontSize: 12,
                        color: isSigned ? '#52c41a' : '#8c8c8c',
                        fontWeight: isSigned ? 600 : 400
                      }}
                    >
                      {day}
                    </div>
                  )
                }
                return cells
              })()}
            </div>
          </Card>
        )
      case 'trend':
        return (
          <Card style={{ borderRadius: 12, border: 'none' }}>
            <div style={{ fontWeight: 600, fontSize: 16, marginBottom: 16 }}>最近7天用量趋势</div>
            {usageData.length > 0 ? (
              <ReactECharts option={getUsageChartOption()} style={{ height: 300 }} />
            ) : (
              <Empty description="暂无用量数据" style={{ padding: 60 }} />
            )}
          </Card>
        )
      case 'model':
        return (
          <Card style={{ borderRadius: 12, border: 'none' }}>
            <div style={{ fontWeight: 600, fontSize: 16, marginBottom: 16 }}>模型用量排行</div>
            {modelUsage.length > 0 ? (
              <ReactECharts option={getModelChartOption()} style={{ height: 300 }} />
            ) : (
              <Empty description="暂无模型用量" style={{ padding: 60 }} />
            )}
          </Card>
        )
      case 'ranking':
        return (
          <Card style={{ borderRadius: 12, border: 'none' }}>
            <div style={{ fontWeight: 600, fontSize: 16, marginBottom: 16 }}>Token消耗排行</div>
            <Table
              dataSource={modelUsage}
              rowKey="model_name"
              pagination={false}
              columns={[
                { title: '排名', width: 60, render: (_, __, i) => <Badge count={i + 1} style={{ backgroundColor: '#667eea' }} /> },
                { title: '模型', dataIndex: 'model_name', ellipsis: true },
                { title: 'Prompt Tokens', dataIndex: 'prompt_tokens', render: (v) => formatQuota(v) },
                { title: 'Completion Tokens', dataIndex: 'completion_tokens', render: (v) => formatQuota(v) },
                {
                  title: '占比',
                  width: 100,
                  render: (_, record) => {
                    const total = modelUsage.reduce((sum: number, r: any) => sum + (r.prompt_tokens || 0) + (r.completion_tokens || 0), 0)
                    const percent = total > 0 ? ((record.prompt_tokens + record.completion_tokens) / total * 100).toFixed(1) : '0'
                    return `${percent}%`
                  }
                }
              ]}
            />
          </Card>
        )
      default:
        return null
    }
  }

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 400 }}>
        <Spin size="large" />
      </div>
    )
  }

  const user = dashboardData?.user || {}

  // Admin dashboard
  if (isAdmin) {
    return (
      <div>
        {/* Admin Stat Cards */}
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <StatCard
              title="今日收入"
              value={`¥${formatMoney(opsStats?.today_revenue || 0)}`}
              suffix=""
              icon={<DollarOutlined style={{ color: '#fff', fontSize: 16 }} />}
              color="#667eea"
            />
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StatCard
              title="今日用量Token"
              value={formatQuota(opsStats?.today_usage_tokens || 0)}
              suffix=""
              icon={<ApiOutlined style={{ color: '#fff', fontSize: 16 }} />}
              color="#52c41a"
            />
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StatCard
              title="活跃用户"
              value={opsStats?.active_users || 0}
              suffix="人"
              icon={<UserOutlined style={{ color: '#fff', fontSize: 16 }} />}
              color="#ff4d4f"
            />
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StatCard
              title="渠道健康率"
              value={`${(opsStats?.channel_health_rate || 0).toFixed(1)}`}
              suffix="%"
              icon={<TrophyOutlined style={{ color: '#fff', fontSize: 16 }} />}
              color="#faad14"
            />
          </Col>
        </Row>

        {/* Second row stats */}
        <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
          <Col xs={24} sm={12} lg={6}>
            <StatCard
              title="总用户数"
              value={opsStats?.total_users || 0}
              suffix="人"
              icon={<TeamOutlined style={{ color: '#fff', fontSize: 16 }} />}
              color="#13c2c2"
            />
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StatCard
              title="总渠道数"
              value={opsStats?.total_channels || 0}
              suffix="个"
              icon={<ApiOutlined style={{ color: '#fff', fontSize: 16 }} />}
              color="#722ed1"
            />
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StatCard
              title="总Token消耗"
              value={formatQuota(opsStats?.total_tokens || 0)}
              suffix=""
              icon={<RiseOutlined style={{ color: '#fff', fontSize: 16 }} />}
              color="#eb2f96"
            />
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <StatCard
              title="总配额"
              value={formatQuota(opsStats?.total_quota || 0)}
              suffix=""
              icon={<WalletOutlined style={{ color: '#fff', fontSize: 16 }} />}
              color="#fa8c16"
            />
          </Col>
        </Row>

        {/* Main Content */}
        <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
          <Col xs={24} lg={16}>
            <Card
              style={{ borderRadius: 12, border: 'none' }}
              styles={{ body: { padding: '16px 20px' } }}
            >
              <Tabs
                activeKey={activeTab}
                onChange={setActiveTab}
                items={adminTabItems}
                tabBarStyle={{ marginBottom: 16 }}
              />
              {renderTabContent()}
            </Card>
          </Col>

          {/* Quick Actions */}
          <Col xs={24} lg={8}>
            <Card
              title="快捷操作"
              style={{ borderRadius: 12, border: 'none' }}
              styles={{ body: { padding: 16 } }}
            >
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 8 }}>
                <Button icon={<LineChartOutlined />} block style={{ borderRadius: 8, height: 44 }} onClick={() => navigate('/usage')}>
                  用量明细
                </Button>
                <Button icon={<ThunderboltOutlined />} block style={{ borderRadius: 8, height: 44 }} onClick={() => navigate('/keys')}>
                  API Keys
                </Button>
                <Button icon={<ApiOutlined />} block style={{ borderRadius: 8, height: 44 }} onClick={() => navigate('/ops')}>
                  运营管理
                </Button>
                <Button icon={<TeamOutlined />} block style={{ borderRadius: 8, height: 44 }} onClick={() => navigate('/teams')}>
                  团队管理
                </Button>
              </div>
            </Card>

            {/* Top Models Card */}
            <Card
              title="用量Top模型"
              style={{ borderRadius: 12, border: 'none', marginTop: 16 }}
              styles={{ body: { padding: 16 } }}
            >
              {modelUsage.slice(0, 5).map((m: any, i: number) => (
                <div key={m.model_name} style={{ display: 'flex', justifyContent: 'space-between', padding: '8px 0', borderBottom: i < 4 ? '1px solid #f5f5f5' : 'none' }}>
                  <span style={{ fontWeight: 500 }}>{m.model_name}</span>
                  <span style={{ color: '#8c8c8c' }}>{formatQuota((m.prompt_tokens || 0) + (m.completion_tokens || 0))}</span>
                </div>
              ))}
              {modelUsage.length === 0 && <Empty description="暂无数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />}
            </Card>
          </Col>
        </Row>
      </div>
    )
  }

  // User dashboard
  return (
    <div>
      {/* Stat Cards */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="账户余额"
            value={`¥${formatMoney(user.quota || 0)}`}
            suffix=""
            icon={<DollarOutlined style={{ color: '#fff', fontSize: 16 }} />}
            color="#667eea"
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="本月请求数"
            value={dashboardData?.usage?.total_requests?.toLocaleString() || '0'}
            suffix="次"
            icon={<ApiOutlined style={{ color: '#fff', fontSize: 16 }} />}
            color="#52c41a"
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="本月Token消耗"
            value={formatQuota(dashboardData?.usage?.total_tokens || 0)}
            suffix=""
            icon={<RiseOutlined style={{ color: '#fff', fontSize: 16 }} />}
            color="#ff4d4f"
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="可用模型"
            value={marketStats?.total_models || 0}
            suffix="个"
            icon={<TrophyOutlined style={{ color: '#fff', fontSize: 16 }} />}
            color="#faad14"
          />
        </Col>
      </Row>

      {/* Main Content */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={16}>
          <Card
            style={{ borderRadius: 12, border: 'none' }}
            styles={{ body: { padding: '16px 20px' } }}
          >
            <Tabs
              activeKey={activeTab}
              onChange={setActiveTab}
              items={userTabItems}
              tabBarStyle={{ marginBottom: 16 }}
            />
            {renderTabContent()}
          </Card>
        </Col>

        {/* API Quick Access */}
        <Col xs={24} lg={8}>
          <Card
            title="API快速访问"
            style={{ borderRadius: 12, border: 'none' }}
            styles={{ body: { padding: 16 } }}
            extra={<Button size="small" type="link" onClick={() => navigate('/keys')}>查看全部</Button>}
          >
            {tokens.slice(0, 3).map((token: any) => (
              <div
                key={token.id}
                style={{
                  padding: '12px',
                  background: '#f5f7fa',
                  borderRadius: 8,
                  marginBottom: 8
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                  <span style={{ fontWeight: 500, fontSize: 14 }}>{token.name}</span>
                  <Button
                    type="text"
                    size="small"
                    icon={<CopyOutlined />}
                    style={{ color: '#667eea' }}
                    onClick={() => {
                      navigator.clipboard.writeText(`sk-${token.key}`)
                    }}
                  >
                    复制
                  </Button>
                </div>
                <div style={{ fontSize: 12, color: '#8c8c8c', fontFamily: 'monospace' }}>
                  sk-{token.key?.slice(0, 20)}...
                </div>
              </div>
            ))}

            {tokens.length === 0 && (
              <Empty description="暂无API Key" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            )}
          </Card>

          {/* Quick Actions */}
          <Card
            title="快捷操作"
            style={{ borderRadius: 12, border: 'none', marginTop: 16 }}
            styles={{ body: { padding: 16 } }}
          >
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 8 }}>
              <Button icon={<WalletOutlined />} block style={{ borderRadius: 8, height: 44 }} onClick={() => navigate('/topup')}>
                充值
              </Button>
              <Button icon={<LineChartOutlined />} block style={{ borderRadius: 8, height: 44 }} onClick={() => navigate('/usage')}>
                用量明细
              </Button>
              <Button icon={<ThunderboltOutlined />} block style={{ borderRadius: 8, height: 44 }} onClick={() => navigate('/keys')}>
                创建Key
              </Button>
              <Button icon={<ExperimentOutlined />} block style={{ borderRadius: 8, height: 44 }} onClick={() => navigate('/market')}>
                模型试用
              </Button>
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Dashboard
