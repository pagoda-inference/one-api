import { useEffect, useState } from 'react'
import { Alert, Button, Card, Input, InputNumber, List, Spin, message, Select, Space, Tag } from 'antd'
import { SyncOutlined, VideoCameraOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useTheme } from '../contexts/ThemeContext'
import axios from 'axios'
import { getPlaygroundModels } from '../services/api'

interface PlaygroundModel {
  id: string
  name: string
  model_type: string
}

interface VideoTaskItem {
  id: string
  model: string
  prompt: string
  status: string
  createdAt: string
  createdAtMs?: number
  taskId?: string
  raw?: any
  videoUrl?: string
}

const T2V_MODEL_ID = 'bedi/wan2.2-t2v-a14b'
const DEFAULT_NEGATIVE_PROMPT = '色调艳丽,过曝,静态,细节模糊不清,字幕,风格,作品,画作,画面,静止,整体发灰,最差质量,低质量,JPEG压缩残留,丑陋的,残缺的,多余的手指,画得不好的手部,画得不好的脸部,畸形的,毁容的,形态畸形的肢体,手指融合,静止不动的画面,杂乱的背景,三条腿,背景人很多,倒着走'
const QUICK_PROMPTS = [
  'A young woman in traditional costume smiles in warm sunlight, cinematic, realistic style',
  'Ocean waves crash on black rocks during sunset, dramatic lighting, ultra-detailed',
  'A turquoise river flowing through mountain valley, aerial shot, clear water, natural motion',
  'A cyberpunk street at night with rain reflections and neon signs, dynamic camera movement',
]

const ONE_HOUR_MS = 60 * 60 * 1000

function pruneExpiredTasks(items: VideoTaskItem[], nowMs: number): VideoTaskItem[] {
  return items.filter((item) => {
    const created = item.createdAtMs || Date.parse(item.createdAt || '')
    if (Number.isNaN(created)) return true
    return nowMs-created < ONE_HOUR_MS
  })
}

function mergeTasks(prev: VideoTaskItem[], incoming: VideoTaskItem[]): VideoTaskItem[] {
  const byKey = new Map<string, VideoTaskItem>()
  for (const item of prev) {
    const key = item.taskId || item.id
    byKey.set(key, item)
  }
  for (const item of incoming) {
    const key = item.taskId || item.id
    const old = byKey.get(key)
    byKey.set(key, old ? { ...old, ...item } : item)
  }
  return Array.from(byKey.values()).sort((a, b) => (b.createdAtMs || 0) - (a.createdAtMs || 0))
}

const videoApi = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  withCredentials: true,
})

videoApi.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

videoApi.interceptors.response.use(
  (response) => response,
  (error) => Promise.reject(error)
)

function formatModelName(modelIdOrName: string): string {
  if (!modelIdOrName) return ''
  if (modelIdOrName.toLowerCase() === 'bedi/wan2.2-t2v-a14b') return 'Wan2.2-T2V-A14B'
  const raw = modelIdOrName.includes('/') ? modelIdOrName.split('/').pop() || modelIdOrName : modelIdOrName
  const normalized = raw
    .replace(/[_]/g, '-')
    .replace(/wan2[\.\-]?2/ig, 'Wan2.2')
    .replace(/t2v/ig, 'T2V')
    .replace(/i2v/ig, 'I2V')
    .replace(/a14b/ig, 'A14B')
  const parts = normalized.split('-').filter(Boolean)
  if (parts.length >= 3) {
    return `${parts[0]}-${parts[1]}-${parts.slice(2).join('-')}`
  }
  return normalized
}

function extractVideoURL(data: any): string {
  if (!data || typeof data !== 'object') return ''
  return (
    data?.content?.video_url ||
    data?.result?.video_url ||
    data?.output?.video_url ||
    data?.video_url ||
    data?.files?.[0]?.url ||
    ''
  )
}

