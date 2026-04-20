import { createContext, useContext, useState, useEffect, ReactNode, useMemo } from 'react'
import { ConfigProvider, theme } from 'antd'

type ThemeMode = 'light' | 'dark'

export interface AppTheme {
  bgPage: string
  bgContainer: string
  bgElevated: string
  bgSidebar: string
  bgHeader: string
  bgInput: string
  bgBubbleUser: string
  bgBubbleAssistant: string
  textPrimary: string
  textSecondary: string
  textTertiary: string
  textOnPrimary: string
  border: string
  borderLight: string
  primary: string
  primaryHover: string
  shadow: string
}

const appThemes: Record<ThemeMode, AppTheme> = {
  light: {
    bgPage: '#f5f7fb',
    bgContainer: '#ffffff',
    bgElevated: '#fcfcfd',
    bgSidebar: '#ffffff',
    bgHeader: '#ffffff',
    bgInput: '#ffffff',
    bgBubbleUser: '#667eea',
    bgBubbleAssistant: '#f5f5f5',
    textPrimary: '#1f2329',
    textSecondary: '#6b7280',
    textTertiary: '#9ca3af',
    textOnPrimary: '#ffffff',
    border: '#e5e7eb',
    borderLight: '#f0f0f0',
    primary: '#667eea',
    primaryHover: '#5a6fe0',
    shadow: '0 1px 3px rgba(0,0,0,0.06)',
  },
  dark: {
    bgPage: '#0b0f14',
    bgContainer: '#11161c',
    bgElevated: '#151b23',
    bgSidebar: '#0d141b',
    bgHeader: '#0f141a',
    bgInput: '#1a212b',
    bgBubbleUser: '#5b6ff5',
    bgBubbleAssistant: '#1a212b',
    textPrimary: 'rgba(255,255,255,0.92)',
    textSecondary: 'rgba(255,255,255,0.68)',
    textTertiary: 'rgba(255,255,255,0.45)',
    textOnPrimary: '#ffffff',
    border: 'rgba(255,255,255,0.10)',
    borderLight: 'rgba(255,255,255,0.06)',
    primary: '#7c8cff',
    primaryHover: '#91a0ff',
    shadow: '0 6px 20px rgba(0,0,0,0.28)',
  },
}

interface ThemeContextType {
  themeMode: ThemeMode
  toggleTheme: () => void
  appTheme: AppTheme
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

export const useTheme = () => {
  const context = useContext(ThemeContext)
  if (!context) {
    throw new Error('useTheme must be used within ThemeProvider')
  }
  return context
}

interface ThemeProviderProps {
  children: ReactNode
}

export const ThemeProvider: React.FC<ThemeProviderProps> = ({ children }) => {
  const [themeMode, setThemeMode] = useState<ThemeMode>(() => {
    const saved = localStorage.getItem('themeMode')
    return saved === 'dark' ? 'dark' : 'light'
  })

  const appTheme = useMemo(() => appThemes[themeMode], [themeMode])

  useEffect(() => {
    localStorage.setItem('themeMode', themeMode)
    document.documentElement.setAttribute('data-theme', themeMode)
    document.body.style.background = appTheme.bgPage

    // Set CSS custom properties for non-React usage
    const root = document.documentElement
    root.style.setProperty('--bg-page', appTheme.bgPage)
    root.style.setProperty('--bg-container', appTheme.bgContainer)
    root.style.setProperty('--bg-elevated', appTheme.bgElevated)
    root.style.setProperty('--bg-sidebar', appTheme.bgSidebar)
    root.style.setProperty('--bg-header', appTheme.bgHeader)
    root.style.setProperty('--bg-input', appTheme.bgInput)
    root.style.setProperty('--bg-bubble-user', appTheme.bgBubbleUser)
    root.style.setProperty('--bg-bubble-assistant', appTheme.bgBubbleAssistant)
    root.style.setProperty('--text-primary', appTheme.textPrimary)
    root.style.setProperty('--text-secondary', appTheme.textSecondary)
    root.style.setProperty('--text-tertiary', appTheme.textTertiary)
    root.style.setProperty('--text-on-primary', appTheme.textOnPrimary)
    root.style.setProperty('--border-color', appTheme.border)
    root.style.setProperty('--border-light', appTheme.borderLight)
    root.style.setProperty('--primary', appTheme.primary)
    root.style.setProperty('--primary-hover', appTheme.primaryHover)
    root.style.setProperty('--shadow', appTheme.shadow)
  }, [themeMode, appTheme])

  const toggleTheme = () => {
    setThemeMode(prev => prev === 'light' ? 'dark' : 'light')
  }

  const antTheme = useMemo(() => ({
    algorithm: themeMode === 'dark' ? theme.darkAlgorithm : theme.defaultAlgorithm,
    token: {
      colorPrimary: appTheme.primary,
      colorBgBase: themeMode === 'dark' ? '#0b0f14' : '#ffffff',
      colorBgContainer: appTheme.bgContainer,
      colorText: appTheme.textPrimary,
      colorTextSecondary: appTheme.textSecondary,
      colorBorder: appTheme.border,
      borderRadius: 10,
    },
    components: {
      Layout: {
        bodyBg: appTheme.bgPage,
        headerBg: appTheme.bgHeader,
        siderBg: appTheme.bgSidebar,
      },
      Card: {
        colorBgContainer: appTheme.bgContainer,
      },
      Input: {
        colorBgContainer: appTheme.bgInput,
      },
      Select: {
        colorBgContainer: appTheme.bgInput,
      },
      Table: {
        colorBgContainer: appTheme.bgContainer,
      },
      Button: {
        primaryShadow: 'none',
      },
    },
  }), [themeMode, appTheme])

  return (
    <ThemeContext.Provider value={{ themeMode, toggleTheme, appTheme }}>
      <ConfigProvider theme={antTheme}>
        {children}
      </ConfigProvider>
    </ThemeContext.Provider>
  )
}
