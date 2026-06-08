export interface RuntimeInfo {
  // isPC: 非 Mac 非 Mobile 的桌面终端（实际指 Windows，需要跑 EDR 检测）
  isPC: boolean
  // isMac: macOS 桌面端，与 PC 一样需要跑 EDR 检测（探测 8445 端口）
  isMac: boolean
  // isMobile: 移动端，视同免检
  isMobile: boolean
  isFeishu: boolean
}

interface UADataLike {
  mobile?: boolean
  platform?: string
}

interface NavigatorWithUAData extends Navigator {
  userAgentData?: UADataLike
}

const MOBILE_UA_RE = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini|Mobile Safari/i
const FEISHU_UA_RE = /Lark|Feishu|LarkLocale|Lark\/|Feishu\//i
// Mac UA 兜底：Chromium 已冻结版本号但 "Macintosh" / "Mac OS X" 字段还在
const MAC_UA_RE = /Macintosh|Mac OS X/i

function readForceOverride(): 'pc' | 'mac' | 'mobile' | null {
  if (!import.meta.env.DEV) return null
  try {
    const force = new URLSearchParams(window.location.search).get('force')
    if (force === 'pc' || force === 'mac' || force === 'mobile') return force
  } catch {
    // ignore
  }
  return null
}

export function detectRuntime(): RuntimeInfo {
  const ua = navigator.userAgent || ''
  const isFeishu = FEISHU_UA_RE.test(ua)

  const force = readForceOverride()
  if (force) {
    const info: RuntimeInfo = {
      isPC: force === 'pc',
      isMac: force === 'mac',
      isMobile: force === 'mobile',
      isFeishu,
    }
    if (import.meta.env.DEV) {
      console.info('[runtime] forced override', { force, info })
    }
    return info
  }

  const nav = navigator as NavigatorWithUAData
  const uaDataMobile = nav.userAgentData?.mobile
  const uaDataPlatform = nav.userAgentData?.platform

  const maxTouchPoints = navigator.maxTouchPoints ?? 0
  const coarsePointer =
    typeof window.matchMedia === 'function'
      ? window.matchMedia('(pointer: coarse)').matches
      : false
  const noHover =
    typeof window.matchMedia === 'function'
      ? window.matchMedia('(hover: none)').matches
      : false
  const uaMobile = MOBILE_UA_RE.test(ua)

  let mobileScore = 0
  if (uaDataMobile === true) mobileScore += 2
  if (uaMobile) mobileScore += 2
  if (coarsePointer) mobileScore += 1
  if (noHover) mobileScore += 1
  if (maxTouchPoints > 1) mobileScore += 1

  const isMobile = uaDataMobile === true || mobileScore >= 3

  // 必须先排除 mobile 再判 Mac：iPadOS Safari 默认请求桌面网站，UA 写成
  // "Macintosh; Intel Mac OS X"，要靠 maxTouchPoints/coarsePointer 先归到 mobile。
  const macByUaData = uaDataPlatform === 'macOS'
  const macByUa = MAC_UA_RE.test(ua)
  const isMac = !isMobile && (macByUaData || macByUa)

  const isPC = !isMobile && !isMac

  const info: RuntimeInfo = { isPC, isMac, isMobile, isFeishu }

  if (import.meta.env.DEV) {
    console.info('[runtime] detected', {
      ua,
      uaDataMobile,
      uaDataPlatform,
      maxTouchPoints,
      coarsePointer,
      noHover,
      uaMobile,
      mobileScore,
      macByUaData,
      macByUa,
      info,
    })
  }

  return info
}
