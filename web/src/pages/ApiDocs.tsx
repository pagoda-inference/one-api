import { useState } from 'react'
import { Card, Tabs, Table, Tag, Button, message, Input, Space, Modal, Form } from 'antd'
import { ApiOutlined, BookOutlined, RocketOutlined, CopyOutlined, EditOutlined } from '@ant-design/icons'

const { TabPane } = Tabs
const { TextArea } = Input

// API 文档配置 - 可以动态修改
const apiDocsConfig = {
  title: 'BEDI 宝塔 API 文档',
  version: 'v1.0',
  baseUrl: '/v1',
  lastUpdated: '2024-03-27',

  // 快速开始配置
  quickStart: {
    title: '快速开始',
    steps: [
      {
        title: '获取 API Key',
        content: '在「API Keys」页面创建一个新的 API Key，请妥善保管，不要泄露。'
      },
      {
        title: '认证方式',
        content: '所有 API 请求需要在 Header 中携带 Authorization 参数：',
        code: 'Authorization: Bearer YOUR_API_KEY'
      },
      {
        title: '发起请求',
        content: '使用以下格式发起聊天请求：',
        code: `curl https://your-domain.com/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -d '{
    "model": "glm-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'`
      }
    ]
  },

  // 接口列表
  endpoints: [
    {
      method: 'POST',
      path: '/v1/chat/completions',
      title: '聊天补全',
      description: '发送对话请求，获取 AI 生成的回答。支持 OpenAI 兼容格式。',
      category: 'chat',
      parameters: [
        { name: 'model', type: 'string', required: true, description: '模型名称，如 glm-4, qwen-72b' },
        { name: 'messages', type: 'array', required: true, description: '对话消息数组' },
        { name: 'temperature', type: 'float', required: false, description: '采样温度，0-2 之间' },
        { name: 'max_tokens', type: 'int', required: false, description: '最大生成 token 数' }
      ],
      examples: {
        curl: `curl https://your-domain.com/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -d '{
    "model": "glm-4",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "Hello!"}
    ],
    "temperature": 0.7,
    "max_tokens": 1000
  }'`,
        python: `import requests

response = requests.post(
    "https://your-domain.com/v1/chat/completions",
    headers={
        "Content-Type": "application/json",
        "Authorization": "Bearer YOUR_API_KEY"
    },
    json={
        "model": "glm-4",
        "messages": [
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": "Hello!"}
        ],
        "temperature": 0.7
    }
)
print(response.json())`,
        javascript: `const response = await fetch('https://your-domain.com/v1/chat/completions', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer YOUR_API_KEY'
  },
  body: JSON.stringify({
    model: 'glm-4',
    messages: [
      {role: 'system', content: 'You are a helpful assistant.'},
      {role: 'user', content: 'Hello!'}
    ],
    temperature: 0.7
  })
});
const data = await response.json();
console.log(data);`
      }
    },
    {
      method: 'POST',
      path: '/v1/embeddings',
      title: '文本嵌入',
      description: '将文本转换为向量表示，用于相似度计算、检索等场景。',
      category: 'embedding',
      parameters: [
        { name: 'model', type: 'string', required: true, description: 'Embedding 模型名称' },
        { name: 'input', type: 'string/array', required: true, description: '要嵌入的文本' }
      ],
      examples: {
        curl: `curl https://your-domain.com/v1/embeddings \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -d '{
    "model": "bge-m3",
    "input": "Hello, world!"
  }'`,
        python: `import requests

response = requests.post(
    "https://your-domain.com/v1/embeddings",
    headers={
        "Content-Type": "application/json",
        "Authorization": "Bearer YOUR_API_KEY"
    },
    json={
        "model": "bge-m3",
        "input": "Hello, world!"
    }
)
print(response.json())`,
        javascript: `const response = await fetch('https://your-domain.com/v1/embeddings', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer YOUR_API_KEY'
  },
  body: JSON.stringify({
    model: 'bge-m3',
    input: 'Hello, world!'
  })
});
const data = await response.json();
console.log(data);`
      }
    },
    {
      method: 'GET',
      path: '/v1/models',
      title: '模型列表',
      description: '获取所有可用模型的列表。',
      category: 'info',
      parameters: [],
      examples: {
        curl: `curl https://your-domain.com/v1/models \\
  -H "Authorization: Bearer YOUR_API_KEY"`,
        python: `import requests

response = requests.get(
    "https://your-domain.com/v1/models",
    headers={"Authorization": "Bearer YOUR_API_KEY"}
)
print(response.json())`,
        javascript: `const response = await fetch('https://your-domain.com/v1/models', {
  headers: {'Authorization': 'Bearer YOUR_API_KEY'}
});
const data = await response.json();
console.log(data);`
      }
    }
  ],

  // 模型分类配置（用于展示）
  modelCategories: [
    {
      name: '对话模型',
      code: 'chat',
      icon: '💬',
      color: '#1890ff',
      models: ['GLM-4', 'Qwen-72B', 'DeepSeek-V3', 'Minimax-M2']
    },
    {
      name: '视觉模型',
      code: 'vlm',
      icon: '🖼️',
      color: '#722ed1',
      models: ['Qwen2.5-VL-72B', 'Qwen3-VL-30B']
    },
    {
      name: 'Embedding',
      code: 'embedding',
      icon: '🔢',
      color: '#52c41a',
      models: ['bge-m3', 'Qwen3-Embedding']
    },
    {
      name: '重排序',
      code: 'reranker',
      icon: '📊',
      color: '#fa8c16',
      models: ['bge-reranker', 'Qwen3-Reranker']
    }
  ]
}

