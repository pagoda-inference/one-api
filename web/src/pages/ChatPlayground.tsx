import { useState, useEffect, useRef } from 'react'
import { useTheme } from '../contexts/ThemeContext'
import {
  Input,
  Button,
  Select,
  Switch,
  Slider,
  Tag,
  message,
  InputNumber,
  Modal,
} from 'antd'
import {
  SendOutlined,
  ClearOutlined,
  SyncOutlined,
  RollbackOutlined,
  PictureOutlined,
  InboxOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { uploadFile } from '../services/api'

const { TextArea } = Input

const TRIAL_API_KEY = import.meta.env.VITE_TRIAL_API_KEY || ''
const SVG_MIME = 'image/svg+xml'

function buildSystemPrompt(basePrompt: string, enableThinking: boolean) {
  const NO_THINK_PROMPT = `
You are a helpful assistant.
You MUST NOT output any thinking process.
Do NOT output <think> or </think>.
Only output the final answer.
`.trim()

  if (enableThinking) return basePrompt || ''

  if (basePrompt && basePrompt.includes('MUST NOT output any thinking process')) {
    return basePrompt
  }

  return basePrompt
    ? `${basePrompt}\n\n${NO_THINK_PROMPT}`
    : NO_THINK_PROMPT
}

function trimLeadingInvisible(text: string) {
  return text.replace(/^[\s\r\n]+/, '')
}

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  thinking?: string
  createdAt: Date
  images?: string[]
}

interface ChatParams {
  model: string
  maxTokens: number
  temperature: number
  topP: number
  topK: number
  frequencyPenalty: number
  presencePenalty: number
  systemPrompt: string
  enableThinking: boolean
  thinkingBudget: number
}

interface ParamControlProps {
  label: string
  value: number
  min: number
  max: number
  step?: number
  precision?: number
  onChange: (value: number) => void
  suffix?: string
}

const ParamControl: React.FC<ParamControlProps> = ({
  label, value, min, max, step = 1, precision, onChange, suffix
}) => {
  const { appTheme } = useTheme()

  return (
    <div style={{ marginBottom: 18 }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 8,
        }}
      >
        <span style={{ fontSize: 14, fontWeight: 500, color: appTheme.textPrimary }}>
          {label}
        </span>
        <InputNumber
          value={value}
          min={min}
          max={max}
          step={step}
          precision={precision}
          onChange={(v) => onChange(v ?? value)}
          style={{ width: 100 }}
          size="small"
        />
      </div>
      <Slider
        value={value}
        min={min}
        max={max}
        step={step}
        onChange={(v) => onChange(v as number)}
      />
      {suffix && (
        <div style={{ fontSize: 11, color: appTheme.textTertiary, marginTop: 2 }}>
          {suffix}
        </div>
      )}
    </div>
  )
}

const trialModels = [
  { value: 'bedi/qwen3-14b', label: 'Qwen3-14B', isVL: false, isReasoning: true },
  { value: 'bedi/qwen3-32b', label: 'Qwen3-32B', isVL: false, isReasoning: true },
  { value: 'bedi/qwen3-vl-8b', label: 'Qwen3-VL-8B', isVL: true, isReasoning: false },
  { value: 'bedi/kimi-k2.6', label: 'Kimi-K2.6', isVL: true, isReasoning: true },
]

const modelCapabilities: Record<
  string,
  {
    hasTopK: boolean
    hasFrequencyPenalty: boolean
    hasEnableThinking: boolean
    hasThinkingBudget: boolean
    hasPresencePenalty: boolean
    isVL: boolean
  }
> = {
  'bedi/qwen3-14b': {
    hasTopK: true,
    hasFrequencyPenalty: true,
    hasEnableThinking: true,
    hasThinkingBudget: true,
    hasPresencePenalty: true,
    isVL: false,
  },
  'bedi/qwen3-32b': {
    hasTopK: true,
    hasFrequencyPenalty: true,
    hasEnableThinking: true,
    hasThinkingBudget: true,
    hasPresencePenalty: true,
    isVL: false,
  },
  'bedi/qwen3-vl-8b': {
    hasTopK: true,
    hasFrequencyPenalty: true,
    hasEnableThinking: false,
    hasThinkingBudget: false,
    hasPresencePenalty: false,
    isVL: true,
  },
  'bedi/kimi-k2.6': {
    hasTopK: true,
    hasFrequencyPenalty: false,
    hasEnableThinking: true,
    hasThinkingBudget: true,
    hasPresencePenalty: true,
    isVL: true,
  },
}

