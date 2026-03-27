import { useState, useEffect } from 'react'
import { Card, Table, Button, Modal, Form, Input, InputNumber, Tag, Space, message, Collapse } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { getInvoices, createInvoice, getTopupOrders, Invoice, TopupOrder } from '../services/api'

const { Panel } = Collapse
const { TextArea } = Input
const Invoices: React.FC = () => {
  const [loading, setLoading] = useState(false)
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [orders, setOrders] = useState<TopupOrder[]>([])
  const [modalVisible, setModalVisible] = useState(false)
  const [createLoading, setCreateLoading] = useState(false)
  const [selectedOrders, setSelectedOrders] = useState<string[]>([])
  const [form] = Form.useForm()

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      setLoading(true)
      const [invoicesRes, ordersRes] = await Promise.all([
        getInvoices({ limit: 50 }),
        getTopupOrders({ status: 'paid', limit: 100 })
      ])

      setInvoices(invoicesRes.data.data || [])
      setOrders(ordersRes.data.data.orders || [])
    } catch (error) {
      console.error('Failed to load data:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (values: any) => {
    if (selectedOrders.length === 0) {
      message.error('请选择关联订单')
      return
    }

    try {
      setCreateLoading(true)
      await createInvoice({
        ...values,
        order_ids: selectedOrders.join(',')
      })
      message.success('发票申请提交成功')
      setModalVisible(false)
      form.resetFields()
      setSelectedOrders([])
      loadData()
    } catch (error) {
      message.error('提交失败')
    } finally {
      setCreateLoading(false)
    }
  }

  const getStatusTag = (status: string) => {
    const map: Record<string, { color: string; text: string }> = {
      pending: { color: 'orange', text: '待处理' },
      approved: { color: 'blue', text: '已批准' },
      issued: { color: 'green', text: '已开票' },
      rejected: { color: 'red', text: '已驳回' }
    }
    const s = map[status] || { color: 'gray', text: status }
    return <Tag color={s.color}>{s.text}</Tag>
  }

  const columns = [
    { title: '发票号', dataIndex: 'id', key: 'id' },
    { title: '发票抬头', dataIndex: 'title', key: 'title' },
    {
      title: '发票金额',
      dataIndex: 'amount',
      key: 'amount',
      render: (v: number) => `¥${v.toFixed(2)}`
    },
    {
      title: '关联订单',
      dataIndex: 'order_ids',
      key: 'order_ids',
      render: (ids: string) => ids.split(',').length + ' 个订单'
    },
    { title: '状态', dataIndex: 'status', key: 'status', render: getStatusTag },
    {
      title: '申请时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (v: number) => new Date(v * 1000).toLocaleString()
    }
  ]

  const orderColumns = [
    { title: '订单号', dataIndex: 'id', key: 'id' },
    {
      title: '金额',
      dataIndex: 'amount',
      key: 'amount',
      render: (v: number) => `¥${v.toFixed(2)}`
    },
    {
      title: '支付时间',
      dataIndex: 'paid_at',
      key: 'paid_at',
      render: (v: number) => v ? new Date(v * 1000).toLocaleString() : '-'
    }
  ]

  const paidOrders = orders.filter(o => o.status === 'paid')

  return (
    <div>
      <Card
        title="发票列表"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setModalVisible(true)}
          >
            申请发票
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

      <Card title="发票说明" style={{ marginTop: 16 }}>
        <Collapse defaultActiveKey={['1']}>
          <Panel header="开票须知" key="1">
            <ul style={{ color: '#666', lineHeight: 2 }}>
              <li>发票内容为"技术服务费"或"API服务费"</li>
              <li>发票税率按国家规定执行</li>
              <li>发票将在申请通过后 3-5 个工作日内开具</li>
              <li>电子发票将发送至您提供的邮箱</li>
              <li>纸质发票将在 5-7 个工作日内寄出</li>
            </ul>
          </Panel>
          <Panel header="开票流程" key="2">
            <ol style={{ color: '#666', lineHeight: 2 }}>
              <li>选择需要开票的充值订单</li>
              <li>填写发票抬头信息</li>
              <li>提交申请等待审核</li>
              <li>审核通过后开具发票</li>
              <li>收到发票（电子/纸质）</li>
            </ol>
          </Panel>
          <Panel header="常见问题" key="3">
            <div style={{ color: '#666', lineHeight: 2 }}>
              <p><strong>Q: 可以开具增值税专用发票吗？</strong></p>
              <p>A: 目前仅支持开具增值税普通发票</p>
              <p><strong>Q: 发票可以跨年开具吗？</strong></p>
              <p>A: 可以，但需在充值当年内申请</p>
              <p><strong>Q: 发票抬头可以修改吗？</strong></p>
              <p>A: 已开具的发票抬头无法修改，请仔细核对</p>
            </div>
          </Panel>
        </Collapse>
      </Card>

      <Modal
        title="申请发票"
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={700}
      >
        <Form form={form} onFinish={handleCreate} layout="vertical">
          <Card title="1. 选择关联订单" size="small" style={{ marginBottom: 16 }}>
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
            <div style={{ marginTop: 8, color: '#666' }}>
              已选择 {selectedOrders.length} 个订单，总金额 ¥
              {orders
                .filter(o => selectedOrders.includes(o.id))
                .reduce((sum, o) => sum + o.amount, 0)
                .toFixed(2)}
            </div>
          </Card>

          <Card title="2. 填写发票信息" size="small">
            <Form.Item
              name="amount"
              label="开票金额"
              rules={[{ required: true, message: '请输入开票金额' }]}
            >
              <InputNumber
                style={{ width: '100%' }}
                min={0.01}
                max={orders.filter(o => selectedOrders.includes(o.id)).reduce((sum, o) => sum + o.amount, 0)}
                precision={2}
                placeholder="请输入开票金额"
              />
            </Form.Item>

            <Form.Item
              name="title"
              label="发票抬头"
              rules={[{ required: true, message: '请输入发票抬头' }]}
            >
              <Input placeholder="请输入公司或个人名称" />
            </Form.Item>

            <Form.Item
              name="tax_no"
              label="纳税人识别号"
              rules={[{ required: true, message: '请输入纳税人识别号' }]}
            >
              <Input placeholder="请输入统一社会信用代码" />
            </Form.Item>

            <Form.Item name="address" label="开票地址">
              <Input placeholder="请输入开票地址（选填）" />
            </Form.Item>

            <Form.Item name="phone" label="电话">
              <Input placeholder="请输入联系电话（选填）" />
            </Form.Item>

            <Form.Item name="bank" label="开户行">
              <Input placeholder="请输入开户银行（选填）" />
            </Form.Item>

            <Form.Item name="account" label="银行账号">
              <Input placeholder="请输入银行账号（选填）" />
            </Form.Item>

            <Form.Item name="email" label="接收邮箱" rules={[{ type: 'email', message: '请输入有效的邮箱地址' }]}>
              <Input placeholder="用于接收电子发票" />
            </Form.Item>

            <Form.Item name="remark" label="备注">
              <TextArea rows={3} placeholder="如有其他要求请在此说明" />
            </Form.Item>
          </Card>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={createLoading}>
                提交申请
              </Button>
              <Button onClick={() => setModalVisible(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default Invoices