import { useState, useEffect } from 'react'
import { Row, Col, Card, Table, Button, Modal, Form, Input, InputNumber, Select, Tag, Space, message, Popconfirm, Tabs, Statistic } from 'antd'
import { TeamOutlined, UserOutlined, PlusOutlined, DeleteOutlined, SettingOutlined, AuditOutlined } from '@ant-design/icons'
import { createTenant, getMyTenants, getTenant, updateTenant, getTenantUsers, inviteUser, removeUser, updateUserRole, allocateQuota, getAuditLogs, leaveTenant, getOpsUsers, Tenant, TenantUser, AuditLog, getAllCompanies, getDepartments, createCompany, createDepartment, deleteTenant, Company, Department } from '../services/api'

const { Option } = Select
const { TabPane } = Tabs

const Teams: React.FC = () => {
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
        message.success('团队创建成功')
        setCreateModalVisible(false)
        createForm.resetFields()
        loadTenants()
      } else {
        message.error(res.data.message || '创建失败')
      }
    } catch (error: any) {
      message.error(error.response?.data?.message || '创建失败')
    }
  }

  const handleCreateCompany = async (values: { name: string; code: string; description?: string }) => {
    try {
      const res = await createCompany(values)
      if (res.data.success) {
        message.success('公司创建成功')
        setCreateCompanyModalVisible(false)
        loadCompanies()
      } else {
        message.error(res.data.message || '创建失败')
      }
    } catch (error: any) {
      message.error(error.response?.data?.message || '创建失败')
    }
  }

  const handleDeleteTenant = async (tenantId: number) => {
    try {
      const res = await deleteTenant(tenantId)
      if (res.data.success) {
        message.success('团队删除成功')
        loadTenants()
        if (currentTenant?.id === tenantId) {
          setCurrentTenant(null)
        }
      } else {
        message.error(res.data.message || '删除失败')
      }
    } catch (error: any) {
      message.error(error.response?.data?.message || '删除失败')
    }
  }

  const handleCreateDepartment = async (values: { name: string; code?: string; description?: string }) => {
    if (!selectedCompanyId) return
    try {
      const res = await createDepartment(selectedCompanyId, values)
      if (res.data.success) {
        message.success('部门创建成功')
        setCreateDeptModalVisible(false)
        loadDepartments(selectedCompanyId)
      } else {
        message.error(res.data.message || '创建失败')
      }
    } catch (error: any) {
      message.error(error.response?.data?.message || '创建失败')
    }
  }

  const searchUsers = async (keyword: string) => {
    try {
      setUserSearchLoading(true)
      const res = await getOpsUsers({ limit: 50, offset: 0 })
      if (res.data.success) {
        const users = res.data.data.users || []
        if (keyword) {
          const kw = keyword.toLowerCase()
          setAllUsers(users.filter((u: any) =>
            u.username?.toLowerCase().includes(kw) ||
            u.email?.toLowerCase().includes(kw) ||
            u.display_name?.toLowerCase().includes(kw)
          ))
        } else {
          setAllUsers(users)
        }
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
        message.success('邀请成功')
        setInviteModalVisible(false)
        inviteForm.resetFields()
        loadTenantUsers(currentTenant.id)
      } else {
        message.error(res.data.message || '邀请失败')
      }
    } catch (error: any) {
      message.error(error.response?.data?.message || '邀请失败')
    }
  }

  const handleRemoveUser = async (userId: number) => {
    if (!currentTenant) return
    try {
      await removeUser(currentTenant.id, userId)
      message.success('已移除')
      loadTenantUsers(currentTenant.id)
    } catch (error) {
      message.error('移除失败')
    }
  }

  const handleUpdateRole = async (userId: number, role: number) => {
    if (!currentTenant) return
    try {
      await updateUserRole(currentTenant.id, userId, { role })
      message.success('角色已更新')
      loadTenantUsers(currentTenant.id)
    } catch (error) {
      message.error('更新失败')
    }
  }

  const handleAllocateQuota = async (values: { target_user_id: number; quota: number }) => {
    if (!currentTenant) return
    try {
      await allocateQuota(currentTenant.id, values)
      message.success('额度已分配')
      setQuotaModalVisible(false)
      quotaForm.resetFields()
      loadTenantUsers(currentTenant.id)
    } catch (error) {
      message.error('分配失败')
    }
  }

  const handleLeave = async () => {
    if (!currentTenant) return
    try {
      await leaveTenant(currentTenant.id)
      message.success('已离开团队')
      setCurrentTenant(null)
      loadTenants()
    } catch (error) {
      message.error('离开失败')
    }
  }

  const getRoleName = (role: number) => {
    const map: Record<number, { color: string; text: string }> = {
      0: { color: 'red', text: '所有者' },
      1: { color: 'blue', text: '管理员' },
      2: { color: 'green', text: '成员' },
      3: { color: 'gray', text: '观察者' }
    }
    return map[role] || { color: 'gray', text: '未知' }
  }

  const formatQuota = (quota: number) => {
    if (quota >= 1000000) return (quota / 1000000).toFixed(2) + 'M'
    if (quota >= 1000) return (quota / 1000).toFixed(2) + 'K'
    return quota.toString()
  }

  const userColumns = [
    { title: '用户', dataIndex: 'username', key: 'username', render: (v: string, r: TenantUser) => (
      <Space>
        <UserOutlined />
        <span>{r.display_name || v}</span>
        <span style={{ color: '#999' }}>@{v}</span>
      </Space>
    )},
    { title: '角色', dataIndex: 'role', key: 'role', render: (v: number) => {
      const role = getRoleName(v)
      return <Tag color={role.color}>{role.text}</Tag>
    }},
    { title: '分配额度', dataIndex: 'quota_alloc', key: 'quota_alloc', render: (v: number) => formatQuota(v) },
    { title: '已用额度', dataIndex: 'used_quota', key: 'used_quota', render: (v: number) => formatQuota(v) },
    { title: '操作', key: 'action', render: (_: any, r: TenantUser) => (
      <Space>
        {(currentRole <= 1 || isRoot) && r.role > 0 && (
          <>
            <Button size="small" onClick={() => {
              quotaForm.setFieldsValue({ target_user_id: r.id })
              setQuotaModalVisible(true)
            }}>分配额度</Button>
            <Select size="small" value={r.role} style={{ width: 80 }} onChange={(v) => handleUpdateRole(r.id, v)}>
              <Option value={1}>管理员</Option>
              <Option value={2}>成员</Option>
              <Option value={3}>观察者</Option>
            </Select>
            <Popconfirm title="确定要移除此用户吗？" onConfirm={() => handleRemoveUser(r.id)}>
              <Button size="small" danger icon={<DeleteOutlined />}>移除</Button>
            </Popconfirm>
          </>
        )}
      </Space>
    )}
  ]

  const auditColumns = [
    { title: '时间', dataIndex: 'created_at', key: 'created_at', render: (v: number) => new Date(v * 1000).toLocaleString() },
    { title: '操作', dataIndex: 'action', key: 'action', render: (v: string) => {
      const actionMap: Record<string, string> = {
        create_user: '添加用户', delete_user: '删除用户', update_user: '更新用户',
        allocate_quota: '分配额度', create_channel: '创建渠道', delete_channel: '删除渠道',
        view_users: '查看用户', leave_tenant: '离开团队'
      }
      return actionMap[v] || v
    }},
    { title: '目标', dataIndex: 'target', key: 'target' },
    { title: '详情', dataIndex: 'details', key: 'details', ellipsis: true },
    { title: 'IP', dataIndex: 'ip', key: 'ip' }
  ]

  return (
    <div>
      <Row gutter={16}>
        <Col xs={24} lg={8}>
          <Card title={isRoot ? "公司/团队" : "我的团队"} extra={
            isRoot ? (
              <>
                {companies.length === 0 ? (
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateCompanyModalVisible(true)}>
                    创建公司
                  </Button>
                ) : (
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalVisible(true)}>
                    创建团队
                  </Button>
                )}
              </>
            ) : (
              <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalVisible(true)}>
                创建团队
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
                    placeholder="选择公司"
                    options={companies.map(c => ({ value: c.id, label: c.name }))}
                  />
                </div>
                {departments.length === 0 ? (
                  <div style={{ textAlign: 'center', padding: 20 }}>
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateDeptModalVisible(true)}>
                      创建部门
                    </Button>
                  </div>
                ) : (
                  departments.map(d => (
                    <div key={d.id}>
                      <div style={{ fontWeight: 600, color: '#666', padding: '4px 0', fontSize: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span>{d.name}</span>
                        <Button type="text" size="small" icon={<PlusOutlined />} onClick={() => setCreateDeptModalVisible(true)} />
                      </div>
                      {tenants.filter(t => t.department_id === d.id).map(t => (
                        <Card key={t.id} size="small" style={{ marginBottom: 4, cursor: 'pointer', background: currentTenant?.id === t.id ? '#e6f7ff' : '#fafafa' }}
                          onClick={() => selectTenant(t.id)}>
                          <Space>
                            <TeamOutlined />
                            <span style={{ fontWeight: currentTenant?.id === t.id ? 'bold' : 'normal' }}>{t.name}</span>
                            <Tag>{t.code}</Tag>
                          </Space>
                        </Card>
                      ))}
                    </div>
                  ))
                )}
              </>
            )}
            {!isRoot && tenants.map(t => (
              <Card key={t.id} size="small" style={{ marginBottom: 8, cursor: 'pointer', background: currentTenant?.id === t.id ? '#f0f0f0' : '#fff' }}
                onClick={() => selectTenant(t.id)}>
                <Space>
                  <TeamOutlined />
                  <span style={{ fontWeight: currentTenant?.id === t.id ? 'bold' : 'normal' }}>{t.name}</span>
                  <Tag>{t.code}</Tag>
                  {isRoot && (
                    <Button
                      type="text"
                      danger
                      size="small"
                      icon={<DeleteOutlined />}
                      onClick={(e) => {
                        e.stopPropagation()
                        handleDeleteTenant(t.id)
                      }}
                    />
                  )}
                </Space>
              </Card>
            ))}
            {!isRoot && tenants.length === 0 && (
              <div style={{ textAlign: 'center', color: '#999', padding: 20 }}>
                暂未加入任何团队
              </div>
            )}
          </Card>
        </Col>

        <Col xs={24} lg={16}>
          {currentTenant ? (
            <Tabs defaultActiveKey="1" onChange={(k) => k === '3' && loadAuditLogs(currentTenant.id)}>
              <TabPane tab={<span><UserOutlined /> 成员管理</span>} key="1">
                <Card title={`${currentTenant.name} - 成员管理`} extra={
                  (currentRole <= 1 || isRoot) && (
                    <Button type="primary" icon={<PlusOutlined />} onClick={() => setInviteModalVisible(true)}>
                      邀请成员
                    </Button>
                  )
                }>
                  <Row gutter={16} style={{ marginBottom: 16 }}>
                    <Col span={8}>
                      <Statistic title="团队额度" value={currentTenant.quota_limit > 0 ? formatQuota(currentTenant.quota_limit) : '无限制'} />
                    </Col>
                    <Col span={8}>
                      <Statistic title="已用额度" value={formatQuota(currentTenant.quota_used)} />
                    </Col>
                    <Col span={8}>
                      <Statistic title="成员数" value={users.length} suffix={`/ ${currentTenant.max_users}`} />
                    </Col>
                  </Row>
                  <Table dataSource={users} columns={userColumns} rowKey="id" size="small" pagination={{ pageSize: 10 }} />
                </Card>
              </TabPane>

              <TabPane tab={<span><SettingOutlined /> 团队设置</span>} key="2">
                <Card title="团队设置">
                  <Form layout="vertical" initialValues={currentTenant} onFinish={(values) => updateTenant(currentTenant!.id, values).then(() => {
                    message.success('设置已更新')
                    selectTenant(currentTenant!.id)
                  })}>
                    <Form.Item name="name" label="团队名称">
                      <Input />
                    </Form.Item>
                    <Form.Item name="max_users" label="最大成员数">
                      <Input type="number" />
                    </Form.Item>
                    <Form.Item name="max_channels" label="最大渠道数">
                      <Input type="number" />
                    </Form.Item>
                    <Row gutter={16}>
                      <Col span={8}>
                        <Form.Item name="rate_limit_rpm" label="RPM (请求/分钟)">
                          <InputNumber style={{ width: '100%' }} placeholder="0=无限制" min={0} />
                        </Form.Item>
                      </Col>
                      <Col span={8}>
                        <Form.Item name="rate_limit_tpm" label="TPM (Token/分钟)">
                          <InputNumber style={{ width: '100%' }} placeholder="0=无限制" min={0} />
                        </Form.Item>
                      </Col>
                      <Col span={8}>
                        <Form.Item name="rate_limit_concurrent" label="并发数">
                          <InputNumber style={{ width: '100%' }} placeholder="0=无限制" min={0} />
                        </Form.Item>
                      </Col>
                    </Row>
                    <Form.Item>
                      <Button type="primary" htmlType="submit">保存设置</Button>
                    </Form.Item>
                  </Form>
                  {currentRole === 0 && (
                    <div style={{ marginTop: 32, borderTop: '1px solid #f0f0f0', paddingTop: 16 }}>
                      <h4 style={{ color: '#ff4d4f' }}>危险操作</h4>
                      <Popconfirm title="确定要离开此团队吗？" onConfirm={handleLeave}>
                        <Button danger>离开团队</Button>
                      </Popconfirm>
                    </div>
                  )}
                </Card>
              </TabPane>

              <TabPane tab={<span><AuditOutlined /> 审计日志</span>} key="3">
                <Card title="审计日志">
                  <Table dataSource={auditLogs} columns={auditColumns} rowKey="id" size="small" pagination={{ pageSize: 10 }} />
                </Card>
              </TabPane>
            </Tabs>
          ) : (
            <Card>
              <div style={{ textAlign: 'center', color: '#999', padding: 40 }}>
                <TeamOutlined style={{ fontSize: 48, marginBottom: 16 }} />
                <p>请选择一个团队或创建新团队</p>
              </div>
            </Card>
          )}
        </Col>
      </Row>

      <Modal title="创建团队" open={createModalVisible} onCancel={() => setCreateModalVisible(false)} footer={null}>
        <Form form={createForm} onFinish={handleCreateTenant} layout="vertical">
          {isRoot && (
            <>
              <Form.Item name="company_id" label="所属公司" rules={[{ required: true, message: '请选择公司' }]}>
                <Select placeholder="选择公司" onChange={handleCompanyChange}>
                  {companies.map(c => <Option key={c.id} value={c.id}>{c.name}</Option>)}
                </Select>
              </Form.Item>
              <Form.Item name="department_id" label="所属部门" rules={[{ required: true, message: '请选择部门' }]}>
                <Select placeholder="选择部门">
                  {departments.map(d => <Option key={d.id} value={d.id}>{d.name}</Option>)}
                </Select>
              </Form.Item>
            </>
          )}
          <Form.Item name="name" label="团队名称" rules={[{ required: true, message: '请输入团队名称' }]}>
            <Input placeholder="如：产品研发部" />
          </Form.Item>
          <Form.Item name="code" label="团队代码" rules={[{ required: true, message: '请输入团队代码' }]}>
            <Input placeholder="唯一标识，如：product-dev" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">创建</Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="创建公司" open={createCompanyModalVisible} onCancel={() => setCreateCompanyModalVisible(false)} footer={null}>
        <Form form={createCompanyForm} onFinish={handleCreateCompany} layout="vertical">
          <Form.Item name="name" label="公司名称" rules={[{ required: true, message: '请输入公司名称' }]}>
            <Input placeholder="如：北电数智" />
          </Form.Item>
          <Form.Item name="code" label="公司代码" rules={[{ required: true, message: '请输入公司代码' }]}>
            <Input placeholder="唯一标识，如：bedi" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea placeholder="公司描述（可选）" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">创建</Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="创建部门" open={createDeptModalVisible} onCancel={() => setCreateDeptModalVisible(false)} footer={null}>
        <Form form={createDeptForm} onFinish={handleCreateDepartment} layout="vertical">
          <Form.Item name="name" label="部门名称" rules={[{ required: true, message: '请输入部门名称' }]}>
            <Input placeholder="如：产研中心" />
          </Form.Item>
          <Form.Item name="code" label="部门代码">
            <Input placeholder="唯一标识（可选），如：rd" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea placeholder="部门描述（可选）" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">创建</Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="邀请成员" open={inviteModalVisible} onCancel={() => { setInviteModalVisible(false); setAllUsers([]) }} footer={null}>
        <Form form={inviteForm} onFinish={handleInviteUser} layout="vertical">
          <Form.Item name="user_id" label="选择用户" rules={[{ required: true, message: '请选择用户' }]}>
            <Select
              showSearch
              placeholder="搜索用户名、邮箱或昵称"
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
          <Form.Item name="role" label="角色" rules={[{ required: true, message: '请选择角色' }]}>
            <Select placeholder="选择角色">
              <Option value={1}>管理员</Option>
              <Option value={2}>成员</Option>
              <Option value={3}>观察者</Option>
            </Select>
          </Form.Item>
          <Form.Item name="quota" label="分配额度">
            <Input type="number" placeholder="输入分配额度" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">邀请</Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="分配额度" open={quotaModalVisible} onCancel={() => setQuotaModalVisible(false)} footer={null}>
        <Form form={quotaForm} onFinish={handleAllocateQuota} layout="vertical">
          <Form.Item name="target_user_id" label="用户ID" rules={[{ required: true }]}>
            <Input disabled />
          </Form.Item>
          <Form.Item name="quota" label="分配额度" rules={[{ required: true, message: '请输入额度' }]}>
            <Input type="number" placeholder="输入分配额度" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">分配</Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default Teams