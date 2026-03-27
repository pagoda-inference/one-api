import { useState, useEffect } from 'react'
import { Row, Col, Card, InputNumber, Select, Button, Table, Tag, Modal, message, Result } from 'antd'
import { WechatOutlined, AlipayOutlined, CreditCardOutlined } from '@ant-design/icons'
import { getTopupOrders, createTopupOrder, cancelTopupOrder, TopupOrder } from '../services/api'

const { Option } = Select

const Topup: React.FC = () => {
  const [loading, setLoading] = useState(false)
  const [orders, setOrders] = useState<TopupOrder[]>([])
  const [amount, setAmount] = useState<number>(100)
  const [payMethod, setPayMethod] = useState<string>('alipay')
  const [modalVisible, setModalVisible] = useState(false)
  const [createdOrder, setCreatedOrder] = useState<TopupOrder | null>(null)

  useEffect(() => {
    loadOrders()
  }, [])

  const loadOrders = async () => {
    try {
      setLoading(true)
      const res = await getTopupOrders({ limit: 20 })
      setOrders(res.data.data.orders || [])
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
      message.error('创建订单失败')
    } finally {
      setLoading(false)
    }
  }

  const handleCancel = async (id: string) => {
    try {
      await cancelTopupOrder(id)
      message.success('订单已取消')
      loadOrders()
    } catch (error) {
      message.error('取消失败')
    }
  }

  const quickAmounts = [50, 100, 200, 500, 1000, 2000]

  const getStatusTag = (status: string) => {
    const map: Record<string, { color: string; text: string }> = {
      pending: { color: 'orange', text: '待支付' },
      paid: { color: 'green', text: '已支付' },
      cancelled: { color: 'gray', text: '已取消' },
      refunded: { color: 'red', text: '已退款' }
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
    { title: '订单号', dataIndex: 'id', key: 'id', width: 200 },
    {
      title: '金额',
      dataIndex: 'amount',
      key: 'amount',
      render: (v: number) => `¥${v.toFixed(2)}`
    },
    {
      title: '获得额度',
      dataIndex: 'quota',
      key: 'quota',
      render: (v: number) => v.toLocaleString()
    },
    {
      title: '支付方式',
      dataIndex: 'pay_method',
      key: 'pay_method',
      render: (v: string) => (
        <span>
          {getPayMethodIcon(v)} {v === 'alipay' ? '支付宝' : v === 'wechat' ? '微信支付' : '银行卡'}
        </span>
      )
    },
    { title: '状态', dataIndex: 'status', key: 'status', render: getStatusTag },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (v: number) => new Date(v * 1000).toLocaleString()
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: TopupOrder) => (
        record.status === 'pending' && (
          <Button size="small" danger onClick={() => handleCancel(record.id)}>
            取消
          </Button>
        )
      )
    }
  ]

  return (
    <div>
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="快速充值">
            <div style={{ marginBottom: 24 }}>
              <div style={{ marginBottom: 8, color: '#666' }}>选择金额</div>
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
              <div style={{ marginBottom: 8, color: '#666' }}>自定义金额</div>
              <InputNumber
                style={{ width: '100%' }}
                min={1}
                max={100000}
                value={amount}
                onChange={(v) => setAmount(v || 0)}
                prefix="¥"
                formatter={(value) => `¥ ${value}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
                parser={(value) => value!.replace(/¥\s?|(,*)/g, '') as any}
              />
            </div>

            <div style={{ marginBottom: 24 }}>
              <div style={{ marginBottom: 8, color: '#666' }}>支付方式</div>
              <Select
                style={{ width: '100%' }}
                value={payMethod}
                onChange={setPayMethod}
              >
                <Option value="alipay">
                  <span><AlipayOutlined /> 支付宝</span>
                </Option>
                <Option value="wechat">
                  <span><WechatOutlined /> 微信支付</span>
                </Option>
                <Option value="card">
                  <span><CreditCardOutlined /> 银行卡</span>
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
              立即充值 ¥{amount}
            </Button>

            <div style={{ marginTop: 16, color: '#999', fontSize: 12 }}>
              <p>1元 = 7,200 quota (约等于1美元)</p>
              <p>充值金额将直接到账，无手续费</p>
            </div>
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="充值说明">
            <div style={{ color: '#666' }}>
              <h4>充值比例</h4>
              <p>1元 = 7,200 quota</p>
              <p>相当于 1美元 = 7.2元 的超低汇率</p>

              <h4 style={{ marginTop: 16 }}>到账时间</h4>
              <p>支付宝/微信支付：即时到账</p>
              <p>银行卡支付：1-3个工作日</p>

              <h4 style={{ marginTop: 16 }}>发票说明</h4>
              <p>充值成功后，可在"发票管理"页面申请开具发票</p>
              <p>发票内容为"技术服务费"</p>

              <h4 style={{ marginTop: 16 }}>退款政策</h4>
              <p>如需退款，请联系客服</p>
              <p>退款将在3-7个工作日内原路返回</p>
            </div>
          </Card>
        </Col>
      </Row>

      <Card title="充值记录" style={{ marginTop: 16 }}>
        <Table
          dataSource={orders}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 10 }}
        />
      </Card>

      <Modal
        title="订单已创建"
        open={modalVisible}
        footer={[
          <Button key="close" onClick={() => setModalVisible(false)}>
            关闭
          </Button>,
          <Button key="pay" type="primary" href={createdOrder?.payment_url} target="_blank">
            前往支付
          </Button>
        ]}
        onCancel={() => setModalVisible(false)}
      >
        {createdOrder && (
          <Result
            status="success"
            title="订单创建成功"
            subTitle={`订单号: ${createdOrder.id}`}
            extra={[
              <p key="amount">充值金额: <strong>¥{createdOrder.amount}</strong></p>,
              <p key="quota">获得额度: <strong>{createdOrder.quota.toLocaleString()} quota</strong></p>,
              <p key="expire">请在 {createdOrder.expired_at ? new Date(createdOrder.expired_at * 1000).toLocaleString() : '24小时内'} 前完成支付</p>
            ]}
          />
        )}
      </Modal>
    </div>
  )
}

export default Topup