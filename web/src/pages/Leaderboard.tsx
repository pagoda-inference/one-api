import { useEffect, useMemo, useState } from 'react'
import { Button, Card, Col, Empty, Row, message, Popconfirm, Space, Table, Tabs, Typography } from 'antd'
import { useTranslation } from 'react-i18next'
import { useTheme } from '../contexts/ThemeContext'
import api from '../services/api'

interface LeaderboardDim {
  name?: string
  acc?: number
  score?: number
  correct?: number
  total?: number
  weight?: number
}

interface LeaderboardRow {
  id?: string
  model?: string
  model_name?: string
  score?: number
  total_score?: number
  total_correct?: number
  total_items?: number
  dims?: LeaderboardDim[]
  domain?: string
  model_type?: string
  timestamp?: string
  evaluated_at?: string
  created_at?: string
  [key: string]: any
}

const rankDomains = ['general', 'general_mm', 'medical', 'skin', 'weighted']
const OVERALL_KEY = '__overall__'

const formatModelName = (name?: string) => {
  if (!name) return '-'
  if (name.includes('/')) return name.split('/').pop() || name
  return name
}

const normalizeLeaderboardRow = (item: any): LeaderboardRow => {
  const report = item?.report || {}
  return {
    ...item,
    model: item?.model || report?.model || item?.model_name,
    score: report?.total_score ?? item?.score ?? item?.total_score,
    total_score: report?.total_score ?? item?.total_score ?? item?.score,
    total_correct: report?.total_correct ?? item?.total_correct,
    total_items: report?.total_items ?? item?.total_items,
    dims: Array.isArray(report?.dims) ? report.dims : item?.dims,
    domain: report?.domain ?? item?.domain,
    model_type: report?.model_type ?? item?.model_type,
    timestamp: item?.timestamp ?? report?.timestamp ?? item?.evaluated_at ?? item?.created_at,
  }
}

const modelTypeOf = (item: LeaderboardRow) => item.model_type || item.report?.model_type || ''
const formatScoreOneDecimal = (value: any) => {
  const n = Number(value)
  if (Number.isFinite(n)) return n.toFixed(1)
  return '-'
}
const getMetricAcc = (row: LeaderboardRow, metricName: string) => {
  if (!Array.isArray(row.dims)) return null
  const dim = row.dims.find((d) => d.name === metricName)
  if (!dim) return null
  const n = Number(dim.acc)
  return Number.isFinite(n) ? n : null
}