const VideoPlayground: React.FC = () => {
  const { t } = useTranslation()
  const { appTheme } = useTheme()

  const [models, setModels] = useState<PlaygroundModel[]>([])
  const [loadingModels, setLoadingModels] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [model, setModel] = useState<string>(T2V_MODEL_ID)
  const [prompt, setPrompt] = useState('')
  const [negativePrompt, setNegativePrompt] = useState(DEFAULT_NEGATIVE_PROMPT)
  const [ratio, setRatio] = useState('16:9')
  const [resolution, setResolution] = useState('480p')
  const [duration] = useState(5)
  const [fps] = useState(16)
  const [seed, setSeed] = useState<number | undefined>(42)
  const [history, setHistory] = useState<VideoTaskItem[]>([])
  const [selectedTask, setSelectedTask] = useState<VideoTaskItem | null>(null)
  const [videoLoadFailed, setVideoLoadFailed] = useState(false)
  const [videoErrorText, setVideoErrorText] = useState('')

  const randomSeed = () => Math.floor(Math.random() * 2147483647)
  const getModelDisplayName = (modelId: string) => {
    const hit = models.find((m) => m.id === modelId)
    return hit?.name ? formatModelName(hit.name) : formatModelName(modelId)
  }

  useEffect(() => {
    const run = async () => {
      setLoadingModels(true)
      try {
        const res = await getPlaygroundModels()
        const list = (res?.data?.data || []) as PlaygroundModel[]
        setModels(list)
        const qsModel = new URLSearchParams(window.location.search).get('model')
        if (qsModel) {
          setModel(qsModel)
        } else {
          const hasT2V = list.some((m) => m.id === T2V_MODEL_ID)
          if (hasT2V) setModel(T2V_MODEL_ID)
        }
      } catch {
        message.error(t('videoPlayground.load_models_failed'))
      } finally {
        setLoadingModels(false)
      }
    }
    run()
  }, [t])

  useEffect(() => {
    const timer = setInterval(() => {
      const now = Date.now()
      setHistory((prev) => {
        const kept = pruneExpiredTasks(prev, now)
        return kept
      })
      setSelectedTask((prev) => {
        if (!prev) return prev
        const created = prev.createdAtMs || Date.parse(prev.createdAt || '')
        if (!Number.isNaN(created) && now-created >= ONE_HOUR_MS) return null
        return prev
      })
    }, 30000)
    return () => clearInterval(timer)
  }, [])

  useEffect(() => {
    const loadTasks = async () => {
      try {
        const res = await videoApi.get('/contents/generations/tasks?page_num=1&page_size=20')
        const payload = res?.data?.data ?? res?.data ?? {}
        const rows = Array.isArray(payload)
          ? payload
          : Array.isArray(payload?.list)
            ? payload.list
            : Array.isArray(payload?.items)
              ? payload.items
              : []
        const mapped: VideoTaskItem[] = rows.map((r: any, idx: number) => {
          const createdAtMs = r?.created_at ? Number(r.created_at) * 1000 : Date.now()
          return ({
          id: String(r?.id || r?.task_id || r?.taskId || `remote-${idx}`),
          model: r?.model || T2V_MODEL_ID,
          prompt: r?.content?.text || r?.content?.[0]?.text || r?.prompt || '',
          status: r?.status || 'unknown',
          createdAt: new Date(createdAtMs).toLocaleString(),
          createdAtMs,
          taskId: r?.id || r?.task_id || r?.taskId || '',
          raw: r,
          videoUrl: extractVideoURL(r),
        })})
        if (mapped.length > 0) {
          const now = Date.now()
          setHistory((prev) => pruneExpiredTasks(mergeTasks(prev, mapped), now))
          setSelectedTask((prev) => prev || mapped[0] || null)
        }
      } catch {
        // ignore, local task list still works
      }
    }
    loadTasks()
    const timer = setInterval(loadTasks, 15000)
    return () => clearInterval(timer)
  }, [])

  const submit = async () => {
    if (!model) {
      message.warning(t('videoPlayground.select_model_first'))
      return
    }
    if (!prompt.trim()) {
      message.warning(t('videoPlayground.prompt_required'))
      return
    }

    setSubmitting(true)
    try {
      const content: any[] = [{ type: 'text', text: prompt.trim() }]
      const payload: any = {
        model,
        content,
        ratio,
        resolution,
        duration,
        fps,
        seed: seed ?? -1,
      }
      if (negativePrompt.trim()) payload.negative_prompt = negativePrompt.trim()
      const res = await videoApi.post('/contents/generations/tasks', payload)
      const data = res?.data?.data || res?.data || {}
      const taskId = data?.task_id || data?.id || data?.taskId || ''
      const status = data?.status || 'submitted'
      const createdAt = new Date().toLocaleString()

      const newTask = {
        id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        model,
        prompt: prompt.trim(),
        status,
        createdAt,
        createdAtMs: Date.now(),
        taskId,
        raw: data,
        videoUrl: extractVideoURL(data),
      }
      setHistory((prev) => [newTask, ...prev])
      setSelectedTask(newTask)
      setVideoLoadFailed(false)
      message.success(t('videoPlayground.submit_success'))
      message.info(t('videoPlayground.save_one_hour_tip'))
    } catch (e: any) {
      const errMsg = e?.response?.data?.error?.message || e?.response?.data?.message || t('videoPlayground.submit_failed')
      message.error(String(errMsg))
    } finally {
      setSubmitting(false)
    }
  }

  useEffect(() => {
    if (!selectedTask?.taskId) return
    const status = (selectedTask.status || '').toLowerCase()
    if (!['queued', 'running', 'submitted', 'pending', 'processing'].includes(status)) return

    let cancelled = false
    const timer = setInterval(async () => {
      if (cancelled) return
      try {
        const res = await videoApi.get(`/contents/generations/tasks/${encodeURIComponent(selectedTask.taskId as string)}`)
        const data = res?.data?.data || res?.data || {}
        const nextStatus = (data?.status || selectedTask.status || '').toLowerCase()
        const nextVideoUrl = extractVideoURL(data)

        setHistory((prev) =>
          prev.map((item) =>
            item.id === selectedTask.id
              ? { ...item, status: nextStatus || item.status, raw: data, videoUrl: nextVideoUrl || item.videoUrl }
              : item
          )
        )

        setSelectedTask((prev) =>
          prev && prev.id === selectedTask.id
            ? { ...prev, status: nextStatus || prev.status, raw: data, videoUrl: nextVideoUrl || prev.videoUrl }
            : prev
        )
        if (nextVideoUrl) setVideoLoadFailed(false)

        if (['succeeded', 'failed', 'cancelled', 'canceled'].includes(nextStatus)) {
          clearInterval(timer)
          if (nextStatus === 'succeeded' && nextVideoUrl) {
            message.success(t('videoPlayground.status_succeeded'))
          } else if (nextStatus === 'failed') {
            message.error(t('videoPlayground.status_failed'))
          }
        }
      } catch {
        // keep polling; transient errors are tolerated
      }
    }, 4000)

    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [selectedTask?.id, selectedTask?.taskId, selectedTask?.status, t])

  const statusText = (s?: string) => {
    const status = (s || '').toLowerCase()
    if (['queued', 'running', 'submitted', 'pending', 'processing'].includes(status)) return t('videoPlayground.status_running')
    if (status === 'succeeded') return t('videoPlayground.status_succeeded')
    if (status === 'failed') return t('videoPlayground.status_failed')
    return s || '-'
  }

  const statusColor = (s?: string) => {
    const status = (s || '').toLowerCase()
    if (['queued', 'running', 'submitted', 'pending', 'processing'].includes(status)) return 'processing'
    if (status === 'succeeded') return 'success'
    if (status === 'failed') return 'error'
    return 'default'
  }

  const getPlayableVideoURL = (rawUrl?: string) => {
    if (!selectedTask?.taskId) return rawUrl || ''
    return `/api/user/market/video/tasks/${encodeURIComponent(selectedTask.taskId)}/content`
  }

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '320px minmax(900px, 1fr)', gap: 16, alignItems: 'stretch' }}>
      <div style={{ display: 'grid', gridTemplateRows: 'auto 1fr', gap: 12, height: '100%' }}>
        <Card title={t('menu.video_generation')}>
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <div>
              <div style={{ marginBottom: 6, color: appTheme.textSecondary }}>Model</div>
              <Select
                value={model || undefined}
                onChange={setModel}
                loading={loadingModels}
                style={{ width: '100%' }}
                options={models
                  .filter((m) => (m.model_type || '').toLowerCase() === 'video')
                  .map((m) => ({ label: formatModelName(m.name || m.id), value: m.id }))}
                optionFilterProp="label"
                placeholder={t('videoPlayground.model_placeholder')}
              />
            </div>

            <div style={{ marginBottom: 2, color: appTheme.textSecondary }}>Video Size</div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
              <Select value={ratio} onChange={setRatio} options={[{ value: '16:9' }, { value: '9:16' }, { value: '1:1' }]} />
              <Select value={resolution} onChange={setResolution} options={[{ value: '480p' }, { value: '720p' }, { value: '1080p' }]} />
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr', gap: 8 }}>
              <InputNumber
                value={seed}
                onChange={(v) => setSeed(v == null ? undefined : Number(v))}
                style={{ width: '100%' }}
                addonBefore="Seed"
                addonAfter={<SyncOutlined onClick={() => setSeed(randomSeed())} style={{ cursor: 'pointer' }} />}
              />
            </div>

            <div style={{ marginBottom: 2, color: appTheme.textSecondary }}>Negative Prompt</div>
            <Input.TextArea value={negativePrompt} onChange={(e) => setNegativePrompt(e.target.value)} rows={6} style={{ maxHeight: 170, overflowY: 'auto' }} placeholder={t('videoPlayground.negative_prompt_placeholder')} />
          </Space>
        </Card>

        <Card title={t('videoPlayground.history')} style={{ height: '100%' }} styles={{ body: { maxHeight: '42vh', overflowY: 'auto' } }}>
          <List
            dataSource={history}
            locale={{ emptyText: t('common.no_data') }}
            renderItem={(item) => (
            <List.Item onClick={() => { setSelectedTask(item); setVideoLoadFailed(false); setVideoErrorText('') }} style={{ cursor: 'pointer', padding: 0, borderBottom: 'none', marginBottom: 10 }}>
                <List.Item.Meta
                  style={{ background: '#f5f7fb', border: '1px solid #e7ebf3', borderRadius: 10, padding: 12 }}
                  title={
                    <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8 }}>
                      <span>{getModelDisplayName(item.model)}</span>
                      <Tag color={statusColor(item.status)}>{statusText(item.status)}</Tag>
                    </div>
                  }
                  description={
                    <div>
                      <div style={{ marginBottom: 6 }}>{item.prompt}</div>
                      <div style={{ color: appTheme.textTertiary }}>{item.createdAt}</div>
                      {item.taskId ? <div style={{ color: appTheme.textTertiary }}>task_id: {item.taskId}</div> : null}
                      {item.videoUrl ? <div style={{ color: appTheme.textTertiary }}>video: ready</div> : null}
                    </div>
                  }
                />
              </List.Item>
            )}
          />
        </Card>
      </div>

      <Card title={selectedTask ? formatModelName(getModelDisplayName(selectedTask.model)) : t('videoPlayground.preview')} style={{ height: '100%' }}>
        <Alert type="warning" showIcon message={t('videoPlayground.billing_tip')} style={{ marginBottom: 12 }} />
        <Alert type="info" showIcon message={t('videoPlayground.save_one_hour_tip')} style={{ marginBottom: 12 }} />
        <div style={{ minHeight: 420, border: `1px dashed ${appTheme.border}`, borderRadius: 8, padding: 16, color: appTheme.textSecondary }}>
          {selectedTask && ['queued', 'running', 'submitted', 'pending', 'processing'].includes((selectedTask.status || '').toLowerCase()) ? (
            <div style={{ minHeight: 388, display: 'flex', alignItems: 'center', justifyContent: 'center', flexDirection: 'column', gap: 14 }}>
              <Spin spinning />
              <div style={{ fontSize: 30 }}>{t('videoPlayground.generating')}</div>
            </div>
          ) : selectedTask?.videoUrl ? (
            videoLoadFailed ? (
              <div style={{ height: 388, display: 'flex', alignItems: 'center', justifyContent: 'center', flexDirection: 'column', gap: 12 }}>
                <div style={{ color: appTheme.textSecondary }}>{t('videoPlayground.preview_failed_tip')}</div>
                <Button href={getPlayableVideoURL(selectedTask.videoUrl)} target="_blank" rel="noreferrer">{t('videoPlayground.open_in_new_tab')}</Button>
                {videoErrorText ? <div style={{ color: appTheme.textTertiary, fontSize: 12 }}>{videoErrorText}</div> : null}
              </div>
            ) : (
              <video
                key={`${selectedTask.videoUrl}-${selectedTask.id}`}
                src={getPlayableVideoURL(selectedTask.videoUrl)}
                controls
                playsInline
                preload="metadata"
                style={{ width: '100%', maxHeight: 420, borderRadius: 8, background: '#000' }}
                onLoadedData={() => {
                  setVideoLoadFailed(false)
                  setVideoErrorText('')
                }}
                onError={(e) => {
                  const mediaErr = (e.currentTarget as HTMLVideoElement)?.error
                  const code = mediaErr?.code
                  let reason = 'unknown'
                  if (code === 1) reason = 'aborted'
                  else if (code === 2) reason = 'network'
                  else if (code === 3) reason = 'decode'
                  else if (code === 4) reason = 'src_not_supported'
                  const raw = selectedTask.videoUrl || ''
                  const playable = getPlayableVideoURL(raw)
                  setVideoErrorText(`video_error=${reason}(${code || 0}), raw=${raw}, playable=${playable}`)
                  setVideoLoadFailed(true)
                }}
              />
            )
          ) : (
            t('videoPlayground.preview_placeholder')
          )}
        </div>
        {selectedTask?.videoUrl ? (
          <div style={{ marginTop: 8, color: appTheme.textTertiary, fontSize: 12 }}>
            {t('videoPlayground.preview_codec_tip')}
          </div>
        ) : null}
        {selectedTask?.videoUrl ? (
          <div style={{ marginTop: 12 }}>
            <Button
              type="primary"
              href={`/api/user/market/video/tasks/${encodeURIComponent(selectedTask.taskId || selectedTask.id)}/content?download=1`}
              target="_blank"
              rel="noreferrer"
            >
              {t('videoPlayground.download_video')}
            </Button>
          </div>
        ) : null}
        <div style={{ display: 'flex', gap: 8, marginTop: 12, flexWrap: 'wrap' }}>
          {QUICK_PROMPTS.map((item) => (
            <Tag
              key={item}
              style={{ cursor: 'pointer', borderRadius: 16, padding: '4px 10px' }}
              onClick={() => setPrompt(item)}
            >
              {item.length > 36 ? `${item.slice(0, 36)}...` : item}
            </Tag>
          ))}
        </div>
        <div style={{ marginTop: 12 }}>
          <Input.TextArea
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            rows={4}
            placeholder={t('videoPlayground.prompt_placeholder')}
          />
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 10 }}>
            <Button type="primary" icon={<VideoCameraOutlined />} loading={submitting} onClick={submit}>
              {t('videoPlayground.generate')}
            </Button>
          </div>
        </div>
      </Card>
    </div>
  )
}

export default VideoPlayground
