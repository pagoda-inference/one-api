import { useState, useEffect } from 'react'
import { useTheme } from '../contexts/ThemeContext'
import { Card, Table, Button, Modal, Form, Input, InputNumber, Tag, Space, message, Collapse, Empty } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { getInvoices, createInvoice, getTopupOrders, Invoice, TopupOrder, User } from '../services/api'
import { useTranslation } from 'react-i18next'

const { Panel } = Collapse
const { TextArea } = Input
const Invoices: React.FC = () => {
  const { t } = useTranslation()
  const { appTheme } = useTheme()
  const [loading, setLoading] = useState(false)
  const [isAdmin, setIsAdmin] = useState(false)
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [orders, setOrders] = useState<TopupOrder[]>([])
  const [modalVisible, setModalVisible] = useState(false)
  const [createLoading, setCreateLoading] = useState(false)
  const [selectedOrders, setSelectedOrders] = useState<string[]>([])
  const [form] = Form.useForm()

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
      const [invoicesRes, ordersRes] = await Promise.all([
        getInvoices({ limit: 100 }),
        getTopupOrders({ status: 'paid', limit: 100 })
      ])

      setInvoices(invoicesRes.data.data || [])
      setOrders(ordersRes.data.data?.orders || [])

      // Check role from localStorage directly
      const userInfoStr = localStorage.getItem('user_info')
      if (userInfoStr) {
        try {
          const userInfo: User = JSON.parse(userInfoStr)
          setIsAdmin((userInfo.role ?? 0) >= 10)
        } catch (error) {
          console.error('Failed to parse user info:', error)
        }
      }
    } catch (error) {
      console.error('Failed to load data:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: any) => {
    if (selectedOrders.length === 0) {
      message.error(t('invoice.select_order'))
      return
    }

    try {
      setCreateLoading(true)
      await createInvoice({
        ...values,
        order_ids: selectedOrders.join(',')
      })
      message.success(t('invoice.submit_application') + ' ' + t('common.success'))
      setModalVisible(false)
      form.resetFields()
      setSelectedOrders([])
      loadData()
    } catch (error) {
      message.error(t('common.operation_failed'))
    } finally {
      setCreateLoading(false)
    }
  }

  const getStatusTag = (status: string) => {
    const map: Record<string, { color: string; text: string }> = {
      pending: { color: 'orange', text: t('invoice.status_pending') },
      approved: { color: 'blue', text: t('invoice.status_approved') },
      issued: { color: 'green', text: t('invoice.status_issued') },
      rejected: { color: 'red', text: t('invoice.status_rejected') }
    }
    const s = map[status] || { color: 'gray', text: status }
    return <Tag color={s.color}>{s.text}</Tag>
  }

  const columns = [
    { title: t('invoice.invoice_id'), dataIndex: 'id', key: 'id' },
    { title: t('invoice.invoice_title'), dataIndex: 'title', key: 'title' },
    {
      title: t('invoice.invoice_amount'),
      dataIndex: 'amount',
      key: 'amount',
      render: (v: number) => `¥${v.toFixed(2)}`
    },
    {
      title: t('invoice.related_orders'),
      dataIndex: 'order_ids',
      key: 'order_ids',
      render: (ids: string) => ids?.split(',').length + ' orders' || '-'
    },
    { title: t('common.status'), dataIndex: 'status', key: 'status', render: getStatusTag },
    {
      title: t('invoice.create_time'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (v: number) => v ? new Date(v * 1000).toLocaleString() : '-'
    }
  ]

  const orderColumns = [
    { title: t('common.type'), dataIndex: 'id', key: 'id' },
    {
      title: t('topup.amount'),
      dataIndex: 'amount',
      key: 'amount',
      render: (v: number) => `¥${v.toFixed(2)}`
    },
    {
      title: t('topup.paid'),
      dataIndex: 'paid_at',
      key: 'paid_at',
      render: (v: number) => v ? new Date(v * 1000).toLocaleString() : '-'
    }
  ]

  const paidOrders = orders.filter(o => o.status === 'paid')

  // Admin view: only show invoice records
  if (isAdmin) {
    return (
      <div>
        <Card title={t('invoice.invoice_record')}>
          {invoices.length > 0 ? (
            <Table
              dataSource={invoices}
              columns={columns}
              rowKey="id"
              loading={loading}
              pagination={{ pageSize: 20 }}
            />
          ) : (
            <Empty description={t('invoice.no_invoice_records')} />
          )}
        </Card>
      </div>
    )
  }

  // User view: show invoice UI and records
  return (
    <div>
      <Card
        title={t('invoice.invoice_list')}
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setModalVisible(true)}
          >
            {t('invoice.apply_invoice')}
          </Button>
        }
      >
        <Table
          dataSource={invoices}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 10 }}
        />
      </Card>

      <Card title={t('invoice.invoice_instructions')} style={{ marginTop: 16 }}>
        <Collapse defaultActiveKey={['1']}>
          <Panel header={t('invoice.billing_notice')} key="1">
            <ul style={{ color: appTheme.textSecondary, lineHeight: 2 }}>
              <li>{t('invoice.billing_content_1')}</li>
              <li>{t('invoice.tax_rate_note')}</li>
              <li>{t('invoice.invoice_issued_3_5_days')}</li>
              <li>{t('invoice.electronic_invoice_sent_email')}</li>
              <li>{t('invoice.paper_invoice_5_7_days')}</li>
            </ul>
          </Panel>
          <Panel header={t('invoice.billing_process')} key="2">
            <ol style={{ color: appTheme.textSecondary, lineHeight: 2 }}>
              <li>{t('invoice.process_step_1')}</li>
              <li>{t('invoice.process_step_2')}</li>
              <li>{t('invoice.process_step_3')}</li>
              <li>{t('invoice.process_step_4')}</li>
              <li>{t('invoice.process_step_5')}</li>
            </ol>
          </Panel>
          <Panel header={t('invoice.faq')} key="3">
            <div style={{ color: appTheme.textSecondary, lineHeight: 2 }}>
              <p><strong>Q: {t('invoice.q1')}</strong></p>
              <p>A: {t('invoice.a1')}</p>
              <p><strong>Q: {t('invoice.q2')}</strong></p>
              <p>A: {t('invoice.a2')}</p>
              <p><strong>Q: {t('invoice.q3')}</strong></p>
              <p>A: {t('invoice.a3')}</p>
            </div>
          </Panel>
        </Collapse>
      </Card>

      <Modal
        title={t('invoice.apply_invoice')}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={700}
      >
        <Form form={form} onFinish={handleCreate} layout="vertical">
          <Card title={t('invoice.step_1_select_orders')} size="small" style={{ marginBottom: 16 }}>
            <Table
              dataSource={paidOrders}
              columns={orderColumns}
              rowKey="id"
              size="small"
              pagination={{ pageSize: 5 }}
              rowSelection={{
                selectedRowKeys: selectedOrders,
                onChange: (keys) => setSelectedOrders(keys as string[]),
                getCheckboxProps: () => ({ disabled: false })
              }}
              onRow={(record) => ({
                onClick: () => {
                  const newSelection = selectedOrders.includes(record.id)
                    ? selectedOrders.filter(id => id !== record.id)
                    : [...selectedOrders, record.id]
                  setSelectedOrders(newSelection)
                }
              })}
            />
            <div style={{ marginTop: 8, color: appTheme.textSecondary }}>
              {t('invoice.orders_selected', { count: selectedOrders.length, amount: orders.filter(o => selectedOrders.includes(o.id)).reduce((sum, o) => sum + o.amount, 0).toFixed(2) })}
            </div>
          </Card>

          <Card title={t('invoice.step_2_fill_info')} size="small">
            <Form.Item
              name="amount"
              label={t('invoice.invoice_amount')}
              rules={[{ required: true, message: t('invoice.enter_amount') }]}
            >
              <InputNumber
                style={{ width: '100%' }}
                min={0.01}
                max={orders.filter(o => selectedOrders.includes(o.id)).reduce((sum, o) => sum + o.amount, 0)}
                precision={2}
                placeholder={t('invoice.enter_amount')}
              />
            </Form.Item>

            <Form.Item
              name="title"
              label={t('invoice.invoice_title')}
              rules={[{ required: true, message: t('invoice.enter_title') }]}
            >
              <Input placeholder={t('invoice.enter_title_placeholder')} />
            </Form.Item>

            <Form.Item
              name="tax_no"
              label={t('invoice.tax_id')}
              rules={[{ required: true, message: t('invoice.enter_tax_id') }]}
            >
              <Input placeholder={t('invoice.enter_tax_id_placeholder')} />
            </Form.Item>

            <Form.Item name="address" label={t('invoice.billing_address')}>
              <Input placeholder={t('invoice.billing_address_optional')} />
            </Form.Item>

            <Form.Item name="phone" label={t('common.phone')}>
              <Input placeholder={t('invoice.phone_optional')} />
            </Form.Item>

            <Form.Item name="bank" label={t('invoice.bank')}>
              <Input placeholder={t('invoice.bank_optional')} />
            </Form.Item>

            <Form.Item name="account" label={t('invoice.bank_account')}>
              <Input placeholder={t('invoice.bank_account_optional')} />
            </Form.Item>

            <Form.Item name="email" label={t('invoice.receive_email')} rules={[{ type: 'email', message: t('common.valid_email_required') }]}>
              <Input placeholder={t('invoice.email_placeholder')} />
            </Form.Item>

            <Form.Item name="remark" label={t('common.remark')}>
              <TextArea rows={3} placeholder={t('invoice.remark_placeholder')} />
            </Form.Item>
          </Card>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={createLoading}>
                {t('invoice.submit_application')}
              </Button>
              <Button onClick={() => setModalVisible(false)}>
                {t('common.cancel')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default Invoices