const Leaderboard: React.FC = () => {
  const { t } = useTranslation()
  const { appTheme } = useTheme()
  const [loading, setLoading] = useState(false)
  const [domain, setDomain] = useState('general')
  const [metric, setMetric] = useState(OVERALL_KEY)
  const [rows, setRows] = useState<LeaderboardRow[]>([])
  const [hiddenIds, setHiddenIds] = useState<string[]>([])
  const user = useMemo(() => {
    try {
      return JSON.parse(localStorage.getItem('user_info') || '{}')
    } catch {
      return {}
    }
  }, [])
  const isRoot = Number(user?.role || 0) >= 100

  const fetchDomainRows = async (d: string) => {
    const res = await api.get('/leaderboard', { params: { domain: d } })
    const data = res?.data?.data ?? res?.data ?? []
    const list = Array.isArray(data) ? data : Array.isArray(data?.items) ? data.items : Array.isArray(data?.rows) ? data.rows : []
    const normalized = list.map((item: any) => normalizeLeaderboardRow(item))
    return normalized.filter((item: LeaderboardRow) => !hiddenIds.includes(String(item.id || item.model || '')))
  }

  const load = async (d = domain) => {
    setLoading(true)
    try {
      if (d === 'general_mm') {
        const generalRows = await fetchDomainRows('general')
        setRows(generalRows.filter((item: LeaderboardRow) => modelTypeOf(item) !== 'text'))
      } else if (d === 'weighted') {
        const [generalRows, medicalRows] = await Promise.all([fetchDomainRows('general'), fetchDomainRows('medical')])
        if (generalRows.length === 0 || medicalRows.length === 0) {
          setRows([])
        } else {
          const weightedRows = await fetchDomainRows('weighted')
          setRows(weightedRows)
        }
      } else {
        setRows(await fetchDomainRows(d))
      }
    } catch {
      message.error(t('leaderboard.load_failed'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load(domain)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [domain, hiddenIds])

  const metricTabs = useMemo(() => {
    const first = rows[0]
    const dims = Array.isArray(first?.dims) ? first.dims : []
    return [
      { key: OVERALL_KEY, label: t('leaderboard.metric_overall') },
      ...dims.map((d) => ({ key: d.name || '', label: d.name || '-' })).filter((x) => x.key),
    ]
  }, [rows, t])

  const displayRows = useMemo(() => {
    const source = [...rows]
    if (metric === OVERALL_KEY) return source
    return source.sort((a, b) => {
      const aDim = (a.dims || []).find((d) => d.name === metric)
      const bDim = (b.dims || []).find((d) => d.name === metric)
      return (bDim?.acc || 0) - (aDim?.acc || 0)
    })
  }, [rows, metric])

  const handleDelete = async (record: LeaderboardRow) => {
    const id = String(record.id || record.task_id || record.model || '')
    if (!id) return
    try {
      const nextHidden = Array.from(new Set([...hiddenIds, id]))
      await api.put('/option/', {
        key: 'LeaderboardHiddenIds',
        value: JSON.stringify(nextHidden),
      })
      setHiddenIds(nextHidden)
      message.success(t('common.delete') + t('common.success'))
    } catch (e: any) {
      message.error(e?.response?.data?.message || t('common.operation_failed'))
    }
  }

  useEffect(() => {
    const loadHidden = async () => {
      try {
        const res = await api.get('/option/')
        const options = res?.data?.data || []
        const found = options.find((item: any) => item.key === 'LeaderboardHiddenIds')
        if (found?.value) {
          const ids = JSON.parse(found.value)
          if (Array.isArray(ids)) setHiddenIds(ids.map(String))
        }
      } catch {}
    }
    loadHidden()
  }, [])

  const columns: any[] = [
    {
      title: t('leaderboard.rank'),
      key: 'rank',
      width: 100,
      render: (_: any, __: LeaderboardRow, idx: number) => {
        const isTop = idx < 3
        const medalStyle = idx === 0
          ? { top: '#ffe68a', bottom: '#c98711', text: '#7a4b00' }
          : idx === 1
            ? { top: '#f3f6fb', bottom: '#9aa5b6', text: '#626c7b' }
            : { top: '#ffcf9a', bottom: '#b86a2c', text: '#7a3f11' }
        return (
          isTop ? (
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2 }}>
              <div style={{ display: 'flex', gap: 2, transform: 'translateY(1px)' }}>
                <span style={{ width: 5, height: 10, borderRadius: 999, background: `linear-gradient(180deg, ${medalStyle.top}, ${medalStyle.bottom})`, opacity: 0.95 }} />
                <span style={{ width: 5, height: 10, borderRadius: 999, background: `linear-gradient(180deg, ${medalStyle.top}, ${medalStyle.bottom})`, opacity: 0.95 }} />
              </div>
              <div
                style={{
                  width: 28,
                  height: 28,
                  borderRadius: '50%',
                  background: `linear-gradient(180deg, ${medalStyle.top}, ${medalStyle.bottom})`,
                  color: medalStyle.text,
                  border: '1px solid rgba(255,255,255,0.26)',
                  display: 'inline-flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 13,
                  fontWeight: 800,
                  boxShadow: '0 8px 16px rgba(0,0,0,0.16)',
                }}
              >
                {idx + 1}
              </div>
            </div>
          ) : (
            <span style={{ color: '#6b72d6', fontSize: 15, fontWeight: 700 }}>#{idx + 1}</span>
          )
        )
      },
    },
    {
      title: t('leaderboard.model'),
      key: 'model',
      width: 280,
      render: (_: any, r: LeaderboardRow) => {
        const modelName = formatModelName(r.model_name || r.model)
        const totalText = r.total_correct != null && r.total_items != null ? `${r.total_correct}/${r.total_items}` : '-'
        const modelType = r.model_type === 'text' ? t('leaderboard.pure_text') : (r.model_type || '-')
        return (
          <Space direction="vertical" size={0}>
            <span style={{ fontWeight: 700, fontSize: 16, lineHeight: 1.2 }}>{modelName}</span>
            <span style={{ color: appTheme.textTertiary, fontSize: 12 }}>
              {modelType} · {totalText} {t('leaderboard.questions')}
            </span>
          </Space>
        )
      },
    },
    {
      title: t('leaderboard.score'),
      key: 'score',
      width: 120,
      render: (_: any, r: LeaderboardRow) => {
        const value = metric === OVERALL_KEY
          ? (r.score ?? r.total_score)
          : getMetricAcc(r, metric)
        return (
          <span style={{ color: '#6366f1', fontWeight: 800, fontSize: 22, lineHeight: 1 }}>
            {formatScoreOneDecimal(value)}
          </span>
        )
      },
    },
    {
      title: t('leaderboard.metrics'),
      key: 'metrics',
      render: (_: any, r: LeaderboardRow) => {
        const dims = Array.isArray(r.dims) ? r.dims : []
        return (
          <Row gutter={[8, 2]} style={{ maxWidth: 360 }}>
            {dims.map((dim) => (
              <Col key={dim.name} span={12}>
                <div style={{ display: 'flex', justifyContent: 'space-between', gap: 6, color: appTheme.textPrimary }}>
                  <span style={{ color: appTheme.textSecondary }}>{dim.name}</span>
                  <span style={{ fontWeight: 700 }}>{dim.acc?.toFixed(1)}</span>
                </div>
              </Col>
            ))}
          </Row>
        )
      },
    },
    {
      title: t('leaderboard.time'),
      key: 'time',
      width: 200,
      render: (_: any, r: LeaderboardRow) => r.timestamp || r.evaluated_at || r.created_at || '-',
    },
  ]

  if (isRoot) {
    columns.push({
      title: t('common.action'),
      key: 'action',
      width: 120,
      render: (_: any, r: LeaderboardRow) => (
        <Popconfirm title={t('common.confirm_delete')} onConfirm={() => handleDelete(r)}>
          <Button danger size="small">{t('common.delete')}</Button>
        </Popconfirm>
      ),
    })
  }

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Typography.Title level={2} style={{ margin: 0, color: appTheme.textPrimary }}>
          模型评测排行榜
        </Typography.Title>
        <div style={{ marginTop: 6, color: appTheme.textSecondary, fontSize: 14 }}>
          {t('leaderboard.platform_desc')}
        </div>
      </div>
      <Card
        style={{ borderRadius: 12, background: appTheme.bgContainer }}
        title={
          <div>
            <Tabs
              activeKey={domain}
              onChange={(key) => {
                setDomain(key)
                setMetric(OVERALL_KEY)
              }}
              items={rankDomains.map((d) => ({ key: d, label: t(`leaderboard.domain_${d}`) }))}
            />
            <div style={{ marginTop: 6, color: appTheme.textPrimary, fontWeight: 700, fontSize: 18 }}>
              {t('leaderboard.subtitle')}
            </div>
            <Tabs
              activeKey={metric}
              onChange={setMetric}
              items={metricTabs}
              style={{ marginTop: 12 }}
            />
          </div>
        }
        extra={<Button onClick={() => load()}>{t('common.refresh')}</Button>}
      >
        <Table
          rowKey={(r) => String(r.id || r.model || Math.random())}
          loading={loading}
          dataSource={displayRows}
          columns={columns}
          pagination={{ pageSize: 20 }}
          scroll={{ x: 1200 }}
          locale={{
            emptyText: domain !== 'general' && domain !== 'general_mm'
              ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('leaderboard.no_domain_data')} />
              : t('common.no_data'),
          }}
        />
      </Card>
    </div>
  )
}

export default Leaderboard
