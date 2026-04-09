import React from 'react'
import ReactDOM from 'react-dom/client'
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import enUS from 'antd/locale/en_US'
import App from './App'
import './locales/i18n'
import { ThemeProvider } from './contexts/ThemeContext'
import { LanguageProvider } from './contexts/LanguageContext'
import './index.css'

const getAntdLocale = () => {
  const lang = localStorage.getItem('language') || 'zh'
  return lang === 'en' ? enUS : zhCN
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <LanguageProvider>
      <ThemeProvider>
        <ConfigProvider locale={getAntdLocale()}>
          <App />
        </ConfigProvider>
      </ThemeProvider>
    </LanguageProvider>
  </React.StrictMode>,
)