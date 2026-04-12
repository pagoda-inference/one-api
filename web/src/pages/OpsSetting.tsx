import { useState, useEffect } from 'react'
import { Row, Col, Card, Form, Input, InputNumber, Button, message, Tabs, Switch } from 'antd'
import { SaveOutlined } from '@ant-design/icons'
import { getOptions, updateOption } from '../services/api'

const { TextArea } = Input

const OpsSetting: React.FC = () => {
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()
  const [activeTab, setActiveTab] = useState('quota')

  const [inputs, setInputs] = useState({
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
    loadOptions()
  }, [])

  const loadOptions = async () => {
    try {
      const res = await getOptions()
      if (res.data.success) {
        const data = res.data.data || []
        const newInputs: any = {}
        data.forEach((item: any) => {
          if (item.key === 'GroupRatio' || item.key === 'ModelRatio' || item.key === 'CompletionRatio') {
            try {
              newInputs[item.key] = JSON.stringify(JSON.parse(item.value), null, 2)
            } catch {
              newInputs[item.key] = item.value || ''
            }
          } else if (item.key === 'AutomaticDisableChannelEnabled' || item.key === 'AutomaticEnableChannelEnabled' || item.key === 'LogConsumeEnabled' || item.key === 'DisplayInCurrencyEnabled' || item.key === 'DisplayTokenStatEnabled') {
            newInputs[item.key] = item.value === 'true' || item.value === '1'
          } else {
            newInputs[item.key] = item.value ? parseFloat(item.value) : 0
          }
        })
        setInputs((prev: any) => ({ ...prev, ...newInputs }))
        form.setFieldsValue(newInputs)
      }
    } catch (error) {
      console.error('Failed to load options:', error)
    }
  }

  const handleSave = async () => {
    try {
      setSaving(true)
      const values = form.getFieldsValue()

      // Update each option
      const updatePromises = Object.entries(values as Record<string, any>).map(([key, value]) => {
        let finalValue = value
        if (typeof value === 'boolean') {
          finalValue = value ? 'true' : 'false'
        } else if (typeof value === 'object') {
          finalValue = JSON.stringify(value)
        } else {
          finalValue = String(value)
        }
        return updateOption(key, finalValue).then(res => {
          if (!res.data.success) {
            return Promise.reject(new Error(res.data.message || `更新 ${key} 失败`))
          }
          return res
        })
      })

      await Promise.all(updatePromises)
      message.success('设置保存成功')
      loadOptions()
    } catch (error: any) {
      message.error(error.message || '保存失败')
      console.error('Save error:', error)
    } finally {
      setSaving(false)
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

  const tabItems = [
    {
      key: 'quota',
      label: '配额设置',
      children: (
        <Card title="配额配置" style={{ marginTop: 16 }}>
          <Row gutter={[16, 16]}>
            <Col span={12}>
              <Form.Item label="新用户初始配额" name="QuotaForNewUser">
                <InputNumber style={{ width: '100%' }} placeholder="-1 表示无限额" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="预消耗配额" name="PreConsumedQuota">
                <InputNumber style={{ width: '100%' }} placeholder="请求前预扣除的配额" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="邀请者奖励配额" name="QuotaForInviter">
                <InputNumber style={{ width: '100%' }} placeholder="邀请者获得的奖励" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="被邀请者奖励配额" name="QuotaForInvitee">
                <InputNumber style={{ width: '100%' }} placeholder="被邀请者获得的奖励" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="配额提醒阈值" name="QuotaRemindThreshold">
                <InputNumber style={{ width: '100%' }} placeholder="配额低于此值时提醒" />
              </Form.Item>
            </Col>
          </Row>
        </Card>
      )
    },
    {
      key: 'ratio',
      label: '倍率设置',
      children: (
        <>
          <Card title="分组倍率" style={{ marginTop: 16 }}>
            <Form.Item
              name="GroupRatio"
              rules={[{ validator: (_, value) => verifyJSON(value) ? Promise.resolve() : Promise.reject('不是合法的 JSON') }]}
            >
              <TextArea
                rows={6}
                placeholder={'{\n  "default": 1.0,\n  "enterprise": 1.2\n}'}
                style={{ fontFamily: 'monospace' }}
              />
            </Form.Item>
            <div style={{ color: '#999', fontSize: 12 }}>
              JSON 格式，键为分组名称，值为倍率。例如 {" "}
              {'{"default": 1.0}'} 表示默认分组倍率为 1.0
            </div>
          </Card>

        </>
      )
    },
    {
      key: 'system',
      label: '系统设置',
      children: (
        <>
          <Card title="渠道自动管理" style={{ marginTop: 16 }}>
            <Row gutter={[16, 16]}>
              <Col span={12}>
                <Form.Item label="自动禁用渠道" name="AutomaticDisableChannelEnabled" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label="自动启用渠道" name="AutomaticEnableChannelEnabled" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label="渠道禁用阈值(秒)" name="ChannelDisableThreshold">
                  <InputNumber style={{ width: '100%' }} placeholder="响应时间超过此值自动禁用" />
                </Form.Item>
              </Col>
            </Row>
          </Card>

          <Card title="显示设置" style={{ marginTop: 16 }}>
            <Row gutter={[16, 16]}>
              <Col span={12}>
                <Form.Item label="以货币形式显示" name="DisplayInCurrencyEnabled" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label="显示 Token 统计" name="DisplayTokenStatEnabled" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label="消费日志" name="LogConsumeEnabled" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label="重试次数" name="RetryTimes">
                  <InputNumber style={{ width: '100%' }} min={0} max={10} />
                </Form.Item>
              </Col>
            </Row>
          </Card>
        </>
      )
    }
  ]

  return (
    <div>
      <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
        <Col>
          <h2 style={{ margin: 0 }}>系统设置</h2>
        </Col>
        <Col>
          <Button
            type="primary"
            icon={<SaveOutlined />}
            loading={saving}
            onClick={handleSave}
          >
            保存设置
          </Button>
        </Col>
      </Row>

      <Form form={form} initialValues={inputs} layout="vertical">
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={tabItems}
        />
      </Form>
    </div>
  )
}

export default OpsSetting