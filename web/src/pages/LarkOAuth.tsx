import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Spin } from 'antd'

const LarkOAuth: React.FC = () => {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const [status, setStatus] = useState('正在处理飞书登录...')
  const navigateRef = useRef<string | null>(null)

  useEffect(() => {
    const code = searchParams.get('code')
    const state = searchParams.get('state')
    const appId = searchParams.get('app_id')

    if (!code || !state) {
      setStatus('参数错误，正在跳转...')
      setTimeout(() => navigate('/login'), 2000)
      return
    }

    // 构建 API URL，包含 app_id（如果有）
    const apiUrl = appId
      ? `/api/oauth/lark?code=${code}&state=${state}&app_id=${appId}`
      : `/api/oauth/lark?code=${code}&state=${state}`

    // 调用后端处理 OAuth 回调
    fetch(apiUrl, {
      credentials: 'include'  // 包含 session cookie
    })
      .then(res => res.json())
      .then(data => {
        // 如果已经跳转过，不再处理
        if (navigateRef.current) return

        if (data.success && data.data) {
          setStatus('登录成功，正在跳转...')
          // 直接保存用户信息到 localStorage
          localStorage.setItem('user_info', JSON.stringify(data.data))
          // 用用户ID作为token标识
          localStorage.setItem('access_token', `lark-session-${data.data.id}`)
          // 标记已跳转
          navigateRef.current = '/dashboard'
          // 跳转到 dashboard
          navigate('/dashboard')
        } else {
          setStatus(data.message || '登录失败，正在跳转...')
          navigateRef.current = '/login'
          setTimeout(() => navigate('/login'), 2000)
        }
      })
      .catch(err => {
        console.error('OAuth callback error:', err)
        // 如果已经跳转过，不再处理
        if (navigateRef.current) return
        setStatus('登录失败，正在跳转...')
        navigateRef.current = '/login'
        setTimeout(() => navigate('/login'), 2000)
      })
  }, [])

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)'
    }}>
      <div style={{
        background: '#fff',
        borderRadius: 16,
        padding: '48px 64px',
        textAlign: 'center',
        boxShadow: '0 8px 32px rgba(0,0,0,0.15)'
      }}>
        <Spin size="large" />
        <p style={{
          marginTop: 24,
          fontSize: 16,
          color: '#333'
        }}>{status}</p>
      </div>
    </div>
  )
}

export default LarkOAuth