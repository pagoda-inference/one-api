import { useState, useEffect, useRef } from 'react'
import { Input, Button, Select, Switch, Slider, Space, Tag, message, InputNumber } from 'antd'
import { SendOutlined, ClearOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'

const { TextArea } = Input

const TRIAL_API_KEY = 'sk-TRIAL_API_KEY_PLACEHOLDER'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  createdAt: Date
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
}) => (
  <div style={{ marginBottom: 18 }}>
    <div style={{
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      marginBottom: 8,
    }}>
      <span style={{ fontSize: 14, fontWeight: 500 }}>{label}</span>
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
    {suffix && <div style={{ fontSize: 11, color: '#8c8c8c', marginTop: 2 }}>{suffix}</div>}
  </div>
)

// Trial models available for experience center
const trialModels = [
  { value: 'bedi/qwen3-14b', label: 'Qwen3-14B', isVL: false, isReasoning: false },
  { value: 'bedi/qwen3-32b', label: 'Qwen3-32B', isVL: false, isReasoning: true },
  { value: 'bedi/qwen3-vl-8b', label: 'Qwen3-VL-8B', isVL: true, isReasoning: false },
]

// Model parameter capabilities
const modelCapabilities: Record<string, {
  hasTopK: boolean
  hasFrequencyPenalty: boolean
  hasEnableThinking: boolean
  hasThinkingBudget: boolean
  hasPresencePenalty: boolean
}> = {
  'bedi/qwen3-14b': {
    hasTopK: true,
    hasFrequencyPenalty: true,
    hasEnableThinking: false,
    hasThinkingBudget: false,
    hasPresencePenalty: true,
  },
  'bedi/qwen3-32b': {
    hasTopK: true,
    hasFrequencyPenalty: true,
    hasEnableThinking: true,
    hasThinkingBudget: true,
    hasPresencePenalty: true,
  },
  'bedi/qwen3-vl-8b': {
    hasTopK: true,
    hasFrequencyPenalty: false,
    hasEnableThinking: false,
    hasThinkingBudget: false,
    hasPresencePenalty: false,
  },
}

