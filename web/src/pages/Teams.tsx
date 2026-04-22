import { useState, useEffect } from 'react'
import { useTheme } from '../contexts/ThemeContext'
import { Row, Col, Card, Table, Button, Modal, Form, Input, InputNumber, Select, Tag, Space, message, Popconfirm, Tabs, Statistic } from 'antd'
import { TeamOutlined, UserOutlined, PlusOutlined, DeleteOutlined, SettingOutlined, AuditOutlined } from '@ant-design/icons'
import { createTenant, getMyTenants, getTenant, updateTenant, getTenantUsers, inviteUser, removeUser, updateUserRole, allocateQuota, getAuditLogs, leaveTenant, getOpsUsers, Tenant, TenantUser, AuditLog, getAllCompanies, getDepartments, createCompany, createDepartment, deleteTenant, Company, Department } from '../services/api'
import { useTranslation } from 'react-i18next'

const { Option } = Select
const { TabPane } = Tabs

const Teams: React.FC = () => {
  const { t } = useTranslation()
  const { appTheme } = useTheme()
  const [, setLoading] = useState(false)
  const [tenants, setTenants] = useState<Tenant[]>([])
  const [currentTenant, setCurrentTenant] = useState<Tenant | null>(null)
  const [users, setUsers] = useState<TenantUser[]>([])
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([])
  const [createModalVisible, setCreateModalVisible] = useState(false)
  const [createCompanyModalVisible, setCreateCompanyModalVisible] = useState(false)
  const [createDeptModalVisible, setCreateDeptModalVisible] = useState(false)
  const [inviteModalVisible, setInviteModalVisible] = useState(false)
  const [quotaModalVisible, setQuotaModalVisible] = useState(false)
  const [currentRole, setCurrentRole] = useState<number>(2)
  const [createForm] = Form.useForm()
  const [createCompanyForm] = Form.useForm()
  const [createDeptForm] = Form.useForm()
  const [inviteForm] = Form.useForm()
  const [quotaForm] = Form.useForm()
  const [allUsers, setAllUsers] = useState<any[]>([])
  const [userSearchLoading, setUserSearchLoading] = useState(false)
  // Company/Department hierarchy
  const [companies, setCompanies] = useState<Company[]>([])
  const [departments, setDepartments] = useState<Department[]>([])
  const [selectedCompanyId, setSelectedCompanyId] = useState<number | null>(null)
  const [isRoot, setIsRoot] = useState(false)

  useEffect(() => {
    loadTenants()
    loadCompanies()
  }, [])

  const loadCompanies = async () => {
    try {
      const res = await getAllCompanies()
      if (res.data.success) {
        setCompanies(res.data.data || [])
        // Root user can see company management even if no companies exist yet
        setIsRoot(true)
        if (res.data.data && res.data.data.length > 0) {
          setSelectedCompanyId(res.data.data[0].id)
          loadDepartments(res.data.data[0].id)
        }
      } else {
        // API returned success:false, user is not root
        setIsRoot(false)
      }
    } catch (error) {
      console.error('Failed to load companies:', error)
      setIsRoot(false)
    }
  }

  const loadDepartments = async (companyId: number) => {
    try {
      const res = await getDepartments(companyId)
      if (res.data.success) {
        setDepartments(res.data.data || [])
      }
    } catch (error) {
      console.error('Failed to load departments:', error)
    }
  }

  const handleCompanyChange = (companyId: number) => {
    setSelectedCompanyId(companyId)
    loadDepartments(companyId)
  }

  const loadTenants = async () => {
    try {
      setLoading(true)
      const res = await getMyTenants()
      setTenants(res.data.data || [])
      if (res.data.data?.length > 0 && !currentTenant) {
        selectTenant(res.data.data[0].id)
      }
    } catch (error) {
      console.error('Failed to load tenants:', error)
    } finally {
      setLoading(false)
    }
  }

  const selectTenant = async (id: number) => {
    try {
      const res = await getTenant(id)
      setCurrentTenant(res.data.data.tenant)
      setCurrentRole(res.data.data.user_role?.role || 2)
      loadTenantUsers(id)
    } catch (error) {
      console.error('Failed to load tenant:', error)
    }
  }

  const loadTenantUsers = async (tenantId: number) => {
    try {
      const res = await getTenantUsers(tenantId)
      setUsers(res.data.data.users || [])
    } catch (error) {
      console.error('Failed to load users:', error)
    }
  }

  const loadAuditLogs = async (tenantId: number) => {
    try {
      const res = await getAuditLogs(tenantId)
      setAuditLogs(res.data.data.logs || [])
    } catch (error) {
      console.error('Failed to load audit logs:', error)
    }
  }

  const handleCreateTenant = async (values: { name: string; code: string; company_id?: number; department_id?: number }) => {
    try {
      const res = await createTenant(values)
      if (res.data.success) {
        message.success(t('teams.team_created'))
        setCreateModalVisible(false)
        createForm.resetFields()
        loadTenants()
      } else {
        message.error(res.data.message || t('common.create_failed'))
      }
    } catch (error: any) {
      message.error(error.response?.data?.message || t('common.create_failed'))
    }
  }

  const handleCreateCompany = async (values: { name: string; code: string; description?: string }) => {
    try {
      const res = await createCompany(values)
      if (res.data.success) {
        message.success(t('teams.company_created'))
        setCreateCompanyModalVisible(false)
        loadCompanies()
      } else {
        message.error(res.data.message || t('common.create_failed'))
      }
    } catch (error: any) {
      message.error(error.response?.data?.message || t('common.create_failed'))
    }
  }

  const handleDeleteTenant = async (tenantId: number) => {
    try {
      const res = await deleteTenant(tenantId)
      if (res.data.success) {
        message.success(t('teams.team_deleted'))
        loadTenants()
        if (currentTenant?.id === tenantId) {
          setCurrentTenant(null)
        }
        if (selectedCompanyId) {
          loadDepartments(selectedCompanyId)
        }
      } else {
        message.error(res.data.message || t('common.delete_failed'))
      }
    } catch (error: any) {
      message.error(error.response?.data?.message || t('common.delete_failed'))
    }
  }

  const handleCreateDepartment = async (values: { name: string; code?: string; description?: string }) => {
    if (!selectedCompanyId) return
    try {
      const res = await createDepartment(selectedCompanyId, values)
      if (res.data.success) {
        message.success(t('teams.department_created'))
        setCreateDeptModalVisible(false)
        loadDepartments(selectedCompanyId)
      } else {
        message.error(res.data.message || t('common.create_failed'))
      }
    } catch (error: any) {
      message.error(error.response?.data?.message || t('common.create_failed'))
    }
  }

  const searchUsers = async (keyword: string) => {
    try {
      setUserSearchLoading(true)
      const res = await getOpsUsers({ limit: 1000, offset: 0, keyword })
      if (res.data.success) {
        setAllUsers(res.data.data.users || [])
      }
    } catch (error) {
      console.error('Failed to search users:', error)
    } finally {
      setUserSearchLoading(false)
    }
  }

  const handleInviteUser = async (values: { user_id: number; role: number; quota?: number }) => {
    if (!currentTenant) return
    try {
      const res = await inviteUser(currentTenant.id, values)
      if (res.data.success) {
        message.success(t('teams.invite_success'))
        setInviteModalVisible(false)
        inviteForm.resetFields()
        loadTenantUsers(currentTenant.id)
      } else {
        message.error(res.data.message || t('common.invite_failed'))
      }
    } catch (error: any) {
      message.error(error.response?.data?.message || t('common.invite_failed'))
    }
  }

  const handleRemoveUser = async (userId: number) => {
    if (!currentTenant) return
    try {
      await removeUser(currentTenant.id, userId)
      message.success(t('teams.member_removed'))
      loadTenantUsers(currentTenant.id)
    } catch (error) {
      message.error(t('teams.remove_failed'))
    }
  }

  const handleUpdateRole = async (userId: number, role: number) => {
    if (!currentTenant) return
    try {
      await updateUserRole(currentTenant.id, userId, { role })
      message.success(t('teams.role_updated'))
      loadTenantUsers(currentTenant.id)
    } catch (error) {
      message.error(t('teams.update_failed'))
    }
  }

  const handleAllocateQuota = async (values: { target_user_id: number; quota: number }) => {
    if (!currentTenant) return
    try {
      await allocateQuota(currentTenant.id, values)
      message.success(t('teams.quota_allocated_success'))
      setQuotaModalVisible(false)
      quotaForm.resetFields()
      loadTenantUsers(currentTenant.id)
    } catch (error) {
      message.error(t('teams.allocate_failed'))
    }
  }

  const handleLeave = async () => {
    if (!currentTenant) return
    try {
      await leaveTenant(currentTenant.id)
      message.success(t('teams.left_team'))
      setCurrentTenant(null)
      loadTenants()
    } catch (error) {
      message.error(t('teams.leave_failed'))
    }
  }

  const getRoleName = (role: number) => {
    const map: Record<number, { color: string; text: string }> = {
      0: { color: 'red', text: t('teams.owner') },
      1: { color: 'blue', text: t('teams.admin') },
      2: { color: 'green', text: t('teams.member') },
      3: { color: 'gray', text: t('teams.observer') }
    }
    return map[role] || { color: 'gray', text: t('teams.unknown') }
  }

  const formatQuota = (quota: number) => {
    if (quota >= 1000000) return (quota / 1000000).toFixed(2) + 'M'
    if (quota >= 1000) return (quota / 1000).toFixed(2) + 'K'
    return quota.toString()
  }

  const userColumns = [
    { title: t('common.user'), dataIndex: 'username', key: 'username', render: (v: string, r: TenantUser) => (
      <Space>
        <UserOutlined />
        <span>{r.display_name || v}</span>
        <span style={{ color: appTheme.textSecondary }}>@{v}</span>
      </Space>
    )},
    { title: t('user.role'), dataIndex: 'role', key: 'role', render: (v: number) => {
      const role = getRoleName(v)
      return <Tag color={role.color}>{role.text}</Tag>
    }},
    { title: t('teams.alloc_quota'), dataIndex: 'quota_alloc', key: 'quota_alloc', render: (v: number) => formatQuota(v) },
    { title: t('teams.quota_used'), dataIndex: 'used_quota', key: 'used_quota', render: (v: number) => formatQuota(v) },
    { title: t('common.action'), key: 'action', render: (_: any, r: TenantUser) => (
      <Space>
        {(currentRole <= 1 || isRoot) && r.role > 0 && (
          <>
            <Button size="small" onClick={() => {
              quotaForm.setFieldsValue({ target_user_id: r.id })
              setQuotaModalVisible(true)
            }}>{t('teams.alloc_quota')}</Button>
            <Select size="small" value={r.role} style={{ width: 80 }} onChange={(v) => handleUpdateRole(r.id, v)}>
              <Option value={1}>{t('teams.admin')}</Option>
              <Option value={2}>{t('teams.member')}</Option>
              <Option value={3}>{t('teams.observer')}</Option>
            </Select>
            <Popconfirm title={t('teams.confirm_remove_user')} onConfirm={() => handleRemoveUser(r.id)}>
              <Button size="small" danger icon={<DeleteOutlined />}>{t('teams.remove')}</Button>
            </Popconfirm>
          </>
        )}
      </Space>
    )}
  ]

  const auditColumns = [
    { title: t('teams.time'), dataIndex: 'created_at', key: 'created_at', render: (v: number) => new Date(v * 1000).toLocaleString() },
    { title: t('common.action'), dataIndex: 'action', key: 'action', render: (v: string) => {
      const actionMap: Record<string, string> = {
        create_user: t('teams.action_create_user'), delete_user: t('teams.action_delete_user'), update_user: t('teams.action_update_user'),
        allocate_quota: t('teams.action_allocate_quota'), create_channel: t('teams.action_create_channel'), delete_channel: t('teams.action_delete_channel'),
        view_users: t('teams.action_view_users'), leave_tenant: t('teams.action_leave_tenant')
      }
      return actionMap[v] || v
    }},
    { title: t('teams.target'), dataIndex: 'target', key: 'target' },
    { title: t('teams.detail'), dataIndex: 'details', key: 'details', ellipsis: true },
    { title: 'IP', dataIndex: 'ip', key: 'ip' }
  ]

  return (
    <div>
      <Row gutter={16}>
        <Col xs={24} lg={8}>
          <Card title={isRoot ? t('teams.company_team') : t('teams.my_team')} extra={
            isRoot ? (
              <>
                {companies.length === 0 ? (
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateCompanyModalVisible(true)}>
                    {t('teams.create_company')}
                  </Button>
                ) : (
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateDeptModalVisible(true)}>
                    {t('teams.create_department')}
                  </Button>
                )}
              </>
            ) : (
              <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalVisible(true)}>
                {t('teams.create_team')}
              </Button>
            )
          }>
            {isRoot && companies.length > 0 && (
              <>
                <div style={{ marginBottom: 12 }}>
                  <Select
                    value={selectedCompanyId}
                    onChange={handleCompanyChange}
                    style={{ width: '100%' }}
                    placeholder={t('teams.select_company')}
                    options={companies.map(c => ({ value: c.id, label: c.name }))}
                  />
                </div>
                {departments.length === 0 ? (
                  <div style={{ textAlign: 'center', padding: 20 }}>
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateDeptModalVisible(true)}>
                      {t('teams.create_department')}
                    </Button>
                  </div>
                ) : (
                  departments.map(d => (
                    <div key={d.id}>
                      <div style={{ fontWeight: 600, color: appTheme.textSecondary, padding: '4px 0', fontSize: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span>{d.name}</span>
                        <Button type="text" size="small" icon={<PlusOutlined />} onClick={() => setCreateModalVisible(true)} />
                      </div>
                      {tenants.filter(tenant => tenant.department_id === d.id).map(tenant => (
                        <Card key={tenant.id} size="small" style={{ marginBottom: 4, cursor: 'pointer', background: currentTenant?.id === tenant.id ? appTheme.bgElevated : appTheme.bgContainer }}
                          onClick={() => selectTenant(tenant.id)}>
                          <Space>
                            <TeamOutlined />
                            <span style={{ fontWeight: currentTenant?.id === tenant.id ? 'bold' : 'normal' }}>{tenant.name}</span>
                            <Tag>{tenant.code}</Tag>
                            <Popconfirm title={t('teams.confirm_delete_team')} onConfirm={(e) => { e?.stopPropagation(); handleDeleteTenant(tenant.id) }} onCancel={(e) => e?.stopPropagation()}>
                              <Button type="text" danger size="small" icon={<DeleteOutlined />} onClick={(e) => e.stopPropagation()} />
                            </Popconfirm>
                          </Space>
                        </Card>
                      ))}
                    </div>
                  ))
                )}
              </>
            )}
            {!isRoot && tenants.map(t => (
              <Card key={t.id} size="small" style={{ marginBottom: 8, cursor: 'pointer', background: currentTenant?.id === t.id ? appTheme.borderLight : appTheme.bgElevated }}
                onClick={() => selectTenant(t.id)}>
                <Space>
                  <TeamOutlined />
                  <span style={{ fontWeight: currentTenant?.id === t.id ? 'bold' : 'normal' }}>{t.name}</span>
                  <Tag>{t.code}</Tag>
                </Space>
              </Card>
            ))}
            {!isRoot && tenants.length === 0 && (
              <div style={{ textAlign: 'center', color: appTheme.textSecondary, padding: 20 }}>
                {t('teams.not_joined_any_team')}
              </div>
            )}
          </Card>
        </Col>

        <Col xs={24} lg={16}>
          {currentTenant ? (
            <Tabs defaultActiveKey="1" onChange={(k) => k === '3' && loadAuditLogs(currentTenant.id)}>
              <TabPane tab={<span><UserOutlined /> {t('teams.member_management')}</span>} key="1">
                <Card title={`${currentTenant.name} - ${t('teams.member_management')}`} extra={
                  (currentRole <= 1 || isRoot) && (
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => setInviteModalVisible(true)}>
                      {t('teams.invite_member')}
                    </Button>
                  )
                }>
                  <Row gutter={16} style={{ marginBottom: 16 }}>
                    <Col span={8}>
                      <Statistic title={t('teams.team_quota')} value={currentTenant.quota_limit > 0 ? formatQuota(currentTenant.quota_limit) : t('common.no_limit')} />
                    </Col>
                    <Col span={8}>
                      <Statistic title={t('teams.quota_used')} value={formatQuota(currentTenant.quota_used)} />
                    </Col>
                    <Col span={8}>
                      <Statistic title={t('teams.member_count')} value={users.length} suffix={`/ ${currentTenant.max_users}`} />
                    </Col>
                  </Row>
                  <Table dataSource={users} columns={userColumns} rowKey="id" size="small" pagination={{ pageSize: 10 }} />
                </Card>
              </TabPane>

              <TabPane tab={<span><SettingOutlined /> {t('teams.team_settings')}</span>} key="2">
                <Card title={t('teams.team_settings')}>
                  <Form layout="vertical" initialValues={currentTenant} onFinish={(values) => updateTenant(currentTenant!.id, values).then(() => {
                    message.success(t('teams.settings_updated'))
                    selectTenant(currentTenant!.id)
                  })}>
                    <Form.Item name="name" label={t('teams.team_name')}>
                      <Input />
                    </Form.Item>
                    <Form.Item name="max_users" label={t('teams.max_members')}>
                      <Input type="number" />
                    </Form.Item>
                    <Form.Item name="max_channels" label={t('teams.max_channels')}>
                      <Input type="number" />
                    </Form.Item>
                    <Row gutter={16}>
                      <Col span={8}>
                        <Form.Item name="rate_limit_rpm" label={t('teams.rpm_limit')}>
                          <InputNumber style={{ width: '100%' }} placeholder={t('common.no_limit')} min={0} />
                        </Form.Item>
                      </Col>
                      <Col span={8}>
                        <Form.Item name="rate_limit_tpm" label={t('teams.tpm_limit')}>
                          <InputNumber style={{ width: '100%' }} placeholder={t('common.no_limit')} min={0} />
                        </Form.Item>
                      </Col>
                      <Col span={8}>
                        <Form.Item name="rate_limit_concurrent" label={t('teams.concurrent_limit')}>
                          <InputNumber style={{ width: '100%' }} placeholder={t('common.no_limit')} min={0} />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Form.Item>
                      <Button type="primary" htmlType="submit">{t('common.save')}</Button>
                    </Form.Item>
                  </Form>
                  {currentRole === 0 && (
                    <div style={{ marginTop: 32, borderTop: '1px solid #f0f0f0', paddingTop: 16 }}>
                      <h4 style={{ color: '#ff4d4f' }}>{t('teams.danger_zone')}</h4>
                      <Popconfirm title={t('teams.confirm_leave_team')} onConfirm={handleLeave}>
                        <Button danger>{t('teams.leave_team')}</Button>
                      </Popconfirm>
                    </div>
                  )}
                </Card>
              </TabPane>

              <TabPane tab={<span><AuditOutlined /> {t('teams.audit_log')}</span>} key="3">
                <Card title={t('teams.audit_log')}>
                  <Table dataSource={auditLogs} columns={auditColumns} rowKey="id" size="small" pagination={{ pageSize: 10 }} />
                </Card>
              </TabPane>
            </Tabs>
          ) : (
            <Card>
              <div style={{ textAlign: 'center', color: appTheme.textSecondary, padding: 40 }}>
                <TeamOutlined style={{ fontSize: 48, marginBottom: 16 }} />
                <p>{t('teams.select_team_prompt')}</p>
              </div>
            </Card>
          )}
        </Col>
      </Row>

      <Modal title={t('teams.create_team')} open={createModalVisible} onCancel={() => setCreateModalVisible(false)} footer={null}>
        <Form form={createForm} onFinish={handleCreateTenant} layout="vertical">
          {isRoot && (
            <>
              <Form.Item name="company_id" label={t('teams.belong_company')} rules={[{ required: true, message: t('teams.select_company') }]}>
                <Select placeholder={t('teams.select_company')} onChange={handleCompanyChange}>
                  {companies.map(c => <Option key={c.id} value={c.id}>{c.name}</Option>)}
                </Select>
              </Form.Item>
              <Form.Item name="department_id" label={t('common.belong_department')} rules={[{ required: true, message: t('common.select_department') }]}>
                <Select placeholder={t('common.select_department')}>
                  {departments.map(d => <Option key={d.id} value={d.id}>{d.name}</Option>)}
                </Select>
              </Form.Item>
            </>
          )}
          <Form.Item name="name" label={t('teams.team_name')} rules={[{ required: true, message: t('teams.enter_team_name') }]}>
            <Input placeholder={t('teams.team_name_placeholder')} />
          </Form.Item>
          <Form.Item name="code" label={t('teams.team_code')} rules={[{ required: true, message: t('teams.enter_team_code') }]}>
            <Input placeholder={t('teams.team_code_placeholder')} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">{t('common.create')}</Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={t('teams.create_company')} open={createCompanyModalVisible} onCancel={() => setCreateCompanyModalVisible(false)} footer={null}>
        <Form form={createCompanyForm} onFinish={handleCreateCompany} layout="vertical">
          <Form.Item name="name" label={t('teams.company_name')} rules={[{ required: true, message: t('teams.enter_company_name') }]}>
            <Input placeholder={t('teams.company_name_placeholder')} />
          </Form.Item>
          <Form.Item name="code" label={t('teams.company_code')} rules={[{ required: true, message: t('teams.enter_company_code') }]}>
            <Input placeholder={t('teams.company_code_placeholder')} />
          </Form.Item>
          <Form.Item name="description" label={t('common.description')}>
            <Input.TextArea placeholder={t('teams.company_description_placeholder')} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">{t('common.create')}</Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={t('teams.create_department')} open={createDeptModalVisible} onCancel={() => setCreateDeptModalVisible(false)} footer={null}>
        <Form form={createDeptForm} onFinish={handleCreateDepartment} layout="vertical">
          <Form.Item name="name" label={t('teams.department_name')} rules={[{ required: true, message: t('teams.enter_department_name') }]}>
            <Input placeholder={t('teams.department_name_placeholder')} />
          </Form.Item>
          <Form.Item name="code" label={t('teams.department_code')}>
            <Input placeholder={t('teams.department_code_placeholder')} />
          </Form.Item>
          <Form.Item name="description" label={t('common.description')}>
            <Input.TextArea placeholder={t('teams.department_description_placeholder')} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">{t('common.create')}</Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={t('teams.invite_member')} open={inviteModalVisible} onCancel={() => { setInviteModalVisible(false); setAllUsers([]) }} footer={null}>
        <Form form={inviteForm} onFinish={handleInviteUser} layout="vertical">
          <Form.Item name="user_id" label={t('teams.select_user')} rules={[{ required: true, message: t('teams.please_select_user') }]}>
            <Select
              showSearch
              placeholder={t('teams.search_user_placeholder')}
              filterOption={false}
              onSearch={searchUsers}
              loading={userSearchLoading}
              onFocus={() => searchUsers('')}
            >
              {allUsers.map((u: any) => (
                <Option key={u.id} value={u.id}>
                  {u.display_name || u.username} ({u.email || u.username})
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="role" label={t('common.role')} rules={[{ required: true, message: t('teams.please_select_role') }]}>
            <Select placeholder={t('teams.select_role')}>
              <Option value={1}>{t('teams.admin')}</Option>
              <Option value={2}>{t('teams.member')}</Option>
              <Option value={3}>{t('teams.observer')}</Option>
            </Select>
          </Form.Item>
          <Form.Item name="quota" label={t('teams.alloc_quota')}>
            <Input type="number" placeholder={t('teams.input_quota_placeholder')} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">{t('teams.invite')}</Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={t('teams.allocate_quota')} open={quotaModalVisible} onCancel={() => setQuotaModalVisible(false)} footer={null}>
        <Form form={quotaForm} onFinish={handleAllocateQuota} layout="vertical">
          <Form.Item name="target_user_id" label={t('common.user_id')} rules={[{ required: true }]}>
            <Input disabled />
          </Form.Item>
          <Form.Item name="quota" label={t('teams.alloc_quota')} rules={[{ required: true, message: t('teams.enter_quota') }]}>
            <Input type="number" placeholder={t('teams.input_quota_placeholder')} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">{t('teams.allocate')}</Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default Teams