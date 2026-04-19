import { useState, useEffect } from 'react'
import { Row, Col, Card, InputNumber, Select, Button, Table, Tag, Modal, message, Result, Empty } from 'antd'
import { WechatOutlined, AlipayOutlined, CreditCardOutlined } from '@ant-design/icons'
import { getTopupOrders, createTopupOrder, cancelTopupOrder, TopupOrder, User } from '../services/api'
import { useTranslation } from 'react-i18next'

const { Option } = Select

const Topup: React.FC = () => {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [isAdmin, setIsAdmin] = useState(false)
  const [orders, setOrders] = useState<TopupOrder[]>([])
  const [amount, setAmount] = useState<number>(100)
  const [payMethod, setPayMethod] = useState<string>('alipay')
  const [modalVisible, setModalVisible] = useState(false)
  const [createdOrder, setCreatedOrder] = useState<TopupOrder | null>(null)

  useEffect(() => {
    checkUserRole()
    loadOrders()
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

  const loadOrders = async () => {
    try {
      setLoading(true)
      const res = await getTopupOrders({ limit: 100 })
      setOrders(res.data.data?.orders || res.data.data || [])

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
      console.error('Failed to load orders:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleTopup = async () => {
    try {
      setLoading(true)
      const res = await createTopupOrder({ amount, pay_method: payMethod })
      const order = res.data.data
      setCreatedOrder(order)
      setModalVisible(true)
      loadOrders()
    } catch (error) {
      message.error(t('topup.create_order_failed'))
    } finally {
      setLoading(false)
    }
  }

  const handleCancel = async (id: string) => {
    try {
      await cancelTopupOrder(id)
      message.success(t('topup.order_cancelled'))
      loadOrders()
    } catch (error) {
      message.error(t('topup.cancel_order_failed'))
    }
  }

  const quickAmounts = [50, 100, 200, 500, 1000, 2000]

  const getStatusTag = (status: string) => {
    const map: Record<string, { color: string; text: string }> = {
      pending: { color: 'orange', text: t('topup.pending_payment') },
      paid: { color: 'green', text: t('topup.paid') },
      cancelled: { color: 'gray', text: t('topup.cancelled') },
      refunded: { color: 'red', text: t('topup.refunded') }
    }
    const s = map[status] || { color: 'gray', text: status }
    return <Tag color={s.color}>{s.text}</Tag>
  }

  const getPayMethodIcon = (method: string) => {
    switch (method) {
      case 'alipay': return <AlipayOutlined />
      case 'wechat': return <WechatOutlined />
      case 'card': return <CreditCardOutlined />
      default: return <CreditCardOutlined />
    }
  }

  const columns = [
    { title: t('common.type'), dataIndex: 'id', key: 'id', width: 200 },
    {
      title: t('topup.amount'),
      dataIndex: 'amount',
      key: 'amount',
      render: (v: number) => `¥${v.toFixed(2)}`
    },
    {
      title: t('topup.quota_obtained'),
      dataIndex: 'quota',
      key: 'quota',
      render: (v: number) => v?.toLocaleString() || '-'
    },
    {
      title: t('topup.payment_method'),
      dataIndex: 'pay_method',
      key: 'pay_method',
      render: (v: string) => (
        <span>
          {getPayMethodIcon(v)} {v === 'alipay' ? t('topup.alipay') : v === 'wechat' ? t('topup.wechat_pay') : t('topup.bank_card')}
        </span>
      )
    },
    { title: t('common.status'), dataIndex: 'status', key: 'status', render: getStatusTag },
    {
      title: t('token.created_time'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (v: number) => v ? new Date(v * 1000).toLocaleString() : '-'
    },
    ...(isAdmin ? [
      {
        title: t('common.action'),
        key: 'action',
        width: 100,
        render: (_: any, record: TopupOrder) => (
          record.status === 'pending' && (
            <Button size="small" danger onClick={() => handleCancel(record.id)}>
              {t('common.cancel')}
            </Button>
          )
        )
      }
    ] : [
      {
        title: t('common.action'),
        key: 'action',
        render: (_: any, record: TopupOrder) => (
          record.status === 'pending' && (
            <Button size="small" danger onClick={() => handleCancel(record.id)}>
              {t('common.cancel')}
            </Button>
          )
        )
      }
    ])
  ]

  // Admin view: only show recharge records
  if (isAdmin) {
    return (
      <div>
        <Card title={t('topup.topup_record')}>
          {orders.length > 0 ? (
            <Table
              dataSource={orders}
              columns={columns}
              rowKey="id"
              loading={loading}
              pagination={{ pageSize: 20 }}
            />
          ) : (
            <Empty description={t('common.no_data')} />
          )}
        </Card>
      </div>
    )
  }

  // User view: show recharge UI and records
  return (
    <div>
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title={t('topup.quick_topup')}>
            <div style={{ marginBottom: 24 }}>
              <div style={{ marginBottom: 8, color: '#666' }}>{t('topup.select_amount')}</div>
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                {quickAmounts.map(v => (
                  <Button
                    key={v}
                    type={amount === v ? 'primary' : 'default'}
                    onClick={() => setAmount(v)}
                  >
                    ¥{v}
                  </Button>
                ))}
              </div>
            </div>

            <div style={{ marginBottom: 24 }}>
              <div style={{ marginBottom: 8, color: '#666' }}>{t('topup.custom_amount')}</div>
              <InputNumber
                style={{ width: '100%' }}
                min={1}
                max={100000}
                value={amount}
                onChange={(v) => setAmount(v || 0)}
                formatter={(value) => `¥ ${value}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
                parser={(value) => value!.replace(/¥\s?|(,*)/g, '') as any}
              />
            </div>

            <div style={{ marginBottom: 24 }}>
              <div style={{ marginBottom: 8, color: '#666' }}>{t('topup.payment_method')}</div>
              <Select
                style={{ width: '100%' }}
                value={payMethod}
                onChange={setPayMethod}
              >
                <Option value="alipay">
                  <span><AlipayOutlined /> {t('topup.alipay')}</span>
                </Option>
                <Option value="wechat">
                  <span><WechatOutlined /> {t('topup.wechat_pay')}</span>
                </Option>
                <Option value="card">
                  <span><CreditCardOutlined /> {t('topup.bank_card')}</span>
                </Option>
              </Select>
            </div>

            <Button
              type="primary"
              size="large"
              block
              loading={loading}
              onClick={handleTopup}
            >
              {t('topup.topup_now')} ¥{amount}
            </Button>

            <div style={{ marginTop: 16, color: '#999', fontSize: 12 }}>
              <p>{t('topup.1rmb_7200quota')}</p>
              <p>{t('topup.no_handling_fee')}</p>
            </div>
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title={t('topup.topup_ratio')}>
            <div style={{ color: '#666' }}>
              <h4>{t('topup.topup_ratio')}</h4>
              <p>{t('topup.1rmb_7200quota')}</p>

              <h4 style={{ marginTop: 16 }}>{t('topup.arrival_time')}</h4>
              <p>{t('topup.alipay_wechat_instant')}</p>
              <p>{t('topup.bank_card_1_3_days')}</p>

              <h4 style={{ marginTop: 16 }}>{t('menu.invoice_management')}</h4>
              <p>{t('topup.invoice_note_1')}</p>

              <h4 style={{ marginTop: 16 }}>{t('topup.refund_policy')}</h4>
              <p>{t('topup.refund_note_1')}</p>
            </div>
          </Card>
        </Col>
      </Row>

      <Card title={t('topup.topup_record')} style={{ marginTop: 16 }}>
        <Table
          dataSource={orders}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 10 }}
        />
      </Card>

      <Modal
        title={t('topup.order_created')}
        open={modalVisible}
        footer={[
          <Button key="close" onClick={() => setModalVisible(false)}>
            {t('common.close')}
          </Button>,
          <Button key="pay" type="primary" href={createdOrder?.payment_url} target="_blank">
            {t('topup.go_to_payment')}
          </Button>
        ]}
        onCancel={() => setModalVisible(false)}
      >
        {createdOrder && (
          <Result
            status="success"
            title={t('topup.order_created_success')}
            subTitle={`${t('common.type')}: ${createdOrder.id}`}
            extra={[
              <p key="amount">{t('topup.amount')}: <strong>¥{createdOrder.amount}</strong></p>,
              <p key="quota">{t('topup.quota_obtained')}: <strong>{createdOrder.quota?.toLocaleString()} quota</strong></p>,
              <p key="expire">{t('topup.please_pay_within', { time: createdOrder.expired_at ? new Date(createdOrder.expired_at * 1000).toLocaleString() : t('topup.within_24_hours') })}</p>
            ]}
          />
        )}
      </Modal>
    </div>
  )
}

export default Topup