const ChatPlayground: React.FC = () => {
  const { t } = useTranslation()
  const [messages, setMessages] = useState<Message[]>([])
  const [messages2, setMessages2] = useState<Message[]>([])
  const [inputValue, setInputValue] = useState('')
  const [loading, setLoading] = useState(false)
  const [loading2, setLoading2] = useState(false)
  const [streamedContent, setStreamedContent] = useState('')
  const [streamedContent2, setStreamedContent2] = useState('')
  const [chatStarted, setChatStarted] = useState(false)
  const [comparisonMode, setComparisonMode] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const messagesEndRef2 = useRef<HTMLDivElement>(null)
  const abortControllerRef = useRef<AbortController | null>(null)
  const abortControllerRef2 = useRef<AbortController | null>(null)

  const [params, setParams] = useState<ChatParams>({
    model: 'bedi/qwen3-32b',
    maxTokens: 8192,
    temperature: 0.7,
    topP: 0.9,
    topK: 20,
    frequencyPenalty: 0,
    presencePenalty: 0,
    systemPrompt: '',
    enableThinking: false,
    thinkingBudget: 8192,
  })

  const [params2, setParams2] = useState<ChatParams>({
    model: 'bedi/qwen3-14b',
    maxTokens: 8192,
    temperature: 0.7,
    topP: 0.9,
    topK: 20,
    frequencyPenalty: 0,
    presencePenalty: 0,
    systemPrompt: '',
    enableThinking: false,
    thinkingBudget: 8192,
  })

  const selectedModel = trialModels.find(m => m.value === params.model)
  const capabilities = modelCapabilities[params.model] || modelCapabilities['bedi/qwen3-32b']
  const capabilities2 = modelCapabilities[params2.model] || modelCapabilities['bedi/qwen3-14b']

  useEffect(() => {
    scrollToBottom()
  }, [messages, streamedContent, messages2, streamedContent2])

  useEffect(() => {
    return () => {
      if (abortControllerRef.current) abortControllerRef.current.abort()
      if (abortControllerRef2.current) abortControllerRef2.current.abort()
    }
  }, [])

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  const scrollToBottom2 = () => {
    messagesEndRef2.current?.scrollIntoView({ behavior: 'smooth' })
  }

  const handleSend = async () => {
    if (!inputValue.trim() || loading) return

    const userMessage: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: inputValue.trim(),
      createdAt: new Date(),
    }

    setMessages(prev => [...prev, userMessage])
    if (comparisonMode) {
      setMessages2(prev => [...prev, userMessage])
    }
    setInputValue('')
    setChatStarted(true)
    setLoading(true)
    setStreamedContent('')
    if (comparisonMode) {
      setLoading2(true)
      setStreamedContent2('')
    }

    // Build messages array for model 1
    const chatMessages: Array<{ role: string; content: string }> = []
    if (params.systemPrompt.trim()) {
      chatMessages.push({ role: 'system', content: params.systemPrompt })
    }
    messages.forEach(msg => {
      chatMessages.push({ role: msg.role, content: msg.content })
    })
    chatMessages.push({ role: 'user', content: userMessage.content })

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
    if (capabilities.hasEnableThinking && params.enableThinking) {
      requestBody.thinking = { type: 'thinking', thinking: { budget_tokens: params.thinkingBudget } }
    }

    // Build messages array for model 2
    const chatMessages2: Array<{ role: string; content: string }> = []
    if (params2.systemPrompt.trim()) {
      chatMessages2.push({ role: 'system', content: params2.systemPrompt })
    }
    messages.forEach(msg => {
      chatMessages2.push({ role: msg.role, content: msg.content })
    })
    chatMessages2.push({ role: 'user', content: userMessage.content })

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
    if (capabilities2.hasEnableThinking && params2.enableThinking) {
      requestBody2.thinking = { type: 'thinking', thinking: { budget_tokens: params2.thinkingBudget } }
    }

    const processStream = async (
      body: Record<string, any>,
      setLoadingFn: (v: boolean) => void,
      setStreamedContentFn: (v: string) => void,
      setMessagesFn: React.Dispatch<React.SetStateAction<Message[]>>,
      abortController: AbortController,
      scrollFn?: () => void
    ) => {
      try {
        const response = await fetch('/v1/chat/completions', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${TRIAL_API_KEY}`,
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
        let fullContent = ''

        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          const chunk = decoder.decode(value)
          const lines = chunk.split('\n')

          for (const line of lines) {
            if (line.startsWith('data: ')) {
              const dataStr = line.slice(6).trim()
              if (dataStr === '[DONE]') continue

              try {
                const data = JSON.parse(dataStr)
                const delta = data.choices?.[0]?.delta?.content
                if (delta) {
                  fullContent += delta
                  setStreamedContentFn(fullContent)
                  if (scrollFn) setTimeout(scrollFn, 0)
                }
              } catch (e) { /* skip */ }
            }
          }
        }

        if (fullContent) {
          const assistantMessage: Message = {
            id: (Date.now() + 1).toString(),
            role: 'assistant',
            content: fullContent,
            createdAt: new Date(),
          }
          setMessagesFn(prev => [...prev, assistantMessage])
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
          setMessagesFn(prev => [...prev, errorMessage])
        }
      } finally {
        setLoadingFn(false)
      }
    }

    abortControllerRef.current = new AbortController()
    processStream(
      requestBody,
      setLoading,
      setStreamedContent,
      setMessages,
      abortControllerRef.current,
      scrollToBottom
    )

    if (comparisonMode) {
      abortControllerRef2.current = new AbortController()
      processStream(
        requestBody2,
        setLoading2,
        setStreamedContent2,
        setMessages2,
        abortControllerRef2.current,
        scrollToBottom2
      )
    }
  }

  const handleStop = () => {
    if (abortControllerRef.current) abortControllerRef.current.abort()
    if (abortControllerRef2.current) abortControllerRef2.current.abort()
  }

  const handleClear = () => {
    setMessages([])
    setMessages2([])
    setStreamedContent('')
    setStreamedContent2('')
    setChatStarted(false)
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div style={{ display: 'flex', height: 'calc(100vh - 64px)', overflow: 'hidden' }}>
      {/* Left: Config panel - 340px width, playground style */}
      <div style={{
        width: 340,
        minWidth: 320,
        borderRight: '1px solid #f0f0f0',
        padding: 24,
        background: '#fcfcfd',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}>
        {/* Model selector */}
        <div style={{ marginBottom: 12 }}>
          <Select
            value={params.model}
            onChange={(value) => setParams(p => ({ ...p, model: value }))}
            options={trialModels.map(m => ({
              value: m.value,
              label: m.label,
            }))}
            style={{ width: '100%' }}
          />
        </div>

        {/* Add comparison button - in fixed area not pushed to bottom */}
        <Button
          block
          size="middle"
          type={comparisonMode ? 'default' : 'primary'}
          onClick={() => setComparisonMode(!comparisonMode)}
          style={{ marginBottom: 16 }}
        >
          {comparisonMode ? t('chat.remove_comparison') : `+ ${t('chat.add_comparison')}`}
        </Button>

        {/* Scrollable params area */}
        <div style={{ flex: 1, overflowY: 'auto', paddingRight: 4, maxHeight: 'calc(100vh - 260px)' }}>
          <ParamControl
            label={t('chat.max_tokens')}
            value={params.maxTokens}
            min={256}
            max={32768}
            step={256}
            onChange={(v) => setParams(p => ({ ...p, maxTokens: v }))}
            suffix="tokens"
          />

          <ParamControl
            label={t('chat.temperature')}
            value={params.temperature}
            min={0}
            max={2}
            step={0.1}
            precision={1}
            onChange={(v) => setParams(p => ({ ...p, temperature: v }))}
          />

          <ParamControl
            label={t('chat.top_p')}
            value={params.topP}
            min={0}
            max={1}
            step={0.05}
            precision={2}
            onChange={(v) => setParams(p => ({ ...p, topP: v }))}
          />

          {capabilities.hasTopK && (
            <ParamControl
              label={t('chat.top_k')}
              value={params.topK}
              min={1}
              max={100}
              step={1}
              onChange={(v) => setParams(p => ({ ...p, topK: v }))}
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
              onChange={(v) => setParams(p => ({ ...p, frequencyPenalty: v }))}
            />
          )}

          {capabilities.hasPresencePenalty && (
            <ParamControl
              label={t('chat.presence_penalty')}
              value={params.presencePenalty}
              min={-2}
              max={2}
              step={0.1}
              precision={1}
              onChange={(v) => setParams(p => ({ ...p, presencePenalty: v }))}
            />
          )}

          {capabilities.hasEnableThinking && (
            <div style={{ marginBottom: 18 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                <span style={{ fontSize: 14, fontWeight: 500 }}>{t('chat.enable_thinking')}</span>
                <Switch
                  checked={params.enableThinking}
                  onChange={(checked) => setParams(p => ({ ...p, enableThinking: checked }))}
                />
              </div>
              {params.enableThinking && (
                <ParamControl
                  label={t('chat.thinking_budget')}
                  value={params.thinkingBudget}
                  min={512}
                  max={32768}
                  step={256}
                  onChange={(v) => setParams(p => ({ ...p, thinkingBudget: v }))}
                  suffix="tokens"
                />
              )}
            </div>
          )}
        </div>

        {/* Second model config panel (comparison mode) */}
        {comparisonMode && (
          <div style={{ marginTop: 16, borderTop: '1px solid #e8e8e8', paddingTop: 16 }}>
            <div style={{ fontSize: 13, color: '#8c8c8c', marginBottom: 12 }}>{t('chat.model_config')} 2</div>
            <Select
              value={params2.model}
              onChange={(value) => setParams2(p => ({ ...p, model: value }))}
              options={trialModels.map(m => ({
                value: m.value,
                label: m.label,
              }))}
              style={{ width: '100%', marginBottom: 12 }}
            />
            <ParamControl
              label={t('chat.temperature')}
              value={params2.temperature}
              min={0}
              max={2}
              step={0.1}
              precision={1}
              onChange={(v) => setParams2(p => ({ ...p, temperature: v }))}
            />
            <ParamControl
              label={t('chat.max_tokens')}
              value={params2.maxTokens}
              min={256}
              max={32768}
              step={256}
              onChange={(v) => setParams2(p => ({ ...p, maxTokens: v }))}
              suffix="tokens"
            />
            {capabilities2.hasTopK && (
              <ParamControl
                label={t('chat.top_k')}
                value={params2.topK}
                min={1}
                max={100}
                step={1}
                onChange={(v) => setParams2(p => ({ ...p, topK: v }))}
              />
            )}
          </div>
        )}
      </div>

      {/* Right: Chat area */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, minHeight: 0 }}>
        {/* Chat header */}
        <div style={{
          padding: '16px 24px',
          borderBottom: '1px solid #f0f0f0',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}>
          <div>
            <span style={{ fontWeight: 500 }}>{selectedModel?.label || t('chat.text_chat')}</span>
            <Tag color="green" style={{ marginLeft: 8 }}>{t('chat.trial')}</Tag>
            {comparisonMode && <Tag color="blue" style={{ marginLeft: 8 }}>vs</Tag>}
          </div>
          <Space>
            <Button size="small" icon={<ClearOutlined />} onClick={handleClear} disabled={loading || loading2}>
              {t('chat.clear')}
            </Button>
          </Space>
        </div>

        {/* Messages area - split view for comparison */}
        <div style={{ flex: 1, overflow: 'hidden', display: 'flex', minHeight: 0 }}>
          {/* Left model chat */}
          <div style={{
            flex: 1,
            minHeight: 0,
            overflowY: 'auto',
            padding: '24px 32px',
            display: 'flex',
            flexDirection: 'column',
            borderRight: comparisonMode ? '1px solid #f0f0f0' : 'none',
          }}>
            <div style={{ fontSize: 14, color: '#8c8c8c', marginBottom: 16, fontWeight: 500 }}>
              {trialModels.find(m => m.value === params.model)?.label || params.model}
            </div>
            {!chatStarted ? (
              <div style={{
                flex: 1,
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                color: '#8c8c8c',
              }}>
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
                    <div style={{
                      maxWidth: '85%',
                      padding: '12px 16px',
                      borderRadius: 16,
                      background: msg.role === 'user' ? '#1890ff' : '#f5f5f5',
                      color: msg.role === 'user' ? '#fff' : '#333',
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-word',
                      fontSize: 14,
                      lineHeight: 1.6,
                    }}>
                      {msg.content}
                    </div>
                  </div>
                ))}
                {loading && streamedContent && (
                  <div style={{ marginBottom: 16, display: 'flex', flexDirection: 'column', alignItems: 'flex-start' }}>
                    <div style={{
                      maxWidth: '85%',
                      padding: '12px 16px',
                      borderRadius: 16,
                      background: '#f5f5f5',
                      color: '#333',
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-word',
                      fontSize: 14,
                      lineHeight: 1.6,
                    }}>
                      {streamedContent}<span style={{ animation: 'blink 1s infinite' }}>|</span>
                    </div>
                  </div>
                )}
                <div ref={messagesEndRef} />
              </>
            )}
          </div>

          {/* Right model chat (comparison mode) */}
          {comparisonMode && (
            <div style={{
              flex: 1,
              minHeight: 0,
              overflowY: 'auto',
              padding: '24px 32px',
              display: 'flex',
              flexDirection: 'column',
            }}>
              <div style={{ fontSize: 14, color: '#8c8c8c', marginBottom: 16, fontWeight: 500 }}>
                {trialModels.find(m => m.value === params2.model)?.label || params2.model}
              </div>
              {!chatStarted ? (
                <div style={{
                  flex: 1,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: '#8c8c8c',
                }}>
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
                      <div style={{
                        maxWidth: '85%',
                        padding: '12px 16px',
                        borderRadius: 16,
                        background: msg.role === 'user' ? '#722ed1' : '#f5f5f5',
                        color: msg.role === 'user' ? '#fff' : '#333',
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word',
                        fontSize: 14,
                        lineHeight: 1.6,
                      }}>
                        {msg.content}
                      </div>
                    </div>
                  ))}
                  {loading2 && streamedContent2 && (
                    <div style={{ marginBottom: 16, display: 'flex', flexDirection: 'column', alignItems: 'flex-start' }}>
                      <div style={{
                        maxWidth: '85%',
                        padding: '12px 16px',
                        borderRadius: 16,
                        background: '#f5f5f5',
                        color: '#333',
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word',
                        fontSize: 14,
                        lineHeight: 1.6,
                      }}>
                        {streamedContent2}<span style={{ animation: 'blink 1s infinite' }}>|</span>
                      </div>
                    </div>
                  )}
                  <div ref={messagesEndRef2} />
                </>
              )}
            </div>
          )}
        </div>

        {/* Input area - compact card style */}
        <div style={{ padding: '8px 24px 12px', background: '#f8fafc' }}>
          <div style={{
            border: '1px solid #e5e7eb',
            borderRadius: 16,
            padding: '10px 12px',
            background: '#fff',
            boxShadow: '0 1px 2px rgba(0,0,0,0.03)',
          }}>
            <TextArea
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyPress}
              placeholder={comparisonMode ? t('chat.input_placeholder_compare') : t('chat.input_placeholder')}
              autoSize={{ minRows: 2, maxRows: 4 }}
              style={{ border: 'none', boxShadow: 'none', resize: 'none', padding: 0 }}
              disabled={loading || loading2}
            />
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 8 }}>
              <div style={{ fontSize: 11, color: '#8c8c8c' }}>
                {t('chat.disclaimer')}
              </div>
              <div style={{ display: 'flex', gap: 8 }}>
                {(loading || loading2) ? (
                  <Button type="primary" danger onClick={handleStop} size="small">
                    {t('chat.stop')}
                  </Button>
                ) : (
                  <Button
                    type="primary"
                    icon={<SendOutlined />}
                    onClick={handleSend}
                    disabled={!inputValue.trim()}
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
      `}</style>
    </div>
  )
}

export default ChatPlayground