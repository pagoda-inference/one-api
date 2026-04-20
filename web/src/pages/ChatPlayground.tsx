import { useState, useEffect, useRef } from 'react'
import { Input, Button, Select, Slider, Switch, Form, Space, Tag, Divider, message } from 'antd'
import { SendOutlined, ClearOutlined, ExperimentOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { relayApi } from '../services/api'

const { TextArea } = Input

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
  const [inputValue, setInputValue] = useState('')
  const [loading, setLoading] = useState(false)
  const [streamedContent, setStreamedContent] = useState('')
  const [chatStarted, setChatStarted] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const abortControllerRef = useRef<AbortController | null>(null)

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

  const selectedModel = trialModels.find(m => m.value === params.model)
  const capabilities = modelCapabilities[params.model] || modelCapabilities['bedi/qwen3-32b']

  useEffect(() => {
    scrollToBottom()
  }, [messages, streamedContent])

  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
    }
  }, [])

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
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
    setInputValue('')
    setChatStarted(true)
    setLoading(true)
    setStreamedContent('')

    // Build messages array
    const chatMessages: Array<{ role: string; content: string | Array<{ type: string; text?: string; image_url?: { url: string } }> }> = []

    // Add system prompt if provided
    if (params.systemPrompt.trim()) {
      chatMessages.push({ role: 'system', content: params.systemPrompt })
    }

    // Add conversation history
    messages.forEach(msg => {
      chatMessages.push({ role: msg.role, content: msg.content })
    })

    // Add current user message
    chatMessages.push({ role: 'user', content: userMessage.content })

    // Build request body
    const requestBody: Record<string, any> = {
      model: params.model,
      messages: chatMessages,
      max_tokens: params.maxTokens,
      temperature: params.temperature,
      top_p: params.topP,
      stream: true,
    }

    // Add optional parameters based on model capabilities
    if (capabilities.hasTopK) {
      requestBody.top_k = params.topK
    }
    if (capabilities.hasFrequencyPenalty) {
      requestBody.frequency_penalty = params.frequencyPenalty
    }
    if (capabilities.hasPresencePenalty) {
      requestBody.presence_penalty = params.presencePenalty
    }
    if (capabilities.hasEnableThinking && params.enableThinking) {
      requestBody.thinking = {
        type: 'thinking',
        thinking: {
          budget_tokens: params.thinkingBudget,
        },
      }
    }

    try {
      abortControllerRef.current = new AbortController()

      const response = await relayApi.post('/chat/completions', requestBody, {
        headers: {
          'Content-Type': 'application/json',
        },
        responseType: 'stream',
        signal: abortControllerRef.current.signal,
      })

      const reader = response.data.getReader()
      const decoder = new TextDecoder()
      let fullContent = ''

      // Read stream
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
                setStreamedContent(fullContent)
              }
            } catch (e) {
              // Skip invalid JSON
            }
          }
        }
      }

      // Add assistant message
      if (fullContent) {
        const assistantMessage: Message = {
          id: (Date.now() + 1).toString(),
          role: 'assistant',
          content: fullContent,
          createdAt: new Date(),
        }
        setMessages(prev => [...prev, assistantMessage])
      }

      setStreamedContent('')
      setLoading(false)
    } catch (error: any) {
      setLoading(false)
      setStreamedContent('')

      if (error.name === 'AbortError') {
        // User cancelled, keep partial response
        if (streamedContent) {
          const assistantMessage: Message = {
            id: (Date.now() + 1).toString(),
            role: 'assistant',
            content: streamedContent,
            createdAt: new Date(),
          }
          setMessages(prev => [...prev, assistantMessage])
        }
        return
      }

      console.error('Chat error:', error)
      const errMsg = error.response?.data?.error?.message || error.message || t('chat.send_failed')
      message.error(errMsg)

      // Add error as assistant message
      const errorMessage: Message = {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: `Error: ${errMsg}`,
        createdAt: new Date(),
      }
      setMessages(prev => [...prev, errorMessage])
    }
  }

  const handleStop = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
    }
  }

  const handleClear = () => {
    setMessages([])
    setStreamedContent('')
    setChatStarted(false)
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div style={{ display: 'flex', height: '100vh', overflow: 'hidden' }}>
      {/* Left: Sub-nav */}
      <div style={{
        width: 200,
        borderRight: '1px solid #f0f0f0',
        padding: '16px 8px',
        background: '#fafafa',
      }}>
        <div style={{ marginBottom: 16 }}>
          <h4 style={{ marginBottom: 8, color: '#595959' }}>{t('chat.experience_center')}</h4>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <Tag color="blue" icon={<ExperimentOutlined />}>{t('chat.text_chat')}</Tag>
          </div>
        </div>

        <Divider style={{ margin: '12px 0' }} />

        <div style={{ fontSize: 12, color: '#8c8c8c', padding: '0 4px' }}>
          <p style={{ marginBottom: 8 }}>{t('chat.trial_models_note')}</p>
          <ul style={{ paddingLeft: 16, margin: 0 }}>
            <li>Qwen3-14B</li>
            <li>Qwen3-32B</li>
            <li>Qwen3-VL-8B</li>
          </ul>
        </div>
      </div>

      {/* Middle: Config panel */}
      <div style={{
        width: 280,
        borderRight: '1px solid #f0f0f0',
        padding: 16,
        overflowY: 'auto',
        background: '#fff',
      }}>
        <h4 style={{ marginBottom: 16 }}>{t('chat.model_config')}</h4>

        <Form layout="vertical" size="small">
          {/* Model selector */}
          <Form.Item label={t('chat.model')}>
            <Select
              value={params.model}
              onChange={(value) => setParams(p => ({ ...p, model: value }))}
              options={trialModels.map(m => ({
                value: m.value,
                label: m.label,
              }))}
            />
          </Form.Item>

          {/* System prompt */}
          <Form.Item label={t('chat.system_prompt')}>
            <TextArea
              rows={3}
              value={params.systemPrompt}
              onChange={(e) => setParams(p => ({ ...p, systemPrompt: e.target.value }))}
              placeholder={t('chat.system_prompt_placeholder')}
            />
          </Form.Item>

          {/* Max tokens */}
          <Form.Item label={t('chat.max_tokens')}>
            <Slider
              min={256}
              max={32768}
              step={256}
              value={params.maxTokens}
              onChange={(value) => setParams(p => ({ ...p, maxTokens: value }))}
              marks={{ 256: '256', 8192: '8K', 16384: '16K', 32768: '32K' }}
            />
            <div style={{ textAlign: 'center', fontSize: 12 }}>{params.maxTokens}</div>
          </Form.Item>

          {/* Temperature */}
          <Form.Item label={t('chat.temperature')}>
            <Slider
              min={0}
              max={2}
              step={0.1}
              value={params.temperature}
              onChange={(value) => setParams(p => ({ ...p, temperature: value }))}
              marks={{ 0: '0', 0.7: '0.7', 1: '1', 2: '2' }}
            />
            <div style={{ textAlign: 'center', fontSize: 12 }}>{params.temperature.toFixed(1)}</div>
          </Form.Item>

          {/* Top-P */}
          <Form.Item label={t('chat.top_p')}>
            <Slider
              min={0}
              max={1}
              step={0.05}
              value={params.topP}
              onChange={(value) => setParams(p => ({ ...p, topP: value }))}
              marks={{ 0: '0', 0.5: '0.5', 1: '1' }}
            />
            <div style={{ textAlign: 'center', fontSize: 12 }}>{params.topP.toFixed(2)}</div>
          </Form.Item>

          {/* Top-K (conditional) */}
          {capabilities.hasTopK && (
            <Form.Item label={t('chat.top_k')}>
              <Slider
                min={1}
                max={100}
                value={params.topK}
                onChange={(value) => setParams(p => ({ ...p, topK: value }))}
                marks={{ 1: '1', 20: '20', 50: '50', 100: '100' }}
              />
              <div style={{ textAlign: 'center', fontSize: 12 }}>{params.topK}</div>
            </Form.Item>
          )}

          {/* Frequency penalty (conditional) */}
          {capabilities.hasFrequencyPenalty && (
            <Form.Item label={t('chat.frequency_penalty')}>
              <Slider
                min={-2}
                max={2}
                step={0.1}
                value={params.frequencyPenalty}
                onChange={(value) => setParams(p => ({ ...p, frequencyPenalty: value }))}
                marks={{ '-2': '-2', 0: '0', 2: '2' }}
              />
              <div style={{ textAlign: 'center', fontSize: 12 }}>{params.frequencyPenalty.toFixed(1)}</div>
            </Form.Item>
          )}

          {/* Presence penalty (conditional) */}
          {capabilities.hasPresencePenalty && (
            <Form.Item label={t('chat.presence_penalty')}>
              <Slider
                min={-2}
                max={2}
                step={0.1}
                value={params.presencePenalty}
                onChange={(value) => setParams(p => ({ ...p, presencePenalty: value }))}
                marks={{ '-2': '-2', 0: '0', 2: '2' }}
              />
              <div style={{ textAlign: 'center', fontSize: 12 }}>{params.presencePenalty.toFixed(1)}</div>
            </Form.Item>
          )}

          {/* Enable thinking (for reasoning models) */}
          {capabilities.hasEnableThinking && (
            <Form.Item label={t('chat.enable_thinking')}>
              <Switch
                checked={params.enableThinking}
                onChange={(checked) => setParams(p => ({ ...p, enableThinking: checked }))}
              />
            </Form.Item>
          )}

          {/* Thinking budget (conditional on enable thinking) */}
          {capabilities.hasEnableThinking && params.enableThinking && (
            <Form.Item label={t('chat.thinking_budget')}>
              <Slider
                min={512}
                max={32768}
                step={256}
                value={params.thinkingBudget}
                onChange={(value) => setParams(p => ({ ...p, thinkingBudget: value }))}
                marks={{ 512: '512', 4096: '4K', 8192: '8K', 16384: '16K', 32768: '32K' }}
              />
              <div style={{ textAlign: 'center', fontSize: 12 }}>{params.thinkingBudget}</div>
            </Form.Item>
          )}
        </Form>
      </div>

      {/* Right: Chat area */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {/* Chat header */}
        <div style={{
          padding: '12px 16px',
          borderBottom: '1px solid #f0f0f0',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}>
          <div>
            <span style={{ fontWeight: 500 }}>{selectedModel?.label || t('chat.text_chat')}</span>
            <Tag color="green" style={{ marginLeft: 8 }}>{t('chat.trial')}</Tag>
          </div>
          <Space>
            <Button size="small" icon={<ClearOutlined />} onClick={handleClear} disabled={loading}>
              {t('chat.clear')}
            </Button>
          </Space>
        </div>

        {/* Messages area */}
        <div style={{
          flex: 1,
          overflowY: 'auto',
          padding: 16,
          display: 'flex',
          flexDirection: 'column',
        }}>
          {!chatStarted ? (
            <div style={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#8c8c8c',
            }}>
              <ExperimentOutlined style={{ fontSize: 48, marginBottom: 16 }} />
              <p>{t('chat.start_conversation')}</p>
              <p style={{ fontSize: 12 }}>{t('chat.trial_models_tip')}</p>
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
                  <div
                    style={{
                      maxWidth: '80%',
                      padding: '10px 14px',
                      borderRadius: 12,
                      background: msg.role === 'user' ? '#1890ff' : '#f5f5f5',
                      color: msg.role === 'user' ? '#fff' : '#333',
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-word',
                    }}
                  >
                    {msg.content}
                  </div>
                </div>
              ))}

              {/* Streaming content */}
              {loading && streamedContent && (
                <div
                  style={{
                    marginBottom: 16,
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'flex-start',
                  }}
                >
                  <div
                    style={{
                      maxWidth: '80%',
                      padding: '10px 14px',
                      borderRadius: 12,
                      background: '#f5f5f5',
                      color: '#333',
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-word',
                    }}
                  >
                    {streamedContent}<span style={{ animation: 'blink 1s infinite' }}>|</span>
                  </div>
                </div>
              )}

              <div ref={messagesEndRef} />
            </>
          )}
        </div>

        {/* Input area */}
        <div style={{
          padding: 16,
          borderTop: '1px solid #f0f0f0',
          background: '#fff',
        }}>
          <div style={{ display: 'flex', gap: 8 }}>
            <TextArea
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyPress}
              placeholder={t('chat.input_placeholder')}
              autoSize={{ minRows: 1, maxRows: 4 }}
              style={{ flex: 1 }}
              disabled={loading}
            />
            <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
              {loading ? (
                <Button type="primary" danger onClick={handleStop}>
                  {t('chat.stop')}
                </Button>
              ) : (
                <Button
                  type="primary"
                  icon={<SendOutlined />}
                  onClick={handleSend}
                  disabled={!inputValue.trim()}
                >
                  {t('chat.send')}
                </Button>
              )}
            </div>
          </div>
          <div style={{ marginTop: 8, fontSize: 11, color: '#8c8c8c' }}>
            {t('chat.disclaimer')}
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