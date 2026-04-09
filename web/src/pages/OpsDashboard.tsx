import { useState, useEffect } from 'react'
import { Row, Col, Card, Table, Statistic, Spin, Progress, Tag, Tabs, Button, Space, Modal, Form, Input, InputNumber, Select, Popconfirm, message, Divider } from 'antd'
import { DollarOutlined, UserOutlined, ApiOutlined, RiseOutlined, SafetyCertificateOutlined, AlertOutlined, DashboardOutlined, LineChartOutlined, PlusOutlined, EditOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import { getOpsStats, getChannelHealth, getSystemHealth, getAlertConfig, updateAlertConfig, exportReport, getOpsUsers, updateUser, getChannels, createChannel, updateChannel, deleteChannel, getProviders, OpsStats, ChannelHealth, SystemHealth, AlertConfig, Channel, Provider } from '../services/api'

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
  const [users, setUsers] = useState<any[]>([])
  const [usersLoading, setUsersLoading] = useState(false)
  const [usersTotal, setUsersTotal] = useState(0)
  const [channels, setChannels] = useState<Channel[]>([])
  const [channelsLoading, setChannelsLoading] = useState(false)
  const [channelsTotal, setChannelsTotal] = useState(0)
  const [channelsOffset, setChannelsOffset] = useState(0)
  const [channelModalVisible, setChannelModalVisible] = useState(false)
  const [editingChannel, setEditingChannel] = useState<Channel | null>(null)
  const [channelForm] = Form.useForm()
  const [providers, setProviders] = useState<Provider[]>([])
  const [editUserModalVisible, setEditUserModalVisible] = useState(false)
  const [editingUser, setEditingUser] = useState<any>(null)
  const [editUserForm] = Form.useForm()

  useEffect(() => {
    loadData()
    loadUsers()
    loadChannels()
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

  const loadUsers = async (offset = 0, limit = 20) => {
    try {
      setUsersLoading(true)
      const res = await getOpsUsers({ limit, offset })
      if (res.data.success) {
        setUsers(res.data.data.users || [])
        setUsersTotal(res.data.data.total || 0)
      }
    } catch (error) {
      console.error('Failed to load users:', error)
    } finally {
      setUsersLoading(false)
    }
  }

  const handleEditUser = (record: any) => {
    setEditingUser(record)
    editUserForm.setFieldsValue({
      group: record.group,
      quota: record.quota,
      role: record.role,
      status: record.status,
    })
    setEditUserModalVisible(true)
  }

  const handleUpdateUser = async () => {
    try {
      const values = await editUserForm.validateFields()
      const res = await updateUser(editingUser.id, values)
      if (res.data.success) {
        message.success('更新成功')
        setEditUserModalVisible(false)
        loadUsers()
      } else {
        message.error(res.data.message || '更新失败')
      }
    } catch (error) {
      message.error('更新失败')
    }
  }

  const loadChannels = async (offset: number = 0, limit: number = 10) => {
    try {
      setChannelsLoading(true)
      setChannelsOffset(offset)
      const res = await getChannels({ offset, limit })
      if (res.data.success) {
        setChannels(res.data.data || [])
        setChannelsTotal(res.data.total || 0)
      }
    } catch (error) {
      console.error('Failed to load channels:', error)
    } finally {
      setChannelsLoading(false)
    }
  }

  const loadProviders = async () => {
    try {
      const res = await getProviders()
      if (res.data.success) {
        setProviders(res.data.data || [])
      }
    } catch (error) {
      console.error('Failed to load providers:', error)
    }
  }

  const handleCreateChannel = () => {
    setEditingChannel(null)
    channelForm.resetFields()
    loadProviders()
    setChannelModalVisible(true)
  }

  const handleEditChannel = (record: Channel) => {
    setEditingChannel(record)
    // Parse models string and model_mapping JSON into modelMappings array format for the form
    let modelMappings: { client: string; upstream: string }[] = []
    if (record.models && record.models.trim()) {
      const modelList = record.models.split(',').filter(Boolean)
      // Parse model_mapping JSON to get upstream names
      let mappingObj: Record<string, string> = {}
      try {
        if (record.model_mapping) {
          mappingObj = JSON.parse(record.model_mapping)
        }
      } catch (e) {
        console.error('Failed to parse model_mapping:', e)
      }
      modelMappings = modelList.map((client: string) => ({
        client: client.trim(),
        upstream: mappingObj[client.trim()] || ''
      }))
    }
    channelForm.setFieldsValue({ ...record, modelMappings })
    // 如果是编辑模式，用占位符触发 Input.Password 显示掩码
    channelForm.setFieldsValue({ key: '********' })
    loadProviders()
    setChannelModalVisible(true)
  }

  const handleDeleteChannel = async (id: number) => {
    try {
      const res = await deleteChannel(id)
      if (res.data.success) {
        message.success('删除成功')
        loadChannels()
      } else {
        message.error(res.data.message || '删除失败')
      }
    } catch (error) {
      message.error('删除失败')
    }
  }

  const handleChannelSubmit = async () => {
    try {
      const values = await channelForm.validateFields()

      // Process model mappings: extract client models and build mapping JSON
      let models = ''
      let modelMapping = '{}'
      if (values.modelMappings && values.modelMappings.length > 0) {
        const clientModels = values.modelMappings.map((m: { client: string; upstream: string }) => m.client).filter(Boolean)
        models = clientModels.join(',')
        const mappingObj: Record<string, string> = {}
        values.modelMappings.forEach((m: { client: string; upstream: string }) => {
          if (m.client && m.upstream) {
            mappingObj[m.client] = m.upstream
          }
        })
        modelMapping = JSON.stringify(mappingObj)
      }

      const payload = {
        ...values,
        models,
        model_mapping: modelMapping,
      }
      delete payload.modelMappings

      // 如果是编辑模式且用户没有输入新 key，则不发送 key 字段，避免覆盖原有值
      // 如果 key 是占位符，说明用户没改，删除不发送
      if (editingChannel && (!values.key || values.key === '********')) {
        delete payload.key
      }

      if (editingChannel) {
        const res = await updateChannel(editingChannel.id, payload)
        if (res.data.success) {
          message.success('更新成功')
          setChannelModalVisible(false)
          loadChannels(channelsOffset, 10)
        } else {
          message.error(res.data.message || '更新失败')
        }
      } else {
        const res = await createChannel(payload)
        if (res.data.success) {
          message.success('创建成功')
          setChannelModalVisible(false)
          loadChannels()
        } else {
          message.error(res.data.message || '创建失败')
        }
      }
    } catch (error) {
      message.error(editingChannel ? '更新失败' : '创建失败')
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
    { title: 'Provider', dataIndex: 'group', key: 'group', width: 100 },
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
              pagination={{ pageSize: 10, total: channelHealth.length, showTotal: (t) => `共 ${t} 条` }}
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

        <TabPane tab={<span><ApiOutlined /> 渠道管理</span>} key="4">
          <Card
            title="渠道列表"
            extra={<Button type="primary" icon={<PlusOutlined />} onClick={handleCreateChannel}>添加渠道</Button>}
          >
            <Table
              dataSource={channels}
              columns={[
                { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
                { title: '名称', dataIndex: 'name', key: 'name' },
                { title: '类型', dataIndex: 'type', key: 'type', render: (v: number) => ['OpenAI', 'Azure', 'Anthropic', 'Google', '自定义'][v] || '未知' },
                { title: 'Group', dataIndex: 'group', key: 'group', width: 100 },
                { title: 'Base URL', dataIndex: 'base_url', key: 'base_url', ellipsis: true },
                { title: '优先级', dataIndex: 'priority', key: 'priority', width: 80 },
                { title: '状态', dataIndex: 'status', key: 'status', render: (v: number) => v === 1 ? <Tag color="green">启用</Tag> : v === 2 ? <Tag color="red">禁用</Tag> : <Tag>未知</Tag> },
                {
                  title: '操作',
                  key: 'action',
                  width: 150,
                  render: (_: any, record: Channel) => (
                    <Space>
                      <Button type="link" size="small" onClick={() => handleEditChannel(record)}>编辑</Button>
                      <Popconfirm title="确定删除？" onConfirm={() => handleDeleteChannel(record.id)}>
                        <Button type="link" size="small" danger>删除</Button>
                      </Popconfirm>
                    </Space>
                  )
                }
              ]}
              rowKey="id"
              loading={channelsLoading}
              pagination={{ pageSize: 10, total: channelsTotal, showTotal: (t) => `共 ${t} 条` }}
              onChange={(pagination) => loadChannels(((pagination.current || 1) - 1) * 10, 10)}
            />
          </Card>
        </TabPane>

        <TabPane tab={<span><UserOutlined /> 用户管理</span>} key="5">
          <Card title="用户列表">
            <Table
              dataSource={users}
              columns={[
                { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
                { title: '用户名', dataIndex: 'username', key: 'username', render: (v: string, r: any) => r.display_name || v },
                { title: '邮箱', dataIndex: 'email', key: 'email' },
                { title: 'Group', dataIndex: 'group', key: 'group', width: 100 },
                { title: '角色', dataIndex: 'role', key: 'role', render: (v: number) => v === 100 ? '超级管理员' : v === 10 ? '管理员' : '普通用户' },
                { title: '状态', dataIndex: 'status', key: 'status', render: (v: number) => v === 1 ? '正常' : v === 2 ? '禁用' : '已删除' },
                { title: '额度', dataIndex: 'quota', key: 'quota', render: (v: number) => v.toLocaleString() },
                { title: '已用额度', dataIndex: 'used_quota', key: 'used_quota', render: (v: number) => v.toLocaleString() },
                { title: '请求次数', dataIndex: 'request_count', key: 'request_count' },
                {
                  title: '操作',
                  key: 'action',
                  width: 100,
                  render: (_: any, record: any) => (
                    <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEditUser(record)}>
                      编辑
                    </Button>
                  )
                },
              ]}
              rowKey="id"
              loading={usersLoading}
              pagination={{ pageSize: 20, total: usersTotal, showTotal: (t) => `共 ${t} 条` }}
              onChange={(pagination) => loadUsers(((pagination.current || 1) - 1) * 20, 20)}
            />
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

      <Modal
        title={editingChannel ? '编辑渠道' : '添加渠道'}
        open={channelModalVisible}
        onCancel={() => setChannelModalVisible(false)}
        onOk={handleChannelSubmit}
        okText="保存"
        cancelText="取消"
        width={700}
      >
        <Form form={channelForm} layout="vertical">
          <Form.Item name="name" label="渠道名称" rules={[{ required: true, message: '请输入渠道名称' }]}>
            <Input placeholder="如: BEDI-集群1" />
          </Form.Item>
          <Form.Item name="type" label="渠道类型" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 50, label: 'OpenAI兼容' },
                { value: 14, label: 'Anthropic' },
                { value: 8, label: '自定义' },
              ]}
              placeholder="选择渠道类型"
            />
          </Form.Item>
          <Form.Item name="base_url" label="Base URL" rules={[{ required: true, message: '请输入Base URL' }]}>
            <Input placeholder="如: https://api.bedicloud.net/v1" />
          </Form.Item>
          <Form.Item name="group" label="Provider" rules={[{ required: true, message: '请选择Provider' }]}>
            <Select
              showSearch
              allowClear
              placeholder="选择 Provider"
              options={providers.map(p => ({ value: p.code, label: p.name }))}
            />
          </Form.Item>
          <Form.Item
            name="key"
            label="API Key"
            rules={editingChannel ? [] : [{ required: true, message: '请输入API Key' }]}
          >
            <Input.Password placeholder="请输入API Key" />
          </Form.Item>

          <Divider>支持的模型</Divider>

          <Form.List name="modelMappings">
            {(fields, { add, remove }) => (
              <>
                {fields.map(({ key, name, ...restField }) => (
                  <Space key={key} style={{ display: 'flex', marginBottom: 8 }} align="baseline">
                    <Form.Item
                      {...restField}
                      name={[name, 'client']}
                      label="客户端模型名"
                      rules={[{ required: true, message: '请输入客户端模型名' }]}
                    >
                      <Input placeholder="如: glm-4" style={{ width: 200 }} />
                    </Form.Item>
                    <span>→</span>
                    <Form.Item
                      {...restField}
                      name={[name, 'upstream']}
                      label="上游实际名称"
                      rules={[{ required: true, message: '请输入上游名称' }]}
                    >
                      <Input placeholder="如: ZhipuAI/GLM-4" style={{ width: 200 }} />
                    </Form.Item>
                    <Button type="link" danger onClick={() => remove(name)}>删除</Button>
                  </Space>
                ))}
                <Form.Item>
                  <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                    添加模型映射
                  </Button>
                </Form.Item>
              </>
            )}
          </Form.List>

          <Divider />

          <Form.Item name="priority" label="优先级">
            <InputNumber style={{ width: '100%' }} placeholder="数值越大越优先" min={0} />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select
              options={[
                { value: 1, label: '启用' },
                { value: 2, label: '禁用' },
              ]}
              placeholder="选择状态"
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="编辑用户"
        open={editUserModalVisible}
        onCancel={() => setEditUserModalVisible(false)}
        onOk={handleUpdateUser}
        okText="保存"
        cancelText="取消"
      >
        <Form form={editUserForm} layout="vertical">
          <Form.Item name="group" label="分组（Group）">
            <Input placeholder="如: bedi, default" />
          </Form.Item>
          <Form.Item name="quota" label="额度">
            <InputNumber style={{ width: '100%' }} placeholder="-1 = 无限制，0 = 无额度" min={-1} />
          </Form.Item>
          <Form.Item name="role" label="角色">
            <Select
              options={[
                { value: 1, label: '普通用户' },
                { value: 10, label: '管理员' },
                { value: 100, label: '超级管理员' },
              ]}
            />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select
              options={[
                { value: 1, label: '正常' },
                { value: 2, label: '禁用' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default OpsDashboard