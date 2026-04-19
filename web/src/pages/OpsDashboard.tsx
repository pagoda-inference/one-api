import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Row, Col, Card, Table, Statistic, Spin, Progress, Tag, Tabs, Button, Space, Modal, Form, Input, InputNumber, Select, Popconfirm, message, Divider, Switch } from 'antd'
import { DollarOutlined, UserOutlined, ApiOutlined, RiseOutlined, SafetyCertificateOutlined, DashboardOutlined, LineChartOutlined, PlusOutlined, EditOutlined, SaveOutlined, SettingOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import { getOpsStats, getChannelHealth, getAlertConfig, updateAlertConfig, exportReport, getOpsUsers, updateUser, getChannels, createChannel, updateChannel, deleteChannel, getProviders, getOptions, updateOption, getChannel, OpsStats, ChannelHealth, AlertConfig, Channel, Provider } from '../services/api'
import { useTranslation } from 'react-i18next'

const { TabPane } = Tabs

const OpsDashboard: React.FC = () => {
  const { t } = useTranslation()
  const [searchParams, setSearchParams] = useSearchParams()
  const initialTab = searchParams.get('tab') || '1'
  const [loading, setLoading] = useState(true)
  const [stats, setStats] = useState<OpsStats | null>(null)
  const [channelHealth, setChannelHealth] = useState<ChannelHealth[]>([])
  const [alertConfig, setAlertConfig] = useState<AlertConfig | null>(null)
  const [overallHealth, setOverallHealth] = useState(0)
  const [enabledCount, setEnabledCount] = useState(0)
  const [totalChannels, setTotalChannels] = useState(0)
  const [users, setUsers] = useState<any[]>([])
  const [usersLoading, setUsersLoading] = useState(false)
  const [usersTotal, setUsersTotal] = useState(0)
  const [usersOffset, setUsersOffset] = useState(0)
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
  const [activeTab, setActiveTab] = useState(initialTab)
  const [settingForm] = Form.useForm()
  const [settingSaving, setSettingSaving] = useState(false)
  const [originalKey, setOriginalKey] = useState('')
  const [channelConfig, setChannelConfig] = useState('')
  const [, setSettings] = useState({
    QuotaForNewUser: 0,
    QuotaForInviter: 0,
    QuotaForInvitee: 0,
    PreConsumedQuota: 500,
    GroupRatio: '',
    QuotaRemindThreshold: 0,
    ChannelDisableThreshold: 0,
    AutomaticDisableChannelEnabled: false,
    AutomaticEnableChannelEnabled: false,
    LogConsumeEnabled: false,
    DisplayInCurrencyEnabled: false,
    DisplayTokenStatEnabled: false,
    RetryTimes: 3,
  })

  useEffect(() => {
    loadData()
    loadUsers()
    loadChannels()
    loadSettings()
  }, [])

  const handleTabChange = (key: string) => {
    setActiveTab(key)
    setSearchParams({ tab: key })
  }

  const loadData = async () => {
    try {
      setLoading(true)
      const [statsRes, healthRes, alertRes] = await Promise.all([
        getOpsStats(),
        getChannelHealth(),
        getAlertConfig()
      ])

      setStats(statsRes.data.data || null)
      setChannelHealth(healthRes.data?.data?.channels || [])
      setOverallHealth(healthRes.data?.data?.overall_health || 0)
      setEnabledCount(healthRes.data?.data?.enabled_count || 0)
      setTotalChannels(healthRes.data?.data?.total_count || 0)
      setAlertConfig(alertRes.data.data || null)
    } catch (error) {
      console.error('Failed to load ops data:', error)
    } finally {
      setLoading(false)
    }
  }

  const loadUsers = async (offset = 0, limit = 20) => {
    try {
      setUsersLoading(true)
      setUsersOffset(offset)
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
        loadUsers(usersOffset, 20)
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

  const handleEditChannel = async (record: Channel) => {
    setEditingChannel(record)
    // 获取完整渠道信息（包含真实 key）
    const res = await getChannel(record.id)
    if (!res.data.success) {
      message.error(t('ops.get_channel_failed'))
      return
    }
    const fullRecord = res.data.data
    // Parse models string and model_mapping JSON into modelMappings array format for the form
    let modelMappings: { client: string; upstream: string }[] = []
    if (fullRecord.models && fullRecord.models.trim()) {
      const modelList = fullRecord.models.split(',').filter(Boolean)
      // Parse model_mapping JSON to get upstream names
      let mappingObj: Record<string, string> = {}
      try {
        if (fullRecord.model_mapping) {
          mappingObj = JSON.parse(fullRecord.model_mapping)
        }
      } catch (e) {
        console.error('Failed to parse model_mapping:', e)
      }
      modelMappings = modelList.map((client: string) => ({
        client: client.trim(),
        upstream: mappingObj[client.trim()] || ''
      }))
    }
    // Parse config JSON to get hide_upstream_model setting
    let hideUpstreamModel = false
    try {
      if (fullRecord.config) {
        const configObj = JSON.parse(fullRecord.config)
        hideUpstreamModel = configObj.hide_upstream_model || false
      }
    } catch (e) {
      console.error('Failed to parse config:', e)
    }
    channelForm.setFieldsValue({ ...fullRecord, modelMappings, hide_upstream_model: hideUpstreamModel })
    // 保存原始 key 和 config 用于提交时比较
    setOriginalKey(fullRecord.key || '')
    setChannelConfig(fullRecord.config || '{}')
    console.log('Loaded channel config:', fullRecord.config)
    console.log('Parsed hide_upstream_model:', hideUpstreamModel)
    loadProviders()
    setChannelModalVisible(true)
  }

  const handleDeleteChannel = async (id: number) => {
    try {
      const res = await deleteChannel(id)
      if (res.data.success) {
        message.success(t('ops.delete_success'))
        loadChannels()
      } else {
        message.error(res.data.message || t('ops.delete_failed'))
      }
    } catch (error) {
      message.error(t('ops.delete_failed'))
    }
  }

  const handleChannelSubmit = async () => {
    try {
      const values = await channelForm.validateFields()
      console.log('channelConfig state:', channelConfig)
      console.log('hide_upstream_model from form:', values.hide_upstream_model)

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

      // Build config JSON - merge with existing config to preserve other fields
      let configObj: Record<string, any> = {}
      try {
        if (channelConfig) {
          configObj = JSON.parse(channelConfig)
        }
      } catch (e) {
        console.error('Failed to parse config:', e)
      }
      if (values.hide_upstream_model !== undefined) {
        configObj.hide_upstream_model = values.hide_upstream_model
      }
      const config = JSON.stringify(configObj)
      console.log('Final config being sent:', config)

      const payload = {
        ...values,
        models,
        model_mapping: modelMapping,
        config,
      }
      delete payload.modelMappings
      delete payload.hide_upstream_model

      // 如果是编辑模式且用户没有输入新 key，则不发送 key 字段，避免覆盖原有值
      if (editingChannel && (values.key === originalKey || !values.key)) {
        delete payload.key
      }

      if (editingChannel) {
        const res = await updateChannel(editingChannel.id, payload)
        if (res.data.success) {
          message.success(t('ops.update_success'))
          setChannelModalVisible(false)
          loadChannels(channelsOffset, 10)
        } else {
          message.error(res.data.message || t('ops.update_failed'))
        }
      } else {
        const res = await createChannel(payload)
        if (res.data.success) {
          message.success(t('ops.create_success'))
          setChannelModalVisible(false)
          loadChannels()
        } else {
          message.error(res.data.message || t('ops.create_failed'))
        }
      }
    } catch (error) {
      message.error(editingChannel ? t('ops.update_failed') : t('ops.create_failed'))
    }
  }

  const handleExport = async (type: string) => {
    try {
      const res = await exportReport({ type })
      message.success(`${t('ops.report_generated')} ${res.data.data.report_url}`)
    } catch (error) {
      message.error(t('ops.export_failed'))
    }
  }

  const loadSettings = async () => {
    try {
      const res = await getOptions()
      if (res.data.success) {
        const data = res.data.data || []
        const newSettings: any = {}

        // Map OptionMap keys to form field names
        const keyMap: Record<string, string> = {
          'SysMaxConcurrentRequests': 'max_concurrent_requests',
          'SysRequestQueueTimeout': 'request_queue_timeout',
          'SysHealthCheckInterval': 'health_check_interval',
          'SysHealthCheckFailThreshold': 'health_check_fail_threshold',
          'SysCircuitBreakerThreshold': 'circuit_breaker_threshold',
          'SysCircuitBreakerTimeout': 'circuit_breaker_timeout',
          'SysRelayTimeout': 'relay_timeout',
        }

        data.forEach((item: any) => {
          if (item.key === 'GroupRatio') {
            try {
              newSettings[item.key] = JSON.stringify(JSON.parse(item.value), null, 2)
            } catch {
              newSettings[item.key] = item.value || ''
            }
          } else if (item.key === 'AutomaticDisableChannelEnabled' || item.key === 'AutomaticEnableChannelEnabled' || item.key === 'LogConsumeEnabled' || item.key === 'DisplayInCurrencyEnabled' || item.key === 'DisplayTokenStatEnabled') {
            newSettings[item.key] = item.value === 'true' || item.value === '1'
          } else if (item.key === 'AlertEnabled') {
            newSettings.alert_enabled = item.value === 'true' || item.value === '1'
          } else if (item.key === 'AlertEmail') {
            newSettings.alert_email = item.value
          } else if (item.key === 'AlertWebhook') {
            newSettings.alert_webhook = item.value
          } else if (keyMap[item.key]) {
            // Map backend keys to form field names
            newSettings[keyMap[item.key]] = item.value ? parseFloat(item.value) : 0
          } else if (item.key !== 'ModelRatio' && item.key !== 'CompletionRatio') {
            newSettings[item.key] = item.value ? parseFloat(item.value) : 0
          }
        })
        setSettings((prev: any) => ({ ...prev, ...newSettings }))
        settingForm.setFieldsValue(newSettings)
      }
    } catch (error) {
      console.error('Failed to load options:', error)
    }
  }

  const handleSaveSettings = async () => {
    try {
      setSettingSaving(true)
      const values = settingForm.getFieldsValue()

      // Prepare alert config data - use exact keys expected by backend
      const alertData: any = {
        alert_email: values.alert_email,
        alert_webhook: values.alert_webhook,
        alert_enabled: values.alert_enabled,
        max_concurrent_requests: values.max_concurrent_requests,
        request_queue_timeout: values.request_queue_timeout,
        health_check_interval: values.health_check_interval,
        health_check_fail_threshold: values.health_check_fail_threshold,
        circuit_breaker_threshold: values.circuit_breaker_threshold,
        circuit_breaker_timeout: values.circuit_breaker_timeout,
        relay_timeout: values.relay_timeout,
      }

      // Update alert and system config via updateAlertConfig
      await updateAlertConfig(alertData)

      // Update other settings (GroupRatio, channel settings, etc.) via updateOption
      const otherKeys = ['GroupRatio', 'AutomaticDisableChannelEnabled', 'AutomaticEnableChannelEnabled', 'ChannelDisableThreshold', 'DisplayInCurrencyEnabled', 'DisplayTokenStatEnabled', 'LogConsumeEnabled', 'RetryTimes', 'QuotaForNewUser', 'QuotaForInviter', 'QuotaForInvitee', 'PreConsumedQuota', 'QuotaRemindThreshold']
      const otherPromises = otherKeys
        .filter(key => values[key] !== undefined)
        .map(key => {
          let finalValue: any = values[key]
          if (typeof values[key] === 'boolean') {
            finalValue = values[key] ? 'true' : 'false'
          } else if (typeof values[key] === 'object') {
            finalValue = JSON.stringify(values[key])
          } else {
            finalValue = String(values[key])
          }
          return updateOption(key, finalValue)
        })

      await Promise.all(otherPromises)
      message.success(t('ops.settings_saved_success'))
      loadSettings()
      loadData() // Refresh alert config display
    } catch (error: any) {
      message.error(error.message || t('ops.save_failed'))
      console.error('Save error:', error)
    } finally {
      setSettingSaving(false)
    }
  }

  const verifyJSON = (str: string) => {
    if (!str || str.trim() === '') return true
    try {
      JSON.parse(str)
      return true
    } catch {
      return false
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
      title: { text: t('ops.daily_revenue_trend'), left: 'center' },
      tooltip: { trigger: 'axis' },
      legend: { data: [t('ops.revenue'), t('ops.topup_amount')], bottom: 0 },
      xAxis: { type: 'category', data: days },
      yAxis: { type: 'value', name: t('ops.amount_yuan') },
      series: [
        { name: t('ops.revenue'), data: revenues, type: 'bar', itemStyle: { color: '#52c41a' } },
        { name: t('ops.topup_amount'), data: topups, type: 'bar', itemStyle: { color: '#1890ff' } }
      ]
    }
  }

  const channelColumns = [
    { title: t('ops.id'), dataIndex: 'id', key: 'id', width: 60 },
    { title: t('ops.channel_name'), dataIndex: 'name', key: 'name' },
    { title: t('ops.channel_type'), dataIndex: 'type_name', key: 'type_name' },
    { title: 'Provider', dataIndex: 'provider', key: 'provider', width: 100 },
    { title: 'Base URL', dataIndex: 'base_url', key: 'base_url', ellipsis: true },
    {
      title: t('ops.success_rate'),
      dataIndex: 'success_rate',
      key: 'success_rate',
      render: (v: number) => <Progress percent={v} size="small" status={v < 80 ? 'exception' : undefined} />
    },
    { title: t('ops.avg_latency'), dataIndex: 'avg_latency', key: 'avg_latency', render: (v: number) => `${v}ms` },
    { title: t('ops.priority'), dataIndex: 'priority', key: 'priority' },
    {
      title: t('ops.status'),
      dataIndex: 'is_enabled',
      key: 'is_enabled',
      render: (v: boolean) => <Tag color={v ? 'green' : 'red'}>{v ? t('ops.enabled') : t('ops.disabled')}</Tag>
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
              title={t('ops.today_revenue')}
              value={stats?.today_revenue || 0}
              prefix={<DollarOutlined />}
              precision={2}
              suffix="¥"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title={t('ops.today_usage')}
              value={stats?.today_usage_tokens || 0}
              prefix={<ApiOutlined />}
              formatter={(v) => formatQuota(Number(v))}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title={t('ops.active_users')}
              value={stats?.active_users || 0}
              prefix={<UserOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title={t('ops.channel_health_rate')}
              value={overallHealth}
              prefix={<SafetyCertificateOutlined />}
              suffix="%"
              precision={1}
            />
            <div style={{ marginTop: 8, color: '#666', fontSize: 12 }}>
              {t('ops.channels_online', { count: enabledCount, total: totalChannels })}
            </div>
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title={t('ops.total_users')} value={stats?.total_users || 0} prefix={<UserOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title={t('ops.total_channels')} value={stats?.total_channels || 0} prefix={<ApiOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title={t('ops.total_tokens')}
              value={stats?.total_tokens || 0}
              formatter={(v) => formatQuota(Number(v))}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title={t('ops.total_quota')}
              value={stats?.total_quota || 0}
              formatter={(v) => formatQuota(Number(v))}
            />
          </Card>
        </Col>
      </Row>

      <Tabs activeKey={activeTab} onChange={handleTabChange} style={{ marginTop: 16 }}>
        <TabPane tab={<span><LineChartOutlined /> {t('ops.revenue_overview')}</span>} key="1">
          <Row gutter={[16, 16]}>
            <Col xs={24} lg={16}>
              <Card title={t('ops.daily_revenue_trend')}>
                <ReactECharts option={getRevenueChartOption()} style={{ height: 300 }} />
              </Card>
            </Col>
            <Col xs={24} lg={8}>
              <Card title={t('ops.quick_actions')}>
                <Space direction="vertical" style={{ width: '100%' }}>
                  <Button block icon={<DollarOutlined />} onClick={() => handleExport('daily')}>
                    {t('ops.export_daily')}
                  </Button>
                  <Button block icon={<RiseOutlined />} onClick={() => handleExport('weekly')}>
                    {t('ops.export_weekly')}
                  </Button>
                  <Button block icon={<DashboardOutlined />} onClick={() => handleExport('monthly')}>
                    {t('ops.export_monthly')}
                  </Button>
                </Space>
              </Card>
            </Col>
          </Row>
        </TabPane>

        <TabPane tab={<span><UserOutlined /> {t('ops.user_management')}</span>} key="2">
          <Card title={t('ops.user_list')}>
            <Table
              dataSource={users}
              columns={[
                { title: t('ops.id'), dataIndex: 'id', key: 'id', width: 60 },
                { title: t('ops.username'), dataIndex: 'username', key: 'username', render: (v: string, r: any) => r.display_name || v },
                { title: t('ops.email'), dataIndex: 'email', key: 'email' },
                { title: t('ops.group'), dataIndex: 'group', key: 'group', width: 100 },
                { title: t('ops.role'), dataIndex: 'role', key: 'role', render: (v: number) => v === 100 ? t('ops.super_admin') : v === 10 ? t('ops.admin') : t('ops.normal_user') },
                { title: t('ops.status'), dataIndex: 'status', key: 'status', render: (v: number) => v === 1 ? t('ops.normal') : v === 2 ? t('ops.disabled') : t('ops.deleted') },
                { title: t('ops.quota'), dataIndex: 'quota', key: 'quota', render: (v: number) => v.toLocaleString() },
                { title: t('ops.used_quota'), dataIndex: 'used_quota', key: 'used_quota', render: (v: number) => v.toLocaleString() },
                { title: t('ops.request_count'), dataIndex: 'request_count', key: 'request_count' },
                {
                  title: t('ops.action'),
                  key: 'action',
                  width: 100,
                  render: (_: any, record: any) => (
                    <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEditUser(record)}>
                      {t('ops.edit')}
                    </Button>
                  )
                },
              ]}
              rowKey="id"
              loading={usersLoading}
              pagination={{ pageSize: 20, total: usersTotal, showTotal: (total) => t('ops.total_records', { count: total }) }}
              onChange={(pagination) => loadUsers(((pagination.current || 1) - 1) * 20, 20)}
            />
          </Card>
        </TabPane>

        <TabPane tab={<span><ApiOutlined /> {t('ops.channel_management')}</span>} key="3">
          <Card
            title={t('ops.channel_list')}
            extra={<Button type="primary" icon={<PlusOutlined />} onClick={handleCreateChannel}>{t('ops.add_channel')}</Button>}
          >
            <Table
              dataSource={channels}
              columns={[
                { title: t('ops.id'), dataIndex: 'id', key: 'id', width: 60 },
                { title: t('ops.channel_name'), dataIndex: 'name', key: 'name' },
                { title: t('ops.channel_type'), dataIndex: 'type_name', key: 'type_name' },
                { title: 'Provider', dataIndex: 'provider', key: 'provider', width: 100 },
                { title: 'Base URL', dataIndex: 'base_url', key: 'base_url', ellipsis: true },
                { title: t('ops.priority'), dataIndex: 'priority', key: 'priority', width: 80 },
                { title: t('ops.status'), dataIndex: 'status', key: 'status', render: (v: number) => v === 1 ? <Tag color="green">{t('ops.enabled')}</Tag> : v === 2 ? <Tag color="red">{t('ops.disabled')}</Tag> : <Tag>{t('ops.unknown_status')}</Tag> },
                {
                  title: t('ops.action'),
                  key: 'action',
                  width: 150,
                  render: (_: any, record: Channel) => (
                    <Space>
                      <Button type="link" size="small" onClick={() => handleEditChannel(record)}>{t('ops.edit')}</Button>
                      <Popconfirm title={t('ops.confirm_delete')} onConfirm={() => handleDeleteChannel(record.id)}>
                        <Button type="link" size="small" danger>{t('ops.delete')}</Button>
                      </Popconfirm>
                    </Space>
                  )
                }
              ]}
              rowKey="id"
              loading={channelsLoading}
              pagination={{ pageSize: 10, total: channelsTotal, showTotal: (total) => t('ops.total_records', { count: total }) }}
              onChange={(pagination) => loadChannels(((pagination.current || 1) - 1) * 10, 10)}
            />
          </Card>
        </TabPane>

        <TabPane tab={<span><ApiOutlined /> {t('ops.channel_health')}</span>} key="4">
          <Card title={t('ops.channel_status')}>
            <Table
              dataSource={channelHealth}
              columns={channelColumns}
              rowKey="id"
              size="small"
              pagination={{ pageSize: 10, total: channelHealth.length, showTotal: (total) => t('ops.total_records', { count: total }) }}
            />
          </Card>
        </TabPane>

        <TabPane tab={<span><UserOutlined /> {t('ops.quota_settings')}</span>} key="6">
          <Form form={settingForm} layout="vertical">
            <Card title={t('ops.quota_config')} style={{ marginTop: 16 }}>
              <Row gutter={[16, 16]}>
                <Col span={12}>
                  <Form.Item label={t('ops.new_user_initial_quota')} name="QuotaForNewUser">
                    <InputNumber style={{ width: '100%' }} placeholder={t('ops.negative_unlimited')} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.pre_consumed_quota')} name="PreConsumedQuota">
                    <InputNumber style={{ width: '100%' }} placeholder={t('ops.pre_consumed_quota_placeholder')} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.inviter_reward_quota')} name="QuotaForInviter">
                    <InputNumber style={{ width: '100%' }} placeholder={t('ops.inviter_reward_placeholder')} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.invitee_reward_quota')} name="QuotaForInvitee">
                    <InputNumber style={{ width: '100%' }} placeholder={t('ops.invitee_reward_placeholder')} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.quota_remind_threshold')} name="QuotaRemindThreshold">
                    <InputNumber style={{ width: '100%' }} placeholder={t('ops.quota_remind_placeholder')} />
                  </Form.Item>
                </Col>
              </Row>
              <Button type="primary" icon={<SaveOutlined />} loading={settingSaving} onClick={handleSaveSettings} style={{ marginTop: 16 }}>
                {t('ops.save_settings')}
              </Button>
            </Card>
          </Form>
        </TabPane>

        <TabPane tab={<span><SettingOutlined /> {t('ops.system_settings')}</span>} key="7">
          <Form form={settingForm} layout="vertical">
            <Card title={t('ops.group_multiplier')} style={{ marginTop: 16 }}>
              <Form.Item
                name="GroupRatio"
                rules={[{ validator: (_, value) => verifyJSON(value) ? Promise.resolve() : Promise.reject(t('ops.not_valid_json')) }]}
              >
                <Input.TextArea
                  rows={6}
                  placeholder={'{\n  "default": 1.0,\n  "enterprise": 1.2\n}'}
                  style={{ fontFamily: 'monospace' }}
                />
              </Form.Item>
              <div style={{ color: '#999', fontSize: 12 }}>
                {t('ops.json_format_example')}
              </div>
            </Card>

            <Card title={t('ops.channel_auto_management')} style={{ marginTop: 16 }}>
              <Row gutter={[16, 16]}>
                <Col span={12}>
                  <Form.Item label={t('ops.auto_disable_channel')} name="AutomaticDisableChannelEnabled" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.auto_enable_channel')} name="AutomaticEnableChannelEnabled" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.channel_disable_threshold')} name="ChannelDisableThreshold">
                    <InputNumber style={{ width: '100%' }} placeholder={t('ops.channel_disable_placeholder')} />
                  </Form.Item>
                </Col>
              </Row>
            </Card>

            <Card title={t('ops.display_settings')} style={{ marginTop: 16 }}>
              <Row gutter={[16, 16]}>
                <Col span={12}>
                  <Form.Item label={t('ops.display_in_currency')} name="DisplayInCurrencyEnabled" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.display_token_stat')} name="DisplayTokenStatEnabled" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.consumption_log')} name="LogConsumeEnabled" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.retry_times')} name="RetryTimes">
                    <InputNumber style={{ width: '100%' }} min={0} max={10} />
                  </Form.Item>
                </Col>
              </Row>
            </Card>

            <Card title={t('ops.alert_settings')} style={{ marginTop: 16 }}>
              <Row gutter={[16, 16]}>
                <Col span={12}>
                  <Form.Item label={t('ops.alert_switch')} name="alert_enabled">
                    <Switch checked={alertConfig?.enabled} onChange={(checked) => {
                      updateAlertConfig({ ...alertConfig, enabled: checked })
                    }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.alert_email')} name="alert_email">
                    <Input placeholder={t('ops.alert_email_placeholder')} />
                  </Form.Item>
                </Col>
                <Col span={24}>
                  <Form.Item label={t('ops.alert_webhook')} name="alert_webhook">
                    <Input placeholder={t('ops.alert_webhook_placeholder')} />
                  </Form.Item>
                </Col>
              </Row>
            </Card>

            <Card title={t('ops.performance_config')} style={{ marginTop: 16 }}>
              <Row gutter={[16, 16]}>
                <Col span={12}>
                  <Form.Item label={t('ops.max_concurrent_requests')} name="max_concurrent_requests">
                    <InputNumber style={{ width: '100%' }} min={1} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.request_queue_timeout')} name="request_queue_timeout">
                    <InputNumber style={{ width: '100%' }} min={0} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.health_check_interval')} name="health_check_interval">
                    <InputNumber style={{ width: '100%' }} min={10} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.health_check_fail_threshold')} name="health_check_fail_threshold">
                    <InputNumber style={{ width: '100%' }} min={1} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.circuit_breaker_threshold')} name="circuit_breaker_threshold">
                    <InputNumber style={{ width: '100%' }} min={1} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.circuit_breaker_timeout')} name="circuit_breaker_timeout">
                    <InputNumber style={{ width: '100%' }} min={1} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('ops.request_timeout')} name="relay_timeout">
                    <InputNumber style={{ width: '100%' }} min={0} />
                  </Form.Item>
                </Col>
              </Row>
            </Card>

            <Button type="primary" icon={<SaveOutlined />} loading={settingSaving} onClick={handleSaveSettings} style={{ marginTop: 16 }}>
              {t('ops.save_settings')}
            </Button>
          </Form>
        </TabPane>
      </Tabs>

      <Modal
        title={editingChannel ? t('ops.edit_channel') : t('ops.add_channel_title')}
        open={channelModalVisible}
        onCancel={() => setChannelModalVisible(false)}
        onOk={handleChannelSubmit}
        okText={t('ops.save')}
        cancelText={t('ops.cancel')}
        width={700}
      >
        <Form form={channelForm} layout="vertical">
          <Form.Item name="name" label={t('ops.channel_name')} rules={[{ required: true, message: t('ops.enter_channel_name') }]}>
            <Input placeholder="如: BEDI-集群1" />
          </Form.Item>
          <Form.Item name="type" label={t('ops.channel_type')} rules={[{ required: true }]}>
            <Select
              options={[
                { value: 50, label: t('ops.openai_compatible') },
                { value: 14, label: t('ops.anthropic') },
                { value: 8, label: t('ops.custom') },
              ]}
              placeholder={t('ops.select_channel_type')}
            />
          </Form.Item>
          <Form.Item name="base_url" label={t('ops.base_url')} rules={[{ required: true, message: t('ops.enter_base_url') }]}>
            <Input placeholder="如: https://api.bedicloud.net/v1" />
          </Form.Item>
          <Form.Item name="provider" label="Provider" rules={[{ required: true, message: t('ops.select_provider') }]}>
            <Select
              showSearch
              allowClear
              placeholder={t('ops.select_provider')}
              options={providers.map(p => ({ value: p.code, label: p.name }))}
            />
          </Form.Item>
          <Form.Item
            name="key"
            label="API Key"
            rules={editingChannel ? [] : [{ required: true, message: t('ops.enter_api_key') }]}
          >
            <Input.Password placeholder={t('ops.enter_api_key')} />
          </Form.Item>

          <Divider>{t('ops.supported_models')}</Divider>

          <Form.List name="modelMappings">
            {(fields, { add, remove }) => (
              <>
                {fields.map(({ key, name, ...restField }) => (
                  <Space key={key} style={{ display: 'flex', marginBottom: 8 }} align="baseline">
                    <Form.Item
                      {...restField}
                      name={[name, 'client']}
                      label={t('ops.client_model_name')}
                      rules={[{ required: true, message: t('ops.enter_client_model_name') }]}
                    >
                      <Input placeholder="如: glm-4" style={{ width: 200 }} />
                    </Form.Item>
                    <span>→</span>
                    <Form.Item
                      {...restField}
                      name={[name, 'upstream']}
                      label={t('ops.upstream_actual_name')}
                      rules={[{ required: true, message: t('ops.enter_upstream_name') }]}
                    >
                      <Input placeholder="如: ZhipuAI/GLM-4" style={{ width: 200 }} />
                    </Form.Item>
                    <Button type="link" danger onClick={() => remove(name)}>{t('ops.delete')}</Button>
                  </Space>
                ))}
                <Form.Item>
                  <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                    {t('ops.add_model_mapping')}
                  </Button>
                </Form.Item>
              </>
            )}
          </Form.List>

          <Divider />

          <Form.Item name="priority" label={t('ops.priority')}>
            <InputNumber style={{ width: '100%' }} placeholder={t('ops.bigger_priority')} min={0} />
          </Form.Item>
          <Form.Item name="status" label={t('ops.status')}>
            <Select
              options={[
                { value: 1, label: t('ops.enabled') },
                { value: 2, label: t('ops.disabled') },
              ]}
              placeholder={t('ops.select_status')}
            />
          </Form.Item>
          <Form.Item name="hide_upstream_model" label={t('ops.hide_upstream_model')} valuePropName="checked">
            <Switch onChange={(checked) => channelForm.setFieldValue('hide_upstream_model', checked)} />
            <span style={{ marginLeft: 8, color: '#999' }}>{t('ops.hide_upstream_model_tip')}</span>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('ops.edit_user')}
        open={editUserModalVisible}
        onCancel={() => setEditUserModalVisible(false)}
        onOk={handleUpdateUser}
        okText={t('ops.save')}
        cancelText={t('ops.cancel')}
      >
        <Form form={editUserForm} layout="vertical">
          <Form.Item name="group" label={t('ops.group_group')}>
            <Input placeholder={t('ops.group_placeholder')} />
          </Form.Item>
          <Form.Item name="quota" label={t('ops.quota_amount')}>
            <InputNumber style={{ width: '100%' }} placeholder={t('ops.quota_placeholder')} min={-1} />
          </Form.Item>
          <Form.Item name="role" label={t('ops.role')}>
            <Select
              options={[
                { value: 1, label: t('ops.normal_user') },
                { value: 10, label: t('ops.admin') },
                { value: 100, label: t('ops.super_admin') },
              ]}
            />
          </Form.Item>
          <Form.Item name="status" label={t('ops.status')}>
            <Select
              options={[
                { value: 1, label: t('ops.normal') },
                { value: 2, label: t('ops.disabled') },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default OpsDashboard