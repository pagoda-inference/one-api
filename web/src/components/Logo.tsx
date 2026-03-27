export default function Logo({ width = 120, height = 24 }: { width?: number; height?: number }) {
  return (
    <img
      src="/logo.svg"
      alt="BEDI 宝塔"
      style={{ width, height, objectFit: 'contain' }}
    />
  )
}
