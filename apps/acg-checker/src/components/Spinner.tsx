interface SpinnerProps {
  size?: number
  className?: string
}

export function Spinner({ size = 48, className = '' }: SpinnerProps) {
  return (
    <span
      role="status"
      aria-label="loading"
      className={`inline-block animate-spin rounded-full border-4 border-slate-200 border-t-amber-500 ${className}`}
      style={{ width: size, height: size }}
    />
  )
}
