export interface RuntimeInfo {
  isPC: boolean
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

function readForceOverride(): 'pc' | 'mobile' | null {
  if (!import.meta.env.DEV) return null
  try {
    const force = new URLSearchParams(window.location.search).get('force')
    if (force === 'pc' || force === 'mobile') return force
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
  const isPC = !isMobile

  const info: RuntimeInfo = { isPC, isMobile, isFeishu }

  if (import.meta.env.DEV) {
    console.info('[runtime] detected', {
      ua,
      uaDataMobile,
      maxTouchPoints,
      coarsePointer,
      noHover,
      uaMobile,
      mobileScore,
      info,
    })
  }

  return info
}
