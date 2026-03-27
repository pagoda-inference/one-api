import { Button, Card } from 'antd'

const Login: React.FC = () => {
  const handleFeishuLogin = () => {
    // 跳转到后端飞书OAuth入口
    window.location.href = '/api/oauth/lark'
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      padding: '20px'
    }}>
      <Card
        style={{
          width: '100%',
          maxWidth: 420,
          borderRadius: 16,
          boxShadow: '0 20px 60px rgba(0,0,0,0.3)',
          overflow: 'hidden'
        }}
        styles={{ body: { padding: 0 } }}
      >
        <div style={{
          background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
          padding: '32px 24px',
          textAlign: 'center'
        }}>
          <h1 style={{ color: '#fff', margin: 0, fontSize: 28, fontWeight: 700 }}>One API</h1>
          <p style={{ color: 'rgba(255,255,255,0.8)', margin: '8px 0 0', fontSize: 14 }}>AI 模型网关平台</p>
        </div>

        <div style={{ padding: '40px 32px', textAlign: 'center' }}>
          <div style={{ marginBottom: 32 }}>
            <div style={{
              width: 64,
              height: 64,
              borderRadius: 16,
              background: 'linear-gradient(135deg, #12b7f5, #0877b0 100%)',
              margin: '0 auto 16px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center'
            }}>
              <svg width="36" height="36" viewBox="0 0 36 36" fill="none">
                <path d="M18 4C10.268 4 4 10.268 4 18s6.268 14 14 14 14-6.268 14-14S25.732 4 18 4z" fill="#12B7F5"/>
                <path d="M18 8c-5.523 0-10 4.477-10 10s4.477 10 10 10 10-4.477 10-10S23.523 8 18 8zm4.5 14.5h-9v-9h9v9z" fill="#fff"/>
              </svg>
            </div>
            <h2 style={{ fontSize: 18, fontWeight: 600, color: '#262626', margin: '0 0 8px' }}>
              使用飞书登录
            </h2>
            <p style={{ color: '#8c8c8c', fontSize: 14, margin: 0 }}>
              使用公司飞书账号登录，快速开始使用
            </p>
          </div>

          <Button
            type="primary"
            size="large"
            block
            onClick={handleFeishuLogin}
            style={{
              height: 52,
              borderRadius: 10,
              fontSize: 16,
              fontWeight: 600,
              background: 'linear-gradient(135deg, #12B7F5 0%, #0877b0 100%)',
              border: 'none',
              boxShadow: '0 4px 12px rgba(18, 183, 245, 0.3)'
            }}
          >
            飞书登录 / 注册
          </Button>

          <p style={{ color: '#bfbfbf', fontSize: 12, marginTop: 24 }}>
            登录即表示同意我们的服务条款
          </p>
        </div>
      </Card>
    </div>
  )
}

export default Login
