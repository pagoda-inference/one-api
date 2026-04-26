/// <reference types="vite/client" />

declare const _default: import("vite").UserConfig;
export default _default;

declare module '*.svg' {
  const content: string
  export default content
}

interface ImportMetaEnv {
  readonly VITE_TRIAL_API_KEY?: string
  readonly VITE_VL_IMAGE_PROXY_BASE_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
