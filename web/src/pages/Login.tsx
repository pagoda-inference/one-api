import { Button, message } from 'antd'
import Logo from '../components/Logo'

const Login: React.FC = () => {
  const handleFeishuLogin = async () => {
    try {
      // First get OAuth state from server
      const stateRes = await fetch('/api/oauth/state')
      const stateData = await stateRes.json()
      if (!stateData.success || !stateData.data) {
        message.error(stateData.message || '获取登录状态失败')
        return
      }
      const state = stateData.data
      // Redirect to Feishu OAuth
      const redirectUri = `${window.location.origin}/oauth/lark`
      const larkAuthUrl = `https://accounts.feishu.cn/open-apis/authen/v1/authorize?redirect_uri=${encodeURIComponent(redirectUri)}&client_id=cli_a94c9bd14ef95bd2&state=${state}`
      window.location.href = larkAuthUrl
    } catch (err) {
      message.error('登录失败，请重试')
    }
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      background: 'linear-gradient(135deg, #f8fbff 0%, #e8f4ff 100%)'
    }}>
      {/* 左侧品牌区域 */}
      <div style={{
        flex: 1,
        background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 60,
        position: 'relative',
        overflow: 'hidden'
      }}>
        {/* 装饰元素 */}
        <div style={{
          position: 'absolute',
          top: '10%',
          left: '10%',
          width: 200,
          height: 200,
          borderRadius: '50%',
          border: '1px solid rgba(255,255,255,0.1)'
        }} />
        <div style={{
          position: 'absolute',
          bottom: '20%',
          right: '5%',
          width: 150,
          height: 150,
          borderRadius: '50%',
          background: 'rgba(255,255,255,0.05)'
        }} />
        <div style={{
          position: 'absolute',
          top: '40%',
          right: '15%',
          width: 80,
          height: 80,
          borderRadius: 20,
          background: 'rgba(255,255,255,0.08)',
          transform: 'rotate(15deg)'
        }} />

        {/* Logo */}
        <div style={{
          background: '#fff',
          borderRadius: 16,
          padding: '16px 32px',
          boxShadow: '0 8px 32px rgba(0,0,0,0.15)',
          marginBottom: 40,
          position: 'relative',
          zIndex: 1
        }}>
          <Logo width={160} height={40} />
        </div>

        <h1 style={{
          color: '#fff',
          fontSize: 32,
          fontWeight: 700,
          margin: 0,
          textAlign: 'center',
          position: 'relative',
          zIndex: 1
        }}>BEDI 宝塔</h1>

        <p style={{
          color: 'rgba(255,255,255,0.9)',
          fontSize: 16,
          marginTop: 16,
          textAlign: 'center',
          maxWidth: 360,
          lineHeight: 1.6,
          position: 'relative',
          zIndex: 1
        }}>
          企业级 AI 模型网关平台<br />
          统一接入 · 灵活路由 · 便捷管理
        </p>

        {/* 装饰性图标 */}
        <div style={{
          display: 'flex',
          gap: 24,
          marginTop: 48,
          position: 'relative',
          zIndex: 1
        }}>
          {[
            { icon: '🤖', label: 'AI 模型' },
            { icon: '🔐', label: '安全认证' },
            { icon: '📊', label: '用量分析' }
          ].map((item, i) => (
            <div key={i} style={{
              textAlign: 'center',
              color: '#fff'
            }}>
              <div style={{
                width: 56,
                height: 56,
                borderRadius: 16,
                background: 'rgba(255,255,255,0.15)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 24,
                marginBottom: 8
              }}>
                {item.icon}
              </div>
              <span style={{ fontSize: 12, opacity: 0.8 }}>{item.label}</span>
            </div>
          ))}
        </div>
      </div>

      {/* 右侧登录区域 */}
      <div style={{
        flex: 1,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 40
      }}>
        <div style={{ width: '100%', maxWidth: 360 }}>
          <h2 style={{
            fontSize: 28,
            fontWeight: 700,
            color: '#1a1a1a',
            margin: '0 0 8px'
          }}>欢迎回来</h2>
          <p style={{
            color: '#666',
            margin: '0 0 32px',
            fontSize: 14
          }}>请使用公司飞书账号登录</p>

          {/* 飞书登录按钮 */}
          <Button
            type="primary"
            size="large"
            block
            onClick={handleFeishuLogin}
            style={{
              height: 56,
              borderRadius: 12,
              fontSize: 16,
              fontWeight: 600,
              background: 'linear-gradient(135deg, #0075ff 0%, #0052cc 100%)',
              border: 'none',
              boxShadow: '0 4px 16px rgba(0, 117, 255, 0.3)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: 12
            }}
          >
            <svg width="24" height="24" viewBox="0 0 36 36" fill="none">
              <path d="M18 4C10.268 4 4 10.268 4 18s6.268 14 14 14 14-6.268 14-14S25.732 4 18 4z" fill="#fff"/>
              <path d="M18 8c-5.523 0-10 4.477-10 10s4.477 10 10 10 10-4.477 10-10S23.523 8 18 8zm4.5 14.5h-9v-9h9v9z" fill="#12B7F5"/>
            </svg>
            飞书登录 / 注册
          </Button>

          <p style={{
            color: '#999',
            fontSize: 12,
            marginTop: 24,
            textAlign: 'center'
          }}>
            登录即表示同意我们的服务条款
          </p>

          {/* 底部版权 */}
          <p style={{
            color: '#ccc',
            fontSize: 12,
            marginTop: 48,
            textAlign: 'center'
          }}>
            © 2024 BEDI 宝塔. 保留所有权利.
          </p>
        </div>
      </div>
    </div>
  )
}

export default Login
