import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { detectRuntime } from './runtime'

interface Signals {
  ua: string
  uaDataMobile?: boolean
  uaDataPlatform?: string
  maxTouchPoints?: number
  coarsePointer?: boolean
  noHover?: boolean
}

function mockEnvironment(s: Signals) {
  Object.defineProperty(navigator, 'userAgent', {
    configurable: true,
    get: () => s.ua,
  })
  Object.defineProperty(navigator, 'maxTouchPoints', {
    configurable: true,
    get: () => s.maxTouchPoints ?? 0,
  })
  if (s.uaDataMobile === undefined && s.uaDataPlatform === undefined) {
    Object.defineProperty(navigator, 'userAgentData', {
      configurable: true,
      get: () => undefined,
    })
  } else {
    Object.defineProperty(navigator, 'userAgentData', {
      configurable: true,
      get: () => ({
        mobile: s.uaDataMobile ?? false,
        platform: s.uaDataPlatform ?? 'X',
      }),
    })
  }
  const mm = (query: string): MediaQueryList =>
    ({
      matches:
        (query.includes('pointer: coarse') && (s.coarsePointer ?? false)) ||
        (query.includes('hover: none') && (s.noHover ?? false)),
      media: query,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
      onchange: null,
    }) as unknown as MediaQueryList
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: mm,
  })
}

describe('detectRuntime', () => {
  beforeEach(() => {
    window.history.replaceState(null, '', '/')
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('detects macOS Chrome as Mac (免检)', () => {
    mockEnvironment({
      ua: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
      uaDataMobile: false,
    })
    const r = detectRuntime()
    expect(r.isMac).toBe(true)
    expect(r.isPC).toBe(false)
    expect(r.isMobile).toBe(false)
    expect(r.isFeishu).toBe(false)
  })

  it('detects macOS via userAgentData.platform even when UA is opaque', () => {
    mockEnvironment({
      ua: 'Mozilla/5.0',
      uaDataMobile: false,
      uaDataPlatform: 'macOS',
    })
    const r = detectRuntime()
    expect(r.isMac).toBe(true)
    expect(r.isPC).toBe(false)
  })

  it('detects Windows Chrome as PC', () => {
    mockEnvironment({
      ua: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
      uaDataMobile: false,
    })
    const r = detectRuntime()
    expect(r.isPC).toBe(true)
    expect(r.isMac).toBe(false)
    expect(r.isMobile).toBe(false)
  })

  it('detects iPhone Safari as mobile', () => {
    mockEnvironment({
      ua: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1',
      maxTouchPoints: 5,
      coarsePointer: true,
      noHover: true,
    })
    const r = detectRuntime()
    expect(r.isMobile).toBe(true)
    expect(r.isPC).toBe(false)
    expect(r.isMac).toBe(false)
  })

  it('detects Android Chrome as mobile', () => {
    mockEnvironment({
      ua: 'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36',
      uaDataMobile: true,
      maxTouchPoints: 5,
      coarsePointer: true,
      noHover: true,
    })
    const r = detectRuntime()
    expect(r.isMobile).toBe(true)
    expect(r.isPC).toBe(false)
    expect(r.isMac).toBe(false)
  })

  it('does NOT classify iPad as Mac (UA looks like Mac but is touch device)', () => {
    // iPadOS 13+ Safari 默认请求"桌面网站"，UA 写成 "Macintosh; Intel Mac OS X"
    mockEnvironment({
      ua: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15',
      maxTouchPoints: 5,
      coarsePointer: true,
      noHover: true,
    })
    const r = detectRuntime()
    expect(r.isMobile).toBe(true)
    expect(r.isMac).toBe(false)
    expect(r.isPC).toBe(false)
  })

  it('trusts uaDataMobile=true even when other signals are weak', () => {
    mockEnvironment({
      ua: 'Mozilla/5.0 (X11; Linux x86_64)',
      uaDataMobile: true,
    })
    expect(detectRuntime().isMobile).toBe(true)
  })

  it('flags Feishu UA but does NOT force mobile when on Mac client', () => {
    mockEnvironment({
      ua: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Lark/7.0.0',
      uaDataMobile: false,
    })
    const r = detectRuntime()
    expect(r.isFeishu).toBe(true)
    expect(r.isMac).toBe(true)
    expect(r.isPC).toBe(false)
    expect(r.isMobile).toBe(false)
  })

  it('flags Feishu mobile client as mobile', () => {
    mockEnvironment({
      ua: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0) Mobile Safari/604.1 Feishu/7.0',
      uaDataMobile: true,
      maxTouchPoints: 5,
      coarsePointer: true,
      noHover: true,
    })
    const r = detectRuntime()
    expect(r.isFeishu).toBe(true)
    expect(r.isMobile).toBe(true)
  })

  it('respects ?force=mobile override in dev', () => {
    window.history.replaceState(null, '', '/check?force=mobile')
    mockEnvironment({
      ua: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0',
      uaDataMobile: false,
    })
    const r = detectRuntime()
    expect(r.isMobile).toBe(true)
    expect(r.isPC).toBe(false)
    expect(r.isMac).toBe(false)
  })

  it('respects ?force=mac override in dev', () => {
    window.history.replaceState(null, '', '/check?force=mac')
    mockEnvironment({
      ua: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0',
      uaDataMobile: false,
    })
    const r = detectRuntime()
    expect(r.isMac).toBe(true)
    expect(r.isPC).toBe(false)
    expect(r.isMobile).toBe(false)
  })

  it('respects ?force=pc override in dev', () => {
    window.history.replaceState(null, '', '/check?force=pc')
    mockEnvironment({
      ua: 'Mozilla/5.0 (iPhone) Mobile Safari/604.1',
      uaDataMobile: true,
      maxTouchPoints: 5,
      coarsePointer: true,
      noHover: true,
    })
    const r = detectRuntime()
    expect(r.isPC).toBe(true)
    expect(r.isMobile).toBe(false)
    expect(r.isMac).toBe(false)
  })
})
