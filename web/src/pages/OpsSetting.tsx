import { useState, useEffect } from 'react'
import { useTheme } from '../contexts/ThemeContext'
import { Row, Col, Card, Form, Input, InputNumber, Button, message, Tabs, Switch } from 'antd'
import { SaveOutlined } from '@ant-design/icons'
import { getOptions, updateOption } from '../services/api'
import { useTranslation } from 'react-i18next'

const { TextArea } = Input

const OpsSetting: React.FC = () => {
  const { t } = useTranslation()
  const { appTheme } = useTheme()
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
            return Promise.reject(new Error(res.data.message || `${t('ops.update_failed')}: ${key}`))
          }
          return res
        })
      })

      await Promise.all(updatePromises)
      message.success(t('ops.settings_saved_success'))
      loadOptions()
    } catch (error: any) {
      message.error(error.message || t('ops.save_failed'))
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
      label: t('ops.quota_settings'),
      children: (
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
        </Card>
      )
    },
    {
      key: 'ratio',
      label: t('ops.multiplier_settings'),
      children: (
        <>
          <Card title={t('ops.group_multiplier')} style={{ marginTop: 16 }}>
            <Form.Item
              name="GroupRatio"
              rules={[{ validator: (_, value) => verifyJSON(value) ? Promise.resolve() : Promise.reject(t('ops.not_valid_json')) }]}
            >
              <TextArea
                rows={6}
                placeholder={'{\n  "default": 1.0,\n  "enterprise": 1.2\n}'}
                style={{ fontFamily: 'monospace' }}
              />
            </Form.Item>
            <div style={{ color: appTheme.textTertiary, fontSize: 12 }}>
              {t('ops.json_format_example')}
            </div>
          </Card>

        </>
      )
    },
    {
      key: 'system',
      label: t('ops.system_settings'),
      children: (
        <>
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
        </>
      )
    }
  ]

  return (
    <div>
      <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
        <Col>
          <h2 style={{ margin: 0 }}>{t('ops.system_settings')}</h2>
        </Col>
        <Col>
          <Button
            type="primary"
            icon={<SaveOutlined />}
            loading={saving}
            onClick={handleSave}
          >
            {t('ops.save_settings')}
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