const ChatPlayground: React.FC = () => {
  const { t } = useTranslation()
  const { appTheme } = useTheme()

  const [messages, setMessages] = useState<Message[]>([])
  const [messages2, setMessages2] = useState<Message[]>([])
  const [inputValue, setInputValue] = useState('')
  const [loading, setLoading] = useState(false)
  const [loading2, setLoading2] = useState(false)
  const [streamedContent, setStreamedContent] = useState('')
  const [streamedContent2, setStreamedContent2] = useState('')
  const [thinkingContent, setThinkingContent] = useState('')
  const [thinkingContent2, setThinkingContent2] = useState('')
  const [isThinking, setIsThinking] = useState(false)
  const [isThinking2, setIsThinking2] = useState(false)
  const [requestThinkingEnabled, setRequestThinkingEnabled] = useState(false)
  const [requestThinkingEnabled2, setRequestThinkingEnabled2] = useState(false)
  const [thinkingExpanded, setThinkingExpanded] = useState(true)
  const [thinkingExpanded2, setThinkingExpanded2] = useState(true)
  const [chatStarted, setChatStarted] = useState(false)
  const [comparisonMode, setComparisonMode] = useState(false)

  const [imageModalOpen, setImageModalOpen] = useState(false)
  const [imageUrlInput, setImageUrlInput] = useState('')
  const [pendingImages, setPendingImages] = useState<string[]>([])

  const fileInputRef = useRef<HTMLInputElement>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const messagesEndRef2 = useRef<HTMLDivElement>(null)
  const messageListRef = useRef<HTMLDivElement>(null)
  const messageListRef2 = useRef<HTMLDivElement>(null)
  const shouldAutoScrollRef = useRef(true)
  const shouldAutoScrollRef2 = useRef(true)
  const lastScrollTopRef = useRef(0)
  const lastScrollTopRef2 = useRef(0)
  const abortControllerRef = useRef<AbortController | null>(null)
  const abortControllerRef2 = useRef<AbortController | null>(null)

  const [params, setParams] = useState<ChatParams>({
    model: 'bedi/qwen3-32b',
    maxTokens: 8192,
    temperature: 0.6,
    topP: 0.95,
    topK: 20,
    frequencyPenalty: 0,
    presencePenalty: 0.00,
    systemPrompt: '',
    enableThinking: false,
    thinkingBudget: 4096,
  })

  const [params2, setParams2] = useState<ChatParams>({
    model: 'bedi/qwen3-14b',
    maxTokens: 8192,
    temperature: 0.6,
    topP: 0.95,
    topK: 20,
    frequencyPenalty: 0,
    presencePenalty: 0.00,
    systemPrompt: '',
    enableThinking: false,
    thinkingBudget: 4096,
  })

  const capabilities = modelCapabilities[params.model] || modelCapabilities['bedi/qwen3-32b']
  const capabilities2 = modelCapabilities[params2.model] || modelCapabilities['bedi/qwen3-14b']
  const vlImageProxyBaseUrl = (import.meta as any).env?.VITE_VL_IMAGE_PROXY_BASE_URL as string | undefined
  const useVlProxyBaseUrl = String((import.meta as any).env?.VITE_VL_USE_PROXY_BASE_URL || '').toLowerCase() === 'true'
  const inlineImageAsBase64 = String((import.meta as any).env?.VITE_VL_INLINE_IMAGE_AS_BASE64 || '').toLowerCase() === 'true'

  const primaryIsVL = capabilities.isVL

  const safeAutoScroll = (
    endRef: React.RefObject<HTMLDivElement>
  ) => {
    requestAnimationFrame(() => {
      endRef.current?.scrollIntoView({ block: 'end' })
    })
  }

  const updateAutoScrollFlag = (
    e: React.UIEvent<HTMLDivElement>,
    flagRef: React.MutableRefObject<boolean>,
    lastScrollTopRef: React.MutableRefObject<number>
  ) => {
    const el = e.currentTarget
    const prevTop = lastScrollTopRef.current
    const scrolledUp = el.scrollTop < prevTop

    const topDelta = Math.abs(el.scrollTop - lastScrollTopRef.current)
    const threshold = 120
    const distanceFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    lastScrollTopRef.current = el.scrollTop

    // For height-only reflow events, only allow enabling auto-scroll when near bottom.
    // Never disable it on those events, otherwise growing thinking content can get "stuck".
    if (topDelta < 1) {
      if (distanceFromBottom < threshold) {
        flagRef.current = true
      }
      return
    }

    // User intent has highest priority: any upward scroll pauses auto-follow.
    if (scrolledUp) {
      flagRef.current = false
      return
    }

    // Keep auto-scroll on unless the user explicitly scrolled up.
    // This avoids false negatives caused by streaming content growth.
    if (distanceFromBottom < threshold) {
      flagRef.current = true
    }
  }

  useEffect(() => {
    if (shouldAutoScrollRef.current) {
      safeAutoScroll(messagesEndRef)
    }
  }, [messages, streamedContent, thinkingContent, loading, isThinking])

  useEffect(() => {
    if (shouldAutoScrollRef2.current) {
      safeAutoScroll(messagesEndRef2)
    }
  }, [messages2, streamedContent2, thinkingContent2, loading2, isThinking2])

  useEffect(() => {
    return () => {
      abortControllerRef.current?.abort()
      abortControllerRef2.current?.abort()
    }
  }, [])

  const handleRemoveComparison = () => {
    setComparisonMode(false)
    setMessages2([])
    setStreamedContent2('')
    setThinkingContent2('')
    setIsThinking2(false)
    setRequestThinkingEnabled2(false)
    setLoading2(false)
    shouldAutoScrollRef.current = true
    shouldAutoScrollRef2.current = true
  }

  const handleSyncParams = () => {
    setParams2((prev) => ({
      ...prev,
      maxTokens: params.maxTokens,
      temperature: params.temperature,
      topP: params.topP,
      topK: params.topK,
      frequencyPenalty: params.frequencyPenalty,
      presencePenalty: params.presencePenalty,
      enableThinking: params.enableThinking,
      thinkingBudget: params.thinkingBudget,
    }))
    message.success('参数已同步')
  }

  const handleSelectLocalImage = () => {
    fileInputRef.current?.click()
  }

  const isSvgImage = (value: string) => {
    const normalized = value.trim().toLowerCase()
    if (!normalized) return false
    if (normalized.startsWith('data:image/svg')) return true
    const noQuery = normalized.split('?')[0].split('#')[0]
    return noQuery.endsWith('.svg')
  }

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    if (!file.type.startsWith('image/')) {
      message.error(t('chat.select_image_file'))
      return
    }
    if (file.type === SVG_MIME) {
      message.error('暂不支持 SVG 图片，请上传 PNG/JPG/WebP')
      e.target.value = ''
      return
    }

    if (file.size > 10 * 1024 * 1024) {
      message.error(t('chat.image_size_limit'))
      return
    }

    const formData = new FormData()
    formData.append('file', file)

    try {
      const res = await uploadFile(formData)
      if (res.data.success) {
        const fileURL = res.data.data.url
        setPendingImages((prev) => [...prev, fileURL])
      } else {
        message.error(res.data.message || t('chat.upload_failed'))
      }
    } catch {
      message.error(t('chat.upload_failed'))
    }

    setImageModalOpen(false)
    setImageUrlInput('')
    e.target.value = ''
  }

  const handleAddImageUrl = () => {
    const value = imageUrlInput.trim()
    if (!value) return
    try {
      new URL(value)
      if (isSvgImage(value)) {
        message.error('暂不支持 SVG 图片，请使用 PNG/JPG/WebP 链接')
        return
      }
      setPendingImages((prev) => [...prev, value])
      setImageUrlInput('')
      setImageModalOpen(false)
    } catch {
      message.error(t('chat.invalid_image_url'))
    }
  }

  const handleRemovePendingImage = (index: number) => {
    setPendingImages((prev) => prev.filter((_, i) => i !== index))
  }

  const handleSend = async () => {
    if ((!inputValue.trim() && pendingImages.length === 0) || loading || loading2) return
    if (pendingImages.some(isSvgImage)) {
      message.error('当前视觉模型暂不支持 SVG 图片，请改用 PNG/JPG/WebP')
      return
    }

    const userMessage: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: inputValue.trim(),
      createdAt: new Date(),
      images: pendingImages.length > 0 ? [...pendingImages] : undefined,
    }

    setMessages((prev) => [...prev, userMessage])
    if (comparisonMode) {
      setMessages2((prev) => [...prev, userMessage])
    }

    setInputValue('')
    setPendingImages([])
    setChatStarted(true)
    shouldAutoScrollRef.current = true
    shouldAutoScrollRef2.current = true

    setLoading(true)
    setStreamedContent('')
    setThinkingContent('')
    const thinkingEnabledForRequest = capabilities.hasEnableThinking && params.enableThinking
    setRequestThinkingEnabled(thinkingEnabledForRequest)
    setIsThinking(thinkingEnabledForRequest)
    setThinkingExpanded(true)

    if (comparisonMode) {
      setLoading2(true)
      setStreamedContent2('')
      setThinkingContent2('')
      const thinkingEnabledForRequest2 = capabilities2.hasEnableThinking && params2.enableThinking
      setRequestThinkingEnabled2(thinkingEnabledForRequest2)
      setIsThinking2(thinkingEnabledForRequest2)
      setThinkingExpanded2(true)
    }

    const resolveImageUrl = (url: string) => {
      const trimmed = url.trim()
      if (!trimmed) return ''
      if (trimmed.startsWith('/api/images/') && vlImageProxyBaseUrl && useVlProxyBaseUrl) {
        const base = vlImageProxyBaseUrl.replace(/\/$/, '')
        const key = trimmed.replace('/api/images/', '')
        return `${base}/${key}`
      }
      try {
        return new URL(trimmed, window.location.origin).toString()
      } catch {
        return ''
      }
    }

    const blobToDataUrl = (blob: Blob) => new Promise<string>((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '')
      reader.onerror = () => reject(new Error('failed_to_read_blob'))
      reader.readAsDataURL(blob)
    })

    const toModelImageUrl = async (rawUrl: string) => {
      const absoluteUrl = resolveImageUrl(rawUrl)
      if (!absoluteUrl) return ''
      if (absoluteUrl.startsWith('data:')) return absoluteUrl
      // Default path: keep remote URL unchanged.
      // Base64 inlining is opt-in via VITE_VL_INLINE_IMAGE_AS_BASE64=true.
      if (!inlineImageAsBase64) return absoluteUrl
      try {
        const resp = await fetch(absoluteUrl)
        if (!resp.ok) return absoluteUrl
        const blob = await resp.blob()
        const dataUrl = await blobToDataUrl(blob)
        return dataUrl || absoluteUrl
      } catch {
        return absoluteUrl
      }
    }

    const buildUserContent = async (text: string, images: string[], isVL: boolean) => {
      if (!isVL || images.length === 0) return text

      const parts: Array<any> = []
      if (text) {
        parts.push({ type: 'text', text })
      }
      const modelImageUrls = await Promise.all(images.map((url) => toModelImageUrl(url)))
      modelImageUrls.forEach((modelImageUrl) => {
        if (!modelImageUrl) return
        parts.push({
          type: 'image_url',
          image_url: { url: modelImageUrl },
        })
      })
      return parts
    }

    const chatMessages: Array<{ role: string; content: any }> = []
    const finalSystemPrompt = buildSystemPrompt(params.systemPrompt, params.enableThinking)
    if (finalSystemPrompt) {
      chatMessages.push({ role: 'system', content: finalSystemPrompt })
    }
    for (const msg of messages) {
      if (msg.role === 'user' && msg.images?.length && capabilities.isVL) {
        chatMessages.push({
          role: msg.role,
          content: await buildUserContent(msg.content, msg.images, true),
        })
      } else {
        chatMessages.push({ role: msg.role, content: msg.content })
      }
    }
    chatMessages.push({
      role: 'user',
      content: await buildUserContent(userMessage.content, userMessage.images || [], capabilities.isVL),
    })

    const requestBody: Record<string, any> = {
      model: params.model,
      messages: chatMessages,
      max_tokens: params.maxTokens,
      temperature: params.temperature,
      top_p: params.topP,
      stream: true,
    }
    if (capabilities.hasTopK) requestBody.top_k = params.topK
    if (capabilities.hasFrequencyPenalty) requestBody.frequency_penalty = params.frequencyPenalty
    if (capabilities.hasPresencePenalty) requestBody.presence_penalty = params.presencePenalty
    if (capabilities.hasEnableThinking) {
      requestBody.thinking = params.enableThinking
        ? { type: 'thinking', thinking: { budget_tokens: params.thinkingBudget } }
        : { type: 'no_think' }
    }

    const chatMessages2: Array<{ role: string; content: any }> = []
    const finalSystemPrompt2 = buildSystemPrompt(params2.systemPrompt, params2.enableThinking)
    if (finalSystemPrompt2) {
      chatMessages2.push({ role: 'system', content: finalSystemPrompt2 })
    }
    for (const msg of messages2) {
      if (msg.role === 'user' && msg.images?.length && capabilities2.isVL) {
        chatMessages2.push({
          role: msg.role,
          content: await buildUserContent(msg.content, msg.images, true),
        })
      } else {
        chatMessages2.push({ role: msg.role, content: msg.content })
      }
    }
    chatMessages2.push({
      role: 'user',
      content: await buildUserContent(userMessage.content, userMessage.images || [], capabilities2.isVL),
    })

    const requestBody2: Record<string, any> = {
      model: params2.model,
      messages: chatMessages2,
      max_tokens: params2.maxTokens,
      temperature: params2.temperature,
      top_p: params2.topP,
      stream: true,
    }
    if (capabilities2.hasTopK) requestBody2.top_k = params2.topK
    if (capabilities2.hasFrequencyPenalty) requestBody2.frequency_penalty = params2.frequencyPenalty
    if (capabilities2.hasPresencePenalty) requestBody2.presence_penalty = params2.presencePenalty
    if (capabilities2.hasEnableThinking) {
      requestBody2.thinking = params2.enableThinking
        ? { type: 'thinking', thinking: { budget_tokens: params2.thinkingBudget } }
        : { type: 'no_think' }
    }

    const processStream = async (
      body: Record<string, any>,
      setLoadingFn: (v: boolean) => void,
      setStreamedContentFn: (v: string) => void,
      setThinkingContentFn: (v: string) => void,
      setIsThinkingFn: (v: boolean) => void,
      setMessagesFn: React.Dispatch<React.SetStateAction<Message[]>>,
      abortController: AbortController,
      allowThinkingDisplay: boolean,
      setRequestThinkingEnabledFn: (v: boolean) => void,
      autoScrollRef: React.MutableRefObject<boolean>,
      endRef: React.RefObject<HTMLDivElement>
    ) => {
      try {
        const response = await fetch('/v1/chat/completions', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${TRIAL_API_KEY}`,
          },
          body: JSON.stringify(body),
          signal: abortController.signal,
        })

        if (!response.ok) {
          const errText = await response.text()
          throw new Error(`HTTP ${response.status}: ${errText}`)
        }

        const reader = response.body?.getReader()
        if (!reader) throw new Error('Failed to get response reader')

        const decoder = new TextDecoder()
        let inThinking = false
        let fullContent = ''
        let thinkingText = ''

        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          const chunk = decoder.decode(value)
          const lines = chunk.split('\n')

          for (const line of lines) {
            if (!line.startsWith('data: ')) continue

            const dataStr = line.slice(6).trim()
            if (!dataStr || dataStr === '[DONE]') continue

            try {
              const data = JSON.parse(dataStr)
              const deltaObj = data.choices?.[0]?.delta || {}
              const reasoningDelta =
                deltaObj.reasoning_content ||
                deltaObj.reasoning ||
                deltaObj.thinking

              if (allowThinkingDisplay && typeof reasoningDelta === 'string' && reasoningDelta) {
                thinkingText += reasoningDelta
                setThinkingContentFn(trimLeadingInvisible(thinkingText))
                setIsThinkingFn(true)
                if (autoScrollRef.current) {
                  safeAutoScroll(endRef)
                }
              }

              const delta = deltaObj.content
              if (!delta) continue

              let remaining = delta

              while (remaining.length > 0) {
                if (!inThinking) {
                  const startIdx = remaining.indexOf('<think>')
                  if (startIdx === -1) {
                    fullContent += remaining
                    setStreamedContentFn(trimLeadingInvisible(fullContent))
                    remaining = ''
                  } else {
                    if (startIdx > 0) {
                      fullContent += remaining.slice(0, startIdx)
                      setStreamedContentFn(trimLeadingInvisible(fullContent))
                    }
                    remaining = remaining.slice(startIdx + '<think>'.length)
                    inThinking = true
                    if (allowThinkingDisplay) {
                      setIsThinkingFn(true)
                    }
                  }
                } else {
                  const endIdx = remaining.indexOf('</think>')
                  if (endIdx === -1) {
                    if (allowThinkingDisplay) {
                      thinkingText += remaining
                      setThinkingContentFn(trimLeadingInvisible(thinkingText))
                    }
                    remaining = ''
                  } else {
                    if (endIdx > 0 && allowThinkingDisplay) {
                      thinkingText += remaining.slice(0, endIdx)
                      setThinkingContentFn(trimLeadingInvisible(thinkingText))
                    }
                    remaining = remaining.slice(endIdx + '</think>'.length)
                    inThinking = false
                    if (allowThinkingDisplay) {
                      setIsThinkingFn(false)
                    }
                  }
                }
              }

              if (autoScrollRef.current) {
                safeAutoScroll(endRef)
              }
            } catch {
              // skip malformed chunk
            }
          }
        }

        const finalThinking = allowThinkingDisplay ? trimLeadingInvisible(thinkingText) : ''
        const finalContent = trimLeadingInvisible(fullContent)

        setThinkingContentFn(finalThinking)
        setStreamedContentFn(finalContent)
        setIsThinkingFn(false)

        if (finalContent) {
          const assistantMessage: Message = {
            id: (Date.now() + 1).toString(),
            role: 'assistant',
            content: finalContent,
            thinking: finalThinking,
            createdAt: new Date(),
          }
          setMessagesFn((prev) => [...prev, assistantMessage])

          setStreamedContentFn('')
          setThinkingContentFn('')
        }

        if (autoScrollRef.current) {
          safeAutoScroll(endRef)
        }
      } catch (error: any) {
        if (error.name !== 'AbortError') {
          const errMsg = error.message || t('chat.send_failed')
          message.error(errMsg)
          const errorMessage: Message = {
            id: (Date.now() + 1).toString(),
            role: 'assistant',
            content: `Error: ${errMsg}`,
            createdAt: new Date(),
          }
          setMessagesFn((prev) => [...prev, errorMessage])
        }
      } finally {
        setIsThinkingFn(false)
        setLoadingFn(false)
        setRequestThinkingEnabledFn(false)
        if (autoScrollRef.current) {
          safeAutoScroll(endRef)
        }
      }
    }

    abortControllerRef.current = new AbortController()
    processStream(
      requestBody,
      setLoading,
      setStreamedContent,
      setThinkingContent,
      setIsThinking,
      setMessages,
      abortControllerRef.current,
      capabilities.hasEnableThinking && params.enableThinking,
      setRequestThinkingEnabled,
      shouldAutoScrollRef,
      messagesEndRef
    )

    if (comparisonMode) {
      abortControllerRef2.current = new AbortController()
      processStream(
        requestBody2,
        setLoading2,
        setStreamedContent2,
        setThinkingContent2,
        setIsThinking2,
        setMessages2,
        abortControllerRef2.current,
        capabilities2.hasEnableThinking && params2.enableThinking,
        setRequestThinkingEnabled2,
        shouldAutoScrollRef2,
        messagesEndRef2
      )
    }
  }

  const handleStop = () => {
    abortControllerRef.current?.abort()
    abortControllerRef2.current?.abort()
    setIsThinking(false)
    setIsThinking2(false)
    setRequestThinkingEnabled(false)
    setRequestThinkingEnabled2(false)
    setLoading(false)
    setLoading2(false)
    setStreamedContent('')
    setStreamedContent2('')
    setThinkingContent('')
    setThinkingContent2('')
    shouldAutoScrollRef.current = true
    shouldAutoScrollRef2.current = true
  }

  const handleClear = () => {
    setMessages([])
    setMessages2([])
    setStreamedContent('')
    setStreamedContent2('')
    setThinkingContent('')
    setThinkingContent2('')
    setIsThinking(false)
    setIsThinking2(false)
    setRequestThinkingEnabled(false)
    setRequestThinkingEnabled2(false)
    setChatStarted(false)
    shouldAutoScrollRef.current = true
    shouldAutoScrollRef2.current = true
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const renderThinkingBlock = (
    content: string,
    expanded: boolean,
    setExpanded: (v: boolean) => void,
    activeThinking: boolean
  ) => (
    <div
      style={{
        maxWidth: '85%',
        flexShrink: 0,
        marginBottom: 8,
        borderRadius: 12,
        background: appTheme.bgElevated,
        border: `1px solid ${appTheme.border}`,
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          padding: '8px 12px',
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          cursor: 'pointer',
          userSelect: 'none',
        }}
        onClick={() => setExpanded(!expanded)}
      >
        <span style={{ fontSize: 14, color: appTheme.primary }}>✦</span>
        <span style={{ fontSize: 12, color: appTheme.textSecondary }}>深度思考</span>
        {activeThinking && <span className="thinking-spinner small" />}
        <span
          style={{
            fontSize: 10,
            color: appTheme.textTertiary,
            transform: expanded ? 'rotate(0deg)' : 'rotate(180deg)',
            transition: 'transform 0.2s',
            marginLeft: 'auto',
          }}
        >
          ▼
        </span>
      </div>
      {expanded && (
        <div
          style={{
            padding: '8px 12px',
            fontSize: 12,
            color: appTheme.textSecondary,
            lineHeight: 1.6,
            borderTop: `1px solid ${appTheme.border}`,
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
          }}
        >
          {trimLeadingInvisible(content) || '正在思考中...'}
        </div>
      )}
    </div>
  )

  const renderSystemPromptBlock = (
    value: string,
    onChange: (value: string) => void
  ) => (
    <div style={{ marginBottom: 18 }}>
      <div
        style={{
          fontSize: 14,
          fontWeight: 500,
          color: appTheme.textPrimary,
          marginBottom: 8,
        }}
      >
        System Prompt
      </div>
      <TextArea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        autoSize={{ minRows: 3, maxRows: 6 }}
        placeholder="可选的系统提示词"
      />
    </div>
  )

  const renderImageThumbs = (images: string[], removable = true) => {
    if (images.length === 0) return null

    return (
      <div
        style={{
          display: 'flex',
          gap: 10,
          flexWrap: 'wrap',
          marginBottom: 10,
        }}
      >
        {images.map((src, index) => (
          <div
            key={`${src}-${index}`}
            style={{
              position: 'relative',
              width: 72,
              height: 72,
              borderRadius: 10,
              border: `1px solid ${appTheme.border}`,
              overflow: 'hidden',
              background: appTheme.bgElevated,
              flexShrink: 0,
            }}
          >
            <img
              src={src}
              alt={`upload-${index}`}
              style={{
                width: '100%',
                height: '100%',
                objectFit: 'cover',
                display: 'block',
              }}
            />
            {removable && (
              <button
                type="button"
                onClick={() => handleRemovePendingImage(index)}
                style={{
                  position: 'absolute',
                  top: 4,
                  right: 4,
                  width: 22,
                  height: 22,
                  borderRadius: '50%',
                  border: 'none',
                  background: 'rgba(0,0,0,0.65)',
                  color: '#fff',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                ×
              </button>
            )}
          </div>
        ))}
      </div>
    )
  }

  const renderPrimaryParams = () => (
    <>
      <div style={{ marginBottom: 12 }}>
        <div
          style={{
            fontSize: 14,
            fontWeight: 500,
            color: appTheme.textPrimary,
            marginBottom: 8,
          }}
        >
          Model
        </div>
        <Select
          value={params.model}
          onChange={(value) => {
            if (value === 'bedi/qwen3-vl-8b') {
              setParams((p) => ({
                ...p,
                model: value,
                maxTokens: 4096,
                temperature: 0.7,
                topP: 0.7,
                topK: 50,
                enableThinking: false,
              }))
            } else if (value === 'bedi/kimi-k2.6') {
              setParams((p) => ({
                ...p,
                model: value,
                maxTokens: 8192,
                temperature: 1.0,
                topP: 0.95,
                topK: 50,
                frequencyPenalty: 0,
                presencePenalty: 0.00,
                systemPrompt: 'You are Kimi, an AI assistant.',
                enableThinking: false,
                thinkingBudget: 4096,
              }))
            } else {
              setParams((p) => ({
                ...p,
                model: value,
                maxTokens: 8192,
                temperature: 0.6,
                topP: 0.95,
                topK: 20,
                enableThinking: false,
                systemPrompt: '',
              }))
            }
          }}
          options={trialModels.map((m) => ({
            value: m.value,
            label: m.label,
          }))}
          style={{ width: '100%' }}
        />
      </div>

      {renderSystemPromptBlock(
        params.systemPrompt,
        (value) => setParams((p) => ({ ...p, systemPrompt: value }))
      )}

      <ParamControl
        label={t('chat.max_tokens')}
        value={params.maxTokens}
        min={256}
        max={32768}
        step={256}
        onChange={(v) => setParams((p) => ({ ...p, maxTokens: v }))}
        suffix="tokens"
      />

      <ParamControl
        label={t('chat.temperature')}
        value={params.temperature}
        min={0}
        max={2}
        step={0.1}
        precision={1}
        onChange={(v) => setParams((p) => ({ ...p, temperature: v }))}
      />

      <ParamControl
        label={t('chat.top_p')}
        value={params.topP}
        min={0}
        max={1}
        step={0.05}
        precision={2}
        onChange={(v) => setParams((p) => ({ ...p, topP: v }))}
      />

      {capabilities.hasTopK && (
        <ParamControl
          label={t('chat.top_k')}
          value={params.topK}
          min={1}
          max={100}
          step={1}
          onChange={(v) => setParams((p) => ({ ...p, topK: v }))}
        />
      )}

      {capabilities.hasFrequencyPenalty && (
        <ParamControl
          label={t('chat.frequency_penalty')}
          value={params.frequencyPenalty}
          min={-2}
          max={2}
          step={0.1}
          precision={1}
          onChange={(v) => setParams((p) => ({ ...p, frequencyPenalty: v }))}
        />
      )}

      {capabilities.hasPresencePenalty && (
        <ParamControl
          label={t('chat.presence_penalty')}
          value={params.presencePenalty}
          min={-2}
          max={2}
          step={0.01}
          precision={2}
          onChange={(v) => setParams((p) => ({ ...p, presencePenalty: v }))}
        />
      )}

      {capabilities.hasEnableThinking && (
        <div style={{ marginBottom: 18 }}>
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: 8,
            }}
          >
            <span style={{ fontSize: 14, fontWeight: 500 }}>{t('chat.enable_thinking')}</span>
            <Switch
              checked={params.enableThinking}
              onChange={(checked) => setParams((p) => ({ ...p, enableThinking: checked }))}
            />
          </div>
          {params.enableThinking && (
            <ParamControl
              label={t('chat.thinking_budget')}
              value={params.thinkingBudget}
              min={512}
              max={32768}
              step={256}
              onChange={(v) => setParams((p) => ({ ...p, thinkingBudget: v }))}
              suffix="tokens"
            />
          )}
        </div>
      )}

      {!comparisonMode && (
        <Button
          block
          size="middle"
          type="primary"
          onClick={() => setComparisonMode(true)}
          style={{ marginTop: 8, marginBottom: 8 }}
        >
          + {t('chat.add_comparison')}
        </Button>
      )}
    </>
  )

  const renderSecondaryParams = () => (
    <>
      <div
        style={{
          marginTop: 16,
          borderTop: `1px solid ${appTheme.border}`,
          paddingTop: 16,
        }}
      >
        <div
          style={{
            display: 'flex',
            gap: 12,
            marginBottom: 16,
          }}
        >
          <Button block icon={<RollbackOutlined />} onClick={handleRemoveComparison}>
            取消对比
          </Button>
          <Button block icon={<SyncOutlined />} onClick={handleSyncParams}>
            同步参数
          </Button>
        </div>

        <div style={{ marginBottom: 12 }}>
          <div
            style={{
              fontSize: 14,
              fontWeight: 500,
              color: appTheme.textPrimary,
              marginBottom: 8,
            }}
          >
            Model
          </div>
          <Select
            value={params2.model}
            onChange={(value) => {
              if (value === 'bedi/qwen3-vl-8b') {
                setParams2((p) => ({
                  ...p,
                  model: value,
                  maxTokens: 4096,
                  temperature: 0.7,
                  topP: 0.7,
                  topK: 50,
                  enableThinking: false,
                }))
              } else if (value === 'bedi/kimi-k2.6') {
                setParams2((p) => ({
                  ...p,
                  model: value,
                  maxTokens: 8192,
                  temperature: 1.0,
                  topP: 0.95,
                  topK: 50,
                  frequencyPenalty: 0,
                  presencePenalty: 0.00,
                  systemPrompt: 'You are Kimi, an AI assistant.',
                  enableThinking: false,
                  thinkingBudget: 4096,
                }))
              } else {
                setParams2((p) => ({
                  ...p,
                  model: value,
                  maxTokens: 8192,
                  temperature: 0.6,
                  topP: 0.95,
                  topK: 20,
                  enableThinking: false,
                }))
              }
            }}
            options={trialModels.map((m) => ({
              value: m.value,
              label: m.label,
            }))}
            style={{ width: '100%' }}
          />
        </div>

        {renderSystemPromptBlock(
          params2.systemPrompt,
          (value) => setParams2((p) => ({ ...p, systemPrompt: value }))
        )}

        <ParamControl
          label={t('chat.max_tokens')}
          value={params2.maxTokens}
          min={256}
          max={32768}
          step={256}
          onChange={(v) => setParams2((p) => ({ ...p, maxTokens: v }))}
          suffix="tokens"
        />

        <ParamControl
          label={t('chat.temperature')}
          value={params2.temperature}
          min={0}
          max={2}
          step={0.1}
          precision={1}
          onChange={(v) => setParams2((p) => ({ ...p, temperature: v }))}
        />

        <ParamControl
          label={t('chat.top_p')}
          value={params2.topP}
          min={0}
          max={1}
          step={0.05}
          precision={2}
          onChange={(v) => setParams2((p) => ({ ...p, topP: v }))}
        />

        {capabilities2.hasTopK && (
          <ParamControl
            label={t('chat.top_k')}
            value={params2.topK}
            min={1}
            max={100}
            step={1}
            onChange={(v) => setParams2((p) => ({ ...p, topK: v }))}
          />
        )}

        {capabilities2.hasFrequencyPenalty && (
          <ParamControl
            label={t('chat.frequency_penalty')}
            value={params2.frequencyPenalty}
            min={-2}
            max={2}
            step={0.1}
            precision={1}
            onChange={(v) => setParams2((p) => ({ ...p, frequencyPenalty: v }))}
          />
        )}

        {capabilities2.hasPresencePenalty && (
          <ParamControl
            label={t('chat.presence_penalty')}
            value={params2.presencePenalty}
            min={-2}
            max={2}
            step={0.01}
            precision={2}
            onChange={(v) => setParams2((p) => ({ ...p, presencePenalty: v }))}
          />
        )}

        {capabilities2.hasEnableThinking && (
          <div style={{ marginBottom: 18 }}>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginBottom: 8,
              }}
            >
              <span style={{ fontSize: 14, fontWeight: 500 }}>{t('chat.enable_thinking')}</span>
              <Switch
                checked={params2.enableThinking}
                onChange={(checked) => setParams2((p) => ({ ...p, enableThinking: checked }))}
              />
            </div>
            {params2.enableThinking && (
              <ParamControl
                label={t('chat.thinking_budget')}
                value={params2.thinkingBudget}
                min={512}
                max={32768}
                step={256}
                onChange={(v) => setParams2((p) => ({ ...p, thinkingBudget: v }))}
                suffix="tokens"
              />
            )}
          </div>
        )}
      </div>
    </>
  )

  return (
    <div style={{ display: 'flex', height: 'calc(100vh - 64px)', overflow: 'hidden' }}>
      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        style={{ display: 'none' }}
        onChange={handleFileChange}
      />

      <Modal
        open={imageModalOpen}
        onCancel={() => {
          setImageModalOpen(false)
          setImageUrlInput('')
        }}
        footer={null}
        width={520}
        title={t('chat.upload_image_file')}
      >
        <div style={{ paddingTop: 8 }}>
          <Button
            block
            size="large"
            icon={<InboxOutlined />}
            onClick={handleSelectLocalImage}
            style={{ marginBottom: 16 }}
          >
            {t('chat.upload_image_button')}
          </Button>

          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 12,
              marginBottom: 16,
              color: appTheme.textTertiary,
            }}
          >
            <div style={{ flex: 1, height: 1, background: appTheme.border }} />
            <span>or</span>
            <div style={{ flex: 1, height: 1, background: appTheme.border }} />
          </div>

          <div style={{ display: 'flex', gap: 10 }}>
            <Input
              value={imageUrlInput}
              onChange={(e) => setImageUrlInput(e.target.value)}
              placeholder={t('chat.input_image_url')}
            />
            <Button onClick={handleAddImageUrl}>{t('confirm')}</Button>
          </div>
        </div>
      </Modal>

      <div
        style={{
          width: 340,
          minWidth: 320,
          borderRight: `1px solid ${appTheme.borderLight}`,
          padding: 24,
          background: appTheme.bgElevated,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        }}
      >
        <div style={{ flex: 1, overflowY: 'auto', paddingRight: 4, maxHeight: 'calc(100vh - 180px)' }}>
          {renderPrimaryParams()}
          {comparisonMode && renderSecondaryParams()}
        </div>
      </div>

      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, minHeight: 0 }}>
        <div
          style={{
            padding: '16px 24px',
            borderBottom: `1px solid ${appTheme.borderLight}`,
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        />

        <div style={{ flex: 1, overflow: 'hidden', display: 'flex', minHeight: 0, background: appTheme.bgPage }}>
          <div
            ref={messageListRef}
            onScroll={(e) => updateAutoScrollFlag(
              e,
              shouldAutoScrollRef,
              lastScrollTopRef
            )}
            style={{
              flex: 1,
              minHeight: 0,
              overflowY: 'auto',
              padding: '24px 32px 40px',
              display: 'flex',
              flexDirection: 'column',
              position: 'relative',
              overscrollBehavior: 'contain',
              scrollBehavior: 'auto',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 16, flexShrink: 0 }}>
              <span style={{ fontSize: 14, fontWeight: 500, color: appTheme.textPrimary }}>
                {trialModels.find((m) => m.value === params.model)?.label || params.model}
              </span>
              <Tag color="green">Trial</Tag>
            </div>

            {!chatStarted ? (
              <div
                style={{
                  flex: 1,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: appTheme.textSecondary,
                }}
              >
                <p style={{ fontSize: 16 }}>{t('chat.start_conversation')}</p>
                <p style={{ fontSize: 12, marginTop: 8 }}>{t('chat.trial_models_tip')}</p>
              </div>
            ) : (
              <>
                {messages.map((msg) => (
                  <div
                    key={msg.id}
                    style={{
                      marginBottom: 16,
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: msg.role === 'user' ? 'flex-end' : 'flex-start',
                    }}
                  >
                    {msg.thinking &&
                      renderThinkingBlock(msg.thinking, thinkingExpanded, setThinkingExpanded, false)}

                    {msg.role === 'user' && msg.images && msg.images.length > 0 && (
                      <div style={{ maxWidth: '85%', marginBottom: 8 }}>
                        {renderImageThumbs(msg.images, false)}
                      </div>
                    )}

                    <div
                      style={{
                        maxWidth: '85%',
                        padding: '12px 16px',
                        borderRadius: 16,
                        background: msg.role === 'user' ? appTheme.bgBubbleUser : appTheme.bgBubbleAssistant,
                        color: msg.role === 'user' ? appTheme.textOnPrimary : appTheme.textPrimary,
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word',
                        fontSize: 14,
                        lineHeight: 1.6,
                      }}
                    >
                      {trimLeadingInvisible(msg.content)}
                    </div>
                  </div>
                ))}

                {loading && !streamedContent && (
                  <div style={{ marginBottom: 16, display: 'flex', flexDirection: 'column', alignItems: 'flex-start' }}>
                    <div
                      style={{
                        maxWidth: '85%',
                        padding: '12px 16px',
                        borderRadius: 16,
                        background: appTheme.bgBubbleAssistant,
                        color: appTheme.textSecondary,
                        display: 'flex',
                        alignItems: 'center',
                        gap: 8,
                      }}
                    >
                      <span className="thinking-spinner" />
                      <span>内容生成中...</span>
                    </div>
                  </div>
                )}

                {requestThinkingEnabled && (loading || isThinking || thinkingContent) &&
                  renderThinkingBlock(
                    thinkingContent,
                    thinkingExpanded,
                    setThinkingExpanded,
                    isThinking || loading
                  )}

                {loading && streamedContent && (
                  <div style={{ marginBottom: 16, display: 'flex', flexDirection: 'column', alignItems: 'flex-start' }}>
                    <div
                      style={{
                        maxWidth: '85%',
                        padding: '12px 16px',
                        borderRadius: 16,
                        background: appTheme.bgBubbleAssistant,
                        color: appTheme.textPrimary,
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word',
                        fontSize: 14,
                        lineHeight: 1.6,
                      }}
                    >
                      {trimLeadingInvisible(streamedContent)}
                      <span style={{ animation: 'blink 1s infinite' }}>|</span>
                    </div>
                  </div>
                )}
                <div ref={messagesEndRef} />
              </>
            )}
          </div>

          {comparisonMode && (
            <div
              style={{
                position: 'relative',
                width: 1,
                background: appTheme.border,
                flexShrink: 0,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <div
                style={{
                  background: appTheme.bgElevated,
                  border: `1px solid ${appTheme.border}`,
                  borderRadius: 12,
                  padding: '4px 10px',
                  fontSize: 12,
                  color: appTheme.textSecondary,
                  fontWeight: 500,
                  position: 'absolute',
                  whiteSpace: 'nowrap',
                }}
              >
                vs
              </div>
            </div>
          )}

          {comparisonMode && (
            <div
              ref={messageListRef2}
              onScroll={(e) => updateAutoScrollFlag(
                e,
                shouldAutoScrollRef2,
                lastScrollTopRef2
              )}
              style={{
                flex: 1,
                minHeight: 0,
                overflowY: 'auto',
                padding: '24px 32px 40px',
                display: 'flex',
                flexDirection: 'column',
                overscrollBehavior: 'contain',
                scrollBehavior: 'auto',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 16, flexShrink: 0 }}>
                <span style={{ fontSize: 14, fontWeight: 500, color: appTheme.textPrimary }}>
                  {trialModels.find((m) => m.value === params2.model)?.label || params2.model}
                </span>
                <Tag color="green">Trial</Tag>
              </div>

              {!chatStarted ? (
                <div
                  style={{
                    flex: 1,
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: appTheme.textSecondary,
                  }}
                >
                  <p style={{ fontSize: 16 }}>{t('chat.start_conversation')}</p>
                </div>
              ) : (
                <>
                  {messages2.map((msg) => (
                    <div
                      key={msg.id}
                      style={{
                        marginBottom: 16,
                        display: 'flex',
                        flexDirection: 'column',
                        alignItems: msg.role === 'user' ? 'flex-end' : 'flex-start',
                      }}
                    >
                      {msg.thinking &&
                        renderThinkingBlock(msg.thinking, thinkingExpanded2, setThinkingExpanded2, false)}

                      {msg.role === 'user' && msg.images && msg.images.length > 0 && (
                        <div style={{ maxWidth: '85%', marginBottom: 8 }}>
                          {renderImageThumbs(msg.images, false)}
                        </div>
                      )}

                      <div
                        style={{
                          maxWidth: '85%',
                          padding: '12px 16px',
                          borderRadius: 16,
                          background: msg.role === 'user' ? appTheme.bgBubbleUser : appTheme.bgBubbleAssistant,
                          color: msg.role === 'user' ? appTheme.textOnPrimary : appTheme.textPrimary,
                          whiteSpace: 'pre-wrap',
                          wordBreak: 'break-word',
                          fontSize: 14,
                          lineHeight: 1.6,
                        }}
                      >
                        {trimLeadingInvisible(msg.content)}
                      </div>
                    </div>
                  ))}

                  {loading2 && !streamedContent2 && (
                    <div style={{ marginBottom: 16, display: 'flex', flexDirection: 'column', alignItems: 'flex-start' }}>
                      <div
                        style={{
                          maxWidth: '85%',
                          padding: '12px 16px',
                          borderRadius: 16,
                          background: appTheme.bgBubbleAssistant,
                          color: appTheme.textSecondary,
                          display: 'flex',
                          alignItems: 'center',
                          gap: 8,
                        }}
                      >
                        <span className="thinking-spinner" />
                        <span>内容生成中...</span>
                      </div>
                    </div>
                  )}

                  {requestThinkingEnabled2 && (loading2 || isThinking2 || thinkingContent2) &&
                    renderThinkingBlock(
                      thinkingContent2,
                      thinkingExpanded2,
                      setThinkingExpanded2,
                      isThinking2 || loading2
                    )}

                  {loading2 && streamedContent2 && (
                    <div style={{ marginBottom: 16, display: 'flex', flexDirection: 'column', alignItems: 'flex-start' }}>
                      <div
                        style={{
                          maxWidth: '85%',
                          padding: '12px 16px',
                          borderRadius: 16,
                          background: appTheme.bgBubbleAssistant,
                          color: appTheme.textPrimary,
                          whiteSpace: 'pre-wrap',
                          wordBreak: 'break-word',
                          fontSize: 14,
                          lineHeight: 1.6,
                        }}
                      >
                        {trimLeadingInvisible(streamedContent2)}
                        <span style={{ animation: 'blink 1s infinite' }}>|</span>
                      </div>
                    </div>
                  )}
                  <div ref={messagesEndRef2} />
                </>
              )}
            </div>
          )}
        </div>

        <div style={{ padding: '8px 32px 12px', background: appTheme.bgPage }}>
          <div
            style={{
              position: 'relative',
              border: `1px solid ${appTheme.border}`,
              borderRadius: 16,
              padding: '10px 12px',
              background: appTheme.bgInput,
              boxShadow: appTheme.shadow,
            }}
          >
            <div
              style={{
                position: 'absolute',
                top: 8,
                right: 12,
                zIndex: 2,
              }}
            >
              <Button
                size="small"
                icon={<ClearOutlined />}
                onClick={handleClear}
                disabled={loading || loading2}
              >
                {t('chat.clear')}
              </Button>
            </div>

            {renderImageThumbs(pendingImages, true)}

            <TextArea
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyPress}
              placeholder={
                primaryIsVL
                  ? t('chat.vl_input_placeholder')
                  : comparisonMode
                    ? t('chat.input_placeholder_compare')
                    : t('chat.input_placeholder')
              }
              autoSize={{ minRows: 2, maxRows: 4 }}
              style={{ border: 'none', boxShadow: 'none', resize: 'none', padding: 0, paddingTop: 32 }}
            />
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 8 }}>
              <div style={{ fontSize: 11, color: appTheme.textTertiary }}>
                {t('chat.disclaimer')}
              </div>
              <div style={{ display: 'flex', gap: 8 }}>
                {primaryIsVL && !comparisonMode && (
                  <Button
                    icon={<PictureOutlined />}
                    onClick={() => setImageModalOpen(true)}
                    size="small"
                  />
                )}

                {loading || loading2 ? (
                  <Button type="primary" danger onClick={handleStop} size="small">
                    {t('chat.stop')}
                  </Button>
                ) : (
                  <Button
                    type="primary"
                    icon={<SendOutlined />}
                    onClick={handleSend}
                    disabled={!inputValue.trim() && pendingImages.length === 0}
                    size="small"
                  >
                    {t('chat.send')}
                  </Button>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>

      <style>{`
        @keyframes blink {
          0%, 100% { opacity: 1; }
          50% { opacity: 0; }
        }
        .thinking-spinner {
          display: inline-block;
          width: 14px;
          height: 14px;
          border: 2px solid currentColor;
          border-top-color: transparent;
          border-radius: 50%;
          animation: spin 0.8s linear infinite;
        }
        .thinking-spinner.small {
          width: 10px;
          height: 10px;
          border-width: 1.5px;
        }
        @keyframes spin {
          to { transform: rotate(360deg); }
        }
      `}</style>
    </div>
  )
}

export default ChatPlayground