const ApiDocs: React.FC = () => {
  const [activeTab, setActiveTab] = useState('quickstart')
  const [activeExample, setActiveExample] = useState<'curl' | 'python' | 'javascript'>('curl')
  const [editModalVisible, setEditModalVisible] = useState(false)
  const [editingContent, setEditingContent] = useState('')

  // Get current user role from localStorage
  const userInfo = JSON.parse(localStorage.getItem('user_info') || '{}')
  const isAdmin = (userInfo.role || 0) >= 10

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    message.success('已复制到剪贴板')
  }

  const getMethodColor = (method: string) => {
    switch (method) {
      case 'GET': return '#52c41a'
      case 'POST': return '#1890ff'
      case 'PUT': return '#fa8c16'
      case 'DELETE': return '#ff4d4f'
      default: return '#8c8c8c'
    }
  }

  const handleEdit = (content: string) => {
    setEditingContent(content)
    setEditModalVisible(true)
  }

  const handleSaveEdit = () => {
    // TODO: 保存到后端
    message.success('文档内容已保存')
    setEditModalVisible(false)
  }

  return (
    <div style={{ padding: 24 }}>
      <Card
        style={{ borderRadius: 12, marginBottom: 24 }}
        styles={{ body: { padding: '24px 32px' } }}
      >
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div>
            <h1 style={{ fontSize: 24, fontWeight: 700, margin: 0, color: '#262626' }}>
              {apiDocsConfig.title}
            </h1>
            <p style={{ color: '#8c8c8c', margin: '8px 0 0', fontSize: 14 }}>
              版本 {apiDocsConfig.version} · 最后更新 {apiDocsConfig.lastUpdated}
            </p>
          </div>
          {isAdmin && (
            <Button icon={<EditOutlined />} onClick={() => handleEdit(JSON.stringify(apiDocsConfig, null, 2))}>
              编辑文档
            </Button>
          )}
        </div>
      </Card>

      <Card style={{ borderRadius: 12 }}>
        <Tabs activeKey={activeTab} onChange={setActiveTab}>
          <TabPane
            tab={<span><RocketOutlined /> 快速开始</span>}
            key="quickstart"
          >
            <div style={{ maxWidth: 800 }}>
              <h2 style={{ fontSize: 20, fontWeight: 600, marginBottom: 24 }}>快速开始</h2>

              {apiDocsConfig.quickStart.steps.map((step, index) => (
                <div key={index} style={{ marginBottom: 32 }}>
                  <h3 style={{ fontSize: 16, fontWeight: 600, marginBottom: 12 }}>
                    <Tag color="blue">{index + 1}</Tag> {step.title}
                  </h3>
                  <p style={{ color: '#595959', lineHeight: 1.6 }}>{step.content}</p>
                  {step.code && (
                    <div style={{
                      background: '#f5f5f5',
                      borderRadius: 8,
                      padding: 16,
                      position: 'relative',
                      marginTop: 12
                    }}>
                      <pre style={{ margin: 0, fontSize: 13, fontFamily: 'monospace', whiteSpace: 'pre-wrap' }}>
                        {step.code}
                      </pre>
                      <Button
                        type="text"
                        icon={<CopyOutlined />}
                        size="small"
                        style={{ position: 'absolute', top: 8, right: 8 }}
                        onClick={() => copyToClipboard(step.code)}
                      />
                    </div>
                  )}
                </div>
              ))}
            </div>
          </TabPane>

          <TabPane
            tab={<span><ApiOutlined /> 接口文档</span>}
            key="endpoints"
          >
            <div style={{ maxWidth: 900 }}>
              <h2 style={{ fontSize: 20, fontWeight: 600, marginBottom: 24 }}>接口列表</h2>

              {apiDocsConfig.endpoints.map((endpoint, index) => (
                <Card
                  key={index}
                  style={{ marginBottom: 24, borderRadius: 12, border: '1px solid #f0f0f0' }}
                  styles={{ body: { padding: 0 } }}
                >
                  <div style={{ padding: '16px 20px', borderBottom: '1px solid #f0f0f0' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                      <Tag
                        color={getMethodColor(endpoint.method)}
                        style={{ fontWeight: 600, fontSize: 12 }}
                      >
                        {endpoint.method}
                      </Tag>
                      <code style={{ fontSize: 14, fontWeight: 600, color: '#262626' }}>
                        {endpoint.path}
                      </code>
                    </div>
                    <p style={{ color: '#8c8c8c', margin: '8px 0 0', fontSize: 14 }}>
                      {endpoint.description}
                    </p>
                  </div>

                  <div style={{ padding: '16px 20px' }}>
                    <h4 style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>请求参数</h4>
                    {endpoint.parameters.length > 0 ? (
                      <Table
                        size="small"
                        pagination={false}
                        columns={[
                          { title: '参数名', dataIndex: 'name', key: 'name', render: (v: string) => <code>{v}</code> },
                          { title: '类型', dataIndex: 'type', key: 'type' },
                          { title: '必填', dataIndex: 'required', key: 'required', render: (v: boolean) => v ? <Tag color="red">是</Tag> : <Tag color="green">否</Tag> },
                          { title: '说明', dataIndex: 'description', key: 'description' }
                        ]}
                        dataSource={endpoint.parameters.map(p => ({ ...p, key: p.name }))}
                      />
                    ) : (
                      <p style={{ color: '#8c8c8c', fontSize: 14 }}>无请求参数</p>
                    )}

                    <h4 style={{ fontSize: 14, fontWeight: 600, margin: '20px 0 12px' }}>示例代码</h4>
                    <Space style={{ marginBottom: 12 }}>
                      {(['curl', 'python', 'javascript'] as const).map(lang => (
                        <Tag
                          key={lang}
                          color={activeExample === lang ? 'blue' : 'default'}
                          style={{ cursor: 'pointer' }}
                          onClick={() => setActiveExample(lang)}
                        >
                          {lang === 'curl' ? 'cURL' : lang === 'python' ? 'Python' : 'JavaScript'}
                        </Tag>
                      ))}
                    </Space>

                    <div style={{
                      background: '#f5f5f5',
                      borderRadius: 8,
                      padding: 16,
                      position: 'relative'
                    }}>
                      <pre style={{ margin: 0, fontSize: 13, fontFamily: 'monospace', whiteSpace: 'pre-wrap' }}>
                        {endpoint.examples[activeExample]}
                      </pre>
                      <Button
                        type="text"
                        icon={<CopyOutlined />}
                        size="small"
                        style={{ position: 'absolute', top: 8, right: 8 }}
                        onClick={() => copyToClipboard(endpoint.examples[activeExample])}
                      />
                    </div>
                  </div>
                </Card>
              ))}
            </div>
          </TabPane>

          <TabPane
            tab={<span><BookOutlined /> 模型说明</span>}
            key="models"
          >
            <div style={{ maxWidth: 800 }}>
              <h2 style={{ fontSize: 20, fontWeight: 600, marginBottom: 24 }}>模型分类</h2>

              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 16 }}>
                {apiDocsConfig.modelCategories.map((cat, index) => (
                  <Card key={index} style={{ borderRadius: 12, borderLeft: `4px solid ${cat.color}` }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 12 }}>
                      <span style={{ fontSize: 24 }}>{cat.icon}</span>
                      <span style={{ fontWeight: 600, fontSize: 16 }}>{cat.name}</span>
                    </div>
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                      {cat.models.map((model, i) => (
                        <Tag key={i} style={{ borderRadius: 4 }}>{model}</Tag>
                      ))}
                    </div>
                  </Card>
                ))}
              </div>
            </div>
          </TabPane>
        </Tabs>
      </Card>

      <Modal
        title="编辑文档"
        open={editModalVisible}
        onOk={handleSaveEdit}
        onCancel={() => setEditModalVisible(false)}
        width={800}
      >
        <Form layout="vertical">
          <Form.Item label="内容">
            <TextArea
              rows={20}
              value={editingContent}
              onChange={e => setEditingContent(e.target.value)}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default ApiDocs
